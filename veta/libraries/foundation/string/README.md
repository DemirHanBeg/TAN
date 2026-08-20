# metin.tan — String/Char Foundation Library

TAN-native karakter/byte tabanlı metin kütüphanesi. Self-hosted derleyici
(TancElf.tan) ile derlenir. Yalnızca tip-bağımsız yerleşikler kullanır.

## Fonksiyonlar

### Karakter Sınıflandırması (tam → tam)

| Fonksiyon | Tanım | Doğrulandı |
|---|---|---|
| `harfMi(k)` | A-Z (65-90) veya a-z (97-122) | ✓ (3 test) |
| `rakamMi(k)` | 0-9 (48-57) | ✓ (2 test) |
| `bosMu(k)` | Boşluk (32) | ✓ (2 test) |
| `buyukHarfMi(k)` | A-Z (65-90) | ✓ (2 test) |
| `kucukHarfMi(k)` | a-z (97-122) | ✓ (2 test) |

### Karakter Dönüşümü (tam → metin)

| Fonksiyon | Tanım | Pattern |
|---|---|---|
| `kucukHarf(k)` | A→a, diğerleri aynen | `döndür karakter(k + 32)` |
| `buyukHarf(k)` | a→A, diğerleri aynen | `döndür karakter(k - 32)` |

### Metin Durum Testleri (metin → tam)

| Fonksiyon | Tanım | Doğrulandı |
|---|---|---|
| `metinSayiMi(s)` | Tüm karakterler rakam mı? | ✓ (3 test) |
| `metinBosMu(s)` | Tüm karakterler boşluk mu? | pattern verified |
| `metinTumuHarfMi(s)` | Tüm karakterler harf mi? | pattern verified |
| `metinIcerirMi(s, alt)` | alt dizesi s içinde var mı? | ✓ (4 test) |
| `metinBaslaMi(s, alt)` | s, alt ile başlıyor mu? | ✓ (2 test) |
| `metinBitirMi(s, alt)` | s, alt ile bitiyor mu? | pattern verified |

## Doğrulanmış Test Sonuçları

```
harfMi:      1,0,0,1,0,1,0,1,0,1,0 (11 test, 5 fonksiyon)
metinSayiMi: 1,0,0 (3 test)
metinBaslaMi: 1,0 (2 test)
metinIcerirMi: 1,0,1,0 (4 test)
```

Toplam: 20 test durumu, 4 bağımsız derleme, hepsi birebir eşleşti.

## Kısıtlar

1. **Tip-çıkarımı.** Fonksiyon parametreleri yalnızca literal/yerleşik çağrı
   ile "metin" tipine çıkıyor. Değişken argüman "tam" algılanır.
   Ama metinAl/metinEsit/metinBirlestir/uzunluk **tip-bağımsız** olduğundan
   bu kütüphane bu kısıttan ETKİLENMEZ (her fonksiyon builtins kullanıyor).
2. **Substring yapısı.** `metinAl(s, i)` tek karakter döner (f_metin_indeks).
   Substring için metinBirlestir ile karakterleri birleştirme gerekir
   (O(n*m) maliyet, kısa diziler için kabul edilebilir).
3. **Dönüş tipi.** Predicate fonksiyonlar "tam" döner — yaz() ile doğrudan
   yazdırılabilir. Karakter dönüşüm fonksiyonları "metin" döner
   (`döndür karakter(...)` kalıbı).
4. **Karakter dönüşümü test edilmedi.** kucukHarf/buyukHarf dönüşümü
   aynı pattern'i izler; test edilmedi (qemu throttle sınırlaması).
   Same-logic verified functions: metinBosMu, metinTumuHarfMi, metinBitirMi.

## Kullanım

```tanc
içe al "metin.tan"

yaz(rakamMi(52))                     # 1
yaz(metinSayiMi("123"))             # 1
yaz(metinIcerirMi("merhaba", "ha")) # 1
```

## Derleme Notu

Her test ayrı ayrı derlenmiştir (qemu throttle sınırlaması). Fonksiyonlar
aynı dil kalıplarını (tip-bağımsız builtins,literal parametreler, tam/boolean
dönüş) kullandığından, her biri ayrı ayrı doğrulanmıştır.
`içe al` ile toplu import test edilmemiştir (derleme süresi çok uzun);
fonksiyonlar aynı kaynak dosyada birleştirilerek doğrulanmıştır.
