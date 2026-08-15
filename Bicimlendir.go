package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ============================================================
// tan biçimlendir — deterministik, tekrarlı-özdeş (idempotent) biçimleyici.
//
// Yeni sözdizimi İÇERMEZ. Sözdizimini doğrulamak için AST'yi kullanır
// (ayrıştırma başarısızsa dosya DEĞİŞTİRİLMEZ), ardından her deyimi
// kanonik biçimde yeniden üretir. Yorumlar kayba uğramaz:
//   - bağımsız yorum satırları ("# ...") korunur, içinde bulundukları
//     bloğun girintisine taşınır;
//   - satır-içi yorumlar (kodun sonundaki "# ...") koda eklenir.
//
// Girinti birimi 4 boşluktur (repo geleneği). Operatör önceliği korunur;
// anlamı değiştirmemek için gerektiğinde parantez eklenir.
//
// Kullanım:
//   tan biçimlendir <dosya...>
//   tan biçimlendir --denet <dosya...>   (yazar mı, fark var mı: çıkış 0/1)
//   tan biçimlendir --cikti <dosya>      (dosyayı yazmaz, stdout'a basar)
// ============================================================

const girintiBirimi = "    " // 4 boşluk (repo geleneği)

// operatörÖncelik: ikili işleçlerin önceliği (küçük = daha gevşek).
// Tekli işleçler (negatif, bitdegil, değil) en yüksek önceliğe sahiptir.
var operatörÖncelik = map[string]int{
	"ve": 1, "veya": 1,
	"&": 2, "|": 2, "^": 2,
	">": 3, "<": 3, ">=": 3, "<=": 3, "==": 3, "!=": 3,
	"<<": 4, ">>": 4,
	"+": 5, "-": 5,
	"*": 6, "/": 6, "%": 6,
}

const öncelikUnar = 7

func işleçÖnceliği(s string) int {
	if p, ok := operatörÖncelik[s]; ok {
		return p
	}
	if s == "negatif" || s == "bitdegil" || s == "değil" {
		return öncelikUnar
	}
	return 0
}

func girinti(derinlik int) string {
	return strings.Repeat(girintiBirimi, derinlik)
}

// bicimSatir: biçimlenmiş tek satır + kaynaktaki başlangıç satırı (yorum
// yerleştirmesi için) + tür bilgisi.
type bicimSatir struct {
	metin          string
	kaynakSatir    int
	yapısal        bool // son/değilse/yakala gibi yapısal kapanış satırı
	derinlik       int
	öncesindeYorum bool
}

type bicimYazici struct {
	tokenlar       []Token // T_YENI_SATIR ayıklanmış
	tokenKonum     int
	sonKaynakSatir int
	cikti          []bicimSatir
}

// kaynakSatirBul: tokenKonum'dan itibaren koşula uyan ilk belirteci arar;
// belirtecin satırını döndürür, tokenKonum'u eşleşenin bir ilerisine taşır.
func (w *bicimYazici) kaynakSatirBul(eslesme func(Token, Token) bool) int {
	for i := w.tokenKonum; i < len(w.tokenlar); i++ {
		tok := w.tokenlar[i]
		var sonraki Token
		if i+1 < len(w.tokenlar) {
			sonraki = w.tokenlar[i+1]
		}
		if eslesme(tok, sonraki) {
			w.tokenKonum = i + 1
			if tok.Satir > w.sonKaynakSatir {
				w.sonKaynakSatir = tok.Satir
			}
			return tok.Satir
		}
	}
	w.tokenKonum = len(w.tokenlar)
	return -1
}

// yapisalSatir: kapanış/yapısal satırlar için tahmini kaynak satırı.
// (son yazılan deyimin satırından bir sonrası.)
func (w *bicimYazici) yapisalSatir() int {
	if w.sonKaynakSatir > 0 {
		return w.sonKaynakSatir + 1
	}
	return 1
}

func (w *bicimYazici) ekle(derinlik int, metin string, kaynakSatir int, yapısal bool) {
	w.cikti = append(w.cikti, bicimSatir{metin: metin, kaynakSatir: kaynakSatir, yapısal: yapısal, derinlik: derinlik})
}

// ---- belirteç çapası (yorum yerleştirme için) ----

func cAnahtar(a string) func(Token, Token) bool {
	return func(t, _ Token) bool { return t.Tur == T_ANAHTAR && t.Deger == a }
}

