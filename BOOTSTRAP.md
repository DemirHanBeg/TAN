> ⚠️ **TARİHSEL — Go dönemi kaydı.** Go referans motoru 2026-08-20'de TAMAMEN kaldırıldı; repo artık %100 self-hosted, sıfır Go. Bu belge o dönemin mimarisini/planını anlatır. Güncel mimari için **CURRENT_ARCHITECTURE.md**. Buradaki "go build", ".go dosyası", "yorumlayıcı/VM" atıfları ARTIK GEÇERLİ DEĞİL.

# TAN Bootstrap Analizi — Go → TAN Gen1 → Gen2 → Gen3 → TAN Derleyici

*Bu belge repository'deki kaynak, script ve git geçmişinin doğrudan analizine dayanır (2026-08-13). Kanıt bulunmayan hiçbir iddia içermez; kanıt bulunmayan noktalar "BİLİNMİYOR" olarak işaretlenmiştir. Tüm satır numaraları çalışma ağacındaki güncel dosyalara aittir.*

---

## 1. Yönetici Özeti

**Ana bulgu:** İstenen zincir `Go → TAN Gen1 → TAN Gen2 → TAN Gen3 → TAN compiler` **zaten kurulu ve sabit noktada doğrulanmış durumda.** Üretim zinciri Go'dan tamamen arındırılmıştır; Go'nun tarihsel TEK üretim rolü `go build -o tan .` + `./tan elf TancElf.tan gen1` idi (SURUM.md satır 20-21, 31-38).

| Zincir adımı | Artifact | Boyut | Kanıt |
|---|---|---|---|
| Go (tarihsel seed üreticisi) | `go build -o tan .` → `./tan elf TancElf.tan gen1` | — | SURUM.md:20-21 |
| TAN Gen1 | `gen1` = `./TancElf TancElf.tan gen1` | 142.944 B | BootstrapGoSuz.sh:38-40 |
| TAN Gen2 | `gen2` = `./gen1 TancElf.tan gen2` | 114.998 B | BootstrapGoSuz.sh:42-44 |
| TAN Gen3 | `gen3` = `./gen2 TancElf.tan gen3` | 114.998 B | BootstrapGoSuz.sh:46-48 |
| Sabit nokta | `cmp gen2 gen3` (SESSİZ) | birebir | BootstrapGoSuz.sh:51 |
| TAN → TAN compiler | `cp gen2 TancElf` → yeni tohum | 114.998 B | BootstrapGoSuz.sh:69-71 |

**Kalan iş compiler'ı "tamamen taşımak" değil:** `TancElf.tan` zaten Go'nun üretim `elf` yolunun çekirdeğini kapsıyor. Faz 1-5 ile optimizer (sabit katlama), `mantık` tip ayrımı, liste eleman tipi, dönüş tipi izleme ve ölü işlev/yardımcı budaması taşındı. Taşınmamış kısımlar: tam tip çıkarımı, ondalık/SSE, sözlük/kayıt runtime'ı, dosya konumlamalı G/Ç, native eşzamanlılık, hata mekanizması. Bu belge bunların her birinin durumunu ve taşınma sırasını verir.

---

## 2. Zincirin Bugünkü Hali

### 2.1 Üretim zinciri (`BootstrapGoSuz.sh`, Go'suz)
```
TancElf (commit'li seed, 114.998 B)
  → ./TancElf TancElf.tan gen1        (142.944 B)
  → ./gen1    TancElf.tan gen2        (114.998 B)
  → ./gen2    TancElf.tan gen3        (114.998 B)
  → cmp gen2 gen3                     (sabit nokta; farklıysa exit 1)
```
Tek dış gereksinim: commit'li `TancElf` ikilisi (BootstrapGoSuz.sh:28-32). Script go/gcc/as/ld tuzaklamaz; tuzağı `TestArkaUcGoSuzTemiz.sh` yapar (PATH'e sahte `go gcc clang as ld.ld gold` koyar, 22-36).

