> ⚠️ **TARİHSEL — Go dönemi kaydı.** Go referans motoru 2026-08-20'de TAMAMEN kaldırıldı; repo artık %100 self-hosted, sıfır Go. Bu belge o dönemin mimarisini/planını anlatır. Güncel mimari için **CURRENT_ARCHITECTURE.md**. Buradaki "go build", ".go dosyası", "yorumlayıcı/VM" atıfları ARTIK GEÇERLİ DEĞİL.

# TAN Bootstrap Yol Haritası — Go → TAN Gen1 → Gen2 → Gen3 → TAN Derleyici

*Hedef zincir: `Go → TAN Gen1 → TAN Gen2 → TAN Gen3 → (mümkünse) TAN → TAN compiler`. Bu belge zincirin bugünkü durumunu, taşıma fazlarını, her fazın kapsamını/doğrulamasını ve bilinen tuzakları verir. Kanıt bulunmayan noktalar "BİLİNMİYOR" olarak işaretlenmiştir. Tüm satır numaraları çalışma ağacındaki dosyalara aittir.*

---

## 1. Zincir Durumu ve Hedef

```
BUGÜN (üretim zinciri, Go'suz):                     HEDEF (uzun vade):
TancElf (commit'li seed, 152.625 B)                  TAN bootstrap artifact
  → TancElf.tan → gen1 (152.625 B)                     → TancElf.tan → TAN compiler
  → gen1  → gen2 (152.625 B)                           → TancElf.tan → TAN compiler
  → gen2  → gen3 (152.625 B)                           → sabit nokta (sabit)
  → cmp gen2 gen3 ✓
```
- **Go, üretim zincirinden ÇIKMIŞTIR.** Go'nun tarihsel tek rolü: `go build -o tan .` → `./tan elf TancElf.tan gen1` (SURUM.md:20-21, 31-38). Bugün Go yalnız çapraz-kontrol aracıdır (`Bootstrap.sh:17-38`, `FarkTesti.sh`, `TestArkaUc.sh`).
- **Kalan iş:** Go compiler'ının henüz TAN'a taşınmamış bileşenlerini (tam tip çıkarımı, ondalık, sözlük, budama, konumlamalı dosya G/Ç, native eşzamanlılık) bootstrap-güvenli fazlarla taşımak. Optimizer çekirdeği (Faz 1) taşındı. Bileşen envanteri ve TAN karşılıkları **BOOTSTRAP.md** §3-§8'dedir.

---

## 2. Her Fazda Uygulanacak Protokol (değişmez kurallar)

1. **Tuzak kontrolü:** yeni gen boş/sıfır-kod ikili üretmiyor — en az bir gerçek program derleyip çalıştır (SURUM.md:207).
2. **Sabit nokta (2 ayak):** `gen2 == gen3` — `cmp -s` sessiz.
3. **Anlam korunumu (3. ayak):** gen2 çıktısı Go'nun `tan elf` çıktısıyla 28 dosyada karşılaştır; yalnız belgelenmiş kapsam boşluklarında ayrışabilir, ikisi de orada AÇIK hata vermeli (SURUM.md:213).
4. **FarkTesti:** yorumlayıcı-vs-native TancElf BAYT kimliği — `./FarkTesti.sh ornekler/*.tan` → `[AYNI]` (FarkTesti.sh:12-18). Yeni özellik TancElf'in her iki çalışma yoluna da aynı eklenmeli.
5. **Shim tekniği (gerekirse):** "yeni özellik + aynı commit'te öz-kullanım" sınıfında (10. bug sınıfı, 9ffe0b5): öz-kullanım nötrlenmiş ara kaynağı derle, ara ikiliyle gerçek kaynağı derle; TancElf.tan kaynağı asla değişmez (KanitGoSuzTarihce.sh:92-111).
6. **Salınım kuralı:** sabit nokta tutmuyorsa EN BÜYÜK (en tam) üretimi tohum seç, sonraki faz o tohumdan ilerler (d6b4fac deseni, KanitGoSuzTarihce.sh:123-135).
7. Her fazın sonunda `cp gen2 TancElf` ile yeni tohum commit'lenir (BootstrapGoSuz.sh:69-71) — commit'li TancElf, mevcut TancElf.tan'ı derleyebilir durumda KALMALIDIR.

---

