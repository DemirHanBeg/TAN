//go:build !js

package main

// ============================================================
// Tan -> x86-64 ASSEMBLY doğrudan kod üreteci
// ------------------------------------------------------------
// C YOK. gcc YOK. libc YOK.
// AST -> x86-64 assembly (.s) -> as -> ld -> statik ELF
// Çıktı: ham syscall kullanan, hiçbir kütüphaneye bağlı olmayan binary.
// Kapsam: tam sayı aritmetiği, değişken, karşılaştırma, ve/veya,
// eğer/değilse, iken, dur/devam, işlev/döndür (özyineleme dahil),
// yaz (sayı ifadesi + metin sabiti).
// ============================================================

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type asmUretici struct {
	sb       strings.Builder
	etiketNo int
	metinler []string        // .rodata metin sabitleri
	yereller map[string]int  // işlev içi: ad -> rbp offset
	genel    map[string]bool // üst düzey değişkenler (.bss)
	donguSon []string        // dur/devam için etiket yığını
	donguBas []string
}

func (a *asmUretici) yaz(format string, args ...interface{}) {
	a.sb.WriteString("    " + fmt.Sprintf(format, args...) + "\n")
}

func (a *asmUretici) etiket(ad string) {
	a.sb.WriteString(ad + ":\n")
}

func (a *asmUretici) yeniEtiket(on string) string {
	a.etiketNo++
	return fmt.Sprintf(".L%s%d", on, a.etiketNo)
}

func asmAd(ad string) string {
	var b strings.Builder
	b.WriteString("v_")
	for _, r := range ad {
		switch r {
		case 'ç':
			b.WriteByte('c')
		case 'ş':
			b.WriteByte('s')
		case 'ğ':
			b.WriteByte('g')
		case 'ü':
			b.WriteByte('u')
		case 'ö':
			b.WriteByte('o')
		case 'ı':
			b.WriteByte('i')
		case 'İ':
			b.WriteByte('I')
		case 'Ç':
			b.WriteByte('C')
		case 'Ş':
			b.WriteByte('S')
		case 'Ğ':
			b.WriteByte('G')
		case 'Ü':
			b.WriteByte('U')
		case 'Ö':
			b.WriteByte('O')
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

// Bir değişkenin bellek adresi (yerel mi genel mi)
func (a *asmUretici) yer(ad string) string {
	if a.yereller != nil {
		if off, ok := a.yereller[ad]; ok {
			return fmt.Sprintf("qword ptr [rbp%+d]", off)
		}
	}
	a.genel[ad] = true
	return fmt.Sprintf("qword ptr [%s]", asmAd(ad))
}

// İfadeyi değerlendir, sonuç RAX'te
func (a *asmUretici) ifade(d Dugum) {
	switch n := d.(type) {
	case SayiDugum:
		if n.TamMi {
			a.yaz("mov rax, %d", n.Tam) // kesin int64
		} else {
			panic(TanHata{Mesaj: fmt.Sprintf(
				"asm backend ondalık sayı desteklemiyor (%g). Tam sayı kullan veya 'tan derle' (C yolu) ile derle.", n.Deger)})
		}

	case MantikDugum:
		if n.Deger {
			a.yaz("mov rax, 1")
		} else {
			a.yaz("mov rax, 0")
		}

	case YokDugum:
		a.yaz("mov rax, 0")

	case DegiskenDugum:
		a.yaz("mov rax, %s", a.yer(n.Ad))

	case IkiliDugum:
		a.ifade(n.Sol)
		a.yaz("push rax")
		a.ifade(n.Sag)
		a.yaz("mov rcx, rax")
		a.yaz("pop rax")
		switch n.Islec {
		case "+":
			a.yaz("add rax, rcx")
		case "-":
			a.yaz("sub rax, rcx")
		case "*":
			a.yaz("imul rax, rcx")
		case "/":
			a.yaz("cqo")
			a.yaz("idiv rcx")
		case "%":
			a.yaz("cqo")
			a.yaz("idiv rcx")
			a.yaz("mov rax, rdx")
		case "==", "!=", ">", "<", ">=", "<=":
			komut := map[string]string{"==": "sete", "!=": "setne", ">": "setg", "<": "setl", ">=": "setge", "<=": "setle"}[n.Islec]
			a.yaz("cmp rax, rcx")
			a.yaz("%s al", komut)
			a.yaz("movzx rax, al")
		case "ve":
			a.yaz("test rax, rax")
			a.yaz("setne al")
			a.yaz("movzx rax, al")
			a.yaz("test rcx, rcx")
			a.yaz("setne cl")
			a.yaz("movzx rcx, cl")
			a.yaz("and rax, rcx")
		case "veya":
			a.yaz("test rax, rax")
			a.yaz("setne al")
			a.yaz("movzx rax, al")
			a.yaz("test rcx, rcx")
			a.yaz("setne cl")
			a.yaz("movzx rcx, cl")
			a.yaz("or rax, rcx")
		default:
			panic(TanHata{Mesaj: "asm: bilinmeyen işleç '" + n.Islec + "'"})
		}

	case CagriDugum:
		if n.Ad == "değil" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "asm: değil() tek argüman ister"})
			}
			a.ifade(n.Argumanlar[0])
			a.yaz("test rax, rax")
			a.yaz("sete al")
			a.yaz("movzx rax, al")
			return
		}
		if len(n.Argumanlar) > 6 {
			panic(TanHata{Mesaj: "asm: en fazla 6 argüman desteklenir"})
		}
		for _, arg := range n.Argumanlar {
			a.ifade(arg)
			a.yaz("push rax")
		}
		kayitlar := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
		for i := len(n.Argumanlar) - 1; i >= 0; i-- {
			a.yaz("pop %s", kayitlar[i])
		}
		a.yaz("call %s", asmAd(n.Ad))

	case MetinDugum:
		panic(TanHata{Mesaj: "asm: metin değerleri bu backend'de sadece yaz() içinde sabit olarak desteklenir"})

	default:
		panic(TanHata{Mesaj: fmt.Sprintf("asm: bu ifade desteklenmiyor (%T)", d)})
	}
}

