package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ---- AST düğümleri ----
type Dugum interface{}

type SayiDugum struct {
	Deger float64 // ondalık gösterim (her zaman dolu)
	Tam   int64   // tam sayıysa kesin değer
	TamMi bool    // sayı tam mı (nokta/e içermiyor mu)
}
type MetinDugum struct{ Deger string }
type MantikDugum struct{ Deger bool }
type YokDugum struct{}
type DegiskenDugum struct {
	Ad    string
	Satir int
}

type IkiliDugum struct {
	Islec string
	Sol   Dugum
	Sag   Dugum
}

type AtamaDugum struct {
	Ad    string
	Deger Dugum
}

type YazDugum struct{ Deger Dugum }

type EgerDugum struct {
	Kosul   Dugum
	Govde   []Dugum
	Degilse []Dugum
}

type IkenDugum struct {
	Kosul Dugum
	Govde []Dugum
}

// her <öge> <liste> içinde ... son  (for-each döngüsü)
type HerDugum struct {
	Degisken string
	Liste    Dugum
	Govde    []Dugum
}

type IslevDugum struct {
	Ad           string
	Parametreler []string
	Govde        []Dugum
}

type CagriDugum struct {
	Ad         string
	Argumanlar []Dugum
	Satir      int
}

type DondurDugum struct{ Deger Dugum }

// dur (break) ve devam (continue)
type DurDugum struct{}
type DevamDugum struct{}

// içe al "dosya.tan"  (modül içe aktarma)
type IceAlDugum struct {
	Dosya string
	Satir int
}

// dene ... yakala hata ... son  (try/catch hata yönetimi)
type DeneDugum struct {
	DeneGovde   []Dugum
	YakalaGovde []Dugum
	HataAdi     string
}

// köprü çağrısı: köprü("go.matematik.karekök", 16)
type KopruDugum struct {
	Hedef      string
	Argumanlar []Dugum
	Satir      int
}

// Liste sabiti: [a, b, c]
type ListeDugum struct {
	Elemanlar []Dugum
}

// Sözlük sabiti: {"ad": deger, ...}
type SozlukDugum struct {
	Anahtarlar []Dugum
	Degerler   []Dugum
}

// İndeksleme (okuma): liste[i]
type IndeksDugum struct {
	Hedef  Dugum
	Indeks Dugum
	Satir  int
}

// İndekse atama: liste[i] = x
type IndeksAtamaDugum struct {
	Hedef  Dugum
	Indeks Dugum
	Deger  Dugum
	Satir  int
}

// ---- Parser ----
type Parser struct {
	tokenlar []Token
	konum    int
}

func YeniParser(tokenlar []Token) *Parser {
	return &Parser{tokenlar: tokenlar, konum: 0}
}

func (p *Parser) simdiki() Token { return p.tokenlar[p.konum] }
func (p *Parser) ilerle() Token {
	t := p.tokenlar[p.konum]
	if p.konum < len(p.tokenlar)-1 {
		p.konum++
	}
	return t
}

func (p *Parser) satirSonlariniAtla() {
	for p.simdiki().Tur == T_YENI_SATIR {
		p.ilerle()
	}
}

func (p *Parser) Ayristir() []Dugum {
	var deyimler []Dugum
	for p.simdiki().Tur != T_SON_DOSYA {
		p.satirSonlariniAtla()
		if p.simdiki().Tur == T_SON_DOSYA {
			break
		}
		deyimler = append(deyimler, p.deyim())
		p.satirSonlariniAtla()
	}
	return deyimler
}

// "son" görene kadar blok oku
func (p *Parser) blokOku(bitiriciler ...string) []Dugum {
	var govde []Dugum
	for {
		p.satirSonlariniAtla()
		t := p.simdiki()
		if t.Tur == T_SON_DOSYA {
			break
		}
		if t.Tur == T_ANAHTAR {
			for _, b := range bitiriciler {
				if t.Deger == b {
					return govde
				}
			}
		}
		govde = append(govde, p.deyim())
		p.satirSonlariniAtla()
	}
	return govde
}

