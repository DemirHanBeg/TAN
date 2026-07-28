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
	"path/filepath"
	"strings"
)

const (
	elfTaban  = 0x400000 // segmentin sanal taban adresi
	basliklar = 64 + 56  // ELF başlığı + 1 program header
	// icParcaBlokBoyut: içParcaLat'ın her çağrısında ayırdığı iplik-bloğu
	// (mmap). Düşük 64 bayt: [0]=bitti-bayrağı [8]=sonuç [16..64]=argüman
	// aktarım alanı (en fazla 6*8 bayt). Kalanı ÇOCUK YIĞINI (tepeden aşağı
	// büyür) — 1 MB, makul programlar için bol pay.
	icParcaBlokBoyut = 1 << 20
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
	rR13 = 13 // içParcaLat: iplik-blok adresini f_<ad>'ın çağırdığı HER işlev boyunca
	// korumak için ayrılmış — mevcut codegen r12-r15'e HİÇ dokunmuyor (yalnız
	// rax/rcx/rdx/rbx/rsp/rbp/rsi/rdi/r8-r11 kullanılıyor), bu yüzden r13
	// çağrı zinciri boyunca güvenle hayatta kalır.
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

// lea r64, [rip+disp32] -> KOD etiketine (duzeltmeler ile, jmp/call ile AYNI
// mekanizma — rel32 alanı HER ZAMAN komutun SONUNU baz alır, opcode fark
// etmez). leaVeri'den farkı: o VERİ (data segment, veriler/veriRef) adı
// çözer, bu KOD etiketi (etiketler/duzeltmeler) çözer — içParcaLat'ın iplik
// çıkış trambolininin ADRESİNİ (jump DEĞİL, bir DEĞER olarak) çocuk yığınına
// yazmak için gerekli.
func (m *makineKodu) leaEtiket(hedef byte, etiket string) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	m.bayt(rex, 0x8D, modrm(0, hedef, 5)) // rm=101 => RIP-göreli
	m.rel32(etiket)
}

// lock cmpxchg [taban], kaynak — RAX ile [taban] karşılaştırılır: eşitse
// [taban]=kaynak ve ZF=1; değilse RAX=[taban] (GÜNCEL değer) ve ZF=0.
// Atomik CAS (compare-and-swap) — kilit/futex'in temel taşı.
func (m *makineKodu) lockCmpxchgBellek(taban byte, kaynak byte) {
	rex := byte(0x48)
	if kaynak >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(0xF0, rex, 0x0F, 0xB1, modrm(0, kaynak, taban))
}

// lock xadd [taban], kaynakHedef — [taban] += kaynakHedef (atomik);
// kaynakHedef = [taban]'ın ESKİ değeri. Atomik fetch-and-add — atomikEkleHam'ın
// temel taşı.
func (m *makineKodu) lockXaddBellek(taban byte, kaynakHedef byte) {
	rex := byte(0x48)
	if kaynakHedef >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(0xF0, rex, 0x0F, 0xC1, modrm(0, kaynakHedef, taban))
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
func (m *makineKodu) rdtsc()   { m.bayt(0x0F, 0x31) } // edx:eax = zaman damgasi sayaci (rastgele() tohumu)

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

// mov r32, [taban+disp8]  (8B /r, 32-bit işlenen — üst 32 bit x86-64 kuralıyla
// otomatik sıfırlanır). movDolayliOku'dan farkı: REX.W YOK (64-bit değil).
func (m *makineKodu) movDolayli32Oku(hedef, taban byte, disp int8) {
	rex := byte(0)
	if hedef >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	if rex != 0 {
		m.bayt(0x40 | rex)
	}
	if disp == 0 && (taban&7) != rRBP {
		m.bayt(0x8B, modrm(0, hedef, taban))
		return
	}
	m.bayt(0x8B, modrm(1, hedef, taban), byte(disp))
}

// mov [taban+disp8], r32  (89 /r, 32-bit işlenen)
func (m *makineKodu) movDolayli32Yaz(taban byte, disp int8, kaynak byte) {
	rex := byte(0)
	if kaynak >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	if rex != 0 {
		m.bayt(0x40 | rex)
	}
	if disp == 0 && (taban&7) != rRBP {
		m.bayt(0x89, modrm(0, kaynak, taban))
		return
	}
	m.bayt(0x89, modrm(1, kaynak, taban), byte(disp))
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

// mov r64, [taban+disp32] -- movDolayliOku'nun disp8 (±127) sınırı olmayan
// hâli. Kayıt alan OFSETİ (8*alanIndeks) 15 alanı aşan kayıt tiplerinde
// disp8'i taşar (movDolayliYaz32 zaten disp32 kullanıyor, YAZMA tarafında
// bu sınır yoktu — OKUMA tarafında da aynı sınırsızlığı sağlamak için eklendi).
func (m *makineKodu) movDolayliOku32(hedef, taban byte, disp int32) {
	rex := byte(0x48)
	if hedef >= 8 {
		rex |= 4
	}
	if taban >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0x8B, modrm(2, hedef, taban))
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

// shr r64, imm8 (C1 /5) — mantiksal (isaretsiz) sag kaydirma
func (m *makineKodu) shrImm(r byte, v byte) {
	rex := byte(0x48)
	if r >= 8 {
		rex |= 1
	}
	m.bayt(rex, 0xC1, modrm(3, 5, r), v)
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
	CSozluk             // yigin isaretcisi: [uzunluk:8][kovaDizisiPtr:8], bkz f_sozluk_*
	CKayit              // yigin isaretcisi: [alan0:8][alan1:8]...[alanN-1:8], SABIT boy (uzunluk basligi YOK, sema kayitSemalari'nda derleme-zamaninda bilinir)
)

type Tip struct {
	Cesit    Cesit
	Eleman   *Tip   // liste/sozluk ise oge/deger tipi
	KayitAdi string // CKayit ise: hangi "kayıt Ad" semasina ait
}

var TipTam = Tip{Cesit: CTam}
var TipMetin = Tip{Cesit: CMetin}
var TipKesir = Tip{Cesit: CKesir}
var TipSozluk = Tip{Cesit: CSozluk}

func TipListe(e Tip) Tip { return Tip{Cesit: CListe, Eleman: &e} }
func TipKayit(ad string) Tip { return Tip{Cesit: CKayit, KayitAdi: ad} }

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
	case CKayit:
		return "kayıt<" + t.KayitAdi + ">"
	}
	return "bilinmeyen"
}

func (t Tip) esitMi(o Tip) bool {
	if t.Cesit != o.Cesit {
		return false
	}
	if t.Cesit == CKayit {
		return t.KayitAdi == o.KayitAdi
	}
	return true
}

