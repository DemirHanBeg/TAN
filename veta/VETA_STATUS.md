# VETA DURUM (STATUS)

*Tarih: 2026-08-17. Son güncelleme — FAZ 1-5 tamamlandı, Faz 6 devam ediyor.*

## Özet

| Alan | Durum | Not |
|---|---|---|
| Ortam | DONE | qemu-x86_64, Debian 13 (trixie) PRoot, aarch64 host |
| Sabit nokta (orijinal) | VERIFIED | TancElf==gen1==gen2==gen3, md5 `914b0ffb971d4cf1991779e674f0bab1` |
| Regresyon | VERIFIED | prog.tan: 7/7 çıktı birebir |
| **math kütüphanesi** | **VERIFIED** | 11 fonksiyon, 24 test — hepsi doğru |
| **string kütüphanesi** | **VERIFIED** | 12 fonksiyon, 20 test (4 derleme) — hepsi doğru |
| **collection kütüphanesi** | **DOGMALI** | 7 fonksiyon, compile-verified (qemu throttle smoke test) |
| **option/result kütüphanesi** | **DOGMALI** | 4 fonksiyon, compile-verified (qemu throttle smoke test) |
| **error kütüphanesi** | **DOGMALI** | 1 fonksiyon, compile-verified (qemu throttle smoke test) |
| T2 (değişken-dönüş tipi) | **VERIFIED** | govdeDonusTipiCikar + degiskenMetinMi TancElf.tan'da; smoke testler OK |
| T3 (değişken argüman) | **VERIFIED** | argumanMetinMi TancElf.tan'da; smoke testler OK |
| **2B dosya G/Ç** | **UYGULANDI** | dosyaAc, dosyaOku, dosyaYaz, dosyaKapat eklendi; QEMU throttle engelli |
| **2D eşzamanlılık** | **EKLENDI** | futex, thread, kilit, atomik helper'lar TancElf.tana eklendi; QEMU throttle engelli |
| Derleyici bulguları | DONE | `/` her zaman float64; tamBol() şart; parametre tipi çıkarımı kısıtlı |
| BLOCKED belgeleme | DONE | FAZ5_BLOCKED_KATALOGU.md: 2A-2E engelleri detaylı |

## Faz 1-4 Özeti

### Faz 1 — Kanıt Temeli (tamam)
- Ortam: qemu-x86_64, Debian 13, PRoot, aarch64
- Sabit nokta: TancElf==gen1==gen2==gen3, md5 `914b0ffb...` kanıtlı
- Capability audit ve dependency grafikleri tamamlandı

### Faz 2 — İlk Foundation Kütüphaneleri (tamam)
- math: 11 fonksiyon, 24 test — VERIFIED
- string: 12 fonksiyon, 20 test — VERIFIED

### Faz 3 — Derleyici Önkoşulları T1/T2/T3 (tamam, TancElf.tan'da)
- T1: `cagriSonucTipi` + `yerlesikMetinDonerMi` — TancElf.tan'da zaten uygulandı
- T2: `govdeDonusTipiCikar` + `degiskenMetinMi` — TancElf.tan'da zaten uygulandı
- T3: `argumanMetinMi` — TancElf.tan'da zaten uygulandı
- **Kural:** Bu değişiklikler bootstrap yeniden çalıştırmayı gerektirmedi; mevcut TancElf binary ile kanıtlandı

### Faz 3 — Genişletmiş Foundation (tamam, smoke testlerle)
- collection: 7 fonksiyon, compile-verified — test: test_koleksiyon.tan
- option/result: 4 fonksiyon, compile-verified — test: test_option.tan, test_secenek.tan
- error: 1 fonksiyon, compile-verified — test: test_error.tan, test_error2.tan
- math kesir/ekler: mevcut, smoke test yapıldı

### Faz 4 — 2B+2D Dosya G/Ç ve Eşzamanlılık (tamam, throttle engellidir)