## 3. Faz 0 — Zemin Doğrulaması (yapılmadan ilerleme)

**Amaç:** mevcut zincirin çalıştığını ve boyut tabanını kanıtlamak; bilinen eksikleri teyit etmek.

**İşler:**
1. `./BootstrapGoSuz.sh` çalıştır → `SABİT NOKTA: gen2 == gen3` beklenir.
2. `./KanitGoSuzTarihce.sh` çalıştır → 7/7 aşama (62d0fce→4aecdf4) beklenir.
3. `./TestArkaUcGoSuzTemiz.sh` çalıştır (go/gcc/as/ld tuzağı altında üretim zinciri).
4. `./FarkTesti.sh ornekler/*.tan` çalıştır → 13/13 `[AYNI]` beklenir.
5. Boyut tabanı kaydet: gen1=142.944 B, gen2=gen3=TancElf=114.998 B.
6. **Doğrula (BİLİNMİYOR olarak işaretlenmiş noktalar):**
   - Çalışma ağacı `TancElf` 114.998 B iken commit'li blob 115.320 B (`c808a012`) — `M TancElf` commit'lenmemiş durumda; Faz 0'da karar ver (kaynağa uygun yeni tohum commit'i).
   - `Bootstrap.sh:26` `AsmTest.tan` yazarken diskte `Asmtest.tan` var — strict Linux mount'ta kırılır; küçük harfe düzelt.
   - `TancElf.tan`'ın `elfUret`'inin sembol tablosu üretip üretmediği: kanıt bulunamadı (BİLİNMİYOR). Go yolu `TAN_SEMBOLSUZ` yoksa symtab yazar (DerleElf.go:5824-5826); TancElf karşılığı varsa belgele.

**Çıktı:** dolu bir `git status` temiz değilse, doğrulanmış zemin raporu.

---

## 4. Faz 1 — Optimizer Çekirdeği — ✅ YAPILDI (14 Ağu 2026)

**Seçim gerekçesi:** BOOTSTRAP.md §9. Go Optimize.go'nun iki ana işlevi `sabitKatla` (164-260) + `cebirsel` (263-300) + koşul düzleme/ölü blok (deyim 49-112, blokDugumu 115-121). TAN karşılığı AST üzerinde DEĞİL, **RPN bant seviyesinde** yazıldı — TancElf.tan AST kullanmaz (TancElf.tan:229-231).

