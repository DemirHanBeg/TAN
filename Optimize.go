package main

// ============================================================
// TAN OPTİMİZE EDİCİ (AST seviyesi)
// ------------------------------------------------------------
// Üç arka ucun da faydalandığı, backend'den bağımsız geçiş.
//
//  1. SABİT KATLAMA        2 + 3 * 4      -> 14
//  2. CEBİRSEL SADELEŞTİRME x + 0, x * 1, x * 0, x - 0, x / 1
//  3. ÖLÜ KOD ELEME        eğer 0 ise ... son   -> tamamen silinir
//                          iken 0 ...           -> tamamen silinir
//  4. SABİT KOŞUL DÜZLEME  eğer 1 ise A değilse B son -> A
//
// Bu geçiş anlamı korur: sadece derleme anında kesin bilinen
// değerleri hesaplar. Taşma davranışı int64 ile birebir aynıdır.
// ============================================================

import "math"

type Optimizer struct {
	Katlanan int // kaç düğüm katlandı (rapor için)
	Silinen  int // kaç ölü blok silindi
}

func YeniOptimizer() *Optimizer { return &Optimizer{} }

// Govde: bir deyim listesini optimize eder.
func (o *Optimizer) Govde(govde []Dugum) []Dugum {
	var sonuc []Dugum
	for _, d := range govde {
		yeni, atla := o.deyim(d)
		if atla {
			o.Silinen++
			continue
		}
		if coklu, ok := yeni.([]Dugum); ok {
			sonuc = append(sonuc, coklu...)
			continue
		}
		sonuc = append(sonuc, yeni)
	}
	return sonuc
}

// deyim: tek bir deyimi optimize eder. atla=true ise deyim silinir.
func (o *Optimizer) deyim(d Dugum) (Dugum, bool) {
	switch n := d.(type) {
	case AtamaDugum:
		n.Deger = o.ifade(n.Deger)
		return n, false

	case IndeksAtamaDugum:
		n.Hedef = o.ifade(n.Hedef)
		n.Indeks = o.ifade(n.Indeks)
		n.Deger = o.ifade(n.Deger)
		return n, false

	case YazDugum:
		n.Deger = o.ifade(n.Deger)
		return n, false

	case DondurDugum:
		if n.Deger != nil {
			n.Deger = o.ifade(n.Deger)
		}
		return n, false

	case EgerDugum:
		n.Kosul = o.ifade(n.Kosul)
		n.Govde = o.Govde(n.Govde)
		n.Degilse = o.Govde(n.Degilse)
		// sabit koşul -> dalı düzle
		if sabit, deger, ok := sabitMantik(n.Kosul); ok && sabit {
			if deger {
				if len(n.Govde) == 0 {
					return nil, true
				}
				return o.blokDugumu(n.Govde), false
			}
			if len(n.Degilse) == 0 {
				return nil, true
			}
			return o.blokDugumu(n.Degilse), false
		}
		return n, false

	case IkenDugum:
		n.Kosul = o.ifade(n.Kosul)
		n.Govde = o.Govde(n.Govde)
		// koşul kesin yanlışsa döngü hiç çalışmaz
		if sabit, deger, ok := sabitMantik(n.Kosul); ok && sabit && !deger {
			return nil, true
		}
		return n, false

	case HerDugum:
		n.Liste = o.ifade(n.Liste)
		n.Govde = o.Govde(n.Govde)
		return n, false

	case IslevDugum:
		n.Govde = o.Govde(n.Govde)
		return n, false

	case CagriDugum:
		return o.ifade(n), false
	}
	return d, false
}

// blokDugumu: birden çok deyimi tek düğüm gibi döndürmek için
func (o *Optimizer) blokDugumu(govde []Dugum) Dugum {
	if len(govde) == 1 {
		return govde[0]
	}
	// çok deyimliyse her zaman doğru olan bir eğer ile sarmala
	return EgerDugum{Kosul: MantikDugum{Deger: true}, Govde: govde}
}

// ifade: bir ifadeyi optimize eder.
func (o *Optimizer) ifade(d Dugum) Dugum {
	switch n := d.(type) {
	case IkiliDugum:
		n.Sol = o.ifade(n.Sol)
		n.Sag = o.ifade(n.Sag)

		// --- 1. SABİT KATLAMA ---
		if katli, ok := o.sabitKatla(n); ok {
			o.Katlanan++
			return katli
		}

		// --- 2. CEBİRSEL SADELEŞTİRME ---
		if sade, ok := o.cebirsel(n); ok {
			o.Katlanan++
			return sade
		}
		return n

	case CagriDugum:
		for i := range n.Argumanlar {
			n.Argumanlar[i] = o.ifade(n.Argumanlar[i])
		}
		return n

	case ListeDugum:
		for i := range n.Elemanlar {
			n.Elemanlar[i] = o.ifade(n.Elemanlar[i])
		}
		return n

	case IndeksDugum:
		n.Hedef = o.ifade(n.Hedef)
		n.Indeks = o.ifade(n.Indeks)
		return n
	}
	return d
}

