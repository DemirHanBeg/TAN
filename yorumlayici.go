package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---- Değer türleri ----
type Deger interface{}

type IslevDeger struct {
	Parametreler []string
	Govde        []Dugum
	Kapsam       *Kapsam
}

// döndür için özel sinyal
type DondurSinyali struct{ Deger Deger }

// dur/devam için sinyaller
type DurSinyali struct{}
type DevamSinyali struct{}

// kontrolSinyali: akışı kesen bir sinyal mi (döndür/dur/devam)
func kontrolSinyali(d Deger) bool {
	switch d.(type) {
	case DondurSinyali, DurSinyali, DevamSinyali:
		return true
	}
	return false
}

// TanListe: işaretçi olduğu için indekse atama ve ekle yerinde çalışır
type TanListe struct {
	Elemanlar []Deger
}

// TanSozluk: anahtar-değer eşlemesi (işaretçi, yerinde değişir)
type TanSozluk struct {
	Cift map[string]Deger
	Sira []string // ekleme sırasını korumak için
}

func YeniSozluk() *TanSozluk {
	return &TanSozluk{Cift: map[string]Deger{}, Sira: []string{}}
}

func (s *TanSozluk) koy(anahtar string, deger Deger) {
	if _, var_ := s.Cift[anahtar]; !var_ {
		s.Sira = append(s.Sira, anahtar)
	}
	s.Cift[anahtar] = deger
}

// ---- Kapsam (değişken ortamı) ----
type Kapsam struct {
	degiskenler map[string]Deger
	ust         *Kapsam
}

func YeniKapsam(ust *Kapsam) *Kapsam {
	return &Kapsam{degiskenler: map[string]Deger{}, ust: ust}
}
func (k *Kapsam) al(ad string) (Deger, bool) {
	if d, ok := k.degiskenler[ad]; ok {
		return d, true
	}
	if k.ust != nil {
		return k.ust.al(ad)
	}
	return nil, false
}
func (k *Kapsam) koy(ad string, d Deger) { k.degiskenler[ad] = d }

// ata: değişken zincirde daha önce tanımlıysa orada günceller,
// değilse mevcut kapsamda oluşturur. Böylece iken/eğer blokları
// üst kapsamdaki değişkeni gerçekten değiştirir.
func (k *Kapsam) ata(ad string, d Deger) {
	for k2 := k; k2 != nil; k2 = k2.ust {
		if _, ok := k2.degiskenler[ad]; ok {
			k2.degiskenler[ad] = d
			return
		}
	}
	k.degiskenler[ad] = d
}

// ---- Yorumlayıcı ----
type Yorumlayici struct {
	global    *Kapsam
	kopru     *Kopru
	alinanlar map[string]bool
	kaynakDizin string // içe al çözümlemesi için
}

func YeniYorumlayici() *Yorumlayici {
	y := &Yorumlayici{global: YeniKapsam(nil), kopru: YeniKopru(), alinanlar: map[string]bool{}}
	kuresel_yorumlayici = y
	return y
}

// kuresel_yorumlayici: sun() gibi yerleşiklerin Tan işlevi çağırabilmesi için
var kuresel_yorumlayici *Yorumlayici

// islevCagir: bir Tan işlevini (IslevDeger) Go tarafından çağırır
func (y *Yorumlayici) islevCagir(islev IslevDeger, args []Deger) Deger {
	yeni := YeniKapsam(islev.Kapsam)
	for i, p := range islev.Parametreler {
		if i < len(args) {
			yeni.koy(p, args[i])
		}
	}
	s := y.blokCalistir(islev.Govde, yeni)
	if ds, ok := s.(DondurSinyali); ok {
		return ds.Deger
	}
	return nil
}

func (y *Yorumlayici) Calistir(deyimler []Dugum) {
	for _, d := range deyimler {
		y.calistirDeyim(d, y.global)
	}
}

