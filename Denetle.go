package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ============================================================
// tan denetle — TAN kaynak denetleyicisi (linter).
//
// Sözdizimini DOĞRULAMAZ (bu iş ayrıştırıcınındır), adlandırma ve
// yapısal KOKULARA odaklanır. Kurallar AST + token taramasına dayanır;
// yeni sözdizimi icat etmez. Raporlar Diyagnostik biçimindedir ve
// --json ile makine-okunur üretilir.
//
// Kullanım:
//   tan denetle <dosya...>
//   tan denetle --json <dosya...>
//
// Çıkış kodu: 0 temiz (ya da yalnız bilgi düzeyi), 1 uyarı/hata bulundu,
// 2 kullanım/argüman hatası.
// ============================================================

// Kurallar (TAN91xx):
//   9101 GOLGE_YERLESIK      uyarı  atama/işlev adı yerleşik bir işlevi gölgeliyor
//   9102 KULLANILMAMIS       uyarı  atanan ama hiç okunmayan değişken
//   9103 ERISILMEZ           uyarı  döndür/dur/devam'dan sonraki deyim
//   9104 TEKRAR_TANIM        uyarı  aynı kapsamda aynı adla ikinci tanım
//   9105 SABIT_KOSUL         uyarı  eğer/iken koşulu sabit değer
//   9106 TASAN_SAYI          uyarı  int64 aralığını aşan tam sayı sabiti
//   9107 KAYIT_ALAN_TEKRAR   uyarı  kayıt alan adı iki kez
//   9108 METOT_ALICI_EKSIK   uyarı  kayıt metodunda alıcı parametresi yok
//   9109 KALICI_ESITLIK      uyarı  ondalık değerlerde == / !=
//   9110 TEK_KULLANIM        bilgi  dosyada hiç çağrılmayan işlev

// denetKural: tek bir kuralın kimliği (kod + öneri).
type denetKural struct {
	kod  string
	onem OnemDuzeyi
	// mesaj: rapora gömülecek tanı metni (adlar çoktan biçimlendirilmiş).
	mesaj func(sb *strings.Builder, ad string) string
	oneri string
}

var denetKurallari = map[string]denetKural{
	"GOLGE_YERLESIK":    {kod: "TAN9101", onem: UYARI, oneri: "Yerleşik adını değişken olarak kullanmak karışıklık yaratır; farklı bir ad seçin."},
	"KULLANILMAMIS":     {kod: "TAN9102", onem: UYARI, oneri: "Atamayı kaldırın ya da değeri kullanın."},
	"ERISILMEZ":         {kod: "TAN9103", onem: UYARI, oneri: "Kodu silin ya da deyim sırasını gözden geçirin."},
	"TEKRAR_TANIM":      {kod: "TAN9104", onem: UYARI, oneri: "Aynı kapsamda tek tanım olmalı; adı değiştirin ya da tekrarlanan tanımı silin."},
	"SABIT_KOSUL":       {kod: "TAN9105", onem: UYARI, oneri: "eğer/iken koşulunda bir değişken kullanın ya da döngüyü kaldırın."},
	"TASAN_SAYI":        {kod: "TAN9106", onem: UYARI, oneri: "Daha küçük bir sayı kullanın ya da değeri ondalık biçimde yazın."},
	"KAYIT_ALAN_TEKRAR": {kod: "TAN9107", onem: UYARI, oneri: "Alan adlarını kayıt içinde benzersiz yapın."},
	"METOT_ALICI_EKSIK": {kod: "TAN9108", onem: UYARI, oneri: "İlk parametre olarak alıcı örneğini ekleyin (ör. 'bu')."},
	"KALICI_ESITLIK":    {kod: "TAN9109", onem: UYARI, oneri: "Ondalık karşılaştırmada tolerans kullanın (ör. |a-b| < 0.0001) ya da tam sayılarla çalışın."},
	"TEK_KULLANIM":      {kod: "TAN9110", onem: BILGI, oneri: "Çağrıyı ekleyin ya da tanımı kaldırın."},
}