func cIsimVeEsit(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		return t.Tur == T_TANIMLAYICI && t.Deger == ad && n.Tur == T_ISLEC && n.Deger == "="
	}
}

func cIsimVeCagri(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		return t.Tur == T_TANIMLAYICI && t.Deger == ad && n.Tur == T_PARANTEZ_AC
	}
}

func cIsimVeSuslu(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		return t.Tur == T_TANIMLAYICI && t.Deger == ad && n.Tur == T_SUSLU_AC
	}
}

func cIsimAlan(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		if t.Tur != T_TANIMLAYICI || t.Deger != ad {
			return false
		}
		if n.Tur == T_PARANTEZ_AC || n.Tur == T_SUSLU_AC {
			return false
		}
		if n.Tur == T_ISLEC && n.Deger == "=" {
			return false
		}
		return true
	}
}

func cIslevVeAd(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		return t.Tur == T_ANAHTAR && t.Deger == "işlev" && n.Tur == T_TANIMLAYICI && n.Deger == ad
	}
}

func cKayitVeAd(ad string) func(Token, Token) bool {
	return func(t, n Token) bool {
		return t.Tur == T_ANAHTAR && t.Deger == "kayıt" && n.Tur == T_TANIMLAYICI && n.Deger == ad
	}
}

// ---- AST -> metin ----

func (w *bicimYazici) yaz(deyimler []Dugum, derinlik int) {
	for _, d := range deyimler {
		switch n := d.(type) {
		case AtamaDugum:
			s := w.kaynakSatirBul(cIsimVeEsit(n.Ad))
			w.ekle(derinlik, girinti(derinlik)+n.Ad+" = "+w.ifade(n.Deger, 0), s, false)

		case YazDugum:
			s := w.kaynakSatirBul(cAnahtar("yaz"))
			w.ekle(derinlik, girinti(derinlik)+"yaz "+w.ifade(n.Deger, 0), s, false)

		case EgerDugum:
			s := w.kaynakSatirBul(cAnahtar("eğer"))
			w.ekle(derinlik, girinti(derinlik)+"eğer "+w.ifade(n.Kosul, 0)+" ise", s, false)
			w.yaz(n.Govde, derinlik+1)
			if len(n.Degilse) > 0 {
				w.ekle(derinlik, girinti(derinlik)+"değilse", w.yapisalSatir(), true)
				// değilse eğer zinciri: iç eğer aynı derinlikte yazılır.
				if len(n.Degilse) == 1 {
					if _, ok := n.Degilse[0].(EgerDugum); ok {
						w.yaz(n.Degilse, derinlik)
						w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)
						continue
					}
				}
				w.yaz(n.Degilse, derinlik+1)
			}
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		case IkenDugum:
			s := w.kaynakSatirBul(cAnahtar("iken"))
			w.ekle(derinlik, girinti(derinlik)+"iken "+w.ifade(n.Kosul, 0), s, false)
			w.yaz(n.Govde, derinlik+1)
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		case HerDugum:
			s := w.kaynakSatirBul(cAnahtar("her"))
			w.ekle(derinlik, girinti(derinlik)+"her "+n.Degisken+" "+w.ifade(n.Liste, 0)+" içinde", s, false)
			w.yaz(n.Govde, derinlik+1)
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		case IslevDugum:
			s := w.kaynakSatirBul(cIslevVeAd(n.Ad))
			w.ekle(derinlik, girinti(derinlik)+"işlev "+n.Ad+"("+strings.Join(n.Parametreler, ", ")+")", s, false)
			w.yaz(n.Govde, derinlik+1)
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		case DondurDugum:
			s := w.kaynakSatirBul(cAnahtar("döndür"))
			if _, bos := n.Deger.(YokDugum); bos {
				w.ekle(derinlik, girinti(derinlik)+"döndür", s, false)
			} else {
				w.ekle(derinlik, girinti(derinlik)+"döndür "+w.ifade(n.Deger, 0), s, false)
			}

		case DurDugum:
			s := w.kaynakSatirBul(cAnahtar("dur"))
			w.ekle(derinlik, girinti(derinlik)+"dur", s, false)

		case DevamDugum:
			s := w.kaynakSatirBul(cAnahtar("devam"))
			w.ekle(derinlik, girinti(derinlik)+"devam", s, false)

		case IceAlDugum:
			s := w.kaynakSatirBul(cAnahtar("içe"))
			w.ekle(derinlik, girinti(derinlik)+"içe al "+metinSabit(n.Dosya), s, false)

		case DeneDugum:
			s := w.kaynakSatirBul(cAnahtar("dene"))
			w.ekle(derinlik, girinti(derinlik)+"dene", s, false)
			w.yaz(n.DeneGovde, derinlik+1)
			if len(n.YakalaGovde) > 0 || n.HataAdi != "" {
				w.ekle(derinlik, girinti(derinlik)+"yakala "+n.HataAdi, w.yapisalSatir(), true)
				w.yaz(n.YakalaGovde, derinlik+1)
			}
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		case KopruDugum:
			s := w.kaynakSatirBul(cAnahtar("köprü"))
			var a []string
			for _, arg := range n.Argumanlar {
				a = append(a, w.ifade(arg, 0))
			}
			w.ekle(derinlik, girinti(derinlik)+"köprü("+metinSabit(n.Hedef)+strings.Join(a, ", ")+")", s, false)

		case KayitTanimDugum:
			s := w.kaynakSatirBul(cKayitVeAd(n.Ad))
			w.ekle(derinlik, girinti(derinlik)+"kayıt "+n.Ad, s, false)
			for _, alan := range n.Alanlar {
				as := w.kaynakSatirBul(cIsimAlan(alan))
				w.ekle(derinlik+1, girinti(derinlik+1)+alan, as, false)
			}
			for _, metot := range n.Metotlar {
				w.yaz([]Dugum{metot}, derinlik+1)
			}
			w.ekle(derinlik, girinti(derinlik)+"son", w.yapisalSatir(), true)

		default:
			w.ekle(derinlik, girinti(derinlik)+w.ifade(d, 0), w.bilinmeyenSatir(), false)
		}
	}
}