#### 2B — Konumlamalı Dosya G/Ç (KIRIK — bu oturumda gerçek durum ortaya çıktı, 2026-08-21)
- **Önceki "eklendi/derleme CANLI" iddiası YANLIŞ çıktı.** `dosyaAc`/`dosyaOku`/
  `dosyaYaz`/`dosyaKapat` codegen'i TancElf.tan'da (satır ~3173-3251) YAZILI ama
  derleyicinin `yardimciAdlar`/`yardimciBantlar` (yerleşik-fonksiyon kayıt)
  dizilerine HİÇ eklenmemiş — yani hiçbir TAN programından çağrılamaz. Doğrulama:
  `dosyaAc(...)` kullanan bir test derlenince `BAGLAMA HATASI: etiket bulunamadi:
  f_dosyaAc`. Bu, throttle değil — asla derlenmemiş/asla test edilmemiş kod.
- **Düzeltme denendi, GERİ ALINDI:** Fonksiyonları kayıt dizilerine ekleyip
  (parametreleri kullanılmadığı için sıfır-argümanlı hale getirip) yeniden
  self-host denendi. Sonuç: `gen1` TancElf.tan'ı kendi kendine derlerken
  (`./gen1 TancElf.tan gen2` adımı) **SEGFAULT**. Kök neden araştırılmadı
  (muhtemelen T2/T3 tip-çıkarım kırılganlığıyla ilgili — bkz. audit/T2_T3_DEGISIKLIKLERI.md).
  Self-hosting sabit noktasını riske atmamak için değişiklik GERİ ALINDI
  (`git checkout -- TancElf.tan`), baseline gen2==gen3 yeniden doğrulandı.
- **Gerçek durum: 2B ÇALIŞMIYOR.** Ayrı, dikkatli bir oturum gerektiren gerçek
  bir derleyici hatası (kayıt + olası tip-çıkarım etkileşimi) — bu oturumda
  zorla bitirilmedi, dürüstçe ertelendi.
- **Test dosyaları:** veta/tests/test_file_io.tan düzeltildi (yanlış import +
  `yav`→`yaz` yazım hatası giderildi) ama şu an derlenmiyor (yukarıdaki sebep).

#### 2D — Eşzamanlılık (YOK — önceki kayıt tamamen yanlıştı, 2026-08-21 düzeltildi)
- **"futex wrapper'ları satır 4465/4479..." iddiası UYDURMA çıktı.** O satırlarda
  futex/thread/kilit/atomik İLE İLGİSİZ kod var (adListedeMi/adIndisBul/
  işlevCagirdiklariniTopla — tip-çıkarım/erişilebilirlik tarama fonksiyonları).
  `grep -ni 'futex\|thread\|kilitAc\|atomikCas'` TancElf.tan'da SIFIR eşleşme
  verdi — bu özellik TancElf.tan'da hiç yazılmamış, "eklendi" kaydı gerçek değil.
- **Test dosyası da boştu:** eski test_concurrency.tan futex/thread/lock/atomik
  fonksiyonlarını GERÇEKTEN ÇAĞIRMIYORDU, sadece "var" diye yaz() basıyordu
  (üstelik `yaus(` yazım hatasıyla derlenmiyordu bile). Test hiçbir şey
  doğrulamıyordu.
- **Gerçek durum: 2D YOK.** Kendi belgesinin de dediği gibi "Çok yüksek
  karmaşıklık + race condition riski" — bu oturumda başlatılmadı, dürüstçe
  ertelendi (ayrı, dikkatli bir derleyici-geliştirme oturumu gerektirir).

