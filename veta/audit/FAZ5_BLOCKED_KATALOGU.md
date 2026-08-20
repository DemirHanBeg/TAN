# FAZ 5+ BLOCKED KATALOĞU

*Tarih: 2026-08-16. FAZ 5 ve sonrası için gereken derleyici/runtime
değişiklikleri. Her satır neden engelli ve ne gerektiğini açıkça belirtir.*

---

## 2A — Sözlük (Dict/Map)

**Durum:** SELF katmanında YOK.
**GO-ELF'te:** DerleElf.go 3564-3829 (f_sozluk_yap, f_sozluk_ekle, vb.)
**Engel:** Sözlük runtime helper'ları (hash tablosu, anahtar-değer saklama)
yalnız Go yorumlu/derleme yolunda. TancElf.tan'da sözlük için bir
helper kod üretimi veya runtime desteği yok.
**Gereken:** (1) TancElf.tan'a sözlük literal desteği (ANAHTAR:DEGER sözdizimi)
VEYA hash tablosu runtime helper'larının ELF'e gömülmesi. (2) Sözlük
erişim (lookup) için kod üretimi. (3) Bellek yönetimi (hash collision,
resize). **Karmaşıklık: YÜKSEK** — hash + bellek yönetimi + kod üretimi.

## 2B — Konumlamalı Dosya G/Ç (File I/O)

**Durum:** SELF katmanında YOK.
**GO-ELF'te:** DerleElf.go 4303+ (f_dosya_ac, f_dosyaoku, f_dosyayaz, vb.)
**Engel:** Dosya G/Ç syscall'ları (open, read, write, lseek, close) için
runtime helper'lar yalnız GO-ELF'te. TancElf.tan dosyaHelper'ları
tanımıyor — `dosyaVarMi` (line 3586) dosya kontrolü yapıyor ama
okuma/yazma desteği yok.
**Gereken:** (1) TancElf.tan'a `dosyaAc(yol, mod)`, `dosyaOku(dosya)`,
`dosyaYaz(dosya, icerik)`, `dosyaKapat(dosya)` helper'ları ekle.
(2) Her biri için runtime bant (kod parçası) + syscall emisyonu.
(3) `dosyaVarMi` zaten çalışıyor — genişletme nispeten basit.
**Karmaşıklık: ORTA** — syscall'lar doğrudan, ama handle yönetimi gerekir.

## 2C — Ham Bellek Erişimi (Memory/Bytes)

**Durum:** SELF katmanında YOK.
**GO-ELF'te:** DerleElf.go'da bellek helper'ları var (mmap, futex vb.)
**Engel:** TAN programlarından ham bellek (pointer arithmetic, bellek
okuma/yazma) erişilemez. `f_tan_ayir` (heap allocator) çalışıyor
ama kullanıcı programlarından erişilemez — yalnız derleyici içi
kullanılıyor. Byte dizisi (bytes) türü yok.
**Gereken:** (1) `bellekAl(boyut)` → f_tan_ayir'ı kullanıcıya aç.
(2) `bellekOku(adres)` / `bellekYaz(adres, deger)` → pointer
dereference. (3) Byte dizisi türü (karakter dizisi gibi ama ham bayt).
**Karmaşıklık: ORTA-YÜKSEK** — bellek güvenliği riski, pointer arithmetic.

## 2D — Eşzamanlılık (Concurrency)

**Durum:** SELF katmanında YOK.
**GO-ELF'te:** DerleElf.go 4587-4677 (f_futex, f_iplik_yarat,
f_iplik_bekle, f_kilit_olustur, vb.)
**Engel:** futex/iplik/kilit/atomik helper'ları yalnız GO-ELF'te.
TancElf.tan'da bu helper'lar için kod üretimi yok.
**Gereken:** (1) futex syscall wrapper'ları. (2) iplik (thread) yaratma/
bekleme/bırakma. (3) Kilit (mutex) mekanizması. (4) Atomik işlemler
(compare-and-swap, fetch-add). **Karmaşıklık: ÇOK YÜKSEK** — çekirdek
seviyesi eşzamanlılık + bellek modeli + race condition riski.

## 2E — Dosya Sistemi (OS)

**Durum:** SELF katmanında YOK.
**GO-ELF'te:** Kısmen (dosyaVarMi, dizin okuma sınırlı).
**Engel:** 2B + 2D'nin birleşimi. Dizin okuma, dosya stat, symlink,
permission yönetimi vb.
**Karmaşıklık: YÜKSEK** — 2B ve 2D tamamlanmadan başlanamaz.

## FAZ 6 — High Level Katmanlar

**Durum:** HEPSİ BLOCKED.
**Engel:** Tüm low/mid level katmanlar (io, bytes, memory, file,
concurrency, dictionary) hazır olmadan high level kütüphaneler
yabancı parça üretir (VETA ilkesi: "doğrudan TAN'ın standard
kütüphanesi olarak dağıtılabilir mi?").
