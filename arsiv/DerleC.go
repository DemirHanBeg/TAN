//go:build !js

package main

// ============================================================
// Tan -> C native derleyici (v2) — DEĞER RUNTIME'LI
// Sayı + metin + liste destekli. String birleştirme, indeksleme,
// dosya oku/yaz, çalıştır. Bu backend, Tan'la yazılmış bir
// derleyiciyi (Tanc.tan) derleyecek kadar zengin. Go runtime YOK;
// çıktı saf C -> ELF, tek bağımlılık libc.
// ============================================================

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const cRuntime = `#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

typedef enum { T_SAYI, T_METIN, T_LISTE } Tur;
typedef struct Deger {
    Tur tur;
    double sayi;
    char* metin;
    struct Deger* ogeler;
    int uzunluk;
} Deger;

static Deger d_sayi(double x){ Deger d; d.tur=T_SAYI; d.sayi=x; d.metin=0; d.ogeler=0; d.uzunluk=0; return d; }
static Deger d_metin(const char* s){ Deger d; d.tur=T_METIN; d.sayi=0; d.metin=strdup(s); d.ogeler=0; d.uzunluk=0; return d; }

static char* sayi_metne(double x){ char* b=malloc(64); if(x==(long long)x) snprintf(b,64,"%lld",(long long)x); else snprintf(b,64,"%g",x); return b; }
static char* deger_metne(Deger d){ if(d.tur==T_METIN) return strdup(d.metin?d.metin:""); if(d.tur==T_SAYI) return sayi_metne(d.sayi); return strdup("[liste]"); }

static int d_dogru_mu(Deger d){ if(d.tur==T_SAYI) return d.sayi!=0; if(d.tur==T_METIN) return d.metin&&d.metin[0]; return d.uzunluk>0; }

static Deger d_topla(Deger a, Deger b){
    if(a.tur==T_METIN||b.tur==T_METIN){ char* sa=deger_metne(a); char* sb=deger_metne(b); char* r=malloc(strlen(sa)+strlen(sb)+1); strcpy(r,sa); strcat(r,sb); Deger d; d.tur=T_METIN; d.sayi=0; d.metin=r; d.ogeler=0; d.uzunluk=0; free(sa); free(sb); return d; }
    return d_sayi(a.sayi+b.sayi);
}
static Deger d_cikar(Deger a, Deger b){ return d_sayi(a.sayi-b.sayi); }
static Deger d_carp(Deger a, Deger b){ return d_sayi(a.sayi*b.sayi); }
static Deger d_bol(Deger a, Deger b){ return d_sayi(a.sayi/b.sayi); }
static Deger d_mod(Deger a, Deger b){ return d_sayi(fmod(a.sayi,b.sayi)); }

static int d_karsilastir(Deger a, Deger b){
    if(a.tur==T_METIN&&b.tur==T_METIN) return strcmp(a.metin,b.metin);
    if(a.sayi<b.sayi) return -1; if(a.sayi>b.sayi) return 1; return 0;
}
static Deger d_esit(Deger a, Deger b){ if(a.tur!=b.tur) return d_sayi(0); return d_sayi(d_karsilastir(a,b)==0?1:0); }
static Deger d_esit_degil(Deger a, Deger b){ return d_sayi(d_dogru_mu(d_esit(a,b))?0:1); }
static Deger d_buyuk(Deger a, Deger b){ return d_sayi(d_karsilastir(a,b)>0?1:0); }
static Deger d_kucuk(Deger a, Deger b){ return d_sayi(d_karsilastir(a,b)<0?1:0); }
static Deger d_buyuk_esit(Deger a, Deger b){ return d_sayi(d_karsilastir(a,b)>=0?1:0); }
static Deger d_kucuk_esit(Deger a, Deger b){ return d_sayi(d_karsilastir(a,b)<=0?1:0); }
static Deger d_ve(Deger a, Deger b){ return d_sayi((d_dogru_mu(a)&&d_dogru_mu(b))?1:0); }
static Deger d_veya(Deger a, Deger b){ return d_sayi((d_dogru_mu(a)||d_dogru_mu(b))?1:0); }
static Deger d_degil(Deger a){ return d_sayi(d_dogru_mu(a)?0:1); }

static void d_yaz(Deger d){ if(d.tur==T_METIN) printf("%s\n", d.metin?d.metin:""); else if(d.tur==T_SAYI){ char* s=sayi_metne(d.sayi); printf("%s\n",s); free(s);} else printf("[liste:%d]\n", d.uzunluk); }

/* --- liste --- */
static Deger d_liste_yap(int n, ...){ Deger d; d.tur=T_LISTE; d.sayi=0; d.metin=0; d.uzunluk=n; d.ogeler = n? malloc(sizeof(Deger)*n):0; va_list ap; va_start(ap,n); for(int i=0;i<n;i++) d.ogeler[i]=va_arg(ap,Deger); va_end(ap); return d; }
static Deger d_indeks(Deger l, Deger i){ int k=(int)i.sayi; if(l.tur==T_METIN){ int n=strlen(l.metin); if(k<0||k>=n) return d_metin(""); char b[2]; b[0]=l.metin[k]; b[1]=0; return d_metin(b);} if(l.tur==T_LISTE){ if(k<0||k>=l.uzunluk) return d_sayi(0); return l.ogeler[k]; } return d_sayi(0); }
static Deger d_ekle(Deger l, Deger o){ Deger d; d.tur=T_LISTE; d.sayi=0; d.metin=0; d.uzunluk=l.uzunluk+1; d.ogeler=malloc(sizeof(Deger)*d.uzunluk); for(int i=0;i<l.uzunluk;i++) d.ogeler[i]=l.ogeler[i]; d.ogeler[l.uzunluk]=o; return d; }

/* --- metin/değer builtinleri (ASCII/byte) --- */
static Deger d_uzunluk(Deger d){ if(d.tur==T_METIN) return d_sayi(strlen(d.metin)); if(d.tur==T_LISTE) return d_sayi(d.uzunluk); return d_sayi(0); }
static Deger d_karakter(Deger n){ char b[2]; b[0]=(char)(int)n.sayi; b[1]=0; return d_metin(b); }
static Deger d_kod(Deger s){ if(s.tur==T_METIN&&s.metin[0]) return d_sayi((unsigned char)s.metin[0]); return d_sayi(0); }
static Deger d_sayi_cevir(Deger s){ if(s.tur==T_SAYI) return s; double f=0; if(s.tur==T_METIN) f=atof(s.metin); return d_sayi(f); }
static Deger d_metin_cevir(Deger x){ char* s=deger_metne(x); Deger d; d.tur=T_METIN; d.sayi=0; d.metin=s; d.ogeler=0; d.uzunluk=0; return d; }
static Deger d_oku(Deger yol){ FILE* f=fopen(yol.metin,"rb"); if(!f) return d_metin(""); fseek(f,0,SEEK_END); long n=ftell(f); fseek(f,0,SEEK_SET); char* b=malloc(n+1); size_t _r=fread(b,1,n,f); (void)_r; b[n]=0; fclose(f); Deger d; d.tur=T_METIN; d.sayi=0; d.metin=b; d.ogeler=0; d.uzunluk=0; return d; }
static Deger d_yaz_dosya(Deger yol, Deger icerik){ FILE* f=fopen(yol.metin,"wb"); if(f){ char* s=deger_metne(icerik); fwrite(s,1,strlen(s),f); fclose(f); free(s);} return d_sayi(0); }
static Deger d_calistir(int n, ...){ char kom[4096]; kom[0]=0; va_list ap; va_start(ap,n); for(int i=0;i<n;i++){ Deger a=va_arg(ap,Deger); char* s=deger_metne(a); if(i) strcat(kom," "); strcat(kom,s); free(s);} va_end(ap); FILE* p=popen(kom,"r"); if(!p) return d_metin(""); char* out=malloc(65536); int t=0; int c; while((c=fgetc(p))!=EOF && t<65535) out[t++]=c; out[t]=0; pclose(p); Deger d; d.tur=T_METIN; d.sayi=0; d.metin=out; d.ogeler=0; d.uzunluk=0; return d; }
static Deger d_birlestir(Deger l){ char* r=malloc(1); r[0]=0; int cap=1; for(int i=0;i<l.uzunluk;i++){ char* s=deger_metne(l.ogeler[i]); cap+=strlen(s); r=realloc(r,cap); strcat(r,s); free(s);} Deger d; d.tur=T_METIN; d.sayi=0; d.metin=r; d.ogeler=0; d.uzunluk=0; return d; }
static Deger d_parcala(Deger s, Deger sep){ Deger d; d.tur=T_LISTE; d.sayi=0; d.metin=0; d.uzunluk=0; d.ogeler=0; char* metin=strdup(s.metin); char* ay=sep.metin; int alen=strlen(ay); char* p=metin; while(1){ char* bul = alen? strstr(p,ay):0; if(!bul){ d=d_ekle(d,d_metin(p)); break;} *bul=0; d=d_ekle(d,d_metin(p)); p=bul+alen; } free(metin); return d; }

/* --- program argümanları --- */
static int _argc; static char** _argv;
static Deger d_argsay(){ return d_sayi(_argc); }
static Deger d_arg(Deger i){ int k=(int)i.sayi; if(k<0||k>=_argc) return d_metin(""); return d_metin(_argv[k]); }
`

