package main

// ============================================================
// TAN SAYI KATMANI
// ------------------------------------------------------------
// Tan'da iki sayı tipi vardır:
//   int64   -> tam sayı   (kesin, 2^63'e kadar hassasiyet kaybı YOK)
//   float64 -> ondalık    (kesirli değerler)
//
// Kural:
//   tam OP tam       -> tam    (bölme hariç)
//   tam OP ondalık   -> ondalık
//   tam / tam        -> bölünüyorsa tam, bölünmüyorsa ondalık
//
// Bu katman, "123456789 * 987654321" gibi işlemlerin float64
// yuvarlamasıyla bozulmasını engeller.
// ============================================================

import "math"

// sayiMi: değer sayısal mı (tam veya ondalık)
func sayiMi(d Deger) bool {
	switch d.(type) {
	case int64, float64:
		return true
	}
	return false
}

// kesir: herhangi bir sayıyı float64'e çevirir
func kesir(d Deger) (float64, bool) {
	switch v := d.(type) {
	case int64:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// tamAl: değeri int64'e çevirir (ondalıksa keser)
func tamAl(d Deger) (int64, bool) {
	switch v := d.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	}
	return 0, false
}

// normalle: float64 sonucu tam sayıysa int64'e indirger.
// Böylece 6/3 gibi işlemler tam sayı olarak kalır.
func normalle(f float64) Deger {
	if f == math.Trunc(f) && !math.IsInf(f, 0) && !math.IsNaN(f) &&
		f >= -9.2e18 && f <= 9.2e18 {
		return int64(f)
	}
	return f
}

// sayiTopla vb.: iki sayıyı tip kurallarına göre işler.
// ikisi de tam ise tam aritmetiği kullanılır (kesin sonuç).
func ikisiTamMi(a, b Deger) (int64, int64, bool) {
	ai, aok := a.(int64)
	bi, bok := b.(int64)
	return ai, bi, aok && bok
}

// sayiIslem: + - * / % için ortak giriş noktası.
// hata: sıfıra bölmede true döner.
func sayiIslem(islec string, sol, sag Deger) (Deger, bool) {
	// İki taraf da tam sayıysa: kesin tam sayı aritmetiği
	if si, gi, ikisi := ikisiTamMi(sol, sag); ikisi {
		switch islec {
		case "+":
			return si + gi, false
		case "-":
			return si - gi, false
		case "*":
			return si * gi, false
		case "/":
			if gi == 0 {
				return nil, true
			}
			// tam bölünüyorsa tam sayı, değilse ondalık
			if si%gi == 0 {
				return si / gi, false
			}
			return float64(si) / float64(gi), false
		case "%":
			if gi == 0 {
				return nil, true
			}
			return si % gi, false
		}
	}

	// En az biri ondalık: float64 aritmetiği
	sf, sok := kesir(sol)
	gf, gok := kesir(sag)
	if !sok || !gok {
		return nil, false
	}
	switch islec {
	case "+":
		return sf + gf, false
	case "-":
		return sf - gf, false
	case "*":
		return sf * gf, false
	case "/":
		if gf == 0 {
			return nil, true
		}
		return sf / gf, false
	case "%":
		if int64(gf) == 0 {
			return nil, true
		}
		return math.Mod(sf, gf), false
	}
	return nil, false
}

// sayiKarsilastir: > < >= <= == != için ortak giriş noktası.
func sayiKarsilastir(islec string, sol, sag Deger) (Deger, bool) {
	// iki taraf da tam ise kesin karşılaştırma
	if si, gi, ikisi := ikisiTamMi(sol, sag); ikisi {
		switch islec {
		case ">":
			return si > gi, true
		case "<":
			return si < gi, true
		case ">=":
			return si >= gi, true
		case "<=":
			return si <= gi, true
		case "==":
			return si == gi, true
		case "!=":
			return si != gi, true
		}
		return nil, false
	}
	sf, sok := kesir(sol)
	gf, gok := kesir(sag)
	if !sok || !gok {
		return nil, false
	}
	switch islec {
	case ">":
		return sf > gf, true
	case "<":
		return sf < gf, true
	case ">=":
		return sf >= gf, true
	case "<=":
		return sf <= gf, true
	case "==":
		return sf == gf, true
	case "!=":
		return sf != gf, true
	}
	return nil, false
}
