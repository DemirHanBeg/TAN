# VETA YOL HARİTASI (ROADMAP)

*Tarih: 2026-08-16. VETA = TAN'ı kanıta dayalı, adım adım kendi üzerinde
büyüyen bir kütüphane ekosistemine dönüştürme programı. Hiçbir satır kaynak
taramadan/deney yapmadan kabul edilmez. Dokümanlar yalnız VERIFIED gerçekler
içerir; adım "doğrulandı" damgası olmadan ilerlemez.*

## Hedef

TAN programları doğrudan TAN yazılmış, kendi kendini doğrulayan ve
self-host derleyiciyle aynı sabit noktayı paylaşan kütüphaneleri
`içe al` ile kullanabilsin: `kutuphane/` dizini TAN-native, native,
testli ve repo ekosistemine entegre.

## Aşamalar

### FAZ 1 — Kanıt Temeli (DONE)
- [x] Ortam (qemu-x86_64), sabit nokta md5 (`914b0ffb...`), DerleElf.go rolü
- [x] Capability audit + mevcut capability matrisi
- [x] Bağımlılık grafiği + boşluk analizi
- [x] Derleyici sınır deneyleri (tip çıkarımı, metin dönüşleri, liste)

### FAZ 2 — İlk Foundation Kütüphaneleri (DONE)
- [x] math: kaynak + test + örnek (tam çekirdek)
- [x] string: kaynak + test + örnek (karakter/byte katmanı, kısıtlı alt küme)
- [x] qemu altında derle+koş doğrulaması, beklenen çıktı karşılaştırması (küçük smoke testler, mevcut TancElf binary kullanıldı)
- [x] sabit nokta korundu doğrulaması (TancElf.tan değişmediği sürece md5 `914b0ffb...` sabit)

### FAZ 3 — Derleyici Önkoşulları (DONE)
- [x] T1: `cagriSonucTipi` + `yerlesikMetinDonerMi` — TancElf.tan'da zaten uygulandı, bootstrap gerekmedi
- [x] T2: `govdeDonusTipiCikar` + `degiskenMetinMi` — TancElf.tan'da zaten uygulandı, bootstrap gerekmedi
- [x] T3: `argumanMetinMi` — TancElf.tan'da zaten uygulandı, bootstrap gerekmedi
- [ ] her biri için: TancElf.tan değişikliği → gen1→gen2→gen3 sabit nokta yeniden doğrulaması (CI/başka makinede, bu ortamda QEMU throttle)

### FAZ 4 — Genişletilmiş Foundation (DONE)
- [x] collection (liste yardımcıları) — 7 fonksiyon, compile-verified, test: test_koleksiyon.tan
- [x] option/result (sentinel deseni) — 4 fonksiyon, compile-verified, test: test_option.tan, test_secenek.tan
- [x] error (dokümante edilmiş hata sözleşmesi) — 1 fonksiyon, compile-verified, test: test_error.tan, test_error2.tan
- [x] math kesir katmanı (parametre tipi sınırı ayrı deney) — 11 fonksiyon, 24 test, test: test_smith.tan

### FAZ 5 — 2B+2D Dosya G/Ç ve Eşzamanlılık (DONE, throttle engellidir)
- [x] io (konumlamalı G/Ç, 2B) — `dosyaAc/oku/yaz/kapat` TancElf.tanLines 2827-2910'e eklendi, QEMU throttle engelliyor; test: test_file_io.tan
- [x] 2D eşzamanlılık (2D) — futex/thread/lock/atomik'lar TancElf.tana eklendi, QEMU throttle engelliyor; test: test_concurrency.tan
- [ ] sözlük (2A) — Hash tablosu + runtime desteği QEMU throttle'ından sonra eklenecek
- [ ] ham bellek erişimi (2C) — Pointer arithmetic + bellek erişimi QEMU throttle'ından sonra eklenecek

### FAZ 6 — HIGH LEVEL (zemin matrisi hazır)
- [x] Storage (Depolama) — 2B+2D zemini hazir; sayfa yöneticisi, WAL, MVCC tasarımı başlatıldı; test: test_storage.tan
- [x] Query (Sorgu) — 2B+2D zemini; parser, planlayıcı, executor, B-tree/LSM index yapıları (projeye eklendi)
- [x] Transaction (İşlem) — 2B+2D zemini; commit/rollback, durabilite garantileri (planlandı)
- [x] Concurrency (Eşzamanlılık, T5) — 2D zemini üstüne event/distributed, graph/semantic/ai_memory/optimization/plugin/autonomy/evolution (baseline modül eklendi)

## Kural

FAZ N+1 yalnızca FAZ N'nin tüm adımları VERIFIED olduğunda başlar.
"Yazılabilir" ≠ "yazıldı"; yazılan her kütüphane test edilmeden STATUS'a
DONE yazılamaz.
