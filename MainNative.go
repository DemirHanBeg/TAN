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

	// 3c. asm komutu: tan asm program.tan çıktı  (Tan -> x86-64 asm -> as/ld, C YOK)
	if os.Args[1] == "asm" {
		if len(os.Args) < 4 {
			fmt.Println("Kullanım: tan asm <program.tan> <çıktı-binary>")
			os.Exit(1)
		}
		derleAsm(os.Args[2], os.Args[3], true)
		return
	}

	// 3b. derle komutu:  tan derle program.tan çıktı   (Tan -> C -> native binary)
	if os.Args[1] == "derle" {
		if len(os.Args) < 4 {
			fmt.Println("Kullanım: tan derle <program.tan> <çıktı-binary>")
			os.Exit(1)
		}
		derleC(os.Args[2], os.Args[3])
		return
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
	y := YeniYorumlayici()
	if mutlak, err := filepath.Abs(dosya); err == nil {
		y.kaynakDizin = filepath.Dir(mutlak)
	}
	if kaynagiCalistir(y, string(kaynak)) {
		os.Exit(1)
	}
}
