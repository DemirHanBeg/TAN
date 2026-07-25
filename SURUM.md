# Sürüm Notları

Tan **semver** kullanır: BÜYÜK.KÜÇÜK.YAMA

- **BÜYÜK** — geriye uyumsuz dil/API değişikliği
- **KÜÇÜK** — geriye uyumlu yeni özellik
- **YAMA** — geriye uyumlu hata düzeltmesi

`1.0.0` öncesinde KÜÇÜK sürümler uyumsuzluk içerebilir.

---

## 0.5.0 — Self-Hosting Tamamlandı

**TancElf.tan artık kanıtlanmış şekilde kendi kendini derliyor.** Sabit nokta:
`gen1 = tan elf TancElf.tan` (Go tohumuyla), `gen2 = gen1 TancElf.tan`,
`gen3 = gen2 TancElf.tan` → `cmp gen2 gen3` SESSİZ (birebir aynı bayt).

### Doğrulama (3 ayak + tuzak kontrolü)

- **Tuzak kontrolü:** gen2 boş/sıfır-kod ikili üretmiyor — gerçek, çalışan
  programlar derliyor (doğrulandı: trivial programdan gerçek çıktı).
- **Ayak (a) — sabit nokta:** `cmp gen2 gen3` sessiz.
- **Ayak (b) — gen2 gerçek derleyici mi:** TancElf'in çoğu özelliğini
  kullanan programları (işlev, özyineleme, liste, metin, ondalık, dosya G/Ç)
  derliyor ve doğru çalıştırıyor.
- **Ayak (c) — anlam korunuyor mu:** gen2 çıktısı, Go'nun KENDİ elf arka ucu
  (`tan elf`) çıktısıyla native-native karşılaştırıldığında (yorumlayıcı
  DEĞİL — adil referans) 28 test dosyasından sadece bilinen kapsam
  boşluklarında (`içe al` modül sistemi, `dene`/`yakala`) ayrışıyor —
  ikisi de o durumlarda AÇIK HATA veriyor, sessiz bozulma yok.

### Self-hosting sürecinde bulunan 9 hata (TancElf.tan'ın KENDİ derleyicisi)

Hepsi AYNI kök desenin örnekleri: TancElf.tan'ın tip çıkarımı yalnız
DEĞİŞKEN ATAMALARINI izler, İŞLEV PARAMETRELERİNİ asla (parametre her zaman
"tam" varsayılır — dokümante temel sınır). Bu yüzden bir metin PARAMETRESİ
üzerinde yapılan `==`, `+` (birleştirme), veya `[i]` (ham indeksleme)
potansiyel olarak yanlış kod üretiyordu — hepsi Go tohumuyla (`tan elf`)
GİZLİ kalıyordu çünkü Go'nun DerleElf.go'su bu sınırlamaya sahip değil,
yalnızca TancElf.tan KENDİ KENDİNİ derlerken ortaya çıkıyordu:

1. `bagla()`/`etiketBul()` — `op == "BAYT"` vb.
2. `bc(op,a1,a2)` — bant kodlamasının temeli, `+` zinciri
3. `cagriSonucTipi(ad)` — kendi `ad ==` karşılaştırması, TÜM `t==`/`d==`
   çağrı noktalarını (pTur/pDeger) etkiliyordu