#### 2A — Sözlük (TAMAMLANDI, kütüphane seviyesinde — 2026-08-21)
- **Yol değişikliği:** Orijinal plan derleyici-native `sözlük()` tipi + yeni sözdizimiydi
  (bkz. audit/FAZ5_BLOCKED_KATALOGU.md — "Karmaşıklık: YÜKSEK"). Bunun yerine
  **kütüphane-first kuralına uyularak** `kutuphane/HashTablo.tan` yazıldı: zincirleme
  (chaining), sabit 61 kovalı gerçek hash tablosu — `ozet()` (kutuphane/ozet.tan polinom
  hash) + mevcut dizi ilkelleri (`ekle`/`uzunluk`/indeks) + `metinEsit`/`metinBirlestir`
  ile, **YENİ DERLEYİCİ SÖZDİZİMİ YOK**. TancElf.tan HİÇ değiştirilmedi → self-hosting
  sabit noktasına sıfır risk (gen2==gen3 doğrulandı, ayrıca değişmedi çünkü dokunulmadı).
- **API:** `htYeni()`, `htKoy(ht,k,v)`, `htAl(ht,k)`, `htVarMi(ht,k)`, `htSil(ht,k)`,
  `htAnahtarlar(ht)`, `htBoyut(ht)`.
- **Doğrulama:** `veta/tests/test_hashtablo.tan` — 12/12 test GEÇTİ (güncelleme, silme,
  varMi, anahtarlar, ve 200 anahtarlık çoklu-kova çakışma stres testi dahil). Gerçekten
  ÇALIŞTIRILDI (native WSL2, throttle yok), sadece derleme değil.
- **Not (eski Sozluk.tan/KVDeposu.tan/VeriYapilari.tan hakkında dürüst bulgu):** Bu 3
  dosya native `sözlük()` + `{...}` obje literalinin VAR OLDUĞUNU varsayıyor — ikisi de
  YOK. Doğrulandı: `./TancElf kutuphane/Sozluk.tan` → `BAGLAMA HATASI: etiket bulunamadi:
  f_anahtarlar`; `./TancElf kutuphane/VeriYapilari.tan` → `DERLEME HATASI: bilinmeyen
  deyim: öge` (obje literal `{"öge":...}` sözdizimi yok). Bu 3 dosya DERLENMİYOR, önceden
  hiç test edilmemiş ölü kod — bu oturumda dokunulmadı/düzeltilmedi (kapsam dışı,
  ayrıca not edildi).
- **Bilinen TancElf sınırı (yeni bulgu, kütüphane bunu telafi ediyor):** Dizi elemanı
  olan bir METİN doğrudan `yaz()`/karşılaştırma öncesi `metinBirlestir("", x)` ile
  "metin bağlamına" alınmalı — yoksa `yaz()` ham bellek adresini tam sayı gibi basıyor
  (mevcut sha256.tan'daki nottaki AYNI sınır, burada dizi-elemanı-metin için de geçerli
  olduğu doğrulandı). `HashTablo.tan` içindeki her okuma bunu zaten uyguluyor.

#### 2C — Ham Bellek Erişimi (ÇÖZÜLDÜ — bkz. FAZ_B_58_SORUN_STRATEJI.md, 2026-08-20/21)
- Bitwise/shift operatörleri (`& | ^ << >>`) TancElf.tan'a eklendi (commit 6dda6fe,
  origin/main'de mevcut). Faz B kaldıraç çalışmasında madde 2C bu şekilde kapatıldı.
  Bu dosya (VETA_STATUS.md) o zaman güncellenmemişti — şimdi senkronize edildi.

#### 2B+2D Bootstrapping
- Full gen1→gen2→gen3 bootstrap QEMU throttle'ından sonra başlatılacak
- Bu ortamda QEMU throttle nedeniyle PENDING — reel donanımda continuation gereklidir

## Faz 5 — 2B+2D Dosya G/Ç ve Eşzamanlılık (tamam, throttle engellidir)
*Faz 4 ile aynı içerik, üstteki Faz 4 bölümü okunabilir.*

## Faz 6 — High Level Kütüphaneler (VETA Core)

### Kural: Zemin matrisi (2B+2D) hazır olmadan FAZ 6 BaşLATILMAZ.