func (p *Parser) deyim() Dugum {
	t := p.simdiki()

	if t.Tur == T_ANAHTAR {
		switch t.Deger {
		case "yaz":
			p.ilerle()
			return YazDugum{p.ifade()}
		case "eğer":
			return p.egerAyristir()
		case "iken":
			return p.ikenAyristir()
		case "her":
			return p.herAyristir()
		case "işlev":
			return p.islevAyristir()
		case "döndür":
			p.ilerle()
			return DondurDugum{p.ifade()}
		case "dur":
			p.ilerle()
			return DurDugum{}
		case "devam":
			p.ilerle()
			return DevamDugum{}
		case "içe":
			p.ilerle() // içe
			satir := p.simdiki().Satir
			if p.simdiki().Deger == "al" {
				p.ilerle() // al
			}
			dosya := p.ilerle().Deger // "dosya.tan" metni
			return IceAlDugum{dosya, satir}
		case "dene":
			return p.deneAyristir()
		}
	}

	// Hedefi ayrıştır; ardından '=' gelirse atamadır.
	sol := p.ifade()
	if p.simdiki().Tur == T_ISLEC && p.simdiki().Deger == "=" {
		p.ilerle() // =
		deger := p.ifade()
		switch h := sol.(type) {
		case DegiskenDugum:
			return AtamaDugum{h.Ad, deger}
		case IndeksDugum:
			return IndeksAtamaDugum{h.Hedef, h.Indeks, deger, h.Satir}
		}
		firlat(0, "atama hedefi geçersiz")
	}
	// yalın ifade (örn. işlev çağrısı)
	return sol
}

func (p *Parser) egerAyristir() Dugum {
	p.ilerle() // eğer
	kosul := p.ifade()
	if p.simdiki().Deger == "ise" {
		p.ilerle()
	}
	govde := p.blokOku("değilse", "son")
	var degilse []Dugum
	if p.simdiki().Deger == "değilse" {
		p.ilerle()
		// "değilse eğer" -> iç içe eğer zinciri
		if p.simdiki().Deger == "eğer" {
			degilse = []Dugum{p.egerAyristir()}
			// iç eğer kendi "son"unu tükettiği için dıştaki son'u atla
			return EgerDugum{kosul, govde, degilse}
		}
		degilse = p.blokOku("son")
	}
	if p.simdiki().Deger == "son" {
		p.ilerle()
	}
	return EgerDugum{kosul, govde, degilse}
}

// dene ... yakala hataAdi ... son
func (p *Parser) deneAyristir() Dugum {
	p.ilerle() // dene
	deneGovde := p.blokOku("yakala")
	hataAdi := ""
	var yakalaGovde []Dugum
	if p.simdiki().Deger == "yakala" {
		p.ilerle()                 // yakala
		hataAdi = p.ilerle().Deger // hata değişkeni adı
		yakalaGovde = p.blokOku("son")
	}
	if p.simdiki().Deger == "son" {
		p.ilerle()
	}
	return DeneDugum{deneGovde, yakalaGovde, hataAdi}
}

func (p *Parser) ikenAyristir() Dugum {
	p.ilerle() // iken
	kosul := p.ifade()
	if p.simdiki().Deger == "ise" {
		p.ilerle()
	}
	govde := p.blokOku("son")
	if p.simdiki().Deger == "son" {
		p.ilerle()
	}
	return IkenDugum{kosul, govde}
}

// her <öge> <liste> içinde ... son
func (p *Parser) herAyristir() Dugum {
	p.ilerle()                   // her
	degisken := p.ilerle().Deger // döngü değişkeni
	liste := p.ifade()           // gezilecek liste ifadesi
	if p.simdiki().Deger == "içinde" {
		p.ilerle()
	}
	govde := p.blokOku("son")
	if p.simdiki().Deger == "son" {
		p.ilerle()
	}
	return HerDugum{degisken, liste, govde}
}

