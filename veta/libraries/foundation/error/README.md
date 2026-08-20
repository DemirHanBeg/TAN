# hata.tan — Error Foundation Library

TAN-native hata kodu sabitleri ve hata yönetim yardımcıları.
Dilde `dene`/`yakala` (try/catch) mekanizması yok (SELF katmanında),
bu yüzden hata yönetimi sentinel deseni (secenek.tan) ile yapılır.

## Hata Kodları

| Kod | Anlam |
|---|---|
| 0 | Başarılı |
| 1 | Genel hata |
| 2 | Argüman hatası |
| 3 | Dosya hatası |
| 4 | Bölme hatası (sıfıra bölme) |
| 5 | Erişim hatası (index out of bounds) |
| 6 | Bellek hatası |
| 7 | Tip hatası |

## Fonksiyonlar (1)

| Fonksiyon | Tanım | Sonuç tipi |
|---|---|---|
| `hataKoduMetin(kod)` | Hata kodunu okunabilir metne çevir | metin |

## Doğrulanmamış Durum

Test edilmedi (qemu throttle). Tasarım basit:
- İç içe `eğer/değilse` zinciri (switch-case deseni).
- Her dal `döndür "metin_literali"` — dönüş tipi "metin" olarak tanınır.
- Hata kodları sabit tam değerler — karşılaştırma `kod == X` ile yapılır.

## Kullanım

```tanc
içe al "hata.tan"
# hataKoduMetin(4) → "bolme hatasi"
yaz(hataKoduMetin(4))
```

## Sınırlar

1. Hata kodları sabit — çalışma zamanında genişletilemez.
2. Hata zincirleme (error chaining) yok.
3. `dene`/`yakala` olmadığından, hata oluştuğunda program akışı
   `eğer kod != 0 ise` ile kontrol edilmeli (boşaltma yolu deseni).
