package main

// ============================================================
// Madde 12 — Bytecode + Sanal Makine (hız katmanı)
// AST'yi ağaçta gezmek yerine, düz bir komut dizisine (bytecode)
// derleyip yığın-tabanlı bir VM'de çalıştırıyoruz. Çok daha hızlı.
// Bu çekirdek: sayı/metin/mantık, değişken, aritmetik,
// karşılaştırma, koşul ve döngü (atlama komutlarıyla).
// ============================================================

type OpKodu uint8

const (
	OP_SABIT OpKodu = iota // sabiti yığına it (operand: sabit indeksi)
	OP_TOPLA               // iki değeri topla
	OP_CIKAR
	OP_CARP
	OP_BOL
	OP_MOD
	OP_ESIT       // ==
	OP_ESIT_DEGIL // !=
	OP_BUYUK      // >
	OP_KUCUK      // <
	OP_BUYUK_ESIT
	OP_KUCUK_ESIT
	OP_VE   // kullanılmıyor: kısa devre atlama ile yapılıyor
	OP_VEYA // kullanılmıyor
	OP_DEGIL
	OP_NEGATIF
	OP_BITVE         // &
	OP_BITVEYA       // |
	OP_BITXOR        // ^
	OP_BITDEGIL      // ~ (tekli)
	OP_SOLA_KAYDIR   // <<
	OP_SAGA_KAYDIR   // >>
	OP_DEGISKEN_OKU  // operand: değişken adı indeksi
	OP_DEGISKEN_YAZ  // operand: değişken adı indeksi
	OP_YAZDIR        // yaz
	OP_ATLA          // koşulsuz atla (operand: hedef)
	OP_YANLISSA_ATLA // yığının tepesi yanlışsa atla (operand: hedef)
	OP_AT            // yığının tepesini at (pop)
	OP_KOPYALA       // yığının tepesini çoğalt (dup) — kısa devre için
	OP_MANTIK        // yığının tepesini mantık değerine çevir (doğru/yanlış)
	OP_DOGRUYSA_ATLA // yığının tepesi doğruysa atla (operand: hedef)
	OP_DUR_PROGRAM   // programı bitir

	OP_ISLEV_TANIM // işlev değerini oluştur (operand: sabit havuzunda *VMIslev indeksi)
	OP_CAGIR       // işlev çağır (operand: argüman sayısı)
	OP_DONDUR      // işlevden dön (yığın tepesi: dönüş değeri)
)

// Komut: op kodu + isteğe bağlı operand
type Komut struct {
	Op      OpKodu
	Operand int
}

// DerlenmisKod: bytecode + sabit havuzu + değişken adları
type DerlenmisKod struct {
	Komutlar   []Komut
	Sabitler   []Deger
	AdlarHavuz []string
}

// VMIslev: VM'in derlediği bir işlevin kendi bytecode'u + parametre adları.
// Her işlev kendi bağımsız DerlenmisKod'unu taşır (kendi Sabitler/AdlarHavuz'u);
// çağrıldığında VM ona taze bir yerel değişken map'i verir — bu sayede
// özyineleme her çağrıda kendi kapsamını kullanır, birbirine karışmaz.
type VMIslev struct {
	Parametreler []string
	Kod          *DerlenmisKod
}

// Derleyici: AST -> bytecode
type Derleyici struct {
	kod *DerlenmisKod
}

func YeniDerleyici() *Derleyici {
	return &Derleyici{kod: &DerlenmisKod{}}
}

func (d *Derleyici) sabitEkle(deger Deger) int {
	d.kod.Sabitler = append(d.kod.Sabitler, deger)
	return len(d.kod.Sabitler) - 1
}

func (d *Derleyici) adEkle(ad string) int {
	for i, a := range d.kod.AdlarHavuz {
		if a == ad {
			return i
		}
	}
	d.kod.AdlarHavuz = append(d.kod.AdlarHavuz, ad)
	return len(d.kod.AdlarHavuz) - 1
}

