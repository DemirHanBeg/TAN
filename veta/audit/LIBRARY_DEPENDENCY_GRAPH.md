# LIBRARY_DEPENDENCY_GRAPH.md

*Tarih: 2026-08-16. Kütüphane bağımlılık grafiği, mevcut TAN capability'sine
(TAN_CAPABILITY_AUDIT.md §4) göre çizilmiştir. Ok yönü "bağımlıdır" demektir.
Düzleştirme kuralları: (1) TAN-native, (2) düşük bağımlılık, (3) self-host
uyumluluğu. BLOCKED = mevcut self-hosted derleyici (TancElf.tan) alt kümesinde
henüz yazılamayan katman.*

---

## 0. Dil-Runtime Zemin (dil çekirdeği, kütüphane DEĞİL)

Bu katman kütüphane olarak değil, derleyicinin gömülü runtime'ıdır
(TancElf.tan 4271-4272 helper kümesi). Tüm kütüphanelerin dolaylı zemini:

```
f_tan_ayir (brk bump allocator) ──► f_bellek_kopyala
    │                                   │
    ├─► f_liste_yap / f_liste_ekle      ├─► f_metin_birlestir
    ├─► f_sayi_metne / f_kesir_metne    ├─► f_metin_indeks (metinAl)
    └─► f_metin_esit                    ├─► f_kod / f_karakter
```

## 1. FOUNDATION (temel, az bağımlı)

```
math                        [HAZIR — bağımsız yaprak, tam sayı çekirdeği]
   └── hiçbir kütüphaneye bağımlı değil

string (byte/karakter katmanı)   [HAZIR — kısıtlı alt küme]
   └── dil yerleşikleri: metinBirlestir, metinAl, metinEsit, uzunluk,
       kod, karakter, metin(n)
   └── tip-çıkarımı sınırı: parametre/dönüş izleme yalnızca kısıtlı
       kalıpları tanır (bkz. TAN_CAPABILITY_AUDIT.md §7)

memory / bytes            [BLOCKED — dil düzeyinde ham bellek API yok]
   ├── f_tan_ayir / f_bellek_kopyala runtime'da VAR ama kullanıcı
   │   programından çağrılamıyor (yalnız derleyici gömüyor)
   └── bytes kütüphanesi için ham işaretçi+boyut değeri TAN'da yok

array / list              [PARTIAL — dilin liste ilkelleri kullanılır]
   └── [..] literal, uzunluk, ekle, listeYap, l[i], l[i]=x
   └── liste PARAMETRESİ olan işlevler tip-çıkarımına güvenemez
       (parametreler "tam" varsayılır) → dikkatli alt küme gerekir

map / set                 [BLOCKED — sözlük SELF katmanında yok]
   └── sözlük yalnız GO-ELF yolunda (DerleElf.go f_sozluk_*)

option / result           [KISMEN — tam sayı kodlarıyla taklit edilebilir]
   └── dilde iki değerli sonuç döndürmek yok → (kod, değer) paralel
       liste / sentinel deseni; BELGELENMESİ ŞART

error                     [KISMEN]
   ├── derleme hatası: yaz(...) + sistemDur(1)
   └── runtime sıfıra bölme: exit(4)
   └── özel hata değeri/istisna yok (dene/yakala SELF'te yok)

## 2. CORE

```
io        [KISMEN — oku / yazBaytlar / dosyaVarMi var; konumlamalı G/Ç YOK]
   └── string
   └── file [BLOCKED 2D — f_dosya_ac_* yalnız GO-ELF'te]

concurrency  [BLOCKED 2E — f_futex/f_iplik/f_kilit/f_atomik yalnız GO-ELF'te]
   └── thread [BLOCKED]
   └── sync   [BLOCKED]
   └── atomic [BLOCKED]

time      [KISMEN — syscall tabanlı saat okuyacak ilkel YOK; UNKNOWN durumu]
process   [KISMEN — arg/argsay + sistemDur + yazBaytlar; exec/fork YOK]
```

## 3. SYSTEM

```
os / network / socket        [BLOCKED — ham syscall emisyonu yalnız derleyici içi]
serialization                [BLOCKED — f_birlestir_*/f_parcala GO-ELF'te]
compression / crypto         [BLOCKED — temel katmanlar hazır değil]
```

## 4. DATA

```
storage   [BLOCKED 2D — konumlamalı dosya G/Ç gerekli]
   └── file (CORE)
database  [BLOCKED — storage + sözlük gerekli]
   └── storage, sözlük (GO-ELF'te)
cache     [BLOCKED — sözlük + concurrency gerekli]
query     [BLOCKED — database gerekli]
transaction [BLOCKED — storage + concurrency gerekli]
```

## 5. HIGH LEVEL

```
event / distributed / graph / semantic / ai_memory / observability /
optimizer / plugin / autonomy / evolution
    └── HEPSİ BLOCKED — yukarıdaki zemin hazır olmadan başlamak
        "sonradan eklenmiş yabancı parça" üretir, VETA ilkesine aykırı.
```

---

## 6. Bağımlılık Zinciri (hedef durum)

```
memory ─► bytes ─► string ─► serialization ─► storage ─► database ─► query
atomic ─► lock ─► futex ─► thread ─► concurrency ─► transaction
file  ─► storage ─► database
io ─► string;  math ─► (bağımsız)
```

## 7. VETA'nın Kritik Yol Bağımlılığı

Self-hosted derleyicinin TİP ÇIKARIMI sınırı (TAN_CAPABILITY_AUDIT.md §7)
`string`/`bytes`/`serialization` katmanının GENEL hali için anahtar:
parametre tipi + değişken-dönüş tipi izleme genişletilmeden metin döndüren
kütüphane işlevleri yalnız "doğrudan literal / bilinen yerleşik" kalıbıyla
yazılabilir. Bu yüzden:

- **P0 (ön koşul):** `govdeDonusTipiCikar` + `yerlesikMetinDonerMi` +
  `argumanMetinMi`'ye `metinBirlestir`/`metinAl` eklemek ve "metin-kanıtlı
  değişken dönüşü" tanımak. (TancElf.tan değişikliği → sabit nokta yeniden
  doğrulaması şart; bu ortamda qemu yavaşlığı yüzünden AYRI oturumda.)
- **Bugün yazılabilir:** math (tam çekirdek), string (karakter/byte katmanı),
  collection (dikkatli alt küme), option/result (sentinel deseni).