**Uygulandı:**
1. **Bant içi sabit katlama:** `ikiliBirlestir` yardımcısı karsilastirmaAyristir/toplamaAyristir/carpmaAyristir sonlarına bağlandı. Bantta ardışık iki `SAYITAM` + `IKILI` örüntüsü derleme-anı int64 aritmetiğiyle (`+ - * % == != < <= > >=`) tek `SAYITAM`'a indirilir. `*` taşma denetimi Go ile aynı (`c/a != b` ise katlama, Optimize.go:181-188); `/` 2^53 (9007199254740992) aşımı → `yaz(DERLEME HATASI…)`+`sistemDur(1)` (Go A1, Optimize.go:195-209); `% 0` ve bölünemeyen `/` runtime'a bırakılır; bölünebilen `/` `tamBol` ile SAYITAM'a katlanır.
2. **Cebirsel sadeleştirme:** `x+0`, `0+x`, `x-0`, `x*1`, `1*x` koşulsuz; `x*0`/`0*x` → 0 yalnız sağ/sol operand yan-etkisizse (CAGRI engeller). `x/1` YOK (Go kuralı, Optimize.go:295-297).
3. **Sabit koşul + ölü dal:** `kosulDogruluk` (SAYITAM/SAYIKESIR/YOK/DOGRU/YANLIS → sabit doğruluk; karışık → -1) + `zincirAtla` (derinlik sayacı; `değilse eğer`'deki "eğer" derinliği artırmaz; `değilse`/`son` durma kuralları). eğer sabit-doğru → gövde + zinciri `değilse eğer`/`değilse` üzerinden atla; eğer sabit-yanlış → gövdeyi at, zincire devam; iken sabit-yanlış → deyim silinir. `iken doğru` (kaynakta mevcut) korunur.
4. **Kritik düzeltme (self-host tuzağı):** yeniden yapılandırılan eğer/iken handler'ları koşul bandını `ifadeBantYaz` ile üretirken `popKayit(0)` (ifadeSonuc'un 974'teki 3. adımı) EKLENMEDİĞİNDE yığın sızdı → gen2 her girdide segfault (`yaz(42)` dahil). Düzeltme: koşul üretiminden sonra `popKayit(0)` şart.

**Doğrulama:**
- `TestOptimize.sh` (YENİ): 7 bölüm — katlama değerleri, 2^53 sınırında/üstü büyük sabitler, cebirsel, ölü dal, döngü içi fold+dal, 2^53 DERLEME HATASI (exit 1), `5 % 0` runtime exit 4 — tümü [GECTI].
- Go referansıyla 2^53 paritesi: `./tan elf` (güncel Go host) aynı hata + çıkış; eski `tan.exe` (Windows, Jul 26) bayat — kural YOK.
- FarkTesti 13/13 `[AYNI]`, TestArkaUc.sh elf 20/20, GercekProgramlar 43/43, TestArkaUcGoSuzTemiz GEÇTİ (go/gcc/as/ld çağrılmadı).
- Sabit nokta: gen1=gen2=gen3=152.625 B; yeni tohum md5 `031c5435…`, kendi kaynağından yeniden sabit nokta g1==g2==g3. `cp gen2 TancElf` yapıldı.

**Risk notu:** `/` 2^53 kuralı runtime davranışla bilinçli çelişir (Go A1 kararı) — birebir taşındı; yalnız SABİT işlenenlerde tetiklenir.

---

## 5. Faz 2 — `mantık` (bool) Tip Ayrımı — ✅ YAPILDI (14 Ağu 2026)

**Amaç:** TancElf.tan'ın `tipYigin`'ini yalnız `tam`/`metin`/`liste` değil `mantik` içerecek şekilde genişletmek; karşılaştırma ve `ve`/`veya`/`değil` sonuçlarını `tam` yerine `mantik` etiketlemek. Go referansı: `Cesit` tipi DerleElf.go:699-781 ve `tipCikar` (783). Bu, `9723e7c`'de düzeltilen `bool==tam sayı` tuzağı sınıfının (git: "kritik bool==tam sayı karşılaştırma tuzağı") TAN tarafında yeniden doğmamasını sağlar.

**Uygulandı:**
- `tipYigin` eleman setine `"mantik"`; karşılaştırma (`IKILI` catch-all), metin eşitliği, `KISADEVREBITTI` (ve/veya), `DEGIL`, `DOGRU`/`YANLIS`/`YOK`, CAGRI `metinEsit` ve `varMı`/`var_mı` sonuçları `mantik` işaretlendi; aritmetik `+ - * / %` `tam` kaldı.
- Koşul bağlamları zaten ≠0 testi kullanıyor — değişiklik gerekmedi.
- **Yükseltme kuralı örtük:** kod üretimi yalnız `== "metin"` kontrolü yaptığından (grep ile doğrulanmış, `== "tam"` kontrolü hiç yok) mantik her sayısal bağlamda tam ile aynı yola düşer. Bu yüzden sabit nokta bozulmadı: **gen1 (eski tohum kod üretimi) == gen2 (yeni kod üretimi) == gen3** — mantik işaretlemesinin kod üretimini BİREBİR değiştirmediğinin doğrudan kanıtı.
- Kapsam notu: TancElf.tan'ın kendi 97 `ve` + 61 `veya` + 26 `değil` kullanımı üretim anında davranışı değiştirmedi (işaretleme yalnız etiket) — başlangıçta öngörülen iki-adaylı shim (protokol #5) GEREKMEDİ.

**Doğrulama:** FarkTesti 13/13 `[AYNI]`, TestArkaUc.sh elf 20/20 (`mantik` + `karsilastirma` dahil), sabit nokta gen1==gen2==gen3 (123.180 B, md5 `6528b973…`), `mantiktest.tan` 20 vaka gen2 = Go ELF referansı birebir.

---

## 6. Faz 3 — Liste Eleman Tipi Çıkarımı (Bilinen Eksik #3) — ✅ YAPILDI (14 Ağu 2026)

**Amaç:** `her x ["bir","iki","uc"] içinde` döngüsünde `x`'in metin olarak çıkarılması (SURUM.md:109-111). Şu an metin-literal listesi elemanı ham gösterici sayısı basılıyor.

**Uygulandı:**
- `listeElemanTipiniBul(bant)` (ifadeBantYaz öncesi yardımcı): RPN bandı `[E1 … En LISTE n]` ve tüm E'ler aynı literal tip (`METIN` veya `SAYITAM`) ise `"metin"`/`"tam"` döner; karışık/ifade/boş liste → `""`.
- `deyimDerle` `her` dalı `içinde` yutulduktan hemen sonra eleman tipini döngü değişkenine `adTipiAyarla` ile yansıtır.
- Literal-durumunda TEK geçiş — sabit nokta gerekmez (yeni bağımlılık yok).

**Doğrulama:** `hertip.tan` 5 döngü / 12 satır gen2 = Go ELF referansı birebir; eski tohum gösterici çöpü basıyordu (`4194494`…). FarkTesti 13/13, TestArkaUc 20/20, sabit nokta 124.392 B (md5 `bdd9b064…`). TancElf.tan'ın kendi 7 `her` kullanımı hep değişken üzerinden — sabit nokta etkilenmedi.

---

## 7. Faz 4 — Dönüş Tipi İzleme İyileştirmesi (Bilinen Eksik #1) — ✅ YAPILDI (14 Ağu 2026)

**Amaç:** "işlevden metin döndürüp değişkene atayıp `+` ile kullanma" yanlış kod üretimini kapatmak (TancElf.tan:19-22).

**Durum tespiti:** DOLAYLI zincirler (`döndür baskaIslev()`) aslında ÇOKTAN çalışıyordu — `govdeDonusTipiCikar`'ın islevTipiAra sabit noktası onları çözüyordu (test: `paketle→gunlukMetin` doğruydu). Gerçek eksik **değişken dönüşü** idi: `kopyala(s) { x = s; döndür x }` gösterici çöpü basıyordu.

**Uygulandı:**
- `metinDegiskenMi(tokenler, bas, bitis, islevAd, islevTipKutu, ad)`: yalın değişken dönüşü için metin-kanıtı — (a) metin parametresi (çağrı-yeri çıkarımı), (b) gövdedeki SON `ad = <RHS>` atamasının RHS'i metin literali / bilinen-metin çağrısı / metin parametresi. Son-atama kazanır (adTipiAyarla semantiği); false positive ASLA; `x = y = s` zinciri bilinçli sınır.
- `govdeDonusTipiCikar`'ın yalın-AD dalı bu yardımcıya bağlandı.
- `parametreTipiKaydet` değişiklik bildirir (1); `parametreTipleriniTara` değişiklik döndürür; dönüş+parametre çıkarımı TEK ortak sabit noktaya (`tipSabitNoktasiTara`, sürücü 3863) birleşti — parametre tipleri artık dönüş tiplerini de besleyebiliyor (eski sıra: önce dönüş sabit noktası, sonra TEK parametre geçişi; geri-besleme kaçıyordu).

**Doğrulama:** `donustip.tan` 5 satır gen2 = Go ELF referansı birebir; eski tohum `kopyala` için gösterici çöpü basıyordu (`4194695`…). FarkTesti 13/13, TestArkaUc 20/20, sabit nokta 126.851 B (md5 `e5ff2eae…`). Kendi kaynağındaki değişken dönüşleri liste (`ekle`/`listeYap`/`[]` son-atama → "tam") ya da hardcoded `alan` (inert) olduğundan sabit nokta bozulmadı.

---

## 8. Faz 5 — Ölü İşlev / Yardımcı Budaması — ✅ YAPILDI (14 Ağu 2026)

**Amaç:** Go `elfUlasilabilirIslevler` (DerleElf.go:5410) karşılığı. Şu an TancElf.tan 147 işlevin TÜMÜNÜ + 17 runtime yardımcısının TÜMÜNÜ her programın içine gömer (sürücü 3526-3545) — `d6b4fac` boyutları da buradan büyük.

**Uygulandı:**
- **A — ölü işlev budaması:** `ulasilabilirIslevleriTara` üst-seviye deyimlerden çağrılan işlevleri BFS ile kapatır (işlevCagirdiklariniTopla); `deyimDerle`'nin `işlev` dalı yalnız kapanıştaki işlevlerin gövdesini derler (`islevTipKutu[4]` boşsa hepsi — eski davranış).
  - Tuzaklar: (1) heterojen `döndür [adlar, basListe, bitisListe]` TİP ZEHİRLENMESİ (ilk eleman metin listesi olduğundan üçü de `TipListe(TipMetin)` çıkıyordu) → **Kutu deseni** (`haritaKutu = [[],[],[]]` + `döndür 0`); (2) `alan(liste, i)` liste elemanı erişimi değil TOKEN string bölme — liste için `liste[i]`; (3) `işlev ad (` tanım başlıkları çağrı sanılıyordu → önceki jeton `işlev` ise atla (cagriMi).
- **B — runtime yardımcı budaması:** `yardimciKullanimiTara(govde, yardimciAdlar)` govde bandındaki REL32 hedeflerini 17'lik whitelist'e karşı tarar; `yardimciKapanisi` bağımlılık kapanışını ekler (`yardimciBagimliliklari`: `f_liste_ekle`/`f_metin_birlestir`/`f_oku`/`f_dosyaVarMi`/`f_yazBaytlar`/`f_arg` → `f_tan_ayir`+`f_bellek_kopyala`; `f_liste_yap`/`f_sayi_metne`/`f_metin_indeks`/`f_karakter` → `f_tan_ayir`; kalanlar yaprak). Sürücü yalnız o kümenin *Bant()'ını gömüyor. Doğruluk garantisi: govde'de yardımcılara GİDEN tek referans yolu `callBant("f_...")` REL32'leri (grep ile tüm hedefler whitelist'te) — kapama tam.

