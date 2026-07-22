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
	"math"
	"os"
	"strings"
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
	rR10 = 10
	rR11 = 11
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


// add r64, imm32 (81 /0)
func (m *makineKodu) addImm32(r byte, v int32) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x81, modrm(3, 0, r))
	m.i32(v)
}

// and r64, imm32 (81 /4)
func (m *makineKodu) andImm32(r byte, v int32) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x81, modrm(3, 4, r))
	m.i32(v)
}

// mov r64, [taban+disp8]
func (m *makineKodu) movDolayliOku(hedef, taban byte, disp int8) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	if disp == 0 && (taban&7) != rRBP {
		m.bayt(rex, 0x8B, modrm(0, hedef, taban))
		return
	}
	m.bayt(rex, 0x8B, modrm(1, hedef, taban), byte(disp))
}

// mov [taban+disp8], r64
func (m *makineKodu) movDolayliYaz(taban byte, disp int8, kaynak byte) {
	rex := byte(0x48)
	if kaynak >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	if disp == 0 && (taban&7) != rRBP {
		m.bayt(rex, 0x89, modrm(0, kaynak, taban))
		return
	}
	m.bayt(rex, 0x89, modrm(1, kaynak, taban), byte(disp))
}

// lea r64, [taban+disp8]
func (m *makineKodu) leaDolayli(hedef, taban byte, disp int8) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x8D, modrm(1, hedef, taban), byte(disp))
}

// inc r64 (FF /0)
func (m *makineKodu) incKayit(r byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xFF, modrm(3, 0, r))
}

// mov al, [taban]  (8A /r)
func (m *makineKodu) movBaytOku(taban byte) {
	m.bayt(0x8A, modrm(0, 0, taban))
}

// mov [taban], al  (88 /r)
func (m *makineKodu) movBaytYazAl(taban byte) {
	m.bayt(0x88, modrm(0, 0, taban))
}

// mov [rbp+disp32], imm32 zaten var: movYerelImm


// mov [taban+disp32], r64
func (m *makineKodu) movDolayliYaz32(taban byte, disp int32, kaynak byte) {
	rex := byte(0x48)
	if kaynak >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x89, modrm(2, kaynak, taban))
	m.i32(disp)
}

// lea r64, [taban + indeks*8 + 8]   (SIB kodlamasi)
func (m *makineKodu) leaOge(hedef, taban, indeks byte) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if indeks >= 8 {
		rex |= 2
	}
	if taban >= 8 {
		rex |= 1
	}
	// mod=01 (disp8), rm=100 => SIB
	m.bayt(rex, 0x8D, modrm(1, hedef, 4))
	// SIB: scale=11(x8), index, base
	m.bayt((3<<6)|((indeks&7)<<3)|(taban&7), 8)
}

// push imm32 (68 id)
func (m *makineKodu) pushImm32(v int32) { m.bayt(0x68); m.i32(v) }

// mov r64, [rsp+disp8]
func (m *makineKodu) movRspOku(hedef byte, disp int8) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8B, modrm(1, hedef, 4), 0x24, byte(disp))
}

// inc qword [rsp]
func (m *makineKodu) incRspSifir() { m.bayt(0x48, 0xFF, modrm(0, 0, 4), 0x24) }

// shl r64, imm8 (C1 /4)
func (m *makineKodu) shlImm(r byte, v byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xC1, modrm(3, 4, r), v)
}

// movzx r64, byte [taban+indeks]  -> tek bayt oku
func (m *makineKodu) movzxBaytDolayli(hedef, taban, indeks byte) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if indeks >= 8 {
		rex |= 2
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x0F, 0xB6, modrm(0, hedef, 4))
	m.bayt((0<<6)|((indeks&7)<<3)|(taban&7))
}


// ---------- SSE (ondalik sayi) komut kodlayicilari ----------

// movq xmm<h>, r64<k>   (66 REX.W 0F 6E /r)
func (m *makineKodu) movqXmmKayit(x, r byte) {
	rex := byte(0x48)
	if x >= 8 {
		rex |= 4
	}
	if r >= 8 {
		rex |= 1
	}
	m.bayt(0x66, rex, 0x0F, 0x6E, modrm(3, x, r))
}

// movq r64<r>, xmm<x>   (66 REX.W 0F 7E /r)
func (m *makineKodu) movqKayitXmm(r, x byte) {
	rex := byte(0x48)
	if x >= 8 {
		rex |= 4
	}
	if r >= 8 {
		rex |= 1
	}
	m.bayt(0x66, rex, 0x0F, 0x7E, modrm(3, x, r))
}

// F2 0F <op> /r : addsd(58) subsd(5C) mulsd(59) divsd(5E)
func (m *makineKodu) sseIkili(op byte, hedef, kaynak byte) {
	m.bayt(0xF2)
	if hedef >= 8 || kaynak >= 8 {
		rex := byte(0x40)
		if hedef >= 8 {
			rex |= 4
		}
		if kaynak >= 8 {
			rex |= 1
		}
		m.bayt(rex)
	}
	m.bayt(0x0F, op, modrm(3, hedef, kaynak))
}

// cvtsi2sd xmm, r64  (F2 REX.W 0F 2A /r)
func (m *makineKodu) cvtTamKesir(x, r byte) {
	rex := byte(0x48)
	if x >= 8 {
		rex |= 4
	}
	if r >= 8 {
		rex |= 1
	}
	m.bayt(0xF2, rex, 0x0F, 0x2A, modrm(3, x, r))
}

// cvttsd2si r64, xmm  (F2 REX.W 0F 2C /r)
func (m *makineKodu) cvtKesirTam(r, x byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 4
	}
	if x >= 8 {
		rex |= 1
	}
	m.bayt(0xF2, rex, 0x0F, 0x2C, modrm(3, r, x))
}

// comisd xmm, xmm  (66 0F 2F /r)
func (m *makineKodu) comisd(a, b byte) {
	if a >= 8 || b >= 8 {
		rex := byte(0x40)
		if a >= 8 {
			rex |= 4
		}
		if b >= 8 {
			rex |= 1
		}
		m.bayt(0x66, rex, 0x0F, 0x2F, modrm(3, a, b))
		return
	}
	m.bayt(0x66, 0x0F, 0x2F, modrm(3, a, b))
}

// cmp qword [rbp+disp32], imm32 (81 /7)
func (m *makineKodu) cmpYerelImm(disp int32, v int32) {
	m.bayt(0x48, 0x81, modrm(2, 7, rRBP))
	m.i32(disp)
	m.i32(v)
}

