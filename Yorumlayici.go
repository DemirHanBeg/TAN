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

// TanKayitTipi: "kayıt Ad ... son" tanımının şeması — alan sırası + metotlar.
// Kapsam'da işlev gibi ada bağlanır (k.koy(d.Ad, tip)).
type TanKayitTipi struct {
	Ad       string
	Alanlar  []string
	Metotlar map[string]IslevDeger
}

// TanKayit: bir kayıt örneği (işaretçi, alan ataması yerinde değişir —
// TanListe/TanSozluk ile aynı referans semantiği).
type TanKayit struct {
	Tip      *TanKayitTipi
	Degerler map[string]Deger
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
	islevSiniri bool // bkz. YeniIslevKapsami + ata() notu
}

func YeniKapsam(ust *Kapsam) *Kapsam {
	return &Kapsam{degiskenler: map[string]Deger{}, ust: ust}
}

// YeniIslevKapsami: bir İŞLEV ÇAĞRISININ (islevCagir/cagriIfadeDegerle/
// metotCagriDegerle/CagriDugum çağrı yolu) kendi aktivasyon kapsamı için
// kullanılır — YeniKapsam ile AYNI ama islevSiniri=true işaretler. Bkz.
// ata() içindeki "değişken sızması" düzeltmesi notu: eğer/iken/her/dene
// BLOK gövdeleri (aynı aktivasyon içindeki iç içe bloklar) için HÂLÂ düz
// YeniKapsam kullanılmalı (blok sınırı yok, dışarıdaki AYNI çağrının
// değişkenini mutasyona uğratmak GEREKİR — ör. "i=0; iken i<n ... i=i+1
// ... son").
func YeniIslevKapsami(ust *Kapsam) *Kapsam {
	return &Kapsam{degiskenler: map[string]Deger{}, ust: ust, islevSiniri: true}
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

// ata: değişken zincirde daha önce tanımlıysa orada günceller, değilse
// MEVCUT İŞLEV AKTİVASYONU içinde (islevSiniri'ye kadar) oluşturur.
//
// BULUNAN CİDDİ BUG (bu oturumda, LSM motoru benchmark'ında yakalandı):
// ÖNCEDEN bu döngü islevSiniri kontrolü YAPMADAN sınırsız yukarı tırmanıyordu
// — bir işlevin kendi YERELİ olması gereken bir değişken (ör. "i" bir
// arama döngüsü sayacı), eğer AYNI ADLI bir değişken ÇAĞIRANIN (hatta üst
// düzey betiğin) kapsamında ZATEN varsa, o ÇAĞIRANIN değişkenini SESSİZCE
// EZİYORDU (fonksiyonun kendi lokal kapsamı GLOBAL'e kadar YÜRÜYORDU,
// islev.Kapsam GENELDE global olduğundan — üst düzey işlevler global'i
// yakalar). Somut belirti: "iken i<N ... x=birIslevCagir() ... i=i+1 ...
// son" biçiminde bir üst-düzey döngü, eğer birIslevCagir() KENDİ İÇİNDE de
// "i" adlı bir yerel kullanıyorsa, işlev çağrısı ÇAĞIRANIN "i"sini resetleyip
// kendi döngüsünü çalıştırıp geri dönüyordu — dış döngü SONSUZA KADAR aynı
// i değerinde takılı kalıyordu (isim çakışması TAMAMEN kazara, iki taraf da
// birbirinden habersiz). Blok (eğer/iken/her/dene) gövdeleri AYNI SORUNU
// YAŞAMAZ çünkü onlar islevSiniri=false ile işaretli, döngü DOĞRU şekilde
// aynı aktivasyonun kendi dışındaki bloğuna kadar tırmanmaya devam eder.
//
// Düzeltme: islevSiniri=true olan bir kapsama (kendi degiskenler'inde
// bulunamadıktan SONRA) ulaşılırsa tırmanma DURUR — değişken bu işlev
// çağrısının kendi aktivasyonunda (ya da onu saran bloklarda) YOKSA,
// dışarıdaki (kapatan/closure) kapsamda RASTGELE aynı isimde bir şey varsa
// bile ona DOKUNULMAZ, bunun yerine BURADA (çağrının başladığı kapsamda)
// YENİ bir yerel değişken oluşturulur — tıpkı çoğu dilin "atama varsayılan
// olarak yerel" kuralı gibi. Gerçek closure MUTASYONU (iç içe işlev
// tanımının SARAN İŞLEV AKTİVASYONUNU değiştirmesi) codebase'de HİÇBİR
// YERDE kullanılmıyor (grep ile doğrulandı — carpanUret/çarp örneği bile
// sadece OKUYOR, atamıyor) — bu yüzden bu kısıtlama regresyon YARATMIYOR.
func (k *Kapsam) ata(ad string, d Deger) {
	for k2 := k; k2 != nil; k2 = k2.ust {
		if _, ok := k2.degiskenler[ad]; ok {
			k2.degiskenler[ad] = d
			return
		}
		if k2.islevSiniri {
			break
		}
	}
	k.degiskenler[ad] = d
}

// ---- Yorumlayıcı ----
type Yorumlayici struct {
	global      *Kapsam
	kopru       *Kopru
	alinanlar   map[string]bool
	kaynakDizin string // içe al çözümlemesi için

	// hamBellek: hamOku/hamYaz/hamAyir için SİMÜLE EDİLMİŞ düz (linear)
	// bellek — gerçek işlem belleği DEĞİL, Go dilimi (slice) üzerinde bump
	// allocator (elf arka ucundaki tan_ayir ile aynı model). "İşaretçi"
	// burada sadece bu dilime bir int64 indekstir; pointer aritmetiği
	// (ptr+1 gibi) normal tam sayı toplamasıyla çalışır.
	hamBellek []byte
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
	yeni := YeniIslevKapsami(islev.Kapsam)
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
	case KayitTanimDugum:
		metotlar := map[string]IslevDeger{}
		for _, m := range d.Metotlar {
			metotlar[m.Ad] = IslevDeger{m.Parametreler, m.Govde, k}
		}
		k.koy(d.Ad, &TanKayitTipi{Ad: d.Ad, Alanlar: d.Alanlar, Metotlar: metotlar})
	case AlanAtamaDugum:
		hedef := y.degerle(d.Hedef, k)
		kayit, ok := hedef.(*TanKayit)
		if !ok {
			firlat(d.Satir, "alan ataması bir kayıt bekliyor")
		}
		if _, var_ := kayit.Degerler[d.Alan]; !var_ {
			firlat(d.Satir, "'%s' tipinde '%s' alanı yok", kayit.Tip.Ad, d.Alan)
		}
		kayit.Degerler[d.Alan] = y.degerle(d.Deger, k)
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
	case KayitOlusturDugum:
		return y.kayitOlustur(d, k)
	case AlanErisimDugum:
		hedef := y.degerle(d.Hedef, k)
		kayit, ok := hedef.(*TanKayit)
		if !ok {
			firlat(d.Satir, "alan erişimi bir kayıt bekliyor")
		}
		deger, var_ := kayit.Degerler[d.Alan]
		if !var_ {
			firlat(d.Satir, "'%s' tipinde '%s' alanı yok", kayit.Tip.Ad, d.Alan)
		}
		return deger
	case MetotCagriDugum:
		return y.metotCagriDegerle(d, k)
	case CagriIfadeDugum:
		return y.cagriIfadeDegerle(d, k)
	}
	return nil
}

