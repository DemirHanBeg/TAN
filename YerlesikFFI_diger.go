//go:build !linux

package main

// FFI yerleşiklerinin linux-DIŞI platformlar (ör. Windows native tan.exe
// derlemesi) için saplamaları — bkz. YerlesikFFI_linux.go'daki asıl
// gerekçe notu. purego'nun dlopen/dlsym API'si unix-benzeri sistemlere
// özgü olduğundan burada gerçek bir implementasyon YOK, sadece açık/
// anlaşılır bir hata veriliyor (sessizce eksik kalmak yerine).

func init() {
	yerlesikler["disKutuphaneAc"] = func(a []Deger, satir int) Deger {
		firlat(satir, "disKutuphaneAc(): FFI şu an sadece Linux derlemesinde (elf/WSL yolu) destekleniyor")
		return nil
	}
	yerlesikler["disIslevBul"] = func(a []Deger, satir int) Deger {
		firlat(satir, "disIslevBul(): FFI şu an sadece Linux derlemesinde (elf/WSL yolu) destekleniyor")
		return nil
	}
	yerlesikler["disIslevCagir"] = func(a []Deger, satir int) Deger {
		firlat(satir, "disIslevCagir(): FFI şu an sadece Linux derlemesinde (elf/WSL yolu) destekleniyor")
		return nil
	}
	yerlesikler["disKutuphaneKapat"] = func(a []Deger, satir int) Deger {
		firlat(satir, "disKutuphaneKapat(): FFI şu an sadece Linux derlemesinde (elf/WSL yolu) destekleniyor")
		return nil
	}
}