**Doğrulama:** `yaz 42` 2267→395 B; olu_olim 2341→758 B; tiny 2342→759 B; bfs_test 2392→809 B (hepsi aynı çıktı). FarkTesti 13/13 `[AYNI]`, TestArkaUc.sh elf 20/20, sabit nokta 123.775 B (md5 `d401fb0b…`). Go muaf kuralı (kayıt metotları) TAN'da kayıt olmadığından geçerli değil; `TestArkaUcGoSuzTemiz.sh` (go/gcc/as/ld tuzağı altında) da yeşil.


---

## 9. Faz 6 — Ondalık (Float) Literalleri + SSE (en yüksek risk)

**Amaç:** `SAYIKESIR` (2948-2965) açık hatasını kaldırıp gerçek ondalık desteği eklemek. Bu, chicken-egg taşır: **TancElf.tan kaynağına ondalık literal GİREMEZ** (derleyici henüz kendini ondalıkla derleyemez). Bu yüzden:

**Bootstrap stratejisi:**
1. TancElf.tan'a `SAYIKESIR` kod üretimi + `f_kesir_metne`/`f_yaz_kesir` runtime'larını ekle (Go: `yardimciKesirMetne` 5083, `yardimciYazKesir` 5216, SSE encoders 607-693).
2. TancElf.tan kaynağının KENDİSİNDE ondalık literal YOK — sabit nokta bu yeni özellikle yeniden doğrulanır (kaynakta kullanım olmadığı için shim gereksiz).
3. Sonraki nesilde kaynak içi ondalık testleri eklenir; üçüncü nesilde gerçek ondalık kullanan TAN kodu TAN derleyiciyle derlenir.
4. Go `/` A1 kararı (her zaman float64) ile TAN `tamBol` ayrımını koru; `+` metin/birleştirme dispatch'inde ondalık tarafını `f_sayi_metne` değil `f_kesir_metne` ile çevir.

