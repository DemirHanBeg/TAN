# mat.tan — Math Foundation Library

TAN-native tam sayı matematik kütüphanesi. Self-hosted derleyici (TancElf.tan)
ile derlenir, `içe al` ile içe alınır veya doğrudan kaynak kodlanır.

## Fonksiyonlar (11)

| Fonksiyon | Tanım | Parametre | Sonuç |
|---|---|---|---|
| `mutlak(x)` | \|x\| | tam | tam |
| `enBuyuk(a, b)` | max(a, b) | tam, tam | tam |
| `enKucuk(a, b)` | min(a, b) | tam, tam | tam |
| `tekMi(n)` | n tek ise 1, değilse 0 | tam | tam (0/1) |
| `ciftMi(n)` | n çift ise 1, değilse 0 | tam | tam (0/1) |
| `faktoriyel(n)` | n! (n ≥ 0, özyinelemeli) | tam | tam |
| `tamUssu(taban, us)` | taban^us (us ≥ 0) | tam, tam | tam |
| `ebob(a, b)` | Euclidean en büyük ortak bölen | tam, tam | tam |
| `ekok(a, b)` | En küçük ortak kat | tam, tam | tam |
| `tamKareKok(n)` | ⌊√n⌋ | tam | tam |
| `asalMi(n)` | Asal testi (deneme bölme) | tam | tam (0/1) |

## Doğrulanmış Test Sonuçları

24 test durumunun tamamı `qemu-x86_64` altında derlenip koşuldu:
```
17, 42, 20, 10, 1, 0, 0, 1, 3628800, 120, 1024, 243,
6, 36, 1, 1, 0, 1, 0, 0, 1, 2, 3, 10
```

## Sınırlar ve Uyarılar

1. **Yalnız tam sayı.** Kesir (float) parametreleri tip-çıkarımı nedeniyle "tam"
   algılanır; kesir fonksiyonları bu kütüphanede DEĞİLDİR.
2. **`/` işleci kesir yolu.** TAN'da `/` HER ZAMAN float64 bölmeye gider
   (A1 kararı, TancElf.tan 3695-3738). Tamsayı bölmek için `tamBol()` kullanın.
   `ekok()` bunu zaten yapıyor.
3. **Overflow riski.** `faktoriyel`, `tamUssu`, `ekok` büyük değerlerde int64
   taşması yapar (2^63-1 sınırlaması).
4. **`içe al` ile kullanım.** `içe al "mat.tan"` (aynı dizin) veya göreli yolla
   alınabilir. Fonksiyonlar birbirini çağırıyor (asalMi → tamKareKok, tekMi);
   forwards reference derleyicide desteklenir.
5. **Derleme hızı.** Qemu altı PRoot/Android'de derleme değişkendir (45s-10dk);
   testler batch ve tek komutta koşulmalıdır.

## Kullanım

```tanc
içe al "mat.tan"

yaz(faktoriyel(10))    # 3628800
yaz(asalMi(97))        # 1
yaz(ekok(12, 18))      # 36
```