### 2.2 Tarihçe (git, `KanitGoSuzTarihce.sh:14-52`)
```
62d0fce → 086baeb → 9ffe0b5(*) → fd06ddc → d6b4fac(**) → 9723e7c → 4aecdf4 (=TancElf.tan HEAD)
```
- `62d0fce`: commit'li `TancElf` seed'inin girildiği nokta (blob `c808a012`, 115.320 B; çalışma ağacında yeniden üretilip 114.998 B olmuş — `git status` M).
- `9ffe0b5`: **10. self-hosting bug sınıfı** — "yeni yerleşik + aynı commit'te öz-kullanım". `dosyaVarMi` yerleşiği eklenip TancElf.tan'ın modül yolu çözücüsü hemen kullandığı için önceki nesil `f_dosyaVarMi` etiketini gömemez → `BAGLAMA HATASI`. Çözüm: **iki-adımlı shim** (öz-kullanım yok edilmiş ara kaynak derlenir, o ara ikili gerçek kaynağı derler; TancElf.tan asla değiştirilmez) — KanitGoSuzTarihce.sh:92-111.
- `d6b4fac`: **salınım commit'i** — parametre tipi çıkarımı henüz yokken boyutlar `109098 ↔ 114000 ↔ 109101` arası salınıyor. Kural: **en büyük (en tam) üretimi seç**; `9723e7c` o tohumdan sabit noktaya ulaşır.
- `4aecdf4`: TancElf.tan'a parametre tipi çıkarımı eklendi; salınım kapanıyor, sabit nokta 3x doğrulandı.

### 2.3 Doğrulama ayakları (SURUM.md:205-217)
- **Tuzak kontrolü:** gen2 boş/sıfır-kod ikili üretmiyor (gerçek programlar derler).
- **Ayak (a) sabit nokta:** `cmp gen2 gen3` sessiz.
- **Ayak (b) gerçek derleyici mi:** çok özellikli programlar (işlev, özyineleme, liste, metin, ondalık, dosya G/Ç) doğru çalışıyor.
- **Ayak (c) anlam korunumu:** gen2 çıktısı Go'nun `tan elf` çıktısıyla 28 dosyada karşılaştırılır; yalnız bilinen kapsam boşlukları (`içe al`, `dene/yakala`) ayrışır, ikisi de orada AÇIK hata verir.

---

## 3. Go Derleyici Bileşen Envanteri (Görev 1)

Derleme zinciri, 20 Go dosyasının `main` paketi. Üretim yolu `elf` (`MainNative.go` → `derleElf` DerleElf.go:5441); yardımcı yollar yorumlayıcı (`Yorumlayici.go`) ve VM (`VM.go` + `Derleyici.go`).

### 3.1 Ön uç (front end)
| Bileşen | Dosya | Satırlar | İşlevler |
|---|---|---|---|
| Lexer | Lexer.go | 230 | `YeniLexer`, `Tokenle`; 15 token türü, 24 Türkçe anahtar kelime (satır 34-44) |
| Parser (AST) | Parser.go | 699 | `YeniParser`, `Ayristir`; 25 `Dugum` düğüm tipi (satır 9-165); özyinelemeli iniş, 8 öncelik katmanı |
| Optimizer | Optimize.go | 343 | `YeniOptimizer`, `Govde`, `deyim`, `ifade`, `sabitKatla`, `cebirsel` (tam envanter aşağıda §9.2) |

### 3.2 Semantik (tip çıkarımı, DerleElf.go:695-1271)
| Bileşen | Satır | Rol |
|---|---|---|
| `Cesit`/`Tip`/`KayitSemasi` tipleri | 699-781 | `CTam CMetin CListe CKesir CSozluk CKayit` (ayrıca `CMantik/CYok`), kayıt sema |
| `tipCikar` | 783 | ifade başına tip ayrımcısı |
| `govdeTipleriniTopla` | 920 | deyim listesinde değişken/yerel tipleri |
| `sozlukElemanTipiBul`/`sozlukElemanlariniCoz` | 949/988 | global sözlük değer tipleri |
| `kayitOlusturSiteleriTopla`(+`...Deyim`) | 1009/1059 | kayıt oluşturma siteleri |
| `kayitAlanTipleriniCoz` | 1099 | kayıt alan tipleri |
| `dondurTipi` | 1121 | dönüş tipi |
| `parametreTipleriniOgren` | 1159 | çağrı-yerinden parametre tipleri |
| **50 turlu sabit nokta döngüsü** | 5551-5616 | `len(parametreTipi)` değişmediğinde kırılır (5613); per-işlev snapshot/restore |