// iceAl: başka bir .tan dosyasını okur, aynı global kapsamda çalıştırır.
// Böylece o dosyadaki işlev ve değişkenler kullanılabilir olur.
// Aynı dosya iki kez alınmaz (döngüsel içe aktarma koruması).
func (y *Yorumlayici) iceAl(ad string, satir int) {
	if y.alinanlar == nil {
		y.alinanlar = map[string]bool{}
	}
	// Modül adını gerçek dosya yoluna çevir (arama yolları: Modul.go)
	yol, bulundu := modulAra(ad, y.kaynakDizin)
	if !bulundu {
		firlat(satir, "modül bulunamadı: %s\n%s", ad, modulAramaYollari(ad, y.kaynakDizin))
	}
	mutlak, err := filepath.Abs(yol)
	if err != nil {
		mutlak = yol
	}
	if y.alinanlar[mutlak] {
		return // döngüsel/tekrar içe alma
	}
	y.alinanlar[mutlak] = true

	kaynak, err := os.ReadFile(yol)
	if err != nil {
		firlat(satir, "modül okunamadı: %v", err)
	}
	// İç içe içe al'lar bu modülün dizinine göre çözülsün
	eskiDizin := y.kaynakDizin
	y.kaynakDizin = filepath.Dir(mutlak)
	defer func() { y.kaynakDizin = eskiDizin }()

	lexer := YeniLexer(string(kaynak))
	parser := YeniParser(lexer.Tokenle())
	for _, d := range parser.Ayristir() {
		y.calistirDeyim(d, y.global)
	}
}

func (y *Yorumlayici) calistirDeyim(dugum Dugum, k *Kapsam) Deger {
	switch d := dugum.(type) {
	case AtamaDugum:
		k.ata(d.Ad, y.degerle(d.Deger, k))
	case IndeksAtamaDugum:
		hedef := y.degerle(d.Hedef, k)
		switch h := hedef.(type) {
		case *TanListe:
			i := y.indeksAl(d.Indeks, k, len(h.Elemanlar), d.Satir)
			h.Elemanlar[i] = y.degerle(d.Deger, k)
		case *TanSozluk:
			anahtar := metne(y.degerle(d.Indeks, k))
			h.koy(anahtar, y.degerle(d.Deger, k))
		default:
			firlat(d.Satir, "indekslenebilir değer değil (liste veya sözlük bekleniyordu)")
		}
	case YazDugum:
		fmt.Fprintln(Cikti, metne(y.degerle(d.Deger, k)))
	case IslevDugum:
		k.koy(d.Ad, IslevDeger{d.Parametreler, d.Govde, k})
	case EgerDugum:
		if dogruMu(y.degerle(d.Kosul, k)) {
			return y.blokCalistir(d.Govde, YeniKapsam(k))
		} else if d.Degilse != nil {
			return y.blokCalistir(d.Degilse, YeniKapsam(k))
		}
	case IkenDugum:
		for dogruMu(y.degerle(d.Kosul, k)) {
			s := y.blokCalistir(d.Govde, YeniKapsam(k))
			switch s.(type) {
			case DondurSinyali:
				return s
			case DurSinyali:
				return nil
			case DevamSinyali:
				continue
			}
		}
	case HerDugum:
		deger := y.degerle(d.Liste, k)
		switch koleksiyon := deger.(type) {
		case *TanListe:
			for _, oge := range koleksiyon.Elemanlar {
				donguKapsam := YeniKapsam(k)
				donguKapsam.koy(d.Degisken, oge)
				s := y.blokCalistir(d.Govde, donguKapsam)
				switch s.(type) {
				case DondurSinyali:
					return s
				case DurSinyali:
					return nil
				case DevamSinyali:
					continue
				}
			}
		case *TanSozluk:
			for _, anahtar := range koleksiyon.Sira {
				donguKapsam := YeniKapsam(k)
				donguKapsam.koy(d.Degisken, anahtar)
				s := y.blokCalistir(d.Govde, donguKapsam)
				switch s.(type) {
				case DondurSinyali:
					return s
				case DurSinyali:
					return nil
				case DevamSinyali:
					continue
				}
			}
		default:
			firlat(0, "'her' yalnızca liste veya sözlük gezebilir")
		}
	case DondurDugum:
		return DondurSinyali{y.degerle(d.Deger, k)}
	case DurDugum:
		return DurSinyali{}
	case DevamDugum:
		return DevamSinyali{}
	case IceAlDugum:
		y.iceAl(d.Dosya, d.Satir)
	case DeneDugum:
		return y.deneCalistir(d, k)
	default:
		return y.degerle(dugum, k)
	}
	return nil
}