func (a *asmUretici) deyim(d Dugum) {
	switch n := d.(type) {
	case AtamaDugum:
		a.ifade(n.Deger)
		a.yaz("mov %s, rax", a.yer(n.Ad))

	case YazDugum:
		if m, ok := n.Deger.(MetinDugum); ok {
			etk := fmt.Sprintf(".Lmetin%d", len(a.metinler))
			a.metinler = append(a.metinler, m.Deger)
			a.yaz("lea rsi, [%s]", etk)
			a.yaz("mov rdx, %d", len(m.Deger)+1)
			a.yaz("call yaz_metin")
		} else {
			a.ifade(n.Deger)
			a.yaz("mov rdi, rax")
			a.yaz("call yaz_sayi")
		}

	case EgerDugum:
		degilseEtk := a.yeniEtiket("degilse")
		sonEtk := a.yeniEtiket("egerson")
		a.ifade(n.Kosul)
		a.yaz("test rax, rax")
		a.yaz("jz %s", degilseEtk)
		for _, s := range n.Govde {
			a.deyim(s)
		}
		a.yaz("jmp %s", sonEtk)
		a.etiket(degilseEtk)
		for _, s := range n.Degilse {
			a.deyim(s)
		}
		a.etiket(sonEtk)

	case IkenDugum:
		basEtk := a.yeniEtiket("dongubas")
		sonEtk := a.yeniEtiket("donguson")
		a.donguBas = append(a.donguBas, basEtk)
		a.donguSon = append(a.donguSon, sonEtk)
		a.etiket(basEtk)
		a.ifade(n.Kosul)
		a.yaz("test rax, rax")
		a.yaz("jz %s", sonEtk)
		for _, s := range n.Govde {
			a.deyim(s)
		}
		a.yaz("jmp %s", basEtk)
		a.etiket(sonEtk)
		a.donguBas = a.donguBas[:len(a.donguBas)-1]
		a.donguSon = a.donguSon[:len(a.donguSon)-1]

	case DurDugum:
		if len(a.donguSon) == 0 {
			panic(TanHata{Mesaj: "asm: dur döngü dışında"})
		}
		a.yaz("jmp %s", a.donguSon[len(a.donguSon)-1])

	case DevamDugum:
		if len(a.donguBas) == 0 {
			panic(TanHata{Mesaj: "asm: devam döngü dışında"})
		}
		a.yaz("jmp %s", a.donguBas[len(a.donguBas)-1])

	case DondurDugum:
		if n.Deger != nil {
			a.ifade(n.Deger)
		} else {
			a.yaz("mov rax, 0")
		}
		a.yaz("leave")
		a.yaz("ret")

	case CagriDugum:
		a.ifade(n)

	default:
		panic(TanHata{Mesaj: fmt.Sprintf("asm: bu deyim desteklenmiyor (%T)", d)})
	}
}

