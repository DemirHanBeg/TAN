//go:build !js

package main

// ============================================================
// KADEME 3 + 4:  TAN'IN KENDİ ASSEMBLER'I VE LINKER'I
// ------------------------------------------------------------
// as YOK. ld YOK. gcc YOK. libc YOK. Hiçbir dış araç yok.
// AST -> x86-64 MAKİNE KODU BAYTLARI -> ELF64 dosyası (elle yazılır)
//
// Kademe 3 (assembler): ModRM/REX kodlaması, etiket çözümleme,
//                       rel32 atlama düzeltmeleri.
// Kademe 4 (linker)   : ELF başlığı, program header, segment yerleşimi,
//                       sanal adres hesabı, giriş noktası.
// ============================================================

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	elfTaban  = 0x400000 // segmentin sanal taban adresi
	basliklar = 64 + 56  // ELF başlığı + 1 program header
)

// ---------- kayıt numaraları ----------
const (
	rRAX = 0
	rRCX = 1
	rRDX = 2
	rRBX = 3
	rRSP = 4
	rRBP = 5
	rRSI = 6
	rRDI = 7
	rR8  = 8
	rR9  = 9
)

type duzeltme struct {
	konum  int    // rel32'nin kod içindeki yeri
	etiket string // hedef etiket
}

type veriBasvuru struct {
	konum int    // disp32'nin kod içindeki yeri (RIP-göreli)
	ad    string // veri etiketi
}

// makineKodu: Tan'ın kendi assembler'ı
type makineKodu struct {
	kod         []byte
	etiketler   map[string]int
	duzeltmeler []duzeltme
	veriler     []veriBasvuru
}

func yeniMakineKodu() *makineKodu {
	return &makineKodu{etiketler: map[string]int{}}
}

func (m *makineKodu) bayt(b ...byte) { m.kod = append(m.kod, b...) }

func (m *makineKodu) i32(v int32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	m.kod = append(m.kod, b[:]...)
}

func (m *makineKodu) i64(v int64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	m.kod = append(m.kod, b[:]...)
}

func (m *makineKodu) etiketKoy(ad string) { m.etiketler[ad] = len(m.kod) }

// rel32 gerektiren komutlar: yer tutucu koy, sonra düzelt
func (m *makineKodu) rel32(etiket string) {
	m.duzeltmeler = append(m.duzeltmeler, duzeltme{konum: len(m.kod), etiket: etiket})
	m.i32(0)
}

func (m *makineKodu) veriRef(ad string) {
	m.veriler = append(m.veriler, veriBasvuru{konum: len(m.kod), ad: ad})
	m.i32(0)
}

// ---------- komut kodlayıcılar (ModRM / REX elle) ----------

// ModRM baytı: mod(2) reg(3) rm(3)
func modrm(mod, reg, rm byte) byte { return (mod << 6) | ((reg & 7) << 3) | (rm & 7) }

func (m *makineKodu) pushKayit(r byte) {
	if r >= 8 {
		m.bayt(0x41)
	}
	m.bayt(0x50 + (r & 7))
}

func (m *makineKodu) popKayit(r byte) {
	if r >= 8 {
		m.bayt(0x41)
	}
	m.bayt(0x58 + (r & 7))
}

// mov r64, imm64  (movabs)
func (m *makineKodu) movImm64(r byte, v int64) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xB8+(r&7))
	m.i64(v)
}

// mov r64, imm32 (işaretli genişletme)
func (m *makineKodu) movImm32(r byte, v int32) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xC7, modrm(3, 0, r))
	m.i32(v)
}

// iki kayıtlı 64-bit işlem: op rm, reg   (89=mov, 01=add, 29=sub, 39=cmp, 85=test, 21=and, 09=or, 31=xor)
func (m *makineKodu) ikiliKayit(opkod byte, rm, reg byte) {
	rex := byte(0x48)
	if reg >= 8 {
		rex |= 4 // REX.R
	}
	if rm >= 8 {
		rex |= 1 // REX.B
	}
	m.bayt(rex, opkod, modrm(3, reg, rm))
}