func denetMesaj(kural, ad string) string {
	switch kural {
	case "GOLGE_YERLESIK":
		return fmt.Sprintf("'%s' adı yerleşik bir işlevi gölgeliyor", ad)
	case "KULLANILMAMIS":
		return fmt.Sprintf("'%s' değişkeni atanıyor ama hiç okunmuyor", ad)
	case "ERISILMEZ":
		return "döndür/dur/devam'dan sonraki deyimlere erişilemez"
	case "TEKRAR_TANIM":
		return fmt.Sprintf("'%s' tanımı bu kapsamda zaten var", ad)
	case "SABIT_KOSUL":
		return "koşul sabit bir değer (değişken içermiyor)"
	case "TASAN_SAYI":
		return "tam sayı sabiti int64 aralığını aşıyor"
	case "KAYIT_ALAN_TEKRAR":
		return fmt.Sprintf("'%s' alanı kayıtta iki kez tanımlanmış", ad)
	case "METOT_ALICI_EKSIK":
		return fmt.Sprintf("'%s' metodunun alıcı parametresi yok", ad)
	case "KALICI_ESITLIK":
		return "ondalık sayılarda == / != karşılaştırması güvenilir değil"
	case "TEK_KULLANIM":
		return fmt.Sprintf("'%s' işlevi bu dosyada hiç çağrılmıyor", ad)
	}
	return ad
}

// denetKapsam: değişken kullanımı izlemesi (işlev/üst düzey başına).
type denetKapsam struct {
	atamalar map[string][]int // ad -> atama satırları
	okunan   map[string]bool  // bu kapsamda okunan adlar
	tanimlar map[string]bool  // bu kapsamda tanımlanan işlev adları
}

func yeniDenetKapsam() *denetKapsam {
	return &denetKapsam{
		atamalar: map[string][]int{},
		okunan:   map[string]bool{},
		tanimlar: map[string]bool{},
	}
}

// denetci: tek dosyanın denetim durumu.
type denetci struct {
	toklar   []Token
	kaynak   string
	dosya    string
	anahtar  map[string][]int // anahtar kelime satırları (sıralı)
	atamaKuy map[string][]int // 'ad = ...' atama satırları (sıralı)
	islevKuy map[string][]int // 'işlev ad' tanım satırları (sıralı)
	raporlar []Diyagnostik
	cagrilan map[string]bool // dosya geneli çağrılan işlev adları
	okunanGo map[string]bool // dosya geneli okunan adlar
	// üst düzey durumu (TEK_KULLANIM / program algısı)
	ustIslevler []string
	ustYaz      bool // üst düzeyde yaz deyimi var (program sinyali)
}

func (dc *denetci) rapor(kural, ad string, satir, sutun int) {
	k := denetKurallari[kural]
	dc.raporlar = append(dc.raporlar, Diyagnostik{
		Kod:    k.kod,
		Onem:   k.onem.Metin(),
		Mesaj:  denetMesaj(kural, ad),
		Dosya:  dc.dosya,
		Satir:  satir,
		Sutun:  sutun,
		Baglam: satirBaglam(dc.kaynak, satir),
		Oneri:  k.oneri,
	})
}

// satirAnahtar: anahtar kelime satırını sırasıyla çeker (AST gezinmesi
// token akışıyla aynı sırada ilerler).
func (dc *denetci) satirAnahtar(kw string) int {
	return dc.kuyruktan(kw, dc.anahtar)
}

func (dc *denetci) atamaSatir(ad string) int {
	return dc.kuyruktan(ad, dc.atamaKuy)
}

func (dc *denetci) islevSatir(ad string) int {
	return dc.kuyruktan(ad, dc.islevKuy)
}

func (dc *denetci) kuyruktan(k string, kuyruklar map[string][]int) int {
	kuyruk := kuyruklar[k]
	if len(kuyruk) == 0 {
		return 0
	}
	s := kuyruk[0]
	kuyruklar[k] = kuyruk[1:]
	return s
}

// atamaSatirlari: 'ad = ...' atamalarının tüm satırları.
func atamaSatirlari(toklar []Token, ad string) []int {
	var satirlar []int
	for i := 0; i < len(toklar)-1; i++ {
		if toklar[i].Tur == T_TANIMLAYICI && toklar[i].Deger == ad &&
			toklar[i+1].Tur == T_ISLEC && toklar[i+1].Deger == "=" {
			satirlar = append(satirlar, toklar[i].Satir)
		}
	}
	return satirlar
}

