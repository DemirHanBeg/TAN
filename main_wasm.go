//go:build js

package main

import (
	"bytes"
	"fmt"
	"syscall/js"
)

// tanCalistir: tarayıcıdan çağrılır. Verilen Tan kaynağını çalıştırır,
// tüm çıktıyı (ve hataları) metin olarak döndürür.
func tanCalistirJS(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Hata: kaynak verilmedi"
	}
	kaynak := args[0].String()

	// Çıktıyı tampona yönlendir
	var tampon bytes.Buffer
	Cikti = &tampon

	// Hata olursa yakala
	sonuc := func() (cikti string) {
		defer func() {
			if r := recover(); r != nil {
				if h, ok := r.(TanHata); ok {
					tampon.WriteString("\n" + h.Error())
				} else if _, ok := r.(vmDesteklemiyor); ok {
					// yok say — ağaç-gezene düşülür (aşağıda ele alınır)
				} else {
					tampon.WriteString(fmt.Sprintf("\nBeklenmedik hata: %v", r))
				}
			}
			cikti = tampon.String()
		}()

		lexer := YeniLexer(kaynak)
		parser := YeniParser(lexer.Tokenle())
		agac := parser.Ayristir()

		// Önce VM'i dene, desteklemezse ağaç-gezen
		if !vmDeneCalistir(agac) {
			y := YeniYorumlayici()
			y.Calistir(agac)
		}
		return tampon.String()
	}()

	return sonuc
}

func main() {
	// tanCalistir adını tarayıcıya (JS global) tanıt
	js.Global().Set("tanCalistir", js.FuncOf(tanCalistirJS))
	fmt.Println("Tan WASM hazır — tanCalistir(kaynak) çağrılabilir")
	// Programı canlı tut (JS çağrılarını beklesin)
	select {}
}