func asmDegiskenleriTopla(govde []Dugum, kume map[string]bool) {
	for _, d := range govde {
		switch n := d.(type) {
		case AtamaDugum:
			kume[n.Ad] = true
		case EgerDugum:
			asmDegiskenleriTopla(n.Govde, kume)
			asmDegiskenleriTopla(n.Degilse, kume)
		case IkenDugum:
			asmDegiskenleriTopla(n.Govde, kume)
		}
	}
}

func (a *asmUretici) islevYaz(n IslevDugum) {
	a.etiket(asmAd(n.Ad))
	a.yaz("push rbp")
	a.yaz("mov rbp, rsp")

	// yerel yerleşim: parametreler + atanan değişkenler
	yereller := map[string]int{}
	sira := []string{}
	for _, p := range n.Parametreler {
		sira = append(sira, p)
	}
	atananlar := map[string]bool{}
	asmDegiskenleriTopla(n.Govde, atananlar)
	for ad := range atananlar {
		var varmi bool
		for _, p := range n.Parametreler {
			if p == ad {
				varmi = true
			}
		}
		if !varmi {
			sira = append(sira, ad)
		}
	}
	for i, ad := range sira {
		yereller[ad] = -8 * (i + 1)
	}
	cerceve := 8 * len(sira)
	if cerceve%16 != 0 {
		cerceve += 8
	}
	if cerceve > 0 {
		a.yaz("sub rsp, %d", cerceve)
	}

	// parametreleri kayıtlardan yerel yuvalara taşı
	kayitlar := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
	for i, p := range n.Parametreler {
		a.yaz("mov qword ptr [rbp%+d], %s", yereller[p], kayitlar[i])
	}
	// diğer yerelleri sıfırla
	for _, ad := range sira[len(n.Parametreler):] {
		a.yaz("mov qword ptr [rbp%+d], 0", yereller[ad])
	}

	eskiYereller := a.yereller
	a.yereller = yereller
	for _, s := range n.Govde {
		a.deyim(s)
	}
	a.yereller = eskiYereller

	a.yaz("mov rax, 0")
	a.yaz("leave")
	a.yaz("ret")
	a.sb.WriteString("\n")
}

