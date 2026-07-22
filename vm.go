package main

import "fmt"

// ============================================================
// Sanal Makine (VM) — bytecode'u yığın üstünde çalıştırır.
// Ağaç gezmek yerine düz komut dizisini işler: çok daha hızlı.
// ============================================================

type SanalMakine struct {
	kod      *DerlenmisKod
	yigin    []Deger
	degerler map[string]Deger
}

func YeniSanalMakine(kod *DerlenmisKod) *SanalMakine {
	return &SanalMakine{
		kod:      kod,
		yigin:    make([]Deger, 0, 256),
		degerler: map[string]Deger{},
	}
}

func (vm *SanalMakine) it(d Deger) { vm.yigin = append(vm.yigin, d) }
func (vm *SanalMakine) al() Deger {
	n := len(vm.yigin)
	d := vm.yigin[n-1]
	vm.yigin = vm.yigin[:n-1]
	return d
}

// cerceve: bir çağrı çerçevesi (call frame). Her işlev çağrısı kendi
// bytecode'unu (kod), kendi komut işaretçisini (ip) ve kendi yerel
// değişken map'ini (degerler) taşır — özyinelemede her çağrı katmanı
// kendi kapsamını kullanır, birbirine karışmaz.
type cerceve struct {
	kod      *DerlenmisKod
	ip       int
	degerler map[string]Deger
}