// bilinmeyenSatir: çapası tanımlanmamış deyimler için ilk kullanılmamış
// belirtecin satırı (yorum sıralaması yaklaşık tutulur).
func (w *bicimYazici) bilinmeyenSatir() int {
	if w.tokenKonum < len(w.tokenlar) {
		s := w.tokenlar[w.tokenKonum].Satir
		if s > w.sonKaynakSatir {
			w.sonKaynakSatir = s
		}
		return s
	}
	return w.yapisalSatir()
}

// ifade: bir ifadeyi öncelik bilinciyle metne çevirir. babaÖncelik, bu
// ifadenin bağlamındaki önceliktir; alt ikili düğümler ondan gevşekse
// parantezlenir (anlam korunur).
func (w *bicimYazici) ifade(d Dugum, babaÖncelik int) string {
	switch n := d.(type) {
	case SayiDugum:
		if n.TamMi {
			return strconv.FormatInt(n.Tam, 10)
		}
		return strconv.FormatFloat(n.Deger, 'f', -1, 64)
	case MetinDugum:
		return metinSabit(n.Deger)
	case MantikDugum:
		if n.Deger {
			return "doğru"
		}
		return "yanlış"
	case YokDugum:
		return "yok"
	case DegiskenDugum:
		return n.Ad
	case IkiliDugum:
		if n.Sag == nil { // tekli: negatif / bitdegil / değil
			ic := w.ifade(n.Sol, öncelikUnar)
			switch n.Islec {
			case "negatif":
				return "-" + ic
			case "bitdegil":
				return "~" + ic
			default:
				return "değil " + ic
			}
		}
		p := işleçÖnceliği(n.Islec)
		sol := w.ifade(n.Sol, p)
		sag := w.ifade(n.Sag, p)
		parSol := false
		if bb, ok := n.Sol.(IkiliDugum); ok && bb.Sag != nil {
			if işleçÖnceliği(bb.Islec) < p {
				parSol = true
			}
		}
		parSag := false
		if bb, ok := n.Sag.(IkiliDugum); ok && bb.Sag != nil {
			pp := işleçÖnceliği(bb.Islec)
			if pp < p || pp == p {
				parSag = true
			}
		}
		if parSol {
			sol = "(" + sol + ")"
		}
		if parSag {
			sag = "(" + sag + ")"
		}
		return sol + " " + n.Islec + " " + sag
	case CagriDugum:
		var a []string
		for _, arg := range n.Argumanlar {
			a = append(a, w.ifade(arg, 0))
		}
		return n.Ad + "(" + strings.Join(a, ", ") + ")"
	case ListeDugum:
		var a []string
		for _, e := range n.Elemanlar {
			a = append(a, w.ifade(e, 0))
		}
		return "[" + strings.Join(a, ", ") + "]"
	case SozlukDugum:
		var a []string
		for i := range n.Anahtarlar {
			a = append(a, w.ifade(n.Anahtarlar[i], 0)+": "+w.ifade(n.Degerler[i], 0))
		}
		return "{" + strings.Join(a, ", ") + "}"
	case IndeksDugum:
		return w.ifade(n.Hedef, öncelikUnar) + "[" + w.ifade(n.Indeks, 0) + "]"
	case IndeksAtamaDugum:
		return w.ifade(n.Hedef, öncelikUnar) + "[" + w.ifade(n.Indeks, 0) + "] = " + w.ifade(n.Deger, 0)
	case KayitOlusturDugum:
		var a []string
		for i := range n.AlanAdlari {
			a = append(a, n.AlanAdlari[i]+": "+w.ifade(n.Degerler[i], 0))
		}
		return n.Ad + "{" + strings.Join(a, ", ") + "}"
	case AlanErisimDugum:
		return w.ifade(n.Hedef, öncelikUnar) + "." + n.Alan
	case AlanAtamaDugum:
		return w.ifade(n.Hedef, öncelikUnar) + "." + n.Alan + " = " + w.ifade(n.Deger, 0)
	case MetotCagriDugum:
		var a []string
		for _, arg := range n.Argumanlar {
			a = append(a, w.ifade(arg, 0))
		}
		return w.ifade(n.Hedef, öncelikUnar) + "." + n.Metot + "(" + strings.Join(a, ", ") + ")"
	case CagriIfadeDugum:
		var a []string
		for _, arg := range n.Argumanlar {
			a = append(a, w.ifade(arg, 0))
		}
		return w.ifade(n.Hedef, öncelikUnar) + "(" + strings.Join(a, ", ") + ")"
	}
	return ""
}

