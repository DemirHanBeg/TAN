//go:build !js

package main

// ============================================================
// TAN PAKET YÖNETİCİSİ
// ------------------------------------------------------------
//   tan paket başlat              tan.json oluşturur
//   tan paket kur <url|ad>        modülü tan_moduller/ altına kurar
//   tan paket kur                 tan.json'daki bağımlılıkları kurar
//   tan paket listele             kurulu modülleri gösterir
//   tan paket sil <ad>            modülü kaldırır
//
// Depo biçimi: git deposu. İçinde tan.json olmalı.
// Sürümleme: semver (BÜYÜK.KÜÇÜK.YAMA), git etiketi ile eşlenir.
// ============================================================

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const paketDizin = "tan_moduller"

func paketKomutu(args []string) {
	if len(args) == 0 {
		paketYardim()
		return
	}
	switch args[0] {
	case "başlat", "baslat", "init":
		paketBaslat()
	case "kur", "install":
		if len(args) < 2 {
			paketBagimliliklariKur()
		} else {
			paketKur(args[1])
		}
	case "listele", "list":
		paketListele()
	case "sil", "remove":
		if len(args) < 2 {
			fmt.Println("Kullanım: tan paket sil <ad>")
			os.Exit(1)
		}
		paketSil(args[1])
	default:
		paketYardim()
	}
}

func paketYardim() {
	fmt.Println(`Tan paket yöneticisi

  tan paket başlat           tan.json oluşturur
  tan paket kur <url|ad>     modül kurar (git deposu)
  tan paket kur              tan.json'daki bağımlılıkları kurar
  tan paket listele          kurulu modülleri listeler
  tan paket sil <ad>         modülü kaldırır

Örnek:
  tan paket kur https://github.com/kullanici/tan-json
  tan paket kur github.com/kullanici/tan-json`)
}

func paketBaslat() {
	if dosyaVar("tan.json") {
		fmt.Println("tan.json zaten var.")
		return
	}
	dizin, _ := os.Getwd()
	bilgi := ModulBilgi{
		Ad:       filepath.Base(dizin),
		Surum:    "0.1.0",
		Giris:    "ana.tan",
		Aciklama: "",
		Lisans:   "MIT",
		Bagimli:  map[string]string{},
	}
	veri, _ := json.MarshalIndent(bilgi, "", "  ")
	if err := os.WriteFile("tan.json", append(veri, '\n'), 0644); err != nil {
		fmt.Printf("tan.json yazılamadı: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("tan.json oluşturuldu.")
	fmt.Println("Sürümleme: semver kullan — BÜYÜK.KÜÇÜK.YAMA")
	fmt.Println("  BÜYÜK: geriye uyumsuz değişiklik")
	fmt.Println("  KÜÇÜK: geriye uyumlu yeni özellik")
	fmt.Println("  YAMA : geriye uyumlu hata düzeltmesi")
}

// urlNormalle: "github.com/a/b" -> "https://github.com/a/b"
func urlNormalle(u string) string {
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") ||
		strings.HasPrefix(u, "git@") {
		return u
	}
	return "https://" + u
}

func paketAdiCikar(u string) string {
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")
	parcalar := strings.Split(u, "/")
	ad := parcalar[len(parcalar)-1]
	// tan- öneki varsa at: tan-json -> json
	ad = strings.TrimPrefix(ad, "tan-")
	return ad
}

func paketKur(hedef string) {
	url := urlNormalle(hedef)
	ad := paketAdiCikar(hedef)
	varis := filepath.Join(paketDizin, ad)

	if err := os.MkdirAll(paketDizin, 0755); err != nil {
		fmt.Printf("dizin oluşturulamadı: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(varis); err == nil {
		fmt.Printf("%s zaten kurulu. Güncellemek için önce sil.\n", ad)
		return
	}

	fmt.Printf("kuruluyor: %s -> %s\n", url, varis)
	komut := exec.Command("git", "clone", "--depth", "1", "--quiet", url, varis)
	komut.Stdout = os.Stdout
	komut.Stderr = os.Stderr
	if err := komut.Run(); err != nil {
		fmt.Printf("kurulum başarısız: %v\n", err)
		fmt.Println("git kurulu mu? URL doğru mu?")
		os.Exit(1)
	}
	// .git dizinini at (paket kaynağı, depo değil)
	os.RemoveAll(filepath.Join(varis, ".git"))

	// manifest doğrula
	if bilgi, ok := paketBilgisiOku(varis); ok {
		fmt.Printf("kuruldu: %s v%s\n", bilgi.Ad, bilgi.Surum)
		if bilgi.Giris != "" {
			fmt.Printf("  giriş: %s\n", bilgi.Giris)
		}
		fmt.Printf("  kullanım: içe al \"%s\"\n", ad)
		// bağımlılıklarını da kur
		for badi, surum := range bilgi.Bagimli {
			hedefDizin := filepath.Join(paketDizin, paketAdiCikar(badi))
			if _, err := os.Stat(hedefDizin); err != nil {
				fmt.Printf("  bağımlılık: %s (%s)\n", badi, surum)
				paketKur(badi)
			}
		}
	} else {
		fmt.Printf("kuruldu: %s  (uyarı: tan.json yok, modül adıyla içe alınamayabilir)\n", ad)
	}
}

func paketBagimliliklariKur() {
	bilgi, ok := paketBilgisiOku(".")
	if !ok {
		fmt.Println("tan.json bulunamadı. Önce: tan paket başlat")
		os.Exit(1)
	}
	if len(bilgi.Bagimli) == 0 {
		fmt.Println("bağımlılık yok.")
		return
	}
	for ad, surum := range bilgi.Bagimli {
		fmt.Printf("-> %s (%s)\n", ad, surum)
		paketKur(ad)
	}
}

func paketListele() {
	girisler, err := os.ReadDir(paketDizin)
	if err != nil {
		fmt.Println("kurulu modül yok. (tan_moduller/ dizini yok)")
		return
	}
	if len(girisler) == 0 {
		fmt.Println("kurulu modül yok.")
		return
	}
	fmt.Println("Kurulu modüller:")
	for _, g := range girisler {
		if !g.IsDir() {
			continue
		}
		yol := filepath.Join(paketDizin, g.Name())
		if bilgi, ok := paketBilgisiOku(yol); ok {
			aciklama := bilgi.Aciklama
			if aciklama != "" {
				aciklama = "  — " + aciklama
			}
			fmt.Printf("  %-20s v%-10s%s\n", g.Name(), bilgi.Surum, aciklama)
		} else {
			fmt.Printf("  %-20s (tan.json yok)\n", g.Name())
		}
	}
}

func paketSil(ad string) {
	yol := filepath.Join(paketDizin, ad)
	if _, err := os.Stat(yol); err != nil {
		fmt.Printf("%s kurulu değil.\n", ad)
		return
	}
	if err := os.RemoveAll(yol); err != nil {
		fmt.Printf("silinemedi: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("silindi: %s\n", ad)
}

func paketBilgisiOku(dizin string) (ModulBilgi, bool) {
	veri, err := os.ReadFile(filepath.Join(dizin, "tan.json"))
	if err != nil {
		return ModulBilgi{}, false
	}
	var bilgi ModulBilgi
	if err := json.Unmarshal(veri, &bilgi); err != nil {
		return ModulBilgi{}, false
	}
	return bilgi, true
}
