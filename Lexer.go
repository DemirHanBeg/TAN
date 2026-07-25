package main

import "unicode"

// ---- Token türleri ----
type TokenTuru int

const (
	T_SON_DOSYA TokenTuru = iota
	T_SAYI
	T_METIN
	T_TANIMLAYICI // değişken/işlev adı
	T_ANAHTAR     // eğer, iken, işlev...
	T_ISLEC       // + - * / > < = ...
	T_PARANTEZ_AC
	T_PARANTEZ_KAPA
	T_KOSELI_AC
	T_KOSELI_KAPA
	T_SUSLU_AC
	T_SUSLU_KAPA
	T_IKI_NOKTA
	T_VIRGUL
	T_YENI_SATIR
)

type Token struct {
	Tur   TokenTuru
	Deger string
	Satir int
}

// Tan'ın anahtar kelimeleri (en Türkçe olanlar seçildi)
var anahtarKelimeler = map[string]bool{
	"yaz": true, "eğer": true, "ise": true, "değilse": true,
	"iken": true, "işlev": true, "döndür": true, "son": true,
	"her": true, "içinde": true,
	"dur": true, "devam": true, "içe": true, "al": true,
	"doğru": true, "yanlış": true, "yok": true,
	"ve": true, "veya": true, "değil": true,
	"köprü": true, // A seçeneği: dış yetenek çağırma
	"dene":  true, "yakala": true,
}

type Lexer struct {
	girdi []rune
	konum int
	satir int
}

func YeniLexer(kaynak string) *Lexer {
	return &Lexer{girdi: []rune(kaynak), konum: 0, satir: 1}
}

func (l *Lexer) simdiki() rune {
	if l.konum >= len(l.girdi) {
		return 0
	}
	return l.girdi[l.konum]
}

func (l *Lexer) sonraki() rune {
	if l.konum+1 >= len(l.girdi) {
		return 0
	}
	return l.girdi[l.konum+1]
}

// Türkçe harfleri de tanımlayıcı sayar (ç ğ ı ö ş ü ve büyükleri)
func harfMi(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func (l *Lexer) Tokenle() []Token {
	var tokenlar []Token
	for l.konum < len(l.girdi) {
		r := l.simdiki()

		// Boşluk ve tab atla (yeni satır anlamlı)
		if r == ' ' || r == '\t' || r == '\r' {
			l.konum++
			continue
		}
		// Yorum: # ... satır sonuna kadar
		if r == '#' {
			for l.konum < len(l.girdi) && l.simdiki() != '\n' {
				l.konum++
			}
			continue
		}
		// Yeni satır (deyim ayırıcı)
		if r == '\n' {
			tokenlar = append(tokenlar, Token{T_YENI_SATIR, "\\n", l.satir})
			l.satir++
			l.konum++
			continue
		}
		// Metin: "..."
		if r == '"' {
			l.konum++
			var sb []rune
			for l.konum < len(l.girdi) && l.simdiki() != '"' {
				c := l.simdiki()
				if c == '\\' && l.konum+1 < len(l.girdi) {
					l.konum++
					kacis := l.simdiki()
					switch kacis {
					case 'n':
						sb = append(sb, '\n')
					case 't':
						sb = append(sb, '\t')
					case 'r':
						sb = append(sb, '\r')
					case '"':
						sb = append(sb, '"')
					case '\\':
						sb = append(sb, '\\')
					default:
						sb = append(sb, kacis)
					}
					l.konum++
					continue
				}
				sb = append(sb, c)
				l.konum++
			}
			l.konum++ // kapanan "
			tokenlar = append(tokenlar, Token{T_METIN, string(sb), l.satir})
			continue
		}
		// Sayı (tek tip: tam + ondalık bir arada)
		if unicode.IsDigit(r) {
			bas := l.konum
			for l.konum < len(l.girdi) && (unicode.IsDigit(l.simdiki()) || l.simdiki() == '.') {
				l.konum++
			}
			tokenlar = append(tokenlar, Token{T_SAYI, string(l.girdi[bas:l.konum]), l.satir})
			continue
		}
		// Tanımlayıcı / anahtar kelime
		if harfMi(r) {
			bas := l.konum
			for l.konum < len(l.girdi) && (harfMi(l.simdiki()) || unicode.IsDigit(l.simdiki())) {
				l.konum++
			}
			kelime := string(l.girdi[bas:l.konum])
			if anahtarKelimeler[kelime] {
				tokenlar = append(tokenlar, Token{T_ANAHTAR, kelime, l.satir})
			} else {
				tokenlar = append(tokenlar, Token{T_TANIMLAYICI, kelime, l.satir})
			}
			continue
		}
		// Parantez ve virgül
		if r == '(' {
			tokenlar = append(tokenlar, Token{T_PARANTEZ_AC, "(", l.satir})
			l.konum++
			continue
		}
		if r == ')' {
			tokenlar = append(tokenlar, Token{T_PARANTEZ_KAPA, ")", l.satir})
			l.konum++
			continue
		}
		if r == '[' {
			tokenlar = append(tokenlar, Token{T_KOSELI_AC, "[", l.satir})
			l.konum++
			continue
		}
		if r == ']' {
			tokenlar = append(tokenlar, Token{T_KOSELI_KAPA, "]", l.satir})
			l.konum++
			continue
		}
		if r == '{' {
			tokenlar = append(tokenlar, Token{T_SUSLU_AC, "{", l.satir})
			l.konum++
			continue
		}
		if r == '}' {
			tokenlar = append(tokenlar, Token{T_SUSLU_KAPA, "}", l.satir})
			l.konum++
			continue
		}
		if r == ':' {
			tokenlar = append(tokenlar, Token{T_IKI_NOKTA, ":", l.satir})
			l.konum++
			continue
		}
		if r == ',' {
			tokenlar = append(tokenlar, Token{T_VIRGUL, ",", l.satir})
			l.konum++
			continue
		}
		// İşleçler (çok karakterli önce: >= <= == !=)
		if r == '>' || r == '<' || r == '=' || r == '!' {
			if l.sonraki() == '=' {
				tokenlar = append(tokenlar, Token{T_ISLEC, string([]rune{r, '='}), l.satir})
				l.konum += 2
				continue
			}
		}
		if r == '+' || r == '-' || r == '*' || r == '/' || r == '%' || r == '>' || r == '<' || r == '=' {
			tokenlar = append(tokenlar, Token{T_ISLEC, string(r), l.satir})
			l.konum++
			continue
		}
		// Tanınmayan karakter -> atla
		l.konum++
	}
	tokenlar = append(tokenlar, Token{T_SON_DOSYA, "", l.satir})
	return tokenlar
}