func (d *Derleyici) yay(op OpKodu, operand int) int {
	d.kod.Komutlar = append(d.kod.Komutlar, Komut{op, operand})
	return len(d.kod.Komutlar) - 1
}

// Derle: deyim listesini bytecode'a çevirir
func (d *Derleyici) Derle(deyimler []Dugum) *DerlenmisKod {
	for _, deyim := range deyimler {
		d.deyimDerle(deyim)
	}
	d.yay(OP_DUR_PROGRAM, 0)
	return d.kod
}

func (d *Derleyici) deyimDerle(dugum Dugum) {
	switch n := dugum.(type) {
	case AtamaDugum:
		d.ifadeDerle(n.Deger)
		d.yay(OP_DEGISKEN_YAZ, d.adEkle(n.Ad))
	case YazDugum:
		d.ifadeDerle(n.Deger)
		d.yay(OP_YAZDIR, 0)
	case EgerDugum:
		d.ifadeDerle(n.Kosul)
		yanlisAtla := d.yay(OP_YANLISSA_ATLA, 0)
		for _, s := range n.Govde {
			d.deyimDerle(s)
		}
		if n.Degilse != nil {
			sonaAtla := d.yay(OP_ATLA, 0)
			d.kod.Komutlar[yanlisAtla].Operand = len(d.kod.Komutlar)
			for _, s := range n.Degilse {
				d.deyimDerle(s)
			}
			d.kod.Komutlar[sonaAtla].Operand = len(d.kod.Komutlar)
		} else {
			d.kod.Komutlar[yanlisAtla].Operand = len(d.kod.Komutlar)
		}
	case IkenDugum:
		basi := len(d.kod.Komutlar)
		d.ifadeDerle(n.Kosul)
		cikis := d.yay(OP_YANLISSA_ATLA, 0)
		for _, s := range n.Govde {
			d.deyimDerle(s)
		}
		d.yay(OP_ATLA, basi)
		d.kod.Komutlar[cikis].Operand = len(d.kod.Komutlar)
	case IslevDugum:
		// Her işlev bağımsız bir Derleyici ile, kendi bytecode'una derlenir.
		ic := YeniDerleyici()
		for _, s := range n.Govde {
			ic.deyimDerle(s)
		}
		// Güvenlik ağı: gövde açık "döndür" ile bitmezse "yok" döndür
		// (ağaç-gezenin islevCagir'daki varsayılan davranışıyla aynı).
		ic.yay(OP_SABIT, ic.sabitEkle(nil))
		ic.yay(OP_DONDUR, 0)
		islev := &VMIslev{Parametreler: n.Parametreler, Kod: ic.kod}
		d.yay(OP_ISLEV_TANIM, d.sabitEkle(islev))
		d.yay(OP_DEGISKEN_YAZ, d.adEkle(n.Ad))
	case DondurDugum:
		d.ifadeDerle(n.Deger)
		d.yay(OP_DONDUR, 0)
	default:
		// ifade-deyim: değerlendir, sonucu at
		d.ifadeDerle(dugum)
		d.yay(OP_AT, 0)
	}
}