// ============================================================
// KOD ÜRETECİ
// ============================================================

// Tip: ELF backend'inin statik tip sistemi.
// Yorumlayici dinamik tiplidir; native backend statik bir altkume derler.
type Cesit int

const (
	CTam   Cesit = iota // int64, kayitta ham deger
	CMetin              // yigin isaretcisi: [uzunluk:8][baytlar...]
	CListe              // yigin isaretcisi: [uzunluk:8][oge0:8][oge1:8]...
	CKesir              // float64, kayitta HAM BIT DESENI olarak tasinir
)

type Tip struct {
	Cesit  Cesit
	Eleman *Tip // liste ise oge tipi
}

var TipTam = Tip{Cesit: CTam}
var TipMetin = Tip{Cesit: CMetin}
var TipKesir = Tip{Cesit: CKesir}

func TipListe(e Tip) Tip { return Tip{Cesit: CListe, Eleman: &e} }

func (t Tip) String() string {
	switch t.Cesit {
	case CTam:
		return "tam sayı"
	case CMetin:
		return "metin"
	case CKesir:
		return "ondalık"
	case CListe:
		if t.Eleman != nil {
			return "liste<" + t.Eleman.String() + ">"
		}
		return "liste"
	}
	return "bilinmeyen"
}

func (t Tip) esitMi(o Tip) bool { return t.Cesit == o.Cesit }

type elfUretici struct {
	m         *makineKodu
	yereller  map[string]int32
	genel     map[string]bool
	metinler  []string
	etiketNo  int
	donguBas  []string
	donguSon  []string
	tipler    map[string]Tip // degisken -> tip
	islevTipi map[string]Tip // islev -> donus tipi
	parametreTipi map[string]Tip // "islev/param" -> tip
	suanIslev string
	yiginVar  bool           // yigin ayirici koda eklendi mi
}

// tipCikar: bir ifadenin tipini derleme aninda belirler.
func (e *elfUretici) tipCikar(d Dugum) Tip {
	switch n := d.(type) {
	case SayiDugum:
		if n.TamMi {
			return TipTam
		}
		return TipKesir
	case MantikDugum, YokDugum:
		return TipTam
	case MetinDugum:
		return TipMetin
	case DegiskenDugum:
		if t, ok := e.tipler[n.Ad]; ok {
			return t
		}
		return TipTam
	case IkiliDugum:
		if n.Islec == "+" {
			if e.tipCikar(n.Sol).Cesit == CMetin || e.tipCikar(n.Sag).Cesit == CMetin {
				return TipMetin
			}
		}
		switch n.Islec {
		case "+", "-", "*", "/":
			if e.tipCikar(n.Sol).Cesit == CKesir || e.tipCikar(n.Sag).Cesit == CKesir {
				return TipKesir
			}
		}
		return TipTam
	case ListeDugum:
		if len(n.Elemanlar) == 0 {
			return TipListe(TipTam)
		}
		return TipListe(e.tipCikar(n.Elemanlar[0]))
	case IndeksDugum:
		ht := e.tipCikar(n.Hedef)
		if ht.Cesit == CMetin {
			return TipMetin // metin[i] -> tek harflik metin
		}
		if ht.Cesit == CListe && ht.Eleman != nil {
			return *ht.Eleman
		}
		return TipTam
	case CagriDugum:
		switch n.Ad {
		case "uzunluk", "kod":
			return TipTam
		case "metin", "karakter", "oku":
			return TipMetin
		case "ekle":
			if len(n.Argumanlar) == 2 {
				return TipListe(e.tipCikar(n.Argumanlar[1]))
			}
			return TipListe(TipTam)
		case "parçala":
			return TipListe(TipMetin)
		}
		if t, ok := e.islevTipi[n.Ad]; ok {
			return t
		}
		return TipTam
	}
	return TipTam
}

// govdeTipleriniTopla: govdedeki atamalari ve her-degiskenlerini
// sirayla isleyip e.tipler'e kaydeder (donus tipi cikarimi icin on-gecis).
func (e *elfUretici) govdeTipleriniTopla(govde []Dugum) {
	for _, d := range govde {
		switch n := d.(type) {
		case AtamaDugum:
			e.tipler[n.Ad] = e.tipCikar(n.Deger)
		case HerDugum:
			lt := e.tipCikar(n.Liste)
			if lt.Cesit == CListe && lt.Eleman != nil {
				e.tipler[n.Degisken] = *lt.Eleman
			} else if lt.Cesit == CMetin {
				e.tipler[n.Degisken] = TipMetin
			} else {
				e.tipler[n.Degisken] = TipTam
			}
			e.govdeTipleriniTopla(n.Govde)
		case EgerDugum:
			e.govdeTipleriniTopla(n.Govde)
			e.govdeTipleriniTopla(n.Degilse)
		case IkenDugum:
			e.govdeTipleriniTopla(n.Govde)
		}
	}
}

// dondurTipi: govdedeki dondur deyimlerinden donus tipini cikarir.
func (e *elfUretici) dondurTipi(govde []Dugum) (Tip, bool) {
	for _, d := range govde {
		switch n := d.(type) {
		case DondurDugum:
			if n.Deger != nil {
				return e.tipCikar(n.Deger), true
			}
		case EgerDugum:
			if t, ok := e.dondurTipi(n.Govde); ok {
				return t, true
			}
			if t, ok := e.dondurTipi(n.Degilse); ok {
				return t, true
			}
		case IkenDugum:
			if t, ok := e.dondurTipi(n.Govde); ok {
				return t, true
			}
		case HerDugum:
			if t, ok := e.dondurTipi(n.Govde); ok {
				return t, true
			}
		}
	}
	return TipTam, false
}

