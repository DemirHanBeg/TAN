> ⚠️ **TARİHSEL — Go dönemi kaydı.** Go referans motoru 2026-08-20'de TAMAMEN kaldırıldı; repo artık %100 self-hosted, sıfır Go. Bu belge o dönemin mimarisini/planını anlatır. Güncel mimari için **CURRENT_ARCHITECTURE.md**. Buradaki "go build", ".go dosyası", "yorumlayıcı/VM" atıfları ARTIK GEÇERLİ DEĞİL.

# Sürüm Notları

Tan **semver** kullanır: BÜYÜK.KÜÇÜK.YAMA

- **BÜYÜK** — geriye uyumsuz dil/API değişikliği
- **KÜÇÜK** — geriye uyumlu yeni özellik
- **YAMA** — geriye uyumlu hata düzeltmesi

`1.0.0` öncesinde KÜÇÜK sürümler uyumsuzluk içerebilir.

---

## Yayınlanmamış — Go'suz Bootstrap (üretim zincirinden Go çıkarıldı)

**Hedef:** `Go → TAN compiler → TancElf.tan → gen1 → gen2 → gen3` zincirini
`TAN bootstrap artifact → TancElf.tan → TAN compiler → TancElf.tan → TAN
compiler` haline getirmek — Go'yu üretim zincirinden tamamen çıkarmak.

**Go bağımlılık haritası (üretim zinciri açısından):** Depodaki 17 Go
dosyasının (~11.100 satır) TEK üretim-zinciri rolü, `go build -o tan .` ile
`tan` binary'sini üretip `./tan elf TancElf.tan gen1` komutuyla self-hosted
derleyicinin İLK native ikilisini doğurmaktı. Geri kalan her şey —
`Yorumlayici.go`/`VM.go`/`Yerlesik.go` (yorumlayıcı+VM+yerleşikler),
`Lexer.go`/`Parser.go`/`Optimize.go` (Go'nun KENDİ, TancElf.tan'dan bağımsız
lexer/parser'ı), `Modul.go`/`Paket.go`/`Paketle.go` (paket yöneticisi CLI'ı),
`MainNative.go`/`MainWasm.go` (giriş noktaları) — geliştirici aracı/test
referansıdır, derleyici ÇIKTISINI üretmez. `DerleElf.go` (5893 satır, Go'nun
kendi elf arka ucu) da üretime girmez, sadece `TestArkaUc.sh`/`FarkTesti.sh`
çapraz-kontrolünde referans olarak kalır.

**Bulgu:** Bu TEK rol (gen1'i doğurmak) START'tan beri gereksizmiş —
gerekçesi tarihsel: TancElf.tan'ın feature geçmişi HİÇ Go'suz olarak test
edilmemişti, hep Go'nun DerleElf.go'su (her özelliği zaten baştan beri
destekleyen) her commit'te sıfırdan "gen1" üretiyordu. `KanitGoSuzTarihce.sh`
bunun ZORUNLU olmadığını kanıtlıyor: depoda commit'li duran (önceki bir
turdan kalma) `TancElf` native ikilisi tohum alınıp, git tarihindeki HER
TancElf.tan sürümü sırayla SADECE bir önceki native nesille (Go'ya hiç
dönmeden) derlenerek mevcut HEAD'e kadar yürütülebiliyor:

```
62d0fce -> 086baeb -> 9ffe0b5(*) -> fd06ddc -> d6b4fac(**) -> 9723e7c -> 4aecdf4(=HEAD)
```

Sonuç: HEAD'deki TancElf.tan, üç kez üst üste doğrulanmış sabit noktayla
(`gen_a == gen_b == gen_c`) TAMAMEN Go'suz üretiliyor. Bu depodaki `TancElf`
ikilisi bu turda BU sonuçla güncellendi (önceki hali birkaç commit gerideydi
ve mevcut kaynağı derleyemiyordu — `f_dosyaVarMi` etiket hatası veriyordu).

