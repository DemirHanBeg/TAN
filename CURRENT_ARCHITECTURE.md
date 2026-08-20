# TAN — Güncel Mimari

*Bu belge 2026-08-20 itibariyle repo'daki kaynağın gerçek durumunu yansıtır.
Go referans motoru bu tarihte TAMAMEN kaldırıldı — repo artık %100 self-hosted.*

*(Eski Go-dönemi mimarisi git geçmişinde ve tarihsel BOOTSTRAP*/SURUM belgelerinde
durur; bu belge onların yerini alır.)*

---

## 1. Genel Bakış

Tan, Türkçe anahtar kelimeli, kendi assembler'ı ve kendi linker'ı ile sıfır dış
bağımlılıkla native x86-64 ELF üreten, **%100 kendi kendini derleyen** bir
programlama dilidir. Üretim zinciri tamamen Go'suzdur.

Çekirdek iki parça:
1. **`TancElf.tan`** (~3550 satır Tan kaynağı) — derleyicinin kendisi, Tan'da yazılı.
2. **`TancElf`** (~146 KB native ELF ikilisi) — commit'li bootstrap tohumu.

`TancElf`, `TancElf.tan`'ı derleyip kendini bayt-birebir yeniden üretir (sabit
nokta). Zincirde Go, gcc, clang, as, ld — hiçbiri yok.

## 2. Dosya Envanteri

### 2.1 Derleyici (Tan)
| Dosya | Rol |
|---|---|
| `TancElf.tan` | Ana self-hosting derleyici — Tan → x86-64 ELF, doğrudan |
| `Tanc.tan`, `Tanc2.tan`, `TancAsm.tan` | Erken Tan-derleyici denemeleri (tarihsel) |

### 2.2 Commit'li native ikililer
| Dosya | Rol |
|---|---|
| `TancElf` | Go'suz bootstrap tohumu (~146 KB, statik) |
| `gen1/gen2/gen3` | Bootstrap nesilleri (sabit-nokta ürünleri, gen2==gen3) |

### 2.3 Tooling (Tan, `araclar/`)
Go host'un sağladığı tooling 2026-08-20'de silindi; Tan'da yeniden yazılıyor:
| Dosya | Rol |
|---|---|
| `araclar/simgeler.tan` | Sembol listeleyici (işlev/değişken, düz + --json) |
| `araclar/denetle.tan` | Blok dengesi linter'ı (eksik/fazla 'son', --json) |
| `araclar/bicimlendir.tan` | Kanonik formatter (deterministik, idempotent) |
| `kutuphane/tanoku.tan` | Ortak tokenize/tarama yardımcıları (hepsi `içe al` eder) |

## 3. Derleme Zinciri (Pipeline)

`TancElf.tan` AST üretmez — düz **RPN "bant" (flat postfix liste)** kullanır:

```
kaynak → oku(girdiYolu) → tokenle (SOH-ayraçlı düz metin listesi)
       → tokenGenislet (içe al statik açılımı, döngü korumalı)
       → tipSabitNoktasiTara (dönüş + parametre tipleri ortak sabit nokta)
       → blokDerle → deyimDerle → ifadeAyristir (RPN bant üretir)
       → ifadeBantYaz (RPN → makine kodu bandı)
       → bagla (iki geçişli linker: BAYT/ETIKET/REL32)
       → elfUret (ELF64 başlık + program başlığı + kod)
       → yazBaytlar(ciktiYolu)
```

- Segment tabanı `0x400000`, tek PT_LOAD (R+W+X), ELF başlığı elle kodlanır
  (`e_machine=62` x86-64, `e_type=2` ET_EXEC).
- Bant kaydı: `"OP<SOH>a1<SOH>a2"` (SOH=karakter(1)). `bagla()` çözümlenemeyen
  etikette `BAGLAMA HATASI: <etiket>` verip exit(1).
- Builtin çağrı: her çağrı `f_<ad>` etiketine gider (runtime routine adları
  KÜÇÜK harf — `argsay` doğru, `argSay` değil).