// KayitSemasi: "kayıt Ad ... son" tanımının derleme-zamanı şeması — alan
// sırası (=bellek ofseti, alanIndeks[ad]*8) + metot govdeleri. Yorumlayıcının
// TanKayitTipi'yle AYNI amaç, statik/native tarafta.
type KayitSemasi struct {
	Ad          string
	Alanlar     []string
	AlanIndeks  map[string]int
	AlanTipleri map[string]Tip
	Metotlar    map[string]IslevDugum
}

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
	kayitSemalari map[string]*KayitSemasi // kayit adi -> sema
	islevParamSayisi map[string]int // islev adi -> parametre sayisi (içParcaLat icin)
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
		if n.Islec == "negatif" {
			return e.tipCikar(n.Sol)
		}
		if n.Islec == "değil" {
			return TipTam
		}
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
	case SozlukDugum:
		if len(n.Degerler) == 0 {
			return TipSozluk
		}
		et := e.tipCikar(n.Degerler[0])
		return Tip{Cesit: CSozluk, Eleman: &et}
	case IndeksDugum:
		ht := e.tipCikar(n.Hedef)
		if ht.Cesit == CMetin {
			return TipMetin // metin[i] -> tek harflik metin
		}
		if ht.Cesit == CListe && ht.Eleman != nil {
			return *ht.Eleman
		}
		if ht.Cesit == CSozluk && ht.Eleman != nil {
			return *ht.Eleman
		}
		return TipTam
	case KayitOlusturDugum:
		return Tip{Cesit: CKayit, KayitAdi: n.Ad}
	case AlanErisimDugum:
		ht := e.tipCikar(n.Hedef)
		if ht.Cesit == CKayit {
			if sema, ok := e.kayitSemalari[ht.KayitAdi]; ok {
				if t, ok2 := sema.AlanTipleri[n.Alan]; ok2 {
					return t
				}
			}
		}
		return TipTam
	case MetotCagriDugum:
		ht := e.tipCikar(n.Hedef)
		if ht.Cesit == CKayit {
			if t, ok := e.islevTipi["kayit_"+ht.KayitAdi+"_"+n.Metot]; ok {
				return t
			}
		}
		return TipTam
	case CagriDugum:
		if _, ok := e.kayitSemalari[n.Ad]; ok {
			// pozisyonel kayıt oluşturma: Ad(v1,v2) — parser bunu sıradan bir
			// CagriDugum olarak üretiyor (T_PARANTEZ_AC), kayıt adı olup
			// olmadığını yalnız SEMANTIK (kayitSemalari kaydı) ayırt edebilir.
			return Tip{Cesit: CKayit, KayitAdi: n.Ad}
		}
		switch n.Ad {
		case "uzunluk", "kod", "tamBol", "metinEsit", "sayı":
			return TipTam
		case "dosyaAcOku", "dosyaAcYaz", "dosyaAcOkuYaz", "dosyaKonumla",
			"dosyaYazBlok", "dosyaSenkron", "dosyaKapat", "dosyaSil",
			"hamOku8", "hamYaz8", "bellekEsle", "bellekCoz",
			"hamOku4", "hamYaz4", "hamOkuBayt", "hamYazBayt",
			"bellekKopyala", "bellekDoldur",
			"dosyaOkuBlokHam", "dosyaYazBlokHam",
			"içParcaLat", "iplikBekle", "kilitOlustur", "kilitle", "kilidiAc",
			"atomikEkleHam":
			return TipTam
		case "taban", "tavan", "yuvarla", "kök", "log", "e_üssü", "eÜssü":
			return TipKesir
		case "harfler":
			return TipListe(TipMetin)
		case "sözlük":
			return TipSozluk
		case "anahtarlar":
			return TipListe(TipMetin)
		case "varMı", "var_mı":
			return TipTam
		case "metin", "karakter", "oku", "arg", "metinBirlestir", "metinAl", "birleştir", "dosyaOkuBlok":
			return TipMetin
		case "listeYap":
			if len(n.Argumanlar) > 1 {
				return TipListe(e.tipCikar(n.Argumanlar[1]))
			}
			return TipListe(TipTam)
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

// sozlukElemanTipiBul: govde (ic ice bloklar dahil) icinde "ad" degiskenine
// yapilan ILK sozluk[anahtar] = deger atamasindaki degerin tipini bulur.
// SADECE govde icinde OLUSTURULUP DOLDURULAN yerel sozlukler icindir —
// parametre olarak gelen sozluklerin Eleman tipi cagri yerinden
// e.parametreTipi ile ayrica tasinir (bkz. parametreTipleriniOgren).
func (e *elfUretici) sozlukElemanTipiBul(govde []Dugum, ad string) (Tip, bool) {
	for _, d := range govde {
		switch n := d.(type) {
		case IndeksAtamaDugum:
			if dv, ok := n.Hedef.(DegiskenDugum); ok && dv.Ad == ad {
				return e.tipCikar(n.Deger), true
			}
		case EgerDugum:
			if t, ok := e.sozlukElemanTipiBul(n.Govde, ad); ok {
				return t, true
			}
			if t, ok := e.sozlukElemanTipiBul(n.Degilse, ad); ok {
				return t, true
			}
		case IkenDugum:
			if t, ok := e.sozlukElemanTipiBul(n.Govde, ad); ok {
				return t, true
			}
		case HerDugum:
			if t, ok := e.sozlukElemanTipiBul(n.Govde, ad); ok {
				return t, true
			}
		}
	}
	return TipTam, false
}

// sozlukElemanlariniCoz: e.tipler icindeki, Eleman'i henuz cozulmemis her
// CSozluk degiskeni icin verilen govdelerin HER BIRINI (sirayla) tarayip
// deger tipini bulur. Birden cok govde kabul eder cunku GLOBAL bir sozluk
// (modul ust-seviyesinde olusturulan) bir ISLEV GOVDESI icinden doldurulmus
// olabilir (bkz. BAgaci.tan: "kYaprakMi = sozluk()" ust seviyede ama
// "kYaprakMi[id] = x" bagacDugumOlustur() icinde) — TEK govde arasa bunu
// hic bulamaz. Cagri yeri bu fonksiyonu her-islev-govdesi-ayri degil, TUM
// govdeleri (anaGovde + islevler) BIRLIKTE vererek cagirmali — aksi halde
// per-islev "e.tipler = eski" restore'u cozumlemeyi siler (bu tam olarak
// yasanan bug'di: cozum bagacDugumOlustur govdesinde BULUNUYORDU ama o
// islevin eski-restore'u ile birlikte KAYBOLUYORDU cunku anaGovde'nin
// kendi e.tipler girisi hic guncellenmemisti).
func (e *elfUretici) sozlukElemanlariniCoz(govdeler ...[]Dugum) {
	for ad, t := range e.tipler {
		if t.Cesit == CSozluk && t.Eleman == nil {
			for _, govde := range govdeler {
				if vt, ok := e.sozlukElemanTipiBul(govde, ad); ok {
					vtKopya := vt
					e.tipler[ad] = Tip{Cesit: CSozluk, Eleman: &vtKopya}
					break
				}
			}
		}
	}
}

// kayitOlusturSiteleriTopla: bir ifade agacinin ICINDE (herhangi bir
// derinlikte) bulunan TUM "Ad(...)" (pozisyonel, CagriDugum'a parser
// tarafindan ayni sekilde ayristirilir — kayit adi olup olmadigi yalniz
// kayitSemalari kaydiyla ayirt edilir) ve "Ad{alan:v}" (KayitOlusturDugum,
// adlandirilmis) kayit olusturma sitelerini toplar. sozlukElemanTipiBul'un
// aksine GENEL bir agac gezisi gerekiyor cunku kayit olusturma herhangi bir
// ifadenin icinde nested olabilir (ör. yaz(Nokta(1,2).x)).
func (e *elfUretici) kayitOlusturSiteleriTopla(d Dugum, sonuc *[]KayitOlusturDugum) {
	if d == nil {
		return
	}
	switch n := d.(type) {
	case KayitOlusturDugum:
		*sonuc = append(*sonuc, n)
		for _, v := range n.Degerler {
			e.kayitOlusturSiteleriTopla(v, sonuc)
		}
	case CagriDugum:
		if sema, ok := e.kayitSemalari[n.Ad]; ok {
			adlar := append([]string{}, sema.Alanlar...)
			if len(adlar) > len(n.Argumanlar) {
				adlar = adlar[:len(n.Argumanlar)]
			}
			*sonuc = append(*sonuc, KayitOlusturDugum{Ad: n.Ad, AlanAdlari: adlar, Degerler: n.Argumanlar})
		}
		for _, a := range n.Argumanlar {
			e.kayitOlusturSiteleriTopla(a, sonuc)
		}
	case IkiliDugum:
		e.kayitOlusturSiteleriTopla(n.Sol, sonuc)
		e.kayitOlusturSiteleriTopla(n.Sag, sonuc)
	case ListeDugum:
		for _, el := range n.Elemanlar {
			e.kayitOlusturSiteleriTopla(el, sonuc)
		}
	case SozlukDugum:
		for _, v := range n.Degerler {
			e.kayitOlusturSiteleriTopla(v, sonuc)
		}
		for _, k := range n.Anahtarlar {
			e.kayitOlusturSiteleriTopla(k, sonuc)
		}
	case IndeksDugum:
		e.kayitOlusturSiteleriTopla(n.Hedef, sonuc)
		e.kayitOlusturSiteleriTopla(n.Indeks, sonuc)
	case AlanErisimDugum:
		e.kayitOlusturSiteleriTopla(n.Hedef, sonuc)
	case MetotCagriDugum:
		e.kayitOlusturSiteleriTopla(n.Hedef, sonuc)
		for _, a := range n.Argumanlar {
			e.kayitOlusturSiteleriTopla(a, sonuc)
		}
	}
}

// kayitOlusturSiteleriToplaDeyim: govdedeki (ic ice bloklar dahil) HER
// deyimin icindeki ifadeleri kayitOlusturSiteleriTopla ile tarar.
func (e *elfUretici) kayitOlusturSiteleriToplaDeyim(govde []Dugum, sonuc *[]KayitOlusturDugum) {
	for _, d := range govde {
		switch n := d.(type) {
		case AtamaDugum:
			e.kayitOlusturSiteleriTopla(n.Deger, sonuc)
		case YazDugum:
			e.kayitOlusturSiteleriTopla(n.Deger, sonuc)
		case DondurDugum:
			e.kayitOlusturSiteleriTopla(n.Deger, sonuc)
		case IndeksAtamaDugum:
			e.kayitOlusturSiteleriTopla(n.Hedef, sonuc)
			e.kayitOlusturSiteleriTopla(n.Indeks, sonuc)
			e.kayitOlusturSiteleriTopla(n.Deger, sonuc)
		case AlanAtamaDugum:
			e.kayitOlusturSiteleriTopla(n.Hedef, sonuc)
			e.kayitOlusturSiteleriTopla(n.Deger, sonuc)
		case CagriDugum:
			e.kayitOlusturSiteleriTopla(n, sonuc)
		case MetotCagriDugum:
			e.kayitOlusturSiteleriTopla(n, sonuc)
		case EgerDugum:
			e.kayitOlusturSiteleriTopla(n.Kosul, sonuc)
			e.kayitOlusturSiteleriToplaDeyim(n.Govde, sonuc)
			e.kayitOlusturSiteleriToplaDeyim(n.Degilse, sonuc)
		case IkenDugum:
			e.kayitOlusturSiteleriTopla(n.Kosul, sonuc)
			e.kayitOlusturSiteleriToplaDeyim(n.Govde, sonuc)
		case HerDugum:
			e.kayitOlusturSiteleriTopla(n.Liste, sonuc)
			e.kayitOlusturSiteleriToplaDeyim(n.Govde, sonuc)
		}
	}
}

// kayitAlanTipleriniCoz: her kayit semasindaki her alanin tipini, o kayit
// tipinin (govdeler icindeki HERHANGI bir yerdeki) olusturma sitelerinden
// cikarir. sozlukElemanlariniCoz ile AYNI on-gecis stratejisi (sabit
// noktaya kadar her turda yeniden cagrilir cunku deger ifadesinin kendi
// tipi de asamali ogreniliyor olabilir). Bir alan icin HICBIR olusturma
// sitesi bulunamazsa TipTam (guvenli varsayilan) kalir.
func (e *elfUretici) kayitAlanTipleriniCoz(govdeler ...[]Dugum) {
	for kayitAdi, sema := range e.kayitSemalari {
		var siteler []KayitOlusturDugum
		for _, govde := range govdeler {
			e.kayitOlusturSiteleriToplaDeyim(govde, &siteler)
		}
		for _, alan := range sema.Alanlar {
			for _, site := range siteler {
				if site.Ad != kayitAdi {
					continue
				}
				for i, aad := range site.AlanAdlari {
					if aad == alan && i < len(site.Degerler) {
						sema.AlanTipleri[alan] = e.tipCikar(site.Degerler[i])
					}
				}
			}
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
//
// NOT: govde parametresi olarak SADECE tek bir kapsamin (ana govde YA DA
// tek bir islevin govdesi) verilmesi sart. e.tipler o an hangi kapsamin
// yerel degiskenlerini tutuyorsa (govdeTipleriniTopla ile doldurulmus),
// cagri argumanlarinin tipi de o kapsama gore cikarilir. Ana govde ile
// TUM islev govdelerini TEK cagriyla (paylasilan/sifirlanmis e.tipler
// durumuyla) taramak, islev-yerel degiskenleri disarida cagri yapan baska
// bir islevin parametre tipini yanlis (varsayilan "tam") ogrenmesine yol
// aciyordu — cagiran her govde icin ayri ayri cagrilmali (bkz. derleElf).
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
		case IndeksAtamaDugum:
			// liste[i] = f(...) / sözlük[k] = f(...): n.Deger'e ULASILMIYORDU
			// (bu case hic yoktu) — bu yola giren cagri yerleri parametre tipi
			// hic ogrenemiyordu (bkz. BAgaci.tan digerleriEkleyerekOlustur()).
			gez(n.Hedef)
			gez(n.Indeks)
			gez(n.Deger)
		case YazDugum:
			gez(n.Deger)
		case SozlukDugum:
			for _, x := range n.Anahtarlar {
				gez(x)
			}
			for _, x := range n.Degerler {
				gez(x)
			}
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
		case KayitOlusturDugum:
			for _, x := range n.Degerler {
				gez(x)
			}
		case AlanErisimDugum:
			gez(n.Hedef)
		case AlanAtamaDugum:
			gez(n.Hedef)
			gez(n.Deger)
		case MetotCagriDugum:
			// bu.metot(args) — "bu" (n.Argumanlar[i]'nin bir kayması) ile
			// AYNI mantik: cagri yerindeki argumanlarin tipinden metodun
			// (sentetik ad ile islevler listesine eklenmis) parametre
			// tiplerini ogren. Argumanlar[0] gercek "bu" DEGIL (bu, Hedef
			// uzerinden baglanir) — bu yuzden Parametreler[i+1] kaydirmasi.
			ht := e.tipCikar(n.Hedef)
			if ht.Cesit == CKayit {
				if isv, ok := adlar[kayitMetotAdi(ht.KayitAdi, n.Metot)]; ok {
					for i, a := range n.Argumanlar {
						pIdx := i + 1
						if pIdx < len(isv.Parametreler) {
							at := e.tipCikar(a)
							if at.Cesit != CTam {
								e.parametreTipi[isv.Ad+"/"+isv.Parametreler[pIdx]] = at
							}
						}
					}
				}
			}
			gez(n.Hedef)
			for _, a := range n.Argumanlar {
				gez(a)
			}
		}
	}
	for _, d := range agac {
		gez(d)
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
		// tekli eksi (-x): Parser bunu Sag=nil IkiliDugum{"negatif", x, nil} olarak
		// veriyor. Once ele alinmali — asagidaki genel yol n.Sag'i KOSULSUZ
		// okur, nil ile cakisir ("elf: bu ifade desteklenmiyor (<nil>)").
		if n.Islec == "negatif" {
			e.ifade(n.Sol)
			if e.tipCikar(n.Sol).Cesit == CKesir {
				m.movImm64(rRCX, int64(-1)<<63) // isaret biti: 0x8000000000000000
				m.xorKayit(rRAX, rRCX)
			} else {
				m.negKayit(rRAX)
			}
			return
		}
		// "değil x" da unary — Parser.go IkiliDugum{"değil", x, nil} olarak
		// veriyor (CagriDugum "değil(x)" cagri bicimiyle KARISTIRILMASIN,
		// o ayri ve zaten CagriDugum kolunda destekli). Ayni nil-Sag tuzagi.
		if n.Islec == "değil" {
			e.ifade(n.Sol)
			m.testKayit(rRAX, rRAX)
			m.setcc(0x94, rRAX) // sete — sifirsa 1
			m.movzx(rRAX, rRAX)
			return
		}
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
		// --- KISA DEVRE: ve / veya ---
		// Sol taraf sonucu belirliyorsa sag taraf HIC calismamali.
		// Onceden iki taraf da degerlendiriliyordu: "b != 0 ve 10 / b > 1"
		// b=0 iken SIGFPE veriyordu.
		if n.Islec == "ve" || n.Islec == "veya" {
			bitis := e.yeniEtiket("kisadevre")
			e.ifade(n.Sol)
			m.testKayit(rRAX, rRAX)
			m.setcc(0x95, rRAX) // setne -> 0/1'e normalle
			m.movzx(rRAX, rRAX)
			m.testKayit(rRAX, rRAX)
			if n.Islec == "ve" {
				m.jcc(0x84, bitis) // jz  -> sol yanlis, sonuc 0
			} else {
				m.jcc(0x85, bitis) // jnz -> sol dogru, sonuc 1
			}
			e.ifade(n.Sag)
			m.testKayit(rRAX, rRAX)
			m.setcc(0x95, rRAX)
			m.movzx(rRAX, rRAX)
			m.etiketKoy(bitis)
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
		// --- METIN ESITLIGI ("==" / "!=") ---
		// Operandlardan biri metin ise pointer degil ICERIK karsilastirilir.
		if (n.Islec == "==" || n.Islec == "!=") && (solT.Cesit == CMetin || sagT.Cesit == CMetin) {
			e.ifade(n.Sol)
			m.pushKayit(rRAX)
			e.ifade(n.Sag)
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_metin_esit")
			if n.Islec == "!=" {
				m.testKayit(rRAX, rRAX)
				m.setcc(0x94, rRAX) // sete — esitse (test sonucu 0) -> 1, yani "farkli degil"
				m.movzx(rRAX, rRAX)
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

	case SozlukDugum:
		// {"a": 1, "b": 2, ...} -- bos sozluk yarat, sirayla koy() cagir.
		// f_sozluk_koy rax'ta sozluk isaretcisini dondurur, bu yuzden dongu
		// arasinda ayrica saklamaya gerek yok (bir sonraki push bunu alir).
		m.call("f_sozluk_yap")
		for i := range n.Anahtarlar {
			m.pushKayit(rRAX) // sozluk (bu ana kadarki)
			e.ifade(n.Degerler[i])
			m.pushKayit(rRAX) // deger
			e.ifade(n.Anahtarlar[i])
			it := e.tipCikar(n.Anahtarlar[i])
			if it.Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if it.Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.movKayit(rRSI, rRAX) // anahtar (metin)
			m.popKayit(rRDX)       // deger
			m.popKayit(rRDI)       // sozluk
			m.call("f_sozluk_koy") // rax = sozluk (degismez, sonraki tura hazir)
		}

	case KayitOlusturDugum:
		// Ad{alan1: v1, alan2: v2} -- adlandirilmis kayit olusturma.
		sema, ok := e.kayitSemalari[n.Ad]
		if !ok {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: bilinmeyen kayıt tipi: " + n.Ad})
		}
		e.kayitOlusturYaz(sema, n.AlanAdlari, n.Degerler, n.Satir)

	case AlanErisimDugum:
		// hedef.alan -- sabit ofsetten okuma (struct layout, uzunluk basligi yok).
		ht := e.tipCikar(n.Hedef)
		sema, ok := e.kayitSemalari[ht.KayitAdi]
		if !ok {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '." + n.Alan + "' erişimi — hedef bir kayıt değil"})
		}
		idx, ok2 := sema.AlanIndeks[n.Alan]
		if !ok2 {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '" + sema.Ad + "' kaydında böyle bir alan yok: " + n.Alan})
		}
		e.ifade(n.Hedef)
		m.movDolayliOku32(rRAX, rRAX, int32(8*idx))

	case MetotCagriDugum:
		// hedef.metot(args) -- "bu" ilk argüman olarak bağlanır, statik
		// (derleme-zamanı) çözümlenen DOĞRUDAN CALL (bkz. kayitMetotAdi notu).
		ht := e.tipCikar(n.Hedef)
		sema, ok := e.kayitSemalari[ht.KayitAdi]
		if !ok {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '." + n.Metot + "(...)' çağrısı — hedef bir kayıt değil"})
		}
		if _, ok2 := sema.Metotlar[n.Metot]; !ok2 {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '" + sema.Ad + "' kaydında böyle bir metot yok: " + n.Metot})
		}
		if len(n.Argumanlar) > 5 {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: metot çağrısında en fazla 5 argüman (+bu)"})
		}
		e.ifade(n.Hedef)
		m.pushKayit(rRAX) // bu
		for _, a := range n.Argumanlar {
			e.ifade(a)
			m.pushKayit(rRAX)
		}
		metotKayitlari := []byte{rRDI, rRSI, rRDX, rRCX, rR8, rR9}
		toplam := len(n.Argumanlar) + 1
		for i := toplam - 1; i >= 0; i-- {
			m.popKayit(metotKayitlari[i])
		}
		m.call("f_" + elfAd(kayitMetotAdi(sema.Ad, n.Metot)))

	case IndeksDugum:
		ht := e.tipCikar(n.Hedef)
		e.ifade(n.Hedef)
		m.pushKayit(rRAX)
		e.ifade(n.Indeks)
		if ht.Cesit == CSozluk {
			it := e.tipCikar(n.Indeks)
			if it.Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if it.Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
		}
		m.movKayit(rRSI, rRAX)
		m.popKayit(rRDI)
		if ht.Cesit == CMetin {
			m.call("f_metin_indeks")
		} else if ht.Cesit == CSozluk {
			m.call("f_sozluk_al")
		} else {
			m.leaOge(rRAX, rRDI, rRSI)
			m.movDolayliOku(rRAX, rRAX, 0)
		}

	case MetinDugum:
		ad := fmt.Sprintf("s%d", len(e.metinler))
		e.metinler = append(e.metinler, n.Deger)
		m.leaVeri(rRAX, ad)

	case CagriDugum:
		if sema, ok := e.kayitSemalari[n.Ad]; ok {
			// Ad(v1,v2) -- pozisyonel kayit olusturma (parser bunu sıradan
			// bir CagriDugum olarak ayristirir, kayit adi olup olmadigi
			// yalniz kayitSemalari kaydiyla ayirt edilir, bkz. tipCikar notu).
			alanAdlari := append([]string{}, sema.Alanlar...)
			if len(alanAdlari) > len(n.Argumanlar) {
				alanAdlari = alanAdlari[:len(n.Argumanlar)]
			}
			e.kayitOlusturYaz(sema, alanAdlari, n.Argumanlar, n.Satir)
			return
		}
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
		if n.Ad == "sistemDur" {
			if len(n.Argumanlar) > 0 {
				e.ifade(n.Argumanlar[0])
				m.movKayit(rRDI, rRAX)
			} else {
				m.xorKayit(rRDI, rRDI)
			}
			m.movImm32(rRAX, 60) // sys_exit
			m.syscall()
			return
		}
		if n.Ad == "uzunluk" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: uzunluk() tek argüman ister"})
			}
			t := e.tipCikar(n.Argumanlar[0])
			if t.Cesit != CMetin && t.Cesit != CListe && t.Cesit != CSozluk {
				panic(TanHata{Satir: n.Satir, Mesaj: fmt.Sprintf("elf: uzunluk() metin, liste veya sözlük ister (satir %d, cesit=%v)", n.Satir, t.Cesit)})
			}
			e.ifade(n.Argumanlar[0])
			m.movDolayliOku(rRAX, rRAX, 0)
			return
		}
		if n.Ad == "listeYap" {
			if len(n.Argumanlar) < 1 {
				panic(TanHata{Mesaj: "elf: listeYap(n, deger) argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			if len(n.Argumanlar) > 1 {
				e.ifade(n.Argumanlar[1])
			} else {
				m.movImm32(rRAX, 0)
			}
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_liste_yap")
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
		if n.Ad == "tamBol" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: tamBol() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRCX, rRAX)
			m.popKayit(rRAX)
			m.cqo()
			m.idivKayit(rRCX)
			return
		}
		if n.Ad == "yazBaytlar" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: yazBaytlar() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_yaz_baytlar")
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
		if n.Ad == "sayı" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: sayı() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_sayi")
			return
		}
		if n.Ad == "dosyaVarMi" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: dosyaVarMi() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_dosya_var_mi")
			return
		}
		if n.Ad == "ekle_dosya" || n.Ad == "ekleDosya" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: ekle_dosya() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_ekle_dosya")
			return
		}
		// --- Rastgele erişimli (positional) dosya G/Ç — fd tabanlı ---
		if n.Ad == "dosyaAcOku" || n.Ad == "dosyaAcYaz" || n.Ad == "dosyaAcOkuYaz" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: " + n.Ad + "() tek argüman (yol) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			switch n.Ad {
			case "dosyaAcOku":
				m.call("f_dosya_ac_oku")
			case "dosyaAcYaz":
				m.call("f_dosya_ac_yaz")
			default:
				m.call("f_dosya_ac_okuyaz")
			}
			return
		}
		if n.Ad == "dosyaKonumla" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: dosyaKonumla() iki argüman (fd, ofset) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_dosya_konumla")
			return
		}
		if n.Ad == "dosyaOkuBlok" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: dosyaOkuBlok() iki argüman (fd, uzunluk) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_dosya_oku_blok")
			return
		}
		if n.Ad == "dosyaYazBlok" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: dosyaYazBlok() iki argüman (fd, metin) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_dosya_yaz_blok")
			return
		}
		if n.Ad == "dosyaSenkron" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: dosyaSenkron() tek argüman (fd) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_dosya_senkron")
			return
		}
		if n.Ad == "dosyaKapat" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: dosyaKapat() tek argüman (fd) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_dosya_kapat")
			return
		}
		if n.Ad == "dosyaSil" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: dosyaSil() tek argüman (yol) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_dosya_sil")
			return
		}
		// --- Ham bellek (word) erişimi + mmap/munmap ---
		if n.Ad == "hamOku8" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: hamOku8() tek argüman (adres) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_ham_oku8")
			return
		}
		if n.Ad == "hamYaz8" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: hamYaz8() iki argüman (adres, deger) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_ham_yaz8")
			return
		}
		if n.Ad == "bellekEsle" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: bellekEsle() tek argüman (boyut) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_bellek_esle")
			return
		}
		if n.Ad == "bellekCoz" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: bellekCoz() iki argüman (adres, boyut) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_bellek_coz")
			return
		}
		if n.Ad == "hamOku4" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: hamOku4() tek argüman (adres) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_ham_oku4")
			return
		}
		if n.Ad == "hamYaz4" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: hamYaz4() iki argüman (adres, deger) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_ham_yaz4")
			return
		}
		if n.Ad == "hamOkuBayt" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: hamOkuBayt() tek argüman (adres) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_ham_oku_bayt")
			return
		}
		if n.Ad == "hamYazBayt" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: hamYazBayt() iki argüman (adres, deger) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_ham_yaz_bayt")
			return
		}
		if n.Ad == "bellekKopyala" {
			if len(n.Argumanlar) != 3 {
				panic(TanHata{Mesaj: "elf: bellekKopyala() üç argüman (hedef, kaynak, uzunluk) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[2])
			m.movKayit(rRDX, rRAX)
			m.popKayit(rRSI)
			m.popKayit(rRDI)
			m.call("f_ham_bellek_tasi")
			return
		}
		if n.Ad == "bellekDoldur" {
			if len(n.Argumanlar) != 3 {
				panic(TanHata{Mesaj: "elf: bellekDoldur() üç argüman (adres, deger, uzunluk) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[2])
			m.movKayit(rRDX, rRAX)
			m.popKayit(rRSI)
			m.popKayit(rRDI)
			m.call("f_bellek_doldur")
			return
		}
		if n.Ad == "dosyaOkuBlokHam" {
			if len(n.Argumanlar) != 3 {
				panic(TanHata{Mesaj: "elf: dosyaOkuBlokHam() üç argüman (fd, adres, uzunluk) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[2])
			m.movKayit(rRDX, rRAX)
			m.popKayit(rRSI)
			m.popKayit(rRDI)
			m.call("f_dosya_oku_blok_ham")
			return
		}
		if n.Ad == "dosyaYazBlokHam" {
			if len(n.Argumanlar) != 3 {
				panic(TanHata{Mesaj: "elf: dosyaYazBlokHam() üç argüman (fd, adres, uzunluk) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[2])
			m.movKayit(rRDX, rRAX)
			m.popKayit(rRSI)
			m.popKayit(rRDI)
			m.call("f_dosya_yaz_blok_ham")
			return
		}
		if n.Ad == "kilitOlustur" {
			if len(n.Argumanlar) != 0 {
				panic(TanHata{Mesaj: "elf: kilitOlustur() argüman almaz"})
			}
			// kilit = 8 baytlık mmap'lı sözcük (mmap sıfır-doldurur -> başlangıç
			// 0 = kilitli-değil), bellekEsle(8) ile BİREBİR aynı çağrı.
			m.movImm32(rRDI, 8)
			m.call("f_bellek_esle")
			return
		}
		if n.Ad == "kilitle" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: kilitle() bir argüman (kilit) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_kilit_al")
			return
		}
		if n.Ad == "kilidiAc" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: kilidiAc() bir argüman (kilit) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_kilit_birak")
			return
		}
		if n.Ad == "atomikEkleHam" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: atomikEkleHam() iki argüman (adres, miktar) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_atomik_ekle_ham")
			return
		}
		if n.Ad == "iplikBekle" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: iplikBekle() bir argüman (iplik) ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_iplik_bekle")
			return
		}
		if n.Ad == "içParcaLat" {
			if len(n.Argumanlar) < 1 {
				panic(TanHata{Mesaj: "elf: içParcaLat(islevAdi, ...args) en az bir argüman ister"})
			}
			dv, ok := n.Argumanlar[0].(DegiskenDugum)
			if !ok {
				panic(TanHata{Mesaj: "elf: içParcaLat() ilk argüman DERLEME-ZAMANI bilinen bir işlev adı olmalı (çalışma-zamanı fonksiyon DEĞERİ native ELF'te desteklenmiyor)"})
			}
			hedefAd := dv.Ad
			paramSayisi, biliniyor := e.islevParamSayisi[hedefAd]
			if !biliniyor {
				panic(TanHata{Mesaj: "elf: içParcaLat(): bilinmeyen işlev: " + hedefAd})
			}
			verilenArgSayisi := len(n.Argumanlar) - 1
			if verilenArgSayisi != paramSayisi {
				panic(TanHata{Mesaj: fmt.Sprintf("elf: içParcaLat(%s): %d argüman bekleniyor, %d verildi", hedefAd, paramSayisi, verilenArgSayisi)})
			}
			if paramSayisi > 6 {
				panic(TanHata{Mesaj: "elf: içParcaLat(): en fazla 6 parametreli işlev desteklenir"})
			}
			etNo := e.yeniEtiket("icprl")
			etParent := "Lparent_" + etNo
			// 1) iplik-blok ayır (BLOK_BOYUT — düşük 64 bayt başlık/argüman
			// aktarımı için ayrılmış, kalanı ÇOCUK YIĞINI): bellekEsle ile AYNI
			// f_bellek_esle rutini.
			m.movImm32(rRDI, icParcaBlokBoyut)
			m.call("f_bellek_esle")
			m.pushKayit(rRAX) // blok adresini YIĞINDA sakla (register basıncı yok)
			// 2) argümanları değerlendirip blok+16+i*8'e yaz — clone syscall'ın
			// KENDİSİ rdi/rsi/rdx/r8'i kullanacağından argümanlar REGISTER'DA
			// TAŞINAMAZ, bu yüzden ÖNCE belleğe, ÇOCUK tarafında GERİ okunuyor.
			for i, argNode := range n.Argumanlar[1:] {
				e.ifade(argNode) // rax = argüman değeri
				m.movRspOku(rRCX, 0)
				m.movDolayliYaz(rRCX, int8(16+i*8), rRAX)
			}
			m.popKayit(rRBX) // rbx = blok adresi (bu dizinin geri kalanında sabit)
			// 3) çocuk yığın tepesi = blok+BLOK_BOYUT-8; dönüş adresi olarak
			// trambolin ETİKETİNİN ADRESİ yazılır (jmp DEĞİL — [rsp]'te bir
			// DEĞER olarak, f_<ad>'ın kendi 'ret'i oraya düşecek).
			m.movKayit(rRAX, rRBX)
			m.addImm32(rRAX, icParcaBlokBoyut-8)
			m.leaEtiket(rRCX, "f_iplik_cikis")
			m.movDolayliYaz(rRAX, 0, rRCX)
			// 4) clone(flags, child_sp, ptid=NULL, ctid=NULL, tls=NULL)
			// CLONE_VM|CLONE_FS|CLONE_FILES|CLONE_SIGHAND|CLONE_THREAD = 0x10F00
			m.movImm32(rRDI, 0x10F00)
			m.movKayit(rRSI, rRAX) // çocuk yığın işaretçisi
			m.xorKayit(rRDX, rRDX)
			m.xorKayit(rR10, rR10)
			m.xorKayit(rR8, rR8)
			m.movImm32(rRAX, 56) // sys_clone
			m.syscall()
			m.testKayit(rRAX, rRAX)
			m.jcc(0x85, etParent) // jnz -> ebeveyn (rax = çocuk tid, 0 DEĞİL)
			// --- ÇOCUK YOLU (rax == 0) ---
			m.movKayit(rR13, rRBX) // r13 = blok adresi, f_<ad> BOYUNCA korunur
			kayitlar := []byte{rRDI, rRSI, rRDX, rRCX, rR8, rR9}
			for i := 0; i < paramSayisi; i++ {
				m.movDolayliOku(kayitlar[i], rRBX, int8(16+i*8))
			}
			m.jmp("f_" + hedefAd) // call DEĞİL — dönüş adresi zaten yığında (adım 3)
			// --- EBEVEYN YOLU ---
			m.etiketKoy(etParent)
			m.movKayit(rRAX, rRBX) // dönüş = iplik tutamacı (blok adresi)
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
		if n.Ad == "harfler" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: harfler() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_harfler")
			return
		}
		if n.Ad == "birleştir" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: birleştir() liste ister"})
			}
			lt := e.tipCikar(n.Argumanlar[0])
			if lt.Cesit != CListe {
				panic(TanHata{Mesaj: "elf: birleştir() liste ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			switch {
			case lt.Eleman != nil && lt.Eleman.Cesit == CMetin:
				m.call("f_birlestir_metin")
			case lt.Eleman != nil && lt.Eleman.Cesit == CKesir:
				m.call("f_birlestir_kesir")
			default:
				m.call("f_birlestir_tam")
			}
			return
		}
		if n.Ad == "sil" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: sil() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			it := e.tipCikar(n.Argumanlar[1])
			if it.Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if it.Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_sozluk_sil")
			return
		}
		if n.Ad == "rastgele" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: rastgele() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_rastgele")
			return
		}
		if n.Ad == "kök" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: kök() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			if e.tipCikar(n.Argumanlar[0]).Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.movqXmmKayit(0, rRAX)
			m.sseIkili(0x51, 0, 0) // sqrtsd xmm0, xmm0 (donanim, tek makine komutu)
			m.movqKayitXmm(rRAX, 0)
			return
		}
		if n.Ad == "log" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: log() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			if e.tipCikar(n.Argumanlar[0]).Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.movKayit(rRDI, rRAX)
			m.call("f_log")
			return
		}
		if n.Ad == "e_üssü" || n.Ad == "eÜssü" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: e_üssü() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			if e.tipCikar(n.Argumanlar[0]).Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.movKayit(rRDI, rRAX)
			m.call("f_e_ussu")
			return
		}
		if n.Ad == "yuvarla" {
			if len(n.Argumanlar) < 1 || len(n.Argumanlar) > 2 {
				panic(TanHata{Mesaj: "elf: yuvarla() bir veya iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			if e.tipCikar(n.Argumanlar[0]).Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.pushKayit(rRAX) // sayı (ondalık ham)
			if len(n.Argumanlar) == 2 {
				e.ifade(n.Argumanlar[1])
			} else {
				m.movImm32(rRAX, 0)
			}
			m.movKayit(rRSI, rRAX) // basamak
			m.popKayit(rRDI)       // sayı
			m.call("f_yuvarla")
			return
		}
		if n.Ad == "sözlük" {
			if len(n.Argumanlar) != 0 {
				panic(TanHata{Mesaj: "elf: sözlük() argüman almaz"})
			}
			m.call("f_sozluk_yap")
			return
		}
		if n.Ad == "anahtarlar" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: anahtarlar() tek argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_sozluk_anahtarlar")
			return
		}
		if n.Ad == "varMı" || n.Ad == "var_mı" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: varMı() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			it := e.tipCikar(n.Argumanlar[1])
			if it.Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if it.Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_sozluk_varmi")
			return
		}
		if n.Ad == "parçala" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: parçala() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_parcala")
			return
		}
		if n.Ad == "taban" || n.Ad == "tavan" {
			// SSE4.1 roundsd YOK (donanim varsayimi yapmiyoruz) — cvttsd2si
			// (sifira dogru kes) + gerekirse 1 duzelt: taban icin kesim
			// deger BUYUKSE (negatif ondalik) 1 azalt, tavan icin kesim
			// deger KUCUKSE 1 arttir.
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: fmt.Sprintf("elf: %s() tek argüman ister", n.Ad)})
			}
			e.ifade(n.Argumanlar[0])
			if e.tipCikar(n.Argumanlar[0]).Cesit != CKesir {
				m.cvtTamKesir(0, rRAX)
				m.movqKayitXmm(rRAX, 0)
			}
			m.movqXmmKayit(0, rRAX) // xmm0 = x
			m.cvtKesirTam(rRCX, 0) // rcx = kes(x)
			m.cvtTamKesir(1, rRCX) // xmm1 = float(rcx)
			m.comisd(1, 0)         // xmm1(kesilmis) vs xmm0(x)
			if n.Ad == "taban" {
				m.setcc(0x97, rRAX) // al=1 kesilmis > x ise (seta)
			} else {
				m.setcc(0x92, rRAX) // al=1 kesilmis < x ise (setb)
			}
			m.movzx(rRAX, rRAX)
			if n.Ad == "taban" {
				m.subKayit(rRCX, rRAX)
			} else {
				m.addKayit(rRCX, rRAX)
			}
			m.cvtTamKesir(0, rRCX) // xmm0 = float(rcx) sonuc
			m.movqKayitXmm(rRAX, 0)
			return
		}
		if n.Ad == "metinEsit" {
			// TancElf.tan kendi kendini derlerken kullanıyor: "==" statik tip
			// izlemesi DEĞİŞKEN/PARAMETRE BAŞINA basit olduğundan (TancElf'in
			// kendi parametreTipleriniOgren'i yok), her-döngü değişkeni veya
			// parametre üzerinden gelen metin değerleri "tam" sanılıp pointer
			// karşılaştırması yapılıyordu. metinEsit() İÇERİK karşılaştırmasını
			// statik tipten BAĞIMSIZ garanti eder.
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: metinEsit() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_metin_esit")
			return
		}
		if n.Ad == "metinAl" {
			// metinEsit/metinBirlestir ile aynı gerekçe: TancElf.tan'ın kendi
			// alan()/tokenle() gibi işlevleri "s[i]" indekslemesini PARAMETRE
			// üzerinden yapıyor; parametre tipi hep "tam" sanıldığından "s[i]"
			// metin-baytı DEĞİL 8-bayt-adımlı LİSTE ögesi gibi okunuyordu —
			// alan() TÜM alan ayıklamasının merkezinde olduğundan bu, sabit
			// nokta denemesinde gen2'nin baştan sona bozuk üretilmesinin kök
			// nedeniydi. metinAl() İÇERİK indekslemesini statik tipten
			// BAĞIMSIZ garanti eder.
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: metinAl() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_metin_indeks")
			return
		}
		if n.Ad == "metinBirlestir" {
			// metinEsit ile aynı gerekçe: statik tip izlemeden BAĞIMSIZ metin
			// birleştirme (parametre/her-döngü değişkeni "tam" sanıldığında
			// "+" tam sayı toplaması yapıp metni bozuyordu).
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: metinBirlestir() iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_metin_birlestir")
			return
		}
		if n.Ad == "arenaAyir" {
			if len(n.Argumanlar) != 1 {
				panic(TanHata{Mesaj: "elf: arenaAyir(boyut) bir argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.movKayit(rRDI, rRAX)
			m.call("f_arena_ayir")
			return
		}
		if n.Ad == "arenaSerbest" {
			if len(n.Argumanlar) != 2 {
				panic(TanHata{Mesaj: "elf: arenaSerbest(isaretci, boyut) iki argüman ister"})
			}
			e.ifade(n.Argumanlar[0])
			m.pushKayit(rRAX)
			e.ifade(n.Argumanlar[1])
			m.movKayit(rRSI, rRAX)
			m.popKayit(rRDI)
			m.call("f_arena_serbest")
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
		yeniTip := e.tipCikar(n.Deger)
		if yeniTip.Cesit == CSozluk && yeniTip.Eleman == nil {
			// "d = sözlük()" her calistiginda BOS (Eleman=nil) tip uretir —
			// sozlukElemanlariniCoz()'un govdeyi tarayip onceden cozdugu
			// deger tipini burada EZMEYELIM (aksi halde bu satirdan hemen
			// sonraki "yaz(d[k])" gibi kullanimlar yine "tam" a duser).
			if eski, ok := e.tipler[n.Ad]; ok && eski.Cesit == CSozluk && eski.Eleman != nil {
				yeniTip = eski
			}
		}
		e.tipler[n.Ad] = yeniTip
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

	case MetotCagriDugum:
		// hedef.metot(args) — sonucu kullanmadan (yan etki için) çağrı.
		e.ifade(n)

	case AlanAtamaDugum:
		// hedef.alan = deger — sabit ofsete yazma (struct layout).
		ht := e.tipCikar(n.Hedef)
		sema, ok := e.kayitSemalari[ht.KayitAdi]
		if !ok {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '." + n.Alan + " = ...' ataması — hedef bir kayıt değil"})
		}
		idx, ok2 := sema.AlanIndeks[n.Alan]
		if !ok2 {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: '" + sema.Ad + "' kaydında böyle bir alan yok: " + n.Alan})
		}
		e.ifade(n.Deger)
		m.pushKayit(rRAX) // deger
		e.ifade(n.Hedef)
		m.popKayit(rRCX) // deger
		m.movDolayliYaz32(rRAX, int32(8*idx), rRCX)

	case IndeksAtamaDugum:
		// liste[i] = deger — yerinde degistirme (DOKUMANTASYON.md'de var,
		// elf arka ucunda hic implemente edilmemisti; TancElf.tan'in
		// gelistirilmesi sirasinda bulundu — "atama hedefi ==" gorulunce
		// "bu deyim desteklenmiyor" ile panikliyordu).
		ht := e.tipCikar(n.Hedef)
		if ht.Cesit == CSozluk {
			it := e.tipCikar(n.Indeks)
			e.ifade(n.Deger)
			m.pushKayit(rRAX) // deger
			e.ifade(n.Hedef)
			m.pushKayit(rRAX) // sozluk isaretcisi
			e.ifade(n.Indeks)
			if it.Cesit != CMetin {
				m.movKayit(rRDI, rRAX)
				if it.Cesit == CKesir {
					m.call("f_kesir_metne")
				} else {
					m.call("f_sayi_metne")
				}
			}
			m.movKayit(rRSI, rRAX) // rsi = anahtar (metin)
			m.popKayit(rRDI)       // rdi = sozluk isaretcisi
			m.popKayit(rRDX)       // rdx = deger
			m.call("f_sozluk_koy")
			return
		}
		if ht.Cesit != CListe {
			panic(TanHata{Satir: n.Satir, Mesaj: "elf: yalnız liste[i] veya sözlük[anahtar] = değer destekleniyor"})
		}
		e.ifade(n.Deger)
		m.pushKayit(rRAX) // deger
		e.ifade(n.Hedef)
		m.pushKayit(rRAX) // liste isaretcisi
		e.ifade(n.Indeks)
		m.movKayit(rRSI, rRAX) // rsi = indeks
		m.popKayit(rRDI)       // rdi = liste isaretcisi
		m.leaOge(rRAX, rRDI, rRSI)
		m.popKayit(rRCX) // rcx = deger
		m.movDolayliYaz(rRAX, 0, rRCX)

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

// asmDegiskenleriTopla: bir gövdedeki atanan değişken adlarını toplar
// (yerel çerçeve boyutu hesaplamak için). Önceden arsiv/DerleAsm.go'da
// (Kademe 2 arka ucuyla) paylaşılıyordu — asm arka ucu arşivlenince
// elf arka ucunun kendi bağımlılığı olarak buraya taşındı.
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

// kayitMetotAdi: bir kayıt metodunun SENTETİK işlev adı — "bu.metot(...)"
// çağrısı hedefin STATİK tipinden (KayitAdi) hangi metot govdesinin
// çağrılacağını DERLEME ZAMANINDA kesin bilir (Tan'da arayüz/polimorfizm
// yok — aynı kayıt-adı+metot-adı HER ZAMAN aynı govdeye karşılık gelir),
// bu yüzden metot çağrısı DOĞRUDAN CALL ile çözülür — DerleElf.go:3366'daki
// "fonksiyon pointer/indirect call yok" sınırı BURAYA uygulanmıyor (o sınır
// bir DEĞİŞKENE atanmış/dinamik çözümlenen fonksiyon değerleri içindi).
func kayitMetotAdi(kayitAdi, metotAdi string) string {
	return "kayit_" + kayitAdi + "_" + metotAdi
}

// kayitOlusturYaz: sema.Alanlar sırasına göre (8*alanSayısı) bayt ayırıp
// alanAdlari/degerler eşleşmesine göre her alanı yazar. KayitOlusturDugum
// (adlandırılmış {alan:v}) VE pozisyonel CagriDugum (Ad(v1,v2)) İKİSİ DE
// bu tek yardımcıyı kullanır — tek fark çağıran tarafın alanAdlari'yi nasıl
// ürettiği (sırasıyla: doğrudan token'dan / sema.Alanlar'dan pozisyonel).
func (e *elfUretici) kayitOlusturYaz(sema *KayitSemasi, alanAdlari []string, degerler []Dugum, satir int) {
	m := e.m
	m.movImm32(rRDI, int32(8*len(sema.Alanlar)))
	m.call("f_tan_ayir")
	m.pushKayit(rRAX) // kayıt işaretçisi
	for i, alan := range alanAdlari {
		if i >= len(degerler) {
			break
		}
		e.ifade(degerler[i])
		m.popKayit(rRCX) // kayıt işaretçisi (bu ana kadarki)
		m.pushKayit(rRCX)
		idx, ok := sema.AlanIndeks[alan]
		if !ok {
			panic(TanHata{Satir: satir, Mesaj: "elf: '" + sema.Ad + "' kaydında böyle bir alan yok: " + alan})
		}
		m.movDolayliYaz32(rRCX, int32(8*idx), rRAX)
	}
	m.popKayit(rRAX)
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
	e.sozlukElemanlariniCoz(n.Govde)
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
	e.genel["__yiginSon"] = true
	m.movGenelYaz("v___yigin", rRAX)
	// brk(mevcut + 64 MB) -> ilk alan
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 64*1024*1024)
	m.pushKayit(rRDI)
	m.movImm32(rRAX, 12)
	m.syscall()
	m.popKayit(rRDI)
	m.movGenelYaz("v___yiginSon", rRDI) // yigin siniri
}

// tan_ayir(rdi = bayt) -> rax = isaretci   (bump allocator)
func (e *elfUretici) yardimciAyir() {
	m := e.m
	m.etiketKoy("f_tan_ayir")
	e.genel["__yigin"] = true
	e.genel["__yiginSon"] = true
	m.addImm32(rRDI, 7)
	m.andImm32(rRDI, -8)             // rdi = 8'e hizalanmis boyut
	m.movGenelOku(rRAX, "v___yigin") // rax = mevcut isaretci (donus degeri)
	m.movKayit(rRCX, rRAX)
	m.addKayit(rRCX, rRDI) // rcx = yeni bump

	// --- SINIR KONTROLU ---
	// Bu kontrol olmadan ayirici sinirin otesine yaziyordu: 64 MB'i asan
	// her program SESSIZCE segfault veriyordu. Olcek bagimli oldugu icin
	// kod uretim hatasi gibi gorunuyordu.
	m.movGenelOku(rRDX, "v___yiginSon")
	m.cmpKayit(rRCX, rRDX)
	m.jcc(0x86, "Layir_tamam") // jbe -> sinir icinde, buyutmeye gerek yok

	// --- YIGINI BUYUT: brk(yeni bump + 64 MB) ---
	m.pushKayit(rRAX)
	m.pushKayit(rRCX)
	m.movKayit(rRDI, rRCX)
	m.addImm32(rRDI, 64*1024*1024)
	m.pushKayit(rRDI)
	m.movImm32(rRAX, 12) // sys_brk
	m.syscall()
	m.popKayit(rRDI)
	m.cmpKayit(rRAX, rRDI)
	m.jcc(0x82, "Layir_bellekYok") // jb -> brk basarisiz
	m.movGenelYaz("v___yiginSon", rRDI)
	m.popKayit(rRCX)
	m.popKayit(rRAX)

	m.etiketKoy("Layir_tamam")
	m.movGenelYaz("v___yigin", rRCX)
	m.ret()

	// Bellek bitti: sessizce bozulmak yerine acik cikis kodu ver.
	m.etiketKoy("Layir_bellekYok")
	m.movImm32(rRAX, 60) // sys_exit
	m.movImm32(rRDI, 3)  // 3 = bellek yetersiz
	m.syscall()
}

// bellek_kopyala(rdi=hedef, rsi=kaynak, rdx=adet)
// arenaAyir(rdi=boyut) -> rax=ptr — Faz A #5 (arena+serbest — bkz.
// NexusCore/FazA_Kapsam.md) için EKLENEN, f_tan_ayir'in (yukarıdaki bump
// allocator) ÜSTÜNE ek bir katman. Var olan liste/sözlük/kayıt codegen'i
// bunu KULLANMIYOR/DEĞİŞMEDİ — sıfır regresyon riski, tamamen yeni/katmanlı.
//
// TASARIM: boyut<=512 ise 64 "sınıf" (8,16,...,512 bayt) için ayrı bir
// serbest-liste tutuluyor (v___arenaTablo, ilk kullanımda f_tan_ayir ile
// TEK SEFER 520 bayt ayrılan bir tablo — leaOge'nin sabit +8 payını
// karşılamak için 8 bayt fazladan). arenaSerbest() ile bırakılan bir blok
// varsa O(1) yeniden kullanılır; yoksa f_tan_ayir'a düşülür (asla
// başarısız olmaz, sadece yeniden kullanamaz). boyut>512 HER ZAMAN
// doğrudan bump allocator'a gider (büyük nesneler için yeniden kullanım
// yok — basit/güvenli varsayılan, "gerçek GC" değil ama dokümanın kendi
// izin verdiği "arena" alternatifi).
//
// ÖNEMLİ SINIR: boyut BAŞLIĞI TUTULMUYOR — arenaSerbest() boyutu
// KULLANICIDAN açıkça alır (hamAyir/hamOku/hamYaz ile AYNI "ham, kullanıcı
// sorumlu" felsefesi). Yanlış boyut vermek serbest listeyi bozar,
// doğrulama YAPILMIYOR (bilinçli, düşük seviye bir araç).
func (e *elfUretici) yardimciArenaAyir() {
	m := e.m
	m.etiketKoy("f_arena_ayir")
	e.genel["__arenaTablo"] = true

	m.addImm32(rRDI, 7)
	m.andImm32(rRDI, -8) // rdi = 8'e hizalanmış boyut

	m.cmpImm32(rRDI, 512)
	m.jcc(0x87, "Larena_buyuk") // ja -> 512'den büyük, yeniden kullanım yok

	m.pushKayit(rRDI) // boyutu (bump-fallback ihtimali için) sakla
	m.movKayit(rRAX, rRDI)
	m.shrImm(rRAX, 3)
	m.decKayit(rRAX) // rax = sınıfIndeksi (0..63)
	m.pushKayit(rRAX)

	m.movGenelOku(rRCX, "v___arenaTablo")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x85, "Larena_tabloHazir") // jnz -> tablo zaten var

	m.movImm32(rRDI, 520) // 64*8 sınıf + leaOge'nin +8 payı
	m.call("f_tan_ayir")
	m.movGenelYaz("v___arenaTablo", rRAX)
	m.movKayit(rRCX, rRAX)

	m.etiketKoy("Larena_tabloHazir")
	m.popKayit(rRAX) // sınıfIndeksi
	m.pushKayit(rRAX)
	m.leaOge(rRDX, rRCX, rRAX)       // rdx = &tablo[sınıfIndeksi]
	m.movDolayliOku32(rRAX, rRDX, 0) // rax = bu sınıfın serbest-liste başı
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Larena_gerideDon") // jz -> serbest blok yok, bump'a düş

	m.movDolayliOku32(rRCX, rRAX, 0) // rcx = [rax] (bir sonraki serbest blok)
	m.movDolayliYaz32(rRDX, 0, rRCX) // tablo[sınıfIndeksi] = rcx
	m.popKayit(rRCX)                 // sınıfIndeksi temizle
	m.popKayit(rRDI)                 // boyutu temizle
	m.ret()                          // rax zaten geri dönen işaretçi (yeniden kullanılan blok)

	m.etiketKoy("Larena_gerideDon")
	m.popKayit(rRCX) // sınıfIndeksi temizle
	m.popKayit(rRDI) // boyutu geri al
	m.call("f_tan_ayir")
	m.ret()

	m.etiketKoy("Larena_buyuk")
	m.call("f_tan_ayir")
	m.ret()
}

// arenaSerbest(rdi=ptr, rsi=boyut): ptr'yi (boyut arenaAyir()'a verilenle
// AYNI olmalı) serbest listeye ekler. boyut>512 ise ya da tablo hiç
// oluşmadıysa (arenaAyir hiç çağrılmamış) sessizce YOKSAYAR — güvenli
// varsayılan, asla çökmez.
func (e *elfUretici) yardimciArenaSerbest() {
	m := e.m
	m.etiketKoy("f_arena_serbest")
	e.genel["__arenaTablo"] = true

	m.addImm32(rRSI, 7)
	m.andImm32(rRSI, -8) // rsi = 8'e hizalanmış boyut

	m.cmpImm32(rRSI, 512)
	m.jcc(0x87, "Lserbest_yoksay") // ja -> çok büyük, yeniden kullanılmıyor

	m.movKayit(rRAX, rRSI)
	m.shrImm(rRAX, 3)
	m.decKayit(rRAX) // rax = sınıfIndeksi

	m.movGenelOku(rRCX, "v___arenaTablo")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lserbest_yoksay") // jz -> tablo hiç oluşmadı, güvenli yoksay

	m.leaOge(rRDX, rRCX, rRAX)       // rdx = &tablo[sınıfIndeksi]
	m.movDolayliOku32(rRCX, rRDX, 0) // rcx = mevcut sınıf başı (eski head)
	m.movDolayliYaz32(rRDI, 0, rRCX) // [ptr] = eski head (bırakılan bloğun İÇİNE yazılıyor)
	m.movDolayliYaz32(rRDX, 0, rRDI) // tablo[sınıfIndeksi] = ptr (yeni head)

	m.etiketKoy("Lserbest_yoksay")
	m.movImm32(rRAX, 0)
	m.ret()
}

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

// listeYap(rdi = n, rsi = baslangic degeri) -> rax = n ogeli liste
// Tek seferde ayirir; ekle() ile n kez buyutmenin O(n^2) maliyetini onler.
func (e *elfUretici) yardimciListeYap() {
	m := e.m
	m.etiketKoy("f_liste_yap")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)  // n
	m.movYerelYaz(-16, rRSI) // baslangic
	m.movKayit(rRAX, rRDI)
	m.shlImm(rRAX, 3)
	m.addImm32(rRAX, 16)
	m.movKayit(rRDI, rRAX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRCX, -8)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunluk
	m.xorKayit(rRCX, rRCX)
	m.etiketKoy("Lly_dongu")
	m.movYerelOku(rRDX, -8)
	m.cmpKayit(rRCX, rRDX)
	m.jcc(0x8D, "Lly_bitti") // jge
	m.movYerelOku(rRAX, -24)
	m.leaOge(rRAX, rRAX, rRCX)
	m.movYerelOku(rRSI, -16)
	m.movDolayliYaz(rRAX, 0, rRSI)
	m.incKayit(rRCX)
	m.jmp("Lly_dongu")
	m.etiketKoy("Lly_bitti")
	m.movYerelOku(rRAX, -24)
	m.leave()
	m.ret()
}

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

// metin_esit(rdi = a, rsi = b) -> rax = 1 esitse 0 degilse (bayt bazli icerik
// karsilastirmasi). ONEMLI: bu fonksiyon olmadan "==" metin (Deger)
// operandlarinda RAW POINTER karsilastirmasi yapiyordu (asagidaki ifade()
// IkiliDugum kolunda cmpKayit dogrudan iki isaretciyi karsilastiriyordu) —
// iki ayri tahsis edilmis ama icerigi AYNI olan metin her zaman "esit degil"
// cikiyordu. Bu, hicbir regresyon testinde yakalanmiyordu (20 testin hicbiri
// metin==metin kontrolu icermiyor) ama bir ayristirici/derleyici (ornegin
// TancElf.tan) icin temel bir gereksinim.
func (e *elfUretici) yardimciMetinEsit() {
	m := e.m
	m.etiketKoy("f_metin_esit")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)  // a
	m.movYerelYaz(-16, rRSI) // b
	m.movDolayliOku(rRAX, rRDI, 0)
	m.movDolayliOku(rRCX, rRSI, 0)
	m.cmpKayit(rRAX, rRCX)
	m.jcc(0x85, "Lme_farkli") // jne — uzunluklar farkli
	m.movYerelYaz(-24, rRAX)  // n = uzunluk
	m.movYerelImm(-32, 0)     // i = 0
	m.etiketKoy("Lme_dongu")
	m.movYerelOku(rRAX, -24)
	m.movYerelOku(rRCX, -32)
	m.cmpKayit(rRCX, rRAX)
	m.jcc(0x8D, "Lme_esit") // jge — i >= n, sona kadar esit
	m.movYerelOku(rRDI, -8)
	m.leaDolayli(rRDI, rRDI, 8)
	m.movYerelOku(rRSI, -32)
	m.movzxBaytDolayli(rR8, rRDI, rRSI)
	m.movYerelOku(rRDI, -16)
	m.leaDolayli(rRDI, rRDI, 8)
	m.movYerelOku(rRSI, -32)
	m.movzxBaytDolayli(rR9, rRDI, rRSI)
	m.cmpKayit(rR8, rR9)
	m.jcc(0x85, "Lme_farkli") // jne
	m.movYerelOku(rRCX, -32)
	m.incKayit(rRCX)
	m.movYerelYaz(-32, rRCX)
	m.jmp("Lme_dongu")
	m.etiketKoy("Lme_esit")
	m.movImm32(rRAX, 1)
	m.jmp("Lme_bitti")
	m.etiketKoy("Lme_farkli")
	m.movImm32(rRAX, 0)
	m.etiketKoy("Lme_bitti")
	m.leave()
	m.ret()
}