// parametreTipleriniOgren: cagri yerlerindeki arguman tiplerinden
// islev parametrelerinin tipini tahmin eder.
func (e *elfUretici) parametreTipleriniOgren(agac []Dugum, islevler []IslevDugum) {
	adlar := map[string]IslevDugum{}
	for _, i := range islevler {
		adlar[i.Ad] = i
	}
	var gez func(d Dugum)
	gez = func(d Dugum) {
		switch n := d.(type) {
		case CagriDugum:
			if isv, ok := adlar[n.Ad]; ok {
				for i, a := range n.Argumanlar {
					if i < len(isv.Parametreler) {
						at := e.tipCikar(a)
						if at.Cesit != CTam {
							e.parametreTipi[isv.Ad+"/"+isv.Parametreler[i]] = at
						}
					}
				}
			}
			for _, a := range n.Argumanlar {
				gez(a)
			}
		case AtamaDugum:
			gez(n.Deger)
		case YazDugum:
			gez(n.Deger)
		case IkiliDugum:
			gez(n.Sol)
			gez(n.Sag)
		case DondurDugum:
			if n.Deger != nil {
				gez(n.Deger)
			}
		case EgerDugum:
			gez(n.Kosul)
			for _, s := range n.Govde {
				gez(s)
			}
			for _, s := range n.Degilse {
				gez(s)
			}
		case IkenDugum:
			gez(n.Kosul)
			for _, s := range n.Govde {
				gez(s)
			}
		case HerDugum:
			gez(n.Liste)
			for _, s := range n.Govde {
				gez(s)
			}
		case ListeDugum:
			for _, x := range n.Elemanlar {
				gez(x)
			}
		case IndeksDugum:
			gez(n.Hedef)
			gez(n.Indeks)
		}
	}
	for _, d := range agac {
		gez(d)
	}
	for _, isv := range islevler {
		for _, s := range isv.Govde {
			gez(s)
		}
	}
}

// basitMi: yan etkisiz, tek komutla yuklenebilen ifade mi
func (e *elfUretici) basitMi(d Dugum) bool {
	switch n := d.(type) {
	case SayiDugum:
		return n.TamMi
	case MantikDugum, YokDugum:
		return true
	case DegiskenDugum:
		return e.tipCikar(n).Cesit == CTam
	}
	return false
}

// basitYukle: basit ifadeyi dogrudan hedef kayda yukler (yigin kullanmadan)
func (e *elfUretici) basitYukle(hedef byte, d Dugum) {
	m := e.m
	switch n := d.(type) {
	case SayiDugum:
		m.movImm64(hedef, n.Tam)
	case MantikDugum:
		if n.Deger {
			m.movImm32(hedef, 1)
		} else {
			m.movImm32(hedef, 0)
		}
	case YokDugum:
		m.movImm32(hedef, 0)
	case DegiskenDugum:
		if off, ok := e.yereller[n.Ad]; ok {
			m.movYerelOku(hedef, off)
		} else {
			e.genel[n.Ad] = true
			m.movGenelOku(hedef, "v_"+elfAd(n.Ad))
		}
	}
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
			m.movImm64(rRAX, int64(math.Float64bits(n.Deger))) // ondalik: ham bit deseni
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
		// metin birlestirme
		if n.Islec == "+" && (e.tipCikar(n.Sol).Cesit == CMetin || e.tipCikar(n.Sag).Cesit == CMetin) {
			// sayi tarafini otomatik metne cevir (yorumlayiciyla ayni davranis)
			e.ifade(n.Sol)
			if e.tipCikar(n.Sol).Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if e.tipCikar(n.Sol).Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.pushKayit(rRAX)
			e.ifade(n.Sag)
			if e.tipCikar(n.Sag).Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if e.tipCikar(n.Sag).Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_metin_birlestir")
			return
		}
		// --- ONDALIK YOL ---
		solT := e.tipCikar(n.Sol)
		sagT := e.tipCikar(n.Sag)
		if solT.Cesit == CKesir || sagT.Cesit == CKesir {
			e.ifade(n.Sol)
			if solT.Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.pushKayit(rRAX)
			e.ifade(n.Sag)
			if sagT.Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.movqXmmKayit(1, rRAX) // xmm1 = sag
			m.popKayit(rRAX)
			m.movqXmmKayit(0, rRAX) // xmm0 = sol
			switch n.Islec {
			case "+":
				m.sseIkili(0x58, 0, 1)
				m.movqKayitXmm(rRAX, 0)
			case "-":
				m.sseIkili(0x5C, 0, 1)
				m.movqKayitXmm(rRAX, 0)
			case "*":
				m.sseIkili(0x59, 0, 1)
				m.movqKayitXmm(rRAX, 0)
			case "/":
				m.sseIkili(0x5E, 0, 1)
				m.movqKayitXmm(rRAX, 0)
			case ">", "<", ">=", "<=", "==", "!=":
				// comisd isaretsiz bayrak kullanir
				kod := map[string]byte{">": 0x97, "<": 0x92, ">=": 0x93, "<=": 0x96, "==": 0x94, "!=": 0x95}[n.Islec]
				m.comisd(0, 1)
				m.setcc(kod, rRAX)
				m.movzx(rRAX, rRAX)
			default:
				panic(TanHata{Mesaj: "elf: ondalık sayıda '" + n.Islec + "' desteklenmiyor"})
			}
			return
		}
		// --- GOZETLEME OPTIMIZASYONU ---
		// Sag taraf "basit" ise (sabit veya degisken) push/pop cifti gereksiz:
		// dogrudan rcx'e yukle. Sicak donguleri belirgin hizlandirir.
		if e.basitMi(n.Sag) {
			e.ifade(n.Sol) // rax = sol
			e.basitYukle(rRCX, n.Sag)
		} else {
			e.ifade(n.Sol)
			m.pushKayit(rRAX)
			e.ifade(n.Sag)
			m.movKayit(rRCX, rRAX) // rcx = sağ
			m.popKayit(rRAX)       // rax = sol
		}
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

	case ListeDugum:
		n2 := len(n.Elemanlar)
		m.movImm32(rRDI, int32(8+n2*8))
		m.call("f_tan_ayir")
		m.pushKayit(rRAX)
		m.movImm32(rRCX, int32(n2))
		m.movDolayliYaz(rRAX, 0, rRCX)
		for i, oge := range n.Elemanlar {
			e.ifade(oge)
			m.popKayit(rRCX)  // liste
			m.pushKayit(rRCX)
			m.movDolayliYaz32(rRCX, int32(8+i*8), rRAX)
		}
		m.popKayit(rRAX)

	case IndeksDugum:
		ht := e.tipCikar(n.Hedef)
		e.ifade(n.Hedef)
		m.pushKayit(rRAX)
		e.ifade(n.Indeks)
		m.movKayit(rRSI, rRAX)
		m.popKayit(rRDI)
		if ht.Cesit == CMetin {
			m.call("f_metin_indeks")
		} else {
			m.leaOge(rRAX, rRDI, rRSI)
			m.movDolayliOku(rRAX, rRAX, 0)
		}

	case MetinDugum:
		ad := fmt.Sprintf("s%d", len(e.metinler))
		e.metinler = append(e.metinler, n.Deger)
		m.leaVeri(rRAX, ad)

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
		if n.Ad == "metin" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: metin() tek argüman ister"})
			}
			if e.tipCikar(n.Argumanlar[0]).Cesit == CMetin {
				e.ifade(n.Argumanlar[0])
				return
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			if e.tipCikar(n.Argumanlar[0]).Cesit == CKesir {
				m.call("f_kesir_metne")
			} else {
				m.call("f_sayi_metne")
			}
			return
		}
		if n.Ad == "uzunluk" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: uzunluk() tek argüman ister"})
			}
			t := e.tipCikar(n.Argumanlar[0])
			if t.Cesit != CMetin && t.Cesit != CListe {
				panic(TanHata{Mesaj: "elf: uzunluk() metin veya liste ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movDolayliOku(rRAX, rRAX, 0)
			return
		}
		if n.Ad == "ekle" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: ekle() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_liste_ekle")
			return
		}
		if n.Ad == "oku" {
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_oku")
			return
		}
		if n.Ad == "yaz_dosya" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: yaz_dosya() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_yaz_dosya")
			return
		}
		if n.Ad == "karakter" {
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_karakter")
			return
		}
		if n.Ad == "kod" {
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_kod")
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
		e.tipler[n.Ad] = e.tipCikar(n.Deger)
		e.ifade(n.Deger)
		if off, ok := e.yereller[n.Ad]; ok {
			m.movYerelYaz(off, rRAX)
		} else {
			e.genel[n.Ad] = true
			m.movGenelYaz("v_"+elfAd(n.Ad), rRAX)
		}

	case YazDugum:
		e.ifade(n.Deger)
		m.movKayit(rRDI, rRAX)
		switch e.tipCikar(n.Deger).Cesit {
		case CMetin:
			m.call("f_yaz_metin_deger")
		case CKesir:
			m.call("f_yaz_kesir")
		default:
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

	case HerDugum:
		bas := e.yeniEtiket("herbas")
		devam := e.yeniEtiket("herdevam")
		temizle := e.yeniEtiket("hertemiz")
		son := e.yeniEtiket("herson")
		// liste ve sayaci yigina koy: [rsp]=sayac, [rsp+8]=liste
		e.ifade(n.Liste)
		m.pushKayit(rRAX)
		m.pushImm32(0)
		e.tipler[n.Degisken] = func() Tip {
			lt := e.tipCikar(n.Liste)
			if lt.Cesit == CListe && lt.Eleman != nil {
				return *lt.Eleman
			}
			return TipTam
		}()
		e.donguBas = append(e.donguBas, devam)
		e.donguSon = append(e.donguSon, temizle)
		m.etiketKoy(bas)
		m.movRspOku(rRCX, 0) // sayac
		m.movRspOku(rRAX, 8) // liste
		m.movDolayliOku(rRDX, rRAX, 0)
		m.cmpKayit(rRCX, rRDX)
		m.jcc(0x8D, temizle) // jge -> bitir
		m.leaOge(rRAX, rRAX, rRCX)
		m.movDolayliOku(rRAX, rRAX, 0)
		if off, ok := e.yereller[n.Degisken]; ok {
			m.movYerelYaz(off, rRAX)
		} else {
			e.genel[n.Degisken] = true
			m.movGenelYaz("v_"+elfAd(n.Degisken), rRAX)
		}
		for _, st := range n.Govde {
			e.deyim(st)
		}
		m.etiketKoy(devam)
		m.incRspSifir()
		m.jmp(bas)
		m.etiketKoy(temizle)
		m.addImm32(rRSP, 16)
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
	eskiTip := map[string]Tip{}
	for k, v := range e.tipler {
		eskiTip[k] = v
	}
	for _, p := range n.Parametreler {
		if t, ok := e.parametreTipi[n.Ad+"/"+p]; ok {
			e.tipler[p] = t
		} else {
			e.tipler[p] = TipTam
		}
	}
	e.yereller = yereller
	e.govdeTipleriniTopla(n.Govde)
	for _, s := range n.Govde {
		e.deyim(s)
	}
	e.yereller = eski
	e.tipler = eskiTip

	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}


// ---------- YIGIN AYIRICI (brk tabanli, libc YOK) ----------
// heap_ptr: bir sonraki bos adres. _start'ta ilklenir.
func (e *elfUretici) yiginIlkle() {
	m := e.m
	// rax = brk(0)  -> mevcut program sonu
	m.movImm32(rRAX, 12) // sys_brk
	m.xorKayit(rRDI, rRDI)
	m.syscall()
	e.genel["__yigin"] = true
	m.movGenelYaz("v___yigin", rRAX)
	// brk(mevcut + 64 MB) -> alan ayir
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 64*1024*1024)
	m.movImm32(rRAX, 12)
	m.syscall()
}