func (p *Parser) islevAyristir() Dugum {
	p.ilerle() // işlev
	ad := p.ilerle().Deger
	p.ilerle() // (
	var parametreler []string
	for p.simdiki().Tur != T_PARANTEZ_KAPA && p.simdiki().Tur != T_SON_DOSYA {
		padTok := p.ilerle()
		if anahtarKelimeler[padTok.Deger] {
			firlat(padTok.Satir, "'%s' ayrılmış bir kelime, parametre adı olamaz", padTok.Deger)
		}
		parametreler = append(parametreler, padTok.Deger)
		if p.simdiki().Tur == T_VIRGUL {
			p.ilerle()
		}
	}
	p.ilerle() // )
	govde := p.blokOku("son")
	if p.simdiki().Deger == "son" {
		p.ilerle()
	}
	return IslevDugum{ad, parametreler, govde}
}

// ---- İfade ayrıştırma (öncelik sıralı) ----
func (p *Parser) ifade() Dugum { return p.mantiksal() }

func (p *Parser) mantiksal() Dugum {
	sol := p.karsilastirma()
	for p.simdiki().Tur == T_ANAHTAR && (p.simdiki().Deger == "ve" || p.simdiki().Deger == "veya") {
		islec := p.ilerle().Deger
		p.satirSonlariniAtla() // işleçten sonra satır sonu: ifade devam ediyor
		sag := p.karsilastirma()
		sol = IkiliDugum{islec, sol, sag}
	}
	return sol
}

func (p *Parser) karsilastirma() Dugum {
	sol := p.toplama()
	for p.simdiki().Tur == T_ISLEC {
		o := p.simdiki().Deger
		if o == ">" || o == "<" || o == ">=" || o == "<=" || o == "==" || o == "!=" {
			p.ilerle()
			p.satirSonlariniAtla()
			sag := p.toplama()
			sol = IkiliDugum{o, sol, sag}
		} else {
			break
		}
	}
	return sol
}

func (p *Parser) toplama() Dugum {
	sol := p.carpma()
	for p.simdiki().Tur == T_ISLEC && (p.simdiki().Deger == "+" || p.simdiki().Deger == "-") {
		islec := p.ilerle().Deger
		p.satirSonlariniAtla()
		sag := p.carpma()
		sol = IkiliDugum{islec, sol, sag}
	}
	return sol
}

func (p *Parser) carpma() Dugum {
	sol := p.sonEk()
	for p.simdiki().Tur == T_ISLEC && (p.simdiki().Deger == "*" || p.simdiki().Deger == "/" || p.simdiki().Deger == "%") {
		islec := p.ilerle().Deger
		p.satirSonlariniAtla()
		sag := p.sonEk()
		sol = IkiliDugum{islec, sol, sag}
	}
	return sol
}

// sonEk: birincil ifadeyi alır, ardından zincirleme [indeks] uygular.
// liste[0], matris[i][j] gibi.
func (p *Parser) sonEk() Dugum {
	dugum := p.birincil()
	for p.simdiki().Tur == T_KOSELI_AC {
		satir := p.simdiki().Satir
		p.ilerle() // [
		indeks := p.ifade()
		p.ilerle() // ]
		dugum = IndeksDugum{dugum, indeks, satir}
	}
	return dugum
}

// birincil: temel değeri okur, ardından son-ek indekslemeyi uygular
// (liste[0], zincirli m[0][1] dahil)
func (p *Parser) birincil() Dugum {
	dugum := p.temel()
	for p.simdiki().Tur == T_KOSELI_AC {
		satir := p.simdiki().Satir
		p.ilerle() // [
		indeks := p.ifade()
		p.ilerle() // ]
		dugum = IndeksDugum{dugum, indeks, satir}
	}
	return dugum
}