**(*) 9ffe0b5 — bulunan 10. self-hosting bootstrap sorunu:** Bu commit AYNI
ANDA (a) yeni bir yerleşik (`dosyaVarMi`) için derleyici desteği ekliyor VE
(b) TancElf.tan'ın kendi modül-yolu çözümleme kodu (`iceAlYoluCoz`) o
yerleşiği HEMEN kendi üzerinde kullanıyor. Önceki nesil derleyici bu adı hiç
tanımadığından üretilen `call f_dosyaVarMi` için gövde hiç gömülmüyor —
"BAGLAMA HATASI: etiket bulunamadi: f_dosyaVarMi". Kök desen SURUM.md'nin
0.5.0 bölümündeki 9 hatayla AYNI sınıf (bkz. `adAra`, `cagriSonucTipi`) ama
YENİ bir alt tür: "yeni özellik + aynı commit'te öz-kullanım" bootstrap
sıralama sorunu — Go bu sorunu hep GİZLEDİ çünkü DerleElf.go her zaman
ÖNCEDEN o yerleşiği zaten biliyordu. **Çözüm** (`KanitGoSuzTarihce.sh`
içinde uygulanmış, TancElf.tan'ın KENDİSİ değiştirilmeden): iki adımlı geçici
"shim" — önce öz-kullanım noktaları geçici nötrlenip derlenir (ara ikili artık
yerleşiğin çalışma-zamanı gövdesini içeriyor), sonra GERÇEK/yamasız kaynak bu
ara ikiliyle derlenir. Standart bootstrap tekniği (yeni derleyicilerde de
sık): "önce öğret, sonra kullan" iki commit'e bölünseydi shim'e hiç gerek
kalmazdı — gelecekte yeni bir yerleşik EKLEYİP AYNI COMMIT'TE TancElf.tan'ın
kendi kaynağında kullanan katkılar için bu notu okuyun.

**(**) d6b4fac — bilinen salınım, kök sebep zaten SURUM.md'de var:** Bu
commit'te self-compile sabit nokta TUTMUYOR (109098 <-> 114000 <-> 109101
bayt arası salınıyor, ama HER ikili kendi içinde deterministik — rastgelelik
yok). Kök sebep: bu commit NEXUS'un metin/liste parametreli yeni
fonksiyonlarını (kripto/soket/arena) ekliyor ama parametre tipi çıkarımı
HENÜZ yok (bu ADIM 2 / 4aecdf4'te eklendi) — bazı çağrı bölgeleri eksik/yanlış
derleniyor, üretim nesli nesle göre değişiyor. `KanitGoSuzTarihce.sh` bu
adımda EN BÜYÜK (en tam) üretimi seçip devam ediyor; bir sonraki commit
(9723e7c) bu tohumdan sabit noktaya ulaşıyor, 4aecdf4'te (parametre tipi
çıkarımı) salınım tamamen kapanıyor. Düzeltilmedi (kapsam dışı bırakıldı):
gerçek düzeltme d6b4fac'ı bizzat parametre-tipi-çıkarımlı hale getirmek
olurdu ki bu zaten 4aecdf4'ün yaptığı şey — tarihi "düzeltmek" yerine "bir
sonraki commit zaten düzeltiyor" olgusunu kullandık.

**Yeni artifact'lar:**
- `BootstrapGoSuz.sh` — birincil, ileri dönük üretim zinciri: commit'li
  `TancElf` + güncel `TancElf.tan` -> gen1/gen2/gen3 -> sabit nokta. Go'ya
  hiç dokunmuyor. `Bootstrap.sh` artık bunu birincil adım olarak çağırıyor;
  Go varsa SADECE isteğe bağlı çapraz-kontrol için ek adım olarak kullanıyor.
- `KanitGoSuzTarihce.sh` — yukarıdaki tüm tarihsel zinciri sıfırdan
  yeniden üreten, bağımsız doğrulanabilir kanıt script'i.

**Tek zorunlu kalan bootstrap artifact'ı:** commit'li `TancElf` ikilisi
(native, statik bağlı x86-64 ELF, ~115 KB). Bu depoda deterministik olarak
duruyor; `KanitGoSuzTarihce.sh` onun YERİNE geçebilecek daha eski bir kopyanın
(62d0fce) bile git geçmişinden yeniden türetilebildiğini kanıtlıyor —
yani bu ikili "gizemli/doğrulanamaz" bir kara kutu değil, kendi tarihinden
adım adım yeniden inşa edilebilir bir sonuç.

