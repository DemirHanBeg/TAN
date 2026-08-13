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

### 2.2 Go'suz birincil zincir (`BootstrapGoSuz.sh`)

```
./TancElf TancElf.tan gen1    → 114998 bayt
./gen1    TancElf.tan gen2    → 114998 bayt
./gen2    TancElf.tan gen3    → 114998 bayt
SABİT NOKTA: gen2 == gen3
```

Hiçbir dış araç çağrılmadı. Bu, `BOOTSTRAP_ARCHITECTURE.md` §1'deki hedef
zincirin (TAN→TAN→TAN→sabit nokta) zaten kurulu olduğunu gösterir.

---

## 3. Nesil Kayıtları (PHASE 4 şeması)

| nesil | kaynak | derleyen | derleyen_kaynagi | cikti_hash (md5) | boyut | deterministik | bagimlilik | sabit_nokta |
|-------|--------|----------|------------------|------------------|-------|---------------|------------|-------------|
| gen1 (çapraz) | TancElf.tan | ./tan (Go tohumu) | `go build` | — | 142944 | test edilecek (§7) | go (yalnızca bu yol) | — |
| gen1 (Go'suz) | TancElf.tan | ./TancElf (tohum) | `./TancElf TancElf.tan gen1` | — | 114998 | EVET (sabit nokta turu) | YOK | — |
| gen2 | TancElf.tan | gen1 | `gen1 TancElf.tan gen2` | `32727d5928eca67cc066580faea85251` | 114998 | EVET | YOK | gen2 == gen3 |
| gen3 | TancElf.tan | gen2 | `gen2 TancElf.tan gen3` | `32727d5928eca67cc066580faea85251` | 114998 | EVET | YOK | gen3 == gen2 |

Tohum artifact'i `TancElf` = `gen2` (birebir, md5 `32727d59…`) — yani tohum,
zincirin kendi sabit nokta çıktısının kopyasıdır.

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

## 6. Kararlar ve Eylemler

| Karar | Durum |
|-------|-------|
| Kanonik tohum = sabit-nokta artifact'i (`TancElf` == gen2) | commit'leniyor |
| `BootstrapGoSuz.sh`, `KanitGoSuzTarihce.sh` üretim zinciri script'leri | git'e alınacak (untracked) |
| `BOOTSTRAP_ARCHITECTURE.md`, `BOOTSTRAP_STATUS.md` | yeni (bu tur) |

---

## 7. Zincir Sertleştirme Sonuçları (canlı test, 14 Ağustos 2026)

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