// islevSatirlari: 'işlev ad' tanımlarının tüm satırları.
func islevSatirlari(toklar []Token, ad string) []int {
	var satirlar []int
	for i := 0; i < len(toklar)-1; i++ {
		if toklar[i].Tur == T_ANAHTAR && toklar[i].Deger == "işlev" &&
			toklar[i+1].Tur == T_TANIMLAYICI && toklar[i+1].Deger == ad {
			satirlar = append(satirlar, toklar[i].Satir)
		}
	}
	return satirlar
}

// kayitSatirlari: 'kayıt ad' tanımlarının tüm satırları.
func kayitSatirlari(toklar []Token, ad string) []int {
	var satirlar []int
	for i := 0; i < len(toklar)-1; i++ {
		if toklar[i].Tur == T_ANAHTAR && toklar[i].Deger == "kayıt" &&
			toklar[i+1].Tur == T_TANIMLAYICI && toklar[i+1].Deger == ad {
			satirlar = append(satirlar, toklar[i].Satir)
		}
	}
	return satirlar
}

// ifadeSabitMi: ifadede değişken/çağrı/erişim yoksa sabittir.
func ifadeSabitMi(d Dugum) bool {
	switch n := d.(type) {
	case SayiDugum, MetinDugum, MantikDugum, YokDugum:
		return true
	case IkiliDugum:
		return ifadeSabitMi(n.Sol) && ifadeSabitMi(n.Sag)
	case ListeDugum:
		for _, e := range n.Elemanlar {
			if !ifadeSabitMi(e) {
				return false
			}
		}
		return true
	case SozlukDugum:
		for i := range n.Degerler {
			if !ifadeSabitMi(n.Anahtarlar[i]) || !ifadeSabitMi(n.Degerler[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// ondalikSabitMi: ifade ondalık (tam olmayan) bir sayı sabiti içeriyor mu.
func ondalikSabitMi(d Dugum) bool {
	switch n := d.(type) {
	case SayiDugum:
		return !n.TamMi
	case IkiliDugum:
		return ondalikSabitMi(n.Sol) || ondalikSabitMi(n.Sag)
	case ListeDugum:
		for _, e := range n.Elemanlar {
			if ondalikSabitMi(e) {
				return true
			}
		}
		return false
	case SozlukDugum:
		for _, e := range n.Degerler {
			if ondalikSabitMi(e) {
				return true
			}
		}
		return false
	}
	return false
}

// ifadeOku: ifadedeki değişken okumalarını ve çağrıları toplar; ayrıca
// KALICI_ESITLIK kuralını denetler.
func (dc *denetci) ifadeOku(d Dugum, kapsam *denetKapsam) {
	switch n := d.(type) {
	case DegiskenDugum:
		kapsam.okunan[n.Ad] = true
		dc.okunanGo[n.Ad] = true
	case IkiliDugum:
		if (n.Islec == "==" || n.Islec == "!=") &&
			(ondalikSabitMi(n.Sol) || ondalikSabitMi(n.Sag)) {
			dc.rapor("KALICI_ESITLIK", "", n.Satir, n.Sutun)
		}
		dc.ifadeOku(n.Sol, kapsam)
		dc.ifadeOku(n.Sag, kapsam)
	case CagriDugum:
		kapsam.okunan[n.Ad] = true
		dc.okunanGo[n.Ad] = true
		dc.cagrilan[n.Ad] = true
		for _, a := range n.Argumanlar {
			dc.ifadeOku(a, kapsam)
		}
	case MetotCagriDugum:
		dc.ifadeOku(n.Hedef, kapsam)
		for _, a := range n.Argumanlar {
			dc.ifadeOku(a, kapsam)
		}
	case CagriIfadeDugum:
		dc.ifadeOku(n.Hedef, kapsam)
		for _, a := range n.Argumanlar {
			dc.ifadeOku(a, kapsam)
		}
	case KopruDugum:
		for _, a := range n.Argumanlar {
			dc.ifadeOku(a, kapsam)
		}
	case ListeDugum:
		for _, e := range n.Elemanlar {
			dc.ifadeOku(e, kapsam)
		}
	case SozlukDugum:
		for i := range n.Degerler {
			dc.ifadeOku(n.Anahtarlar[i], kapsam)
			dc.ifadeOku(n.Degerler[i], kapsam)
		}
	case IndeksDugum:
		dc.ifadeOku(n.Hedef, kapsam)
		dc.ifadeOku(n.Indeks, kapsam)
	case IndeksAtamaDugum:
		dc.ifadeOku(n.Hedef, kapsam)
		dc.ifadeOku(n.Indeks, kapsam)
		dc.ifadeOku(n.Deger, kapsam)
	case AlanAtamaDugum:
		dc.ifadeOku(n.Hedef, kapsam)
		dc.ifadeOku(n.Deger, kapsam)
	case AlanErisimDugum:
		dc.ifadeOku(n.Hedef, kapsam)
	case KayitOlusturDugum:
		for _, dg := range n.Degerler {
			dc.ifadeOku(dg, kapsam)
		}
	}
}

// kullanilmamislariRaporla: bir kapsamda atanan ama okunmayan adları bildirir.
// Harita yinelemesi sıralı olmadığından adlar önce konuma (satır, ad) göre
// sıralanır — çıktı böylece deterministtir.
func (dc *denetci) kullanilmamislariRaporla(kapsam *denetKapsam) {
	adlar := make([]string, 0, len(kapsam.atamalar))
	for ad := range kapsam.atamalar {
		adlar = append(adlar, ad)
	}
	sort.Slice(adlar, func(i, j int) bool {
		si := 0
		if dizi := kapsam.atamalar[adlar[i]]; len(dizi) > 0 {
			si = dizi[0]
		}
		sj := 0
		if dizi := kapsam.atamalar[adlar[j]]; len(dizi) > 0 {
			sj = dizi[0]
		}
		if si != sj {
			return si < sj
		}
		return adlar[i] < adlar[j]
	})
	for _, ad := range adlar {
		if kapsam.okunan[ad] {
			continue
		}
		satir := 0
		if dizi := kapsam.atamalar[ad]; len(dizi) > 0 {
			satir = dizi[0]
		}
		dc.rapor("KULLANILMAMIS", ad, satir, 0)
	}
}

// deyimOku: deyim listesini gezer; kapsam/çıktı yan etkilerini toplar.
func (dc *denetci) deyimOku(deyimler []Dugum, kapsam *denetKapsam, ustDuzey bool) {
	for i, d := range deyimler {
		switch n := d.(type) {
		case AtamaDugum:
			satir := dc.atamaSatir(n.Ad)
			if _, golge := yerlesikler[n.Ad]; golge {
				dc.rapor("GOLGE_YERLESIK", n.Ad, satir, 0)
			}
			kapsam.atamalar[n.Ad] = append(kapsam.atamalar[n.Ad], satir)
			dc.ifadeOku(n.Deger, kapsam)
		case IslevDugum:
			satir := dc.islevSatir(n.Ad)
			if ustDuzey {
				dc.ustIslevler = append(dc.ustIslevler, n.Ad)
			}
			if _, golge := yerlesikler[n.Ad]; golge {
				dc.rapor("GOLGE_YERLESIK", n.Ad, satir, 0)
			}
			if kapsam.tanimlar[n.Ad] {
				dc.rapor("TEKRAR_TANIM", n.Ad, satir, 0)
			}
			kapsam.tanimlar[n.Ad] = true
			yeni := yeniDenetKapsam()
			for _, p := range n.Parametreler {
				yeni.okunan[p] = true
			}
			dc.deyimOku(n.Govde, yeni, false)
			dc.kullanilmamislariRaporla(yeni)
			// içteki okumalar dış kapsamı da "kullanılmış" sayar (korunumlu).
			for ad := range yeni.okunan {
				kapsam.okunan[ad] = true
			}
		case KayitTanimDugum:
			alanlar := map[string]bool{}
			for _, a := range n.Alanlar {
				if alanlar[a] {
					dc.rapor("KAYIT_ALAN_TEKRAR", a, dc.ilkSatir(kayitSatirlari(dc.toklar, n.Ad)), 0)
				}
				alanlar[a] = true
			}
			metotAdlari := map[string]bool{}
			for _, m := range n.Metotlar {
				satir := dc.islevSatir(m.Ad)
				if metotAdlari[m.Ad] {
					dc.rapor("TEKRAR_TANIM", m.Ad, satir, 0)
				}
				metotAdlari[m.Ad] = true
				if len(m.Parametreler) == 0 {
					dc.rapor("METOT_ALICI_EKSIK", m.Ad, satir, 0)
				}
				yeni := yeniDenetKapsam()
				for _, p := range m.Parametreler {
					yeni.okunan[p] = true
				}
				dc.deyimOku(m.Govde, yeni, false)
				dc.kullanilmamislariRaporla(yeni)
				for ad := range yeni.okunan {
					kapsam.okunan[ad] = true
				}
			}
		case EgerDugum:
			satir := dc.satirAnahtar("eğer")
			if ifadeSabitMi(n.Kosul) {
				dc.rapor("SABIT_KOSUL", "", satir, 0)
			}
			dc.ifadeOku(n.Kosul, kapsam)
			dc.deyimOku(n.Govde, kapsam, false)
			dc.deyimOku(n.Degilse, kapsam, false)
		case IkenDugum:
			satir := dc.satirAnahtar("iken")
			if ifadeSabitMi(n.Kosul) {
				dc.rapor("SABIT_KOSUL", "", satir, 0)
			}
			dc.ifadeOku(n.Kosul, kapsam)
			dc.deyimOku(n.Govde, kapsam, false)
		case HerDugum:
			satir := dc.satirAnahtar("her")
			dc.ifadeOku(n.Liste, kapsam)
			kapsam.atamalar[n.Degisken] = append(kapsam.atamalar[n.Degisken], satir)
			dc.deyimOku(n.Govde, kapsam, false)
		case DeneDugum:
			dc.deyimOku(n.DeneGovde, kapsam, false)
			dc.deyimOku(n.YakalaGovde, kapsam, false)
		case YazDugum:
			dc.ifadeOku(n.Deger, kapsam)
		case DondurDugum:
			dc.ifadeOku(n.Deger, kapsam)
		case CagriDugum, IndeksAtamaDugum, AlanAtamaDugum:
			dc.ifadeOku(d, kapsam)
		case IceAlDugum, DurDugum, DevamDugum:
		}

		if ustDuzey {
			switch d.(type) {
			case IslevDugum, KayitTanimDugum, IceAlDugum:
			case YazDugum:
				dc.ustYaz = true
			}
		}

		// erişilmez deyim: blok sonlandırıcıdan sonraki deyimler
		var sonlandirici string
		switch d.(type) {
		case DondurDugum:
			sonlandirici = "döndür"
		case DurDugum:
			sonlandirici = "dur"
		case DevamDugum:
			sonlandirici = "devam"
		}
		if sonlandirici != "" {
			satir := dc.satirAnahtar(sonlandirici)
			if i < len(deyimler)-1 {
				dc.rapor("ERISILMEZ", "", satir, 0)
			}
		}
	}
}

func (dc *denetci) ilkSatir(satirlar []int) int {
	if len(satirlar) == 0 {
		return 0
	}
	return satirlar[0]
}

// dosyayiDenetle: tek dosyayı denetler, raporları döndürür.
func dosyayiDenetle(dosya string) ([]Diyagnostik, error) {
	kaynak, err := os.ReadFile(dosya)
	if err != nil {
		return nil, err
	}

	toklar, err := guvenliTokenle(string(kaynak))
	if err != nil {
		return nil, err
	}
	if _, err = guvenliAyristir(toklar); err != nil {
		return nil, err
	}

	dc := &denetci{
		toklar:   toklar,
		kaynak:   string(kaynak),
		dosya:    dosya,
		anahtar:  map[string][]int{},
		atamaKuy: map[string][]int{},
		islevKuy: map[string][]int{},
		cagrilan: map[string]bool{},
		okunanGo: map[string]bool{},
	}
	for i, t := range toklar {
		if t.Tur == T_ANAHTAR {
			dc.anahtar[t.Deger] = append(dc.anahtar[t.Deger], t.Satir)
		}
		if t.Tur == T_TANIMLAYICI && i+1 < len(toklar) &&
			toklar[i+1].Tur == T_ISLEC && toklar[i+1].Deger == "=" {
			dc.atamaKuy[t.Deger] = append(dc.atamaKuy[t.Deger], t.Satir)
		}
		if t.Tur == T_ANAHTAR && t.Deger == "işlev" && i+1 < len(toklar) &&
			toklar[i+1].Tur == T_TANIMLAYICI {
			dc.islevKuy[toklar[i+1].Deger] = append(dc.islevKuy[toklar[i+1].Deger], t.Satir)
		}
	}

	// TASAN_SAYI: int64 dışı tam sayı sabitleri (token düzeyinde).
	for _, t := range toklar {
		if t.Tur == T_SAYI && tamSayiSabiti(t.Deger) {
			if _, err := strconv.ParseInt(t.Deger, 10, 64); err != nil {
				dc.rapor("TASAN_SAYI", "", t.Satir, t.Sutun)
			}
		}
	}

	agac, _ := guvenliAyristir(toklar)
	ust := yeniDenetKapsam()
	dc.deyimOku(agac, ust, true)

	// üst düzey atanan + dosya geneli hiç okunmayan adlar
	dc.kullanilmamislariRaporla(ust)

	// TEK_KULLANIM: yalnız "program" dosyalarında (üst düzey yaz deyimi var).
	// Kütüphaneler içe aktarılmak üzere yazılır; işlevleri dışarıdan çağrılır.
	if dc.ustYaz {
		for _, ad := range dc.ustIslevler {
			if !dc.cagrilan[ad] {
				dc.rapor("TEK_KULLANIM", ad, dc.ilkSatir(islevSatirlari(toklar, ad)), 0)
			}
		}
	}
	return dc.raporlar, nil
}

// tamSayiSabiti: tümü rakam olan (nokta/e içermeyen) sayı sabiti.
func tamSayiSabiti(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// guvenliTokenle: tokenleştirmeyi panic'ten korur.
func guvenliTokenle(kaynak string) (toklar []Token, hata error) {
	defer func() {
		if r := recover(); r != nil {
			if th, ok := r.(TanHata); ok {
				hata = th
				return
			}
			panic(r)
		}
	}()
	return YeniLexer(kaynak).Tokenle(), nil
}

// guvenliAyristir: ayrıştırmayı panic'ten korur (firlat → TanHata).
func guvenliAyristir(toklar []Token) (agac []Dugum, hata error) {
	defer func() {
		if r := recover(); r != nil {
			if th, ok := r.(TanHata); ok {
				hata = th
				return
			}
			panic(r)
		}
	}()
	return YeniParser(toklar).Ayristir(), nil
}

// ---- CLI ----

func denetleKullanim() {
	fmt.Println("Kullanım: tan denetle [--json] <dosya...>")
	fmt.Println("  --json  makine-okunur rapor üretir")
	fmt.Println("Çıkış kodu: 0 temiz (ya da yalnız bilgi), 1 uyarı/hata, 2 kullanım hatası")
}

func denetleKomutu(args []string) {
	jsonMi := false
	var dosyalar []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonMi = true
		case "--yardim", "-h":
			denetleKullanim()
			return
		default:
			dosyalar = append(dosyalar, a)
		}
	}
	if len(dosyalar) == 0 {
		denetleKullanim()
		os.Exit(2)
	}

	tumu := []Diyagnostik{}
	sorunVar := false
	kaldi := 0
	for _, d := range dosyalar {
		raporlar, err := dosyayiDenetle(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tan denetle: %s: %v\n", d, err)
			kaldi++
			sorunVar = true
			continue
		}
		tumu = append(tumu, raporlar...)
		for _, r := range raporlar {
			if r.Onem == UYARI.Metin() || r.Onem == HATA.Metin() || r.Onem == OLUMCU.Metin() {
				sorunVar = true
			}
		}
	}

	if jsonMi {
		b, err := json.Marshal(tumu)
		if err == nil {
			fmt.Println(string(b))
		}
	} else {
		for _, r := range tumu {
			fmt.Print(r.insanMetni())
		}
		uyari, bilgi := 0, 0
		for _, r := range tumu {
			switch r.Onem {
			case UYARI.Metin():
				uyari++
			case BILGI.Metin():
				bilgi++
			}
		}
		fmt.Printf("=== denetle: %d uyarı, %d bilgi ===\n", uyari, bilgi)
	}
	if kaldi > 0 {
		os.Exit(1)
	}
	if sorunVar {
		os.Exit(1)
	}
}
