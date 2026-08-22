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
| **2B dosya G/Ç** | **VERIFIED (2026-08-22)** | dosyaAc/dosyaOkuKonum/dosyaYazKonum/dosyaKapat gerçekten çalışıyor, commit `01a23f9` |
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

#### 2B — Konumlamalı Dosya G/Ç (ÇÖZÜLDÜ, GERÇEKTEN ÇALIŞIYOR — 2026-08-22, commit `01a23f9`)
- **Önceki "KIRIK" bulgusu (2026-08-21) doğruydu, kök sebep daha derinmiş.**
  Eski `dosyaAc`/`dosyaOku`/`dosyaYaz`/`dosyaKapat` sadece kayıt dizilerine
  eklenmemiş DEĞİLDİ — kodun KENDİSİ de yanlış register varsayımlarıyla
  yazılmıştı (parametreleri `rdi`/`rsi` (TAN'ın kanıtlanmış ABI'si, reg7/6)
  yerine `rax`/`rdx`'ten (reg0/2) okuyordu). Önceki oturumun "kayıt dizisine
  ekleyince gen1→gen2 SEGFAULT" bulgusu muhtemelen bu bozuk koddandı, T2/T3
  tip-çıkarımıyla ilgisizdi.
- **Çözüm: YAMA değil YENİDEN YAZIM.** Kanıtlanmış `okuBant`/`yazDosyaBant`
  deseni (parametre reg7/reg6/reg2, TAN string→C-string via `f_tan_ayir`+
  `f_bellek_kopyala`) birebir takip edilerek sıfırdan yazıldı. Yeni API
  (net isimler, eski isimlerle çakışma yok — `page_manager.tan` henüz bu
  isimleri hiç kullanmıyordu):
  - `dosyaAc(yol, mod) -> fd` (mod: 0=salt-okunur, 1=oku-yaz+oluştur)
  - `dosyaOkuKonum(fd, konum, uzunluk) -> metin` (pread64)
  - `dosyaYazKonum(fd, konum, icerik) -> yazılan bayt` (pwrite64)
  - `dosyaKapat(fd) -> 0`
- **Doğrulama:** Gerçek rastgele-erişim testi — fd aç, offset 0'a "HELLO",
  offset 100'e "WORLD" yaz, GERİ OKU — ikisi de DOĞRU (sparse dosya, gerçek
  pozisyonlama çalışıyor). Self-hosting sabit nokta korundu (gen2==gen3,
  yeni builtin'lerle TancElf.tan iki kez üst üste derlendi). Tam regresyon
  (TestAraclar.sh 16/16, TestArkaUcGoSuzTemiz.sh, TestFormatIdempotent
  67/67) yeni derleyiciyle yeşil.
- **Gerçek durum: 2B ÇALIŞIYOR.** Storage'ın (Faz 6) önkoşulu tamam.

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

#### Storage (Depolama) — GERÇEK, ÇALIŞIYOR (2026-08-22, commit `f2919f2`)
- **Eski `libraries/storage/source/*` (page_manager/wal/transaction/
  transaction_detail) TEMEL ALINMADI** — %99 yorum/tasarım notuydu, `sabit`
  anahtar kelimesi TAN'da hiç yok, hiç derlenmiyordu (bu durum tespiti hâlâ
  doğru, aşağıdaki yeni implementasyon SIFIRDAN yazıldı).
- **`kutuphane/PageManager.tan` — gerçek sayfa yöneticisi.** Sayfa 0 =
  HEADER (meta veri, toplam sayfa sayısı). `pmAc/pmSayfaAyir/pmSayfaOku/
  pmSayfaYaz/pmSayfaSayisi/pmKapat`. 2B'nin (`dosyaAc`/`dosyaOkuKonum`/
  `dosyaYazKonum`/`dosyaKapat`, commit `01a23f9`) üzerine kurulu.
  Doğrulandı: yaz+oku+**kapat-tekrar-aç kalıcılığı** (gerçek disk testi).
- **`kutuphane/Wal.tan` — undo-log tabanlı WAL.** Sabit-boyutlu kayıtlar
  (satır-tabanlı format KULLANILMADI — sayfa içeriği rastgele bayt/newline
  taşıyabilir). `walAc/walEkle/walKayitOku/walSonLsn/walKapat`.