// tan_ayir(rdi = bayt) -> rax = isaretci   (bump allocator)
func (e *elfUretici) yardimciAyir() {
	m := e.m
	m.etiketKoy("f_tan_ayir")
	e.genel["__yigin"] = true
	m.movGenelOku(rRAX, "v___yigin") // rax = mevcut
	m.addImm32(rRDI, 7)
	m.andImm32(rRDI, -8) // 8'e hizala
	m.addKayit(rRDI, rRAX)
	m.movGenelYaz("v___yigin", rRDI)
	m.ret()
}

// bellek_kopyala(rdi=hedef, rsi=kaynak, rdx=adet)
func (e *elfUretici) yardimciKopyala() {
	m := e.m
	m.etiketKoy("f_bellek_kopyala")
	m.testKayit(rRDX, rRDX)
	m.jcc(0x84, "Lkopya_son")
	m.etiketKoy("Lkopya_dongu")
	m.movBaytOku(rRSI)   // mov al, [rsi]
	m.movBaytYazAl(rRDI) // mov [rdi], al
	m.incKayit(rRSI)
	m.incKayit(rRDI)
	m.decKayit(rRDX)
	m.jcc(0x85, "Lkopya_dongu") // jnz
	m.etiketKoy("Lkopya_son")
	m.ret()
}

