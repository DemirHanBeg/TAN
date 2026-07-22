package main

// ============================================================
// KÖPRÜ KATMANI (A seçeneği) — İSKELE KURALI
// ------------------------------------------------------------
// Bu katman GEÇİCİDİR. Tan'ın çekirdeği (lexer/parser/yorumlayıcı)
// hiçbir dış şeye muhtaç değildir. Buradaki her yetenek, ileride
// saf Tan ile yeniden yazılıp bu dosyadan SİLİNECEK.
// Her köprü kaydı "iskele: true" ile işaretli — sökülebilir olduğu
// unutulmasın diye. Çekirdek kalır, iskele sökülür.
// ============================================================

type KopruYetenek struct {
	Islev    func(args []Deger) Deger
	Iskele   bool // her zaman true: bu geçici bir dolgu
	Aciklama string
}

type Kopru struct {
	yetenekler map[string]KopruYetenek
}

func YeniKopru() *Kopru {
	k := &Kopru{yetenekler: map[string]KopruYetenek{}}

	// ============================================================
	// İSKELE SÖKÜLDÜ (Madde 11).
	// Eskiden burada Go'dan ödünç matematik/metin işlevleri vardı.
	// Hepsinin saf Tan karşılığı yazıldı → kütüphane/temel.tan.
	// Artık ödünç değil, Tan'ın kendisi. Köprü altyapısı duruyor:
	// gelecekte OS/GPU gibi GERÇEKTEN dış yetenekler için.
	// ============================================================

	return k
}


func (k *Kopru) kaydet(hedef, aciklama string, islev func([]Deger) Deger) {
	k.yetenekler[hedef] = KopruYetenek{Islev: islev, Iskele: true, Aciklama: aciklama}
}

func (k *Kopru) Cagir(hedef string, args []Deger, satir int) Deger {
	y, ok := k.yetenekler[hedef]
	if !ok {
		firlat(satir, "köprü hedefi bağlı değil: '%s'", hedef)
	}
	return y.Islev(args)
}

func sayiAl(a []Deger, i int) float64 {
	if i < len(a) {
		if f, ok := kesir(a[i]); ok {
			return f
		}
	}
	return 0
}
func metinAl(a []Deger, i int) string {
	if i < len(a) {
		if s, ok := a[i].(string); ok {
			return s
		}
		return metne(a[i])
	}
	return ""
}
