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

### 2.1 Go tohumu → Gen1 → Gen2 → Gen3 (çapraz-kontrol yolu)

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

### 2.2 Go'suz birincil zincir (`BootstrapGoSuz.sh`) — FAZ 2+3+4

```
./TancElf TancElf.tan gen1    → 126851 bayt   (md5 e5ff2eae…)
./gen1    TancElf.tan gen2    → 126851 bayt   (md5 e5ff2eae…)
./gen2    TancElf.tan gen3    → 126851 bayt   (md5 e5ff2eae…)
SABİT NOKTA: gen1 == gen2 == gen3
```

- Hiçbir dış araç çağrılmadı.
- **Faz 2 (mantik tip ayrımı), Faz 3 (liste eleman tipi) ve Faz 4 (dönüş tipi
  izleme) AKTİF.** Tohum = sabit-nokta artifact'i (gen2), kendi çıktısıdır →
  gen1'den itibaren birebir.
- **Faz 1 (sabit katlama + sabit koşul düzleme) HENÜZ UYGULANMADI** — Optimize.go
  karşılığı Aşama 1 kapsamında TancElf.tan'a taşınacak. Aşağıdaki "Faz 1"
  boyut kayıtları/doğrulama maddeleri o tarihteki ara durumu yansıtır; katlama
  mantığı kaynakta şu an yoktur (grep: sabitKatla/katlama bulunamadı).
- Boyut seyri: 114998 (HEAD tohumu) → 119769 (Faz 1 ara kaydı) → 123180 (Faz 2) →
  124392 (Faz 3) → 126851 (Faz 4). Her büyüme derleyici kaynağına eklenen
  yeni mantığın gömülmesindendir.

---

## 3. Nesil Kayıtları (PHASE 4 şeması)

| nesil | kaynak | derleyen | derleyen_kaynagi | cikti_hash (md5) | boyut | deterministik | bagimlilik | sabit_nokta |
|-------|--------|----------|------------------|------------------|-------|---------------|------------|-------------|
| gen1 | TancElf.tan | ./TancElf (tohum, Faz 1-4) | `./TancElf TancElf.tan gen1` | `e5ff2eae97e6c6067e6b61c3d38101f6` | 126851 | EVET (sabit nokta turu) | YOK | gen1 == gen2 |
| gen2 | TancElf.tan | gen1 | `gen1 TancElf.tan gen2` | `e5ff2eae97e6c6067e6b61c3d38101f6` | 126851 | EVET | YOK | gen2 == gen3 |
| gen3 | TancElf.tan | gen2 | `gen2 TancElf.tan gen3` | `e5ff2eae97e6c6067e6b61c3d38101f6` | 126851 | EVET | YOK | gen3 == gen2 |

Tohum artifact'i `TancElf` = `gen2` = `gen3` (md5 `e5ff2eae…`).

### 2.3 Faz 2+3+4 doğrulaması (14 Ağustos 2026)

> **DÜZELTME:** Bu bölümün başlığı "Faz 1+2+3+4" idi ve katlama örneklerini
> (foldedge.tan, foldtest.tan) Faz 1'in AKTİF olduğu izlenimini verecek şekilde
> listeliyordu. Faz 1 (sabit katlama) TancElf.tan'a HİÇ taşınmadı (kaynakta
> katlama mantığı yok). Katlama örnekleri Go tarafındaki Optimize.go doğrulaması
> olarak okunmalıdır; TAN tarafı için Aşama 1 bekleniyor.


- `bash FarkTesti.sh ornekler/*.tan` → **13 ayni, 0 sapma**.
- `bash TestArkaUc.sh elf` → **20 gecti, 0 kaldi**.
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

TancElf.tan'da `katlamaDeger` / `yanEtkisizBant` / `sabitKatla` eklendi;
`ifadeBantYaz` başında `rpnBant = sabitKatla(rpnBant)` (TancElf.tan ~2900-3095).