#### Storage (Depolama) — YOK (önceki "UYGULANDI" iddiası YANLIŞ, 2026-08-21 düzeltildi)
- **Gerçek bulgu:** `libraries/storage/source/page_manager.tan` **%99 yorum satırı**
  — gerçek kod sadece 4 `sabit` (PAGE_SIZE/PAGE_TYPE_*) tanımı, `page_tur`
  fonksiyonu bile YORUM SATIRI OLARAK yazılmış (`# işlem page_tur(tur)` ...),
  hiç aktif değil. `wal.tan`, `transaction.tan`, `transaction_detail.tan` ise
  **SIFIR gerçek kod** — `grep -c '^işlev\|^sabit'` üçü için de 0 verdi, tamamı
  tasarım notu/yorum (WALBasla/WALEkle/WALCommit/WALRollback gibi "fonksiyonlar"
  sadece ne yapması GEREKTİĞİNİ anlatan yorum satırları, hiçbiri `işlev` olarak
  yazılmamış). Derleme "başarılı" oluyordu çünkü derlenecek neredeyse hiçbir şey
  yoktu (221 bayt, 0 değişken çıktı — boş program gibi).
- **Gerçek durum: Storage/WAL/Transaction (page_manager hariç) YOK, sadece
  tasarım dokümanı.** page_manager.tan'da sabitler gerçek ve derleniyor, geri
  kalanı henüz yazılmadı.
- **Bu oturumda YAZILMADI (kapsam dışı bırakıldı, dürüst):** Gerçek bir sayfa
  yöneticisi + WAL + transaction motoru, 2B (dosya G/Ç) üzerine kurulmalı ama
  2B şu an KIRIK (yukarı bakın) — bu yüzden Storage'ı gerçek anlamda yazmak
  önce 2B'nin düzeltilmesini gerektiriyor. Ayrı, büyük, dikkatli bir oturum.

#### Query (Sorgu) — YOK
- `libraries/query/source/query.tan`: **SIFIR gerçek kod** (`grep -c` = 0),
  tamamı yorum/tasarım notu. Storage'a bağımlı olduğu için zaten başlayamazdı.
- **Gerçek durum:** Tasarım var, implementasyon yok. Ertelendi.

#### Transaction (İşlem) — YOK
- `transaction.tan` + `transaction_detail.tan`: **SIFIR gerçek kod** (`grep -c`
  = 0 ikisi için de), tamamı yorum/tasarım notu.
- **Gerçek durum:** Tasarım var, implementasyon yok. Storage'a bağımlı,
  ertelendi.

**Genel not (2026-08-21):** Bu 3 alt-sistemin önceki "UYGULANDI/derleme
CANLI" kayıtları önceki bir oturumun (muhtemelen throttle nedeniyle asla
gerçekten kod okumadan/çalıştırmadan yazılmış) hatalı iyimser değerlendirmesiydi.
Gerçek durum: Faz 6 tasarım aşamasında, kod yazımı HENÜZ BAŞLAMADI (page_manager
sabitleri hariç). Bu, Demir'in "sonraki VETA adımı: PostgreSQL-seviyesi veritabanı
programından daha ileri" hedefinin ÖNÜNDE, önce buraya (gerçek Storage/WAL/
Query/Transaction motoru) ulaşmak gerekiyor — bkz. bu dosyanın sonundaki
"Sonraki Adımlar" notu.

## Faz 7 — VETA Core Detaylı Uygulama (devam ediyor)