### 3.3 Kod üretimi (DerleElf.go)
| Bileşen | Satır | Rol |
|---|---|---|
| `makineKodu` struct + ~60 emit fonksiyonu | 56-693 | x86-64 kodlayıcı: ModRM/REX, mov/add/sub/cmp/imul/idiv, SSE, lock cmpxchg/xadd, byte ops, rel32/lea, syscall |
| `ifade` | 1313 | ifade derleyicisi (~1166 satır; akümülatör tarzı, rax) |
| `deyim` | 2480 | deyim derleyicisi (atama, yaz, eğer, iken, her, dur/devam, döndür, çağrı, metot çağrısı, alan/indeks atama) |
| `islevYaz` | 2774 | prolog (push rbp / sub rsp 16 hizası), 6 parametreyi spill, epilog |
| `elfAd` | 2690 | Türkçe harf etiket eşlemesi |
| `elfUlasilabilirIslevler` + `elfCagrilanAdlariTopla` | 5410/5324 | **ölü işlev budaması** (BFS; metotlar muaf) |
| `derleElf` sürücü | 5441 | okuma→parse→`içe al`→optimize→ayrıştır→budama→tip→emit→layout→ELF64→symtab→yaz |
| `sembolTablosuEkle` | 5867 | .symtab/.strtab/.shstrtab (TAN_SEMBOLSUZ ile kapatılır) |

### 3.4 Runtime yardımcı emisyonu (`yardimci*`, DerleElf.go:2878-5287)
Aşağıdaki yardımcıların HER BİRİ `f_*` makine-kodu alt programını gömerek programa eklenir:
`f_hata_cik`, `f_hata_sifira_bolme`, `f_tan_ayir`, `f_arena_ayir`, `f_arena_serbest`, `f_bellek_kopyala`, `f_yaz_metin_deger`, `f_sayi_metne`, `f_metin_birlestir`, `f_liste_yap`, `f_liste_ekle`, `f_metin_indeks`, `f_metin_esit`, `f_harfler`, `f_metin_araligi`, `f_parcala`, `f_sozluk_hash/yap/koy/al/varmi/sil/anahtarlar`, `f_karakter`, `f_kod`, `f_oku`, `f_sayi`, `f_dosya_var_mi`, `f_yaz_baytlar`, `f_argsay`, `f_arg`, `f_yaz_dosya`, `f_ekle_dosya`, `f_dosya_ac_*` (10: okuyaz/oku/yaz/konumla/oku_blok/yaz_blok/oku_blok_ham/yaz_blok_ham/senkron/kapat/sil), `f_futex_wait/wake`, `f_iplik_cikis/bekle`, `f_kilit_al/birak`, `f_atomik_ekle_ham`, `f_yuvarla`, `f_birlestir_metin/tam/kesir`, `f_rastgele`, `f_e_ussu`, `f_log`, `f_kesir_metne`, `f_yaz_kesir`, `f_yaz_metin`, `f_yaz_sayi`, `f_bellek_esle`, `f_bellek_coz`.

### 3.5 Yerleşik işlevler (Yerlesik.go)
112 kayıtlı isim (satır 40-698) + camelCase alias'ları (700-707) + sabit genişlikli tam sayı kurucuları `u8..i64` (723-730) + FFI (`YerlesikFFI_linux.go`: `disKutuphaneAc/Bul/Cagir/Kapat`). Yorumlayıcı tümünü çağırabilir; ELF yolu yalnız kendi `f_*` eşleşmesi olanları.

### 3.6 Modül/paket sistemi
`Modul.go` `modulAra` (39-106, 7 basamaklı arama), `Paket.go` paket komutları (29-54), `Paketle.go` tek-dosya paketleme (TANPAKET1, satır 16). ELF yolu `modulAra` KULLANMAZ — `elfIceAlGenislet` (5289) kendi statik açılımını yapar (`içe al` token akışında çözülür, `os.ReadFile`+`filepath.Abs/Dir` ile).

---

## 4. Bileşen-Bileşen TAN Karşılıkları (Görev 2)

`TancElf.tan` = 3552 satır, 147 `işlev`, 6 BÖLÜM. Go karşılığı dosya başlığında (TancElf.tan:3-4): `DerleElf.go + Lexer.go + Parser.go + Optimize.go`.

