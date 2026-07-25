package main

import "fmt"

// TanHata: çalışma zamanı hatası. Satır bilgisi varsa gösterir.
// Sessizce nil dönmek yerine bunu "fırlat" ile fırlatıp, üst
// katmanda yakalayıp temiz bir mesaj basıyoruz.
type TanHata struct {
	Satir int
	Mesaj string
}

func (h TanHata) Error() string {
	if h.Satir > 0 {
		return fmt.Sprintf("HATA (satır %d): %s", h.Satir, h.Mesaj)
	}
	return "HATA: " + h.Mesaj
}

// firlat: biçimli bir çalışma zamanı hatası fırlatır.
func firlat(satir int, bicim string, args ...interface{}) {
	panic(TanHata{Satir: satir, Mesaj: fmt.Sprintf(bicim, args...)})
}