func (d *Derleyici) ifadeDerle(dugum Dugum) {
	switch n := dugum.(type) {
	case SayiDugum:
		if n.TamMi {
			d.yay(OP_SABIT, d.sabitEkle(n.Tam))
		} else {
			d.yay(OP_SABIT, d.sabitEkle(n.Deger))
		}
	case MetinDugum:
		d.yay(OP_SABIT, d.sabitEkle(n.Deger))
	case MantikDugum:
		d.yay(OP_SABIT, d.sabitEkle(n.Deger))
	case YokDugum:
		d.yay(OP_SABIT, d.sabitEkle(nil))
	case DegiskenDugum:
		d.yay(OP_DEGISKEN_OKU, d.adEkle(n.Ad))
	case IkiliDugum:
		// --- KISA DEVRE: ve / veya ---
		// Sol taraf sonucu belirliyorsa sag taraf HIC degerlendirilmez.
		// Onceden iki taraf da calisiyordu: "b != 0 ve 10 / b > 1"
		// b=0 iken sifira bolme hatasi veriyordu. Yorumlayici dogru
		// yapiyordu, VM yapmiyordu — ayni kaynak farkli sonuc.
		if n.Islec == "ve" || n.Islec == "veya" {
			d.ifadeDerle(n.Sol)
			d.yay(OP_KOPYALA, 0)
			var atla int
			if n.Islec == "ve" {
				atla = d.yay(OP_YANLISSA_ATLA, 0) // yanlissa sag'i atla
			} else {
				atla = d.yay(OP_DOGRUYSA_ATLA, 0) // dogruysa sag'i atla
			}
			d.yay(OP_AT, 0) // kopyayi dus, sag'in sonucu gecerli olacak
			d.ifadeDerle(n.Sag)
			d.kod.Komutlar[atla].Operand = len(d.kod.Komutlar)
			// Sonucu mantik degerine cevir: yorumlayici da bool donduruyor,
			// aksi halde "1 ve 1" VM'de 1, yorumlayicida dogru verirdi.
			d.yay(OP_MANTIK, 0)
			return
		}
		if n.Islec == "değil" {
			d.ifadeDerle(n.Sol)
			d.yay(OP_DEGIL, 0)
			return
		}
		if n.Islec == "negatif" {
			d.ifadeDerle(n.Sol)
			d.yay(OP_NEGATIF, 0)
			return
		}
		if n.Islec == "bitdegil" {
			d.ifadeDerle(n.Sol)
			d.yay(OP_BITDEGIL, 0)
			return
		}
		d.ifadeDerle(n.Sol)
		d.ifadeDerle(n.Sag)
		switch n.Islec {
		case "+":
			d.yay(OP_TOPLA, 0)
		case "-":
			d.yay(OP_CIKAR, 0)
		case "*":
			d.yay(OP_CARP, 0)
		case "/":
			d.yay(OP_BOL, 0)
		case "%":
			d.yay(OP_MOD, 0)
		case "==":
			d.yay(OP_ESIT, 0)
		case "!=":
			d.yay(OP_ESIT_DEGIL, 0)
		case ">":
			d.yay(OP_BUYUK, 0)
		case "<":
			d.yay(OP_KUCUK, 0)
		case ">=":
			d.yay(OP_BUYUK_ESIT, 0)
		case "<=":
			d.yay(OP_KUCUK_ESIT, 0)
		case "&":
			d.yay(OP_BITVE, 0)
		case "|":
			d.yay(OP_BITVEYA, 0)
		case "^":
			d.yay(OP_BITXOR, 0)
		case "<<":
			d.yay(OP_SOLA_KAYDIR, 0)
		case ">>":
			d.yay(OP_SAGA_KAYDIR, 0)
		default:
			panic(vmDesteklemiyor{})
		}
	case CagriDugum:
		// Yerleşik işlevler (uzunluk, ekle, harfler, köprü vb.) henüz VM
		// kapsamı dışında — güvenle ağaç-gezene düş.
		if _, ok := yerlesikler[n.Ad]; ok {
			panic(vmDesteklemiyor{})
		}
		// İşlev değerini adıyla bul (normal değişken okuma gibi),
		// ardından argümanları sırayla it, en son OP_CAGIR ile çağır.
		d.ifadeDerle(DegiskenDugum{Ad: n.Ad, Satir: n.Satir})
		for _, arg := range n.Argumanlar {
			d.ifadeDerle(arg)
		}
		d.yay(OP_CAGIR, len(n.Argumanlar))
	default:
		// VM'nin desteklemediği ifade (işlev çağrısı, liste, sözlük, köprü):
		// bu durumda derleme başarısız sayılır; çağıran ağaç-gezene düşer.
		panic(vmDesteklemiyor{})
	}
}

// vmDesteklemiyor: bu program VM çekirdeğinin kapsamı dışında bir şey içeriyor
type vmDesteklemiyor struct{}