// ---- Go tarafı: kod üreteci ----

type cUretici struct {
	sb      strings.Builder
	girinti int
	sayac   int
}

func (c *cUretici) satir(kod string) {
	c.sb.WriteString(strings.Repeat("    ", c.girinti))
	c.sb.WriteString(kod)
	c.sb.WriteByte('\n')
}

func cAd(ad string) string {
	var b strings.Builder
	b.WriteString("t_")
	for _, r := range ad {
		switch r {
		case 'ç':
			b.WriteByte('c')
		case 'Ç':
			b.WriteByte('C')
		case 'ş':
			b.WriteByte('s')
		case 'Ş':
			b.WriteByte('S')
		case 'ğ':
			b.WriteByte('g')
		case 'Ğ':
			b.WriteByte('G')
		case 'ü':
			b.WriteByte('u')
		case 'Ü':
			b.WriteByte('U')
		case 'ö':
			b.WriteByte('o')
		case 'Ö':
			b.WriteByte('O')
		case 'ı':
			b.WriteByte('i')
		case 'İ':
			b.WriteByte('I')
		default:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
	}
	return b.String()
}

func cMetin(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

var builtinC = map[string]string{
	"uzunluk": "d_uzunluk", "karakter": "d_karakter", "kod": "d_kod",
	"sayı": "d_sayi_cevir", "metin": "d_metin_cevir", "oku": "d_oku",
	"yaz_dosya": "d_yaz_dosya", "birleştir": "d_birlestir", "parçala": "d_parcala",
	"ekle": "d_ekle", "arg": "d_arg", "argsay": "d_argsay",
}

func (c *cUretici) ifade(d Dugum) string {
	switch n := d.(type) {
	case SayiDugum:
		return "d_sayi(" + strconv.FormatFloat(n.Deger, 'g', -1, 64) + ")"
	case MetinDugum:
		return "d_metin(" + cMetin(n.Deger) + ")"
	case MantikDugum:
		if n.Deger {
			return "d_sayi(1)"
		}
		return "d_sayi(0)"
	case YokDugum:
		return "d_sayi(0)"
	case DegiskenDugum:
		return cAd(n.Ad)
	case ListeDugum:
		parcalar := make([]string, len(n.Elemanlar))
		for i, e := range n.Elemanlar {
			parcalar[i] = c.ifade(e)
		}
		return fmt.Sprintf("d_liste_yap(%d%s)", len(n.Elemanlar), oncekiVirgul(parcalar))
	case IndeksDugum:
		return fmt.Sprintf("d_indeks(%s, %s)", c.ifade(n.Hedef), c.ifade(n.Indeks))
	case IkiliDugum:
		sol := c.ifade(n.Sol)
		sag := c.ifade(n.Sag)
		fn := map[string]string{"+": "d_topla", "-": "d_cikar", "*": "d_carp", "/": "d_bol", "%": "d_mod",
			"==": "d_esit", "!=": "d_esit_degil", ">": "d_buyuk", "<": "d_kucuk", ">=": "d_buyuk_esit", "<=": "d_kucuk_esit",
			"ve": "d_ve", "veya": "d_veya"}[n.Islec]
		if fn == "" {
			panic(TanHata{Mesaj: "C derleyici: bilinmeyen işleç '" + n.Islec + "'"})
		}
		return fmt.Sprintf("%s(%s, %s)", fn, sol, sag)
	case CagriDugum:
		args := make([]string, len(n.Argumanlar))
		for i, a := range n.Argumanlar {
			args[i] = c.ifade(a)
		}
		if n.Ad == "çalıştır" {
			return fmt.Sprintf("d_calistir(%d%s)", len(args), oncekiVirgul(args))
		}
		if cfn, ok := builtinC[n.Ad]; ok {
			return fmt.Sprintf("%s(%s)", cfn, strings.Join(args, ", "))
		}
		return fmt.Sprintf("%s(%s)", cAd(n.Ad), strings.Join(args, ", "))
	default:
		panic(TanHata{Mesaj: fmt.Sprintf("C derleyici: bu ifade henüz desteklenmiyor (%T)", d)})
	}
}

func oncekiVirgul(p []string) string {
	if len(p) == 0 {
		return ""
	}
	return ", " + strings.Join(p, ", ")
}

func (c *cUretici) deyim(d Dugum) {
	switch n := d.(type) {
	case AtamaDugum:
		c.satir(fmt.Sprintf("%s = %s;", cAd(n.Ad), c.ifade(n.Deger)))
	case IndeksAtamaDugum:
		c.satir(fmt.Sprintf("{ Deger* _h=&%s; int _i=(int)(%s).sayi; if(_h->tur==T_LISTE && _i>=0 && _i<_h->uzunluk) _h->ogeler[_i]=%s; }",
			c.ifade(n.Hedef), c.ifade(n.Indeks), c.ifade(n.Deger)))
	case YazDugum:
		c.satir(fmt.Sprintf("d_yaz(%s);", c.ifade(n.Deger)))
	case EgerDugum:
		c.satir(fmt.Sprintf("if (d_dogru_mu(%s)) {", c.ifade(n.Kosul)))
		c.girinti++
		for _, s := range n.Govde {
			c.deyim(s)
		}
		c.girinti--
		if len(n.Degilse) > 0 {
			c.satir("} else {")
			c.girinti++
			for _, s := range n.Degilse {
				c.deyim(s)
			}
			c.girinti--
		}
		c.satir("}")
	case IkenDugum:
		c.satir(fmt.Sprintf("while (d_dogru_mu(%s)) {", c.ifade(n.Kosul)))
		c.girinti++
		for _, s := range n.Govde {
			c.deyim(s)
		}
		c.girinti--
		c.satir("}")
	case HerDugum:
		c.sayac++
		lst := fmt.Sprintf("_lst%d", c.sayac)
		idx := fmt.Sprintf("_i%d", c.sayac)
		c.satir(fmt.Sprintf("{ Deger %s = %s; for(int %s=0; %s<%s.uzunluk; %s++) {", lst, c.ifade(n.Liste), idx, idx, lst, idx))
		c.girinti++
		c.satir(fmt.Sprintf("Deger %s = %s.ogeler[%s];", cAd(n.Degisken), lst, idx))
		for _, s := range n.Govde {
			c.deyim(s)
		}
		c.girinti--
		c.satir("} }")
	case DondurDugum:
		c.satir(fmt.Sprintf("return %s;", c.ifade(n.Deger)))
	case DurDugum:
		c.satir("break;")
	case DevamDugum:
		c.satir("continue;")
	case CagriDugum:
		c.satir(c.ifade(n) + ";")
	default:
		panic(TanHata{Mesaj: fmt.Sprintf("C derleyici: bu deyim henüz desteklenmiyor (%T)", d)})
	}
}

func degiskenleriTopla(govde []Dugum, kume map[string]bool) {
	for _, d := range govde {
		switch n := d.(type) {
		case AtamaDugum:
			kume[n.Ad] = true
		case EgerDugum:
			degiskenleriTopla(n.Govde, kume)
			degiskenleriTopla(n.Degilse, kume)
		case IkenDugum:
			degiskenleriTopla(n.Govde, kume)
		case HerDugum:
			degiskenleriTopla(n.Govde, kume)
		}
	}
}

func (c *cUretici) islevYaz(n IslevDugum) {
	parametreler := make([]string, len(n.Parametreler))
	for i, p := range n.Parametreler {
		parametreler[i] = "Deger " + cAd(p)
	}
	c.satir(fmt.Sprintf("Deger %s(%s) {", cAd(n.Ad), strings.Join(parametreler, ", ")))
	c.girinti++
	yerel := map[string]bool{}
	degiskenleriTopla(n.Govde, yerel)
	for _, p := range n.Parametreler {
		delete(yerel, p)
	}
	for ad := range yerel {
		c.satir(fmt.Sprintf("Deger %s = d_sayi(0);", cAd(ad)))
	}
	for _, s := range n.Govde {
		c.deyim(s)
	}
	c.satir("return d_sayi(0);")
	c.girinti--
	c.satir("}")
	c.satir("")
}

func derleC(dosya string, cikti string) {
	kaynak, err := os.ReadFile(dosya)
	if err != nil {
		fmt.Printf("Dosya okunamadı: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if r := recover(); r != nil {
			if h, ok := r.(TanHata); ok {
				fmt.Fprintln(os.Stderr, "Derleme hatası: "+h.Mesaj)
				os.Exit(1)
			}
			panic(r)
		}
	}()

	lexer := YeniLexer(string(kaynak))
	parser := YeniParser(lexer.Tokenle())
	agac := parser.Ayristir()

	// --- OPTIMIZE GECISI: sabit katlama, cebirsel sadelestirme, olu kod ---
	opt := YeniOptimizer()
	agac = opt.Govde(agac)
	if os.Getenv("TAN_OPT_RAPOR") != "" {
		fmt.Fprintf(os.Stderr, "optimize: %d katlama, %d ölü blok\n", opt.Katlanan, opt.Silinen)
	}

	var islevler []IslevDugum
	var anaGovde []Dugum
	for _, d := range agac {
		if isv, ok := d.(IslevDugum); ok {
			islevler = append(islevler, isv)
		} else {
			anaGovde = append(anaGovde, d)
		}
	}

	c := &cUretici{}
	c.sb.WriteString("// Tan derleyicisi tarafından üretildi — saf C, Go yok.\n")
	c.sb.WriteString("#include <stdarg.h>\n")
	c.sb.WriteString(cRuntime)
	c.sb.WriteString("\n")

	for _, isv := range islevler {
		tipler := make([]string, len(isv.Parametreler))
		for i := range tipler {
			tipler[i] = "Deger"
		}
		c.satir(fmt.Sprintf("Deger %s(%s);", cAd(isv.Ad), strings.Join(tipler, ", ")))
	}
	c.satir("")
	for _, isv := range islevler {
		c.islevYaz(isv)
	}

	c.satir("int main(int argc, char** argv) {")
	c.girinti++
	c.satir("_argc=argc; _argv=argv;")
	yerel := map[string]bool{}
	degiskenleriTopla(anaGovde, yerel)
	for ad := range yerel {
		c.satir(fmt.Sprintf("Deger %s = d_sayi(0);", cAd(ad)))
	}
	for _, s := range anaGovde {
		c.deyim(s)
	}
	c.satir("return 0;")
	c.girinti--
	c.satir("}")

	// Ara C'yi geçici dizine yaz (paket dizinini kirletmesin, go build'i kırmasın)
	tmp, err := os.MkdirTemp("", "tanc")
	if err != nil {
		fmt.Printf("geçici dizin açılamadı: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)
	cDosya := tmp + "/uretilen.c"
	if err := os.WriteFile(cDosya, []byte(c.sb.String()), 0644); err != nil {
		fmt.Printf("C yazılamadı: %v\n", err)
		os.Exit(1)
	}
	komut := exec.Command("gcc", "-O2", "-o", cikti, cDosya, "-lm")
	komut.Stdout = os.Stdout
	komut.Stderr = os.Stderr
	if err := komut.Run(); err != nil {
		fmt.Printf("gcc hatası: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Native binary üretildi: %s\n", cikti)
}
