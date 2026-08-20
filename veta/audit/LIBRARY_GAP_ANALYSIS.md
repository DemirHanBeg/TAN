# LIBRARY_GAP_ANALYSIS.md

*Tarih: 2026-08-16. TAN'ın self-hosted (SELF) katmanındaki eksik kütüphaneler,
önem sırasına göre. Her satır TAN_CAPABILITY_AUDIT.md'deki gerçek duruma dayanır.
Öncelik = bağımlılık grafiği (LIBRARY_DEPENDENCY_GRAPH.md) × kullanım değeri ×
mevcut engel.*

---

## 1. Özet Tablo

| # | Kütüphane | SELF durum | GO-ELF durum | Engel | Öncelik |
|---|---|---|---|---|---|
| 1 | math (tam) | HAZIR | HAZIR | yok | **BUGÜN** |
| 2 | string (karakter/byte) | HAZIR (kısıtlı) | HAZIR | tip çıkarımı sınırı | **BUGÜN** |
| 3 | collection (liste) | PARTIAL | HAZIR | liste parametre tipi | Yüksek |
| 4 | option/result | KISMEN | HAZIR | çoklu dönüş yok | Yüksek |
| 5 | error | KISMEN | HAZIR | dene/yakala yok | Yüksek |
| 6 | bytes / memory | BLOCKED | HAZIR | ham bellek API yok | P0 önkoşul |
| 7 | io | KISMEN | HAZIR | konumlamalı G/Ç yok (2D) | P0 önkoşul |
| 8 | file | BLOCKED | HAZIR | 2D: f_dosya_ac_* yok | P0 önkoşul |
| 9 | concurrency | BLOCKED | HAZIR | 2E: futex/iplik/kilit yok | P0 önkoşul |
| 10 | thread/sync/atomic | BLOCKED | HAZIR | 2E | P0 önkoşul |
| 11 | time | UNKNOWN→ENGEL | KISMEN | saat syscall ilkeli yok | Düşük |
| 12 | serialization | BLOCKED | PARTIAL | string/bytes zemin | Orta |
| 13 | storage/database/cache/query/transaction | BLOCKED | PARTIAL | io+file+sözlük | Uzak |
| 14 | os/network/socket | BLOCKED | MISSING | ham syscall API yok | Uzak |
| 15 | compression/crypto | BLOCKED | MISSING | temel katmanlar | Uzak |
| 16 | event/distributed/graph/semantic/ai_memory/observability/optimizer/plugin/autonomy/evolution | BLOCKED | çoğu MISSING | tüm zemin | Uzak |

## 2. Öncelikli Boşluklar — Gerekçe

### 2.1 P0 Önkoşullar (self-hosted derleyici iyileştirmeleri)

Bunlar kütüphane DEĞİL, derleyici yetenek boşluklarıdır; kapatılmadan üst
katmanlar yazılamaz:

| Kod | Boşluk | Kanıt | Çözüm yönü |
|---|---|---|---|
| T1 | Dönüş tipi çıkarımı `metinBirlestir`/`metinAl` çağrılarını metin saymıyor | TancElf.tan 2896, 2952 (`yerlesikMetinDonerMi`), 3016 | listeye ekle |
| T2 | Değişken üzerinden metin dönüşü tanınmıyor (`döndür t`) | TancElf.tan 3024 (`hepsiMetinMi = 0`) | "metin-kanıtlı değişken" takibi |
| T3 | Parametre tipi yalnız çağrı-yerinde literal/yerleşik çağrıyla çıkıyor | TancElf.tan 3242 `argumanMetinMi` | değişken argüman izleme |
| T4 | Konumlamalı dosya G/Ç (2D) SELF'te yok | f_dosya_ac_* yalnız DerleElf.go 4303+ | SELF'e taşı |
| T5 | Eşzamanlılık (2E) SELF'te yok | f_futex/f_iplik/f_kilit/f_atomik yalnız DerleElf.go 4587+ | SELF'e taşı |
| T6 | Ham bellek/bytes API kullanıcıdan erişilebilir değil | f_tan_ayir/f_bellek_kopyala yalnız derleyici gömüyor | kullanıcı ilkeli tanımla |
| T7 | Sözlük SELF'te yok | f_sozluk_* yalnız DerleElf.go 3564+ | 2A: SELF'e taşı |

### 2.2 Bugün Yazılabilir Kütüphaneler (engel yok)

1. **math** — `mutlak`, `enBuyuk`, `enKucuk`, `faktoriyel`, `tamUssu`,
   `ebob`, `ekok`, `tekMi`, `ciftMi`, `asalMi` (tam çekirdek; kesir katmanı
   ayrı deney — parametre tipi sınırı).
2. **string (karakter/byte)** — `harfMi`, `rakamMi`, `bosMu`, `kucukHarfMi`,
   `buyukHarfMi`, `metinSayiMi`, `metinIcerirMi`, `kucukHarf`, `buyukHarf`
   (yalnız tip-bağımsız yerleşikler + literal-çağrı deseni).
3. **collection** — `listeKopyala`, `listeToplam`, `listeEnBuyuk`,
   `listeTers` (liste parametreleri tip yığınına güvenmez; işlevler tam döner).

## 3. "Yerinde Saymama" Kuralı

Yukarıdaki 16 satırın HEPSİ "yazılacak" değildir. VETA ilerlemesi:
1. P0 (T1-T3) self-hosted derleyicide kapatılana kadar 1-3 yazılır/doğrulanır.
2. 2D/2E (T4/T5) taşınmadan 7-10 başlatılmaz.
3. HIGH LEVEL katmanlar (16) için zemin matrisi hazır olmadan adım atılmaz
   — "sonradan eklenmiş yabancı parça" üretmek VETA ilkesine aykırıdır
   (KUTUPHANE KURALI, soru: «doğrudan TAN'ın standard kütüphanesi olarak
   dağıtılabilir mi?» — cevap hayır ise tasarımı yeniden düşün).