// harfler(rdi = metin) -> rax = liste (her ogesi f_metin_indeks ile uretilmis
// tek-baytlik metin). Bayt bazlidir (rune degil) — elf arka ucunun butun metin
// indeksleme/kod/karakter yardimcilariyla AYNI kural (metinAl, f_metin_indeks).
func (e *elfUretici) yardimciHarfler() {
	m := e.m
	m.etiketKoy("f_harfler")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 40)
	m.movYerelYaz(-8, rRDI)        // metin
	m.movDolayliOku(rRAX, rRDI, 0) // n = uzunluk
	m.movYerelYaz(-16, rRAX)       // n
	m.movKayit(rRCX, rRAX)
	m.shlImm(rRCX, 3)
	m.addImm32(rRCX, 8)
	m.movKayit(rRDI, rRCX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX) // liste
	m.movYerelOku(rRCX, -16)
	m.movDolayliYaz(rRAX, 0, rRCX) // liste uzunlugu = n
	m.xorKayit(rRCX, rRCX)         // i = 0
	m.etiketKoy("Lhf_dongu")
	m.movYerelOku(rRDX, -16)
	m.cmpKayit(rRCX, rRDX)
	m.jcc(0x8D, "Lhf_bitti") // jge
	m.movYerelYaz(-32, rRCX) // i sakla (cagri rcx'i bozar)
	m.movYerelOku(rRDI, -8)
	m.movKayit(rRSI, rRCX)
	m.call("f_metin_indeks") // rax = tek-baytlik metin
	m.movYerelOku(rRDX, -24)
	m.movYerelOku(rRCX, -32)
	m.leaOge(rRDX, rRDX, rRCX) // liste + i*8 + 8
	m.movDolayliYaz(rRDX, 0, rRAX)
	m.movYerelOku(rRCX, -32)
	m.incKayit(rRCX)
	m.jmp("Lhf_dongu")
	m.etiketKoy("Lhf_bitti")
	m.movYerelOku(rRAX, -24)
	m.leave()
	m.ret()
}