func (m *makineKodu) movKayit(hedef, kaynak byte) { m.ikiliKayit(0x89, hedef, kaynak) }
func (m *makineKodu) addKayit(hedef, kaynak byte) { m.ikiliKayit(0x01, hedef, kaynak) }
func (m *makineKodu) subKayit(hedef, kaynak byte) { m.ikiliKayit(0x29, hedef, kaynak) }
func (m *makineKodu) cmpKayit(sol, sag byte)      { m.ikiliKayit(0x39, sol, sag) }
func (m *makineKodu) testKayit(a, b byte)         { m.ikiliKayit(0x85, a, b) }
func (m *makineKodu) andKayit(hedef, kaynak byte) { m.ikiliKayit(0x21, hedef, kaynak) }
func (m *makineKodu) orKayit(hedef, kaynak byte)  { m.ikiliKayit(0x09, hedef, kaynak) }
func (m *makineKodu) xorKayit(hedef, kaynak byte) { m.ikiliKayit(0x31, hedef, kaynak) }

// imul r64, r64  (REX.W 0F AF /r)
func (m *makineKodu) imulKayit(hedef, kaynak byte) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if kaynak >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x0F, 0xAF, modrm(3, hedef, kaynak))
}

func (m *makineKodu) cqo() { m.bayt(0x48, 0x99) }

// idiv r64 (F7 /7)
func (m *makineKodu) idivKayit(r byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xF7, modrm(3, 7, r))
}

// div r64 (F7 /6) — işaretsiz
func (m *makineKodu) divKayit(r byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xF7, modrm(3, 6, r))
}

// neg r64 (F7 /3)
func (m *makineKodu) negKayit(r byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xF7, modrm(3, 3, r))
}

// dec r64 (FF /1)
func (m *makineKodu) decKayit(r byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xFF, modrm(3, 1, r))
}

// setcc al/cl  (0F 9x)
func (m *makineKodu) setcc(kod byte, r byte) {
	m.bayt(0x0F, kod, modrm(3, 0, r))
}

// movzx r64, r8  (REX.W 0F B6 /r)
func (m *makineKodu) movzx(hedef, kaynak byte) {
	m.bayt(0x48, 0x0F, 0xB6, modrm(3, hedef, kaynak))
}

// cmp r64, imm32 (81 /7)
func (m *makineKodu) cmpImm32(r byte, v int32) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x81, modrm(3, 7, r))
	m.i32(v)
}

// sub rsp, imm32 (81 /5)
func (m *makineKodu) subImm32(r byte, v int32) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x81, modrm(3, 5, r))
	m.i32(v)
}

// mov [rbp+disp32], r64
func (m *makineKodu) movYerelYaz(disp int32, kaynak byte) {
	rex := byte(0x48)
	if kaynak >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x89, modrm(2, kaynak, rRBP))
	m.i32(disp)
}

// mov r64, [rbp+disp32]
func (m *makineKodu) movYerelOku(hedef byte, disp int32) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8B, modrm(2, hedef, rRBP))
	m.i32(disp)
}

// mov [rbp+disp32], imm32 (C7 /0)
func (m *makineKodu) movYerelImm(disp int32, v int32) {
	m.bayt(0x48, 0xC7, modrm(2, 0, rRBP))
	m.i32(disp)
	m.i32(v)
}

// mov r64, [rip+disp32]  -> genel değişken oku
func (m *makineKodu) movGenelOku(hedef byte, ad string) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8B, modrm(0, hedef, 5)) // rm=101 => RIP-göreli
	m.veriRef(ad)
}

// mov [rip+disp32], r64  -> genel değişken yaz
func (m *makineKodu) movGenelYaz(ad string, kaynak byte) {
	rex := byte(0x48)
	if kaynak >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x89, modrm(0, kaynak, 5))
	m.veriRef(ad)
}

// lea r64, [rip+disp32]
func (m *makineKodu) leaVeri(hedef byte, ad string) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8D, modrm(0, hedef, 5))
	m.veriRef(ad)
}

// lea r64, [rbp+disp8]
func (m *makineKodu) leaRbp(hedef byte, disp int8) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8D, modrm(1, hedef, rRBP), byte(disp))
}

func (m *makineKodu) jmp(etiket string)  { m.bayt(0xE9); m.rel32(etiket) }
func (m *makineKodu) call(etiket string) { m.bayt(0xE8); m.rel32(etiket) }
func (m *makineKodu) jcc(kod byte, etiket string) {
	m.bayt(0x0F, kod)
	m.rel32(etiket)
}

func (m *makineKodu) ret()     { m.bayt(0xC3) }
func (m *makineKodu) leave()   { m.bayt(0xC9) }
func (m *makineKodu) syscall() { m.bayt(0x0F, 0x05) }

