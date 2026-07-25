package main

// ============================================================
// TAN MODÜL SİSTEMİ
// ------------------------------------------------------------
// içe al "matematik"              -> modül adı ile (önerilen)
// içe al "kutuphane/Matematik.tan" -> eski göreli yol (uyumluluk)
//
// ARAMA SIRASI:
//   1. içe alan dosyanın dizini          ./matematik.tan
//   2. içe alan dosyanın kutuphane/      ./kutuphane/Matematik.tan
//   3. proje paket dizini                ./tan_moduller/matematik/matematik.tan
//   4. TAN_YOL ortam değişkeni           $TAN_YOL/matematik.tan
//   5. kullanıcı modülleri               ~/.tan/moduller/matematik/matematik.tan
//   6. standart kütüphane (tan yanında)  <exe dizini>/kutuphane/Matematik.tan
//
// Bir paket dizininde tan.json varsa, "giris" alanı okunur.
// ============================================================

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ModulBilgi: tan.json dosyasının şeması
type ModulBilgi struct {
	Ad       string            `json:"ad"`
	Surum    string            `json:"surum"`
	Giris    string            `json:"giris"`
	Aciklama string            `json:"aciklama,omitempty"`
	Lisans   string            `json:"lisans,omitempty"`
	Bagimli  map[string]string `json:"bagimliliklar,omitempty"`
}

// modulAra: bir modül adını gerçek dosya yoluna çevirir.
// kaynakDizin: içe al'ı yapan dosyanın bulunduğu dizin.
func modulAra(ad string, kaynakDizin string) (string, bool) {
	// .tan uzantısı verilmişse doğrudan yol kabul et (eski davranış)
	if strings.HasSuffix(ad, ".tan") {
		adaylar := []string{
			filepath.Join(kaynakDizin, ad),
			ad, // çalışma dizinine göre
		}
		for _, a := range adaylar {
			if dosyaVar(a) {
				return a, true
			}
		}
		return "", false
	}

	// Modül adı ile arama
	var adaylar []string

	// 1-2. içe alan dosyanın yanı ve kutuphane/ alt dizini
	adaylar = append(adaylar,
		filepath.Join(kaynakDizin, ad+".tan"),
		filepath.Join(kaynakDizin, "kutuphane", ad+".tan"),
	)

	// 3. proje paket dizini
	if p, ok := paketGirisi(filepath.Join("tan_moduller", ad)); ok {
		adaylar = append(adaylar, p)
	}
	adaylar = append(adaylar, filepath.Join("tan_moduller", ad, ad+".tan"))

	// 4. TAN_YOL
	if yol := os.Getenv("TAN_YOL"); yol != "" {
		for _, parca := range filepath.SplitList(yol) {
			adaylar = append(adaylar,
				filepath.Join(parca, ad+".tan"),
				filepath.Join(parca, ad, ad+".tan"),
			)
		}
	}

	// 5. kullanıcı modülleri
	if ev, err := os.UserHomeDir(); err == nil {
		kok := filepath.Join(ev, ".tan", "moduller", ad)
		if p, ok := paketGirisi(kok); ok {
			adaylar = append(adaylar, p)
		}
		adaylar = append(adaylar, filepath.Join(kok, ad+".tan"))
	}

	// 6. standart kütüphane: tan binary'sinin yanı
	if exe, err := os.Executable(); err == nil {
		dizin := filepath.Dir(exe)
		adaylar = append(adaylar,
			filepath.Join(dizin, "kutuphane", ad+".tan"),
			filepath.Join(dizin, ad+".tan"),
		)
	}

	// çalışma dizinindeki kutuphane/
	adaylar = append(adaylar, filepath.Join("kutuphane", ad+".tan"))

	for _, a := range adaylar {
		if dosyaVar(a) {
			return a, true
		}
	}
	return "", false
}

// paketGirisi: bir paket dizininde tan.json varsa giriş dosyasını döndürür.
func paketGirisi(dizin string) (string, bool) {
	manifest := filepath.Join(dizin, "tan.json")
	veri, err := os.ReadFile(manifest)
	if err != nil {
		return "", false
	}
	var bilgi ModulBilgi
	if err := json.Unmarshal(veri, &bilgi); err != nil {
		return "", false
	}
	if bilgi.Giris == "" {
		return "", false
	}
	yol := filepath.Join(dizin, bilgi.Giris)
	if dosyaVar(yol) {
		return yol, true
	}
	return "", false
}

func dosyaVar(yol string) bool {
	bilgi, err := os.Stat(yol)
	return err == nil && !bilgi.IsDir()
}

// modulAramaYollari: hata mesajında kullanıcıya nereye baktığımızı söyler.
func modulAramaYollari(ad string, kaynakDizin string) string {
	var b strings.Builder
	b.WriteString("aranan yerler:\n")
	b.WriteString("  " + filepath.Join(kaynakDizin, ad+".tan") + "\n")
	b.WriteString("  " + filepath.Join(kaynakDizin, "kutuphane", ad+".tan") + "\n")
	b.WriteString("  " + filepath.Join("tan_moduller", ad, ad+".tan") + "\n")
	if yol := os.Getenv("TAN_YOL"); yol != "" {
		b.WriteString("  $TAN_YOL: " + yol + "\n")
	} else {
		b.WriteString("  ($TAN_YOL tanımlı değil)\n")
	}
	if ev, err := os.UserHomeDir(); err == nil {
		b.WriteString("  " + filepath.Join(ev, ".tan", "moduller", ad) + "\n")
	}
	b.WriteString("  <tan binary dizini>/kutuphane/" + ad + ".tan")
	return b.String()
}