// cagriIfadeDegerle: "hedef(args)" — hedef bir ad değil keyfi bir ifade
// (liste[0], carpanUret(2) gibi). Fonksiyon pointer/callback çağrısı.
func (y *Yorumlayici) cagriIfadeDegerle(d CagriIfadeDugum, k *Kapsam) Deger {
	hedef := y.degerle(d.Hedef, k)
	islev, ok := hedef.(IslevDeger)
	if !ok {
		firlat(d.Satir, "çağrılan değer bir işlev değil")
	}
	yeni := YeniIslevKapsami(islev.Kapsam)
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

// kayitOlustur: "Ad{alan: deger, ...}" örneği kurar. Belirtilmeyen alanlar
// yok (nil) ile başlar.
func (y *Yorumlayici) kayitOlustur(d KayitOlusturDugum, k *Kapsam) Deger {
	v, ok := k.al(d.Ad)
	if !ok {
		firlat(d.Satir, "tanımsız kayıt tipi '%s'", d.Ad)
	}
	tip, ok := v.(*TanKayitTipi)
	if !ok {
		firlat(d.Satir, "'%s' bir kayıt tipi değil", d.Ad)
	}
	degerler := map[string]Deger{}
	for _, alan := range tip.Alanlar {
		degerler[alan] = nil
	}
	for i, alanAdi := range d.AlanAdlari {
		if _, var_ := degerler[alanAdi]; !var_ {
			firlat(d.Satir, "'%s' tipinde '%s' alanı yok", tip.Ad, alanAdi)
		}
		degerler[alanAdi] = y.degerle(d.Degerler[i], k)
	}
	return &TanKayit{Tip: tip, Degerler: degerler}
}

// metotCagriDegerle: "hedef.metot(args)" — ilk parametre ("bu") alıcı
// örneğe bağlanır, kalan argümanlar sırayla ikinci parametreden itibaren.
func (y *Yorumlayici) metotCagriDegerle(d MetotCagriDugum, k *Kapsam) Deger {
	hedef := y.degerle(d.Hedef, k)
	kayit, ok := hedef.(*TanKayit)
	if !ok {
		firlat(d.Satir, "metot çağrısı bir kayıt bekliyor")
	}
	metot, var_ := kayit.Tip.Metotlar[d.Metot]
	if !var_ {
		firlat(d.Satir, "'%s' tipinde '%s' metodu yok", kayit.Tip.Ad, d.Metot)
	}
	yeni := YeniIslevKapsami(metot.Kapsam)
	if len(metot.Parametreler) > 0 {
		yeni.koy(metot.Parametreler[0], kayit) // bu
	}
	for i, arg := range d.Argumanlar {
		pi := i + 1
		if pi < len(metot.Parametreler) {
			yeni.koy(metot.Parametreler[pi], y.degerle(arg, k))
		}
	}
	s := y.blokCalistir(metot.Govde, yeni)
	if ds, ok := s.(DondurSinyali); ok {
		return ds.Deger
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
	case string:
		// Metin indeksleme: s[i] -> tek harflik metin (ELF arka ucuyla ayni)
		hrf := []rune(h)
		i := y.indeksAl(d.Indeks, k, len(hrf), d.Satir)
		return string(hrf[i])
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
	if tip, ok := v.(*TanKayitTipi); ok {
		// Nokta(1, 2) -> alanlara tanım sırasıyla ata (kısayol, {alan:deger}'in alternatifi)
		degerler := map[string]Deger{}
		for i, alan := range tip.Alanlar {
			if i < len(d.Argumanlar) {
				degerler[alan] = y.degerle(d.Argumanlar[i], k)
			} else {
				degerler[alan] = nil
			}
		}
		return &TanKayit{Tip: tip, Degerler: degerler}
	}
	islev, ok := v.(IslevDeger)
	if !ok {
		firlat(d.Satir, "'%s' bir işlev değil", d.Ad)
	}
	yeni := YeniIslevKapsami(islev.Kapsam)
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
		if s, ok := v.(*TanSabitTam); ok {
			maske := sabitMaske(s.Genislik)
			return &TanSabitTam{s.Genislik, s.Imzali, (^s.Bit + 1) & maske}
		}
		return nil
	}
	if d.Islec == "bitdegil" {
		v := y.degerle(d.Sol, k)
		if s, ok := v.(*TanSabitTam); ok {
			return &TanSabitTam{s.Genislik, s.Imzali, ^s.Bit & sabitMaske(s.Genislik)}
		}
		i, ok := v.(int64)
		if !ok {
			firlat(0, "'~' yalnızca tam sayıda çalışır")
		}
		return ^i
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

	// sabit genişlikli tamsayı işlemleri (u8/u16/u32/u64/i8/i16/i32/i64)
	// — iki operand da AYNI genişlik+işaretlilikte olmalı, taşma tanımlı sarma.
	if sa, ok := sol.(*TanSabitTam); ok {
		sb, ok2 := sag.(*TanSabitTam)
		if !ok2 || !sabitUyumlu(sa, sb) {
			bAdi := "?"
			if ok2 {
				bAdi = sabitAdi(sb)
			}
			firlat(0, "tip uyuşmazlığı: '%s' ile '%s' birlikte kullanılamaz ('%s')", sabitAdi(sa), bAdi, d.Islec)
		}
		maske := sabitMaske(sa.Genislik)
		switch d.Islec {
		case "+", "-", "*", "/", "%":
			sonuc, sifirBolme := sabitIslem(d.Islec, sa, sb)
			if sifirBolme {
				firlat(0, "sıfıra bölme")
			}
			return sonuc
		case ">", "<", ">=", "<=", "==", "!=":
			return sabitKarsilastir(d.Islec, sa, sb)
		case "&":
			return &TanSabitTam{sa.Genislik, sa.Imzali, sa.Bit & sb.Bit}
		case "|":
			return &TanSabitTam{sa.Genislik, sa.Imzali, sa.Bit | sb.Bit}
		case "^":
			return &TanSabitTam{sa.Genislik, sa.Imzali, sa.Bit ^ sb.Bit}
		case "<<":
			return &TanSabitTam{sa.Genislik, sa.Imzali, (sa.Bit << uint(sb.Bit)) & maske}
		case ">>":
			if sa.Imzali {
				av := sabitIsaretUzat(sa.Bit, sa.Genislik)
				return &TanSabitTam{sa.Genislik, true, uint64(av>>uint(sb.Bit)) & maske}
			}
			return &TanSabitTam{sa.Genislik, false, sa.Bit >> uint(sb.Bit)}
		}
	}

	// bit operatörleri: yalnızca tam sayı (int64) operandlarda
	switch d.Islec {
	case "&", "|", "^", "<<", ">>":
		si, siOk := sol.(int64)
		sj, sjOk := sag.(int64)
		if !siOk || !sjOk {
			firlat(0, "bit operatörü ('%s') yalnızca tam sayılarda çalışır", d.Islec)
		}
		switch d.Islec {
		case "&":
			return si & sj
		case "|":
			return si | sj
		case "^":
			return si ^ sj
		case "<<":
			return si << uint(sj)
		case ">>":
			return si >> uint(sj)
		}
	}

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
	case *TanSabitTam:
		return v.Bit != 0
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
	case *TanKayit:
		parcalar := make([]string, len(v.Tip.Alanlar))
		for i, alan := range v.Tip.Alanlar {
			deger := v.Degerler[alan]
			degMetin := metne(deger)
			if s, ok := deger.(string); ok {
				degMetin = "\"" + s + "\""
			}
			parcalar[i] = alan + ": " + degMetin
		}
		return v.Tip.Ad + "{" + strings.Join(parcalar, ", ") + "}"
	case *TanSabitTam:
		return sabitMetne(v)
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