// mov byte ptr [rcx], imm8  (C6 /0)
func (m *makineKodu) movBaytImm(r byte, v byte) {
	m.bayt(0xC6, modrm(0, 0, r), v)
}

// mov [rcx], dl  (88 /r)
func (m *makineKodu) movBaytKayit(rmR byte, regR byte) {
	m.bayt(0x88, modrm(0, regR, rmR))
}

// add dl, imm8 (80 /0)
func (m *makineKodu) addBaytImm(r byte, v byte) {
	m.bayt(0x80, modrm(3, 0, r), v)
}

// ============================================================
// KOD ÜRETECİ
// ============================================================

type elfUretici struct {
	m         *makineKodu
	yereller  map[string]int32
	genel     map[string]bool
	metinler  []string
	etiketNo  int
	donguBas  []string
	donguSon  []string
}

func (e *elfUretici) yeniEtiket(on string) string {
	e.etiketNo++
	return fmt.Sprintf("L%s%d", on, e.etiketNo)
}

func (e *elfUretici) ifade(d Dugum) {
	m := e.m
	switch n := d.(type) {
	case SayiDugum:
		if n.TamMi {
			m.movImm64(rRAX, n.Tam) // kesin int64 (hassasiyet kaybı yok)
		} else {
			// Sessizce kesmek yerine açıkça reddet: yanlış sonuç vermektense hata ver.
			panic(TanHata{Mesaj: fmt.Sprintf(
				"elf backend ondalık sayı desteklemiyor (%g). Tam sayı kullan veya 'tan derle' (C yolu) ile derle.", n.Deger)})
		}

	case MantikDugum:
		if n.Deger {
			m.movImm32(rRAX, 1)
		} else {
			m.movImm32(rRAX, 0)
		}

	case YokDugum:
		m.movImm32(rRAX, 0)

	case DegiskenDugum:
		if off, ok := e.yereller[n.Ad]; ok {
			m.movYerelOku(rRAX, off)
		} else {
			e.genel[n.Ad] = true
			m.movGenelOku(rRAX, "v_"+elfAd(n.Ad))
		}

	case IkiliDugum:
		e.ifade(n.Sol)
		m.pushKayit(rRAX)
		e.ifade(n.Sag)
		m.movKayit(rRCX, rRAX) // rcx = sağ
		m.popKayit(rRAX)       // rax = sol
		switch n.Islec {
		case "+":
			m.addKayit(rRAX, rRCX)
		case "-":
			m.subKayit(rRAX, rRCX)
		case "*":
			m.imulKayit(rRAX, rRCX)
		case "/":
			m.cqo()
			m.idivKayit(rRCX)
		case "%":
			m.cqo()
			m.idivKayit(rRCX)
			m.movKayit(rRAX, rRDX)
		case "==", "!=", ">", "<", ">=", "<=":
			kod := map[string]byte{"==": 0x94, "!=": 0x95, ">": 0x9F, "<": 0x9C, ">=": 0x9D, "<=": 0x9E}[n.Islec]
			m.cmpKayit(rRAX, rRCX)
			m.setcc(kod, rRAX) // sete al
			m.movzx(rRAX, rRAX)
		case "ve", "veya":
			m.testKayit(rRAX, rRAX)
			m.setcc(0x95, rRAX)
			m.movzx(rRAX, rRAX)
			m.testKayit(rRCX, rRCX)
			m.setcc(0x95, rRCX)
			m.movzx(rRCX, rRCX)
			if n.Islec == "ve" {
				m.andKayit(rRAX, rRCX)
			} else {
				m.orKayit(rRAX, rRCX)
			}
		default:
			panic(TanHata{Mesaj: "elf: bilinmeyen işleç '" + n.Islec + "'"})
		}

	case CagriDugum:
		if n.Ad == "değil" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: değil() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.testKayit(rRAX, rRAX)
			m.setcc(0x94, rRAX)
			m.movzx(rRAX, rRAX)
			return
		}
		if len(n.Argumanlar) > 6 {
			panic(TanHata{Mesaj: "elf: en fazla 6 argüman"})
		}
		for _, a := range n.Argumanlar {
			e.ifade(a)
			m.pushKayit(rRAX)
		}
		kayitlar := []byte{rRDI, rRSI, rRDX, rRCX, rR8, rR9}
		for i := len(n.Argumanlar) - 1; i >= 0; i-- {
			m.popKayit(kayitlar[i])
		}
		m.call("f_" + elfAd(n.Ad))

	default:
		panic(TanHata{Mesaj: fmt.Sprintf("elf: bu ifade desteklenmiyor (%T)", d)})
	}
}

