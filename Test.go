package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ============================================================
// tan test — yerel test çerçevesi (v1).
//
// Sözdizimi değişikliği YOKTUR: test dosyaları sıradan .tan
// programlarıdır; exit 0 = başarılı (GercekProgramlar.sh ile aynı
// sözleşme). Doğrulama için yeni yerleşik işlevler:
//
//	bekle(koşul)          koşul doğru değilse TAN9001 fırlatır
//	bekleEsit(a, b)       metin gösterimleri eşit değilse TAN9002
//
// Tür (birim/entegrasyon/regresyon) dizinden çözülür, dosya başındaki
// "# tür: ..." yorumu ezebilir.
//
// Kullanım:
//	tan test [--liste|--json|--ayrinti] [dosya|dizin ...]
// ============================================================

// testTurleri: geçerli test türleri.
var testTurleri = map[string]bool{
	"birim":       true,
	"entegrasyon": true,
	"regresyon":   true,
}

type testKaydi struct {
	Dosya  string // göreli görünen yol
	Mutlak string
	Tur    string
}

// turVarsayilan: dizin konumundan tür çıkarır; bilinmiyorsa "birim".
func turVarsayilan(dosya string) string {
	normal := filepath.ToSlash(dosya)
	for _, k := range []string{"testler/birim/", "testler/entegrasyon/", "testler/regresyon/"} {
		if strings.Contains(normal, k) {
			return strings.TrimSuffix(strings.TrimPrefix(k, "testler/"), "/")
		}
	}
	if strings.Contains(normal, "testler/") {
		return "entegrasyon"
	}
	return "birim"
}

// turYorumdanOku: dosya başındaki "# tür: <değer>" yorumunu arar.
// Geçersiz değer yok sayılır (dizin varsayılanı geçerli kalır).
func turYorumdanOku(dosya string) string {
	kaynak, err := os.ReadFile(dosya)
	if err != nil {
		return ""
	}
	uz := len(kaynak)
	if uz > 2000 {
		uz = 2000
	}
	ust := string(kaynak[:uz])
	for _, satir := range strings.Split(ust, "\n") {
		s := strings.TrimSpace(satir)
		if !strings.HasPrefix(s, "#") {
			continue
		}
		idx := strings.Index(s, ":")
		if idx < 0 {
			continue
		}
		anahtar := strings.ToLower(strings.TrimSpace(s[1:idx]))
		deger := strings.ToLower(strings.TrimSpace(s[idx+1:]))
		if anahtar != "tür" && anahtar != "tur" {
			continue
		}
		if testTurleri[deger] {
			return deger
		}
	}
	return ""
}

// testBul: tek bir dosyayı test kaydına çevirir (tür yorumu öncelikli).
func testBul(dosya string) (testKaydi, bool) {
	bilgi, err := os.Stat(dosya)
	if err != nil || bilgi.IsDir() {
		return testKaydi{}, false
	}
	mutlak, err := filepath.Abs(dosya)
	if err != nil {
		mutlak = dosya
	}
	t := testKaydi{Dosya: dosya, Mutlak: mutlak, Tur: turVarsayilan(dosya)}
	if yorum := turYorumdanOku(mutlak); yorum != "" {
		t.Tur = yorum
	}
	return t, true
}

// testleriKeşfet: varsayılan keşif — çalışma dizini kalıpları + testler/.
func testleriKesfet(calisma string) []testKaydi {
	var dosyalar []string
	ekle := func(desen string) {
		eslesen, _ := filepath.Glob(desen)
		dosyalar = append(dosyalar, eslesen...)
	}
	// Çalışma dizinindeki adlandırma kalıpları (birim testleri).
	for _, d := range []string{"*_test.tan", "Test*.tan", "test*.tan"} {
		ekle(filepath.Join(calisma, d))
	}
	// testler/ hiyerarşisi — yalnızca tür kategorisi dizinleri otomatik keşfedilir.
	// testler/ kökündeki eski program tarzı dosyalar (bazıları uzun süren ölçüm/
	// kıyaslama betikleri) otomatik keşfe dahil EDİLMEZ; istenirse açıkça:
	//   tan test testler/   (dizin argümanı ile).
	taban := filepath.Join(calisma, "testler")
	for _, d := range []string{"birim/*.tan", "entegrasyon/*.tan", "regresyon/*.tan"} {
		ekle(filepath.Join(taban, d))
	}
	sort.Strings(dosyalar)
	görülen := map[string]bool{}
	var testler []testKaydi
	for _, dosya := range dosyalar {
		t, ok := testBul(dosya)
		if !ok || görülen[t.Mutlak] {
			continue
		}
		görülen[t.Mutlak] = true
		// Göreli göster: çalışma dizinine göre, yoksa mutlak.
		if gor, err := filepath.Rel(calisma, t.Mutlak); err == nil && !strings.HasPrefix(gor, "..") {
			t.Dosya = gor
		} else {
			t.Dosya = t.Mutlak
		}
		testler = append(testler, t)
	}
	return testler
}

// testSonuc: tek testin çalıştırma sonucu.
type testSonuc struct {
	Dosya  string `json:"dosya"`
	Tur    string `json:"tur"`
	Gecti  bool   `json:"gecti"`
	SureMS int64  `json:"sure_ms"`
	Cikti  string `json:"cikti,omitempty"`
}