func (y *Yorumlayici) blokCalistir(govde []Dugum, k *Kapsam) Deger {
	for _, d := range govde {
		s := y.calistirDeyim(d, k)
		if kontrolSinyali(s) {
			return s
		}
	}
	return nil
}

// deneCalistir: DeneGovde'yi çalıştırır; TanHata panic'i olursa recover
// ile yakalar, mesajı HataAdi değişkenine bağlayıp YakalaGovde'yi çalıştırır.
// TanHata dışındaki panic'ler (örn. vmDesteklemiyor) tekrar fırlatılır.
// döndür/dur/devam sinyalleri panic değil normal dönüş olduğundan
// buraya hiç uğramaz; dene/yakala gövdelerinden şeffafça geçer.
func (y *Yorumlayici) deneCalistir(d DeneDugum, k *Kapsam) Deger {
	var sonuc Deger
	hataYakalandi := false
	var hataMesaj string

	func() {
		defer func() {
			if r := recover(); r != nil {
				if h, ok := r.(TanHata); ok {
					hataYakalandi = true
					hataMesaj = h.Mesaj
					return
				}
				panic(r) // TanHata değil: yukarı taşı
			}
		}()
		sonuc = y.blokCalistir(d.DeneGovde, YeniKapsam(k))
	}()

	if hataYakalandi {
		yakalaKapsam := YeniKapsam(k)
		yakalaKapsam.koy(d.HataAdi, hataMesaj)
		return y.blokCalistir(d.YakalaGovde, yakalaKapsam)
	}
	return sonuc
}

func (y *Yorumlayici) degerle(dugum Dugum, k *Kapsam) Deger {
	switch d := dugum.(type) {
	case SayiDugum:
		if d.TamMi {
			return d.Tam
		}
		return d.Deger
	case MetinDugum:
		return d.Deger
	case MantikDugum:
		return d.Deger
	case YokDugum:
		return nil
	case DegiskenDugum:
		if v, ok := k.al(d.Ad); ok {
			return v
		}
		firlat(d.Satir, "tanımsız değişken '%s'", d.Ad)
		return nil
	case IkiliDugum:
		return y.ikiliDegerle(d, k)
	case CagriDugum:
		return y.cagriDegerle(d, k)
	case KopruDugum:
		var args []Deger
		for _, a := range d.Argumanlar {
			args = append(args, y.degerle(a, k))
		}
		return y.kopru.Cagir(d.Hedef, args, d.Satir)
	case ListeDugum:
		ogeler := make([]Deger, 0, len(d.Elemanlar))
		for _, o := range d.Elemanlar {
			ogeler = append(ogeler, y.degerle(o, k))
		}
		return &TanListe{Elemanlar: ogeler}
	case SozlukDugum:
		s := YeniSozluk()
		for i := range d.Anahtarlar {
			anahtar := metne(y.degerle(d.Anahtarlar[i], k))
			s.koy(anahtar, y.degerle(d.Degerler[i], k))
		}
		return s
	case IndeksDugum:
		return y.indeksDegerle(d, k)
	}
	return nil
}

// indeksAl: indeks düğümünü sayıya çevirir, sınır denetimi yapar
func (y *Yorumlayici) indeksAl(indeksDugum Dugum, k *Kapsam, uzunluk, satir int) int {
	iv := y.degerle(indeksDugum, k)
	f, ok := kesir(iv)
	if !ok {
		firlat(satir, "indeks sayı olmalı")
	}
	i := int(f)
	if i < 0 || i >= uzunluk {
		firlat(satir, "indeks sınır dışı: %d (uzunluk %d)", i, uzunluk)
	}
	return i
}