// yaz_metin_deger(rdi = metin isaretcisi)  [uzunluk:8][baytlar]
func (e *elfUretici) yardimciYazMetinDeger() {
	m := e.m
	m.etiketKoy("f_yaz_metin_deger")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movDolayliOku(rRDX, rRDI, 0) // rdx = uzunluk
	m.leaDolayli(rRSI, rRDI, 8)    // rsi = veri
	m.movImm32(rRAX, 1)
	m.movImm32(rRDI, 1)
	m.syscall()
	// satir sonu
	m.leaRbp(rRSI, -1)
	m.movBaytImm(rRSI, 10)
	m.movImm32(rRDX, 1)
	m.movImm32(rRAX, 1)
	m.movImm32(rRDI, 1)
	m.syscall()
	m.leave()
	m.ret()
}


// sayi_metne(rdi = int64) -> rax = yigindaki metin isaretcisi
func (e *elfUretici) yardimciSayiMetne() {
	m := e.m
	m.etiketKoy("f_sayi_metne")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 64)
	m.movKayit(rRAX, rRDI)
	m.leaRbp(rRCX, 0) // rcx = son baytin bir sonrasi
	m.xorKayit(rR8, rR8)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x8D, "Lsm_poz") // jge
	m.movImm32(rR8, 1)
	m.negKayit(rRAX)
	m.etiketKoy("Lsm_poz")
	m.cmpImm32(rRAX, 0)
	m.jcc(0x85, "Lsm_jegit") // jne
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '0')
	m.jmp("Lsm_isaret")
	m.etiketKoy("Lsm_jegit")
	m.movImm32(rR9, 10)
	m.etiketKoy("Lsm_dongu")
	m.cmpImm32(rRAX, 0)
	m.jcc(0x84, "Lsm_isaret") // je
	m.xorKayit(rRDX, rRDX)
	m.divKayit(rR9)
	m.addBaytImm(rRDX, '0')
	m.decKayit(rRCX)
	m.movBaytKayit(rRCX, rRDX)
	m.jmp("Lsm_dongu")
	m.etiketKoy("Lsm_isaret")
	m.cmpImm32(rR8, 0)
	m.jcc(0x84, "Lsm_bitti") // je
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '-')
	m.etiketKoy("Lsm_bitti")
	// uzunluk = rbp - rcx
	m.movKayit(rRDX, rRBP)
	m.subKayit(rRDX, rRCX)
	m.movYerelYaz(-40, rRCX) // basamak baslangici
	m.movYerelYaz(-48, rRDX) // uzunluk
	m.movKayit(rRDI, rRDX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-32, rRAX)
	m.movYerelOku(rRCX, -48)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunlugu yaz
	m.leaDolayli(rRDI, rRAX, 8)
	m.movYerelOku(rRSI, -40)
	m.movYerelOku(rRDX, -48)
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRAX, -32)
	m.leave()
	m.ret()
}

// metin_birlestir(rdi = a, rsi = b) -> rax = yeni metin
func (e *elfUretici) yardimciBirlestir() {
	m := e.m
	m.etiketKoy("f_metin_birlestir")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)  // a
	m.movYerelYaz(-16, rRSI) // b
	m.movDolayliOku(rRAX, rRDI, 0)
	m.movDolayliOku(rRCX, rRSI, 0)
	m.addKayit(rRAX, rRCX)
	m.movYerelYaz(-24, rRAX) // toplam uzunluk
	m.addImm32(rRAX, 16)
	m.movKayit(rRDI, rRAX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-32, rRAX) // yeni isaretci
	m.movYerelOku(rRCX, -24)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunlugu yaz
	// a'yi kopyala
	m.movYerelOku(rRSI, -8)
	m.movDolayliOku(rRDX, rRSI, 0)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movYerelOku(rRDI, -32)
	m.leaDolayli(rRDI, rRDI, 8)
	m.call("f_bellek_kopyala")
	// b'yi kopyala (a'nin uzunlugu kadar ilerle)
	m.movYerelOku(rRSI, -8)
	m.movDolayliOku(rRCX, rRSI, 0) // a uzunlugu
	m.movYerelOku(rRSI, -16)
	m.movDolayliOku(rRDX, rRSI, 0) // b uzunlugu
	m.leaDolayli(rRSI, rRSI, 8)
	m.movYerelOku(rRDI, -32)
	m.leaDolayli(rRDI, rRDI, 8)
	m.addKayit(rRDI, rRCX)
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRAX, -32)
	m.leave()
	m.ret()
}


// liste_ekle(rdi = liste, rsi = oge) -> rax = yeni liste
func (e *elfUretici) yardimciListeEkle() {
	m := e.m
	m.etiketKoy("f_liste_ekle")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)
	m.movYerelYaz(-16, rRSI)
	m.movDolayliOku(rRAX, rRDI, 0) // eski uzunluk
	m.movYerelYaz(-24, rRAX)
	m.incKayit(rRAX)
	m.shlImm(rRAX, 3) // *8
	m.addImm32(rRAX, 8)
	m.movKayit(rRDI, rRAX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-32, rRAX)
	m.movYerelOku(rRCX, -24)
	m.incKayit(rRCX)
	m.movDolayliYaz(rRAX, 0, rRCX) // yeni uzunluk
	// eski ogeleri kopyala
	m.leaDolayli(rRDI, rRAX, 8)
	m.movYerelOku(rRSI, -8)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movYerelOku(rRDX, -24)
	m.shlImm(rRDX, 3)
	m.call("f_bellek_kopyala")
	// yeni ogeyi sona yaz
	m.movYerelOku(rRAX, -32)
	m.movYerelOku(rRCX, -24)
	m.leaOge(rRAX, rRAX, rRCX)
	m.movYerelOku(rRSI, -16)
	m.movDolayliYaz(rRAX, 0, rRSI)
	m.movYerelOku(rRAX, -32)
	m.leave()
	m.ret()
}

// metin_indeks(rdi = metin, rsi = indeks) -> rax = tek harflik yeni metin
func (e *elfUretici) yardimciMetinIndeks() {
	m := e.m
	m.etiketKoy("f_metin_indeks")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.leaDolayli(rRDI, rRDI, 8) // veri baslangici
	m.movzxBaytDolayli(rRDX, rRDI, rRSI)
	m.movYerelYaz(-8, rRDX)
	m.movImm32(rRDI, 24)
	m.call("f_tan_ayir")
	m.movImm32(rRCX, 1)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunluk = 1
	m.movYerelOku(rRDX, -8)
	m.leaDolayli(rRDI, rRAX, 8)
	m.movBaytKayit(rRDI, rRDX) // mov [rdi], dl
	m.leave()
	m.ret()
}