// metin_araligi(rdi = metin, rsi = baslangic, rdx = uzunluk) -> rax = yeni
// metin (alt dize kopyasi). f_metin_indeks'in genellestirilmisi (uzunluk=1
// ile sinirli degil) — parçala()'nin hem aday-karsilastirma hem de sonuc
// parcalarini uretmesi icin kullanilir.
func (e *elfUretici) yardimciMetinAraligi() {
	m := e.m
	m.etiketKoy("f_metin_araligi")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 40)
	m.movYerelYaz(-8, rRDI)  // metin
	m.movYerelYaz(-16, rRSI) // baslangic
	m.movYerelYaz(-24, rRDX) // uzunluk
	m.movKayit(rRAX, rRDX)
	m.addImm32(rRAX, 8)
	m.movKayit(rRDI, rRAX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-32, rRAX) // yeni metin
	m.movYerelOku(rRCX, -24)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunluk yaz
	m.movYerelOku(rRSI, -8)
	m.leaDolayli(rRSI, rRSI, 8) // kaynak verisi basi
	m.movYerelOku(rRCX, -16)
	m.addKayit(rRSI, rRCX) // + baslangic
	m.movYerelOku(rRDI, -32)
	m.leaDolayli(rRDI, rRDI, 8) // hedef verisi basi
	m.movYerelOku(rRDX, -24)    // kopyalanacak bayt sayisi
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRAX, -32)
	m.leave()
	m.ret()
}

// parçala(rdi = metin, rsi = ayraç) -> rax = liste (ayraca gore bolunmus
// metin parcalari, sol-en-once eslesme — strings.Split ile ayni kural).
// ayraç boşsa (uzunluk 0) harfler() ile ayni davranisa duser (her bayt ayri
// oge). f_liste_ekle ile buyur (dogru sonuc parca sayisi az oldugundan
// O(n^2) maliyeti onemsiz — liste.go'daki listeYap()'in onledigi durum
// BUYUK n icindir, burada n = parca sayisi, tipik olarak kucuk).
func (e *elfUretici) yardimciParcala() {
	m := e.m
	m.etiketKoy("f_parcala")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 64) // -8.. -56 kullanilir, 64'e yuvarlandi
	m.movYerelYaz(-8, rRDI)        // metin
	m.movYerelYaz(-16, rRSI)       // ayrac
	m.movDolayliOku(rRAX, rRSI, 0) // nAyrac
	m.movYerelYaz(-24, rRAX)
	m.testKayit(rRAX, rRAX)
	m.jcc(0x85, "Lpc_normal") // jne — ayrac bos degil, normal yola git
	m.movYerelOku(rRDI, -8)
	m.call("f_harfler") // ayrac bos -> harfler() ile ayni
	m.leave()
	m.ret()
	m.etiketKoy("Lpc_normal")
	m.movYerelOku(rRAX, -8)
	m.movDolayliOku(rRAX, rRAX, 0) // nMetin
	m.movYerelYaz(-32, rRAX)
	m.movImm32(rRDI, 0) // bos liste: n=0
	m.movImm32(rRSI, 0)
	m.call("f_liste_yap")
	m.movYerelYaz(-40, rRAX) // liste (buyuyecek)
	m.movYerelImm(-48, 0)    // parcaBasi
	// i, parcaBasi ile ayni baslar; asagidaki dongude ayrica -56'da tutulur
	m.movYerelImm(-56, 0) // i (arama konumu)
	m.etiketKoy("Lpc_dongu")
	// eger i + nAyrac > nMetin ise arama biter
	m.movYerelOku(rRAX, -56)
	m.movYerelOku(rRCX, -24)
	m.addKayit(rRAX, rRCX)
	m.movYerelOku(rRDX, -32)
	m.cmpKayit(rRAX, rRDX)
	m.jcc(0x8F, "Lpc_arama_bitti") // jg — sigmiyor
	// aday = metin_araligi(metin, i, nAyrac)
	m.movYerelOku(rRDI, -8)
	m.movYerelOku(rRSI, -56)
	m.movYerelOku(rRDX, -24)
	m.call("f_metin_araligi")
	m.movKayit(rRDI, rRAX)
	m.movYerelOku(rRSI, -16)
	m.call("f_metin_esit") // rax = 1 eslesirse
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lpc_ilerle") // jz — eslesmedi, i++
	// eslesti: parca = metin_araligi(metin, parcaBasi, i-parcaBasi); liste'ye ekle
	m.movYerelOku(rRDI, -8)
	m.movYerelOku(rRSI, -48)
	m.movYerelOku(rRDX, -56)
	m.movYerelOku(rRCX, -48)
	m.subKayit(rRDX, rRCX) // uzunluk = i - parcaBasi
	m.call("f_metin_araligi")
	m.movKayit(rRSI, rRAX)
	m.movYerelOku(rRDI, -40)
	m.call("f_liste_ekle")
	m.movYerelYaz(-40, rRAX)
	// i += nAyrac; parcaBasi = i
	m.movYerelOku(rRAX, -56)
	m.movYerelOku(rRCX, -24)
	m.addKayit(rRAX, rRCX)
	m.movYerelYaz(-56, rRAX)
	m.movYerelYaz(-48, rRAX)
	m.jmp("Lpc_dongu")
	m.etiketKoy("Lpc_ilerle")
	m.movYerelOku(rRAX, -56)
	m.incKayit(rRAX)
	m.movYerelYaz(-56, rRAX)
	m.jmp("Lpc_dongu")
	m.etiketKoy("Lpc_arama_bitti")
	// son parca: metin_araligi(metin, parcaBasi, nMetin - parcaBasi)
	m.movYerelOku(rRDI, -8)
	m.movYerelOku(rRSI, -48)
	m.movYerelOku(rRDX, -32)
	m.movYerelOku(rRCX, -48)
	m.subKayit(rRDX, rRCX)
	m.call("f_metin_araligi")
	m.movKayit(rRSI, rRAX)
	m.movYerelOku(rRDI, -40)
	m.call("f_liste_ekle")
	m.leave()
	m.ret()
}

// ============================================================
// SÖZLÜK (dict) çalışma zamanı — ayrı-zincirleme hashtable, SABİT 256 kova.
// Ayırıcı asla realloc/free yapmadığından (brk-tabanlı bump), kova dizisi
// büyümez — çakışmalar zincir (dugum.sonraki) ile çözülür. Bu, çok büyük
// sözlüklerde O(1) yerine O(zincir uzunluğu) demek ama DOĞRU kalır; bu
// derleyicinin genel felsefesiyle aynı ("Correct, not fast" — README).
//
// Bellek düzeni:
//   sözlük  [uzunluk:8][kovaDizisiPtr:8]                    16 bayt, sabit
//   kovalar [kova0:8][kova1:8]...[kova255:8]                 düz dizi, HEADER YOK
//            (leaOge KULLANILMAZ — o liste/metin'in +8 header'ı içindir)
//   düğüm   [anahtar:8(metin)][değer:8(ham)][sonraki:8]       24 bayt
//
// anahtarlar() sırası kova sırasına göredir — yorumlayıcının EKLEME sırası
// GARANTİSİ ile aynı DEĞİLDİR (bilinçli basitleştirme, bkz. dict semantiği
// zaten sıra-bağımsız kullanılan yerlerde — ör. ağırlıklı toplam).
// ============================================================
const sozlukKovaSayisi = 256

// sozluk_hash(rdi = metin) -> rax = kova indeksi (0..255)
func (e *elfUretici) yardimciSozlukHash() {
	m := e.m
	m.etiketKoy("f_sozluk_hash")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)         // metin
	m.movDolayliOku(rRAX, rRDI, 0)  // n
	m.movYerelYaz(-16, rRAX)        // n
	m.movYerelImm(-24, 0)           // h = 0
	m.movYerelImm(-32, 0)           // i = 0
	m.etiketKoy("Lsh_dongu")
	m.movYerelOku(rRAX, -32)
	m.movYerelOku(rRDX, -16)
	m.cmpKayit(rRAX, rRDX)
	m.jcc(0x8D, "Lsh_bitti") // jge
	m.movYerelOku(rRDI, -8)
	m.leaDolayli(rRDI, rRDI, 8)
	m.movYerelOku(rRSI, -32)
	m.movzxBaytDolayli(rRAX, rRDI, rRSI) // rax = bayt[i]
	m.movYerelOku(rRCX, -24)             // rcx = h (eski)
	m.movKayit(rRDX, rRCX)
	m.shlImm(rRDX, 5)    // rdx = h<<5
	m.subKayit(rRDX, rRCX) // rdx = h*31
	m.addKayit(rRDX, rRAX) // rdx = h*31 + bayt[i]
	m.movYerelYaz(-24, rRDX)
	m.movYerelOku(rRAX, -32)
	m.incKayit(rRAX)
	m.movYerelYaz(-32, rRAX)
	m.jmp("Lsh_dongu")
	m.etiketKoy("Lsh_bitti")
	m.movYerelOku(rRAX, -24)
	m.andImm32(rRAX, sozlukKovaSayisi-1)
	m.leave()
	m.ret()
}