func (y *Yorumlayici) indeksDegerle(d IndeksDugum, k *Kapsam) Deger {
	hedef := y.degerle(d.Hedef, k)
	switch h := hedef.(type) {
	case *TanListe:
		i := y.indeksAl(d.Indeks, k, len(h.Elemanlar), d.Satir)
		return h.Elemanlar[i]
	case *TanSozluk:
		anahtar := metne(y.degerle(d.Indeks, k))
		deger, var_ := h.Cift[anahtar]
		if !var_ {
			firlat(d.Satir, "sözlükte anahtar yok: '%s'", anahtar)
		}
		return deger
	}
	firlat(d.Satir, "indekslenebilir değer değil (liste veya sözlük bekleniyordu)")
	return nil
}

func (y *Yorumlayici) cagriDegerle(d CagriDugum, k *Kapsam) Deger {
	// Önce yerleşik işlevler (dilin kendi parçası, ödünç değil)
	if yerlesik, ok := yerlesikler[d.Ad]; ok {
		var args []Deger
		for _, a := range d.Argumanlar {
			args = append(args, y.degerle(a, k))
		}
		return yerlesik(args, d.Satir)
	}
	v, ok := k.al(d.Ad)
	if !ok {
		firlat(d.Satir, "'%s' adında işlev yok", d.Ad)
	}
	islev, ok := v.(IslevDeger)
	if !ok {
		firlat(d.Satir, "'%s' bir işlev değil", d.Ad)
	}
	yeni := YeniKapsam(islev.Kapsam)
	for i, p := range islev.Parametreler {
		if i < len(d.Argumanlar) {
			yeni.koy(p, y.degerle(d.Argumanlar[i], k))
		}
	}
	s := y.blokCalistir(islev.Govde, yeni)
	if ds, ok := s.(DondurSinyali); ok {
		return ds.Deger
	}
	return nil
}

func (y *Yorumlayici) ikiliDegerle(d IkiliDugum, k *Kapsam) Deger {
	if d.Islec == "değil" {
		return !dogruMu(y.degerle(d.Sol, k))
	}
	if d.Islec == "negatif" {
		v := y.degerle(d.Sol, k)
		if i, ok := v.(int64); ok {
			return -i
		}
		if f, ok := v.(float64); ok {
			return -f
		}
		return nil
	}
	sol := y.degerle(d.Sol, k)
	// kısa devre
	if d.Islec == "ve" {
		return dogruMu(sol) && dogruMu(y.degerle(d.Sag, k))
	}
	if d.Islec == "veya" {
		return dogruMu(sol) || dogruMu(y.degerle(d.Sag, k))
	}
	sag := y.degerle(d.Sag, k)

	// metin birleştirme
	if sifreMetin(sol) || sifreMetin(sag) {
		if d.Islec == "+" {
			return metne(sol) + metne(sag)
		}
	}
	if sayiMi(sol) && sayiMi(sag) {
		switch d.Islec {
		case "+", "-", "*", "/", "%":
			sonuc, sifirBolme := sayiIslem(d.Islec, sol, sag)
			if sifirBolme {
				firlat(0, "sıfıra bölme")
			}
			if sonuc != nil {
				return sonuc
			}
		case ">", "<", ">=", "<=", "==", "!=":
			if sonuc, tamam := sayiKarsilastir(d.Islec, sol, sag); tamam {
				return sonuc
			}
		}
	}
	if d.Islec == "==" {
		return metne(sol) == metne(sag)
	}
	if d.Islec == "!=" {
		return metne(sol) != metne(sag)
	}
	// metin karşılaştırma (sıralama için) — iki taraf da metinse
	sm, solMetin := sol.(string)
	gm, sagMetin := sag.(string)
	if solMetin && sagMetin {
		switch d.Islec {
		case ">":
			return sm > gm
		case "<":
			return sm < gm
		case ">=":
			return sm >= gm
		case "<=":
			return sm <= gm
		}
	}
	return nil
}

// ---- Yardımcılar ----
func dogruMu(d Deger) bool {
	switch v := d.(type) {
	case bool:
		return v
	case nil:
		return false
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	}
	return true
}
func sifreMetin(d Deger) bool { _, ok := d.(string); return ok }