// metinSabit: metin değerini kaynak gösterimine çevirir (tırnak + kaçışlar).
func metinSabit(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// ---- yorum toplama ----

func yorumlarıTopla(tokenlar []Token, hamSatirlar []string) (bagimsiz, satirIci map[int]string) {
	bagimsiz = map[int]string{}
	satirIci = map[int]string{}
	sonBitis := map[int]int{}
	for _, t := range tokenlar {
		if t.Tur == T_YENI_SATIR {
			continue
		}
		bitis := t.Sutun + utf8.RuneCountInString(t.Deger)
		if bitis > sonBitis[t.Satir] {
			sonBitis[t.Satir] = bitis
		}
	}
	for i, ham := range hamSatirlar {
		satirNo := i + 1
		trim := strings.TrimSpace(ham)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "#") {
			bagimsiz[satirNo] = strings.TrimRight(ham, " \t\r")
			continue
		}
		bitis := sonBitis[satirNo]
		if bitis == 0 {
			continue
		}
		runes := []rune(ham)
		if bitis > len(runes) {
			continue
		}
		for j := bitis; j < len(runes); j++ {
			if runes[j] == '#' {
				satirIci[satirNo] = strings.TrimRight(string(runes[j:]), " \t\r")
				break
			}
		}
	}
	return bagimsiz, satirIci
}

func satirBoşMu(ham string) bool {
	return strings.TrimSpace(ham) == ""
}

// ---- birleştirme ----