// testiCalistir: dosyayı yorumlayıcıyla (VM, gerekirse ağaç-gezen) çalıştırır.
// stdout bir tampona alınır; başarısız testin çıktısı raporlanır.
func testiCalistir(t testKaydi) testSonuc {
	s := testSonuc{Dosya: t.Dosya, Tur: t.Tur}
	kaynak, err := os.ReadFile(t.Mutlak)
	if err != nil {
		return s
	}
	y := YeniYorumlayici()
	y.kaynakDosya = t.Mutlak
	y.kaynakDizin = filepath.Dir(t.Mutlak)

	oncekiCikti := Cikti
	oncekiArgs := tanScriptArgs
	var tampon bytes.Buffer
	Cikti = &tampon
	tanScriptArgs = []string{t.Mutlak}

	basla := time.Now()
	hataVar := kaynagiCalistir(y, string(kaynak))
	s.SureMS = time.Since(basla).Milliseconds()

	Cikti = oncekiCikti
	tanScriptArgs = oncekiArgs

	s.Cikti = tampon.String()
	s.Gecti = !hataVar
	return s
}

// raporYaz: Türkçe insan-okunur rapor.
func raporYaz(sonuclar []testSonuc, ayrinti bool) (gecti, kaldi int) {
	enUzun := 0
	for _, s := range sonuclar {
		if len(s.Dosya) > enUzun {
			enUzun = len(s.Dosya)
		}
	}
	for _, s := range sonuclar {
		if s.Gecti {
			gecti++
		} else {
			kaldi++
		}
		fmt.Printf("[%s] %-*s  %s (%dms)\n", s.Tur, enUzun, s.Dosya, isim(s.Gecti), s.SureMS)
		if !s.Gecti {
			if s.Cikti != "" {
				for _, satir := range strings.Split(strings.TrimRight(s.Cikti, "\n"), "\n") {
					fmt.Println("      " + satir)
				}
			}
		} else if ayrinti && s.Cikti != "" {
			for _, satir := range strings.Split(strings.TrimRight(s.Cikti, "\n"), "\n") {
				fmt.Println("      " + satir)
			}
		}
	}
	return gecti, kaldi
}

func isim(gecti bool) string {
	if gecti {
		return "GECTI"
	}
	return "HATA"
}

func testKullanim() {
	fmt.Println("Kullanım: tan test [--liste] [--json] [--ayrinti] [dosya|dizin ...]")
	fmt.Println("  (argümansız: çalışma dizini + testler/ keşfi)")
	fmt.Println("  --liste    keşfedilen testleri çalıştırmadan listeler")
	fmt.Println("  --json     sonuçları tek satır JSON olarak basar")
	fmt.Println("  --ayrinti  başarılı testlerin çıktısını da gösterir")
}

func testKomutu(args []string) {
	liste, jsonCikti, ayrinti := false, false, false
	var arglar []string
	for _, a := range args {
		switch a {
		case "--liste":
			liste = true
		case "--json":
			jsonCikti = true
		case "--ayrinti":
			ayrinti = true
		case "--yardim", "-h":
			testKullanim()
			return
		default:
			arglar = append(arglar, a)
		}
	}

	calisma, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tan test: çalışma dizini okunamadı: %v\n", err)
		os.Exit(2)
	}

	var testler []testKaydi
	if len(arglar) > 0 {
		görülen := map[string]bool{}
		for _, a := range arglar {
			bilgi, err := os.Stat(a)
			if err != nil {
				fmt.Fprintf(os.Stderr, "tan test: '%s' bulunamadı\n", a)
				os.Exit(2)
			}
			var dosyalar []string
			if bilgi.IsDir() {
				dosyalar, _ = filepath.Glob(filepath.Join(a, "*.tan"))
				sort.Strings(dosyalar)
			} else {
				dosyalar = []string{a}
			}
			for _, d := range dosyalar {
				t, ok := testBul(d)
				if !ok || görülen[t.Mutlak] {
					continue
				}
				görülen[t.Mutlak] = true
				if gor, err := filepath.Rel(calisma, t.Mutlak); err == nil && !strings.HasPrefix(gor, "..") {
					t.Dosya = gor
				}
				testler = append(testler, t)
			}
		}
	} else {
		testler = testleriKesfet(calisma)
	}

	if len(testler) == 0 {
		if jsonCikti {
			fmt.Println("[]")
		} else {
			fmt.Println("tan test: test bulunamadı")
		}
		return
	}

	if liste {
		for _, t := range testler {
			fmt.Printf("[%s] %s\n", t.Tur, t.Dosya)
		}
		return
	}

	sonuclar := make([]testSonuc, 0, len(testler))
	for _, t := range testler {
		sonuclar = append(sonuclar, testiCalistir(t))
	}

	if jsonCikti {
		b, err := json.Marshal(sonuclar)
		if err == nil {
			fmt.Println(string(b))
		}
		gecti := 0
		for _, s := range sonuclar {
			if s.Gecti {
				gecti++
			}
		}
		if gecti != len(sonuclar) {
			os.Exit(1)
		}
		return
	}

	fmt.Printf("=== tan test: %d dosya ===\n", len(testler))
	gecti, kaldi := raporYaz(sonuclar, ayrinti)
	fmt.Printf("=== SONUÇ: %d gecti, %d kaldi ===\n", gecti, kaldi)
	if kaldi > 0 {
		os.Exit(1)
	}
}