// sozluk_yap() -> rax = yeni boş sözlük
func (e *elfUretici) yardimciSozlukYap() {
	m := e.m
	m.etiketKoy("f_sozluk_yap")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 16)
	m.movImm32(rRDI, sozlukKovaSayisi*8)
	m.call("f_tan_ayir")
	m.movYerelYaz(-8, rRAX) // kova dizisi (düz, header yok — brk taze bellek sıfırlanmış gelir)
	m.movImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-16, rRAX) // sözlük
	m.movImm32(rRCX, 0)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunluk = 0
	m.movYerelOku(rRCX, -8)
	m.movDolayliYaz(rRAX, 8, rRCX) // kova dizisi ptr
	m.movYerelOku(rRAX, -16)
	m.leave()
	m.ret()
}

// sozluk_koy(rdi = sözlük, rsi = anahtar[metin], rdx = değer[ham]) -> rax = sözlük
func (e *elfUretici) yardimciSozlukKoy() {
	m := e.m
	m.etiketKoy("f_sozluk_koy")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)  // sözlük
	m.movYerelYaz(-16, rRSI) // anahtar
	m.movYerelYaz(-24, rRDX) // değer
	m.movYerelOku(rRDI, -16)
	m.call("f_sozluk_hash") // rax = kova
	m.movYerelYaz(-32, rRAX)
	m.movYerelOku(rRDI, -8)
	m.movDolayliOku(rRDI, rRDI, 8) // kova dizisi ptr
	m.movYerelOku(rRAX, -32)
	m.shlImm(rRAX, 3)
	m.addKayit(rRDI, rRAX)  // rdi = &kovalar[kova]
	m.movYerelYaz(-40, rRDI) // kova hücresi adresi
	m.movDolayliOku(rRCX, rRDI, 0) // rcx = zincirin baş düğümü
	m.etiketKoy("Lkoy_ara")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lkoy_yeni") // jz
	m.movYerelYaz(-48, rRCX) // düğüm ptr
	m.movDolayliOku(rRDI, rRCX, 0) // düğüm.anahtar
	m.movYerelOku(rRSI, -16)
	m.call("f_metin_esit")
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lkoy_sonraki") // jz
	// eşleşti: değeri güncelle
	m.movYerelOku(rRAX, -48)
	m.movYerelOku(rRCX, -24)
	m.movDolayliYaz(rRAX, 8, rRCX)
	m.movYerelOku(rRAX, -8)
	m.leave()
	m.ret()
	m.etiketKoy("Lkoy_sonraki")
	m.movYerelOku(rRCX, -48)
	m.movDolayliOku(rRCX, rRCX, 16) // düğüm.sonraki
	m.jmp("Lkoy_ara")
	m.etiketKoy("Lkoy_yeni")
	m.movImm32(rRDI, 24)
	m.call("f_tan_ayir")
	m.movYerelOku(rRCX, -16)
	m.movDolayliYaz(rRAX, 0, rRCX) // anahtar
	m.movYerelOku(rRCX, -24)
	m.movDolayliYaz(rRAX, 8, rRCX) // değer
	m.movYerelOku(rRDI, -40)
	m.movDolayliOku(rRCX, rRDI, 0) // eski baş (0 olabilir)
	m.movDolayliYaz(rRAX, 16, rRCX) // yeni.sonraki = eski baş
	m.movDolayliYaz(rRDI, 0, rRAX)  // kova hücresi = yeni düğüm
	m.movYerelOku(rRAX, -8)
	m.movDolayliOku(rRCX, rRAX, 0)
	m.incKayit(rRCX)
	m.movDolayliYaz(rRAX, 0, rRCX) // uzunluk++
	m.movYerelOku(rRAX, -8)
	m.leave()
	m.ret()
}

// sozluk_al(rdi = sözlük, rsi = anahtar[metin]) -> rax = değer (yoksa 0)
func (e *elfUretici) yardimciSozlukAl() {
	m := e.m
	m.etiketKoy("f_sozluk_al")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)
	m.movYerelYaz(-16, rRSI)
	m.movYerelOku(rRDI, -16)
	m.call("f_sozluk_hash")
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRDI, -8)
	m.movDolayliOku(rRDI, rRDI, 8)
	m.movYerelOku(rRAX, -24)
	m.shlImm(rRAX, 3)
	m.addKayit(rRDI, rRAX)
	m.movDolayliOku(rRCX, rRDI, 0)
	m.etiketKoy("Lal_ara")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lal_yok")
	m.movYerelYaz(-32, rRCX)
	m.movDolayliOku(rRDI, rRCX, 0)
	m.movYerelOku(rRSI, -16)
	m.call("f_metin_esit")
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lal_sonraki")
	m.movYerelOku(rRAX, -32)
	m.movDolayliOku(rRAX, rRAX, 8)
	m.leave()
	m.ret()
	m.etiketKoy("Lal_sonraki")
	m.movYerelOku(rRCX, -32)
	m.movDolayliOku(rRCX, rRCX, 16)
	m.jmp("Lal_ara")
	m.etiketKoy("Lal_yok")
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}

// sozluk_varmi(rdi = sözlük, rsi = anahtar[metin]) -> rax = 1/0
func (e *elfUretici) yardimciSozlukVarmi() {
	m := e.m
	m.etiketKoy("f_sozluk_varmi")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)
	m.movYerelYaz(-16, rRSI)
	m.movYerelOku(rRDI, -16)
	m.call("f_sozluk_hash")
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRDI, -8)
	m.movDolayliOku(rRDI, rRDI, 8)
	m.movYerelOku(rRAX, -24)
	m.shlImm(rRAX, 3)
	m.addKayit(rRDI, rRAX)
	m.movDolayliOku(rRCX, rRDI, 0)
	m.etiketKoy("Lvm_ara")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lvm_yok")
	m.movYerelYaz(-32, rRCX)
	m.movDolayliOku(rRDI, rRCX, 0)
	m.movYerelOku(rRSI, -16)
	m.call("f_metin_esit")
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lvm_sonraki")
	m.movImm32(rRAX, 1)
	m.leave()
	m.ret()
	m.etiketKoy("Lvm_sonraki")
	m.movYerelOku(rRCX, -32)
	m.movDolayliOku(rRCX, rRCX, 16)
	m.jmp("Lvm_ara")
	m.etiketKoy("Lvm_yok")
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}

// sozluk_anahtarlar(rdi = sözlük) -> rax = liste (anahtar metinleri, kova sırasında)
// sozluk_sil(rdi = sözlük, rsi = anahtar[metin]) -> rax = 1 bulunup
// silindiyse, 0 yoksa. Zincirden UNLINK eder (prev.sonraki = mevcut.sonraki,
// ya da ilk düğümdeyse kova hücresi güncellenir), uzunluk--. Bu olmadan
// KVDeposu.tan gibi modüllerde "sil" sadece değeri boşaltabiliyordu — anahtar
// sözlükte KALMAYA devam ediyordu, varMı() hâlâ 1 dönüyordu (KVDeposu.tan
// yazılırken bulunan gerçek bug — sözlüğe hiç sil() eklenmemişti).
func (e *elfUretici) yardimciSozlukSil() {
	m := e.m
	m.etiketKoy("f_sozluk_sil")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)  // sözlük
	m.movYerelYaz(-16, rRSI) // anahtar
	m.movYerelOku(rRDI, -16)
	m.call("f_sozluk_hash")
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRDI, -8)
	m.movDolayliOku(rRDI, rRDI, 8)
	m.movYerelOku(rRAX, -24)
	m.shlImm(rRAX, 3)
	m.addKayit(rRDI, rRAX)
	m.movYerelYaz(-32, rRDI)       // kova hücresi adresi
	m.movDolayliOku(rRCX, rRDI, 0) // mevcut = baş düğüm
	m.movYerelImm(-40, 0)          // önceki = yok (0)
	m.etiketKoy("Lsil_ara")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lsil_yok") // jz
	m.movYerelYaz(-48, rRCX)
	m.movDolayliOku(rRDI, rRCX, 0) // mevcut.anahtar
	m.movYerelOku(rRSI, -16)
	m.call("f_metin_esit")
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lsil_sonraki") // jz
	// bulundu — unlink
	m.movYerelOku(rRCX, -48)
	m.movDolayliOku(rRDX, rRCX, 16) // mevcut.sonraki
	m.movYerelOku(rRAX, -40)
	m.testKayit(rRAX, rRAX)
	m.jcc(0x84, "Lsil_ilkti") // jz
	m.movDolayliYaz(rRAX, 16, rRDX) // önceki.sonraki = mevcut.sonraki
	m.jmp("Lsil_uzunluk")
	m.etiketKoy("Lsil_ilkti")
	m.movYerelOku(rRAX, -32)
	m.movDolayliYaz(rRAX, 0, rRDX) // kova hücresi = mevcut.sonraki
	m.etiketKoy("Lsil_uzunluk")
	m.movYerelOku(rRAX, -8)
	m.movDolayliOku(rRCX, rRAX, 0)
	m.decKayit(rRCX)
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.movImm32(rRAX, 1)
	m.leave()
	m.ret()
	m.etiketKoy("Lsil_sonraki")
	m.movYerelOku(rRAX, -48)
	m.movYerelYaz(-40, rRAX) // önceki = mevcut
	m.movDolayliOku(rRCX, rRAX, 16)
	m.jmp("Lsil_ara")
	m.etiketKoy("Lsil_yok")
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}

func (e *elfUretici) yardimciSozlukAnahtarlar() {
	m := e.m
	m.etiketKoy("f_sozluk_anahtarlar")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 64)
	m.movYerelYaz(-8, rRDI)        // sözlük
	m.movDolayliOku(rRAX, rRDI, 0) // n (eleman sayısı)
	m.movYerelYaz(-16, rRAX)
	m.movKayit(rRCX, rRAX)
	m.shlImm(rRCX, 3)
	m.addImm32(rRCX, 8)
	m.movKayit(rRDI, rRCX)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX) // liste
	m.movYerelOku(rRCX, -16)
	m.movDolayliYaz(rRAX, 0, rRCX) // liste uzunluğu = n
	m.movYerelOku(rRDI, -8)
	m.movDolayliOku(rRDI, rRDI, 8) // kova dizisi ptr
	m.movYerelYaz(-32, rRDI)
	m.movYerelImm(-40, 0) // kova = 0
	m.movYerelImm(-48, 0) // yazPos = 0
	m.etiketKoy("Lak_kova_dongu")
	m.movYerelOku(rRAX, -40)
	m.cmpImm32(rRAX, sozlukKovaSayisi)
	m.jcc(0x8D, "Lak_bitti") // jge
	m.movYerelOku(rRAX, -40)
	m.shlImm(rRAX, 3)
	m.movYerelOku(rRDI, -32)
	m.addKayit(rRDI, rRAX)         // rdi = &kovalar[kova]
	m.movDolayliOku(rRCX, rRDI, 0) // rcx = bu kovanin bas dugumu
	m.etiketKoy("Lak_dugum_dongu")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lak_kova_sonraki") // jz
	m.movYerelYaz(-56, rRCX)        // düğüm ptr
	m.movYerelOku(rRAX, -24)        // liste ptr
	m.movYerelOku(rRDX, -48)        // yazPos
	m.leaOge(rRAX, rRAX, rRDX)      // liste + yazPos*8 + 8
	m.movYerelOku(rRCX, -56)
	m.movDolayliOku(rRCX, rRCX, 0) // düğüm.anahtar
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.movYerelOku(rRAX, -48)
	m.incKayit(rRAX)
	m.movYerelYaz(-48, rRAX) // yazPos++
	m.movYerelOku(rRCX, -56)
	m.movDolayliOku(rRCX, rRCX, 16) // düğüm.sonraki
	m.jmp("Lak_dugum_dongu")
	m.etiketKoy("Lak_kova_sonraki")
	m.movYerelOku(rRAX, -40)
	m.incKayit(rRAX)
	m.movYerelYaz(-40, rRAX)
	m.jmp("Lak_kova_dongu")
	m.etiketKoy("Lak_bitti")
	m.movYerelOku(rRAX, -24)
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

// dosya_var_mi(rdi = yol) -> rax = 1/0 — oku() ONCESI kontrol icin (oku()
// eksik dosyada hicbir hata denetimi yapmadan cop uzunluk uretir, dene/
// yakala elf'te yok, bu yuzden CAGRI TARAFI once bunu sormali).
// sayi(rdi = metin) -> rax = tam sayı (int64).
// KAPSAM: sadece TAM SAYI ayrıştırır (isteğe bağlı önde '-', sonra rakamlar,
// ilk rakam-olmayanda durur). Yorumlayıcının sayı()'sı nokta/üs varsa
// ONDALIK döndürür (dinamik tip) — elf statik tip derlediğinden bir CagriDugum
// tek bir sabit dönüş tipine sahip olmalı; bu yüzden burada BİLEREK sadece
// tam sayı yolu var (WAL/log ayrıştırma gibi kullanımların ihtiyacı bu).
// Ondalık metin ayrıştırma ayrı bir işlev (ör. "kesirSayi") olarak eklenebilir.
func (e *elfUretici) yardimciSayi() {
	m := e.m
	m.etiketKoy("f_sayi")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)
	m.movDolayliOku(rRAX, rRDI, 0)
	m.movYerelYaz(-16, rRAX) // n
	m.movYerelImm(-24, 0)    // sonuç
	m.movYerelImm(-32, 0)    // i
	m.movYerelImm(-40, 1)    // işaret

	m.movYerelOku(rRAX, -16)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x84, "Lsy_dongu") // jz — boş metin, döngü zaten çalışmaz
	m.movYerelOku(rRDI, -8)
	m.leaDolayli(rRDI, rRDI, 8)
	m.xorKayit(rRDX, rRDX)
	m.movzxBaytDolayli(rRCX, rRDI, rRDX) // ilk bayt
	m.cmpImm32(rRCX, 45)                 // '-'
	m.jcc(0x85, "Lsy_dongu")             // jne
	m.movYerelImm(-40, -1)
	m.movYerelImm(-32, 1)

	m.etiketKoy("Lsy_dongu")
	m.movYerelOku(rRAX, -32)
	m.movYerelOku(rRDX, -16)
	m.cmpKayit(rRAX, rRDX)
	m.jcc(0x8D, "Lsy_bitti") // jge
	m.movYerelOku(rRDI, -8)
	m.leaDolayli(rRDI, rRDI, 8)
	m.movYerelOku(rRDX, -32)
	m.movzxBaytDolayli(rRCX, rRDI, rRDX)
	m.cmpImm32(rRCX, 48)
	m.jcc(0x8C, "Lsy_bitti") // jl — '0' altı, rakam değil
	m.cmpImm32(rRCX, 57)
	m.jcc(0x8F, "Lsy_bitti") // jg — '9' üstü, rakam değil
	m.subImm32(rRCX, 48)
	m.movYerelOku(rRAX, -24)
	m.movImm32(rRDX, 10)
	m.imulKayit(rRAX, rRDX)
	m.addKayit(rRAX, rRCX)
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRAX, -32)
	m.incKayit(rRAX)
	m.movYerelYaz(-32, rRAX)
	m.jmp("Lsy_dongu")
	m.etiketKoy("Lsy_bitti")
	m.movYerelOku(rRAX, -24)
	m.movYerelOku(rRCX, -40)
	m.imulKayit(rRAX, rRCX)
	m.leave()
	m.ret()
}

func (e *elfUretici) yardimciDosyaVarMi() {
	m := e.m
	m.etiketKoy("f_dosya_var_mi")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
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
	m.movYerelOku(rRDI, -24)
	m.xorKayit(rRSI, rRSI)
	m.xorKayit(rRDX, rRDX)
	m.movImm32(rRAX, 2) // sys_open (O_RDONLY)
	m.syscall()
	m.movYerelYaz(-32, rRAX) // fd (negatifse hata)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x8C, "Ldvm_yok") // jl
	m.movYerelOku(rRDI, -32)
	m.movImm32(rRAX, 3) // close
	m.syscall()
	m.movImm32(rRAX, 1)
	m.leave()
	m.ret()
	m.etiketKoy("Ldvm_yok")
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}

// yaz_dosya(rdi = yol, rsi = icerik) -> rax = 0
// open(yol, O_WRONLY|O_CREAT|O_TRUNC=577, 0755)

// yazBaytlar(rdi = yol metni, rsi = bayt listesi) -> rax = 0
// Listenin her ogesinin DUSUK BAYTINI ham olarak dosyaya yazar.
// metin() yolundan farki: UTF-8 kodlamasi YAPILMAZ, 200 -> tek bayt 200.
func (e *elfUretici) yardimciYazBaytlar() {
	m := e.m
	m.etiketKoy("f_yaz_baytlar")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 64)
	m.movYerelYaz(-8, rRDI)  // yol
	m.movYerelYaz(-16, rRSI) // liste
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
	// liste ogelerini bayt tamponuna sikistir
	m.movYerelOku(rRSI, -16)
	m.movDolayliOku(rRAX, rRSI, 0) // uzunluk
	m.movYerelYaz(-40, rRAX)
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-48, rRAX) // bayt tamponu
	m.xorKayit(rRCX, rRCX)
	m.etiketKoy("Lyb_dongu")
	m.movYerelOku(rRDX, -40)
	m.cmpKayit(rRCX, rRDX)
	m.jcc(0x8D, "Lyb_bitti") // jge
	m.movYerelOku(rRSI, -16)
	m.leaOge(rRSI, rRSI, rRCX)
	m.movDolayliOku(rRDX, rRSI, 0) // oge degeri
	m.movYerelOku(rRDI, -48)
	m.addKayit(rRDI, rRCX)
	m.movBaytKayit(rRDI, rRDX) // mov [rdi], dl
	m.incKayit(rRCX)
	m.jmp("Lyb_dongu")
	m.etiketKoy("Lyb_bitti")
	// open(yol, O_WRONLY|O_CREAT|O_TRUNC, 0755)
	m.movYerelOku(rRDI, -32)
	m.movImm32(rRSI, 577)
	m.movImm32(rRDX, 493)
	m.movImm32(rRAX, 2)
	m.syscall()
	m.movYerelYaz(-56, rRAX)
	// write(fd, tampon, uzunluk)
	m.movYerelOku(rRDI, -56)
	m.movYerelOku(rRSI, -48)
	m.movYerelOku(rRDX, -40)
	m.movImm32(rRAX, 1)
	m.syscall()
	// close
	m.movYerelOku(rRDI, -56)
	m.movImm32(rRAX, 3)
	m.syscall()
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
}


// ---------- PROGRAM ARGUMANLARI (libc yok, ham stack) ----------
// _start'ta rsp -> [argc][argv0][argv1]...  Bu tabani saklariz.
func (e *elfUretici) argvYakala() {
	m := e.m
	e.genel["__argv"] = true
	m.movGenelYaz("v___argv", rRSP)
}

