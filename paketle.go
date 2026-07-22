package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// Paketlenmiş dosyanın en SONUNA yazılan yapı:
//
//	[motor][kaynak][kaynak-uzunluğu: 8 bayt][sihir]
//
// Sadece dosyanın SONUNA bakılır; bu yüzden yorumlayıcının kendi
// içindeki sabitler yanlış eşleşme yaratmaz.
var sihir = []byte("TANPAKET1")

// gomuluKaynagiOku: çalışan binary'nin sonunda gömülü program var mı?
func gomuluKaynagiOku() (string, bool) {
	yol, err := os.Executable()
	if err != nil {
		return "", false
	}
	veri, err := os.ReadFile(yol)
	if err != nil {
		return "", false
	}
	n := len(veri)
	kuyrukBoyu := 8 + len(sihir)
	if n < kuyrukBoyu {
		return "", false
	}
	if !bytes.Equal(veri[n-len(sihir):], sihir) {
		return "", false
	}
	uzunlukBaslangic := n - kuyrukBoyu
	uzunluk := binary.BigEndian.Uint64(veri[uzunlukBaslangic : uzunlukBaslangic+8])
	kaynakBaslangic := uzunlukBaslangic - int(uzunluk)
	if kaynakBaslangic < 0 {
		return "", false
	}
	return string(veri[kaynakBaslangic:uzunlukBaslangic]), true
}

// motorKismi: gömülü kuyruk varsa temiz motoru döndürür
func motorKismi(veri []byte) []byte {
	n := len(veri)
	kuyrukBoyu := 8 + len(sihir)
	if n < kuyrukBoyu || !bytes.Equal(veri[n-len(sihir):], sihir) {
		return veri
	}
	uzunlukBaslangic := n - kuyrukBoyu
	uzunluk := binary.BigEndian.Uint64(veri[uzunlukBaslangic : uzunlukBaslangic+8])
	kaynakBaslangic := uzunlukBaslangic - int(uzunluk)
	if kaynakBaslangic < 0 {
		return veri
	}
	return veri[:kaynakBaslangic]
}

// paketle: bir .tan dosyasını yorumlayıcıyla birleştirip tek exe üretir
func paketle(tanDosya, ciktiDosya string) {
	kaynak, err := os.ReadFile(tanDosya)
	if err != nil {
		fmt.Printf("Kaynak okunamadı: %v\n", err)
		os.Exit(1)
	}
	kendiYol, err := os.Executable()
	if err != nil {
		fmt.Printf("Yorumlayıcı bulunamadı: %v\n", err)
		os.Exit(1)
	}
	motor, err := os.ReadFile(kendiYol)
	if err != nil {
		fmt.Printf("Yorumlayıcı okunamadı: %v\n", err)
		os.Exit(1)
	}
	motor = motorKismi(motor)

	var tampon bytes.Buffer
	tampon.Write(motor)
	tampon.Write(kaynak)
	uzunluk := make([]byte, 8)
	binary.BigEndian.PutUint64(uzunluk, uint64(len(kaynak)))
	tampon.Write(uzunluk)
	tampon.Write(sihir)

	if err := os.WriteFile(ciktiDosya, tampon.Bytes(), 0755); err != nil {
		fmt.Printf("Çıktı yazılamadı: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Paketlendi: %s  (%d bayt)\n", ciktiDosya, tampon.Len())
	fmt.Printf("Tek başına çalışır: ./%s\n", ciktiDosya)
}