- **`kutuphane/Islem.tan` — transaction katmanı.** `islemBaslat/
  islemSayfaYaz` (write-ahead: önce WAL, sonra gerçek sayfa)/`islemCommit/
  islemRollback` (kendi WAL lsn aralığını geriye tarayıp undo yapar).
- **BULUNAN BUG (yeni):** TAN fonksiyonları top-level (dosya-seviyesi)
  global değişkene GÜVENİLİR ERİŞEMİYOR — modül-seviyesi bir sayaç +
  onu okuyan yardımcı fonksiyon SEGFAULT verdi. Düzeltme: global state
  kaldırıldı, çağıran kendi sayacını (bir "kutu") parametre olarak besliyor.
- **ACID durumu (dürüst):** Atomicity EVET (rollback doğrulandı — tek
  sayfa VE çok-sayfalı senaryo), Consistency uygulama sorumluluğu,
  **Isolation YOK** (tek-thread varsayımı, 2D hâlâ yazılmadı),
  **Durability KISMİ** (fsync yok — 2B'nin sınırı; crash-recovery/REDO bu
  eklemenin kapsamı DIŞINDA, sadece çalışan process içi rollback var).
- **Doğrulama:** `testler/storage_testleri.tan` — 13/13 GEÇTİ (temel
  ayır/yaz/oku, kalıcılık, commit, rollback tek-sayfa, rollback çok-sayfa),
  `TestAraclar.sh`'ye otomatik dahil (kalıcı regresyon), iki kez üst üste
  çalıştırılıp idempotent olduğu doğrulandı. `veta/tests/test_file_io.tan`/
  `test_storage.tan`/`test_wal.tan`/`test_transaction.tan` (önceden hiç
  derlenmeyen sahte testlerdi) yeni API ile çalışır hale getirildi.
  TancElf.tan bu turda DEĞİŞMEDİ (saf kütüphane kodu) — self-hosting
  riski yok, ama yine de tam regresyon (TestArkaUcGoSuzTemiz.sh,
  TestFormatIdempotent) yeşil doğrulandı.

#### Query (Sorgu) — YOK (Storage'ın üzerine kurulacak, henüz başlanmadı)
- `libraries/query/source/query.tan`: **SIFIR gerçek kod**, tamamı yorum/
  tasarım notu. Storage artık HAZIR (yukarı bakın) — Query artık
  gerçekten başlanabilir, ama bu oturumda YAZILMADI.
- **Gerçek durum:** Tasarım var, implementasyon yok. Sıradaki adım.

#### Transaction (İşlem) — Storage seviyesinde ÇÖZÜLDÜ (`kutuphane/Islem.tan`)
- Eski `transaction.tan`/`transaction_detail.tan` (tasarım notu, sıfır kod)
  hâlâ boş, ama gerçek transaction mantığı artık `kutuphane/Islem.tan`'da
  var ve çalışıyor (yukarı bakın — Storage bölümü). Query katmanının
  KENDİ transaction ihtiyaçları (çok-tablolu/karmaşık sorgu işlemleri)
  ayrıca ele alınabilir ama temel commit/rollback hazır.

**Genel not (2026-08-22 güncelleme):** 2026-08-21'deki "Faz 6 tasarım
aşamasında, kod yazımı HENÜZ BAŞLAMADI" tespiti bu oturumda KAPANDI —
Storage artık gerçek, test edilmiş, kalıcı bir motor. Demir'in "VETA'yı
PostgreSQL-seviyesi programdan ileri taşıma" hedefinin önündeki asıl
engel (Storage yokluğu) kalktı. Sıradaki adım Query (SIFIR kod, henüz
başlanmadı) + Isolation/2D (tek-thread sınırı hâlâ geçerli).

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
6. ✅ **2B gerçekten düzeltildi (2026-08-22, commit `01a23f9`)** — kök sebep bozuk register kullanımıymış (T2/T3 değil), yeniden yazıldı, gerçek rastgele-erişim testiyle doğrulandı. Storage'ın önkoşulu tamam.
7. ⚙️ **Storage'ı gerçekten yaz** (page_manager + WAL + transaction) — artık 2B üzerine kurulabilir, SIRADAKİ adım.
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