// derleAsm: .tan -> x86-64 assembly -> as -> ld -> statik ELF (C ve libc YOK)
func derleAsm(dosya string, cikti string, asmSakla bool) {
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

	a := &asmUretici{genel: map[string]bool{}}

	// --- gövde kodunu önce üret (hangi genel değişkenler kullanılıyor öğrenilsin) ---
	var govde strings.Builder
	a.sb = strings.Builder{}
	for _, isv := range islevler {
		a.islevYaz(isv)
	}
	a.etiket("_start")
	for _, s := range anaGovde {
		a.deyim(s)
	}
	a.yaz("mov rax, 60")
	a.yaz("xor rdi, rdi")
	a.yaz("syscall")
	govde.WriteString(a.sb.String())

	// --- nihai dosyayı birleştir ---
	var son strings.Builder
	son.WriteString("# Tan derleyicisi tarafından üretildi — doğrudan x86-64 assembly.\n")
	son.WriteString("# C yok, gcc yok, libc yok. Ham syscall.\n")
	son.WriteString(".intel_syntax noprefix\n\n")

	son.WriteString(".section .rodata\n")
	for i, m := range a.metinler {
		son.WriteString(fmt.Sprintf(".Lmetin%d: .ascii \"%s\\n\"\n", i, asmKacis(m)))
	}
	son.WriteString("\n.section .bss\n")
	for ad := range a.genel {
		son.WriteString(fmt.Sprintf("%s: .space 8\n", asmAd(ad)))
	}

	son.WriteString("\n.section .text\n.global _start\n\n")
	son.WriteString(asmYardimcilar)
	son.WriteString(govde.String())

	tmp, err := os.MkdirTemp("", "tanasm")
	if err != nil {
		fmt.Printf("geçici dizin: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	sDosya := tmp + "/uretilen.s"
	if asmSakla {
		sDosya = cikti + ".s"
	}
	if err := os.WriteFile(sDosya, []byte(son.String()), 0644); err != nil {
		fmt.Printf("asm yazılamadı: %v\n", err)
		os.Exit(1)
	}

	oDosya := tmp + "/uretilen.o"
	komut := exec.Command("as", "--64", "-o", oDosya, sDosya)
	komut.Stdout = os.Stdout
	komut.Stderr = os.Stderr
	if err := komut.Run(); err != nil {
		fmt.Printf("as hatası: %v\n", err)
		os.Exit(1)
	}

	komut = exec.Command("ld", "-o", cikti, oDosya)
	komut.Stdout = os.Stdout
	komut.Stderr = os.Stderr
	if err := komut.Run(); err != nil {
		fmt.Printf("ld hatası: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Assembly'den native binary üretildi: %s\n", cikti)
	if asmSakla {
		fmt.Printf("Assembly kaynağı: %s\n", sDosya)
	}
}

func asmKacis(s string) string {
	var b strings.Builder
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
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Elle yazılmış x86-64 çalışma-zamanı yardımcıları (libc yerine ham syscall)
const asmYardimcilar = `# --- yaz_metin(rsi=adres, rdx=uzunluk) ---
yaz_metin:
    mov rax, 1
    mov rdi, 1
    syscall
    ret

# --- yaz_sayi(rdi=sayi) : int64 -> ondalık metin -> stdout ---
yaz_sayi:
    push rbp
    mov rbp, rsp
    sub rsp, 32
    mov rax, rdi
    lea rcx, [rbp-1]
    mov byte ptr [rcx], 10
    xor r8, r8
    cmp rax, 0
    jge .Lpozitif
    mov r8, 1
    neg rax
.Lpozitif:
    cmp rax, 0
    jne .Ljegit
    dec rcx
    mov byte ptr [rcx], 48
    jmp .Lisaret
.Ljegit:
    mov r9, 10
.Ldongu:
    cmp rax, 0
    je .Lisaret
    xor rdx, rdx
    div r9
    add dl, 48
    dec rcx
    mov [rcx], dl
    jmp .Ldongu
.Lisaret:
    cmp r8, 0
    je .Lyazdir
    dec rcx
    mov byte ptr [rcx], 45
.Lyazdir:
    mov rsi, rcx
    lea rdx, [rbp]
    sub rdx, rcx
    mov rax, 1
    mov rdi, 1
    syscall
    leave
    ret

`
