//go:build !js

package main

import (
	"fmt"
	"path/filepath"
	"os"
	"strings"
)

func main() {
	// 1. Bu binary'ye gömülü bir program var mı? Varsa onu çalıştır.
	//    (paketlenmiş uygulama modu — tek dosya, Go gerekmez)
	if kaynak, gomulu := gomuluKaynagiOku(); gomulu {
		y := YeniYorumlayici()
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

	// 3b. derle komutu: ARŞİVLENDİ (self-hosting sonrası, bkz. arsiv/DerleC.go).
	// Kaynak dış-araç bağımlı Kademe 1 arka ucuydu (gcc gerektiriyordu);
	// "tan elf" onu gereksiz kıldı.
	if os.Args[1] == "derle" {
		fmt.Println("tan derle arşivlendi — bkz. arsiv/DerleC.go. Yerine 'tan elf' kullanın.")
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
	}
	if kaynagiCalistir(y, string(kaynak)) {
		os.Exit(1)
	}
}
