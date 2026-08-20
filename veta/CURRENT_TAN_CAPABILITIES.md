# CURRENT_TAN_CAPABILITIES.md

*Tarih: 2026-08-16. Özellik başına durum: **EXISTS** / **PARTIAL** / **MISSING** /
**UNKNOWN**. "Mevcut" derken üç katman ayrımı yapılır:*
- **SELF** = self-hosted ELF yolu (TancElf.tan) — VETA'nın hedef katmanı
- **GO-ELF** = Go referans arka ucu (DerleElf.go)
- **GO-YORUM** = Go yorumlayıcı/VM (Go-only araç)

*Kural: UNKNOWN bırakmak, uydurmaktan iyidir. Burada UNKNOWN yok — her satır
kaynak taraması ya canlı deneyle desteklenir.*

---

## Language

| Özellik | SELF | GO-ELF | GO-YORUM | Kanıt |
|---|---|---|---|---|
| Türkçe anahtar kelimeler | EXISTS | EXISTS | EXISTS | TancElf.tan `anahtarMi`; Lexer.go (kaynakta var) |
| Yorumlar (`#`) | EXISTS | EXISTS | EXISTS | TancElf.tan satır 1-45 |
| Üst-seviye deyimler | EXISTS | EXISTS | EXISTS | TancElf.tan 4250+; canlı test |
| Satır/ifade sonlandırıcı (yeni satır) | EXISTS | EXISTS | EXISTS | tokenle SOH biçimi |
| Kaçış dizileri `\n \t \r` | EXISTS | EXISTS | EXISTS | TancElf.tan `kacisCoz` 71-82 |

## Types

| Özellik | SELF | GO-ELF | GO-YORUM | Kanıt |
|---|---|---|---|---|
| int64 (tam) | EXISTS | EXISTS | EXISTS | canlı: 123456789*987654321 birebir |
| float64 (kesir) | EXISTS | EXISTS | EXISTS | canlı: 100/7 → 14.285714 |
| mantık (doğru/yanlış) | PARTIAL | PARTIAL | EXISTS | FAZ 2 yapıldı (kayıt); yorumlayıcı tam |
| metin | EXISTS | EXISTS | EXISTS | canlı |
| liste | EXISTS | EXISTS | EXISTS | canlı |
| sözlük | MISSING | EXISTS | EXISTS | TancElf.tan'da f_sozluk yok; DerleElf.go 3564+ |
| kayıt (record) | MISSING | EXISTS | EXISTS | DerleElf.go kayıt üretimi; TancElf.tan yok |
| sabit genişlik tamsayı (u8..i64) | MISSING | EXISTS | EXISTS | SabitTam.go (çalışma ağacında yok ama Go yolunda mevcut) |
| köprü (bridge) | MISSING | MISSING | PARTIAL | Kopru.go iskeleti boş |

## Variables

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Yerel değişken | EXISTS | EXISTS | EXISTS |
| Global/üst-seviye değişken | EXISTS | EXISTS | EXISTS |
| Atama | EXISTS | EXISTS | EXISTS |
| Değişken tipi takibi | PARTIAL (atama bazlı, parametre yok) | PARTIAL | EXISTS (dinamik) |
| Sözlük elemanı yazma | MISSING | EXISTS | EXISTS |
| Kayıt alanı yazma | MISSING | EXISTS | EXISTS |

## Functions

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `işlev` tanımı | EXISTS | EXISTS | EXISTS |
| Çağrı (System V AMD64, ≤6 param) | EXISTS | EXISTS | EXISTS |
| Özyineleme | EXISTS | EXISTS | EXISTS (canlı: fib(20)=6765) |
| `döndür` | EXISTS | EXISTS | EXISTS |
| Dönüş tipi çıkarımı | PARTIAL (yalnız literal/doğrudan çağrı) | EXISTS | — |
| Parametre tipi çıkarımı | MISSING | EXISTS | — |
| Metot çağrısı | MISSING | EXISTS | EXISTS |

