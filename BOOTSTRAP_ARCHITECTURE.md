# BOOTSTRAP_ARCHITECTURE — TAN Derleyici Bootstrap Mimarisi

> Bu belge, üretim derleyici zincirinin **hedef mimarisini** ve **prosedürlerini**
> tanımlar. Güncel doğrulanmış durum için bkz. `BOOTSTRAP_STATUS.md`.

---

## 1. Hedef (nihai üretim zinciri)

Amaç **Go'suz, kendi kendini üreten TAN derleyici zinciri**:

```
TAN derleyici kaynağı (TancElf.tan)
        ↓
TAN derleyici yürütülebiliri (tohum/seed)
        ↓
TAN Gen1
        ↓
TAN Gen2
        ↓
TAN Gen3
        ↓
sabit nokta (fixed point / self-hosting)
```

Başarı koşulu: **temiz bir ortam**, TAN derleyici yürütülebiliri + TAN derleyici
kaynaklarıyla bir sonraki TAN derleyici neslini **Go derleyicisini çağırmadan**
üretebilir.

### Gen1 tanımı (kesin)

- Gen1, üretim mimarisinde **"Go'nun ürettiği derleyici" DEĞİLDİR.**
- Gen1 = **tohum TAN derleyicisi tarafından üretilen ilk nesil.**
- Tohumdan sonraki HER nesli önceki TAN nesli derler.

```
Gen1 → Gen2 → Gen3 → Gen4 → … sabit nokta
```

### Go'nun rolü (sınırlı)

Go derleyicisi, üretim zincirinde **zorunlu değildir**. Yalnızca şunlar için
kalabilir:

| Rol | Kullanım |
|-----|----------|
| Tarihsel bootstrap | İlk tohum ikilisini üretmek (tek seferlik) |
| Geliştirme aracı | `tan` Go ikilisi, referans/çapraz-kontrol |
| Acil durum yedeği | Zincir kırılırsa tohumu yeniden üretmek |

Go, `TancElf.tan` kaynağından `TancElf` tohum ikilisini ürettikten SONRA üretim
zincirinden ayrılır. Sonraki tüm adımlar TAN→TAN'dır.

---

## 2. Artifact'lar

| Artifact | Tür | Üretilir | Zincirdeki rolü |
|----------|-----|----------|-----------------|
| `TancElf.tan` | TAN kaynağı | elle bakım (FROZEN — değiştirilemez) | derleyici kaynağı |
| `TancElf` | native x86-64 ELF ikili | Go tohumuyla (tarihsel) → sabit nokta artifact'i | **tohum (seed)** |
| `gen1` | ELF ikili | `TancElf TancElf.tan gen1` | ilk TAN üretimi nesli |
| `gen2` | ELF ikili | `gen1 TancElf.tan gen2` | ikinci nesil |
| `gen3` | ELF ikili | `gen2 TancElf.tan gen3` | üçüncü nesil |
| `genN+1` | ELF ikili | `genN TancElf.tan genN+1` | sonraki nesiller |

