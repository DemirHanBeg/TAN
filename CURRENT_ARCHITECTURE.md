# TAN — Güncel Mimari

*Bu belge repository'deki kaynak kodun doğrudan analizine dayanır (2026-08-13). Kanıt bulunmayan hiçbir iddia içermez; kanıt bulunmayan noktalar "BİLİNMİYOR" olarak işaretlenmiştir.*

---

## 1. Genel Bakış

Tan, Türkçe anahtar kelimeli, kendi assembler'ı ve kendi linker'ı ile sıfır dış bağımlılıkla native x86-64 ELF üreten, kendi kendini derleyen (self-hosting) bir programlama dili projesidir.

Depo iki katmandan oluşur:

1. **Go referans/seed motoru** (20 Go dosyası): lexer, parser, AST optimizer, ağaç-gezen yorumlayıcı, bytecode VM, ELF arka ucu, paket yöneticisi, FFI.
2. **Kendi kendini derleyen Tan portu**: `TancElf.tan` (3552 satır) + commit'li `TancElf` native ikilisi. Go'dan tamamen bağımsız üretim zincirinin kalbidir.

Aynı Tan kaynağı **4 farklı yoldan** çalıştırılabilir (DOKUMANTASYON.md "Arka Uçlar" bölümü):

| Yol | Komut | Üretici | Dış araç | Kapsam |
|---|---|---|---|---|
| Yorumlayıcı + VM | `tan program.tan` | Go | Go runtime | Tam dil |
| Tan → C → gcc | `tan derle` | Go (`arsiv/DerleC.go`) | gcc, libc | **ARŞİVLENDİ** |
| Tan → asm → as/ld | `tan asm` | Go (`arsiv/DerleAsm.go`) | binutils | **ARŞİVLENDİ** |
| Tan → makine kodu → ELF | `tan elf` | Go (`DerleElf.go`) **veya** Tan (`TancElf.tan`) | hiçbiri | int64, ondalık, metin, liste, dosya G/Ç |

Not: `arsiv/` dizini commit `61594c6`'da (2026-08-09) çalışma ağacından silinmiştir; yalnız git geçmişinde durur. `tan derle` ve `tan asm` komutları bugün arşiv konumunu gösteren bir mesajla çıkar (MainNative.go:55-69).

---

## 2. Dosya Envanteri (kanıt)

### 2.1 Go dosyaları (28 adet, ~15.400 satır)