func (p *Parser) temel() Dugum {
	t := p.simdiki()
	// tekli eksi:  -5, -x, -(a+b)
	if t.Tur == T_ISLEC && t.Deger == "-" {
		p.ilerle()
		return IkiliDugum{"negatif", p.temel(), nil}
	}
	switch t.Tur {
	case T_SAYI:
		p.ilerle()
		// Nokta veya üs yoksa TAM SAYI olarak sakla (hassasiyet kaybı olmasın)
		if !strings.ContainsAny(t.Deger, ".eE") {
			if n, err := strconv.ParseInt(t.Deger, 10, 64); err == nil {
				return SayiDugum{Deger: float64(n), Tam: n, TamMi: true}
			}
		}
		var f float64
		fmt.Sscanf(t.Deger, "%g", &f)
		return SayiDugum{Deger: f}
	case T_METIN:
		p.ilerle()
		return MetinDugum{t.Deger}
	case T_ANAHTAR:
		if t.Deger == "doğru" {
			p.ilerle()
			return MantikDugum{true}
		}
		if t.Deger == "yanlış" {
			p.ilerle()
			return MantikDugum{false}
		}
		if t.Deger == "yok" {
			p.ilerle()
			return YokDugum{}
		}
		if t.Deger == "değil" {
			p.ilerle()
			return IkiliDugum{"değil", p.birincil(), nil}
		}
		if t.Deger == "köprü" {
			return p.kopruAyristir()
		}
	case T_PARANTEZ_AC:
		p.ilerle()
		ic := p.ifade()
		p.ilerle() // )
		return ic
	case T_KOSELI_AC:
		p.ilerle() // [
		var ogeler []Dugum
		// Köşeli parantez içinde satır sonları anlamsızdır — çok satırlı liste yazılabilsin
		for p.simdiki().Tur != T_KOSELI_KAPA && p.simdiki().Tur != T_SON_DOSYA {
			p.satirSonlariniAtla()
			if p.simdiki().Tur == T_KOSELI_KAPA {
				break
			}
			ogeler = append(ogeler, p.ifade())
			p.satirSonlariniAtla()
			if p.simdiki().Tur == T_VIRGUL {
				p.ilerle()
			}
			p.satirSonlariniAtla()
		}
		p.ilerle() // ]
		return ListeDugum{ogeler}
	case T_SUSLU_AC:
		p.ilerle() // {
		var anahtarlar []Dugum
		var degerler []Dugum
		for p.simdiki().Tur != T_SUSLU_KAPA && p.simdiki().Tur != T_SON_DOSYA {
			p.satirSonlariniAtla()
			if p.simdiki().Tur == T_SUSLU_KAPA {
				break
			}
			anahtarlar = append(anahtarlar, p.ifade())
			if p.simdiki().Tur == T_IKI_NOKTA {
				p.ilerle() // :
			}
			degerler = append(degerler, p.ifade())
			if p.simdiki().Tur == T_VIRGUL {
				p.ilerle()
			}
			p.satirSonlariniAtla()
		}
		p.ilerle() // }
		return SozlukDugum{anahtarlar, degerler}
	case T_TANIMLAYICI:
		ad := p.ilerle().Deger
		// işlev çağrısı mı?
		if p.simdiki().Tur == T_PARANTEZ_AC {
			p.ilerle()
			var argumanlar []Dugum
			for p.simdiki().Tur != T_PARANTEZ_KAPA && p.simdiki().Tur != T_SON_DOSYA {
				argumanlar = append(argumanlar, p.ifade())
				if p.simdiki().Tur == T_VIRGUL {
					p.ilerle()
				}
			}
			p.ilerle() // )
			return CagriDugum{ad, argumanlar, t.Satir}
		}
		return DegiskenDugum{ad, t.Satir}
	}
	p.ilerle()
	return YokDugum{}
}

// köprü("hedef", arg1, arg2)  -> A seçeneği, iskele katmanı
func (p *Parser) kopruAyristir() Dugum {
	p.ilerle()             // köprü
	p.ilerle()             // (
	hedefTok := p.ilerle() // "hedef" metni
	var argumanlar []Dugum
	for p.simdiki().Tur == T_VIRGUL {
		p.ilerle()
		argumanlar = append(argumanlar, p.ifade())
	}
	p.ilerle() // )
	return KopruDugum{hedefTok.Deger, argumanlar, hedefTok.Satir}
}