// karakter(rdi = kod) -> rax = tek harflik metin
func (e *elfUretici) yardimciKarakter() {
	m := e.m
	m.etiketKoy("f_karakter")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)
	m.movImm32(rRDI, 24)
	m.call("f_tan_ayir")
	m.movImm32(rRCX, 1)
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.movYerelOku(rRDX, -8)
	m.leaDolayli(rRDI, rRAX, 8)
	m.movBaytKayit(rRDI, rRDX)
	m.leave()
	m.ret()
}

// kod(rdi = metin) -> rax = ilk baytin degeri
func (e *elfUretici) yardimciKod() {
	m := e.m
	m.etiketKoy("f_kod")
	m.leaDolayli(rRDI, rRDI, 8)
	m.xorKayit(rRSI, rRSI)
	m.movzxBaytDolayli(rRAX, rRDI, rRSI)
	m.ret()
}


// oku(rdi = yol metni) -> rax = dosya icerigi (metin)
// open(2)=2, read=0, close=3.  libc YOK.
func (e *elfUretici) yardimciOku() {
	m := e.m
	m.etiketKoy("f_oku")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	// yolu C-metnine cevir: yigina kopyala + sonuna 0
	m.movYerelYaz(-8, rRDI)
	m.movDolayliOku(rRAX, rRDI, 0)
	m.movYerelYaz(-16, rRAX)
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX)
	m.movKayit(rRDI, rRAX)
	m.movYerelOku(rRSI, -8)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movYerelOku(rRDX, -16)
	m.call("f_bellek_kopyala")
	// open(yol, O_RDONLY=0)
	m.movYerelOku(rRDI, -24)
	m.xorKayit(rRSI, rRSI)
	m.xorKayit(rRDX, rRDX)
	m.movImm32(rRAX, 2) // sys_open
	m.syscall()
	m.movYerelYaz(-32, rRAX) // fd
	// arabellek ayir (16 MB)
	m.movImm32(rRDI, 16*1024*1024)
	m.call("f_tan_ayir")
	m.movYerelYaz(-40, rRAX)
	// read(fd, buf+8, 16MB-16)
	m.movYerelOku(rRDI, -32)
	m.movYerelOku(rRSI, -40)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movImm32(rRDX, 16*1024*1024-16)
	m.xorKayit(rRAX, rRAX) // sys_read
	m.syscall()
	// uzunlugu yaz
	m.movYerelOku(rRCX, -40)
	m.movDolayliYaz(rRCX, 0, rRAX)
	// close(fd)
	m.movYerelOku(rRDI, -32)
	m.movImm32(rRAX, 3)
	m.syscall()
	m.movYerelOku(rRAX, -40)
	m.leave()
	m.ret()
}

// yaz_dosya(rdi = yol, rsi = icerik) -> rax = 0
// open(yol, O_WRONLY|O_CREAT|O_TRUNC=577, 0755)
func (e *elfUretici) yardimciYazDosya() {
	m := e.m
	m.etiketKoy("f_yaz_dosya")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)
	m.movYerelYaz(-16, rRSI)
	// yolu C-metnine cevir
	m.movDolayliOku(rRAX, rRDI, 0)
	m.movYerelYaz(-24, rRAX)
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-32, rRAX)
	m.movKayit(rRDI, rRAX)
	m.movYerelOku(rRSI, -8)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movYerelOku(rRDX, -24)
	m.call("f_bellek_kopyala")
	// open
	m.movYerelOku(rRDI, -32)
	m.movImm32(rRSI, 577) // O_WRONLY|O_CREAT|O_TRUNC
	m.movImm32(rRDX, 493) // 0755
	m.movImm32(rRAX, 2)
	m.syscall()
	m.movYerelYaz(-40, rRAX)
	// write(fd, icerik+8, uzunluk)
	m.movYerelOku(rRDI, -40)
	m.movYerelOku(rRSI, -16)
	m.movDolayliOku(rRDX, rRSI, 0)
	m.leaDolayli(rRSI, rRSI, 8)
	m.movImm32(rRAX, 1)
	m.syscall()
	// close
	m.movYerelOku(rRDI, -40)
	m.movImm32(rRAX, 3)
	m.syscall()
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}