// sabitKatla: iki sabit sayının işlemini derleme anında hesaplar.
func (o *Optimizer) sabitKatla(n IkiliDugum) (Dugum, bool) {
	sol, solOk := n.Sol.(SayiDugum)
	sag, sagOk := n.Sag.(SayiDugum)
	if !solOk || !sagOk {
		return nil, false
	}

	// iki taraf da TAM SAYI ise kesin int64 aritmetiği
	if sol.TamMi && sag.TamMi {
		a, b := sol.Tam, sag.Tam
		switch n.Islec {
		case "+":
			return tamDugum(a + b), true
		case "-":
			return tamDugum(a - b), true
		case "*":
			// taşma kontrolü: taşarsa katlama, çalışma anına bırak
			if a != 0 {
				c := a * b
				if c/a != b {
					return nil, false
				}
				return tamDugum(c), true
			}
			return tamDugum(0), true
		case "/":
			if b == 0 {
				return nil, false // sıfıra bölme: çalışma anına bırak
			}
			if a%b == 0 {
				return tamDugum(a / b), true
			}
			// TAM BÖLÜNMÜYORSA KATLAMA YAPMA.
			// Sebep: arka uçlar tam sayı bölmesinde farklı davranıyor —
			// elf/asm tam sayı bölmesi (14), yorumlayıcı ve C yolu ondalık
			// (14.285714). Derleme aninda katlarsak hangi arka uca
			// derlendigini bilemeyiz, anlam kayar. Calisma anina birak.
			return nil, false
		case "%":
			if b == 0 {
				return nil, false
			}
			return tamDugum(a % b), true
		case "==":
			return MantikDugum{Deger: a == b}, true
		case "!=":
			return MantikDugum{Deger: a != b}, true
		case ">":
			return MantikDugum{Deger: a > b}, true
		case "<":
			return MantikDugum{Deger: a < b}, true
		case ">=":
			return MantikDugum{Deger: a >= b}, true
		case "<=":
			return MantikDugum{Deger: a <= b}, true
		}
		return nil, false
	}

	// en az biri ondalık
	a, b := sol.Deger, sag.Deger
	switch n.Islec {
	case "+":
		return kesirDugum(a + b), true
	case "-":
		return kesirDugum(a - b), true
	case "*":
		return kesirDugum(a * b), true
	case "/":
		if b == 0 {
			return nil, false
		}
		return kesirDugum(a / b), true
	case "==":
		return MantikDugum{Deger: a == b}, true
	case "!=":
		return MantikDugum{Deger: a != b}, true
	case ">":
		return MantikDugum{Deger: a > b}, true
	case "<":
		return MantikDugum{Deger: a < b}, true
	case ">=":
		return MantikDugum{Deger: a >= b}, true
	case "<=":
		return MantikDugum{Deger: a <= b}, true
	}
	return nil, false
}

// cebirsel: x+0, x*1, x*0, x-0, x/1 gibi sadeleştirmeler
func (o *Optimizer) cebirsel(n IkiliDugum) (Dugum, bool) {
	solSifir := tamSabitMi(n.Sol, 0)
	sagSifir := tamSabitMi(n.Sag, 0)
	solBir := tamSabitMi(n.Sol, 1)
	sagBir := tamSabitMi(n.Sag, 1)

	switch n.Islec {
	case "+":
		if sagSifir {
			return n.Sol, true
		}
		if solSifir {
			return n.Sag, true
		}
	case "-":
		if sagSifir {
			return n.Sol, true
		}
	case "*":
		if sagBir {
			return n.Sol, true
		}
		if solBir {
			return n.Sag, true
		}
		// x * 0 -> 0  (yan etki yoksa; sabit/değişken için güvenli)
		if sagSifir && yanEtkisiz(n.Sol) {
			return tamDugum(0), true
		}
		if solSifir && yanEtkisiz(n.Sag) {
			return tamDugum(0), true
		}
	case "/":
		if sagBir {
			return n.Sol, true
		}
	}
	return nil, false
}

// yanEtkisiz: ifadeyi silmek güvenli mi (işlev çağrısı içermiyor mu)
func yanEtkisiz(d Dugum) bool {
	switch n := d.(type) {
	case SayiDugum, MetinDugum, MantikDugum, YokDugum, DegiskenDugum:
		return true
	case IkiliDugum:
		return yanEtkisiz(n.Sol) && yanEtkisiz(n.Sag)
	case IndeksDugum:
		return yanEtkisiz(n.Hedef) && yanEtkisiz(n.Indeks)
	}
	return false
}

func tamSabitMi(d Dugum, v int64) bool {
	s, ok := d.(SayiDugum)
	return ok && s.TamMi && s.Tam == v
}

// sabitMantik: koşul derleme anında kesin biliniyor mu
func sabitMantik(d Dugum) (sabit bool, deger bool, ok bool) {
	switch n := d.(type) {
	case MantikDugum:
		return true, n.Deger, true
	case SayiDugum:
		if n.TamMi {
			return true, n.Tam != 0, true
		}
		return true, n.Deger != 0, true
	}
	return false, false, false
}

func tamDugum(v int64) SayiDugum {
	return SayiDugum{Deger: float64(v), Tam: v, TamMi: true}
}

func kesirDugum(v float64) SayiDugum {
	if v == math.Trunc(v) && math.Abs(v) < 9.2e18 {
		return tamDugum(int64(v))
	}
	return SayiDugum{Deger: v}
}
