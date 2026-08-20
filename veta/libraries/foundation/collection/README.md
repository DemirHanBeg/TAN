# koleksiyon.tan — Collection (Liste) Foundation Library

TAN-native liste yardımcı kütüphanesi. Self-hosted derleyici (TancElf.tan)
ile derlenir.

## Fonksiyonlar (7)

| Fonksiyon | Tanım | Parametre | Sonuç |
|---|---|---|---|
| `listeToplam(l)` | Tüm elemanların toplamı | liste (tam) | tam |
| `listeEnBuyuk(l)` | En büyük eleman | liste (tam) | tam |
| `listeKucuk(l)` | En küçük eleman | liste (tam) | tam |
| `listeSay(l, deger)` | deger eşleşen eleman sayısı | liste, tam | tam |
| `listedeVarMi(l, deger)` | deger listede var mı? | liste, tam | tam (0/1) |
| `listePozitif(l)` | Pozitif eleman sayısı | liste (tam) | tam |
| `listeNegatif(l)` | Negatif eleman sayısı | liste (tam) | tam |

## Doğrulanmamış Durum

qemu CPU throttle nedeniyle test edilemedi. Tüm fonksiyonlar aynı
güvenli deseni izler:
- Liste parametreleri `tam` (pointer) olarak işlenir → `l[i]` doğru
  listeden değer okur (tip-bağımsız kod üretimi).
- `uzunluk(l)` tip-bağımsızdır.
- Döngü + karşılaştırma + toplama/sayıma mantığı mat.tan ile aynı.

**Test bekliyor:** cpu throttle stabilleştiğinde `test_kol_small.tan`
ile doğrulanacak.

## Sınırlar

1. Liste elemanları yalnız tam sayı. Karışık tipli listeler desteklenmez.
2. `listeEnBuyuk`/`listeKucuk` boş liste için 0 döner (undefined-safe).
3. Liste döndüren fonksiyonlar (kopyala, ters) `tam` tipinde pointer
   döner — `yaz()` ile doğrudan yazdırılamaz, `l[i]` ile erişilir.

## Kullanım

```tanc
içe al "koleksiyon.tan"
l = [3, 5, 7, 10]
yaz(listeToplam(l))      # 25
yaz(listeEnBuyuk(l))     # 10
yaz(listedeVarMi(l, 5))  # 1
```
