package main

import "strconv"

// ============================================================
// TAN SABİT GENİŞLİKLİ TAMSAYILAR (u8/u16/u32/u64/i8/i16/i32/i64)
// ------------------------------------------------------------
// TanSabitTam TEK bir Go tipiyle sekiz varyantı da temsil eder:
// Genislik (8/16/32/64) + Imzali (i* mi u* mi) + ham 64-bit bit
// deseni (Bit alanı her zaman Genislik kadar anlamlı bit tutar,
// üst bitler sıfırlanmış saklanır).
//
// Toplama/çıkarma/çarpma iki'nin tümleyeninde işaretli/işaretsiz
// AYNI bit sonucunu verir (donanımdaki gibi) — bu yüzden Bit
// üzerinde doğrudan yürütülüp maskelenir. Bölme/mod işaret
// duyarlıdır: işaretliyse önce işaret genişletilip int64
// bölmesi (sıfıra doğru keser) uygulanır.
//
// Taşma (overflow) davranışı: sabitOlustur() ve her aritmetik
// işlem sonucu Genislik bit ile MASKELENİR — yani 2^Genislik'e
// göre sarma (wraparound) TANIMLI davranıştır, panik değil.
// ============================================================

type TanSabitTam struct {
	Genislik int // 8, 16, 32 veya 64
	Imzali   bool
	Bit      uint64 // ham bit deseni, üst (64-Genislik) bit her zaman 0
}

func sabitMaske(genislik int) uint64 {
	if genislik >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << uint(genislik)) - 1
}

// sabitIsaretUzat: Genislik bitlik iki'nin tümleyeni deseni int64'e
// işaret genişletilerek çevrilir (imzali tipler için).
func sabitIsaretUzat(bit uint64, genislik int) int64 {
	if genislik >= 64 {
		return int64(bit)
	}
	isaretBiti := uint64(1) << uint(genislik-1)
	if bit&isaretBiti != 0 {
		return int64(bit | ^sabitMaske(genislik))
	}
	return int64(bit)
}

// sabitOlustur: herhangi bir int64 değeri Genislik bite indirger
// (sarma/taşma burada tanımlı şekilde uygulanır).
func sabitOlustur(genislik int, imzali bool, deger int64) *TanSabitTam {
	return &TanSabitTam{Genislik: genislik, Imzali: imzali, Bit: uint64(deger) & sabitMaske(genislik)}
}

// sabitDegerCikar: bir Deger'i (tam/ondalık/başka sabit-tam) int64'e
// çevirir — u8(x) gibi kurucuların ortak girişi.
func sabitDegerCikar(d Deger) (int64, bool) {
	switch v := d.(type) {
	case *TanSabitTam:
		return sabitIsaretUzat(v.Bit, v.Genislik), true
	default:
		return tamAl(d)
	}
}

// sabitMetne: gösterim — imzalıysa işaret genişletilmiş imzalı ondalık,
// değilse (bit zaten maskeli olduğundan) doğrudan işaretsiz ondalık.
func sabitMetne(v *TanSabitTam) string {
	if v.Imzali {
		return strconv.FormatInt(sabitIsaretUzat(v.Bit, v.Genislik), 10)
	}
	return strconv.FormatUint(v.Bit, 10)
}

// sabitUyumlu: iki sabit-tam değer aynı genişlik+işaretlilikte mi
func sabitUyumlu(a, b *TanSabitTam) bool {
	return a.Genislik == b.Genislik && a.Imzali == b.Imzali
}

func sabitAdi(v *TanSabitTam) string {
	on := "u"
	if v.Imzali {
		on = "i"
	}
	return on + strconv.Itoa(v.Genislik)
}

// sabitIslem: + - * / % için ortak giriş noktası. a/b AYNI tip olmalı.
// hata: sıfıra bölmede true döner.
func sabitIslem(islec string, a, b *TanSabitTam) (*TanSabitTam, bool) {
	maske := sabitMaske(a.Genislik)
	switch islec {
	case "+":
		return &TanSabitTam{a.Genislik, a.Imzali, (a.Bit + b.Bit) & maske}, false
	case "-":
		return &TanSabitTam{a.Genislik, a.Imzali, (a.Bit - b.Bit) & maske}, false
	case "*":
		return &TanSabitTam{a.Genislik, a.Imzali, (a.Bit * b.Bit) & maske}, false
	case "/":
		if a.Imzali {
			av, bv := sabitIsaretUzat(a.Bit, a.Genislik), sabitIsaretUzat(b.Bit, b.Genislik)
			if bv == 0 {
				return nil, true
			}
			return &TanSabitTam{a.Genislik, true, uint64(av/bv) & maske}, false
		}
		if b.Bit == 0 {
			return nil, true
		}
		return &TanSabitTam{a.Genislik, false, (a.Bit / b.Bit) & maske}, false
	case "%":
		if a.Imzali {
			av, bv := sabitIsaretUzat(a.Bit, a.Genislik), sabitIsaretUzat(b.Bit, b.Genislik)
			if bv == 0 {
				return nil, true
			}
			return &TanSabitTam{a.Genislik, true, uint64(av%bv) & maske}, false
		}
		if b.Bit == 0 {
			return nil, true
		}
		return &TanSabitTam{a.Genislik, false, (a.Bit % b.Bit) & maske}, false
	}
	return nil, false
}

// sabitKarsilastir: > < >= <= == != için ortak giriş noktası.
func sabitKarsilastir(islec string, a, b *TanSabitTam) bool {
	if islec == "==" {
		return a.Bit == b.Bit
	}
	if islec == "!=" {
		return a.Bit != b.Bit
	}
	if a.Imzali {
		av, bv := sabitIsaretUzat(a.Bit, a.Genislik), sabitIsaretUzat(b.Bit, b.Genislik)
		switch islec {
		case ">":
			return av > bv
		case "<":
			return av < bv
		case ">=":
			return av >= bv
		case "<=":
			return av <= bv
		}
		return false
	}
	switch islec {
	case ">":
		return a.Bit > b.Bit
	case "<":
		return a.Bit < b.Bit
	case ">=":
		return a.Bit >= b.Bit
	case "<=":
		return a.Bit <= b.Bit
	}
	return false
}