Tohum ikilisi, bir sabit nokta turunun ÇIKTISIdır (`gen2`'nin kopyası): yani
zincir kendi ürünüyle tohumlanır. Deterministik olarak git'te durur
(`BOOTSTRAP_STATUS.md` hash tablosu).

---

## 3. Üretim zinciri prosedürü

### 3.1 Birincil (Go'suz) — `BootstrapGoSuz.sh`

```
TancElf TancElf.tan gen1     # tohum → Gen1
gen1    TancElf.tan gen2     # Gen1 → Gen2
gen2    TancElf.tan gen3     # Gen2 → Gen3
cmp -s gen2 gen3             # sabit nokta: SESSİZSE birebir aynı
```

Bu script hiçbir dış aracı (go/gcc/clang/as/ld/libc) çağırmaz. Go yalnızca
`TestArkaUc.sh`/`FarkTesti.sh` gibi **istek dışı çapraz-kontrol** araçlarında
referans olarak kullanılabilir.

### 3.2 Temiz ortam yeniden inşası — `TemizBootstrap` (testler)

Bir makinenin gerçekten Go'suz üretebildiğini kanıtlamak için:

1. Yalnızca `TancElf` + `TancElf.tan`'ı boş bir dizine kopyala.
2. Zinciri 3.1'deki gibi çalıştır.
3. Sabit noktayı doğrula.
4. İki AYRI koşunun çıktıları hash-hash eşleşmeli (determinizm).

### 3.3 Tarihsel zincir — `KanitGoSuzTarihce.sh`

Git geçmişindeki her commit için "önceki nesil ikilisiyle kaynağı derle → sabit
noktaya ulaş" iddiasını sıfırdan yeniden üretir. Tohumun "kara kutu" olmadığının
bağımsız kanıtıdır.

---

## 4. Nesil kayıt şeması (PHASE 4)

Her üretilen nesil için kayıt zorunludur:

| Alan | Açıklama | Örnek |
|------|----------|-------|
| `nesil` | üretilen nesil adı | `gen2` |
| `kaynak` | derlenen kaynak (hash ile) | `TancElf.tan` (sha256…) |
| `derleyen` | nesli derleyen ikili | `gen1` |
| `derleyen_kaynagi` | derleyenin üretim yolu | `TancElf TancElf.tan gen1` |
| `cikti_hash` | üretilen ikilinin hash'i | `md5:…` |
| `boyut` | bayt cinsinden boyut | `114998` |
| `deterministik` | iki koşu aynı hash mi | `EVET` |
| `bagimlilik` | üretim için gerekli araçlar | `YOK (yalnız TAN)` |
| `sabit_nokta` | komşu nesillerle birebir aynı mı | `gen2 == gen3: EVET` |

Kayıtlar `BOOTSTRAP_STATUS.md`'de tutulur.

---

## 5. Sabit nokta doğrulaması (PHASE 5)

- **Kriter:** `GenN == GenN+1` **byte-for-byte** (`cmp -s` / hash eşitliği).
- "Derleyici çalışıyor" tek başına self-hosting sayılmaz.
- Byte-for-byte eşitlik sağlanamazsa, **kesin deterministik olmayan bileşen**
  tek tek saptanmalıdır (zaman damgası, meta veri, sembol sıralaması, build id,
  sırasızlık/nondeterminizm) ve yalnızca O bileşen raporlanmalıdır — genel
  "sabit nokta" iddiası kurulmaz.

---

## 6. Kurallar (dokunulmazlar)

1. **`TancElf.tan` DEĞİŞTİRİLMEZ** (FROZEN). Bootstrap sorunları bu dosyada
   yama ile çözülmez.
2. **`f_harfMi` (veya herhangi bir bootstrap hatası) için ÖZEL DURUM yazılmaz.**
   Hata kaynağını gizleyecek/bypass edecek hiçbir kod eklenmez.
3. **Bootstrap hataları asla bastırılmaz/gizlenmez.** `bagla()`'nın "BAGLAMA
   HATASI: etiket bulunamadi" raporlaması, sessiz yutmayı engelleyen bilinçli
   bir korumadır ve korunur.
4. Hedef tutturulana (veya katı kanıtlı bir teknik engel belgelenene) kadar
   ilgisiz geliştirici araçları eklenmez.

---

## 7. Bilinen bootstrap hata ailesi (koruma bilgisi)

TancElf.tan'ın tip çıkarımı yalnız **değişken atamalarını** izler; **işlev
parametreleri** her zaman "tam" varsayılır. Bu temel sınır, şu sınıf hataları
üretmiştir (hepsi Go tohumuyla gizli kalır; yalnızca TAN→TAN derlemesinde
ortaya çıkar):

1. `bagla()`/`etiketBul()` — `op == "BAYT"` vb.
2. `bc(op,a1,a2)` — bant kodlama tabanı
3. `cagriSonucTipi(ad)` — tüm `t==`/`d==` çağrıları
4. `metneTamSayi(s)` — `s[i]` ham indeks
5. `metinVerisiBant(deger)` — metin literal indeksi
6. **`"f_" + ad` (işlev etiketi, TANIM tarafı)** — f_harfMi ailesi; `metinBirlestir`
   ile kapatıldı
7. `yeniEtiket(...)` — kontrol akışı etiketleri
8. `adAra(...)` — değişken yuva çözümlemesi
9. Sessiz yutma (bilinmeyen deyim) — artık `"DERLEME HATASI"` ile durur
10. "yeni yerleşik + aynı commit'te öz-kullanım" bootstrap sıralaması
    (`f_dosyaVarMi`) — shim tekniğiyle aşılır; iki commit'e bölünerek
    önlenebilirdi

Yeni yerleşik ekleyen katkılar: yerleşiği TancElf.tan'ın KENDİ kaynağında
kullanmadan ÖNCE derleyici desteğini ayrı commit'te ekleyin.

---

## 8. TancElf.tan'ın bilinen eksikleri (kullanıcı programları için)

Derleyici kendi kaynağını derlerken bunları kullanmaz (bootstrap'ı engellemez),
ama genel amaçlı kullanıcı programlarında tam ikame olmasını engeller:

- Ondalık (float) sayı literalleri desteklenmiyor (kasıtlı, net hata verir).
- `yazDosya`/`yaz_dosya` kullanıcı programından çağrılamıyor.
- `her x ["bir","iki"] içinde` döngüsünde değişken tipi yanlış çıkarılıyor.

Bunlar ayrı, regresyonla kapatılabilir görevlerdir — bootstrap hedefiyle ilgisi
yoktur.