## 4. Derleyici Bileşenleri (TancElf.tan içinde)

- **Lexer:** `tokenle(kaynak)` (satır ~96) — token'lar `"TUR<SOH>DEGER<SOH>SATIR"`.
  Token türleri: SATIRSONU, METIN, SAYI, ANAHTAR, AD, PARAC, PARKAPA, KOSAC,
  KOSKAPA, SUSAC, SUSKAPA, IKINOKTA, VIRGUL, ISLEC, DOSYASONU.
- **Parser:** `ifadeAyristir` (satır ~408) — RPN bant. Öncelik: mantıksal → bitsel
  → karşılaştırma → kaydırma → toplama → çarpma → sonEk → temel. `ve/veya` kısa
  devre `kisaDevreBirlestir` ile bant içine jcc/jmp gömerek.
- **Tip analizi:** `tipSabitNoktasiTara` (dönüş+parametre ortak sabit nokta).
  Metin literal + metin-kanıtlı değişken/çağrı dönüşü tanınır.
- **Kod üretimi:** `ifadeBantYaz`, `deyimDerle`, `islevDerle`. Sistem V AMD64
  konvansiyonu, ≤6 parametre (rdi/rsi/rdx/rcx/r8/r9). Yerel değişkenler
  `rbp-8*(i+1)`. REX/ModRM elle. SSE (ondalık), atomik (`lock cmpxchg/xadd`).
- **Runtime helper'lar:** derlenen her programa gömülür — `f_tan_ayir` (brk bump
  allocator), `f_metin_birlestir`, `f_metin_esit`, `f_metin_indeks`, `f_liste_*`,
  `f_kod`, `f_karakter`, `f_oku`, `f_yazBaytlar`, `f_argsay`, `f_arg`,
  `f_sozluk_*`, dosya G/Ç (dosyaAc/Oku/Yaz/Kapat), eşzamanlılık primitifleri.
- Metin gösterimi: `[uzunluk:8][bayt...]` işaretçi; sabitler veri bölümünde.

## 5. Modül Sistemi (`içe al`)

`içe al "yol"` — `tokenGenislet` + `iceAlYoluCoz` ile **derleme-zamanı statik
açılım**, döngü korumalı (`dosyaVarMi`). Yol cwd-göreli çözülür (araçlar `tan/`
dizininden derlenmeli). Tooling bunu kullanıyor (`araclar/*` → `kutuphane/tanoku.tan`).

## 6. Yerleşikler (doğrulanmış)

`oku(yol)`, `arg(i)`, `argsay()`, `uzunluk`, `metinAl(s,i)` (BAYT-bazlı),
`metinBirlestir`, `metin(sayı)`, `karakter(kod)`, `kod(karakter)`, `ekle(liste,x)`
(YENİ liste döndürür), `yaz`, `metinEsit(a,b)` (çok-karakterli metin karşılaştırma
— `==` DEĞİL). Liste `l[i]`, `l[i][j]`, atama `l[i]=v`. Escape'ler: `\n \t \" \\`.

## 7. Bilinen Sınırlar

- `dene/yakala` codegen'de YOK (lexer ayrıştırır, net derleme hatası verir).
- `kayıt` (record) self-hosted TancElf'te YOK — işlev + değişken.
- x86-64 Linux only; DWARF yok; codegen naive (register allocator/DCE yok).
- UTF-8: `metinAl`/`kod`/indeksleme BAYT-bazlı — çok-baytlı Türkçe harf dikkat.
- Toolchain hâlâ Tan'da yeniden inşa ediliyor (Go silindi): LSP, test koşucusu,
  structured diagnostics, paket yöneticisi, api/bağımlılık — yol haritasında.

## 8. Test ve Doğrulama

`go test` YOK. Go-free doğrulama script'leri: `TestArkaUcGoSuzTemiz.sh`
(3 nesil sabit nokta + temiz ortam, go/gcc/as/ld tuzağı), `BootstrapGoSuz.sh`,
`KanitGoSuzTarihce.sh` (git geçmişinden Go'suz yeniden inşa).
