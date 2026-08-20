# VETA

**Verification-first Evolution of TAN Abilities**

VETA, TAN dilini kanıta dayalı, adım adım kendi üzerinde büyüyen bir
kütüphane ekosistemine dönüştürme programıdır. Bu dizin, çalışmanın
tümünü barındırır: denetim (audit), yol haritası (roadmap), durum
(status) ve kütüphane implementasyonları (libraries/).

## İlkeler

1. **Kanıt önce gelir.** Her iddia kaynak taramasına veya çalıştırılmış
   deneye dayanır. `UNKNOWN`/belirsiz durum yerine deney yapılır.
2. **Adım adım.** FAZ N+1, FAZ N'nin tüm adımları doğrulanmadan başlamaz
   (VETA_ROADMAP.md).
3. **Derleyici dokunulmazlığı.** TancElf.tan değişikliği gerektiren adımlar
   (FAZ 3, P0: T1-T3) sabit nokta yeniden doğrulamasına tabidir ve bu ortamda
   ayrı oturumda yapılır. Kütüphane çalışmaları derleyiciyi değiştirmez.
4. **Yerinde sayma.** "Yazılabilir" ≠ "yazıldı". Kütüphane test edilmeden
   DONE sayılmaz. HIGH LEVEL katmanlar zemin hazır olmadan başlamaz.
5. **Repo entegrasyonu.** Her kütüphane `içe al` ile doğrudan TAN programından
   kullanılabilmeli ve repo'nun standard kütüphanesi gibi dağıtılabilmeli.

## Dizin Yapısı

```
veta/
├── README.md                     # bu dosya
├── VETA_ROADMAP.md               # aşamalar ve doğrulama kuralları
├── VETA_STATUS.md                # oturum sonu gerçek durum
├── CURRENT_TAN_CAPABILITIES.md   # mevcut capability matrisi (SELF/GO-ELF/GO-YORUM)
├── audit/
│   ├── TAN_CAPABILITY_AUDIT.md   # ortam, sabit nokta, 2A-2E kanıtları
│   ├── LIBRARY_DEPENDENCY_GRAPH.md
│   └── LIBRARY_GAP_ANALYSIS.md
├── libraries/
│   ├── foundation/               # FAZ 2: math, string, collection, ...
│   ├── core/                     # FAZ 5+: io, bytes, memory, sözlük
│   ├── storage/ transaction/ query/ ...   # FAZ 6 (zemin hazır olunca)
│   └── (kalan üst katmanlar zemin matrisine göre açılır)
├── tests/                        # kütüphane testlerinin toplu koşum merkezi
├── examples/
└── experiments/                  # kanıt deneyleri (strtest*, prog.tan, ...)
```

## Kütüphane Sözleşmesi

- Her kütüphane `source/` (TAN kaynağı), `tests/`, `examples/`, `README.md`
  içerir.
- Test dosyası kütüphaneyi `içe al` ile alır, fonksiyonları çağırır ve
  `yaz(...)` ile beklenen değerleri basar; doğrulama çıktı karşılaştırmasıyla
  yapılır (çıktıdaki son iki satır: `hepsi tamam` + sıfır dışı sistemDurumu
  yoksa başarılı).
- Kütüphaneler yalnızca mevcut self-hosted alt kümenin güvenli desenlerini
  kullanır (bkz. TAN_CAPABILITY_AUDIT.md §7). Sınırlar her kütüphanenin
  README'sinde açıkça belgelenir.

## Doğrulama Zinciri

Her test: `qemu-x86_64 TancElf <test.tan> <ikili>` derler, `qemu-x86_64 <ikili>`
çalıştırır, çıktı beklenenle karşılaştırılır. Derleme süresi 45s-6dk arasıdır;
testler batch ve tek komutta koşulur.