4. `metneTamSayi(s)` — `s[i]` ham indeks (her SAYI token'ı)
5. `metinVerisiBant(deger)` — `deger[i]` ham indeks (metin literalleri)
6. `"f_" + ad` (işlev etiketi üretimi, TANIM tarafı) — 129 fonksiyonun
   etiketini bozuyordu
7. `yeniEtiket(sayacKutu, onEk)` — TÜM kontrol akışı (eğer/iken/her/işlev)
   etiketlerini etkiliyordu
8. `adAra(adlarKutu, ad)` — EN CİDDİSİ: değişken yuva çözümlemesi hiç
   eşleşmiyordu, her referans YENİ yuva açıyordu (16 yerine 119 değişken)
9. **Sessiz yutma kapatıldı:** `deyimDerle`'nin tanınmayan-deyim dalı artık
   sessizce atlamak yerine `"DERLEME HATASI: bilinmeyen deyim: X"` basıp
   duruyor — gelecekteki "gizli 10. hata" bu kapıdan giremez.

### Go'nun rolü — Arşivlendi (silinmedi)

Go artık TancElf'i **doğurmak için ZORUNLU değil** — self-hosted derleyici
kendi kendini üretebiliyor (`gen1`/`gen2`/`gen3` zinciri). Ama Go referans
olarak KALIYOR: `TestArkaUc.sh`/`FarkTesti.sh` çapraz kontrolü Go'nun
`tan elf` çıktısına dayanıyor (Ken Thompson "Trusting Trust" gerekçesi —
tek bağımsız referans yolu kaybolursa gen'e giren bir hata bir daha
çapraz kontrol edilemez).

- `arsiv/DerleC.go`, `arsiv/DerleAsm.go` — Kademe 1 (Tan→C→gcc) ve Kademe 2
  (Tan→asm→as/ld) arka uçları arşivlendi (dış araç bağımlıydılar, `tan elf`
  sıfır-dış-araç olduğundan gereksiz kaldılar). `tan derle`/`tan asm`
  komutları artık arşiv konumunu gösteren bir mesajla çıkıyor.
- `DerleElf.go` (yorumlayıcı + VM + elf arka ucu) **KALDI** — `tan elf`
  hâlâ build zincirinde, referans/çapraz-kontrol rolünde.

---

## Yayınlanmamış (self-hosting çalışması sırasında bulunan elf motor hataları)

TancElf.tan (Tan'da yazılmış, doğrudan ELF üreten derleyici) geliştirilirken
elf arka ucunda (`DerleElf.go`) şu ÖNCEDEN BİLİNMEYEN hatalar bulundu ve
düzeltildi (20/20 regresyon testi her düzeltmeden sonra doğrulandı):

- **`liste[i] = değer` (indeks ataması) elf arka ucunda hiç yazılmamıştı.**
  DOKUMANTASYON.md'de "yerinde değiştirme" olarak belgeliydi ama kod
  üretiminde `IndeksAtamaDugum` için hiçbir dal yoktu — "bu deyim
  desteklenmiyor" ile çöküyordu. `yardimciMetinEsit` gibi bir kod yolu asla
  test edilmemişti çünkü 20 testin hiçbiri liste indeksine atama yapmıyordu.
- **`metin == metin` ham gösterici (pointer) karşılaştırıyordu**, içerik
  değil — iki ayrı tahsis edilmiş ama aynı içerikli metin her zaman "farklı"
  çıkıyordu. Regresyon testlerinin hiçbiri metin eşitliği kontrol etmiyordu.
  `f_metin_esit` (bayt bazlı karşılaştırma) eklendi, `==`/`!=` koduna bağlandı.
- **Tip çıkarım döngüsü sabit 3 tur dönüyordu** — bu sayı keyfiydi, hiçbir
  yere yazılı değildi. Çağrı grafiğinde her tur bilgiyi yalnız BİR katman
  ileri taşıyor; 3'ten derin özyineli-iniş zincirlerinde (ör. çok katmanlı
  bir ayrıştırıcı) parametre tipleri sessizce `tam`a düşüyordu. Artık gerçek
  sabit noktaya kadar (üst sınır 50 tur) dönüyor.
- **Üst seviye (ana gövde) değişken tipleri döngü başlamadan yalnız BİR KEZ**
  taranıyordu — `x = kullaniciIslevi(...)` gibi atamalarda, o işlevin dönüş
  tipi henüz öğrenilmemişken tip sessizce `tam`a düşüyordu. Artık her turda
  yenileniyor.
- `arg()`/`argsay()` yorumlayıcıda (`Yerlesik.go`) hiç yoktu (yalnız C arka
  ucunda vardı) — `tanc2.tan` bile bu yüzden yorumlayıcıyla çalıştırılamıyordu.
  Eklendi.
- **Yorumlayıcıda `ekle(liste, öge)` YERİNDE mutasyon yapıyordu** —
  DOKUMANTASYON.md "yeni liste döndürür (yerinde değil)" der ama
  `Yerlesik.go`'daki uygulama `liste.Elemanlar = append(...)` ile aynı
  `*TanListe` göstericisini değiştirip döndürüyordu. `b = [a]` gibi bir
  sarmalamadan sonra `ekle(b[0], x)` çağrısı `a`'yı da (aynı göstericiyi
  paylaştığı için) sessizce mutasyona uğratıyordu — elf arka ucu (kopyalayan)
  doğru davranıyordu, yorumlayıcı yanlıştı. TancElf.tan'ın kendi kendini
  derlemesi sırasında (`islevDerle`'de `yerelAdlarKutu = [parametreler]`
  deseni) bulundu. Artık her zaman yeni `*TanListe` döndürüyor.
