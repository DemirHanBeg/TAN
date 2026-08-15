//go:build !js

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 1. Bu binary'ye gömülü bir program var mı? Varsa onu çalıştır.
	//    (paketlenmiş uygulama modu — tek dosya, Go gerekmez)
	if kaynak, gomulu := gomuluKaynagiOku(); gomulu {
		y := YeniYorumlayici()
		y.kaynakDosya = "gomulu"
		if kaynagiCalistir(y, kaynak) {
			os.Exit(1)
		}
		return
	}

	// 2. Argüman yok → REPL
	if len(os.Args) < 2 {
		repl()
		return
	}

	// 3. paketle komutu:  tan paketle program.tan çıktı
	if os.Args[1] == "paketle" {
		if len(os.Args) < 4 {
			fmt.Println("Kullanım: tan paketle <program.tan> <çıktı-dosyası>")
			os.Exit(1)
		}
		paketle(os.Args[2], os.Args[3])
		return
	}

	// 3a. paket komutu: tan paket <alt-komut>
	if os.Args[1] == "paket" {
		paketKomutu(os.Args[2:])
		return
	}

	// 3e. test komutu: tan test [--liste|--json|--ayrinti] [dosya|dizin ...]
	if os.Args[1] == "test" {
		testKomutu(os.Args[2:])
		return
	}

	// 3f. biçimlendir komutu: tan biçimlendir [--denet|--cikti] <dosya...>
	if os.Args[1] == "biçimlendir" {
		bicimlendirKomutu(os.Args[2:])
		return
	}

	// 3g. denetle komutu: tan denetle [--json] <dosya...>
	if os.Args[1] == "denetle" {
		denetleKomutu(os.Args[2:])
		return
	}

	// 3h. derle komutu: tan derle <program.tan> [-o <çıktı>]  (ELF, sıfır dış araç)
	// Eski gcc tabanlı derle'nin (arsiv/DerleC.go) yerini alır; bkz. Derle.go.
	if os.Args[1] == "derle" {
		derleKomutu(os.Args[2:])
		return
	}

	// 3i. simgeler komutu: tan simgeler [--json] <dosya...>  (sembol envanteri)
	if os.Args[1] == "simgeler" {
		simgelerKomutu(os.Args[2:])
		return
	}

	// 3j. bagimlilik komutu: tan bagimlilik [--json] <dosya|dizin...>  (modül grafiği)
	if os.Args[1] == "bagimlilik" {
		bagimlilikKomutu(os.Args[2:])
		return
	}

	// 3k. api komutu: tan api [--json] <dosya...>  (imzalı dışa-açık yüzey)
	if os.Args[1] == "api" {
		apiKomutu(os.Args[2:])
		return
	}

	// 3l. ast komutu: tan ast [--json] <dosya...>  (ayrıştırma ağacı)
	if os.Args[1] == "ast" {
		astKomutu(os.Args[2:])
		return
	}

	// 3m. incele komutu: tan incele [--json] <dosya|dizin...>  (ölçüm profili)
	if os.Args[1] == "incele" {
		inceleKomutu(os.Args[2:])
		return
	}

	// 3n. tanı komutu: tan tanı [--json] <dosya|dizin...>  (tanı raporları)
	if os.Args[1] == "tanı" {
		taniKomutu(os.Args[2:])
		return
	}

	// 3o. etki komutu: tan etki [--json] <dosya> <sembol>  (çağrı grafiği)
	if os.Args[1] == "etki" {
		etkiKomutu(os.Args[2:])
		return
	}

	// 3d. elf komutu: tan elf program.tan çıktı  (makine kodu + ELF, SIFIR dış araç)
	if os.Args[1] == "elf" {
		if len(os.Args) < 4 {
			fmt.Println("Kullanım: tan elf <program.tan> <çıktı-binary>")
			os.Exit(1)
		}
		derleElf(os.Args[2], os.Args[3])
		return
	}

	// 3c. asm komutu: ARŞİVLENDİ (self-hosting sonrası, bkz. arsiv/DerleAsm.go).
	// Kaynak dış-araç bağımlı Kademe 2 arka ucuydu; "tan elf" (sıfır dış araç,
	// self-hosted TancElf.tan tarafından da üretiliyor) onu gereksiz kıldı.
	if os.Args[1] == "asm" {
		fmt.Println("tan asm arşivlendi — bkz. arsiv/DerleAsm.go. Yerine 'tan elf' kullanın.")
		os.Exit(1)
	}

	// 4. Normal dosya çalıştırma
	dosya := os.Args[1]
	if !strings.HasSuffix(dosya, ".tan") {
		fmt.Println("Uyarı: Tan dosyaları .tan uzantılı olmalı")
	}
	kaynak, err := os.ReadFile(dosya)
	if err != nil {
		fmt.Printf("Dosya okunamadı: %v\n", err)
		os.Exit(1)
	}
	tanScriptArgs = os.Args[1:]
	y := YeniYorumlayici()
	if mutlak, err := filepath.Abs(dosya); err == nil {
		y.kaynakDizin = filepath.Dir(mutlak)
		y.kaynakDosya = mutlak
	} else {
		y.kaynakDosya = dosya
	}
	if kaynagiCalistir(y, string(kaynak)) {
		os.Exit(1)
	}
}
