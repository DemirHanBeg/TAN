# secenek.tan — Option/Result Foundation Library

TAN-native sentinel deseni ile çoklu değer dönüşü kütüphanesi.
Dilde `option`/`result` türü olmadığından (TAN_CAPABILITY_AUDIT.md §7),
liste tabanlı sentinel deseni kullanılır.

## Tasarım

```
Başarılı sonuç: [1, deger]  — liste[0]=1 (kod), liste[1]=değer
Hata sonucu:    [0, 0]      — liste[0]=0 (kod), liste[1]=tanımsız
```

## Fonksiyonlar (4)

| Fonksiyon | Tanım | Sonuç tipi |
|---|---|---|
| `secenekYap(deger)` | [1, deger] oluştur | liste (tam) |
| `secenekYapHata()` | [0, 0] oluştur | liste (tam) |
| `secenekBasariliMi(s)` | s[0]==1 mi? | tam (0/1) |
| `secenekDeger(s)` | s[1] değerini al | tam |

## Doğrulanmamış Durum

Test edilmedi (qemu throttle). Tasarım basit:
- `ekle([1], deger)` → liste oluşturup eleman ekleme (builtin `ekle`).
- `s[0]`, `s[1]` → indeks erişimi (liste eleman okuma).
- Tüm işlemler tip-bağımsız builtins veya indeks erişimi.

## Kullanım

```tanc
içe al "secenek.tan"
sonuc = secenekYap(42)
eğer secenekBasariliMi(sonuc) == 1 ise
    yaz(secenekDeger(sonuc))   # 42
değilse
    yaz(0)
son
```

## Sınırlar

1. Yalnız tam değerler. Metin değerler için ek koruma gerekir
   (tip-çıkarımı metin değişkenleri tanımıyor).
2. Hata kodu ayrımı yok — tüm hatalar `[0, 0]`.
3. İç içe sonuçlar desteklenmez (result-of-result).