- **`DerleElf.go`'nun yığın ayırıcısında (`f_tan_ayir`) SINIR KONTROLÜ yoktu.**
  `_start`'ta `brk` ile 64 MB ayrılıyordu ama bump allocator bu sınırı hiç
  denetlemiyordu — 64 MB'ı aşan her program haritalanmamış belleğe yazıp
  SESSİZCE segfault veriyordu. Ölçek bağımlı olduğu için "şu iki işlev + şu
  desen birlikte kullanılınca çöküyor" gibi bir KOD ÜRETİM hatası gibi
  göründü; kök sebep basitçe ayrılan hacimdi. `v___yiginSon` sınır değişkeni
  eklendi; sınır aşılırsa `brk` ile 64 MB daha istenir, `brk` başarısız
  olursa sessizce bozulmak yerine `exit(3)`. TancElf.tan'ın kendi çıktı
  programlarındaki aynı desen (`tanAyirBant`/`yiginIlkleBant`) de aynı
  şekilde düzeltildi — önce r12/r13 (register), sonra veri alanı mekanizması
  eklenince (metin desteği için gerekliydi) Go arka ucuyla AYNI yaklaşıma
  taşındı: `v_yigin`/`v_yiginSon` artık RIP-göreli veri yuvaları, register
  değil. Gerekçe: r12/r13 callee-saved — üretilen bir işlev onları
  kaydetmeden kullansa yığın durumu sessizce bozulurdu; veri alanına
  taşımak bu kırılganlığı ortadan kaldırdı.
- **`ekle()` düzeltmesi kütüphanede (`kutuphane/`, 60+ çağrı yeri) ve
  `Talay.tan`/`Noral.tan`/`Tokenizer.tan`'da ESKİ yerinde-mutasyon
  davranışına yaslanıyordu** — `ekle(liste, x)` dönüş değeri atılmış halde
  çağrılıyordu. `ekle()` saf hale gelince (yukarı bkz.) bu çağrılar hiçbir
  şey yapmaz oldu (5 gerçek program: Talay, Noral, Model, TestIstatistik,
  TestTablo çöktü). Tümü `liste = ekle(liste, x)` biçimine çevrildi.
  **Regresyon boşluğu bulundu ve kapatıldı**: `TestArkaUc.sh` (elf sentetik
  testleri) ve `FarkTesti.sh` (`ornekler/`) bu 6 gerçek programı ve
  `testler/` dizinini HİÇ kapsamıyordu — yeni `GercekProgramlar.sh` bunları
  her turda koşup "HATA" çıktısını denetliyor.