// argsay() -> rax = argc
func (e *elfUretici) yardimciArgsay() {
	m := e.m
	m.etiketKoy("f_argsay")
	e.genel["__argv"] = true
	m.movGenelOku(rRAX, "v___argv")
	m.movDolayliOku(rRAX, rRAX, 0)
	m.ret()
}

// arg(rdi = i) -> rax = i'inci arguman (Tan metni)
// argv[i] NUL sonlu C metnidir; uzunlugu tarayip yigina kopyalariz.
func (e *elfUretici) yardimciArg() {
	m := e.m
	m.etiketKoy("f_arg")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	e.genel["__argv"] = true
	m.movGenelOku(rRAX, "v___argv")
	// leaOge = taban + indeks*8 + 8 ; +8 zaten argc'yi atliyor,
	// yani arg(1) dogrudan argv[1]'e denk gelir.
	m.leaOge(rRAX, rRAX, rRDI)
	m.movDolayliOku(rRSI, rRAX, 0) // rsi = C metni
	m.movYerelYaz(-8, rRSI)
	// uzunlugu tara
	m.xorKayit(rRCX, rRCX)
	m.etiketKoy("Larg_tara")
	m.movYerelOku(rRSI, -8)
	m.addKayit(rRSI, rRCX)
	m.movBaytOku(rRSI) // al = [rsi]
	m.movzx(rRAX, rRAX)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x84, "Larg_bitti") // je
	m.incKayit(rRCX)
	m.jmp("Larg_tara")
	m.etiketKoy("Larg_bitti")
	m.movYerelYaz(-16, rRCX) // uzunluk
	m.movKayit(rRDI, rRCX)
	m.addImm32(rRDI, 16)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRCX, -16)
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.leaDolayli(rRDI, rRAX, 8)
	m.movYerelOku(rRSI, -8)
	m.movYerelOku(rRDX, -16)
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRAX, -24)
	m.leave()
	m.ret()
}

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

// ekle_dosya(rdi = yol, rsi = içerik) — dosyanın SONUNA ekler (append),
// yaz_dosya ile AYNI iskelet, tek fark O_TRUNC yerine O_APPEND. Log/WAL
// tarzı depolama (B+Tree gibi) icin gerekli — TAN'da rastgele-erisimli
// dosya seek/pwrite YOK, bu yuzden "gercek" disk motorlari EKLEME-ONCELIKLI
// (append-only log + baslangicta tam okuma ile bellek-ici indeks yeniden
// kurma) tasarlanmali.
func (e *elfUretici) yardimciEkleDosya() {
	m := e.m
	m.etiketKoy("f_ekle_dosya")
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
	m.movImm32(rRSI, 1089) // O_WRONLY|O_CREAT|O_APPEND
	m.movImm32(rRDX, 493)  // 0755
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

// yardimciPositionalIO — RASTGELE ERİŞİMLİ (positional) dosya G/Ç rutinleri.
// oku()/yaz_dosya() TÜM dosyayı işler; bunlar bir DOSYA TANITICISI (fd)
// üzerinden çalışır: aç -> konumla(lseek) -> oku/yaz -> senkron(fsync) ->
// kapat. Sayfa-tabanlı depolama motorunun (B+Tree, buffer pool) tabanı.
// Not: aç bayrakları makine kodunda SABİT (dallanma yok) — güvenli codegen.
func (e *elfUretici) yardimciPositionalIO() {
	m := e.m

	// --- aç: ortak C-string kur + open(path, BAYRAK, mode) ---
	// f_dosya_ac_okuyaz(rdi=yol metin) -> rax=fd   [O_RDWR|O_CREAT=66]
	m.etiketKoy("f_dosya_ac_okuyaz")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
	m.movYerelYaz(-8, rRDI)         // yol pointer
	m.movDolayliOku(rRAX, rRDI, 0)  // uzunluk = *yol
	m.movYerelYaz(-16, rRAX)
	m.movKayit(rRDI, rRAX)
	m.addImm32(rRDI, 16)            // uzunluk+16 (tan_ayir sıfırlar -> C-string null'lu)
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX)        // C-string buffer
	m.movKayit(rRDI, rRAX)
	m.movYerelOku(rRSI, -8)
	m.leaDolayli(rRSI, rRSI, 8)     // yol+8 (veri başı)
	m.movYerelOku(rRDX, -16)
	m.call("f_bellek_kopyala")
	m.movYerelOku(rRDI, -24)        // path
	m.movImm32(rRSI, 66)           // O_RDWR|O_CREAT
	m.movImm32(rRDX, 420)          // 0644
	m.movImm32(rRAX, 2)            // sys_open
	m.syscall()
	m.leave()
	m.ret()

	// f_dosya_ac_oku(rdi=yol) -> rax=fd   [O_RDONLY=0]
	m.etiketKoy("f_dosya_ac_oku")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
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
	m.movYerelOku(rRDI, -24)
	m.xorKayit(rRSI, rRSI)         // O_RDONLY
	m.xorKayit(rRDX, rRDX)         // mode gereksiz
	m.movImm32(rRAX, 2)
	m.syscall()
	m.leave()
	m.ret()

	// f_dosya_ac_yaz(rdi=yol) -> rax=fd   [O_WRONLY|O_CREAT|O_TRUNC=577]
	m.etiketKoy("f_dosya_ac_yaz")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 48)
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
	m.movYerelOku(rRDI, -24)
	m.movImm32(rRSI, 577)         // O_WRONLY|O_CREAT|O_TRUNC
	m.movImm32(rRDX, 420)         // 0644
	m.movImm32(rRAX, 2)
	m.syscall()
	m.leave()
	m.ret()

	// f_dosya_konumla(rdi=fd, rsi=ofset) -> rax=yeni konum   [lseek, whence=SEEK_SET]
	m.etiketKoy("f_dosya_konumla")
	m.xorKayit(rRDX, rRDX)        // whence = SEEK_SET (0)
	m.movImm32(rRAX, 8)          // sys_lseek
	m.syscall()
	m.ret()

	// f_dosya_oku_blok(rdi=fd, rsi=uzunluk) -> rax=metin   [read]
	m.etiketKoy("f_dosya_oku_blok")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI)       // fd
	m.movYerelYaz(-16, rRSI)      // istenen uzunluk
	m.movKayit(rRDI, rRSI)
	m.addImm32(rRDI, 16)          // buffer = uzunluk+16
	m.call("f_tan_ayir")
	m.movYerelYaz(-24, rRAX)      // metin buffer
	m.movYerelOku(rRDI, -8)       // fd
	m.movYerelOku(rRSI, -24)
	m.leaDolayli(rRSI, rRSI, 8)   // buf = buffer+8
	m.movYerelOku(rRDX, -16)      // count
	m.xorKayit(rRAX, rRAX)        // sys_read (0)
	m.syscall()
	m.movYerelOku(rRCX, -24)
	m.movDolayliYaz(rRCX, 0, rRAX) // *buffer = gerçek okunan bayt (metin uzunluğu)
	m.movYerelOku(rRAX, -24)      // dönüş = metin
	m.leave()
	m.ret()

	// f_dosya_yaz_blok(rdi=fd, rsi=metin) -> rax=yazılan bayt   [write]
	m.etiketKoy("f_dosya_yaz_blok")
	m.movDolayliOku(rRDX, rRSI, 0) // count = *metin (uzunluk)
	m.leaDolayli(rRSI, rRSI, 8)    // buf = metin+8
	m.movImm32(rRAX, 1)           // sys_write
	m.syscall()
	m.ret()

	// f_dosya_oku_blok_ham(rdi=fd, rsi=adres, rdx=uzunluk) -> rax=okunan bayt
	// [read] — dosyaOkuBlok'tan farkı: metin AYIRMAZ, DOĞRUDAN verilen ham
	// adrese (bellekEsle) okur. Linux read(2) ABI'si zaten rdi/rsi/rdx —
	// argümanlar HİÇ taşınmadan doğrudan syscall'a geçiyor.
	m.etiketKoy("f_dosya_oku_blok_ham")
	m.xorKayit(rRAX, rRAX) // sys_read (0)
	m.syscall()
	m.ret()

	// f_dosya_yaz_blok_ham(rdi=fd, rsi=adres, rdx=uzunluk) -> rax=yazılan bayt
	// [write] — dosyaYazBlok'tan farkı: metin BEKLEMEZ, ham adresten
	// DOĞRUDAN yazar.
	m.etiketKoy("f_dosya_yaz_blok_ham")
	m.movImm32(rRAX, 1) // sys_write
	m.syscall()
	m.ret()

	// f_dosya_senkron(rdi=fd) -> rax=0   [fsync]
	m.etiketKoy("f_dosya_senkron")
	m.movImm32(rRAX, 74)          // sys_fsync
	m.syscall()
	m.xorKayit(rRAX, rRAX)
	m.ret()

	// f_dosya_kapat(rdi=fd) -> rax=0   [close]
	m.etiketKoy("f_dosya_kapat")
	m.movImm32(rRAX, 3)           // sys_close
	m.syscall()
	m.xorKayit(rRAX, rRAX)
	m.ret()

	// f_dosya_sil(rdi=yol metin) -> rax=0   [unlink]
	m.etiketKoy("f_dosya_sil")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
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
	m.movYerelOku(rRDI, -24)      // C-string yol
	m.movImm32(rRAX, 87)         // sys_unlink
	m.syscall()
	m.movImm32(rRAX, 0)          // 0 döndür (yorumlayıcı ile uyumlu)
	m.leave()
	m.ret()

	// f_ham_oku8(rdi=adres) -> rax = *(int64*)adres
	m.etiketKoy("f_ham_oku8")
	m.movDolayliOku(rRAX, rRDI, 0)
	m.ret()

	// f_ham_yaz8(rdi=adres, rsi=deger) -> rax=0 : *(int64*)adres = deger
	m.etiketKoy("f_ham_yaz8")
	m.movDolayliYaz(rRDI, 0, rRSI)
	m.movImm32(rRAX, 0)
	m.ret()

	// f_bellek_esle(rdi=boyut) -> rax=adres : anonim mmap
	// mmap(NULL, boyut, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANONYMOUS, -1, 0)
	m.etiketKoy("f_bellek_esle")
	m.movKayit(rRSI, rRDI)       // rsi = len = boyut
	m.movImm32(rRDI, 0)          // rdi = addr = NULL
	m.movImm32(rRDX, 3)          // rdx = prot = PROT_READ|PROT_WRITE
	m.movImm32(rR10, 0x22)       // r10 = flags = MAP_PRIVATE|MAP_ANONYMOUS
	m.movImm32(rR8, -1)          // r8 = fd = -1 (işaretli genişletme)
	m.movImm32(rR9, 0)           // r9 = offset = 0
	m.movImm32(rRAX, 9)          // sys_mmap
	m.syscall()
	m.ret()

	// f_bellek_coz(rdi=adres, rsi=boyut) -> rax=0 : munmap
	m.etiketKoy("f_bellek_coz")
	m.movImm32(rRAX, 11)         // sys_munmap
	m.syscall()
	m.movImm32(rRAX, 0)
	m.ret()

	// f_ham_oku4(rdi=adres) -> rax = *(uint32*)adres (32-bit oku, üst bitler sıfır)
	m.etiketKoy("f_ham_oku4")
	m.movDolayli32Oku(rRAX, rRDI, 0)
	m.ret()

	// f_ham_yaz4(rdi=adres, rsi=deger) -> rax=0 : *(uint32*)adres = deger (alt 32 bit)
	m.etiketKoy("f_ham_yaz4")
	m.movDolayli32Yaz(rRDI, 0, rRSI)
	m.movImm32(rRAX, 0)
	m.ret()

	// f_ham_oku_bayt(rdi=adres) -> rax = *(uint8*)adres (üst bitler sıfır)
	m.etiketKoy("f_ham_oku_bayt")
	m.xorKayit(rRAX, rRAX)
	m.movBaytOku(rRDI)
	m.ret()

	// f_ham_yaz_bayt(rdi=adres, rsi=deger) -> rax=0 : *(uint8*)adres = deger (alt bayt)
	// movBaytKayit kaynak kaydı DL/CL/AL/BL ile sınırlı (REX olmadan yüksek
	// bayt kodlamasına düşer) — deger önce rdx'e taşınır (kod tabanında
	// yerleşik desen, bkz. yardimciMetinIndeks).
	m.etiketKoy("f_ham_yaz_bayt")
	m.movKayit(rRDX, rRSI)
	m.movBaytKayit(rRDI, rRDX)
	m.movImm32(rRAX, 0)
	m.ret()

	// f_ham_bellek_tasi(rdi=hedef, rsi=kaynak, rdx=uzunluk) -> rax=0 : memmove
	// (çakışma-güvenli — hedef>kaynak ise SONDAN BAŞA, aksi TAKDİRDE BAŞTAN
	// SONA kopyalar; f_bellek_kopyala'dan farkı budur, o yalnız ileri kopyalar
	// ve içeride TAZE ayrılan, çakışmayan bloklar için kullanılır).
	m.etiketKoy("f_ham_bellek_tasi")
	m.testKayit(rRDX, rRDX)
	m.jcc(0x84, "Lhbt_son") // jz
	m.cmpKayit(rRDI, rRSI)
	m.jcc(0x86, "Lhbt_ileri") // jbe: hedef<=kaynak -> ileri kopya güvenli
	// hedef > kaynak: sondan başa kopyala (çakışma güvenli)
	m.addKayit(rRDI, rRDX)
	m.addKayit(rRSI, rRDX)
	m.decKayit(rRDI)
	m.decKayit(rRSI)
	m.etiketKoy("Lhbt_geri")
	m.movBaytOku(rRSI)
	m.movBaytYazAl(rRDI)
	m.decKayit(rRSI)
	m.decKayit(rRDI)
	m.decKayit(rRDX)
	m.jcc(0x85, "Lhbt_geri") // jnz
	m.jmp("Lhbt_son")
	m.etiketKoy("Lhbt_ileri")
	m.movBaytOku(rRSI)
	m.movBaytYazAl(rRDI)
	m.incKayit(rRSI)
	m.incKayit(rRDI)
	m.decKayit(rRDX)
	m.jcc(0x85, "Lhbt_ileri") // jnz
	m.etiketKoy("Lhbt_son")
	m.movImm32(rRAX, 0)
	m.ret()

	// f_bellek_doldur(rdi=adres, rsi=deger, rdx=uzunluk) -> rax=0 : memset
	// (uzunluk baytı deger'in alt baytıyla doldurur)
	m.etiketKoy("f_bellek_doldur")
	m.movKayit(rRCX, rRSI) // cl = deger (alt bayt)
	m.testKayit(rRDX, rRDX)
	m.jcc(0x84, "Lbd_son") // jz
	m.etiketKoy("Lbd_dongu")
	m.movBaytKayit(rRDI, rRCX)
	m.incKayit(rRDI)
	m.decKayit(rRDX)
	m.jcc(0x85, "Lbd_dongu") // jnz
	m.etiketKoy("Lbd_son")
	m.movImm32(rRAX, 0)
	m.ret()
}

// yardimciEszamanlilik — ADIM 4: native iş parçacığı (clone) + kilit/futex +
// atomik. içParcaLat'ın KENDİ (inline) codegen'i ifade()'de; burada SADECE
// paylaşılan, sabit rutinler var (trambolin + futex sarmalayıcıları + kilit
// + atomik). Fonksiyon-DEĞERİ native ELF'te yok (bilinen ayrı boşluk) —
// içParcaLat bu yüzden DERLEME-ZAMANI bilinen bir işlev ADI alır, çalışma-
// zamanı kapatma/closure DEĞİL.
func (e *elfUretici) yardimciEszamanlilik() {
	m := e.m

	// f_futex_wait(rdi=adres, rsi=beklenenDeger) -> rax=syscall sonucu.
	// [adres] HÂLÂ beklenenDeger İSE uyur (kernel atomik kontrol eder —
	// kaçırılan uyandırma YOK); değilse hemen döner (EAGAIN, çağıran
	// tekrar kontrol eder).
	m.etiketKoy("f_futex_wait")
	m.movKayit(rRDX, rRSI) // rdx = val (beklenen değer)
	m.movImm32(rRSI, 0)    // rsi = FUTEX_WAIT
	m.xorKayit(rR10, rR10) // timeout = NULL
	m.movImm32(rRAX, 202)  // sys_futex
	m.syscall()
	m.ret()

	// f_futex_wake(rdi=adres, rsi=uyandırılacakSayı) -> rax=syscall sonucu.
	m.etiketKoy("f_futex_wake")
	m.movKayit(rRDX, rRSI) // rdx = sayı
	m.movImm32(rRSI, 1)    // rsi = FUTEX_WAKE
	m.movImm32(rRAX, 202)
	m.syscall()
	m.ret()

	// f_iplik_cikis: içParcaLat'ın çocuk yığınına elle yerleştirdiği "dönüş
	// adresi" — f_<ad>'ın kendi leave/ret'i BURAYA düşer (call DEĞİL jmp ile
	// girildiği için, ret normalde çağıranın adresine dönerdi; biz o
	// adresi elle bu etikete ayarladık). r13 = iplik-blok adresi (İÇPARCALAT
	// tarafından f_<ad>'a atlanmadan HEMEN ÖNCE ayarlandı, aradaki HİÇBİR
	// kod r12-r15'e dokunmuyor).
	m.etiketKoy("f_iplik_cikis")
	m.movDolayliYaz(rR13, 8, rRAX) // [blok+8] = sonuç (f_<ad>'ın rax'ı)
	m.movImm32(rRCX, 1)
	m.movDolayliYaz(rR13, 0, rRCX) // [blok+0] = 1 (bitti bayrağı)
	m.movKayit(rRDI, rR13)         // futex_wake(blok+0, 1) — blok+0 = bitti bayrağının KENDİSİ
	m.movImm32(rRSI, 1)
	m.call("f_futex_wake")
	m.movImm32(rRAX, 60) // sys_exit (exit_group DEĞİL — sadece BU iplik ölür)
	m.xorKayit(rRDI, rRDI)
	m.syscall()

	// f_iplik_bekle(rdi=blokAdresi) -> rax=sonuç. Bitti bayrağı set olana
	// kadar futex_wait ile uyu-kontrol-et döngüsü (standart futex-as-
	// condvar deseni — kaçırılan uyandırmaya karşı GÜVENLİ: kernel WAIT'te
	// değeri atomik kontrol eder, çoktan değişmişse hemen döner).
	m.etiketKoy("f_iplik_bekle")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.etiketKoy("Libk_dongu")
	m.movDolayliOku(rRAX, rRDI, 0) // rax = [blok+0] (bitti bayrağı)
	m.testKayit(rRAX, rRAX)
	m.jcc(0x85, "Libk_bitti") // jnz
	m.pushKayit(rRDI)
	m.xorKayit(rRSI, rRSI) // beklenen = 0 (henüz bitmedi)
	m.call("f_futex_wait")
	m.popKayit(rRDI)
	m.jmp("Libk_dongu")
	m.etiketKoy("Libk_bitti")
	m.movDolayliOku(rRAX, rRDI, 8) // rax = [blok+8] (sonuç)
	m.leave()
	m.ret()

	// f_kilit_al(rdi=kilitAdresi) -> rax=0. lock cmpxchg ile CAS(0->1);
	// başarısızsa GÜNCEL değeri (rax, genelde 1) bekleyerek futex_wait,
	// tekrar dene.
	m.etiketKoy("f_kilit_al")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.etiketKoy("Lkal_dongu")
	m.xorKayit(rRAX, rRAX) // beklenen = 0 (kilitli-değil)
	m.movImm32(rRCX, 1)    // yeni = 1 (kilitli)
	m.lockCmpxchgBellek(rRDI, rRCX)
	m.jcc(0x84, "Lkal_alindi") // je (ZF=1 -> CAS başarılı)
	m.pushKayit(rRDI)
	m.movKayit(rRSI, rRAX) // beklenen = GÜNCEL [rdi] değeri (cmpxchg rax'a yazdı)
	m.call("f_futex_wait")
	m.popKayit(rRDI)
	m.jmp("Lkal_dongu")
	m.etiketKoy("Lkal_alindi")
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()

	// f_kilit_birak(rdi=kilitAdresi) -> rax=0. [rdi]=0, futex_wake(1).
	m.etiketKoy("f_kilit_birak")
	m.xorKayit(rRCX, rRCX)
	m.movDolayliYaz(rRDI, 0, rRCX) // [rdi] = 0 (kilitli-değil)
	m.movImm32(rRSI, 1)
	m.call("f_futex_wake") // rdi zaten kilit adresi
	m.movImm32(rRAX, 0)
	m.ret()

	// f_atomik_ekle_ham(rdi=adres, rsi=miktar) -> rax=YENİ değer. lock xadd
	// ile atomik fetch-and-add.
	m.etiketKoy("f_atomik_ekle_ham")
	m.movKayit(rRAX, rRSI)  // rax = miktar (xadd kaynağı)
	m.lockXaddBellek(rRDI, rRAX) // [rdi]+=rax (atomik); rax = ESKİ değer
	m.addKayit(rRAX, rRSI)  // rax = eski + miktar = YENİ değer
	m.ret()
}