// kesir_metne(rdi = double bit deseni) -> rax = metin
// 6 ondalik basamak, sondaki sifirlar kirpilir. libc YOK.
func (e *elfUretici) yardimciKesirMetne() {
	m := e.m
	m.etiketKoy("f_kesir_metne")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 160)
	m.movYerelYaz(-48, rRDI)
	// isaret biti
	m.movKayit(rRAX, rRDI)
	m.movImm64(rRCX, -9223372036854775808) // 0x8000...
	m.andKayit(rRAX, rRCX)
	m.movYerelYaz(-56, rRAX)
	// mutlak deger
	m.movKayit(rRAX, rRDI)
	m.movImm64(rRCX, 9223372036854775807) // 0x7FFF...
	m.andKayit(rRAX, rRCX)
	m.movqXmmKayit(0, rRAX)
	// tam kisim
	m.cvtKesirTam(rRCX, 0)
	m.movYerelYaz(-64, rRCX)
	m.cvtTamKesir(1, rRCX)
	m.sseIkili(0x5C, 0, 1) // subsd xmm0, xmm1  -> kesirli kisim
	// *1e6
	m.movImm64(rRAX, int64(uint64(4696837146684686336)))
	m.movqXmmKayit(2, rRAX)
	m.sseIkili(0x59, 0, 2) // mulsd
	// +0.5 (yuvarlama)
	m.movImm64(rRAX, int64(uint64(4602678819172646912)))
	m.movqXmmKayit(2, rRAX)
	m.sseIkili(0x58, 0, 2) // addsd
	m.cvtKesirTam(rRDX, 0)
	m.movYerelYaz(-72, rRDX)

	// --- sondaki sifirlari kirp ---
	m.movImm32(rR10, 6)
	m.movYerelOku(rRAX, -72)
	m.etiketKoy("Lkm_trim")
	m.cmpImm32(rR10, 1)
	m.jcc(0x8E, "Lkm_fdone") // jle
	m.movImm32(rR9, 10)
	m.xorKayit(rRDX, rRDX)
	m.divKayit(rR9)
	m.testKayit(rRDX, rRDX)
	m.jcc(0x85, "Lkm_undo") // jnz
	m.decKayit(rR10)
	m.jmp("Lkm_trim")
	m.etiketKoy("Lkm_undo")
	m.movImm32(rR9, 10)
	m.imulKayit(rRAX, rR9)
	m.addKayit(rRAX, rRDX)
	m.etiketKoy("Lkm_fdone")

	// --- basamaklari geriye dogru yaz (tampon: rbp-40 .. rbp-1) ---
	m.leaRbp(rRCX, 0)
	// kesirli kisim sifir ise nokta ve basamaklari atla -> "4" yaz, "4.0" degil
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lkm_tamsayi") // jz
	m.movKayit(rR11, rR10)
	m.etiketKoy("Lkm_fd")
	m.testKayit(rR11, rR11)
	m.jcc(0x84, "Lkm_dot") // jz
	m.movImm32(rR9, 10)
	m.xorKayit(rRDX, rRDX)
	m.divKayit(rR9)
	m.addBaytImm(rRDX, '0')
	m.decKayit(rRCX)
	m.movBaytKayit(rRCX, rRDX)
	m.decKayit(rR11)
	m.jmp("Lkm_fd")
	m.etiketKoy("Lkm_dot")
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '.')
	m.etiketKoy("Lkm_tamsayi")
	// tam kisim
	m.movYerelOku(rRAX, -64)
	m.testKayit(rRAX, rRAX)
	m.jcc(0x85, "Lkm_id") // jnz
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '0')
	m.jmp("Lkm_sign")
	m.etiketKoy("Lkm_id")
	m.etiketKoy("Lkm_idl")
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lkm_sign") // jz
	m.movImm32(rR9, 10)
	m.xorKayit(rRDX, rRDX)
	m.divKayit(rR9)
	m.addBaytImm(rRDX, '0')
	m.decKayit(rRCX)
	m.movBaytKayit(rRCX, rRDX)
	m.jmp("Lkm_idl")
	m.etiketKoy("Lkm_sign")
	m.cmpYerelImm(-56, 0)
	m.jcc(0x84, "Lkm_fin") // je
	m.decKayit(rRCX)
	m.movBaytImm(rRCX, '-')
	m.etiketKoy("Lkm_fin")
	// uzunluk = rbp - rcx
	m.movKayit(rRDX, rRBP)
	m.subKayit(rRDX, rRCX)
	m.movYerelYaz(-80, rRCX)
	m.movYerelYaz(-88, rRDX)
	m.movKayit(rRDI, rRDX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-96, rRAX)
	m.movYerelOku(rRCX, -88)
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.leaDolayli(rRDI, rRAX, 8)
	m.movYerelOku(rRSI, -80)
	m.movYerelOku(rRDX, -88)
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRAX, -96)
	m.leave()
	m.ret()
}