func (vm *SanalMakine) Calistir() {
	// En dıştaki çerçeve: giriş programı, global değişken map'iyle.
	cercereler := []*cerceve{{kod: vm.kod, ip: 0, degerler: vm.degerler}}

	for len(cercereler) > 0 {
		f := cercereler[len(cercereler)-1]
		komutlar := f.kod.Komutlar
		if f.ip >= len(komutlar) {
			// Gövde OP_DONDUR olmadan bitti (olmamalı — güvenlik ağı var).
			cercereler = cercereler[:len(cercereler)-1]
			if len(cercereler) == 0 {
				return
			}
			vm.it(nil)
			continue
		}
		k := komutlar[f.ip]
		switch k.Op {
		case OP_SABIT:
			vm.it(f.kod.Sabitler[k.Operand])
		case OP_DEGISKEN_OKU:
			ad := f.kod.AdlarHavuz[k.Operand]
			if d, ok := f.degerler[ad]; ok {
				vm.it(d)
			} else {
				vm.it(vm.degerler[ad]) // yerelde yoksa global'e düş (işlevler burada tanımlı)
			}
		case OP_DEGISKEN_YAZ:
			ad := f.kod.AdlarHavuz[k.Operand]
			f.degerler[ad] = vm.al()
		case OP_ISLEV_TANIM:
			vm.it(f.kod.Sabitler[k.Operand])
		case OP_CAGIR:
			argSayisi := k.Operand
			args := make([]Deger, argSayisi)
			for i := argSayisi - 1; i >= 0; i-- {
				args[i] = vm.al()
			}
			islevDeger := vm.al()
			islev, ok := islevDeger.(*VMIslev)
			if !ok {
				firlat(0, "çağrılan değer bir işlev değil")
			}
			yeniDegerler := map[string]Deger{}
			for i, p := range islev.Parametreler {
				if i < len(args) {
					yeniDegerler[p] = args[i]
				}
			}
			f.ip++ // dönüş adresi: çağıranın bir sonraki komutu
			cercereler = append(cercereler, &cerceve{kod: islev.Kod, ip: 0, degerler: yeniDegerler})
			continue
		case OP_DONDUR:
			donusDeger := vm.al()
			cercereler = cercereler[:len(cercereler)-1]
			if len(cercereler) == 0 {
				return
			}
			vm.it(donusDeger)
			continue
		case OP_TOPLA:
			b := vm.al()
			a := vm.al()
			vm.it(vmTopla(a, b))
		case OP_CIKAR:
			b := vm.al()
			a := vm.al()
			sonuc, sifir := sayiIslem("-", a, b)
			if sifir {
				firlat(0, "sıfıra bölme")
			}
			vm.it(sonuc)
		case OP_CARP:
			b := vm.al()
			a := vm.al()
			sonuc, sifir := sayiIslem("*", a, b)
			if sifir {
				firlat(0, "sıfıra bölme")
			}
			vm.it(sonuc)
		case OP_BOL:
			b := vm.al()
			a := vm.al()
			sonuc, sifir := sayiIslem("/", a, b)
			if sifir {
				firlat(0, "sıfıra bölme")
			}
			vm.it(sonuc)
		case OP_MOD:
			b := vm.al()
			a := vm.al()
			sonuc, sifir := sayiIslem("%", a, b)
			if sifir {
				firlat(0, "sıfıra bölme")
			}
			vm.it(sonuc)
		case OP_ESIT:
			b := vm.al()
			a := vm.al()
			vm.it(vmEsit(a, b))
		case OP_ESIT_DEGIL:
			b := vm.al()
			a := vm.al()
			vm.it(!vmEsit(a, b))
		case OP_BUYUK:
			b := vm.al()
			a := vm.al()
			sonuc, tamam := sayiKarsilastir(">", a, b)
			if !tamam {
				vm.it(false)
			} else {
				vm.it(sonuc)
			}
		case OP_KUCUK:
			b := vm.al()
			a := vm.al()
			sonuc, tamam := sayiKarsilastir("<", a, b)
			if !tamam {
				vm.it(false)
			} else {
				vm.it(sonuc)
			}
		case OP_BUYUK_ESIT:
			b := vm.al()
			a := vm.al()
			sonuc, tamam := sayiKarsilastir(">=", a, b)
			if !tamam {
				vm.it(false)
			} else {
				vm.it(sonuc)
			}
		case OP_KUCUK_ESIT:
			b := vm.al()
			a := vm.al()
			sonuc, tamam := sayiKarsilastir("<=", a, b)
			if !tamam {
				vm.it(false)
			} else {
				vm.it(sonuc)
			}
		case OP_VE:
			b := vm.al()
			a := vm.al()
			vm.it(vmDogru(a) && vmDogru(b))
		case OP_VEYA:
			b := vm.al()
			a := vm.al()
			vm.it(vmDogru(a) || vmDogru(b))
		case OP_DEGIL:
			vm.it(!vmDogru(vm.al()))
		case OP_NEGATIF:
			nd := vm.al()
			if i, ok := nd.(int64); ok {
				vm.it(-i)
			} else if f, ok := nd.(float64); ok {
				vm.it(-f)
			} else {
				vm.it(nil)
			}
		case OP_YAZDIR:
			fmt.Fprintln(Cikti, metne(vm.al()))
		case OP_ATLA:
			f.ip = k.Operand
			continue
		case OP_YANLISSA_ATLA:
			if !vmDogru(vm.al()) {
				f.ip = k.Operand
				continue
			}
		case OP_AT:
			vm.al()
		case OP_DUR_PROGRAM:
			return
		}
		f.ip++
	}
}

// VM yardımcıları
func vmDogru(d Deger) bool {
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

func vmEsit(a, b Deger) bool {
	if sayiMi(a) && sayiMi(b) {
		if sonuc, tamam := sayiKarsilastir("==", a, b); tamam {
			if bv, ok := sonuc.(bool); ok {
				return bv
			}
		}
	}
	return metne(a) == metne(b)
}

// vmTopla: sayı+sayı veya metin birleştirme
func vmTopla(a, b Deger) Deger {
	if sayiMi(a) && sayiMi(b) {
		sonuc, _ := sayiIslem("+", a, b)
		if sonuc != nil {
			return sonuc
		}
	}
	// en az biri metin -> birleştir
	return metne(a) + metne(b)
}