- **`ve`/`veya` `TancElf.tan`'da KISA DEVRESİZ ve YANLIŞ derleniyordu.**
  Önceki turda "kapsam dışı" diye bırakılmıştı — YANLIŞ karar: TancElf
  kendi kaynağında (`harfMi`, `rakamMi`, ...) 35 yerde ve/veya kullanıyor,
  kendi kendini derlerken KENDİ SÖZCÜK ÇÖZÜMLEYİCİSİNİ bozuk üretirdi —
  sabit nokta testinin önünde doğrudan engeldi. Düzeltme: RPN (Bölüm 2)
  düz/postfix olduğu için "IKILI ve/veya" işlendiğinde iki operand da ZATEN
  koşulsuz çalıştırılmış oluyordu — kısa devre yapılamazdı. `sayacKutu`
  RPN ayrıştırıcı zincirinin TAMAMINA (`ifadeAyristir`…`temelAyristir`)
  taşındı; `mantiksalAyristir` artık sağ operandın RPN'ini jcc/jmp/etiket
  ile SARIP RPN'İN İÇİNE gömüyor (`kisaDevreBirlestir`) — sağ taraf sol
  tarafın sonucuna göre HİÇ ÇALIŞMAYABİLİR. `ifadeBantYaz`'a ham
  BAYT/REL32/ETIKET pass-through dalları + tip yığını muhasebesi için üç
  işaretleyici (`KISADEVREBASLA/ORTA/BITTI`) eklendi. Bölme-sıfıra-göre
  hatasıyla doğrulandı: kısa devre olmadan çöken desen artık çökmüyor.
  **Bulgu**: Go YORUMLAYICISI zaten doğru kısa devre yapıyordu ama Go
  **elf** arka ucu (`DerleElf.go`) YAPMIYOR — `ornekler/Mantik1.tan`
  Go-elf'te segfault veriyor, yorumlayıcıda vermiyor. Düzeltilmedi
  (kullanıcıya bildirildi, karar bekleniyor) — üçlü çapraz kontrol bu
  özellik için Go-elf yerine Go yorumlayıcısı referans alınarak yapıldı.
- **`deyimDerle` "ifade deyimi" (sonucu atanmayan çıplak çağrı, ör.
  `yazBaytlar(yol, liste)`) HİÇ DESTEKLEMİYORDU** — `AD(...)` bir atamanın
  sağ tarafı değilse SESSİZCE atlanıyordu (yorum: "ifade deyimi henüz
  desteklenmiyor"). Dosya G/Ç eklenirken bulundu — `yazBaytlar()` tam
  olarak bu deseni kullanıyor, dosya hiç yazılmadan program sessizce devam
  ediyordu. `AD` sonrası `PARAC` görülünce konum `AD`'a geri alınıp
  `ifadeSonuc` ile tam ifade derleniyor, sonuç `rax`'a pop edilip
  kullanılmıyor.
- **Dosya G/Ç eklendi**: `oku(yol)`, `yazBaytlar(yol, liste)` — ham
  `open`/`read`/`write`/`close` syscall, libc yok. Yol metnini NULL
  sonlandırmak için gereken boydan 16 bayt fazla ayrılıyor; brk() ile
  büyütülen dokunulmamış bellek çekirdek tarafından sıfırlanmış geldiği
  için (bump ayırıcı asla eski alanı yeniden kullanmıyor) ekstra
  sıfırlama kodu gerekmiyor — Go referansıyla aynı yaklaşım.

Ayrıca (henüz düzeltilmedi, bilinen sınırlar):
- `kod()`/`karakter()`/metin indeksleme elf arka ucunda RUNE değil BAYT
  bazlı çalışıyor — DOKUMANTASYON.md "rune" der ama çok baytlı UTF-8
  (ç ğ ı ö ş ü İ) doğru işlenmiyor. TancElf.tan lexer'ı ve metin çalışma
  zamanı buna göre bayt-bazlı uyarlandı (Go tarafı düzeltilmedi) — çok
  baytlı METİN LİTERALLERİ interpreter/native TancElf arasında farklı bayt
  üretebilir, örnekler bilinçli ASCII.
- **SABİT NOKTA TESTİ ŞU AN TUTMUYOR — `./TancElf TancElf.tan TancElf2`
  WSL VM'ini çökertiyor** (işlem "Exit code 1" ile sessizce ölüyor, ardından
  gelen HİÇBİR komut çalışmıyor — `free -h` sonradan normale dönüyor,
  kalıcı hasar yok ama tek çalıştırmada VM'i devirdiği kesin, tekrarlanabilir).
  Kök sebep KESİNLEŞMEDİ ama en güçlü hipotez: `listeBirlestir`/`ekle`'nin
  O(n²) büyüme deseni (her `kod = listeBirlestir(kod, X)` çağrısı MEVCUT
  boyut kadar kopyalıyor, bump ayırıcı asla serbest bırakmıyor) küçük
  programlarda (birkaç KB çıktı) sorun çıkarmıyordu ama TancElf.tan'ın
  KENDİSİ ~72 KB makine kodu üretiyor — kendi kendini derlerken bu O(n²)
  desen muhtemelen WSL'in ~3.7 GB bellek sınırını (host'ta `.wslconfig`
  yok, varsayılan) katlanarak aşıyor. 64 MB yığın sınır hatasından
  (yukarı bkz.) FARKLI bir sorun — o zaten düzeltildi, bu çok daha büyük
  ölçekte ortaya çıkıyor. Düzeltme GEREKTİRİYOR: kapasite bazlı (amortize)
  liste büyümesi — ki bu, liste bellek düzenini ([uzunluk:8][öge...])
  DEĞİŞTİRMEDEN yapılamaz, DOKUMANTASYON.md'nin "Go arka ucuyla birebir
  aynı düzen" kuralına dokunur. Kullanıcı onayı olmadan denenmedi.