func (e *elfUretici) deyim(d Dugum) {
	m := e.m
	switch n := d.(type) {
	case AtamaDugum:
		e.ifade(n.Deger)
		if off, ok := e.yereller[n.Ad]; ok {
			m.movYerelYaz(off, rRAX)
		} else {
			e.genel[n.Ad] = true
			m.movGenelYaz("v_"+elfAd(n.Ad), rRAX)
		}

	case YazDugum:
		if mt, ok := n.Deger.(MetinDugum); ok {
			ad := fmt.Sprintf("s%d", len(e.metinler))
			e.metinler = append(e.metinler, mt.Deger+"\n")
			m.leaVeri(rRSI, ad)
			m.movImm32(rRDX, int32(len(mt.Deger)+1))
			m.call("f_yaz_metin")
		} else {
			e.ifade(n.Deger)
			m.movKayit(rRDI, rRAX)
			m.call("f_yaz_sayi")
		}

	case EgerDugum:
		degilse := e.yeniEtiket("degilse")
		son := e.yeniEtiket("egerson")
		e.ifade(n.Kosul)
		m.testKayit(rRAX, rRAX)
		m.jcc(0x84, degilse) // jz
		for _, s := range n.Govde {
			e.deyim(s)
		}
		m.jmp(son)
		m.etiketKoy(degilse)
		for _, s := range n.Degilse {
			e.deyim(s)
		}
		m.etiketKoy(son)

	case IkenDugum:
		bas := e.yeniEtiket("dongubas")
		son := e.yeniEtiket("donguson")
		e.donguBas = append(e.donguBas, bas)
		e.donguSon = append(e.donguSon, son)
		m.etiketKoy(bas)
		e.ifade(n.Kosul)
		m.testKayit(rRAX, rRAX)
		m.jcc(0x84, son)
		for _, s := range n.Govde {
			e.deyim(s)
		}
		m.jmp(bas)
		m.etiketKoy(son)
		e.donguBas = e.donguBas[:len(e.donguBas)-1]
		e.donguSon = e.donguSon[:len(e.donguSon)-1]

	case DurDugum:
		if len(e.donguSon) == 0 {
			panic(TanHata{Mesaj: "elf: dur döngü dışında"})
		}
		m.jmp(e.donguSon[len(e.donguSon)-1])

	case DevamDugum:
		if len(e.donguBas) == 0 {
			panic(TanHata{Mesaj: "elf: devam döngü dışında"})
		}
		m.jmp(e.donguBas[len(e.donguBas)-1])

	case DondurDugum:
		if n.Deger != nil {
			e.ifade(n.Deger)
		} else {
			m.movImm32(rRAX, 0)
		}
		m.leave()
		m.ret()

	case CagriDugum:
		e.ifade(n)

	default:
		panic(TanHata{Mesaj: fmt.Sprintf("elf: bu deyim desteklenmiyor (%T)", d)})
	}
}

func elfAd(ad string) string {
	var b []rune
	for _, r := range ad {
		switch r {
		case 'ç':
			b = append(b, 'c')
		case 'ş':
			b = append(b, 's')
		case 'ğ':
			b = append(b, 'g')
		case 'ü':
			b = append(b, 'u')
		case 'ö':
			b = append(b, 'o')
		case 'ı':
			b = append(b, 'i')
		case 'İ':
			b = append(b, 'I')
		default:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b = append(b, r)
			} else {
				b = append(b, '_')
			}
		}
	}
	return string(b)
}

func (e *elfUretici) islevYaz(n IslevDugum) {
	m := e.m
	m.etiketKoy("f_" + elfAd(n.Ad))
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)

	sira := append([]string{}, n.Parametreler...)
	atananlar := map[string]bool{}
	asmDegiskenleriTopla(n.Govde, atananlar)
	for ad := range atananlar {
		varmi := false
		for _, p := range n.Parametreler {
			if p == ad {
				varmi = true
			}
		}
		if !varmi {
			sira = append(sira, ad)
		}
	}
	yereller := map[string]int32{}
	for i, ad := range sira {
		yereller[ad] = int32(-8 * (i + 1))
	}
	cerceve := int32(8 * len(sira))
	if cerceve%16 != 0 {
		cerceve += 8
	}
	if cerceve > 0 {
		m.subImm32(rRSP, cerceve)
	}

	kayitlar := []byte{rRDI, rRSI, rRDX, rRCX, rR8, rR9}
	for i, p := range n.Parametreler {
		m.movYerelYaz(yereller[p], kayitlar[i])
	}
	for _, ad := range sira[len(n.Parametreler):] {
		m.movYerelImm(yereller[ad], 0)
	}

	eski := e.yereller
	e.yereller = yereller
	for _, s := range n.Govde {
		e.deyim(s)
	}
	e.yereller = eski

	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}

