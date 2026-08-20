# GÖREV: araclar/simgeler.tan yaz (TAN tooling — ilk port)

## Amaç
TAN kaynak dosyasını okuyup içindeki SİMGELERİ (işlev tanımları + üst-seviye
değişkenler) listeleyen, TAN dilinde yazılmış bağımsız bir araç. LSP'nin temeli.

## KISIT (çok önemli)
- SADECE bu dosyayı oluştur: `araclar/simgeler.tan`
- Başka HİÇBİR dosyaya dokunma (TancElf.tan, DerleElf.go vs. DEĞİŞTİRME).
- `içe al` KULLANMA — bağımsız, tek dosya. Gerekli mantığı kendi içinde yaz.

## TAN dili syntax hatırlatma (bu dilde yazıyorsun, Go/Python değil)
```
işlev ad(p1, p2)
    x = 5
    eğer x > 3 ise
        yaz("büyük")
    son
    döndür x
son

her t liste içinde
    yaz(t)
son
```
- Blok sonu: `son`. Koşul: `eğer ... ise ... son`. Döngü: `her X liste içinde ... son`.
- Atama: `x = deger`. Karşılaştırma: `==` `!=` `<` `>` `<=` `>=`.
- Yorum satırı: `#`
- Metin birleştirme yerleşiği: `metinBirlestir(a, b)` (— `+` metinlerde GÜVENİLMEZ,
  daima metinBirlestir kullan).

## Kullanılabilir YERLEŞİK / TancElf fonksiyonları (bunları çağır, yeniden yazma)
- `dosyaOku(yol)` → dosya içeriğini metin döndürür (2B helper, mevcut).
- `uzunluk(x)` → metin/liste uzunluğu.
- `metinAl(kaynak, i)` → i. karakter (metin indekslemede kaynak[i] YERINE bunu kullan).
- `metinBirlestir(a, b)` → iki metni birleştirir.
- `metin(sayi)` → sayıyı metne çevirir.
- `karakter(kod)` → kod numarasından karakter (örn `karakter(1)` = SOH ayracı, `karakter(10)` = yeni satır).
- `yaz(metin)` → stdout'a yazar.
- `arg(1)` → 1. komut satırı argümanı (girdi dosya yolu). `argSay()` → argüman sayısı.
- Liste: `[]` boş liste, `ekle(liste, x)` eleman ekler, `liste[i]` indeks.

## ALGORİTMA (satır-tabanlı tarama — basit tut, tokenizer'a gerek yok)
1. `yol = arg(1)` ile girdi dosya yolunu al. Yoksa kullanım mesajı yaz, çık.
2. `--json` argümanı var mı kontrol et (arg'lar arasında). `json = doğru/yanlış`.
3. `kaynak = dosyaOku(yol)`.
4. Kaynağı SATIR SATIR gez (yeni satır = karakter(10) ile böl veya karakter karakter tara).
5. Her satır için:
   - Satır `işlev ` ile başlıyorsa (baştaki boşlukları atla): işlev adını ve
     parametreleri çıkar. `işlev topla(a, b)` → ad=`topla`, parametreler=`a, b`.
   - Satır üst seviyede (girintisiz) `AD = ...` biçimindeyse: değişken bildirimi,
     ad = `=` işaretinden önceki kelime.
6. Bulunan her simgeyi kaydet: tür ("işlev"/"değişken"), ad, satır numarası.

## ÇIKTI FORMATI

### Düz (varsayılan):
```
işlev topla(a, b)  satır 1
değişken x  satır 4
```

### --json ile (TAM olarak bu biçim, AI-tüketilebilir):
```
{"simgeler":[{"tur":"işlev","ad":"topla","satir":1,"parametreler":"a, b"},{"tur":"değişken","ad":"x","satir":4}]}
```
Tek satır JSON. Metin değerleri çift tırnak içinde. `satir` sayı (tırnaksız).

## TEST (yazdıktan sonra WSL'de doğrulanacak — sen sadece dosyayı yaz)
Örnek girdi programıyla:
```
işlev topla(a, b)
    döndür a + b
son
x = topla(3, 5)
yaz(x)
```
Beklenen düz çıktı: `işlev topla(a, b)  satır 1` ve `değişken x  satır 4`.

## NOT
- `kayıt` DESTEKLEME — TancElf'te kayıt yok, sadece işlev + değişken.
- Basit ve DOĞRU tut. Derlenmesi ve çalışması şart. Süslü olması değil.