// yaz_kesir(rdi = double bit deseni)
func (e *elfUretici) yardimciYazKesir() {
	m := e.m
	m.etiketKoy("f_yaz_kesir")
	m.call("f_kesir_metne")
	m.movKayit(rRDI, rRAX)
	m.call("f_yaz_metin_deger")
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

	e := &elfUretici{m: yeniMakineKodu(), genel: map[string]bool{}, tipler: map[string]Tip{}, islevTipi: map[string]Tip{}, parametreTipi: map[string]Tip{}}

	// --- islev donus tiplerini cikar (sabit noktaya kadar yinele) ---
	// Cagri yerlerinden parametre tiplerini, dondur deyimlerinden donus tipini bul.
	for tur := 0; tur < 3; tur++ {
		for _, isv := range islevler {
			// parametre tiplerini cagri yerlerinden tahmin et
			eski := map[string]Tip{}
			for k, v := range e.tipler {
				eski[k] = v
			}
			for _, p := range isv.Parametreler {
				if _, var_ := e.tipler[p]; !var_ {
					e.tipler[p] = TipTam
				}
			}
			// parametre tiplerini uygula
			for _, p := range isv.Parametreler {
				if t, ok := e.parametreTipi[isv.Ad+"/"+p]; ok {
					e.tipler[p] = t
				}
			}
			e.govdeTipleriniTopla(isv.Govde) // yerel degisken tipleri
			if t, bulundu := e.dondurTipi(isv.Govde); bulundu {
				e.islevTipi[isv.Ad] = t
			}
			e.tipler = eski
		}
		// cagri yerlerinden parametre tipi ogren
		e.parametreTipleriniOgren(agac, islevler)
	}

	// yardımcılar + kullanıcı işlevleri + _start
	e.yardimciYazMetin()
	e.yardimciYazSayi()
	e.yardimciAyir()
	e.yardimciKopyala()
	e.yardimciYazMetinDeger()
	e.yardimciBirlestir()
	e.yardimciSayiMetne()
	e.yardimciListeEkle()
	e.yardimciMetinIndeks()
	e.yardimciKarakter()
	e.yardimciKod()
	e.yardimciOku()
	e.yardimciYazDosya()
	e.yardimciKesirMetne()
	e.yardimciYazKesir()
	for _, isv := range islevler {
		e.islevYaz(isv)
	}
	e.govdeTipleriniTopla(anaGovde) // ana govde tipleri
	e.m.etiketKoy("_start")
	e.yiginIlkle() // yigin ayiriciyi hazirla (brk)
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
	for i, str := range e.metinler {
		// 8 hizasina getir
		for (veriOfs+len(veri))%8 != 0 {
			veri = append(veri, 0)
		}
		veriAdres[fmt.Sprintf("s%d", i)] = veriOfs + len(veri)
		var uzun [8]byte
		binary.LittleEndian.PutUint64(uzun[:], uint64(len(str)))
		veri = append(veri, uzun[:]...)
		veri = append(veri, []byte(str)...)
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

	// ---------- SEMBOL TABLOSU (.symtab / .strtab / .shstrtab) ----------
	// nm ve gdb islev adlarini gorebilsin diye. Program calismasini
	// etkilemez; yalnizca hata ayiklama bilgisidir.
	if os.Getenv("TAN_SEMBOLSUZ") == "" {
		dosyaBaytlari = sembolTablosuEkle(dosyaBaytlari, m, kodOfs, veriAdres, veriOfs, len(veri))
	}

	if err := os.WriteFile(cikti, dosyaBaytlari, 0755); err != nil {
		fmt.Printf("ELF yazılamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ELF doğrudan yazıldı: %s  (%d bayt, kod %d bayt, veri %d bayt)\n",
		cikti, len(dosyaBaytlari), kodBoy, len(veri))
	fmt.Println("Kullanılan dış araç: YOK (as/ld/gcc/libc hiçbiri)")
}


// ============================================================
// SEMBOL TABLOSU — nm/gdb icin .symtab, .strtab, .shstrtab
// ============================================================

type sembol struct {
	ad    string
	deger uint64
	tur   byte // 2 = FUNC, 1 = OBJECT
	bolum uint16
}

func sembolTablosuEkle(dosya []byte, m *makineKodu, kodOfs int,
	veriAdres map[string]int, veriOfs int, veriBoy int) []byte {

	// --- sembolleri topla ---
	var semboller []sembol
	adlar := make([]string, 0, len(m.etiketler))
	for ad := range m.etiketler {
		adlar = append(adlar, ad)
	}
	// belirlenimci sira
	for i := 0; i < len(adlar); i++ {
		for j := i + 1; j < len(adlar); j++ {
			if adlar[j] < adlar[i] {
				adlar[i], adlar[j] = adlar[j], adlar[i]
			}
		}
	}
	for _, ad := range adlar {
		if strings.HasPrefix(ad, "L") { // yerel atlama etiketleri
			continue
		}
		semboller = append(semboller, sembol{
			ad:    ad,
			deger: uint64(elfTaban + kodOfs + m.etiketler[ad]),
			tur:   2, // STT_FUNC
			bolum: 1, // .text
		})
	}
	veriAdlar := make([]string, 0, len(veriAdres))
	for ad := range veriAdres {
		veriAdlar = append(veriAdlar, ad)
	}
	for i := 0; i < len(veriAdlar); i++ {
		for j := i + 1; j < len(veriAdlar); j++ {
			if veriAdlar[j] < veriAdlar[i] {
				veriAdlar[i], veriAdlar[j] = veriAdlar[j], veriAdlar[i]
			}
		}
	}
	for _, ad := range veriAdlar {
		semboller = append(semboller, sembol{
			ad:    ad,
			deger: uint64(elfTaban + veriAdres[ad]),
			tur:   1, // STT_OBJECT
			bolum: 2, // .data
		})
	}

	// --- .strtab ---
	strtab := []byte{0}
	adOfs := map[string]uint32{}
	for _, s := range semboller {
		adOfs[s.ad] = uint32(len(strtab))
		strtab = append(strtab, []byte(s.ad)...)
		strtab = append(strtab, 0)
	}

	// --- .symtab ---
	var symtab []byte
	sym := func(nameOfs uint32, info byte, shndx uint16, deger uint64, boy uint64) {
		var b [24]byte
		binary.LittleEndian.PutUint32(b[0:], nameOfs)
		b[4] = info
		b[5] = 0
		binary.LittleEndian.PutUint16(b[6:], shndx)
		binary.LittleEndian.PutUint64(b[8:], deger)
		binary.LittleEndian.PutUint64(b[16:], boy)
		symtab = append(symtab, b[:]...)
	}
	sym(0, 0, 0, 0, 0) // bos sembol
	for _, s := range semboller {
		// info = (BIND<<4) | TYPE ; BIND=1 (GLOBAL)
		sym(adOfs[s.ad], (1<<4)|s.tur, s.bolum, s.deger, 0)
	}

	// --- .shstrtab ---
	bolumAdlari := []string{"", ".text", ".data", ".symtab", ".strtab", ".shstrtab"}
	shstrtab := []byte{}
	bolumOfs := map[string]uint32{}
	for _, ad := range bolumAdlari {
		bolumOfs[ad] = uint32(len(shstrtab))
		shstrtab = append(shstrtab, []byte(ad)...)
		shstrtab = append(shstrtab, 0)
	}

	// --- yerlesim: mevcut dosyanin sonuna ekle ---
	hizala := func(b []byte, n int) []byte {
		for len(b)%n != 0 {
			b = append(b, 0)
		}
		return b
	}
	dosya = hizala(dosya, 8)
	symtabOfs := len(dosya)
	dosya = append(dosya, symtab...)
	strtabOfs := len(dosya)
	dosya = append(dosya, strtab...)
	shstrtabOfs := len(dosya)
	dosya = append(dosya, shstrtab...)
	dosya = hizala(dosya, 8)
	shoff := len(dosya)

	// --- bolum basliklari (6 adet x 64 bayt) ---
	bolumYaz := func(ad string, tur uint32, bayraklar uint64, adres uint64,
		ofs int, boy int, link uint32, info uint32, hiza uint64, girisBoy uint64) {
		var b [64]byte
		binary.LittleEndian.PutUint32(b[0:], bolumOfs[ad])
		binary.LittleEndian.PutUint32(b[4:], tur)
		binary.LittleEndian.PutUint64(b[8:], bayraklar)
		binary.LittleEndian.PutUint64(b[16:], adres)
		binary.LittleEndian.PutUint64(b[24:], uint64(ofs))
		binary.LittleEndian.PutUint64(b[32:], uint64(boy))
		binary.LittleEndian.PutUint32(b[40:], link)
		binary.LittleEndian.PutUint32(b[44:], info)
		binary.LittleEndian.PutUint64(b[48:], hiza)
		binary.LittleEndian.PutUint64(b[56:], girisBoy)
		dosya = append(dosya, b[:]...)
	}
	bolumYaz("", 0, 0, 0, 0, 0, 0, 0, 0, 0)                                             // NULL
	bolumYaz(".text", 1, 0x6, uint64(elfTaban+kodOfs), kodOfs, len(m.kod), 0, 0, 16, 0) // PROGBITS, ALLOC|EXEC
	bolumYaz(".data", 1, 0x3, uint64(elfTaban+veriOfs), veriOfs, veriBoy, 0, 0, 8, 0)   // PROGBITS, ALLOC|WRITE
	bolumYaz(".symtab", 2, 0, 0, symtabOfs, len(symtab), 4, 1, 8, 24)                   // SYMTAB
	bolumYaz(".strtab", 3, 0, 0, strtabOfs, len(strtab), 0, 0, 1, 0)                    // STRTAB
	bolumYaz(".shstrtab", 3, 0, 0, shstrtabOfs, len(shstrtab), 0, 0, 1, 0)              // STRTAB

	// --- ELF basligindaki bolum alanlarini guncelle ---
	binary.LittleEndian.PutUint64(dosya[0x28:], uint64(shoff)) // e_shoff
	binary.LittleEndian.PutUint16(dosya[0x3A:], 64)            // e_shentsize
	binary.LittleEndian.PutUint16(dosya[0x3C:], 6)             // e_shnum
	binary.LittleEndian.PutUint16(dosya[0x3E:], 5)             // e_shstrndx
	return dosya
}
