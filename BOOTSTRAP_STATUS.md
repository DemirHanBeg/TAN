# BOOTSTRAP_STATUS — Güncel Doğrulanmış Durum

> Mimari tanım ve prosedürler için bkz. `BOOTSTRAP_ARCHITECTURE.md`.
> Tüm ölçümler **canlı çalıştırılarak** elde edilmiştir (14 Ağustos 2026).

---

## 1. Özet

- **TAN→TAN üretim zinciri ÇALIŞIYOR ve SABİT NOKTA VERİYOR.**
- `f_harfMi` hatası **mevcut durumda üremiyor**; tarihsel bir self-hosting
  hatasıdır (aile #6) ve kaynakta zaten kapatılmıştır (bkz. §4).
- Güncel GERÇEK engel, commit'li `TancElf` tohumunun **bayat** olmasıydı
  (`f_dosyaVarMi`, belgelenmiş 10. sorun — bkz. §5). Bu, çalışma ağacındaki
  sabit-nokta artifact'inin commit'lenmesiyle kaldırılıyor (§6).

---

## 2. Canlı Doğrulama Kanıtı

### 2.1 Go tohumu → Gen1 → Gen2 → Gen3 (çapraz-kontrol yolu) — TARİHSEL

*Bu bölüm Faz 1-4 öncesi (114998 bayt) Go-çapraz sonucunu belgeler; güncel
durum §2.2'dedir.*

```
go build → ./tan
./tan elf TancElf.tan gen1    → 142944 bayt
gen1 TancElf.tan gen2         → 114998 bayt
gen2 TancElf.tan gen3         → 114998 bayt
cmp -s gen2 gen3              → SESSİZ  (sabit nokta)
```

- Aynı sonuç **pristine HEAD klonu**nda da alındı (yalnızca commit'li
  kaynaklar; çalışma ağacı yamasız).
- `gen1` (Go tohumu) ≠ `gen2` (TAN üretimi) OLMASI **beklenen ve sağlıklıdır**:
  Go referans motoru ile TAN motoru bayt-bayt özdeş üretmek zorunda değildir;
  sabit nokta, TAN üretimi nesiller arasında aranır.

### 2.2 Go'suz birincil zincir (`BootstrapGoSuz.sh`) — FAZ 1+2+3+4

```
./TancElf TancElf.tan gen1    → 152625 bayt   (md5 031c5435…)
./gen1    TancElf.tan gen2    → 152625 bayt   (md5 031c5435…)
./gen2    TancElf.tan gen3    → 152625 bayt   (md5 031c5435…)
SABİT NOKTA: gen1 == gen2 == gen3
```

- Hiçbir dış araç çağrılmadı (`TestArkaUcGoSuzTemiz.sh`: go/gcc/as/ld hiç
  çağrılmadı — GEÇTİ).
- **Faz 1 (sabit katlama + cebirsel + sabit koşul/ölü dal), Faz 2 (mantik tip
  ayrımı), Faz 3 (liste eleman tipi) ve Faz 4 (dönüş tipi izleme) AKTİF.**
  Tohum = sabit-nokta artifact'i (gen2), kendi çıktısıdır → gen1'den itibaren
  birebir.
- Boyut seyri: 114998 (HEAD tohumu) → 119769 (Faz 1 ara kaydı) → 123180
  (Faz 2) → 124392 (Faz 3) → 126851 (Faz 4) → 152625 (Faz 1 katlama + Faz 5
  budama + Faz 6 ondalık sonrası). Faz 1 fold'un eklenmesi derleyici kaynağına
  gömülen fold yardımcılarının boyut büyümesidir (derlenen KOD küçülür).

---

## 3. Nesil Kayıtları (PHASE 4 şeması)

| nesil | kaynak | derleyen | derleyen_kaynagi | cikti_hash (md5) | boyut | deterministik | bagimlilik | sabit_nokta |
|-------|--------|----------|------------------|------------------|-------|---------------|------------|-------------|
| gen1 | TancElf.tan | ./TancElf (tohum, Faz 1-6) | `./TancElf TancElf.tan gen1` | `031c5435be12f28e567e3f50933621d6` | 152625 | EVET (sabit nokta turu) | YOK | gen1 == gen2 |
| gen2 | TancElf.tan | gen1 | `gen1 TancElf.tan gen2` | `031c5435be12f28e567e3f50933621d6` | 152625 | EVET | YOK | gen2 == gen3 |
| gen3 | TancElf.tan | gen2 | `gen2 TancElf.tan gen3` | `031c5435be12f28e567e3f50933621d6` | 152625 | EVET | YOK | gen3 == gen2 |

Tohum artifact'i `TancElf` = `gen2` = `gen3` (md5 `031c5435…`). Yeni tohum
kendi kaynağından yeniden doğrulandı: `TancElf → g1 → g2 → g3` ile
g1 == g2 == g3.

### 2.3 Faz 1+2+3+4 doğrulaması (14 Ağustos 2026)

- `bash FarkTesti.sh ornekler/*.tan` → **13 ayni, 0 sapma**.
- `bash TestArkaUc.sh elf` → **20 gecti, 0 kaldi**.
- `bash GercekProgramlar.sh` → **43 gecti, 0 kaldi**.
- `bash TestOptimize.sh` (Faz 1 optimizer regresyonu, YENİ) → **7/7 gecti**:
  katlama değerleri, 2^53 sınırında/üstü büyük sabitler, cebirsel sadeleştirme,
  sabit koşul/ölü dal, döngü içi fold, 2^53 DERLEME HATASI (exit 1),
  `5 % 0` runtime exit 4.
- **2^53 paritesi (Go referansıyla):** `./tan elf` (güncel Go host, Linux)
  `9007199254740993 / 1` için aynı derleme hatası + exit 1 verir; TAN-native
  (gen2) aynı. **Not:** `tan.exe` (Windows, Jul 26) BAYAT — 2^53 kuralı yok;
  Linux `tan` (Aug 14) güncel.
- Katlama örneği `foldedge.tan` (zincir + karşılaştırma + negatif + cebirsel +
  değişkenli + kısa devre): gen2 ve Go referansı çıktıları **aynı** (tek fark:
  `/` — TancElf'te **tam** bölme, `10/2/2` → `2`; Go referansı float `2.5`.
  Bu fold değil, bilinen TancElf tasarım farkıdır).
- Fold üretimi: foldtest.tan 2933 (tohum, foldsuz) → 2829 bayt (gen2, fold'lu).
- Faz 2 mantik testi `mantiktest.tan` (20 vaka): **gen2 çıktısı Go ELF
  referansıyla BİREBİR aynı** (20/20 satır). Not: Go YORUMLAYICISI mantik
  sonuçlarını "doğru"/"yanlış" basar ve mantik+aritmetikte yapıştırma yapar —
  bu bilinen yorumlayıcı/ELF sapmasıdır; ELF arka ucu (referans) 0/1 sayısal
  semantiğiyle birebir eşleşir.
- Faz 3 her-literal testi `hertip.tan` (5 döngü, 12 satır: metin liste, tam
  liste, metin+`+` birleştirme, tam+`+`, `"["+m+"]"`): **gen2 çıktısı Go ELF
  referansıyla BİREBİR aynı**. Eski tohum aynı dosyada metin listeleri
  gösterici çöpü olarak basıyordu (`4194494`…) — Bilinen Eksik #3'ün tam
  semptomu; Faz 3 ile kapatıldı.
- Faz 4 dönüş-tipi testi `donustip.tan` (5 satır: doğrudan literal, dolaylı
  zincir `paketle→gunlukMetin`, değişken dönüşü `kopyala`, `+`'da metin
  birleştirme): **gen2 çıktısı Go ELF referansıyla BİREBİR aynı**
  (`merhaba`, `merhaba!`, `abc`, `defZ`, `merhabamerhaba`). Eski tohum
  `kopyala` için gösterici çöpü basıyordu (`4194695`, `4194735Z`) — Bilinen
  Eksik #1'in tam semptomu; Faz 4 ile kapatıldı. (İkili md5 farklı — Go
  referansı tam tip çıkarımıyla farklı ama eşdeğer kod üretebilir; davranış
  karşılaştırması esastır.)

---

## 4. f_harfMi — Kök Neden Analizi Raporu

### 4.1 Semptom (raporlanan)

```
BAGLAMA HATASI: etiket bulunamadi: f_harfMi
```

### 4.2 Mekanizma (kaynak iziyle doğrulandı)

- `harfMi` bir **kullanıcı işlevidir**; `f_harfMi` kaynakta hiç geçmez —
  derleme sırasında **üretilen** bir etikettir.
- **Çağrı tarafı** (`ifadeBantYaz CAGRI` dispatch'i): `"f_" + ad` ile doğru
  `REL32 f_harfMi` üretir.
- **Tanım tarafı** (`islevDerle`): ESKİ sürümde `"f_" + ad` (düz `+`) kullanıyor,
  `ad` işlevin KENDİ PARAMETRESİ olduğundan tip çıkarımı onu "tam" sayıyor ve
  etiket adı işlevin adı yerine **göstericinin metne çevrilmiş hali** oluyordu.
- Sonuç: `REL32 f_harfMi` var, `ETIKET f_harfMi` (doğru adla) yok →
  `bagla()` haklı olarak "etiket bulunamadi: f_harfMi" der.
- Bu, SURUM.md'nin 9 self-hosting hatası listesindeki **#6** ("`"f_" + ad`,
  işlev etiketi üretimi, TANIM tarafı — 129 fonksiyonun etiketini bozuyordu").

### 4.3 Kapatılma

- `islevDerle` artık `bcEtiket(metinBirlestir("f_", ad))` kullanıyor
  (9723e7c:3159'da zaten düzeltilmiş halde; sonraki ADIM 2/4aecdf4 parametre
  tipi çıkarımını ekleyip aileyi kökten kapattı).
- Bu yüzden **mevcut TancElf.tan'da tanım tarafı doğru etiketi üretir** ve hata
  üremez.

### 4.4 Ürememe kanıtı

- Çalışma ağacı: gen1→gen2→gen3 sabit nokta, hata yok.
- Pristine HEAD klonu (taze `go build`): aynı, hata yok.
- `BootstrapGoSuz.sh` (Go'suz): sabit nokta, hata yok.

### 4.5 Sonuç

`f_harfMi` raporu, **4aecdf4'ten önceki** (parametre tipi çıkarımı öncesi) bir
duruma aittir. Bugünkü kaynak ve tohumla tekrarlanamaz. **Hiçbir özel durum
yazılmadı, hiçbir hata gizlenmedi; TancElf.tan değiştirilmedi.**

---

## 5. Gerçek Güncel Engel: f_dosyaVarMi (10. sorun)

- Commit'li `TancElf` ikilisi 62d0fce'den kalma (md5 `412778686d8fe8bfe536147737e12a92`,
  115320 bayt) ve güncel `TancElf.tan`'ı derleyemiyor:
  `BAGLAMA HATASI: etiket bulunamadi: f_dosyaVarMi`.
- Kök desen: `dosyaVarMi` yerleşiği eklenirken (9ffe0b5) TancElf.tan'ın KENDİ
  modül-yolu çözümleme kodu (`iceAlYoluCoz`) o yerleşiği AYNI commit'te
  kullanıyor; önceki nesil derleyici `dosyaVarMi`'yi bilmediğinden çağrı
  gövdesi hiç gömülmüyor. SURUM.md'de belgelenmiş 10. bootstrap sorunudur.
- Çözüm stratejisi (`KanitGoSuzTarihce.sh`'de uygulanmış): iki adımlı geçici
  shim. **TancElf.tan değiştirilmez.**
- **Bugünkü çözüm:** çalışma ağacındaki sabit-nokta artifact'i
  (`TancElf` == `gen2`, md5 `32727d59…`) kanonik tohum olarak commit'leniyor
  (§6) — bayat tohum sorunu kaynağında kalkıyor.

---

## 6. Faz 1 — Sabit Katlama: Uygulama Notları (14 Ağustos 2026)

> **DÜZELTME (14 Ağu):** Bu bölümün eski hali `katlamaDeger`/`sabitKatla`/
> `kosulSabit` isimleriyle bir implementasyon anlatıyordu; kaynakta o isimler
> HİÇ YOKTUR (grep ile doğrulandı). Gerçek implementasyon aşağıdadır.

**Uygulanan mimari (bant seviyesinde, Go AST optimizasyonu değil):**
- `ikiliBirlestir(bant, sag, islec)` (TancElf.tan:677) üç ayrıştırıcının
  son noktasıdır (karsilastirmaAyristir 727, toplamaAyristir 739,
  carpmaAyristir 751): sol/sağ bant `tekTamSabit` ile tekil SAYITAM ise
  `sabitSayisalBant` (579) ile katla; katlanamadıysa CEBİRSEL kuralları dene;
  olmadıysa `[sol][sag][IKILI islec]` yay.
- `sabitSayisalBant(aTxt, bTxt, islec)`: `+ - * % == != < <= > >=` için
  metin→int64→metin (`tamMetne`, idiv tabanlı) aritmetik.
  - `*` taşma denetimi Go ile aynı: `c/a != b` ise katlamaz (Optimize.go:181-188).
  - `/` A1 (Go 2^53 kuralı, Optimize.go:195-209): operandlardan biri
    `9007199254740992` mutlak sınırını aşarsa `mutlak53uAsiyorMu` (saf metin
    karşılaştırması) → `yaz("DERLEME HATASI: …2^53…")` + `sistemDur(1)`.
    Aşmıyorsa `a % b == 0` ise `tamBol` ile SAYITAM'a katlanır (TancElf
    `tamBol` semantiği), değilse runtime'a bırakılır (float64 → `3.5`).
  - `% 0` ve `/ 0`: katlanmaz, runtime (`HATA: sıfıra bölme`, exit 4).
- **CEBİRSEL (Go `cebirsel()` ile birebir):** `x+0`, `0+x`, `x-0`, `x*1`,
  `1*x` koşulsuz; `x*0`/`0*x` → `0` yalnız diğer operand `yanEtkisizBant`
  ise (CAGRI engeller). `x/1` YOK (Go kuralı, Optimize.go:295-297).
- **Metin koruması:** `tekTamSabit`/`tamSabitMetin` yalnız `SAYITAM` kabul
  eder; METIN operanda `""` döner — `"ab" + 0 → "ab0"` yanlış katlama olmaz.

**Bulunan ve düzeltilen hata (önemli, self-host tuzağı):** eğer/iken
handler'ları yeniden yapılandırılırken koşul bandı `ifadeSonuc` yerine
`ifadeBantYaz` + `popKayit(0)` ayrı çağrıldı; ilk sürümde `popKayit(0)`
EKLENMEDİ. `ifadeSonuc` (974) üç adımlıdır: `ifadeAyristir` + `ifadeBantYaz`
+ `popKayit(0)` — son adım koşul değerini yığından alıp cmpImm32(0,0) için
yığını dengeler. Eksik kalınca yığın sızdı → **gen2 her girdide segfault**
(`yaz(42)` dahil; sayısal fold'un kapatılmasıyla sürdüğü doğrulandı — sorun
fold'da değildi). Düzeltme: eğer (sabit-değil yolu) ve iken handler'larına
`ifadeBantYaz(kosulBant,…)` sonrası `popKayit(0)` eklendi. Düzeltme sonrası
gen2 == gen3 sabit nokta + tam regresyon yeşil.

**Roadmap'ten bilinçli sapmalar:**
- Go'nun `/` 2^53 kuralı runtime davranışla ÇELİŞİR (Go A1 kararı) — birebir
  taşındı, yalnız SABİT işlenenlerde tetiklenir; runtime değişken `/` için
  sınır yok.
- Kural 3 (`[S v][NEGATIF] → [S (0-v)]`) `ikiliBirlestir`'de DEĞİL,
  `ifadeBantYaz` NEGATIF kolunda yerleşiktir (mevcut davranış, korundu).

### 6b. Faz 1 madde 3 — Sabit Koşul Düzleme / Ölü Blok

`kosulDogruluk(bant)` (1027) + `zincirAtla(tokenler, pozKutu, degilseDursun)`
(1060) yardımcıları eklendi; `deyimDerle`'nin `eğer` (1139) ve `iken` (1202)
kolları koşul bandı üretildikten sonra doğruluğu ölçer:

- `kosulDogruluk`: bant tek girişli SAYITAM/SAYIKESIR/YOK/DOGRU/YANLIS ise
  `metinSifirMi` ile 0/1; karışık → -1 (normal yol).
- **`eğer <sabit-yanlış>`:** gövdeyi DERLE ama at (token tüketimi için),
  `zincirAtla(...,1)` ile `değilse eğer`/`değilse` zincirine devam.
- **`eğer <sabit-doğru>`:** gövdeyi derle, `zincirAtla(...,0)` ile kalan
  `değilse eğer`/`değilse`/`son` yığınını derinlik sayacıyla yut.
- **`iken <sabit-yanlış>`:** deyim tamamen silinir (gövde derle-at).
- **`iken <sabit-doğru>`:** korunur (derleyicinin kendi `iken doğru` döngüleri).
- `zincirAtla` derinlik sayacı: `eğer` açtıkça +1, `son` kapatınca -1 (0'da
  durur); `değilse eğer`'deki "eğer" derinliği ARTIRMAZ (önceki token takibi
  ile); koşulsuz `değilse` yalnız `degilseDursun=1` ise durdurur.

**Bulunan ve düzeltilen hatalar:**
1. İlk deneme `iken <koşul> ise` yazdı — TAN'da `iken` **`ise` ALMAZ**
   (`iken <koşul> ... son`); derleyicinin kendi kaynağı "bilinmeyen deyim: ise"
   ile patladı. Düzeltildi.
2. Sabit-yanlış dalında gövde `blokDerle` ile DERLENMEDEN yutuluyordu —
   içerideki deyimlerden çağrılan işlevler/yardımcılar gömülmüyordu. Düzeltme:
   gövde her durumda `blokDerle` ile derlenir (ölü dalın deyimleri yine de
   token-akışı için işlenir), sabit dal yalnız KOD ÜRETİMİNDE atlanır.
3. `popKayit(0)` yığın dengesi — yukarıdaki kritik hata.

### 6c. Faz 2 — Mantik (bool) Tip Ayrımı

`tipYigin` üçüncü bir etiket kazandı: **"mantik"**. Kaynakları işaretlenen
üreticiler: karşılaştırma (`IKILI` → `> < >= <= == !=`), metin eşitliği,
`ve`/`veya` (`KISADEVREBITTI`), `değil`, `doğru`/`yanlış`/`yok`,
`metinEsit`, `varMı`/`var_mı`. Aritmetik (`+ - * / %`) "tam" üretmeye devam
eder.

**Kritik tasarım kararı — kod üretimi "mantik"e GÖRE DEĞİŞMEZ:** tip yığını
konsumenlerinin TAMAMI yalnızca `== "metin"` kontrolü yapar (grep ile
doğrulandı); `== "tam"`/`!= "tam"` kontrolü HİÇ YOKTUR. Bu yüzden "mantik"
her sayısal bağlamda "tam" ile AYNI kod yoluna düşer:

- "mantik → tam" yükseltme kuralı böylece ÖRTÜK: mantik aritmetikte sayı gibi
  işlenir (`(a>b) + 1` → tam toplama), metin birleştirmede `f_sayi_metne`
  üzerinden geçer (`"x" + (a>b)` → `x1`), `yaz` sayısal çıktı verir, atama
  değişkeni "mantik" olarak kaydeder (davranış aynı).
- İşaretleme değeri: (a) bool==tam karışıklığının kaynağı artık İZLENEBİLİR
  (Go referansındaki 9723e7c tuzağı sınıfı için diyagnostik), (b) Faz 6 ondalık
  ayrımına temel (mantik hiçbir zaman kesir sayılmaz).

**Kanıt:** mantiktest.tan 20 vaka gen2 ve Go ELF referansıyla birebir aynı;
Faz 1 testleriyle birlikte sabit nokta TUTTU (gen1==gen2==gen3). gen1 (ESKİ
tohum kod üretimi) == gen2 (YENİ kod üretimi) olması, mantik işaretlemesinin
kod üretimini birebir değiştirmediğinin doğrudan kanıtıdır.

### 6d. Faz 3 — Liste Eleman Tipi Çıkarımı (Bilinen Eksik #3)

`listeElemanTipiniBul(bant)` eklendi: RPN bandı TEK literal-liste deseni
(`[E1 … En LISTE n]`, E'ler TÜMÜ METIN ya da TÜMÜ SAYITAM) ise eleman tipini
`"metin"`/`"tam"` döndürür; karışık/ifade elemanlı/boş liste → `""`.

`deyimDerle`'nin `her` dalı, `içinde` anahtarından hemen sonra bandı bu
yardımcıdan geçiriyor; sonuç `""` değilse döngü değişkeni
`adTipiAyarla` ile o tiple etiketleniyor. Metin-literal listede değişken
artık `metin` sayılıyor — `+` birleştirme, `yaz`, INDEKS doğru dalı seçiyor.

**Sabit nokta güvenliği:** TancElf.tan'ın kendi 7 `her` döngüsünün HEPSİ
değişken üzerinde (`anahtarlar`, `b`, `digerListe`, `baytlar`) — literal yok.
Bu yüzden derleyicinin kendi üretimi değişmedi; yalnız yeni işlev gömüldü.

**Kanıt:** hertip.tan (5 döngü / 12 satır) gen2 çıktısı Go ELF referansıyla
birebir. Eski tohum metin listelerde gösterici çöpü basıyordu (`4194494`…).

### 6e. Faz 4 — Dönüş Tipi İzleme (Bilinen Eksik #1)

ÖNCEKİ çıkarım ("döndür <metin literal>" | "döndür <bilinen-metin çağrısı>"
| "döndür <AD>(...)" ile islevTipiAra zinciri) DOLAYLI zincirleri zaten
çözüyordu (paketle→gunlukMetin → "metin", sabit nokta sayesinde). ASIL eksik
**değişken üzerinden dönüş** idi: `kopyala(s) { x = s; döndür x }` — eski
tohum x'i "tam" sanıp gösterici çöpü basıyordu.

**Eklenenler:**
- `metinDegiskenMi(tokenler, bas, bitis, islevAd, islevTipKutu, ad)`: yalın
  değişken dönüşünün metin-kanıtı — (a) ad, çağrı-yeri çıkarımıyla "metin"
  bilinen parametre ise; (b) gövdedeki SON `ad = <RHS>` atamasının RHS'i
  metin literali / bilinen-metin çağrısı / metin parametresi ise. (Son-atama
  kazanır: adTipiAyarla semantiğiyle birebir.) Diğer her şey "tam" — false
  positive ASLA; `x = y = s` zinciri bilinçli sınır (izlenmez).
- `govdeDonusTipiCikar`'ın yalın-AD dalı artık bu yardımcıya bakıyor.
- `parametreTipiKaydet` değişiklik bildirir (1) hale geldi; `parametreTipleriniTara`
  değişiklik döndürüyor.
- `islevDonusTipleriTara` + `parametreTipleriniTara` ikilisi TEK ortak sabit
  nokta `tipSabitNoktasiTara`'da birleşti (sürücü 3863): her tur önce dönüş
  tipleri, sonra parametre tipleri; biri değişirse tur tekrar. Parametre
  tipleri artık dönüş tiplerini de besleyebildiğinden iki yönlü yakınsama
  garanti (tipler yalnız "tam"→"metin" yönünde hareket eder, sonlu).
  Eski düzen (önce dönüş sabit noktası, sonra TEK parametre geçişi) bu
  geri-beslemeyi kaçırıyordu.

**Sabit nokta güvenliği (analiz + kanıt):** TancElf.tan'ın kendi
`döndür <değişken>`'leri ya liste (`listeBirlestir`, `tamponSonuc`,
`ilkNHaric`, bant üreticileri — son atama ekle/listeYap/[] → "tam") ya da
zaten hardcoded `yerlesikMetinDonerMi` listesindeki `alan` ("metin",
kayıtlı tip değişimi İNERT çünkü cagriSonucTipi/argumanMetinMi hardcoded'u
ÖNCE kontrol ediyor). Uygulamada sabit nokta tuttu: gen1==gen2==gen3
(126851 B).

**Kanıt:** donustip.tan (5 satır) gen2 = Go ELF referansı birebir; eski tohum
`kopyala` için gösterici çöpü basıyordu (`4194695`…).

---

## 7. Kararlar ve Eylemler

| Karar | Durum |
|-------|-------|
| Kanonik tohum = sabit-nokta artifact'i (`TancElf` == gen2, Faz 1-6) | **YAPILDI** (14 Ağu; md5 `031c5435…`, 152625 B) |
| Faz 1 (sabit katlama + cebirsel + sabit koşul/ölü dal), Faz 2 (mantik), Faz 3 (liste eleman tipi), Faz 4 (dönüş tipi izleme) | **YAPILDI** (14 Ağu, doğrulandı) |
| Faz 5 (ölü işlev/yardımcı budaması), Faz 6 (ondalık + SSE) | **YAPILDI** (önceki aşamalar, roadmap §5/§6) |
| `TestOptimize.sh` (Faz 1 optimizer regresyonu) | **YENİ** (14 Ağu; 7/7) |
| `BootstrapGoSuz.sh`, `KanitGoSuzTarihce.sh` üretim zinciri script'leri | git'e alınacak (untracked) |
| `BOOTSTRAP_ARCHITECTURE.md`, `BOOTSTRAP_STATUS.md` | yeni (bu tur) |

---

## 8. Zincir Sertleştirme Sonuçları (fold ÖNCESİ, tarihsel — 14 Ağustos 2026)

*Bu bölüm fold eklenmeden önceki (114998 bayt) sabit noktayı belgeler; güncel
durum §2-§6'daki fold'lu (119769 bayt) zincirdir.*

Yalnızca `TancElf` + `TancElf.tan` boş bir dizine kopyalandı, PATH'te `go`
olmadığı doğrulandı; zincir iki AYRI kopyada (A ve B) koşuldu.

```
go:   YOK     (üretim zincirinde çağrılmıyor)
gcc/as/ld: makinede VAR ama script hiçbirini çağırmıyor
çıktı: statically linked, "not a dynamic executable" (libc gerekmez)
```

| Ölçüm | Sonuç |
|-------|-------|
| Nesiller | gen1..gen5 boyut = 114998 bayt, md5 `32727d59…` **birebir aynı** |
| Sabit nokta | gen2==gen3, gen3==gen4, gen4==gen5 → **EVET (byte-for-byte)** |
| Determinizm | A vs B, gen1..gen5 → **birebir aynı (DETERMINISTIK)** |
| Tohum uyumu | seed `TancElf` == gen2 → **EVET** (tohum sabit-nokta artifact'i) |
| Bağımlılık | **YOK** (yalnız TAN ikilisi + TAN kaynağı) |

Dikkat: zincir **gen1'den itibaren** sabit noktada — tohum zaten sabit-nokta
çıktısı olduğundan ilk üretim nesli kendisiyle birebir aynı çıkıyor. Bu,
"çalışıyor" değil **kanıtlanmış sabit nokta**dır (§5 kriteri).

### Açık İşler

- [ ] `KanitGoSuzTarihce.sh`'nin güncel HEAD ile tam koşusu (tarihsel kanıt —
      bağımsız, isteğe bağlı).
- [ ] Go'nun üretim prosedüründen çıkarılmasının formal doğrulaması: üretim
      adımları `BootstrapGoSuz.sh` (Go çağırmaz) + yukarıdaki temiz-ortam testi
      ile kanıtlandı; üretim belgelerinde Go yalnızca tarihsel tohum/referans
      olarak geçiyor.