**Doğrulama:** `ondalik` testi (TestArkaUc.sh test 17) TAN eşdeğeri; 3. ayak (c) Go-elf ile 28 dosya; sabit nokta (boyut değişimi beklenir — `movq xmm` talimatları gen1/2'de görülür).

---

## 10. Faz 7 — Sözlük (uzun vade, opsiyonel)

**Amaç:** TancElf.tan'a gerçek `sözlük` tipi. Paralel-liste taklidi (BOOTSTRAP.md §5) çalışıyor ama islevTipKutu/adlarKutu gibi yapılar büyüdükçe taşınması zor.
- Go runtime: `f_sozluk_hash/yap/koy/al/varmi/sil/anahtarlar` (DerleElf.go:3525-3790, 256 kovalı ayrık zincir).
- Önce runtime yardımcılarını taşı (budama Faz 5 ile birleşir), sonra `tipYigin`'e `"sozluk"` ve `sözlük` literal/indeks kod geni.
- **NOT:** Go'nun yorumlayıcısı sözlüğü `map[string]Deger` ile yapıyor (Yorumlayici.go:47); TAN-elf ve TancElf sözlük RUNTIME'ı aynı olmalı (bayt kimliği için FarkTesti).

---

## 11. Yapılmaması Gerekenler (kapsam dışı)

- **Yorumlayıcı + VM'nin TAN'a taşınması** — TancElf doğrudan ELF üretir; yorumlayıcı Go-only araç kategorisindedir (BOOTSTRAP.md §4). Üretim zinciri için gerekli değil.
- **`dene`/`yakala` ve `köprü`** — Go elf'inde de YOK (BACKLOG, SURUM.md:171-173); TancElf hata deseni `yaz`+`sistemDur(1)`'dir (1054-1055).
- **Paket yöneticisi (`tan paket`) ve paketleme** — Go-only geliştirici araçları; bootstrap zinciriyle ilgisiz.
- **`içe al` çalışma-zamanı modül yükleme** — TAN-elf ve TancElf derleme-zamanı statik açılım kullanır (tokenGenislet 3435); aynı kalsın.
- **FFI (`disKutuphane*`)** — YerlesikFFI linux-only Go yerleşiği; TAN tarafı ayrı bir görev.

---

## 12. Faz Takvimi ve Bağımlılıklar

```
Faz 0  [ön koşul] zincir + boyut tabanı doğrulama      ✅ (zemin)
Faz 1  sabit katlama/cebirsel/ölü blok      (bağımsız) ✅ YAPILDI (14 Ağu)
Faz 2  mantık tip ayrımı                    (bağımsız) ✅ YAPILDI (14 Ağu)
Faz 3  liste eleman tipi                    (Faz 2'ye yaslanır) ✅ YAPILDI (14 Ağu)
Faz 4  dönüş tipi izleme                    (Faz 2+3'ün üstüne) ✅ YAPILDI (14 Ağu)
Faz 5  ölü işlev/yardımcı budaması          (bağımsız; Faz 6/7'yi küçültür) ✅ YAPILDI (14 Ağu)
Faz 6  ondalık + SSE                        (Faz 5 sonrası: yardımcı kümesi netleşir) ✅ YAPILDI (14 Ağu)
Faz 7  sözlük                               (Faz 5 sonrası) ⏳ (planlandı)
```

Her faz 1 → 2 → 3 → 4 → 5 → 6 → 7 sırasıyla; fazlar arası `cp gen2 TancElf` tohum commit'i.

---

## 13. Ölçülebilir Hedefler (başarı kriterleri)

| Metrik | Bugün (Faz 1-6 sonrası) | Faz 5 sonrası hedef |
|---|---|---|
| gen2 == gen3 (sabit nokta) | ✅ 152.625 B (md5 `031c5435…`) | ✅ korunur |
| FarkTesti ornekler | 13/13 `[AYNI]` | 13/13 korunur |
| TestArkaUc.sh elf | 20/20 | 20/20 |
| TestOptimize.sh (Faz 1 optimizer) | ✅ 7/7 | — |
| `yaz 42` ikili boyutu | ✅ 395 B (yalnız `f_yaz_metin`+`yaz_sayi`+`f_tan_ayir`+`f_bellek_kopyala`) | min. yardımcı kümesi ✅ |
| TancElf.tan belgeli sınırlar (#1,#2,#3) | 3 açık | 0 açık |

---

## 14. Referanslar
- BOOTSTRAP.md (bileşen envanteri, TAN karşılıkları, eksikler, bağımlılıklar, bileşen seçimi)
- BootstrapGoSuz.sh, KanitGoSuzTarihce.sh, TestArkaUcGoSuzTemiz.sh, FarkTesti.sh, GercekProgramlar.sh
- SURUM.md:13-118 (Go'suz bootstrap), 199-217 (self-hosting doğrulaması), 96-115 (bilinen eksikler)
- TancElf.tan:1-31 (başlık/sınırlar), 3485-3552 (sürücü), Optimize.go:49-300 (taşınacak mantık)
- Git desenleri: 9ffe0b5 (shim), d6b4fac (salınım), 4aecdf4 (parametre tipi çıkarımı)