| Dosya | Satır | Rol |
|---|---|---|
| `MainNative.go` | 128 | Native giriş noktası; komut dağıtıcı (REPL, paketle, paket, test, biçimlendir, denetle, derle, simgeler, bagimlilik, api, elf, asm, dosya çalıştırma) |
| `MainWasm.go` | 62 | WASM giriş noktası; `tanCalistir` JS global'ini açar |
| `Lexer.go` | 244 | Tokenizer; token türleri, Türkçe anahtar kelime tablosu |
| `Parser.go` | 719 | Özyinelemeli inişli ayrıştırıcı; AST düğümleri (Dugum) |
| `Optimize.go` | 343 | AST seviyesi optimizasyon (sabit katlama, cebirsel sadeleştirme, ölü kod eleme, sabit koşul düzleme) |
| `Sayi.go` | 171 | int64/float64 sayı sistemi (`/` her zaman float64; `tamBol` kesen bölme) |
| `SabitTam.go` | 161 | Sabit genişlikli tam sayılar (u8/u16/u32/u64/i8/i16/i32/i64, `TanSabitTam`) |
| `Hata.go` | 118 | `TanHata` struct + `firlat()` (panic/recover tabanlı hata) |
| `Diyagnostik.go` | 378 | Tanı mesajları (TANxxxx kod tablosu, öneriler) |
| `Yorumlayici.go` | 1022 | Ağaç-gezen yorumlayıcı; `Deger` evrensel tipi; kapsam sistemi; `kaynagiCalistir` |
| `Derleyici.go` | 302 | AST → bytecode derleyici (VM için); `vmDesteklemiyor` düşüş sinyali |
| `VM.go` | 313 | Yığın-tabanlı bytecode sanal makine |
| `DerleElf.go` | 6001 | Kademe 3+4: kendi assembler'ı + linker'ı; AST → x86-64 makine kodu → ELF64 |
| `Derle.go` | 80 | `tan derle` sarmalayıcısı (ELF arka ucuna; eski C/asm yolları `arsiv/`'de) |
| `Yerlesik.go` | 1427 | Yerleşik işlev kaydı (112 kayıtlı isim) |
| `YerlesikFFI_linux.go` | 103 | purego tabanlı FFI (`disKutuphaneAc/Bul/Cagir/Kapat`), yalnız linux |
| `YerlesikFFI_diger.go` | 28 | FFI stub'ları (linux dışında hata fırlatır) |
| `Modul.go` | 151 | Modül arama sırası, `tan.json` giriş çözümleme |
| `Paket.go` | 899 | Paket yöneticisi (`yeni/ekle/sil/indir/kur/doğrula/liste/önbellek`) |
| `Paketle.go` | 94 | Tek-dosya paketleme (`TANPAKET1` sihirli kuyruk) |
| `Test.go` | 332 | `tan test` koşucusu (`test` blokları, test adı çıkarımı, --liste/--json/--ayrinti) |
| `Bicimlendir.go` | 606 | `tan biçimlendir` biçimleyici (--denet/--cikti) |
| `Denetle.go` | 626 | `tan denetle` statik denetçi (gölgeleme, ölü değişken, çift tanım); `guvenliTokenle`/`guvenliAyristir` (panic koruması) |
| `Simgeler.go` | 299 | `tan simgeler` sembol envanteri (işlev/kayıt/alan/metot/değişken + konum) |
| `Bagimlilik.go` | 356 | `tan bagimlilik` `içe al` grafiği (çözümleme, ters bağımlılıklar, döngü tespiti) |
| `Api.go` | 407 | `tan api` tür-çıkarımlı API yüzeyi (DerleElf tip çıkarımını yeniden kullanır) |
| `Kopru.go` | 66 | Köprü iskeleti — şu an BOŞ (yetenek haritası sökülmüş) |
| `Cikti.go` | 10 | `Cikti io.Writer` (stdout; WASM'de tampona yönlendirilir) |

### 2.2 Tan dilinde compiler'lar

| Dosya | Satır | Durum |
|---|---|---|
| `TancElf.tan` | 3552 | **Ana self-hosting derleyici** — doğrudan ELF üretir, 147 kullanıcı işlevi |
| `Tanc.tan` | 135 | İlk deneme: Tan → C → gcc (yalnız atama + yaz) |
| `TancAsm.tan` | 270 | Tan → x86-64 asm → as/ld (yalnız atama + yaz + aritmetik) |
| `Tanc2.tan` | 365 | TancAsm + kontrol akışı (`eger`/`iken`) + karşılaştırma |

### 2.3 Commit'li native ikililer

| Dosya | Boyut | Rol |
|---|---|---|
| `TancElf` | 114.998 bayt | Go'suz bootstrap tohumu (elf backend) |
| `gen1` | 142.944 bayt | Önceki nesil ara derleyici (bootstrap ara ürünü) |
| `gen2` | 114.998 bayt | Sabit nokta ürünü |
| `gen3` | 114.998 bayt | Sabit nokta ürünü (gen2 ile birebir) |
| `TancElfYaz23` | 317 bayt | ELF yazıcısının minimal smoke testi (2+3=5 yazdırır) |
| `tan` / `tan_linux` | 11.203.246 bayt | Go derleyici + yorumlayıcı + VM (Linux ELF) |
| `tan.exe` | 11.026.432 bayt | Go derleyici + yorumlayıcı + VM (Windows PE) |

---

## 3. Derleme Zinciri (Pipeline)

### 3.1 Go yolu (`tan elf`, DerleElf.go)

```
kaynak → os.ReadFile → YeniLexer.Tokenle → YeniParser.Ayristir (AST)
       → elfIceAlGenislet (içe al statik açılımı)
       → YeniOptimizer().Govde (sabit katlama vb.)
       → islevler/kayitTanimlari/anaGovde sınıflandırması
       → elfUlasilabilirIslevler (ölü işlev budaması)
       → 50 turlu sabit-nokta tip çıkarımı
           (govdeTipleriniTopla + sozlukElemanlariniCoz + kayitAlanTipleriniCoz
            + dondurTipi + parametreTipleriniOgren)
       → runtime helper'lar + kullanıcı işlevleri + _start emisyonu
       → rel32/veri fixup çözümü
       → ELF64 başlığı + 1 PT_LOAD program başlığı (elle yazılır)
       → isteğe bağlı .symtab/.strtab/.shstrtab (TAN_SEMBOLSUZ yoksa)
       → os.WriteFile(cikti, 0755)
```

- Segment tabanı: `elfTaban = 0x400000`; başlık boyu `64 + 56 = 120` bayt.
- Tek PT_LOAD segmenti: `p_flags = 7` (R+W+X), `p_vaddr = 0x400000`, hizalama 4096.
- ELF başlığı elle kodlanır: `e_machine = 62` (x86-64), `e_type = 2` (ET_EXEC).

### 3.2 Self-hosted yol (`TancElf` / `TancElf.tan`)

Go yolundan farkı: AST üretilmez, **RPN "bant" (flat postfix liste)** kullanılır.

```
kaynak → oku(girdiYolu) → tokenle (SOH-ayraçlı düz metin listesi)
       → tokenGenislet (içe al statik açılımı, döngü korumasıyla)
       → tipSabitNoktasiTara (dönüş + parametre tipleri ORTAK sabit nokta, ≤200 tur)
       → blokDerle → deyimDerle → ifadeAyristir (RPN bant üretir)
       → ifadeBantYaz (RPN → makine kodu bandı)
       → bagla (iki geçişli linker: BAYT/ETIKET/REL32)
       → elfUret (ELF64 başlık + program başlığı + kod)
       → yazBaytlar(ciktiYolu)
```

Bant kayıt biçimi: her öğe `"OP<SOH>a1<SOH>a2"` (SOH = karakter(1)). Üç bant giriş türü: `BAYT` (1 bayt), `ETIKET` (etiket tanımı), `REL32` (doldurulacak 4 bayt rel32). `bagla()` çözümlenemeyen etikette `BAGLAMA HATASI: etiket bulunamadi` verip exit(1) yapar.

### 3.3 Yorumlayıcı yolu (`tan program.tan`)

```
kaynak → YeniLexer → YeniParser.Ayristir
       → vmDeneCalistir(agac)     # önce bytecode VM dene
           ├─ başarılı → VM çalışır, biter
           └─ vmDesteklemiyor{} panic'i → y.guvenliCalistir(agac)  # ağaç-gezene düş
```

VM yalnızca sayı/metin/mantık literalleri, değişken, aritmetik, karşılaştırma, bit işlemleri, `eğer`/`iken`, `yaz` ve kullanıcı işlevlerini destekler; yerleşik işlev çağrısı, liste, sözlük, kayıt, `her`, `dur`/`devam`, `içe al`, `dene`, indeksleme içeren her program VM'de `vmDesteklemiyor{}` panikler ve otomatik ağaç-gezene düşer.

### 3.4 WASM yolu

`GOOS=js GOARCH=wasm go build -o web/tan.wasm .` → `MainWasm.go` derlenir; `tanCalistir(kaynak)` JS global'ini kurar, VM'i dener, düşerse yorumlayıcıyı çalıştırır, çıktıyı tampona alır ve döndürür. `tan.wasm` repo'da DEĞİL (README ve web/BENIOKU.txt: ~9.9 MB, ayrı derlenir). README: "WASM build has not been tested in a real browser."

---

## 4. Derleyici Bileşenleri (istem 1-7'nin yanıtları)

### 4.1 Giriş noktası
- **Native (Go)**: `MainNative.go:12` `func main()`. Sırası: gömülü program kontrolü → REPL → `paketle` → `paket` → `elf` → `asm` (arşiv mesajı) → `derle` (arşiv mesajı) → dosya çalıştırma.
- **ELF backend**: `DerleElf.go:5441` `func derleElf(dosya, cikti string)`.
- **Self-hosted (Tan)**: `TancElf.tan` sürücüsü satır 3489-3552 (üst-seviye deyimler; `arg(1)`/`arg(2)` ile girdi/çıktı yolunu okur).
- **WASM**: `MainWasm.go:53` `func main()`.
- **TancElf.tan ürettiği programlarda**: `_start` etiketi; `argvYakala` → `yiginIlkle` → üst-seviye deyimler → `mov rax,60; xor rdi,rdi; syscall` (exit 0).

### 4.2 Lexer
- **Go**: `Lexer.go` `YeniLexer(kaynak).Tokenle()` → rune bazlı, 15 token türü (T_SON_DOSYA … T_NOKTA), `#` yorum, `\n` deyim ayırıcı, çok-karakterli işleçler (`>= <= == != << >>`), `unicode.IsLetter` Türkçe harfleri de tanır.
- **Self-hosted**: `TancElf.tan` `tokenle(kaynak)` (satır 96) — tokenlar `"TUR<SOH>DEGER<SOH>SATIR"` biçiminde düz metin listesi; `tamponOlustur/tamponEkle` (amortize O(1)) ile O(n²) ekleme deseninden kaçınılmış. Token türleri: `SATIRSONU, METIN, SAYI, ANAHTAR, AD, PARAC, PARKAPA, KOSAC, KOSKAPA, SUSAC, SUSKAPA, IKINOKTA, VIRGUL, ISLEC, DOSYASONU`.

### 4.3 Parser
- **Go**: `Parser.go` `YeniParser(tokenlar).Ayristir()` → `[]Dugum`. Özyinelemeli iniş, öncelik katmanları: `mantiksal (ve/veya)` → `bitsel (&|^)` → `karsilastirma` → `kaydirma (<<>>)` → `toplama (+-)` → `carpma (*/%)` → `sonEk` → `birincil` → `temel`. `kayıt`, `dene/yakala`, `köprü`, zincirli indeks/alan/metot çağrıları, dolaylı çağrı (`CagriIfadeDugum`) desteklenir.
- **Self-hosted**: `TancElf.tan` BÖLÜM 2 `ifadeAyristir` (satır 408) → RPN bant üretir (AST değil). Aynı öncelik katmanları; `ve/veya` kısa devresi `kisaDevreBirlestir` (satır 443) ile bant içine jcc/jmp/etiket gömerek yapılır.

### 4.4 AST
- **Go**: `Parser.go` satır 9-165. Düğümler (Dugum): `SayiDugum, MetinDugum, MantikDugum, YokDugum, DegiskenDugum, IkiliDugum, AtamaDugum, YazDugum, EgerDugum, IkenDugum, HerDugum, IslevDugum, CagriDugum, DondurDugum, DurDugum, DevamDugum, IceAlDugum, DeneDugum, KopruDugum, ListeDugum, SozlukDugum, IndeksDugum, IndeksAtamaDugum, KayitTanimDugum, KayitOlusturDugum, AlanErisimDugum, AlanAtamaDugum, MetotCagriDugum, CagriIfadeDugum`.
- **Self-hosted**: AST yok; düz RPN bant. Bu bilinçli bir tasarım farkıdır.

### 4.5 Semantic / tip analizi
- **Go**: `DerleElf.go` sabit-nokta tip çıkarımı: `tipCikar` (satır 783), `govdeTipleriniTopla` (920), `sozlukElemanlariniCoz` (988), `kayitAlanTipleriniCoz` (1099), `dondurTipi` (1121), `parametreTipleriniOgren` (1159). Tipler: `CTam, CMetin, CListe, CKesir, CSozluk, CKayit`. `/` işleci her zaman `CKesir` döner.
- **Self-hosted**: kısıtlı. `tipSabitNoktasiTara` (dönüş + parametre tipleri ORTAK sabit nokta, ≤200 tur; Faz 4) → 5 elemanlı `islevTipKutu` `[dönüş-adları, dönüş-tipleri, parametre-adları, parametre-tipleri]` + Faz 5 (A) ölü budama için `[erişilebilir-adlar]`. Metin literal / bilinen metin-döndüren çağrının DOĞRUDAN sonucu + FAZ 4 itibariyle metin-kanıtlı DEĞİŞKEN dönüşü (`metinDegiskenMi`) tanınır.
- **Yorumlayıcı**: dinamik tipler, `Deger interface{}`.

### 4.6 Kod üretimi
- **Go**: `DerleElf.go` `ifade` (1313), `deyim` (2480), `islevYaz` (2774). REX/ModRM elle kodlanır; SSE (ondalık), atomikler (`lock cmpxchg/xadd`), `clone`+futex (native eşzamanlılık) desteklenir. Naive — register allocator yok, DCE yok.
- **Self-hosted**: `TancElf.tan` BÖLÜM 4: `ifadeBantYaz` (2896), `deyimDerle` (775), `islevDerle` (3341). Sistem V AMD64 çağrı konvansiyonu, en fazla 6 parametre (rdi/rsi/rdx/rcx/r8/r9). Yerel değişkenler `rbp-8*(i+1)` yuvalarında.

### 4.7 Runtime
- **Go/ELF**: derlenen her programa gömülen runtime helper'ları (DerleElf.go 2878-5281): `f_tan_ayir` (brk tabanlı bump allocator, sınır kontrolü + `exit(3)`), `f_arena_ayir/serbest` (boyut sınıflı free list, ≤512 bayt), `f_bellek_kopyala`, `f_sayi_metne`, `f_kesir_metne`, `f_metin_birlestir`, `f_metin_esit`, `f_liste_yap/ekle`, `f_metin_indeks`, `f_kod`, `f_karakter`, `f_oku`, `f_yaz_dosya`, `f_parcala`, `f_birlestir_*`, `f_sozluk_*` (256 kovalı ayrık zincirleme), `f_rastgele`, `f_e_ussu`, `f_log`, `f_yuvarla`, fd-tabanlı dosya G/Ç, `f_ham_*` ham bellek, `f_futex_*`/`f_kilit_*`/`f_iplik_*` eşzamanlılık, `f_arg`/`f_argsay`.
- **Yorumlayıcı**: `Deger` değerleri; `Kapsam` sözlük tabanlı kapsamlar; `islevSiniri` ile işlev sınırı (işlev yereli küreseli ezemez).
- Metin gösterimi: `[uzunluk:8][bayt...]` (işaretçi); sabitler `.rodata`/veri bölümünde.

---

## 5. Modül Sistemi (`içe al`)

- **Yorumlayıcı**: `Yorumlayici.iceAl` (Yorumlayici.go:276-308) — `modulAra` ile 7 basamaklı arama (aşağıda), mutlak yola çevirir, `alinanlar` haritasıyla döngüsel/tekrar içe almayı engeller (en fazla bir kez yüklenir), içe alan dosyanın dizinini `kaynakDizin` yapar, tüm üst-seviye deyimleri paylaşılan global kapsamda çalıştırır.
- **Self-hosted**: `TancElf.tan` `tokenGenislet` (3435) + `iceAlYoluCoz` (3419) — token seviyesinde statik açılım, döngü korumalı. `dosyaVarMi` (TancElf.tan satır 2303) kullanır.
- **Arama sırası** (Modul.go `modulAra` satır 39-106):
  1. `.tan` ile bitiyorsa: `kaynakDizin/ad`, sonra cwd
  2. `kaynakDizin/<ad>.tan`
  3. `kaynakDizin/kutuphane/<ad>.tan`
  4. `tan_moduller/<ad>` (tan.json `giris` öncelikli), `tan_moduller/<ad>/<ad>.tan`
  5. `$TAN_YOL` (her parça için `<parça>/<ad>.tan`, `<parça>/<ad>/<ad>.tan`)
  6. `~/.tan/moduller/<ad>`
  7. exe yanındaki `kutuphane/<ad>.tan`, `kutuphane/` yoksa `<exe dizini>/<ad>.tan`
  8. cwd `kutuphane/<ad>.tan`

**Not (elf yolu)**: README'ye göre `içe al` artık TancElf.tan'da da çalışıyor (SURUM.md ADIM 3: "içe al zaten kapalıydı"); Go yorumlayıcısındaki gibi çalışma-zamanı değil, derleme-zamanı statik açılımdır.

---

## 6. CLI (MainNative.go)

| Komut | Davranış |
|---|---|
| `tan` (argümansız) | REPL |
| `tan program.tan` | Yorumlayıcı (+ otomatik VM düşüşü) |
| `tan elf prog.tan çıktı` | Go'nun ELF arka ucu; sıfır dış araç |
| `tan derle prog.tan [-o çıktı]` | ELF derleyicisi (Derle.go; eski C/asm yolları `arsiv/`'de donuk) |
| `tan test [--liste\|--json\|--ayrinti] <dosya\|dizin>` | `test` bloğu koşucusu (Test.go) |
| `tan biçimlendir [--denet\|--cikti] <dosya...>` | biçimleyici (Bicimlendir.go) |
| `tan denetle [--json] <dosya...>` | statik denetçi (Denetle.go) |
| `tan simgeler [--json] <dosya...>` | sembol envanteri (Simgeler.go) |
| `tan bagimlilik [--json] <dosya\|dizin...>` | `içe al` grafiği + döngü tespiti (Bagimlilik.go) |
| `tan api [--json] <dosya...>` | tür-çıkarımlı API yüzeyi (Api.go) |
| `tan paket yeni/ekle/sil/indir/kur/doğrula/liste/önbellek` | Paket yöneticisi (Paket.go) |
| `tan paketle prog.tan çıktı` | Tek-dosya paketleme (motor+kaynak+sihir) |
| `tan asm prog.tan çıktı` | ARŞİVLENDİ — mesaj basar, exit(1) |
| `tan` (paketlenmiş ikili) | Gömülü programı çalıştırır |

## 7. Ortam Değişkenleri

| Değişken | Etki |
|---|---|
| `TAN_SEMBOLSUZ=1` | ELF çıktısına `.symtab/.strtab/.shstrtab` yazmaz (nm/gdb sembolsüz) |
| `TAN_OPT_RAPOR=1` | Optimize raporu (katlanan/silinen düğüm sayısı) |
| `TAN_TIP_DEBUG` | Tip çıkarım turu günlüğü (DerleElf.go) |
| `TAN_YOL` | Modül arama yolu ekler |

---

## 8. Bilinen Sınırlar (kanıtlı)

- **`elf` arka ucu**: sözlük/`dene`/`yakala` — Go `DerleElf.go` yolunda sözlük var ama README'ye göre `içe al` ve `dene/yakala` interpreter-only'dır (eskiden); SURUM.md ADIM 3'e göre `içe al` self-hosting'e taşındı, `dene/yakala` BACKLOG'ta.
- **`TancElf.tan`** (SURUM.md "bilinen eksikler"): (1) ondalık sayı literalleri hiç desteklenmiyor (kasıtlı, net derleme hatası), (2) `yazDosya`/`yaz_dosya` kullanıcı programından çağrılamıyor, (3) `her x ["bir","iki"] içinde` döngüsünde döngü değişkeni tipi yanlış çıkarılıyor.
- x86-64 Linux only; DWARF yok; codegen naive; WASM gerçek tarayıcıda test edilmemiş.
- UTF-8 rune-vs-bayt: `kod()`/`karakter()`/metin indeksleme elf yolunda BAYT bazlı, yorumlayıcıda RUNE bazlı — çok baytlı Türkçe harfler farklı bayt üretebilir (örnekler bilinçli ASCII).
- `arsiv/`'deki `derle`/`asm` arka uçları donmuş durumda (git geçmişinde).

---

## 9. Test ve Doğrulama Mimarisi

- `_test.go` / `go test` YOKTUR (CLAUDE.md). Doğrulama bash script'leriyle yapılır: `TestArkaUc.sh`, `FarkTesti.sh`, `GercekProgramlar.sh`, `Bootstrap.sh`, `BootstrapGoSuz.sh`, `KanitGoSuzTarihce.sh`, `TestArkaUcGoSuzTemiz.sh`, `Olcum.sh`.
- Geliştirici komut regresyonları (her komut için ayrı script): `TestTest.sh` (17), `TestDiyagnostik.sh` (12), `TestBicimlendir.sh` (20), `TestDenetle.sh` (26), `TestPaket.sh` (51), `TestDerle.sh` (29), `TestSimgeler.sh` (16), `TestBagimlilik.sh` (17), `TestApi.sh` (12). Hepsi `=== SONUC: N gecti, 0 kaldi ===` ile biter, kalan 0 ise exit 0.
- 20/20 ELF regresyon testi (TestArkaUc.sh), 2 ayaklı sabit nokta (gen2==gen3), 3 ayaklı doğrulama (tuzak + sabit nokta + anlam korunumu), çapraz-kontrol (`FarkTesti.sh` yorumlayıcı vs native TancElf bayt-birebir), temiz ortam testi (go/gcc/as/ld tuzağı).
- Detaylar için `EXISTING_TOOLING.md`'ye bakın.