func metne(d Deger) string {
	switch v := d.(type) {
	case nil:
		return "yok"
	case bool:
		if v {
			return "doğru"
		}
		return "yanlış"
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == math.Trunc(v) && !math.IsInf(v, 0) && math.Abs(v) < 1e15 {
			return fmt.Sprintf("%d", int64(v))
		}
		// Bilimsel gösterim (1e-06) yerine düz ondalık; gereksiz sıfırları at
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	case *TanListe:
		parcalar := make([]string, len(v.Elemanlar))
		for i, e := range v.Elemanlar {
			if s, ok := e.(string); ok {
				parcalar[i] = "\"" + s + "\""
			} else {
				parcalar[i] = metne(e)
			}
		}
		return "[" + strings.Join(parcalar, ", ") + "]"
	case *TanSozluk:
		parcalar := make([]string, len(v.Sira))
		for i, anahtar := range v.Sira {
			deger := v.Cift[anahtar]
			degMetin := metne(deger)
			if s, ok := deger.(string); ok {
				degMetin = "\"" + s + "\""
			}
			parcalar[i] = "\"" + anahtar + "\": " + degMetin
		}
		return "{" + strings.Join(parcalar, ", ") + "}"
	}
	return fmt.Sprintf("%v", d)
}

// guvenliCalistir: çalışma zamanı hatasını yakalar, temiz basar.
// Dosya modunda hata programı durdurur; REPL'de satırı atlar.
func (y *Yorumlayici) guvenliCalistir(agac []Dugum) (yakalandi bool) {
	defer func() {
		if r := recover(); r != nil {
			if h, ok := r.(TanHata); ok {
				fmt.Fprintln(os.Stderr, h.Error())
				yakalandi = true
			} else {
				panic(r)
			}
		}
	}()
	y.Calistir(agac)
	return false
}

func kaynagiCalistir(y *Yorumlayici, kaynak string) (hataVar bool) {
	defer func() {
		if r := recover(); r != nil {
			if h, ok := r.(TanHata); ok {
				fmt.Fprintln(os.Stderr, h.Error())
				hataVar = true
				return
			}
			panic(r)
		}
	}()
	lexer := YeniLexer(kaynak)
	parser := YeniParser(lexer.Tokenle())
	agac := parser.Ayristir()
	if vmDeneCalistir(agac) {
		return false
	}
	return y.guvenliCalistir(agac)
}

// vmDeneCalistir: programı bytecode'a derleyip VM'de çalıştırmayı dener.
// Kapsam dışıysa (panic vmDesteklemiyor) false döner; çağıran ağaç-gezene düşer.
func vmDeneCalistir(agac []Dugum) (basarili bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(vmDesteklemiyor); ok {
				basarili = false
				return
			}
			panic(r) // başka bir hata: yukarı taşı
		}
	}()
	derleyici := YeniDerleyici()
	kod := derleyici.Derle(agac)
	vm := YeniSanalMakine(kod)
	vm.Calistir()
	return true
}

// repl: etkileşimli kabuk. Değişkenler satırlar arası korunur;
// bir ifade yazarsan değeri yazdırılır.
func repl() {
	fmt.Println("Tan REPL — çıkmak için: çık")
	y := YeniYorumlayici()
	okuyucu := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("tan> ")
		if !okuyucu.Scan() {
			fmt.Println()
			return
		}
		satir := strings.TrimSpace(okuyucu.Text())
		if satir == "" {
			continue
		}
		if satir == "çık" || satir == "cik" {
			return
		}
		replSatiriCalistir(y, satir)
	}
}

func replSatiriCalistir(y *Yorumlayici, kaynak string) {
	defer func() {
		if r := recover(); r != nil {
			if h, ok := r.(TanHata); ok {
				fmt.Fprintln(os.Stderr, h.Error())
			} else {
				panic(r)
			}
		}
	}()
	lexer := YeniLexer(kaynak)
	parser := YeniParser(lexer.Tokenle())
	for _, d := range parser.Ayristir() {
		sonuc := y.calistirDeyim(d, y.global)
		// Yalın bir ifade değer ürettiyse yazdır (yaz zaten kendi basar).
		if sonuc != nil {
			if _, dondur := sonuc.(DondurSinyali); !dondur {
				fmt.Fprintln(Cikti, metne(sonuc))
			}
		}
	}
}

// ---- Ana program ----