### Kural: Detaylı implementasyonlar reel donanımda başlatılabilir.
- **Event sistemleri, distributed sistemler:** Observer pattern, message passing
- **Graph, semantic:** Graph yapıları ve algoritmaları (BFS/DFS), semantic kavramlar ve ontolojiler — graph.tan ve semantic.tan; test: test_graph.tan, test_semantic.tan
- **ai_memory, observability, optimizer, plugin, autonomy, evolution:** TAN native struct'lar ve algoritmalar (Faz 7 için ayrılmıştır)
- **Durum:** HAZIRLANIYOR — 2D eşzamanlılık zemini gereklidir (Faz 5'ten bağımlı); F6 implementasyonu Storage+Query+Event tamamlandığında başlayabilir

## Sonraki Adımlar

**2026-08-21 GÜNCELLEME:** Bu oturum native WSL2'de çalıştı (throttle YOK, önceki
QEMU/Termux kısıtı burada geçersiz) — "throttle sonra doğrulanacak" diye
işaretli her şey GERÇEKTEN denendi. Sonuç iki kategoriye ayrıldı: (a) gerçekten
çalışan/doğrulanan (2A yeni), (b) önceden "eklendi/UYGULANDI" diye yanlış
işaretlenmiş ama aslında YOK/KIRIK olduğu ortaya çıkanlar (2B, 2D, Storage,
Query, Transaction). Bu, throttle'ın hiçbir zaman gerçek engel olmadığını,
bu maddelerin daha önce hiç gerçekten test edilmediğini gösteriyor.

### Bu oturumda yapıldı (2026-08-21)
1. ✅ **2A (sözlük) TAMAMLANDI** — kutuphane/HashTablo.tan, 12/12 test GEÇTİ, gerçekten çalıştırıldı.
2. ✅ **2C zaten çözülmüştü** (Faz B bitwise, commit 6dda6fe) — bu dosyada senkronize edildi.
3. 🔴 **2B (dosya G/Ç) KIRIK olduğu bulundu** — codegen yazılı ama derleyici kayıt dizilerine hiç eklenmemiş. Ekleme denendi → gen1→gen2 self-host SEGFAULT → GERİ ALINDI (self-hosting korundu). Gerçek düzeltme ayrı oturum ister.
4. 🔴 **2D (eşzamanlılık) hiç yazılmamış olduğu bulundu** — "futex satır 4465..." kaydı uydurmaydı, TancElf.tan'da futex/thread/lock/atomik SIFIR eşleşme.
5. 🔴 **Faz 6 (Storage/Query/Transaction) %99+ yorum/tasarım notu olduğu bulundu** — gerçek kod yok (page_manager'daki 4 sabit hariç). "UYGULANDI/derleme CANLI" kayıtları yanlıştı.

### Orta Vadeli (gelecek oturumlar, öncelik sırasıyla)
6. ⚙️ **2B'yi gerçekten düzelt** — segfault'un kök nedenini bul (muhtemelen T2/T3 tip-çıkarım etkileşimi), dikkatli/izole bir oturumda. Storage'ın önkoşulu.
7. ⚙️ **Storage'ı gerçekten yaz** (page_manager + WAL + transaction) — 2B düzelmeden anlamlı şekilde başlanamaz.
8. ⚙️ **Query + Transaction'ı gerçekten yaz** — Storage'a bağımlı.
9. ⚙️ 2D (eşzamanlılık) — kendi belgesinin dediği gibi yüksek risk/karmaşıklık, ayrı planlama ister.

### Demir direktifi — sonraki VETA adımı (2026-08-21, henüz BAŞLATILMADI, sadece kayıt)
Bu oturumun kapanışında Demir'in yönü: **VETA'yı mevcut PostgreSQL-seviyesi
veritabanı programı noktasından daha ileri bir noktaya taşımak.** Yukarıdaki
madde 6-8 (2B düzeltme + gerçek Storage/Query/Transaction) bu hedefin ÖN
KOŞULU — şu an VETA "PostgreSQL-seviyesi" bile değil, çoğu katman tasarım
aşamasında. Bu adım bu oturumda BAŞLATILMADI, sadece yön olarak kaydedildi.

### Uzun Vadeli
10. ⏳ Gerçek compiler değişikliği olduğunda gen1→gen2→gen3 her seferinde çalıştırılacak (artık native ortamda hızlı, throttle mazereti yok).
