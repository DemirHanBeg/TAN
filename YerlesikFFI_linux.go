//go:build linux

package main

// FFI / dış kütüphane (Faz A #3 — bkz. NexusCore/FazA_Kapsam.md).
// github.com/ebitengine/purego sarılıyor (cgo YOK — bu geliştirme
// ortamında Linux'a çapraz derleme için çalışan bir cgo C derleyicisi
// yok, purego pure-Go assembly trampolinleriyle dlopen/dlsym/çağrıyı
// cgo'suz yapıyor — kendi kriptonu yazma tuzağının FFI karşılığına
// düşülmedi, kanıtlanmış/gerçek dünyada kullanılan bir kütüphane sarıldı).
//
// SADECE linux derlemesinde var (purego'nun Dlopen/Dlsym/Dlclose API'si
// unix-benzeri sistemlere özgü, Windows'ta bu isimler tanımsız — ayrı
// dosyada olması TAM BU YÜZDEN: Windows derlemesi (tan.exe) bu dosyayı
// hiç görmüyor, YerlesikFFI_diger.go'daki "desteklenmiyor" saplamalarını
// kullanıyor). SINIR: disIslevCagir() argümanları yalnız tam sayı/metin/
// mantıksal/yok kabul eder (SyscallN tam sayı registerlarıyla çağırıyor)
// — KAYAN NOKTALI C parametreleri desteklenmiyor (xmm register'ları ayrı
// bir çağrı yolu ister, kapsam dışı).

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

func init() {
	yerlesikler["disKutuphaneAc"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "disKutuphaneAc(yol) bir argüman ister")
		}
		tutamac, err := purego.Dlopen(metne(a[0]), purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			firlat(satir, "disKutuphaneAc(): %v", err)
		}
		return &TanDisKutuphane{Tutamac: tutamac}
	}

	yerlesikler["disIslevBul"] = func(a []Deger, satir int) Deger {
		if len(a) < 2 {
			firlat(satir, "disIslevBul(kutuphane, ad) iki argüman ister")
		}
		k, ok := a[0].(*TanDisKutuphane)
		if !ok {
			firlat(satir, "disIslevBul() bir dış kütüphane (disKutuphaneAc sonucu) bekliyor")
		}
		gosterici, err := purego.Dlsym(k.Tutamac, metne(a[1]))
		if err != nil {
			firlat(satir, "disIslevBul(): %v", err)
		}
		return &TanDisIslev{Gosterici: gosterici}
	}

	yerlesikler["disIslevCagir"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "disIslevCagir(islev, ...args) en az bir argüman ister")
		}
		islev, ok := a[0].(*TanDisIslev)
		if !ok {
			firlat(satir, "disIslevCagir() bir dış işlev (disIslevBul sonucu) bekliyor")
		}
		var canliTutucular [][]byte
		args := make([]uintptr, 0, len(a)-1)
		for _, d := range a[1:] {
			switch v := d.(type) {
			case int64:
				args = append(args, uintptr(v))
			case bool:
				if v {
					args = append(args, 1)
				} else {
					args = append(args, 0)
				}
			case nil:
				args = append(args, 0)
			case string:
				tampon := append([]byte(v), 0)
				canliTutucular = append(canliTutucular, tampon)
				args = append(args, uintptr(unsafe.Pointer(&tampon[0])))
			default:
				firlat(satir, "disIslevCagir(): desteklenmeyen argüman tipi (yalnız tam sayı/metin/mantıksal/yok)")
			}
		}
		r1, _, _ := purego.SyscallN(islev.Gosterici, args...)
		runtime.KeepAlive(canliTutucular)
		return int64(r1)
	}

	yerlesikler["disKutuphaneKapat"] = func(a []Deger, satir int) Deger {
		if len(a) < 1 {
			firlat(satir, "disKutuphaneKapat() bir argüman ister")
		}
		k, ok := a[0].(*TanDisKutuphane)
		if !ok {
			firlat(satir, "disKutuphaneKapat() bir dış kütüphane bekliyor")
		}
		if err := purego.Dlclose(k.Tutamac); err != nil {
			firlat(satir, "disKutuphaneKapat(): %v", err)
		}
		return int64(0)
	}
}