// yaz_metin(rsi=adres, rdx=uzunluk) — ham write syscall
func (e *elfUretici) yardimciYazMetin() {
	m := e.m
	m.etiketKoy("f_yaz_metin")
	m.movImm32(rRAX, 1) // sys_write
	m.movImm32(rRDI, 1) // stdout
	m.syscall()
	m.ret()
}

// yaz_sayi(rdi=sayı) — int64'ü ondalığa çevirip yaz
func (e *elfUretici) yardimciYazSayi() {
	m := e.m
	m.etiketKoy("f_yaz_sayi")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movKayit(rRAX, rRDI)
	m.leaRbp(rRCX, -1)
	m.movBaytImm(rRCX, 10) // '\n'
	m.xorKayit(rR8, rR8)   // işaret bayrağı
	m.cmpImm32(rRAX, 0)
	m.jcc(0x8D, "Lpoz") // jge
	m.movImm32(rR8, 1)
	m.negKayit(rRAX)
	m.etiketKoy("Lpoz")
	m.cmpImm32(rRAX, 0)
	m.jcc(0x85, "Ljegit") // jne
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '0')
	m.jmp("Lisaret")
	m.etiketKoy("Ljegit")
	m.movImm32(rR9, 10)
	m.etiketKoy("Ldongu")
	m.cmpImm32(rRAX, 0)
	m.jcc(0x84, "Lisaret") // je
	m.xorKayit(rRDX, rRDX)
	m.divKayit(rR9)
	m.addBaytImm(rRDX, '0') // add dl, '0'
	m.decKayit(rRCX)
	m.movBaytKayit(rRCX, rRDX) // mov [rcx], dl
	m.jmp("Ldongu")
	m.etiketKoy("Lisaret")
	m.cmpImm32(rR8, 0)
	m.jcc(0x84, "Lyazdir") // je
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '-')
	m.etiketKoy("Lyazdir")
	m.movKayit(rRSI, rRCX)
	m.leaRbp(rRDX, 0)
	m.subKayit(rRDX, rRCX)
	m.movImm32(rRAX, 1)
	m.movImm32(rRDI, 1)
	m.syscall()
	m.leave()
	m.ret()
}