| # | Go bileşeni | TAN karşılığı (TancElf.tan) | Durum |
|---|---|---|---|
| 1 | Lexer (Lexer.go) | BÖLÜM 1: `tokenle`(96), `tokenKodla`(84), `anahtarMi`(58), `kacisCoz`(71) | ✅ taşındı |
| 2 | Parser → AST (Parser.go) | BÖLÜM 2: RPN "bant" üreticisi `ifadeAyristir`(408)+7 katman | ✅ **tasarım farkı**: AST değil düz postfix bant (TancElf.tan:229-231) |
| 3 | Optimizer (Optimize.go) | — (yok) | ❌ **tamamen eksik** (§9) |
| 4 | Tip çıkarımı (DerleElf.go:783-1271) | BÖLÜM 4 ön-geçişler: `tipSabitNoktasiTara` (dönüş + parametre ORTAK ≤200 tur sabit nokta; Faz 4), içinde `islevDonusTipleriBirGecis` + `metinDegiskenMi` + `parametreTipleriniTara`, `islevTipKutu` (5 elemanlı paralel liste: dönüş ad/tipler + parametre ad/tipler + Faz 5 (A) erişilebilir adlar) | ⚠️ kısmen: yalnız `tam`/`metin` ayrımı, sözlük/kayıt/liste eleman/bool yok |
| 5 | Kod üretimi `ifade` (1313) | BÖLÜM 4 `ifadeBantYaz`(2896) | ✅ alt küme için; sabit katlama YOK |
| 6 | Kod üretimi `deyim` (2480) | BÖLÜM 3b `deyimDerle`(775), `blokDerle`(753) | ✅ alt küme için |
| 7 | İşlev kod üretimi `islevYaz` (2774) | `islevDerle`(3341) | ✅ (6 parametre üst sınırı, fall-through 0) |
| 8 | Assembler `makineKodu` (56-693) | BÖLÜM 4 emit fonksiyonları (`modrmBayt`1182, `movImm64`1252, `ikiliKayit`1272, `bagla`1645...) | ✅ alt küme: SSE, lock, bitwise YOK |
| 9 | Runtime yardımcı emisyonu `yardimci*` (2878-5287) | BÖLÜM 4 bant kurucuları: `tanAyirBant`1839, `listeEkleBant`1896, `listeYapBant`1937, `sayiMetneBant`2010, `metinBirlestirBant`2088, `metinEsitBant`2128, `metinIndeksBant`2160, `kodBant`2182, `karakterBant`2191, `yazMetinBant`2211, `okuBant`2247, `dosyaVarMiBant`2303, `yazBaytlarBant`2342, `argsayBant`2416, `argBant`2423, `yazSayiBant`1764 | ⚠️ 17/50+ yardımcı taşındı (§6 tablo) |
| 10 | ELF yazıcı (5763-5819 + sembolTablosuEkle) | `elfUret`(1713) | ✅ (symtab desteği TancElf'ta BİLİNMİYOR) |
| 11 | Ölü işlev budaması `elfUlasilabilirIslevler` (5410) | Faz 5 (A): `ulasilabilirIslevleriTara` + `islevCagirdiklariniTopla` (BFS; kayıt metodu yok — muaf kuralı geçersiz); Faz 5 (B): `yardimciKullanimiTara` + `yardimciKapanisi` + `yardimciBagimliliklari` (yalnız kullanılan runtime yardımcıları gömülür) | ✅ YAPILDI (14 Ağu) |
| 12 | `içe al` açılımı `elfIceAlGenislet` (5289) | `tokenGenislet`(3435), `iceAlYoluCoz`(3419), `dosyaDizini`(3398), döngü korumalı | ✅ taşındı |
| 13 | Optimizer raporu `TAN_OPT_RAPOR` | — | ❌ yok |
| 14 | `dene`/`yakala` (yorumlayıcı) | — (anahtar tablosu hariç yok; TancElf hata → `yaz`+`sistemDur(1)`) | ❌ (Go elf'inde de YOK — BACKLOG) |
| 15 | `köprü`/FFI (Kopru.go, YerlesikFFI) | — | ❌ (Kopru.go iskeleti sökülmüş) |
| 16 | Paket yöneticisi `tan paket` (Paket.go) | — | ❌ (Go-only araç) |
| 17 | Yorumlayıcı + VM | — | ❌ (TancElf doğrudan ELF üretir; yorumlayıcı portu kapsam dışı) |

**Özet:** 17 Go üretim bileşeninin 8'i tam, 2'si kısmi, 7'si eksik. Eksiklerin büyük kısmı yorumlayıcı/paket/FFI gibi üretim dışı ya da sözlük/ondalık/eşzamanlılık gibi büyük runtime'lar.

---

## 5. TAN'da Eksik Dil Özellikleri (Görev 3)

`TancElf.tan`'ın derlediği dil alt kümesi (dosya başlığı TancElf.tan:10-17): tam sayı ifadeleri (`+ - * / %` + tüm karşılaştırmalar), değişken atama, `yaz`, `eğer/değilse/son`, `iken/son`, `işlev`+çağrı (≤6 parametre, özyineleme) + `döndür`, liste `[1,2,3]`+indeks+indeks ataması+`uzunluk`/`ekle`+iç içe liste, `her...içinde`+`dur`/`devam`, metin (literal, `+`, `s[i]`, `uzunluk`, `kod`, `karakter`, `metin(sayı)`, `==`/`!=` içerik).

**Yorumlayıcıda ve Go-elf'te var, TancElf.tan'ın derlediği alt kümede YOK:**

| Özellik | Go yorumlayıcı | Go elf | TancElf.tan | Kanıt |
|---|---|---|---|---|
| Ondalık (float) literaller + SSE | ✅ | ✅ | ❌ kasıtlı açık hata | TancElf.tan:2948-2965; SURUM.md:100-104 |
| Ondalık aritmetik/işlev döndürmesi | ✅ | ✅ | ❌ | aynı |
| `sözlük` tipi + f_sozluk_* | ✅ | ✅ | ❌ (paralel listelerle taklit) | DerleElf.go:3525-3790 vs TancElf.tan:0 eşleşme |
| `kayıt` tipi + sema + metot | ✅ | ✅ | ❌ | DerleElf.go:756-781, 1099 vs TancElf.tan yok |
| `dene`/`yakala` | ✅ | ❌ | ❌ | DerleElf.go:3930-3931, 4955 yorum |
| `köprü` | ✅ (iskoleti sökük) | ❌ | ❌ | Kopru.go:27-32 |
| Bitwise `& \| ^ << >>` | ✅ | ✅ | ❌ ("Tan'da bitwise işleç YOK") | TancElf.tan:1072 |
| İşlev değeri / dolaylı çağrı (`CagriIfadeDugum`) | ✅ | ❌ | ❌ | DerleElf.go:2476 |
| `mantık` tipi ayrımı (bool vs tam) | ✅ | ⚠️ | ❌ (bool tam olarak taşınır; `tipYigin` yalnız tam/metin/liste) | TancElf.tan:3061, 3109, 3120 |
| Dönüş tipi izleme (işlevden metin döndürüp `+` kullanma) | ✅ | ✅ | ❌ **belgeli sınır** | TancElf.tan:19-22 |
| Çok baytlı UTF-8 literal tutarlılığı | rune | bayt | ❌ **belgeli sınır** | TancElf.tan:23-25 |
| `yazDosya`/`yaz_dosya` (kullanıcı programından) | ✅ | ✅ | ❌ **belgeli eksik** | SURUM.md:105-108 |
| `her x ["bir","iki"] içinde` (metin-literal listesi eleman tipi) | ✅ | ✅ | ❌ **belgeli eksik** | SURUM.md:109-111 |

---

## 6. TAN'da Eksik Standart Kütüphane Özellikleri (Görev 4)

### 6.1 Runtime yardımcı karşılaştırması (Go `f_*` vs TancElf.tan `f_*`)

TancElf.tan'ın tanımlayabildiği runtime yardımcıları: `yaz_sayi`, `f_tan_ayir`, `f_bellek_kopyala`, `f_liste_ekle`, `f_liste_yap`, `f_sayi_metne`, `f_metin_birlestir`, `f_metin_esit`, `f_metin_indeks`, `f_kod`, `f_karakter`, `f_oku`, `f_dosyaVarMi`, `f_yazBaytlar`, `f_argsay`, `f_arg`, `f_yaz_metin` = **17 yardımcı** (sürücü ~3795). **Faz 5 (B) itibariyle KOŞULSUZ gömülmez:** `yardimciKullanimiTara` govde bandındaki REL32 hedeflerini tarar, `yardimciKapanisi` bağımlılık kapanışını (iç `f_tan_ayir`/`f_bellek_kopyala` çağrıları) ekler, yalnız o küme emit edilir — `yaz 42` ikilisi 2267→395 bayta düştü (yalnız `f_yaz_metin`+`yaz_sayi`+`f_tan_ayir`+`f_bellek_kopyala`).

Go DerleElf.go'da var, TancElf.tan'da YOK (~35+):

| Kategori | Eksik `f_*` |
|---|---|
| Hata | `f_hata_cik`, `f_hata_sifira_bolme` (TancElf `sistemDur` kullanır) |
| Arena | `f_arena_ayir`, `f_arena_serbest` |
| Metin | `f_harfler`, `f_metin_araligi`, `f_parcala`, `f_yaz_metin_deger` |
| Sözlük | `f_sozluk_hash/yap/koy/al/varmi/sil/anahtarlar` (7) |
| Sayı | `f_sayi` (metin→sayı), `f_kesir_metne`, `f_yaz_kesir`, `f_yuvarla`, `f_e_ussu`, `f_log`, `f_rastgele` |
| Dosya (konumlamalı) | `f_yaz_dosya`, `f_ekle_dosya`, `f_dosya_ac_okuyaz/oku/yaz/konumla/oku_blok/yaz_blok/oku_blok_ham/yaz_blok_ham/senkron/kapat/sil`, `f_bellek_esle`, `f_bellek_coz` |
| Eşzamanlılık | `f_futex_wait/wake`, `f_iplik_cikis/bekle`, `f_kilit_al/birak`, `f_atomik_ekle_ham` |
| Listeleme | `f_birlestir_metin/tam/kesir` |

### 6.2 Yerleşik işlev (builtin) kapsamı
TancElf.tan özel-dağıtımlı yerleşikler: `uzunluk`, `ekle`, `listeYap`, `metin`, `kod`, `karakter`, `metinEsit`, `metinBirlestir`, `metinAl`, `sistemDur`, `tamBol` (ifadeBantYaz 3084-3197); genel yol `callBant("f_"+ad)` (3206). Kullanıcı programlarından erişilebilen TAN-elf yerleşik seti: 17 runtime yardımcı + `oku`/`yazBaytlar`/`arg`/`argsay`/`dosyaVarMi`/`yaz`.

Yorumlayıcıda var ama TAN-elf yolunda YOK (tam liste, Yerlesik.go): `çıkar liste liste_mi sözlük harfler birleştir rastgele satırlar parçala kırp zaman getir gönder json_çöz json_yap sun çalıştır sha256Ozet aesSifrele aesCoz hamAyir hamOku hamYaz hamOku4 hamYaz4 hamOku8 hamYaz8 hamOkuBayt hamYazBayt bellekKopyala bellekDoldur atomik* kanal* soket* iplik* kilit* u8..i64 disKutuphane* disIslev*`.

---

## 7. Runtime Bağımlılıkları (Görev 5)

### 7.1 TancElf.tan'ın ürettiği ikilinin runtime bağımlılığı: **SIFIR** (libc yok)
- Heap: `brk` syscall tabanlı bump allocator (`f_tan_ayir`, TancElf.tan:1839-1877; Go karşılığı DerleElf.go:2907).
- Dosya: ham `open(2)/read(0)/write(1)/close(3)`; program çıkışı `exit(60)`.
- Girdi/çıktı: stdout write(1); argv `v___argv = rsp` yığın üstünden (argvYakalaBant 2412).
- Statik bağlı x86-64 ELF, `ldd` boş (TestArkaUcGoSuzTemiz.sh:100-107).
- 3 gömülü global: `v_yigin`, `v_yiginSon`, `v___argv` (TancElf.tan:3543-3545).

### 7.2 TancElf.tan'ın KENDİSİNİN derleme-zamanı bağımlılığı
- Syscall'lar: oku/okuma (girdi), yazBaytlar (çıktı), dosyaVarMi (içe al açılımı), arg/argsay (yollar).
- Sürücü sırası (TancElf.tan:3852-3864): `arg(1)`→girdi, `arg(2)`→çıktı, `oku`→kaynak, `tokenle`→tokenler, `tokenGenislet`→`içe al` açılımı, `tipSabitNoktasiTara` (dönüş+parametre tip ön-geçişi, ortak sabit nokta), `blokDerle`→govde, tampon birleştir, `bagla`→baytlar, `elfUret`→ELF, `yazBaytlar`→çıktı.
- Tampon disiplini (O(n²) tuzakları): `tamponOlustur/tamponEkle/tamponEkleTumu/tamponSonuc` (276-324), bayt varyantı (1122-1166), ve sürücüdeki `bantTumuKutu` (3508-3515, bulunan 2. büyük O(n²) kaynağı — `listeBirlestir` TEK çağrısı `O(|govde|²)` idi).

---

## 8. Go'ya Özel Bağımlılıklar (Görev 6)

Go üretim yolunun (`DerleElf.go`) kullandığı Go stdlib ve dil araçları — her biri TAN'a nasıl aktarılıyor/aktarılmıyor:

| Go bağımlılığı | Kullanım yeri | TAN karşılığı |
|---|---|---|
| `encoding/binary` (PutUint16/32/64, LE) | makine kodu + ELF başlıkları (82-5996) | elle taban-256 (`tabanBol256` 1078, `tamdanBaytlar` 1085) |
| `fmt.Sprintf` | etiket adları, raporlar (1310-5710) | `metinBirlestir` + `metin(sayı)` |
| `fmt.Printf/Fprintln/Fprintf` + `os.Stderr` | hata/rapor (5444-5849) | `yaz(...)` + `sistemDur(1)` |
| `math.Float64bits` | ondalık sabitleri GPR'lere bit olarak gömme (1320-5070) | TancElf'ta ondalık yok → taşıma engeli |
| `os.ReadFile/WriteFile/Getenv/Exit` | sürücü (5442-5839) | ham syscall'lar; `Getenv` karşılığı BİLİNMİYOR |
| `path/filepath.Abs/Dir` | `içe al` açılımı (5301-5316) | `dosyaDizini` (3398) |
| `strings.HasPrefix` | sembol tablosu filtresi (5885) | `metinAl` + `metinEsit` |
| `panic`/`recover` + `TanHata` | hata akışı (Hata.go, 5447-5454) | TancElf'ta try/catch YOK; `yaz`+`sistemDur(1)` |
| `append`/`make`/`len`/`copy` | Go dil seviyesi | `ekle`/`listeYap`/`uzunluk`/`tampon*` |
| **AST `Dugum` interface + 25 düğüm** | tüm ön uç, optimizer, tip çıkarımı, budama | **düz RPN bant** (tasarım farkı — taşıma, yeni veri modeli gerektirir) |
| `interface{}` dinamik `Deger` | yorumlayıcı/VM (Yorumlayici.go:17) | TAN portu kapsam dışı |
| `sync`/`net`/`crypto`/`encoding/json` | yorumlayıcı yerleşikleri | TAN-elf yolunda yok |
| `runtime`/`sync/atomic` (Go runtime) | yorumlayıcı goroutine/mutex | native tarafı ham futex/clone ile yapılır (DerleElf.go:4547) |

**Yapısal engel:** Go'nun optimizer/tip çıkarımı/budaması AST üzerinde çalışır; TancElf.tan AST kullanmaz. Bu yüzden "port", birebir satır-satır değil, **RPN bant / token akışı seviyesinde yeniden ifade** demektir (§9).

---

## 9. Taşınabilir En Küçük Bileşen Seçimi (Görev 7)

### 9.1 Adaylar (TAN'ın desteklediği alt küme göz önünde)
| Aday | Go kaynağı | Tahmini TAN satırı | Yeni runtime? | Yeni dil özelliği? | Sabit nokta riski |
|---|---|---|---|---|---|
| **A. RPN-bant seviyesi sabit katlama + cebirsel sadeleştirme** | Optimize.go `sabitKatla`/`cebirsel` (164-300) | ~150-220 | hayır | hayır | düşük |
| B. `mantık` tipi ayrımı (bool vs tam) | DerleElf.go tipCikar | ~60-100 | hayır | hayır | düşük-orta |
| C. Ölü işlev budaması | `elfUlasilabilirIslevler` (5410) | ~80-120 | hayır | hayır | orta |
| D. Liste eleman tipi çıkarımı (her-loop) | `parametreTipleriniOgren` (1159) | ~60-100 | hayır | hayır | orta |
| E. Sabit koşul düzleme + ölü blok eleme | Optimize.go `deyim` (49-112) | ~100-150 | hayır | hayır | orta |
| F. Sözlük runtime'ı | f_sozluk_* (3525-3790) | ~350-500 | evet (7 helper) | evet (sözlük) | yüksek |
| G. Ondalık literaller | SSE kodgen (1320-5070) | ~400+ | evet (kesir_metne vb.) | evet (SAYIKESIR) | yüksek |

### 9.2 Seçim: **A — sabit katlama + cebirsel sadeleştirme (RPN-bant seviyesi)**

Gerekçe:
1. **Boyut:** Go karşılığı 343 satırın en küçük, en izole parçası; TAN'da ~150-220 satır. (B/C/D daha küçük görünse de: B ve D mevcut tip çıkarımına derin dokunur; C, çağrı grafiği için token/ifade çözümlemesi gerektirir — "AD + (" ayrımı parse zorunluluğu yüzünden aslında büyür.)
2. **Sıfır yeni bağımlılık:** yeni runtime yardımcısı yok, yeni dil özelliği yok, yeni global yok. Yalnızca mevcut `ifadeBantYaz`'ın `IKILI`/`SAYITAM` kollarına ön-filtre.
3. **Oracle güvenliği:** FarkTesti yorumlayıcı-vs-native TancElf bayt kimliğini denetler (FarkTesti.sh:12-14); katlama TancElf'in HER İKİ çalışma yoluna da aynı eklendiğinden bayt kimliği korunur. 3. ayak (c) anlam-korunumu düzeyinde çalıştığından (SURUM.md:213) Go-elf ile katlama farklılığı da sorun olmaz.
4. **Anında ölçülebilir fayda:** derlenen ikili boyutu küçülür (şu an sabit `2+3*4` bile üç makine talimatı üretir), codegen hızı artar, ve RPN bant hattı üzerinde İLK optimizasyon katmanı kurulur — sonraki tüm taşımaların (E, B, D, C) şablonu olur.
5. **Sabit nokta riski düşük:** optimize edici TancElf.tan'ın KENDİ kaynağına uygulanır (kendini derlerken katlama tetiklenirse üretim küçülür). `d6b4fac` salınım deseni doğuran "yeni özellik + öz-kullanım" sınıfına girmez çünkü derleyici kaynağı kendisi tarafından yeniden derlenir ve sabit nokta iki-ayaklı doğrulanır.

**Aday E (sabit koşul düzleme + ölü blok) A ile birlikte tek fazda önerilir** — ikisi de `Optimize.go`'nun aynı geçişidir (Govde/deyim/ifade). A+E = Go Optimize.go'nun anlam bakımından tam karşılığı.

**Sonraki adaylar (sıralı):** B (`mantık` ayrımı) → D (liste eleman tipi, bilinen eksik #3'ü kapatır) → C (budama, ikili boyut) → G (ondalık, en riskli) → F (sözlük) — bkz. BOOTSTRAP_ROADMAP.md. (B, D ve C — Faz 2/3/5 — 14 Ağu'da YAPILDI.)

---

## 10. Bootstrap Migration Planı Özeti (Görev 8)

Detaylı, fazlı plan ve doğrulama protokolü **BOOTSTRAP_ROADMAP.md**'dedir. Özet:

1. **Faz 0:** Mevcut zinciri yeniden doğrula (`BootstrapGoSuz.sh` + `KanitGoSuzTarihce.sh` + `TestArkaUcGoSuzTemiz.sh`); boyut tabanı: gen1=142.944 B, gen2=gen3=TancElf=114.998 B.
2. **Faz 1:** Optimizer çekirdeği (A+E) TancElf.tan'a, RPN-bant seviyesinde. Doğrulama: FarkTesti, sabit nokta, boyut küçülmesi.
3. **Faz 2-4:** `mantık` ayrımı, liste eleman tipi (eksik #3), dönüş tipi izleme iyileştirmesi (eksik #1) — TancElf.tan'ın kendi belgeli sınırlarını kapatır.
4. **Faz 5:** Ölü işlev/yardımcı budaması (Go `elfUlasilabilirIslevler` karşılığı) — ✅ YAPILDI (A: işlev BFS kapanışı; B: runtime yardımcı topolojik taraması; `yaz 42` 2267→395 B).
5. **Faz 6 (uzun vade):** ondalık (SSE), sonra sözlük; her biri kendi sabit-nokta/shim doğrulamasıyla.
6. Her fazda: **iki-adımlı shim tekniği** (9ffe0b5 deseni) gerekiyorsa uygulanır; **salınım kuralı** (en büyük üretimi seç, d6b4fac deseni) izlenir; **tuzak + sabit nokta + anlam korunumu** üç ayağı koşulur.

---

## 11. Referanslar
- `BootstrapGoSuz.sh` (71 satır), `Bootstrap.sh` (45), `KanitGoSuzTarihce.sh` (164), `TestArkaUc.sh` (338, 20 test), `TestArkaUcGoSuzTemiz.sh` (116), `FarkTesti.sh` (24), `GercekProgramlar.sh` (48), `Olcum.sh` (79)
- `SURUM.md` (özellikle 13-118 Go'suz Bootstrap bölümü, 199-217 self-hosting doğrulaması)
- `TancElf.tan` (3552 satır), `DerleElf.go` (5998), `Optimize.go` (343), `Lexer.go` (230), `Parser.go` (699), `Yorumlayici.go` (944), `Yerlesik.go`
- Git: `c0835c7`(ilk) `412c26f`(0.5.0 self-hosting) `62d0fce`(seed) `9ffe0b5`(shim) `d6b4fac`(salınım) `4aecdf4`(parametre tipi çıkarımı)