// yuvarla(rdi = sayı[ondalık ham], rsi = basamak[int64]) -> rax = ondalık ham
// SSE4.1 roundsd YOK (taban/tavan ile aynı gerekçe) — carpan=10^basamak ile
// olcekle, sıfıra-dogru-kes + |kesir|>=0.5 ise 1 duzelt (yarım-noktalar
// SIFIRDAN UZAGA yuvarlanir — Go'nun math.Round'u ile ayni kural), sonra
// carpandan geri boll.
func (e *elfUretici) yardimciYuvarla() {
	m := e.m
	m.etiketKoy("f_yuvarla")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 80)
	m.movYerelYaz(-8, rRDI)  // sayı
	m.movYerelYaz(-16, rRSI) // basamak
	m.movImm64(rRAX, int64(math.Float64bits(1.0)))
	m.movYerelYaz(-24, rRAX) // çarpan = 1.0
	m.movYerelImm(-32, 0)    // i = 0
	m.etiketKoy("Lyv_pow_dongu")
	m.movYerelOku(rRAX, -32)
	m.movYerelOku(rRDX, -16)
	m.cmpKayit(rRAX, rRDX)
	m.jcc(0x8D, "Lyv_pow_bitti") // jge
	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRDX, int64(math.Float64bits(10.0)))
	m.movqXmmKayit(1, rRDX)
	m.sseIkili(0x59, 0, 1) // mulsd
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-24, rRAX)
	m.movYerelOku(rRAX, -32)
	m.incKayit(rRAX)
	m.movYerelYaz(-32, rRAX)
	m.jmp("Lyv_pow_dongu")
	m.etiketKoy("Lyv_pow_bitti")

	// çarpılmış = sayı * çarpan
	m.movYerelOku(rRAX, -8)
	m.movqXmmKayit(0, rRAX)
	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-40, rRAX) // çarpılmış

	// trunc = kes(çarpılmış); frac = çarpılmış - float(trunc)
	m.movYerelOku(rRAX, -40)
	m.movqXmmKayit(0, rRAX)
	m.cvtKesirTam(rRCX, 0) // rcx = trunc
	m.movYerelYaz(-48, rRCX)
	m.cvtTamKesir(1, rRCX) // xmm1 = float(trunc)
	m.movYerelOku(rRAX, -40)
	m.movqXmmKayit(0, rRAX) // xmm0 = çarpılmış
	m.sseIkili(0x5C, 0, 1)  // subsd -> frac
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-56, rRAX) // frac

	// kosul1 = frac >= 0.5
	m.movYerelOku(rRAX, -56)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRDX, int64(math.Float64bits(0.5)))
	m.movqXmmKayit(1, rRDX)
	m.comisd(0, 1)
	m.setcc(0x93, rRAX) // setae
	m.movzx(rRAX, rRAX)
	m.movYerelYaz(-64, rRAX)

	// kosul2 = frac <= -0.5  <=>  -0.5 >= frac
	m.movYerelOku(rRAX, -56)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRDX, int64(math.Float64bits(-0.5)))
	m.movqXmmKayit(1, rRDX)
	m.comisd(1, 0)
	m.setcc(0x93, rRAX) // setae
	m.movzx(rRAX, rRAX)
	m.movYerelYaz(-72, rRAX)

	// trunc += kosul1 - kosul2
	m.movYerelOku(rRCX, -48)
	m.movYerelOku(rRAX, -64)
	m.movYerelOku(rRDX, -72)
	m.subKayit(rRAX, rRDX)
	m.addKayit(rRCX, rRAX)
	m.movYerelYaz(-48, rRCX)

	// sonuç = float(trunc) / çarpan
	m.cvtTamKesir(0, rRCX)
	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x5E, 0, 1) // divsd
	m.movqKayitXmm(rRAX, 0)
	m.leave()
	m.ret()
}

// birlestirListesi: birleştir(liste) icin ORTAK dongu — etiket adiyla UC
// AYRI giris noktasi uretilir (metin/tam/kesir), donusum bos ise oge zaten
// metin sayilip dogrudan birlestirilir, degilse verilen f_sayi_metne/
// f_kesir_metne etiketinden gecirilir. Kod tekrari (indirect call yerine)
// bilincli tercih — bu derleyicide fonksiyon-pointer/indirect-call destegi
// yok, Go tarafinda tek sablon uc kez ornekleniyor.
func (e *elfUretici) yardimciBirlestirListesi(etiket string, donusum string) {
	m := e.m
	m.etiketKoy(etiket)
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 32)
	m.movYerelYaz(-8, rRDI) // liste
	m.movImm32(rRDI, 8)
	m.call("f_tan_ayir")
	m.movImm32(rRCX, 0)
	m.movDolayliYaz(rRAX, 0, rRCX)
	m.movYerelYaz(-16, rRAX) // sonuç = ""
	m.movYerelOku(rRAX, -8)
	m.movDolayliOku(rRAX, rRAX, 0)
	m.movYerelYaz(-24, rRAX) // n
	m.movYerelImm(-32, 0)    // i
	m.etiketKoy("L" + etiket + "_dongu")
	m.movYerelOku(rRAX, -32)
	m.movYerelOku(rRDX, -24)
	m.cmpKayit(rRAX, rRDX)
	m.jcc(0x8D, "L"+etiket+"_bitti") // jge
	m.movYerelOku(rRAX, -8)
	m.movYerelOku(rRCX, -32)
	m.leaOge(rRAX, rRAX, rRCX)
	m.movDolayliOku(rRAX, rRAX, 0) // oge ham deger
	if donusum != "" {
		m.movKayit(rRDI, rRAX)
		m.call(donusum) // rax = metin
	}
	m.movKayit(rRSI, rRAX)   // oge (metin)
	m.movYerelOku(rRDI, -16) // sonuç (a)
	m.call("f_metin_birlestir")
	m.movYerelYaz(-16, rRAX)
	m.movYerelOku(rRAX, -32)
	m.incKayit(rRAX)
	m.movYerelYaz(-32, rRAX)
	m.jmp("L" + etiket + "_dongu")
	m.etiketKoy("L" + etiket + "_bitti")
	m.movYerelOku(rRAX, -16)
	m.leave()
	m.ret()
}

// rastgele(rdi = n) -> rax = [0,n) araliginda tam sayi, n<=0 ise 0.
// getrandom/dev-urandom YOK (elf backend'de dosya-sistemi baglami minimal) —
// RDTSC ile BIR KEZ tohumlanan xorshift64* ile uretiliyor. Kriptografik
// KALITE gerektirmiyor (kullanim: Noral.tan agirlik ilklemesi gibi) — bu
// yuzden interpreter'in math/rand'iyla BIT-BIT AYNI DIZI beklenmiyor/
// beklenemez (farkli PRNG, farkli tohum kaynagi).
func (e *elfUretici) yardimciRastgele() {
	m := e.m
	m.etiketKoy("f_rastgele")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	e.genel["__rasgeleIlk"] = true
	e.genel["__rasgeleDurum"] = true

	m.cmpImm32(rRDI, 0)
	m.jcc(0x8F, "Lr_pozitif") // jg
	m.movImm32(rRAX, 0)
	m.leave()
	m.ret()
	m.etiketKoy("Lr_pozitif")
	m.pushKayit(rRDI) // n'i koru

	m.movGenelOku(rRCX, "v___rasgeleIlk")
	m.testKayit(rRCX, rRCX)
	m.jcc(0x85, "Lr_tohumlanmis") // jnz
	m.rdtsc()
	m.shlImm(rRDX, 32)
	m.orKayit(rRAX, rRDX)
	m.movImm64(rRCX, 1)
	m.orKayit(rRAX, rRCX) // asla 0 olmasin (xorshift durumu 0'da takilir kalir)
	m.movGenelYaz("v___rasgeleDurum", rRAX)
	m.movImm32(rRCX, 1)
	m.movGenelYaz("v___rasgeleIlk", rRCX)
	m.etiketKoy("Lr_tohumlanmis")

	// xorshift64
	m.movGenelOku(rRAX, "v___rasgeleDurum")
	m.movKayit(rRCX, rRAX)
	m.shlImm(rRCX, 13)
	m.xorKayit(rRAX, rRCX)
	m.movKayit(rRCX, rRAX)
	m.shrImm(rRCX, 7)
	m.xorKayit(rRAX, rRCX)
	m.movKayit(rRCX, rRAX)
	m.shlImm(rRCX, 17)
	m.xorKayit(rRAX, rRCX)
	m.movGenelYaz("v___rasgeleDurum", rRAX)

	m.movImm64(rRCX, 0x7FFFFFFFFFFFFFFF)
	m.andKayit(rRAX, rRCX) // isaret bitini temizle -> her zaman >=0
	m.popKayit(rRCX)       // n
	m.cqo()
	m.idivKayit(rRCX)
	m.movKayit(rRAX, rRDX) // kalan = [0, n)
	m.leave()
	m.ret()
}

const elfLn2 = 0.6931471805599453
const elfInvLn2 = 1.4426950408889634

// e_ussu(rdi = x[ondalık ham]) -> rax = e^x[ham]
// Donanimda dogrudan bir "exp" komutu YOK (x87 F2XM1 kullanilmiyor — SSE'ye
// sadik kaliyoruz). Klasik aralik-indirgeme: x = k*ln2 + r, |r|<=ln2/2,
// e^x = 2^k * e^r. e^r, Horner iç içe Taylor serisiyle (15 terim, |r|
// kucuk oldugundan cift kesinlik icin fazlasiyla yeterli). 2^k, IEEE754 bit
// deseni dogrudan insa edilerek (üs alanina k eklenerek) TEK CARPMAYLA elde
// edilir — pow2 dongusu YOK.
func (e *elfUretici) yardimciEUssu() {
	m := e.m
	m.etiketKoy("f_e_ussu")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 88)
	m.movYerelYaz(-8, rRDI) // x

	// y = x * (1/ln2)
	m.movYerelOku(rRAX, -8)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRDX, int64(math.Float64bits(elfInvLn2)))
	m.movqXmmKayit(1, rRDX)
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-16, rRAX) // y

	// bias = (y>=0) ? 0.5 : -0.5
	m.movYerelOku(rRAX, -16)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRDX, 0) // 0.0
	m.movqXmmKayit(1, rRDX)
	m.comisd(0, 1)
	m.setcc(0x93, rRCX) // setae -> y>=0
	m.movzx(rRCX, rRCX)
	m.testKayit(rRCX, rRCX)
	m.jcc(0x84, "Lex_negbias") // jz
	m.movImm64(rRAX, int64(math.Float64bits(0.5)))
	m.jmp("Lex_biashazir")
	m.etiketKoy("Lex_negbias")
	m.movImm64(rRAX, int64(math.Float64bits(-0.5)))
	m.etiketKoy("Lex_biashazir")
	m.movYerelYaz(-24, rRAX) // bias

	// y += bias ; k = kes(y)
	m.movYerelOku(rRAX, -16)
	m.movqXmmKayit(0, rRAX)
	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x58, 0, 1)
	m.cvtKesirTam(rRCX, 0) // rcx = k
	m.movYerelYaz(-32, rRCX)

	// r = x - k*ln2
	m.cvtTamKesir(1, rRCX) // xmm1 = float(k)
	m.movImm64(rRAX, int64(math.Float64bits(elfLn2)))
	m.movqXmmKayit(2, rRAX) // xmm2 = ln2
	m.sseIkili(0x59, 1, 2)  // xmm1 = k*ln2
	m.movYerelOku(rRAX, -8)
	m.movqXmmKayit(0, rRAX) // xmm0 = x
	m.sseIkili(0x5C, 0, 1)  // xmm0 = x - k*ln2 = r
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-40, rRAX) // r

	// acc = 1.0 ; i = 15 ; while i>=1: acc = 1 + (r/i)*acc
	m.movImm64(rRAX, int64(math.Float64bits(1.0)))
	m.movYerelYaz(-48, rRAX) // acc
	m.movYerelImm(-56, 15)   // i
	m.etiketKoy("Lex_poly_dongu")
	m.movYerelOku(rRAX, -56)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x8E, "Lex_poly_bitti") // jle
	m.movYerelOku(rRCX, -56)
	m.cvtTamKesir(0, rRCX) // xmm0 = float(i)
	m.movYerelOku(rRAX, -40)
	m.movqXmmKayit(1, rRAX) // xmm1 = r
	m.sseIkili(0x5E, 1, 0)  // xmm1 = r/i
	m.movYerelOku(rRAX, -48)
	m.movqXmmKayit(0, rRAX) // xmm0 = acc
	m.sseIkili(0x59, 1, 0)  // xmm1 = (r/i)*acc
	m.movImm64(rRAX, int64(math.Float64bits(1.0)))
	m.movqXmmKayit(0, rRAX) // xmm0 = 1.0
	m.sseIkili(0x58, 0, 1)  // xmm0 = 1 + (r/i)*acc
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-48, rRAX) // acc
	m.movYerelOku(rRAX, -56)
	m.decKayit(rRAX)
	m.movYerelYaz(-56, rRAX)
	m.jmp("Lex_poly_dongu")
	m.etiketKoy("Lex_poly_bitti")

	// sonuç = acc * 2^k  (2^k'nin IEEE754 bit deseni dogrudan insa edilir)
	m.movYerelOku(rRAX, -32)
	m.addImm32(rRAX, 1023)
	m.shlImm(rRAX, 52)
	m.movqXmmKayit(1, rRAX) // xmm1 = 2^k
	m.movYerelOku(rRAX, -48)
	m.movqXmmKayit(0, rRAX) // xmm0 = acc
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.leave()
	m.ret()
}

// log(rdi = x[ondalık ham, x>0]) -> rax = ln(x)[ham]
// x = m * 2^e (m in [1,2)) IEEE754 bit alanlarindan DOGRUDAN cikarilir (frexp
// esdegeri, hesaplama YOK — sadece bit maskeleme). ln(m), f=(m-1)/(m+1)
// donusumuyle HIZLI yakinsayan seriyle: ln(m) = 2*(f + f^3/3 + f^5/5 + ...).
// x<=0 icin tanimsiz (NaN/cop bit deseni donebilir) — dene/yakala elf'te
// zaten yok, calisma-zamani hata kontrolu bu derleyicinin genelinde YOK.
func (e *elfUretici) yardimciLog() {
	m := e.m
	m.etiketKoy("f_log")
	m.pushKayit(rRBP)
	m.movKayit(rRBP, rRSP)
	m.subImm32(rRSP, 80)
	m.movYerelYaz(-8, rRDI) // x bit deseni

	// e = ((bits >> 52) & 0x7FF) - 1023
	m.movYerelOku(rRAX, -8)
	m.shrImm(rRAX, 52)
	m.movImm64(rRCX, 0x7FF)
	m.andKayit(rRAX, rRCX)
	m.subImm32(rRAX, 1023)
	m.movYerelYaz(-16, rRAX) // e

	// m bits = (bits & 0x000FFFFFFFFFFFFF) | (1023<<52)
	m.movYerelOku(rRAX, -8)
	m.movImm64(rRCX, 0x000FFFFFFFFFFFFF)
	m.andKayit(rRAX, rRCX)
	m.movImm64(rRDX, int64(1023)<<52)
	m.orKayit(rRAX, rRDX)
	m.movYerelYaz(-24, rRAX) // m

	// pay = m-1, payda = m+1
	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRCX, int64(math.Float64bits(1.0)))
	m.movqXmmKayit(1, rRCX)
	m.sseIkili(0x5C, 0, 1) // m-1
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-32, rRAX) // pay

	m.movYerelOku(rRAX, -24)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRCX, int64(math.Float64bits(1.0)))
	m.movqXmmKayit(1, rRCX)
	m.sseIkili(0x58, 0, 1) // m+1
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-40, rRAX) // payda

	// f = pay/payda
	m.movYerelOku(rRAX, -32)
	m.movqXmmKayit(0, rRAX)
	m.movYerelOku(rRAX, -40)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x5E, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-32, rRAX) // f (pay yerine kalici)

	// f2 = f*f
	m.movYerelOku(rRAX, -32)
	m.movqXmmKayit(0, rRAX)
	m.movYerelOku(rRAX, -32)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-40, rRAX) // f2 (payda yerine kalici)

	// term = f ; sum = f
	m.movYerelOku(rRAX, -32)
	m.movYerelYaz(-48, rRAX) // term
	m.movYerelYaz(-56, rRAX) // sum
	m.movYerelImm(-64, 3)    // denom
	m.movYerelImm(-72, 8)    // sayac

	m.etiketKoy("Llog_dongu")
	m.movYerelOku(rRAX, -72)
	m.cmpImm32(rRAX, 0)
	m.jcc(0x8E, "Llog_bitti") // jle

	// term *= f2
	m.movYerelOku(rRAX, -48)
	m.movqXmmKayit(0, rRAX)
	m.movYerelOku(rRAX, -40)
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-48, rRAX)

	// sum += term/denom
	m.movYerelOku(rRCX, -64)
	m.cvtTamKesir(1, rRCX) // xmm1 = float(denom)
	m.movYerelOku(rRAX, -48)
	m.movqXmmKayit(0, rRAX) // xmm0 = term
	m.sseIkili(0x5E, 0, 1)  // xmm0 = term/denom
	m.movYerelOku(rRAX, -56)
	m.movqXmmKayit(1, rRAX) // xmm1 = sum
	m.sseIkili(0x58, 0, 1)  // xmm0 = sum + term/denom
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-56, rRAX)

	m.movYerelOku(rRAX, -64)
	m.addImm32(rRAX, 2)
	m.movYerelYaz(-64, rRAX) // denom += 2
	m.movYerelOku(rRAX, -72)
	m.decKayit(rRAX)
	m.movYerelYaz(-72, rRAX) // sayac--
	m.jmp("Llog_dongu")
	m.etiketKoy("Llog_bitti")

	// ln(m) = 2*sum
	m.movYerelOku(rRAX, -56)
	m.movqXmmKayit(0, rRAX)
	m.movImm64(rRCX, int64(math.Float64bits(2.0)))
	m.movqXmmKayit(1, rRCX)
	m.sseIkili(0x59, 0, 1)
	m.movqKayitXmm(rRAX, 0)
	m.movYerelYaz(-56, rRAX) // ln(m)

	// sonuç = e*ln2 + ln(m)
	m.movYerelOku(rRCX, -16)
	m.cvtTamKesir(0, rRCX) // xmm0 = float(e)
	m.movImm64(rRAX, int64(math.Float64bits(elfLn2)))
	m.movqXmmKayit(1, rRAX)
	m.sseIkili(0x59, 0, 1) // xmm0 = e*ln2
	m.movYerelOku(rRAX, -56)
	m.movqXmmKayit(1, rRAX) // xmm1 = ln(m)
	m.sseIkili(0x58, 0, 1)  // xmm0 = e*ln2 + ln(m)
	m.movqKayitXmm(rRAX, 0)
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

	// --- YUVARLAMA TASMASI ---
	// kesirli kisim 0.9999995'in UZERINDEYSE (ör. log(e) ~ 0.999999999999998)
	// yukaridaki "+0.5, kes" 1000000 uretir — 6 basamaga SIGMAZ. Bu
	// yakalanmadan "sondaki sifirlari kirp" dongusu 1000000'i sifirlara
	// bolerek "0.0" gibi YANLIS bir sonuca kirpiyordu (tam kisim asla
	// arttirilmadigi icin). Tasma varsa tam kismi 1 arttir, kesri sifirla.
	m.cmpImm32(rRDX, 1000000)
	m.jcc(0x85, "Lkm_tasmayok") // jne
	m.movYerelOku(rRAX, -64)
	m.incKayit(rRAX)
	m.movYerelYaz(-64, rRAX)
	m.movImm32(rRDX, 0)
	m.movYerelYaz(-72, rRDX)
	m.etiketKoy("Lkm_tasmayok")

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