// ============================================================
// derleElf: uçtan uca — makine kodu + ELF, sıfır dış araç
// ============================================================
func derleElf(dosya string, cikti string) {
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

	var islevler []IslevDugum
	var anaGovde []Dugum
	for _, d := range agac {
		if isv, ok := d.(IslevDugum); ok {
			islevler = append(islevler, isv)
		} else {
			anaGovde = append(anaGovde, d)
		}
	}

	e := &elfUretici{m: yeniMakineKodu(), genel: map[string]bool{}}

	// yardımcılar + kullanıcı işlevleri + _start
	e.yardimciYazMetin()
	e.yardimciYazSayi()
	for _, isv := range islevler {
		e.islevYaz(isv)
	}
	e.m.etiketKoy("_start")
	for _, s := range anaGovde {
		e.deyim(s)
	}
	e.m.movImm32(rRAX, 60) // sys_exit
	e.m.xorKayit(rRDI, rRDI)
	e.m.syscall()

	m := e.m

	// ---------- KADEME 4: yerleşim ve bağlama (linker) ----------
	kodBoy := len(m.kod)
	kodOfs := basliklar
	veriOfs := kodOfs + kodBoy
	// 8'e hizala
	if veriOfs%8 != 0 {
		pad := 8 - (veriOfs % 8)
		veriOfs += pad
		kodBoy += pad
		m.kod = append(m.kod, make([]byte, pad)...)
	}

	// veri yerleşimi: önce metin sabitleri, sonra genel değişkenler
	veriAdres := map[string]int{}
	var veri []byte
	for i, s := range e.metinler {
		veriAdres[fmt.Sprintf("s%d", i)] = veriOfs + len(veri)
		veri = append(veri, []byte(s)...)
	}
	// 8 hizası
	for len(veri)%8 != 0 {
		veri = append(veri, 0)
	}
	genelSirali := []string{}
	for ad := range e.genel {
		genelSirali = append(genelSirali, ad)
	}
	// belirlenimci sıra
	for i := 0; i < len(genelSirali); i++ {
		for j := i + 1; j < len(genelSirali); j++ {
			if genelSirali[j] < genelSirali[i] {
				genelSirali[i], genelSirali[j] = genelSirali[j], genelSirali[i]
			}
		}
	}
	for _, ad := range genelSirali {
		veriAdres["v_"+elfAd(ad)] = veriOfs + len(veri)
		veri = append(veri, make([]byte, 8)...)
	}

	// rel32 düzeltmeleri (etiketler arası atlama/çağrı)
	for _, d := range m.duzeltmeler {
		hedef, ok := m.etiketler[d.etiket]
		if !ok {
			fmt.Fprintf(os.Stderr, "bağlama hatası: '%s' etiketi bulunamadı\n", d.etiket)
			os.Exit(1)
		}
		rel := int32(hedef - (d.konum + 4))
		binary.LittleEndian.PutUint32(m.kod[d.konum:], uint32(rel))
	}

	// RIP-göreli veri başvuruları
	for _, v := range m.veriler {
		adr, ok := veriAdres[v.ad]
		if !ok {
			fmt.Fprintf(os.Stderr, "bağlama hatası: '%s' verisi bulunamadı\n", v.ad)
			os.Exit(1)
		}
		// komutun bitişi = kodOfs + konum + 4  (dosya ofseti = sanal ofset)
		rel := int32(adr - (kodOfs + v.konum + 4))
		binary.LittleEndian.PutUint32(m.kod[v.konum:], uint32(rel))
	}

	toplam := kodOfs + len(m.kod) + len(veri)
	giris := elfTaban + kodOfs + m.etiketler["_start"]

	// ---------- ELF64 başlığı (elle) ----------
	var dosyaBaytlari []byte
	yaz8 := func(v byte) { dosyaBaytlari = append(dosyaBaytlari, v) }
	yaz16 := func(v uint16) {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], v)
		dosyaBaytlari = append(dosyaBaytlari, b[:]...)
	}
	yaz32 := func(v uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], v)
		dosyaBaytlari = append(dosyaBaytlari, b[:]...)
	}
	yaz64 := func(v uint64) {
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], v)
		dosyaBaytlari = append(dosyaBaytlari, b[:]...)
	}

	// e_ident
	yaz8(0x7F)
	yaz8('E')
	yaz8('L')
	yaz8('F')
	yaz8(2) // 64-bit
	yaz8(1) // little endian
	yaz8(1) // ELF sürümü
	yaz8(0) // System V ABI
	for i := 0; i < 8; i++ {
		yaz8(0)
	}
	yaz16(2)      // e_type = ET_EXEC
	yaz16(0x3E)   // e_machine = x86-64
	yaz32(1)      // e_version
	yaz64(uint64(giris)) // e_entry
	yaz64(64)     // e_phoff
	yaz64(0)      // e_shoff
	yaz32(0)      // e_flags
	yaz16(64)     // e_ehsize
	yaz16(56)     // e_phentsize
	yaz16(1)      // e_phnum
	yaz16(0)      // e_shentsize
	yaz16(0)      // e_shnum
	yaz16(0)      // e_shstrndx

	// program header (PT_LOAD, RWX)
	yaz32(1) // p_type = PT_LOAD
	yaz32(7) // p_flags = R+W+X
	yaz64(0) // p_offset
	yaz64(uint64(elfTaban))
	yaz64(uint64(elfTaban))
	yaz64(uint64(toplam)) // p_filesz
	yaz64(uint64(toplam)) // p_memsz
	yaz64(0x1000)         // p_align

	dosyaBaytlari = append(dosyaBaytlari, m.kod...)
	dosyaBaytlari = append(dosyaBaytlari, veri...)

	if err := os.WriteFile(cikti, dosyaBaytlari, 0755); err != nil {
		fmt.Printf("ELF yazılamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ELF doğrudan yazıldı: %s  (%d bayt, kod %d bayt, veri %d bayt)\n",
		cikti, len(dosyaBaytlari), kodBoy, len(veri))
	fmt.Println("Kullanılan dış araç: YOK (as/ld/gcc/libc hiçbiri)")
}