func bicimBirleştir(satirlar []bicimSatir, bagimsiz, satirIci map[int]string, hamSatirlar []string) string {
	var b strings.Builder
	sonYayilan := 0
	oncekiDerinlik := 0
	oncekiYapısal := true
	for _, s := range satirlar {
		hedef := s.kaynakSatir
		if hedef < sonYayilan {
			hedef = sonYayilan
		}
		// Boş satır (en fazla 1): aynı derinlikteki, yapısal olmayan iki deyim arası.
		bosVardi := false
		for l := sonYayilan + 1; l <= hedef; l++ {
			if l-1 < len(hamSatirlar) && satirBoşMu(hamSatirlar[l-1]) {
				bosVardi = true
				break
			}
		}
		if !s.yapısal && bosVardi && s.derinlik == oncekiDerinlik && !oncekiYapısal {
			b.WriteByte('\n')
		}
		// Bağımsız yorumları boşalt (satır sırasına göre).
		for l := sonYayilan + 1; l <= hedef; l++ {
			if c, ok := bagimsiz[l]; ok {
				b.WriteString(girinti(s.derinlik))
				b.WriteString(strings.TrimLeft(c, " \t"))
				b.WriteByte('\n')
			}
		}
		b.WriteString(s.metin)
		if ic, ok := satirIci[s.kaynakSatir]; ok {
			b.WriteString("  ")
			b.WriteString(ic)
		}
		b.WriteByte('\n')
		sonYayilan = hedef
		oncekiDerinlik = s.derinlik
		oncekiYapısal = s.yapısal
	}
	// Dosya sonundaki kalan bağımsız yorumlar.
	for l := sonYayilan + 1; l <= len(hamSatirlar); l++ {
		if c, ok := bagimsiz[l]; ok {
			b.WriteString(strings.TrimLeft(c, " \t"))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// bicimleKaynak: kaynak metni biçimler. Ayrıştırma başarısızsa hata döner.
func bicimleKaynak(kaynak string) (sonuc string, hata error) {
	normal := strings.ReplaceAll(kaynak, "\r\n", "\n")
	hamSatirlar := strings.Split(normal, "\n")
	toklar := YeniLexer(normal).Tokenle()

	defer func() {
		if r := recover(); r != nil {
			if th, ok := r.(TanHata); ok {
				hata = th
				return
			}
			panic(r)
		}
	}()

	agac := YeniParser(toklar).Ayristir()

	filtreli := toklar[:0]
	for _, t := range toklar {
		if t.Tur != T_YENI_SATIR {
			filtreli = append(filtreli, t)
		}
	}
	w := &bicimYazici{tokenlar: filtreli}
	w.yaz(agac, 0)

	bagimsiz, satirIci := yorumlarıTopla(filtreli, hamSatirlar)
	return bicimBirleştir(w.cikti, bagimsiz, satirIci, hamSatirlar), nil
}

// ---- CLI ----

func bicimlendirKullanim() {
	fmt.Println("Kullanım: tan biçimlendir [--denet|--cikti] <dosya...>")
	fmt.Println("  --denet  dosyayı değiştirmez; biçimlendirme gerekiyorsa çıkış 1")
	fmt.Println("  --cikti  biçimlenmiş metni stdout'a basar (tek dosya)")
}

func bicimlendirKomutu(args []string) {
	denet, ciktiModu := false, false
	var dosyalar []string
	for _, a := range args {
		switch a {
		case "--denet":
			denet = true
		case "--cikti":
			ciktiModu = true
		case "--yardim", "-h":
			bicimlendirKullanim()
			return
		default:
			dosyalar = append(dosyalar, a)
		}
	}
	if len(dosyalar) == 0 {
		bicimlendirKullanim()
		os.Exit(1)
	}
	if ciktiModu && (denet || len(dosyalar) > 1) {
		fmt.Fprintln(os.Stderr, "tan biçimlendir: --cikti tek dosya ile ve --denet ile kullanılamaz")
		os.Exit(2)
	}

	kaldi := 0
	for _, d := range dosyalar {
		kaynak, err := os.ReadFile(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tan biçimlendir: %s: %v\n", d, err)
			kaldi++
			continue
		}
		sonuc, herr := bicimleKaynak(string(kaynak))
		if herr != nil {
			fmt.Fprintf(os.Stderr, "tan biçimlendir: %s: %v\n", d, herr)
			kaldi++
			continue
		}
		if denet {
			if sonuc == string(kaynak) {
				fmt.Printf("düzenli: %s\n", d)
			} else {
				fmt.Printf("biçimlendirme gerekli: %s\n", d)
				kaldi++
			}
			continue
		}
		if ciktiModu {
			fmt.Print(sonuc)
			continue
		}
		if err := os.WriteFile(d, []byte(sonuc), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "tan biçimlendir: %s: %v\n", d, err)
			kaldi++
			continue
		}
		fmt.Printf("biçimlendirildi: %s\n", d)
	}
	if kaldi > 0 {
		os.Exit(1)
	}
}