---

## 0.3.0

### Eklendi
- **Modül sistemi**: `içe al "matematik"` modül adıyla çalışıyor. Altı basamaklı
  arama yolu (dosya dizini, kutuphane/, tan_moduller/, $TAN_YOL, ~/.tan/moduller/,
  binary yanı). Döngüsel içe alma korumalı.
- **Paket yöneticisi**: `tan paket başlat | kur | listele | sil`.
  Git depolarından kurulum, `tan.json` manifesti, bağımlılık çözümleme.
- **Optimize edici** (AST seviyesi, üç arka uçta da etkin): sabit katlama,
  cebirsel sadeleştirme (`x+0`, `x*1`, `x*0`, `x/1`), ölü kod eleme.
  Rapor: `TAN_OPT_RAPOR=1`
- **Gözetleme optimizasyonu** (elf): basit sağ işlenenlerde push/pop çifti
  kaldırıldı. Sıcak döngülerde **153 kat** hızlanma.
- **Sembol tablosu** (elf): `.symtab`, `.strtab`, `.shstrtab` bölümleri.
  `nm` ve `gdb` işlev adlarını görüyor. Kapatmak için `TAN_SEMBOLSUZ=1`.
- **Ölçüm takımı**: `Olcum.sh` — arka uçları hız ve boyut olarak karşılaştırır.
- **Dokümantasyon**: `DOKUMANTASYON.md` — tam dil referansı.

### Düzeltildi
- **Ciddi hata**: `int64(0)` doğruluk kontrolünde yanlış yerine doğru sayılıyordu.
  `eğer 0 ise` bloğu çalışıyordu. 0.2.0'daki int64 değişiminden gelen regresyon.
  Hem yorumlayıcıda hem VM'de düzeltildi. Ölçüm takımı yakaladı.
- Optimize edici tam bölünmeyen tam sayı bölmesini artık katlamıyor
  (arka uçlar farklı davrandığı için anlam kayması oluyordu).

### Performans (aynı program, aynı makine)
| Test | Yorumlayıcı/VM | C yolu | elf |
|---|---|---|---|
| 20M yinelemeli döngü | 3119 ms | 769 ms | **55 ms** |
| fib(30) | 973 ms | 32 ms | **8 ms** |

---

## 0.2.0

### Eklendi
- **Yığın ayırıcı** (`brk` syscall, libc yok) — elf arka ucunda metin ve liste.
- **Metin çalışma zamanı**: birleştirme, indeksleme, `uzunluk`, `kod`, `karakter`,
  sayı↔metin çevrimi. Hepsi elle yazılmış makine kodu.
- **Liste**: literal, indeksleme, `ekle`, `her ... içinde`.
- **Ondalık sayı** (SSE): `movq`, `addsd`, `subsd`, `mulsd`, `divsd`,
  `cvtsi2sd`, `cvttsd2si`, `comisd` elle kodlandı. Kendi float→metin çeviricisi.
- **Dosya G/Ç**: `oku`, `yazDosya` — ham `open`/`read`/`write`/`close`.
- **İşlev tip çıkarımı**: dönüş ve parametre tipleri çok geçişli çıkarımla
  belirleniyor; metin/liste/ondalık döndüren işlevler native derleniyor.