**TancElf.tan'ın (self-hosted derleyici) DerleElf.go'ya (Go referansı) göre
BİLİNEN eksikleri — Go'yu zincirden çıkarmanın önünde engel DEĞİL (derleyici
kendi kaynağını derlerken bunları kullanmıyor) ama TancElf.tan'ın genel amaçlı
kullanıcı programları için TAM İKAME olmasının önünde engel:**
- **Ondalık (float) sayı literalleri hiç desteklenmiyor** — kasıtlı, açıkça
  belgeli (`TancElf.tan` içinde: "ondalık (kesir) sayı literalleri bu
  derleyicide henüz desteklenmiyor... gerçek IEEE754 desteği eklemek yerine
  (kapsamlı, ayrı bir görev)... TEMİZ, AÇIK hata"). Derleme zamanında net
  hata veriyor, sessiz bozulma yok.
- **`yazDosya`/`yaz_dosya` kullanıcı programı içinden çağrılamıyor**
  ("BAGLAMA HATASI: etiket bulunamadi") — derleyicinin KENDİSİ dosya
  okuma/yazmayı kendi `_start` mekanizmasıyla yapıyor ama bunu genel bir
  kullanıcı yerleşiği olarak dışa açmıyor.
- **`her x liste içinde` döngüsünde liste bir METİN LİTERALİ listesiyse**
  (`["bir","iki","uc"]`), döngü değişkeninin tipi yanlış çıkarılıyor —
  metin yerine ham gösterici (pointer) sayısı basılıyor. ✅ **KAPATILDI
  (Faz 3, 14 Ağu 2026)** — `listeElemanTipiniBul` ile döngü değişkeni
  literaldi metin/tam listelerde doğru etiketleniyor; `hertip.tan` testi
  gen2 = Go ELF referansı birebir. Karışık/ifade elemanlı listelerde hâlâ
  "tam" varsayılıyor (bilinçli sınır).

**Ek (Faz 4, 14 Ağu 2026):** işlev dönüş tipi çıkarımındaki sınır da kısmen
kapandı — `döndür <metin literali>` / `<bilinen-metin çağrısı>` /
`<metin-kanıtlı değişken>` artık `tipSabitNoktasiTara`'da (dönüş+parametre
ortak sabit noktası) izleniyor. Karmaşık dönüş ifadeleri (`döndür a+b`)
hâlâ "tam" sayılır (bkz. TancElf.tan başlığı, Bilinen sınırlar #1).

Bu üçü de derleme-zamanı NET hata ya da yanlış-ama-gözlemlenebilir çıktı
üretiyor (segfault/sessiz bozulma yok) — gelecekteki iş için doğru
önceliklendirilmiş, izole, regresyon testiyle kapatılabilir görevler.

---

## Yayınlanmamış — Rastgele Erişimli (Positional) Dosya G/Ç

NEXUS depolama motorunun tabanı. Şimdiye dek yalnız `oku`/`yaz_dosya`/
`ekle_dosya` vardı — hepsi TÜM dosya veya sadece append. Sayfa-tabanlı
B+Tree/buffer pool bu API ile yazılamıyordu. Eklenen fd-tabanlı yerleşikler:

- `dosyaAcOku(yol)` / `dosyaAcYaz(yol)` / `dosyaAcOkuYaz(yol)` → fd (open)
- `dosyaKonumla(fd, ofset)` → yeni konum (lseek, SEEK_SET)
- `dosyaOkuBlok(fd, uzunluk)` → metin (read)
- `dosyaYazBlok(fd, metin)` → yazılan bayt (write)
- `dosyaSenkron(fd)` → fsync (dayanıklılık/crash-safe)
- `dosyaKapat(fd)` → close
- `dosyaSil(yol)` → unlink (native karşılığı eklendi; yorumlayıcıda vardı)

Ayrıca bellek eşleme + ham word erişimi (buffer pool tabanı):
- `bellekEsle(boyut)` → mmap anonim (native gerçek OS mmap), adres döndürür
- `bellekCoz(adres, boyut)` → munmap
- `hamOku8(adres)` / `hamYaz8(adres, deger)` → 8 baytlık word oku/yaz (LE)

İki modda da doğrulandı (bkz. testler/BellekEslemeTest.tan). Native mmap'te
extended register (r8/r9/r10) encoding ile 6-argümanlı syscall. Ham bellek
native'de gerçek pointer, yorumlayıcıda simüle düz bellek — round-trip aynı.
Bununla NEXUS depolama katmanı artık sayfa-tabanlı buffer pool yazabilir.

**İki modda da doğrulandı** (yorumlayıcı + native ELF, birebir aynı çıktı;
bkz. `testler/RastgeleErisimTest.tan`). Native tarafta open bayrakları makine
kodunda SABİT (dallanma yok) — güvenli codegen. 20/20 elf regresyonu korundu.
Native rutinler libc'siz saf syscall (open/lseek/read/write/fsync/close).

NEXUS taban eksiklerinden #1 (rastgele IO) ve #2 (fsync) kapatıldı.

---

## Yayınlanmamış — "Kalan Sistem Eksikleri" devir promptu: ADIM 1-5 kapandı

Yukarıdaki bölümün "kalan ölümcül eksik" listesi (GC/free, native eşzamanlılık,
parametre tip çıkarımı) bu turda kapatıldı. Sıra: 1→2→3→4→5, her adımdan
sonra `bash TestArkaUc.sh elf` 20/20 + `gen1/gen2/gen3` self-hosting sabit
noktası doğrulandı.

**ADIM 1 — ham bellek tamamlandı:** `hamOku4`/`hamYaz4` (32-bit),
`hamOkuBayt`/`hamYazBayt` (native'de İLK KEZ), `bellekKopyala` (memmove,
ÇAKIŞMA-GÜVENLİ — hem ileri hem geri örtüşme test edildi), `bellekDoldur`
(memset). Bkz. `testler/BellekIslemleriTest.tan`.

**ADIM 2 — TancElf.tan'a (self-hosted derleyici) parametre tipi çıkarımı:**
DerleElf.go'nun (Go referans) `parametreTipleriniOgren`'i zaten vardı; eksik
olan self-hosted taraftı. `tipSabitNoktasiTara` (dönüş tipi; Faz 4 sonrası
parametre çıkarımıyla ORTAK sabit nokta) ile AYNI kısıtlı felsefe — sadece
metin literali/bilinen metin-döndüren çağrının/FAZ 4'te metin-kanıtlı
değişkenin DOĞRUDAN sonucu tanınır. `islevTipKutu` 2 elemandan 4'e
genişletildi (7. parametre ABI sınırına çarpardı). Bkz.
`testler/ParametreTipiTest.tan`.

**ADIM 3 — kısmen (`içe al` zaten kapalıydı; `dene`/`yakala` BACKLOG'a
alındı, NEXUS için hata-kodu deseni tercih edildi):** Bunun yerine ADIM 1
primitiflerinin gerçek NEXUS senaryosunda (4KB sabit-boyut checksum'lı
sayfa yöneticisi) yeterliliği sınandı. Eksik primitif bulundu:
`dosyaOkuBlok`/`dosyaYazBlok` SADECE "metin" ile çalışıyordu — ham
bellek↔dosya köprüsü yoktu. Eklendi: `dosyaOkuBlokHam`/`dosyaYazBlokHam`
(read/write(2) doğrudan ham adrese). Bkz. `testler/SayfaYoneticisiTest.tan`
(checksum, rastgele erişim, bozulma tespiti hepsi doğrulandı).

**ADIM 4 — native eşzamanlılık:** `clone`/`futex`/`lock cmpxchg`/
`lock xadd` ile gerçek iş parçacığı. Fonksiyon-DEĞERİ native ELF'te
YOK olduğundan `içParcaLat(islevAdi, args...)` derleme-zamanı bilinen
işlev adı alır (çalışma-zamanı closure değil). Bkz.
`testler/EszamanlilikTest.tan` — native ELF'te 8/8 tekrarlı çalıştırmada
DETERMİNİSTİK (yarış koşulu yok).

**ADIM 5 — ölçüldü, GC YAPILMADI:** `arenaAyir`/`arenaSerbest` (manuel
malloc/free) 1.000→20.000.000 alloc-free döngüsünde RSS SABİT kaldı
(256 KB). Normal `metin`/`liste` (bump, hiç serbest bırakılmıyor) ise
DOĞRUSAL büyüyor (beklenen, ayrı yol). Karar: NEXUS'un asıl deseni
(manuel yönetilen tampon) için arena zaten yeterli — gerçek GC şimdilik
gerekmiyor. Bkz. `testler/ArenaOlcumTest.tan` / `testler/BumpOlcumTest.tan`.

**Kalan (ADIM 6, en son — NEXUS için zorunlu değil):** ARM64 backend +
DWARF hata ayıklama bilgisi.

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