## Control flow

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `eğer`/`değilse`/`son` | EXISTS | EXISTS | EXISTS |
| `iken` | EXISTS | EXISTS | EXISTS |
| `her ... içinde` | EXISTS | EXISTS | EXISTS |
| `dur`/`devam` | EXISTS | EXISTS | EXISTS |
| Kısa devre `ve`/`veya` | EXISTS | EXISTS | EXISTS |
| `dene`/`yakala` | MISSING (net hata) | MISSING | EXISTS |
| Kısa devre dışı koşul zinciri | EXISTS | EXISTS | EXISTS |

## Memory

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| brk bump allocator (`f_tan_ayir`) | EXISTS | EXISTS | — (runtime) |
| Sınır kontrolü + exit(3) | EXISTS | EXISTS | — |
| `f_bellek_kopyala` | EXISTS | EXISTS | — |
| arena/free list | MISSING | EXISTS (f_arena_*) | — |
| `mmap` (`f_bellek_esle/coz`) | MISSING | EXISTS | — |
| Ham bellek okuma/yazma (f_ham_*) | MISSING | EXISTS | — |
| GC / serbest bırakma | MISSING (bilinçli) | MISSING (bilinçli) | — |

## Pointers

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Kullanıcı işaretçisi | MISSING (dil düzeyinde) | PARTIAL (f_ham_* ham bellek) | PARTIAL |
| Metin/liste işaretçisi (dahili) | EXISTS | EXISTS | — |

## Arrays / Lists

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Liste literal `[1,2,3]` | EXISTS | EXISTS | EXISTS |
| `uzunluk` | EXISTS | EXISTS | EXISTS |
| `ekle` | EXISTS | EXISTS | EXISTS |
| `listeYap(n, değer)` | EXISTS | EXISTS | EXISTS |
| İndeks okuma `l[i]` | EXISTS | EXISTS | EXISTS |
| İndeks atama `l[i]=x` | EXISTS | EXISTS | EXISTS |
| İç içe liste | EXISTS | EXISTS | EXISTS |
| `her ... içinde` listesi | EXISTS | EXISTS | EXISTS |