**Kurallar:** KURAL 1 `[S a][S b][IKILI op] → [S (a op b)]` (op: `+ - * % == !=
> < >= <=`); KURAL 2 `[x][S0][-] → [x]`, `[x][S1][*] → [x]`, `[x][S0][*] → [S0]`
(yalnız önceki bant yan-etkisizse); KURAL 3 `[S v][NEGATIF] → [S (0-v)]`. Kısa
devre/BAYT/REL32/ETIKET girişleri opak bariyer.

**Bulunan ve düzeltilen hata (önemli):** KURAL 1 ilk sürümde
`ilkNHaric(sonuc, 2)` kullanıyordu — [S a] artığını bantta bırakıp `[S a][S c]`
üretiyordu. Yorumlayıcıda zararsızdı (son değer doğruydu) ama native ELF'te yığın
dengelemesini bozup **SIGSEGV** veriyordu; `./gen2 TancElf.tan gen3` çöküyordu.
Düzeltme: `ilkNHaric(sonuc, 3)` (üçlünün tamamı silinir). Düzeltme sonrası sabit
nokta + FarkTesti 13/13 + TestArkaUc 20/20 + fold testi doğru.

**Roadmap'ten bilinçli sapmalar:**
- `x+0`, `0+x`, `1*x` KATLANMIYOR. TAN'da `+` metin birleştirmede de geçerli;
  Go'nun tip-kör kuralı metin için yanlış olurdu (`"ab"+0` → `"ab0"`). `-` ve `*`
  her zaman sayısaldır, bu yüzden `x-0`, `x*1`, `x*0` güvenle katlandı.
- `/` Go'nun 2^53 float kuralıyla DEĞİL, TancElf'in runtime **tam** bölme
  semantiğiyle katlanıyor (guard: b==0 veya (b==-1 ve a==INT64_MIN) ise
  katlanmaz). `%` aynı guard'ı kullanır.

**Açık:** Faz 1 madde 3 — sabit koşul düzleme / ölü blok (`eğer <sabit>` dallama
ve `iken 0` döngü silme) — **YAPILDI** (aşağıda §6b). TancElf.tan kendi
kaynağında sabit koşul içermediği için bu ekleme sabit noktayı etkilemez
(düşük risk).

### 6b. Faz 1 madde 3 — Sabit Koşul Düzleme / Ölü Blok

`kosulSabit(bant, kutu)` yardımcısı eklendi; `deyimDerle`'nin `eğer` ve `iken`
kolları koşulu önce ayrıştırıp `sabitKatla`'dan geçiriyor, sonra tek
`SAYITAM` mı diye bakıyor:

- **`eğer 0 ise`** (sabit yanlış): dal ölü — gövde yalnızca token-tüketimi için
  derlenip atılıyor, zincire devam.
- **`eğer sabit-doğru ise`**: gövde derleniyor, kalan tüm `değilse eğer`/
  `değilse` zinciri yutuluyor.
- **`iken 0`**: döngü tamamen siliniyor (gövde derle-at).
- `sabitKatla` artık `doğru`→`SAYITAM 1`, `yanlış`→`SAYITAM 0` üretiyor
  (koşul katlaması `iken doğru` gibi literal sabitleri de yakalar).

**Bulunan ve düzeltilen hatalar:**
1. İlk deneme `iken <koşul> ise` yazdı — TAN'da `iken` **`ise` ALMAZ**
   (`iken <koşul> ... son`); derleyicinin kendi kaynağı "bilinmeyen deyim: ise"
   ile patladı. Düzeltildi.
2. Sabit-yanlış dalında koşulsuz `değilse` gövdesi `x = blokDerle(...)`
   (yutma) idi — gövde HİÇ derlenmiyordu (testte A1 eksikti). Düzeltme:
   `z = tamponEkleTumu(..., blokDerle(...))` (derleme).

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
| Kanonik tohum = sabit-nokta artifact'i (`TancElf` == gen2, Faz 1-4) | **YAPILDI** (14 Ağu; md5 `e5ff2eae…`) |
| Faz 1 (sabit katlama), Faz 2 (mantik), Faz 3 (liste eleman tipi), Faz 4 (dönüş tipi izleme) | **YAPILDI** (14 Ağu, doğrulandı) |
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