// elfIceAlGenislet: "içe al" statik derleme zamaninda cozulur — elf tek
// gecisli/tum-program derleyicisi oldugundan (yorumlayicinin aksine calisma
// aninda dosya okuyamaz), her IceAlDugum'un yerine cozulmus modulun (rekursif
// olarak genisletilmis) govdesi yerlestirilir. Yorumlayicinin y.iceAl() ile
// AYNI kural: modul en fazla bir kez yuklenir (mutlak yol ile), donguculer
// otomatik engellenir (Yorumlayici.go:134-166 ile karsilastir).
func elfIceAlGenislet(agac []Dugum, kaynakDizin string, alinanlar map[string]bool) []Dugum {
	var sonuc []Dugum
	for _, d := range agac {
		ia, ok := d.(IceAlDugum)
		if !ok {
			sonuc = append(sonuc, d)
			continue
		}
		yol, bulundu := modulAra(ia.Dosya, kaynakDizin)
		if !bulundu {
			panic(TanHata{Satir: ia.Satir, Mesaj: fmt.Sprintf("modül bulunamadı: %s\n%s", ia.Dosya, modulAramaYollari(ia.Dosya, kaynakDizin))})
		}
		mutlak, err := filepath.Abs(yol)
		if err != nil {
			mutlak = yol
		}
		if alinanlar[mutlak] {
			continue // döngüsel/tekrar içe alma — atla (yorumlayıcıyla aynı kural)
		}
		alinanlar[mutlak] = true
		kaynak, err := os.ReadFile(yol)
		if err != nil {
			panic(TanHata{Satir: ia.Satir, Mesaj: fmt.Sprintf("modül okunamadı: %v", err)})
		}
		lexer := YeniLexer(string(kaynak))
		parser := YeniParser(lexer.Tokenle())
		modAgac := parser.Ayristir()
		sonuc = append(sonuc, elfIceAlGenislet(modAgac, filepath.Dir(mutlak), alinanlar)...)
	}
	return sonuc
}

// elfCagrilanAdlariTopla: bir AST dugumu icinde (rekursif) gecen TUM
// CagriDugum adlarini hedef kumeye ekler. elfUlasilabilirIslevler'in
// yapi taslarindan biri.
func elfCagrilanAdlariTopla(d Dugum, hedef map[string]bool) {
	switch n := d.(type) {
	case AtamaDugum:
		elfCagrilanAdlariTopla(n.Deger, hedef)
	case IndeksAtamaDugum:
		elfCagrilanAdlariTopla(n.Hedef, hedef)
		elfCagrilanAdlariTopla(n.Indeks, hedef)
		elfCagrilanAdlariTopla(n.Deger, hedef)
	case YazDugum:
		elfCagrilanAdlariTopla(n.Deger, hedef)
	case EgerDugum:
		elfCagrilanAdlariTopla(n.Kosul, hedef)
		for _, s := range n.Govde {
			elfCagrilanAdlariTopla(s, hedef)
		}
		for _, s := range n.Degilse {
			elfCagrilanAdlariTopla(s, hedef)
		}
	case IkenDugum:
		elfCagrilanAdlariTopla(n.Kosul, hedef)
		for _, s := range n.Govde {
			elfCagrilanAdlariTopla(s, hedef)
		}
	case HerDugum:
		elfCagrilanAdlariTopla(n.Liste, hedef)
		for _, s := range n.Govde {
			elfCagrilanAdlariTopla(s, hedef)
		}
	case DondurDugum:
		if n.Deger != nil {
			elfCagrilanAdlariTopla(n.Deger, hedef)
		}
	case CagriDugum:
		hedef[n.Ad] = true
		// içParcaLat(ad, ...): ilk argüman ÇALIŞMA-ZAMANI değeri değil,
		// derleme-zamanı bilinen bir işlev ADI (bkz. ifade() codegen'i) —
		// normal CagriDugum/DegiskenDugum gezinmesi bunu bir "çağrı" olarak
		// GÖRMEZ (DegiskenDugum bu tarayıcıda hiç ele alınmıyor), bu yüzden
		// hedef işlev BAŞKA yerden çağrılmıyorsa "kullanılmayan işlev" budama
		// geçişinde YANLIŞLIKLA silinirdi — özel durum.
		if n.Ad == "içParcaLat" && len(n.Argumanlar) > 0 {
			if dv, ok := n.Argumanlar[0].(DegiskenDugum); ok {
				hedef[dv.Ad] = true
			}
		}
		for _, a := range n.Argumanlar {
			elfCagrilanAdlariTopla(a, hedef)
		}
	case IkiliDugum:
		elfCagrilanAdlariTopla(n.Sol, hedef)
		if n.Sag != nil {
			elfCagrilanAdlariTopla(n.Sag, hedef)
		}
	case IndeksDugum:
		elfCagrilanAdlariTopla(n.Hedef, hedef)
		elfCagrilanAdlariTopla(n.Indeks, hedef)
	case ListeDugum:
		for _, x := range n.Elemanlar {
			elfCagrilanAdlariTopla(x, hedef)
		}
	case SozlukDugum:
		for _, x := range n.Anahtarlar {
			elfCagrilanAdlariTopla(x, hedef)
		}
		for _, x := range n.Degerler {
			elfCagrilanAdlariTopla(x, hedef)
		}
	case KayitOlusturDugum:
		for _, x := range n.Degerler {
			elfCagrilanAdlariTopla(x, hedef)
		}
	case AlanErisimDugum:
		elfCagrilanAdlariTopla(n.Hedef, hedef)
	case AlanAtamaDugum:
		elfCagrilanAdlariTopla(n.Hedef, hedef)
		elfCagrilanAdlariTopla(n.Deger, hedef)
	case MetotCagriDugum:
		elfCagrilanAdlariTopla(n.Hedef, hedef)
		for _, a := range n.Argumanlar {
			elfCagrilanAdlariTopla(a, hedef)
		}
	}
}

// elfUlasilabilirIslevler: anaGovde'den (dogrudan/dolayli cagri zinciriyle)
// erisilebilen islev adlarinin kumesini dondurur (BFS).
func elfUlasilabilirIslevler(anaGovde []Dugum, islevler []IslevDugum) map[string]bool {
	govdeler := map[string][]Dugum{}
	for _, isv := range islevler {
		govdeler[isv.Ad] = isv.Govde
	}
	ulasilan := map[string]bool{}
	var kuyruk []string
	topla := func(govde []Dugum) {
		cagrilan := map[string]bool{}
		for _, d := range govde {
			elfCagrilanAdlariTopla(d, cagrilan)
		}
		for ad := range cagrilan {
			if _, varMi := govdeler[ad]; varMi && !ulasilan[ad] {
				ulasilan[ad] = true
				kuyruk = append(kuyruk, ad)
			}
		}
	}
	topla(anaGovde)
	for len(kuyruk) > 0 {
		ad := kuyruk[0]
		kuyruk = kuyruk[1:]
		topla(govdeler[ad])
	}
	return ulasilan
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

	// --- ICE AL: modul agaclarini derleme zamaninda ic ice yerlestir ---
	anaDizinMutlak, err := filepath.Abs(filepath.Dir(dosya))
	if err != nil {
		anaDizinMutlak = filepath.Dir(dosya)
	}
	agac = elfIceAlGenislet(agac, anaDizinMutlak, map[string]bool{})

	// --- OPTIMIZE GECISI: sabit katlama, cebirsel sadelestirme, olu kod ---
	opt := YeniOptimizer()
	agac = opt.Govde(agac)
	if os.Getenv("TAN_OPT_RAPOR") != "" {
		fmt.Fprintf(os.Stderr, "optimize: %d katlama, %d ölü blok\n", opt.Katlanan, opt.Silinen)
	}

	var islevler []IslevDugum
	var anaGovde []Dugum
	var kayitTanimlari []KayitTanimDugum
	for _, d := range agac {
		switch dd := d.(type) {
		case IslevDugum:
			islevler = append(islevler, dd)
		case KayitTanimDugum:
			kayitTanimlari = append(kayitTanimlari, dd)
		default:
			anaGovde = append(anaGovde, d)
		}
	}

	// --- KULLANILMAYAN ISLEVLERI AT ---
	// icealGenislet() ile artik butun modul agaci tek programa katiliyor —
	// bir kutuphane dosyasindaki HIC CAGRILMAYAN islevler de (ör. bu programda
	// kullanilmayan yardimci fonksiyonlar) govdeYaz/tip-cikarim asamasindan
	// gecmek zorunda kalirdi. Boyle bir islev jenerik bir parametre alip
	// (ornegin uzunluk(liste) gibi) TIP-KATI bir yerlesik cagirirsa, cagri
	// yeri olmadigindan parametre tipi asla ogrenilemez ve "tam" a duser —
	// derleme HATASIYLA duruyordu (halbuki program o islevi hic kullanmiyor).
	// Ana govdeden ULASILABILIR islevleri bul, geri kalanini derleme.
	// NOT: kayit metotlarinin govdeleri de KOK olarak eklenir (additif,
	// budama YAPMAZ) — bir metot govdesi baska bir kutuphane islevini
	// cagiriyorsa o islev de reachability grafiginde "kullanilan" sayilmali,
	// aksi halde metot pruning'den SONRA eklendigi icin (asagida) o islev
	// yanlislikla atilmis olabilirdi.
	kokGovde := append([]Dugum{}, anaGovde...)
	for _, kt := range kayitTanimlari {
		for _, mt := range kt.Metotlar {
			kokGovde = append(kokGovde, mt.Govde...)
		}
	}
	ulasilanIslevler := elfUlasilabilirIslevler(kokGovde, islevler)
	var kullanilanIslevler []IslevDugum
	for _, isv := range islevler {
		if ulasilanIslevler[isv.Ad] {
			kullanilanIslevler = append(kullanilanIslevler, isv)
		}
	}
	islevler = kullanilanIslevler

	e := &elfUretici{m: yeniMakineKodu(), genel: map[string]bool{}, tipler: map[string]Tip{}, islevTipi: map[string]Tip{}, parametreTipi: map[string]Tip{}, kayitSemalari: map[string]*KayitSemasi{}, islevParamSayisi: map[string]int{}}
	for _, isv := range islevler {
		e.islevParamSayisi[isv.Ad] = len(isv.Parametreler)
	}

	// --- KAYIT SEMALARI: alan sirasi/ofset + metotlar ---
	// Metotlar HER ZAMAN derlenir (yukaridaki "kullanilmayan islevleri at"
	// budamasindan MUAFTIR) — kayit tanimlari genelde kullanicinin kendi
	// programinda dogrudan yazilir (buyuk, cogunlukla kullanilmayan bir
	// kutuphaneden ice aktarilan onlarca yardimci fonksiyon gibi degil),
	// bu yuzden "kullanilmiyor olabilir" riski dusuk — buna karsin
	// reachability grafigini (yukarida) metot govdeleriyle KOK olarak
	// genisletmek, metodun cagirdigi BASKA islevlerin yanlislikla
	// atilmasini onluyor.
	for _, kt := range kayitTanimlari {
		sema := &KayitSemasi{Ad: kt.Ad, Alanlar: kt.Alanlar, AlanIndeks: map[string]int{}, AlanTipleri: map[string]Tip{}, Metotlar: map[string]IslevDugum{}}
		for i, alan := range kt.Alanlar {
			sema.AlanIndeks[alan] = i
			sema.AlanTipleri[alan] = TipTam
		}
		for _, mt := range kt.Metotlar {
			sema.Metotlar[mt.Ad] = mt
			sentetikAd := kayitMetotAdi(kt.Ad, mt.Ad)
			metotKopya := mt
			metotKopya.Ad = sentetikAd
			islevler = append(islevler, metotKopya)
			if len(mt.Parametreler) > 0 {
				e.parametreTipi[sentetikAd+"/"+mt.Parametreler[0]] = Tip{Cesit: CKayit, KayitAdi: kt.Ad}
			}
		}
		e.kayitSemalari[kt.Ad] = sema
	}

	// --- islev donus tiplerini cikar (sabit noktaya kadar yinele) ---
	// Cagri yerlerinden parametre tiplerini, dondur deyimlerinden donus tipini bul.
	// NOT: sabit "3 tur" DERIN cagri zincirlerinde (ör. 8+ katmanli ozyineli-
	// inis ayristirici) yetmiyordu — her tur, tip bilgisini cagri grafiginde
	// yalniz BIR katman ileri tasiyor, bu yuzden N-katmanli zincir en az N
	// tur ister. Simdi GERCEK sabit noktaya kadar (parametreTipi degismeyene
	// dek) don, guvenlik siniri 50 tur (yakinsamayan patolojik durum icin).
	for tur := 0; tur < 50; tur++ {
		oncekiSayi := len(e.parametreTipi)
		// ust seviye (anaGovde) degisken tiplerini HER TURDA yenile: bir
		// ust seviye atamasi "x = kullaniciIslevi(...)" ise, tipi ancak
		// kullaniciIslevi'nin donus tipi (e.islevTipi) o ana kadar
		// ogrenilmisse dogru cikar. Tek seferlik (dongu oncesi) cagri
		// yetmiyordu — ilk turda islevTipi bos oldugu icin sessizce "tam"a
		// duesuyordu (bkz. "tokenler = tokenle(kaynak)" -> "tam" hatasi).
		e.govdeTipleriniTopla(anaGovde)
		// GLOBAL sozluklerin deger tipini anaGovde + TUM islev govdeleri
		// birlikte taranarak coz (bkz. sozlukElemanlariniCoz yorumu) — bu,
		// per-islev "eski" restore'undan ONCE, e.tipler["kX"] uzerinde
		// (anaGovde'ye ait, hicbir islevin eski-restore'u tarafindan
		// SILINMEYEN girdi) calisir.
		{
			govdeler := make([][]Dugum, 0, len(islevler)+1)
			govdeler = append(govdeler, anaGovde)
			for _, isv := range islevler {
				govdeler = append(govdeler, isv.Govde)
			}
			e.sozlukElemanlariniCoz(govdeler...)
			e.kayitAlanTipleriniCoz(govdeler...)
		}
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
			e.sozlukElemanlariniCoz(isv.Govde)
			if t, bulundu := e.dondurTipi(isv.Govde); bulundu {
				e.islevTipi[isv.Ad] = t
			}
			// bu islevin govdesindeki cagrilari, TAM DA bu islevin yerel
			// degisken tipleriyle (henuz sifirlanmamisken) tara.
			e.parametreTipleriniOgren(isv.Govde, islevler)
			e.tipler = eski
		}
		// ana govdedeki (ust seviye) cagrilari ana govdenin tipleriyle tara
		e.parametreTipleriniOgren(agac, islevler)
		if os.Getenv("TAN_TIP_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "tip-cikarim: tur=%d oğrenilenParametreSayisi=%d\n", tur, len(e.parametreTipi))
		}
		if len(e.parametreTipi) == oncekiSayi {
			break // sabit noktaya ulasildi — daha fazla tur yeni bilgi eklemiyor
		}
	}

	// yardımcılar + kullanıcı işlevleri + _start
	e.yardimciYazMetin()
	e.yardimciYazSayi()
	e.yardimciAyir()
	e.yardimciArenaAyir()
	e.yardimciArenaSerbest()
	e.yardimciKopyala()
	e.yardimciYazMetinDeger()
	e.yardimciBirlestir()
	e.yardimciSayiMetne()
	e.yardimciListeEkle()
	e.yardimciListeYap()
	e.yardimciMetinIndeks()
	e.yardimciMetinEsit()
	e.yardimciKarakter()
	e.yardimciKod()
	e.yardimciHarfler()
	e.yardimciMetinAraligi()
	e.yardimciParcala()
	e.yardimciSozlukHash()
	e.yardimciSozlukYap()
	e.yardimciSozlukKoy()
	e.yardimciSozlukAl()
	e.yardimciSozlukVarmi()
	e.yardimciSozlukAnahtarlar()
	e.yardimciSozlukSil()
	e.yardimciOku()
	e.yardimciYazDosya()
	e.yardimciYazBaytlar()
	e.yardimciArgsay()
	e.yardimciArg()
	e.yardimciKesirMetne()
	e.yardimciYuvarla()
	e.yardimciEUssu()
	e.yardimciLog()
	e.yardimciRastgele()
	e.yardimciBirlestirListesi("f_birlestir_metin", "")
	e.yardimciBirlestirListesi("f_birlestir_tam", "f_sayi_metne")
	e.yardimciBirlestirListesi("f_birlestir_kesir", "f_kesir_metne")
	e.yardimciEkleDosya()
	e.yardimciDosyaVarMi()
	e.yardimciPositionalIO()
	e.yardimciEszamanlilik()
	e.yardimciSayi()
	e.yardimciYazKesir()
	for _, isv := range islevler {
		e.islevYaz(isv)
	}
	e.govdeTipleriniTopla(anaGovde) // ana govde tipleri
	{
		govdeler := make([][]Dugum, 0, len(islevler)+1)
		govdeler = append(govdeler, anaGovde)
		for _, isv := range islevler {
			govdeler = append(govdeler, isv.Govde)
		}
		e.sozlukElemanlariniCoz(govdeler...)
		e.kayitAlanTipleriniCoz(govdeler...)
	}
	e.m.etiketKoy("_start")
	e.argvYakala() // argv tabanini sakla (rsp henuz bozulmadi)
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