## Strings

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Literal | EXISTS | EXISTS | EXISTS |
| `+` birleştirme | PARTIAL (tip yığını bağımlı) | PARTIAL | EXISTS |
| `metinBirlestir` (tip-bağımsız) | EXISTS | EXISTS | EXISTS |
| `metinEsit` (içerik) | EXISTS | EXISTS | EXISTS |
| `s[i]` indeks (BAY'T bazlı) | EXISTS | EXISTS | — |
| `uzunluk` | EXISTS | EXISTS | EXISTS |
| `kod`/`karakter` | EXISTS | EXISTS | EXISTS |
| `metin(n)` dönüşüm | EXISTS | EXISTS | EXISTS |
| `harfler` (parçala-karakter) | MISSING | EXISTS | EXISTS |
| `parçala` (ayraçla böl) | MISSING | EXISTS | EXISTS |
| `metinAraliği` (alt metin) | MISSING | EXISTS | EXISTS |
| Çok baytlı UTF-8 rune tutarlılığı | UNKNOWN→BELGELİ FARK | UNKNOWN→BELGELİ FARK | UNKNOWN→BELGELİ FARK |

## Records / Dictionary

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `sözlük` yap/koy/al/varmi/sil/anahtarlar/hash | MISSING | EXISTS | EXISTS |
| `kayıt` tanım + alan + metot | MISSING | EXISTS | EXISTS |

## Bitwise

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `& \| ^ << >>` | MISSING (açık hata/kapsam dışı) | EXISTS | EXISTS |

## Arithmetic

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `+ - * / %` int64 | EXISTS | EXISTS | EXISTS |
| int/int bölünürse tam, değilse kesir | EXISTS | EXISTS | EXISTS |
| `/` her zaman kesir (yorumlayıcı) | — | — | EXISTS |
| float64 aritmetik + karşılaştırma | EXISTS | EXISTS | EXISTS |
| 2^53 sınır denetimi (sabit katlama) | EXISTS | EXISTS | — |
| Sıfıra bölme → net hata + exit(4) | EXISTS | EXISTS | — |
| `tamBol` (kesen bölme) | EXISTS | EXISTS | — |
| `f_yuvarla`, `f_e_ussu`, `f_log`, `f_rastgele` | MISSING | EXISTS | EXISTS |

## Atomic / Lock / Futex / Threads

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `lock cmpxchg/xadd` atomikler | MISSING | EXISTS (f_atomik_ekle_ham 4677) | EXISTS |
| futex wait/wake | MISSING | EXISTS (4587-4606) | EXISTS |
| kilit al/birak | MISSING | EXISTS (4645-4674) | EXISTS |
| iplik oluştur/bekle/çık | MISSING | EXISTS (clone + 4607-4643) | EXISTS (goroutine) |

## File I/O

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| `oku` (tüm dosya) | EXISTS | EXISTS | EXISTS |
| `yazBaytlar` (tüm dosya) | EXISTS | EXISTS | EXISTS |
| `dosyaVarMi` | EXISTS | EXISTS | EXISTS |
| `yaz_dosya`/`ekle_dosya` | MISSING | EXISTS | EXISTS |
| Konumlamalı aç/kapat/oku_blok/yaz_blok/konumla/senkron (f_dosya_ac_* 10) | MISSING | EXISTS | — |
| SEEK/TELL/EOF | MISSING | PARTIAL | — |
| Dosya boyutu | MISSING | PARTIAL | — |
| `arg`/`argsay` | EXISTS | EXISTS | EXISTS |

## ELF generation / Syscalls

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| ELF64 yazımı (elle başlık) | EXISTS | EXISTS | — |
| Program başlığı (1 PT_LOAD, R+W+X) | EXISTS | EXISTS | — |
| .symtab/.strtab opsiyonel (TAN_SEMBOLSUZ) | PARTIAL | EXISTS | — |
| rel32/veri fixup (`bagla`) | EXISTS | EXISTS | — |
| Ham syscall emisyonu | EXISTS | EXISTS | — |
| ARG / çıkış kodu (exit 0/1/3/4) | EXISTS | EXISTS | — |
| DWARF debug | MISSING | MISSING | — |

## Networking / Serialization

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Soket syscall'ları | MISSING | MISSING | MISSING |
| JSON/metin serileştirme | MISSING | PARTIAL (f_birlestir_*, f_parcala) | PARTIAL |
| İkili serileştirme | MISSING | PARTIAL (yazBaytlar) | PARTIAL |

## Error handling / Modules / Misc

| Özellik | SELF | GO-ELF | GO-YORUM |
|---|---|---|---|
| Derleme hatası → net mesaj + exit | EXISTS | EXISTS | EXISTS |
| Runtime hata (sıfıra bölme) | EXISTS | EXISTS | EXISTS |
| `içe al` (modül) | EXISTS (statik açılım) | EXISTS | EXISTS (runtime) |
| Modül arama (TAN_YOL, tan_moduller...) | MISSING (yalnız dizin göreli) | MISSING | EXISTS (Modul.go 7 basamak) |
| Paket yöneticisi | MISSING | MISSING | EXISTS |
| `test` bloğu koşucusu | MISSING | MISSING | EXISTS |
| Biçimlendirici / statik denetçi / simgeler / bağımlılık grafiği / api | MISSING | MISSING | EXISTS |
| FFI (disKutuphaneAc/Bul/Cagir) | MISSING | MISSING | EXISTS (purego, linux) |
| Generics / Reflection / Metaprogramming | MISSING | MISSING | PARTIAL (içe al açılımı bir tür makro) |

## Self-hosting durumu

| Özellik | Durum | Kanıt |
|---|---|---|
| Kendini derler | EXISTS | commit'li TancElf==gen1==gen2==gen3 (md5 aynı) |
| Sabit nokta | EXISTS | md5 `914b0ffb...` 4 artifact'te aynı |
| Go'suz üretim | EXISTS | TestArkaUcGoSuzTemiz.sh tuzak deseni |
| Deterministik | EXISTS | sabit taban + RIP-rel; gen4=gen5 argümanı |

## Not: DOKUMANTASYON ile SATIR SAYISI UYUMSUZLUĞU

CURRENT_ARCHITECTURE.md ve GO_KALDIRMA_PLANI.md `TancElf.tan` için "3552 satır",
`DerleElf.go` için "5998/6001 satır" der; çalışma ağacında **TancElf.tan 4310 satır**
ölçüldü. Belgeliklerin satır sayıları güncel değil; davranış doğrulamaları
etkilenmez ama sayıya dayanan hiçbir VETA kararı kaynak üzerinden tekrar doğrulanır.