- Çok satırlı liste literali, satır sonunda işleçle ifade devamı.
- `metin()` yerleşiği yorumlayıcıya eklendi.

### Değiştirildi
- **Sayı sistemi**: int64 (kesin) ve float64 ayrımı. `123456789 * 987654321`
  artık tam doğru: `121932631112635269`.
- Köprü katmanı boşaltıldı — `Kopru.go` artık hiçbir Go paketi import etmiyor.
- elf/asm arka uçları ondalık sayıda artık sessizce kesmiyor, açık hata veriyor.

---

## 0.1.0

İlk yayın.

- Yorumlayıcı, bytecode VM, WASM hedefi, REPL
- **Kademe 1**: Tan → C → gcc
- **Kademe 2**: Tan → x86-64 assembly → as/ld
- **Kademe 3+4**: kendi assembler'ı ve kendi linker'ı — sıfır dış araç,
  1048 baytlık statik ELF
- Tan ile yazılmış derleyiciler: `Tanc.tan`, `TancAsm.tan`, `Tanc2.tan`
- 31 modüllük kütüphane
- Gerçek programlar: `Kesim.tan` (kesim optimizasyonu), `Noral.tan`
  (geri yayılımlı ağ), `Model.tan` (bigram dil modeli), `Talay.tan`
- MIT lisansı

## 0.4.0 — TancElf (kısmi self-hosting)

### Eklendi
- **`TancElf.tan`** — Tan'da yazılmış, doğrudan ELF üreten derleyici.
  Lexer, RPN ifade ayrıştırıcı, deyim derleyici, makine kodu üreteci,
  ELF64 yazıcı — hepsi Tan'da. Go tohumuyla derlenip native çalışıyor.
  Destek: tam sayı ifadeleri, değişken atama, `yaz`, `eğer/değilse`, `iken`.
- **`FarkTesti.sh`** — TancElf'in iki yolunun (yorumlayıcı / native) aynı
  baytı ürettiğini doğrular. 5/5 birebir.
- **`tamBol(a, b)`** — her arka uçta aynı davranan tam sayı bölmesi.
- **`yazBaytlar(yol, liste)`** — ham bayt yazımı, UTF-8 kodlaması yapmaz.
- **`arg(i)` / `argsay()`** — `elf` arka ucunda program argümanları
  (ham stack'ten, libc yok).
- Metin indeksleme `s[i]` yorumlayıcıya eklendi (ELF'te zaten vardı).
- `DOKUMANTASYON.md`: "Taşınabilirlik Kuralları" bölümü.

### Düzeltildi
- **Arka uçlar arası anlam kayması** — `1000 / 256` yorumlayıcıda `3.90625`,
  `elf`'te `3` idi. Aynı kaynak farklı sonuç verdiği sürece self-hosting
  imkânsızdı. `tamBol()` ile çözüldü.
- **Metin = rune mu bayt mı** — `karakter(200)` yorumlayıcıda 2 bayt (UTF-8),
  `elf`'te 1 bayt üretiyordu. İkili dosya yazımı bozuluyordu. `yazBaytlar()`
  ile çözüldü.
- **Türkçe harfler lexer'da bölünüyordu** — `harfMi` bayt aralığına (128-191)
  göre yazılmıştı; yorumlayıcıda `kod("ğ")` = 287 olduğu için `eğer` sözcüğü
  `e` + `er` diye ikiye ayrılıyordu. Üst sınır kaldırıldı.
- **Yarım yeniden adlandırma** — dosyalar küçük harf, içerik PascalCase
  çağırıyordu. Sebep: büyük/küçük harf duyarsız dosya sistemi (Windows/WSL).
  51 dosya gerçekten taşındı.
- **Dosya adlarında Türkçe büyük harf** — `İstatistik.tan` (U+0130) ASCII'ye
  indirildi. Türkçe büyük harf kuralı tanımlayıcılar için geçerli, dosya
  adları için değil.
- 73 dosyada CRLF → LF. `.gitattributes` eklendi.
