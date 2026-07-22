# Tan Dil Referansı

## İçindekiler
1. [Temeller](#temeller)
2. [Değerler ve Tipler](#değerler-ve-tipler)
3. [İşleçler](#i̇şleçler)
4. [Akış Denetimi](#akış-denetimi)
5. [İşlevler](#i̇şlevler)
6. [Veri Yapıları](#veri-yapıları)
7. [Modüller](#modüller)
8. [Yerleşik İşlevler](#yerleşik-i̇şlevler)
9. [Arka Uçlar](#arka-uçlar)
10. [Paket Yönetimi](#paket-yönetimi)

---

## Adlandırma Kuralı

Tan'da **alt çizgi kullanılmaz.** Birden çok kelimeden oluşan adlar camelCase yazılır:

| Yanlış | Doğru |
|---|---|
| `yaz_dosya` | `yazDosya` |
| `en_büyük` | `enBüyük` |
| `asal_mı` | `asalMı` |

Dosya adları PascalCase: `TancElf.tan`, `DerleElf.go`, `TestArkaUc.sh`.

Türkçe büyük harf kuralına uyulur: `i` → `İ`, `ı` → `I`.

*Uyumluluk:* eski alt çizgili yerleşik adlar (`yaz_dosya` gibi) hâlâ çalışır,
ama yeni kodda kullanma.

---

## Temeller

Dosya uzantısı `.tan`. Kaynak UTF-8. Satır sonu deyimi bitirir; noktalı virgül yok.

```tan
yaz("Merhaba Tan")
```

Yorum satırı `#` ile başlar:

```tan
# bu bir yorum
x = 5   # satır sonu yorumu
```

**Satır devamı:** satır sonunda bir işleç bırakırsan ifade sonraki satırda sürer.

```tan
toplam = 1 +
         2 +
         3
```

---

## Değerler ve Tipler

| Tip | Örnek | Not |
|---|---|---|
| Tam sayı | `42`, `-7` | int64, **kesin** — 2^63'e kadar hassasiyet kaybı yok |
| Ondalık | `3.14`, `1.5` | float64 |
| Metin | `"merhaba"` | UTF-8, kaçışlar: `\n` `\t` `\"` `\\` |
| Mantık | `doğru`, `yanlış` | |
| Liste | `[1, 2, 3]` | çok satırlı yazılabilir |
| Sözlük | `{"ad": "Tan"}` | |
| Yok | `yok` | boş değer |

### Sayı kuralları

Tan tam sayı ile ondalığı ayırır. Bu, büyük sayılarda doğruluğu korur:

```tan
yaz(123456789 * 987654321)   # 121932631112635269  (kesin)
```

Kurallar:
- `tam OP tam` → tam
- `tam OP ondalık` → ondalık
- `tam / tam` → bölünüyorsa tam, bölünmüyorsa ondalık *(yorumlayıcı ve C yolunda; `elf`/`asm` yolunda her zaman tam — bkz. [Arka Uçlar](#arka-uçlar))*

### Değişkenler

Bildirim yok, atama yeter:

```tan
x = 5
ad = "Demir"
```

---

## İşleçler

Önceliğe göre, düşükten yükseğe:

| Öncelik | İşleçler |
|---|---|
| 1 | `ve`, `veya` |
| 2 | `==` `!=` `>` `<` `>=` `<=` |
| 3 | `+` `-` |
| 4 | `*` `/` `%` |

`değil(x)` mantıksal değilleme. `+` metinlerde birleştirme yapar; bir taraf sayıysa otomatik metne çevrilir:

```tan
yaz("sayı: " + 42)        # sayı: 42
yaz("pi " + metin(3.14))  # pi 3.14
```

---

## Akış Denetimi

### Koşul

```tan
eğer x > 10 ise
    yaz("büyük")
değilse eğer x > 5 ise
    yaz("orta")
değilse
    yaz("küçük")
son
```

### Döngü

```tan
i = 0
iken i < 10
    yaz(i)
    i = i + 1
son
```

### Liste üzerinde dolaşma

```tan
her x [1, 2, 3] içinde
    yaz(x)
son
```

### Döngü denetimi

```tan
iken doğru
    eğer bitti ise
        dur          # döngüden çık
    son
    eğer atla ise
        devam        # sonraki yinelemeye geç
    son
son
```

---

## İşlevler

```tan
işlev topla(a, b)
    döndür a + b
son

yaz(topla(3, 4))    # 7
```

Özyineleme desteklenir:

```tan
işlev faktoriyel(n)
    eğer n <= 1 ise
        döndür 1
    son
    döndür n * faktoriyel(n - 1)
son

yaz(faktoriyel(20))   # 2432902008176640000
```

`döndür` yoksa işlev `0` döndürür.

---

## Veri Yapıları

### Liste

```tan
l = [10, 20, 30]
yaz(l[0])            # 10
yaz(uzunluk(l))      # 3
l = ekle(l, 40)      # yeni liste döndürür
l[0] = 99            # yerinde değiştirme
```

Çok satırlı:

```tan
siparis = [
    [2400, 4],
    [1850, 6]
]
```

### Sözlük

```tan
s = {"ad": "Tan", "sürüm": 1}
yaz(s["ad"])
```

### Metin

Metin indekslenebilir; sonuç tek harflik metindir:

```tan
s = "Tan"
yaz(s[0])            # T
yaz(uzunluk(s))      # 3
yaz(kod(s))          # 84
yaz(karakter(65))    # A
```

---

## Modüller

```tan
içe al "matematik"                 # modül adıyla (önerilen)
içe al "kutuphane/Matematik.tan"   # göreli yol (uyumluluk)
```

### Arama sırası

1. İçe alan dosyanın dizini — `./matematik.tan`
2. İçe alan dosyanın `kutuphane/` alt dizini
3. Proje paket dizini — `./tan_moduller/matematik/`
4. `$TAN_YOL` ortam değişkeni
5. Kullanıcı modülleri — `~/.tan/moduller/matematik/`
6. Standart kütüphane — `tan` binary'sinin yanındaki `kutuphane/`

Döngüsel içe almalar otomatik engellenir; bir modül en fazla bir kez yüklenir.

---

## Yerleşik İşlevler

| İşlev | Açıklama |
|---|---|
| `yaz(x)` | ekrana yazar |
| `uzunluk(x)` | metin, liste veya sözlük uzunluğu |
| `metin(x)` | değeri metne çevirir |
| `sayı(x)` | metni sayıya çevirir |
| `ekle(liste, öge)` | öge eklenmiş **yeni** liste |
| `kod(metin)` | ilk karakterin sayısal kodu |
| `karakter(n)` | koddan tek harflik metin |
| `harfler(metin)` | harflerin listesi |
| `parçala(metin, ayraç)` | metni böler, liste döndürür |
| `oku(yol)` | dosyayı metin olarak okur |
| `yazDosya(yol, içerik)` | dosyaya yazar |
| `çalıştır(...)` | dış komut çalıştırır, çıktısını döndürür |
| `arg(i)`, `argsay()` | program argümanları |

Kütüphanede ayrıca: matematik, istatistik, matris, vektör, küme, tarih, finans, metin işleme, tablo, grafik, renk, birim ve daha fazlası — `kutuphane/` dizinine bak.

---

## Arka Uçlar

Tan aynı kaynağı dört farklı yoldan çalıştırabilir:

| Komut | Yol | Dış araç | Kapsam |
|---|---|---|---|
| `tan program.tan` | yorumlayıcı + bytecode VM | Go çalışma zamanı | **tam dil** |
| `tan derle p.tan çıktı` | Tan → C → gcc | gcc, libc | tam dile yakın |
| `tan asm p.tan çıktı` | Tan → x86-64 asm → as/ld | binutils | yalnız int64 |
| `tan elf p.tan çıktı` | Tan → makine kodu → ELF | **hiçbiri** | int64, ondalık, metin, liste, dosya G/Ç |

### `elf` yolu neyi kendi yapıyor

Kendi assembler'ı (REX/ModRM kodlaması, etiket çözümleme), kendi linker'ı (ELF64 başlığı, program header, segment yerleşimi), kendi yığın ayırıcısı (`brk` syscall), kendi metin ve ondalık çalışma zamanı. `printf` yok — sayı-metin çevrimi elle yazılmış makine kodu, çıktı ham `write` syscall.

### Arka uçlar arası farklar

**Tam sayı bölmesi:** `elf` ve `asm` yolunda `100 / 7` = `14` (tam sayı bölmesi, C ve Go gibi). Yorumlayıcı ve C yolunda `14.285714`. Bu bilinçli bir tercihtir: `elf` statik tipli bir altküme derler, çalışma anında tip değiştiremez.

**Desteklenmeyenler (`elf`):** sözlük, çöp toplayıcı yok (bump allocator geri almaz), iç içe liste tip çıkarımı sınırlı.

**Desteklenmeyenler (`asm`):** metin, liste, ondalık — yalnız int64.

### Hata ayıklama

`elf` çıktısı sembol tablosu içerir; `nm` ve `gdb` işlev adlarını görür:

```bash
tan elf program.tan çıktı
nm çıktı
```

Sembolsüz üretmek için: `TAN_SEMBOLSUZ=1 tan elf ...`

### Optimizasyon

Derleme sırasında sabit katlama, cebirsel sadeleştirme ve ölü kod eleme uygulanır. Raporu görmek için:

```bash
TAN_OPT_RAPOR=1 tan elf program.tan çıktı
```

---

## Paket Yönetimi

```bash
tan paket başlat              # tan.json oluştur
tan paket kur <url|ad>        # modül kur
tan paket kur                 # tan.json'daki bağımlılıkları kur
tan paket listele             # kurulu modüller
tan paket sil <ad>            # modülü kaldır
```

### tan.json

```json
{
  "ad": "benim-modulum",
  "surum": "0.1.0",
  "giris": "ana.tan",
  "aciklama": "Kısa açıklama",
  "lisans": "MIT",
  "bagimliliklar": {
    "github.com/kullanici/tan-json": "^1.0.0"
  }
}
```

### Sürümleme

Semver: **BÜYÜK.KÜÇÜK.YAMA**

- **BÜYÜK** — geriye uyumsuz değişiklik
- **KÜÇÜK** — geriye uyumlu yeni özellik
- **YAMA** — geriye uyumlu hata düzeltmesi

Paketler git deposu olarak dağıtılır; sürümler git etiketiyle eşlenir.

---

## Hızlı Başlangıç

```bash
go build -o tan .              # motoru derle (tek seferlik)
./tan Ornek.tan                # yorumlayıcıyla çalıştır
./tan elf Ornek.tan çıktı      # native binary, sıfır bağımlılık
./çıktı

./TestArkaUc.sh elf          # regresyon testleri
./Olcum.sh                     # hız ölçümü
```
