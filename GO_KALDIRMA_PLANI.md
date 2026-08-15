# GO_KALDIRMA_PLANI — Go Bileşen Denetimi ve Kaldırma Planı (Faz 7)

*Kullanıcı direktifi (14 Ağustos 2026): Aşama 1'den sonra "ULTIMATE GO REMOVAL
REQUIREMENT" yolundan devam. Her Go bileşeni için: sorumluluk → TAN karşılığı
bul/yarat → Go'ya karşı doğrula → entegre et → zinciri yeniden kur → sabit
nokta + regresyon → REMOVABLE işaretle. Final faz: Go kaynakları + go.mod
kaldırma, temiz ortamda üretim. Başarı koşulu: Go'suz üretim derleyicisi,
GenN == GenN+1, temiz bootstrap.*

**Kapsam kuralı:** "REMOVABLE" = bu Go dosyası SİLİNDİĞİNDE üretim zinciri
(`TancElf.tan` → ikili) hâlâ kuruluyor ve sabit nokta veriyor. Üretim zinciri
yalnız `TancElf` ikilisi + `TancElf.tan` kaynağı + script'lerden oluşur; Go
dosyaları yalnız Go host'u (tarihsel tohum üreticisi + çapraz-kontrol aracı)
inşa eder. Bu yüzden "REMOVABLE" sütunu hemen hemen hepsi için EVET'tir; asıl
karar, silmenin HİÇBİR yetenek kaybettirmemesi için hangi eksiklerin TAN'a
taşınacağıdır (parite kararı).

---

## 1. Denetim Tablosu (GO COMPONENT | TAN REPLACEMENT | VERIFIED | REMOVABLE)

| Go bileşeni | Sorumluluk | TAN karşılığı (TancElf.tan) | VERIFIED | REMOVABLE |
|---|---|---|---|---|
| **DerleElf.go** (5998) | ELF arka ucu: AST→kod→ELF; tip çıkarımı; runtime helper emisyonu; budama; `içe al` açılımı; sembol tablosu | TancElf.tan (tüm BÖLÜM 3-4: bant üreticileri, `ifadeBantYaz`, `deyimDerle`, `islevDerle`, emit fonksiyonları, `elfUret` 1713, `tokenGenislet` 3435, `ulasilabilirIslevleriTara` Faz 5, `tipSabitNoktasiTara` Faz 4) | ✅ sabit nokta gen1==gen2==gen3; FarkTesti 13/13; TestArkaUc elf 20/20; GercekProgramlar 43/43; TestArkaUcGoSuzTemiz GEÇTİ | ✅ |
| **Lexer.go** (230) | 15 token, 24 anahtar kelime | BÖLÜM 1: `tokenle`(96), `tokenKodla`, `anahtarMi`, `kacisCoz` | ✅ (öz-kullanım: derleyici kendi kaynağını tokenler) | ✅ |
| **Parser.go** (699) | AST, 25 düğüm, 8 öncelik katmanı | BÖLÜM 2: RPN bant üreticileri `ifadeAyristir`+7 katman (**tasarım farkı:** AST yok, düz postfix bant) | ✅ | ✅ |
| **Optimize.go** (343) | `sabitKatla`(164), `cebirsel`(263), koşul/ölü blok(49-121) | Faz 1: `ikiliBirlestir`(677), `sabitSayisalBant`(579), `tekTamSabit`/`tamSabitMetin`/`mutlak53uAsiyorMu`/`tamMetne`, `kosulDogruluk`(1027), `zincirAtla`(1060) | ✅ TestOptimize.sh 7/7; 2^53 paritesi güncel Go host ile exit 1 birebir; sabit nokta | ✅ |
| **Hata.go** | `TanHata` struct + panic/recover | TancElf hata deseni: `yaz(...)` + `sistemDur(1)` (hata mesajı + exit kodu) | ✅ (2^53, sıfıra bölme, DERLEME HATASI yolları doğrulandı) | ✅ |
| **Sayi.go** (171) | int64/float64 tip sistemi (`/` her zaman float64, `tamBol` keser) | `tamBol` (ifadeBantYaz), `f_sayi_metne`/`f_kesir_metne`, SAYITAM/SAYIKESIR ayrımı | ✅ (FAZ 6 ondalık dahil) | ✅ |
| **SabitTam.go** (161) | sabit-genişlik tam sayı kurucuları `u8..i64` | yok (TancElf alt kümesinde sabit-genişlik tipleri yok) | ⚠️ parite gereği: TancElf.tan bu yerleşikleri KULLANMIYOR → üretim için gerekmez; kullanıcı programı desteği istenirse ayrı iş | ✅ |
| **Yerlesik.go** (899) + **YerlesikFFI_linux/diger.go** | 112 yerleşik isim + camelCase alias + FFI | TancElf özel-dağıtımlı yerleşikler (`uzunluk`, `ekle`, `metin`, `kod`, `karakter`, `metinEsit`, `sistemDur`, `tamBol`...) + `callBant("f_"+ad)`; ELF yolu kendi f_* eşleşmesini kullanır — Yerlesik.go ELF için GEREKSİZ | ✅ (ELF yolunun yerleşik seti TAN'da) | ✅ |
| **Yorumlayici.go** (944) | yorumlayıcı | yok — **kapsam dışı** (BOOTSTRAP §11; TancElf doğrudan ELF üretir) | — | ✅ |
| **VM.go** (302 içinde Derleyici.go) | bytecode VM + derleyici | yok — kapsam dışı (Go-only araç) | — | ✅ |
| **MainNative.go / MainWasm.go** | giriş noktaları (native/wasm) | yok (TancElf kendi sürücüsüne sahip, sürücü ~4610) | — | ✅ |
| **Kopru.go** (66) | `köprü` iskeleti | yok — kapsam dışı (Go elf'inde de YOK; BACKLOG) | — | ✅ |
| **Paket.go** (899) + **Paketle.go** (94) | `tan paket` yöneticisi + TANPAKET1 paketleme | yok — Go-only geliştirici araçları (kapsam dışı) | — | ✅ |
| **Modul.go** (151) | `modulAra` 7 basamaklı arama | **gereksiz:** ELF yolu `modulAra` KULLANMAZ — kendi statik açılımı `iceAlYoluCoz`/`tokenGenislet` (TancElf.tan:3398-3435) | ✅ (`içe al` TAN'da) | ✅ |
| **Cikti.go** (10) | çıktı yardımcıları | yok — Go-only | — | ✅ |
| **go.mod** | Go modül tanımı | — | — | ✅ (final fazda) |

**Özet:** 21 Go dosyası + go.mod. Hepsi üretim zinciri için REMOVABLE; parite
boşlukları yalnız kullanıcı programı özelliklerinde (sözlük, kayıt,
sabit-genişlik tipleri, bitwise, konumlamalı dosya G/Ç, eşzamanlılık) —
TancElf.tan'ın KENDİSİ bunların hiçbirini kullanmaz.

---

## 2. Parite Boşlukları (Go'da var, TAN'da yok) — Taşıma Kararı

BOOTSTRAP.md §5/§6 tablolarından güncel durum (Faz 1-6 sonrası):

| Özellik | Go-elf | TancElf.tan | Karar |
|---|---|---|---|
| Ondalık literal + SSE aritmetik | ✅ | ✅ **FAZ 6 YAPILDI** (SAYIKESIR 789, SSE 1809, f_kesir_metne 2835, f_yaz_kesir 2972) | taşındı |
| `mantık` tip ayrımı | ⚠️ | ✅ **FAZ 2 YAPILDI** | taşındı |
| Liste eleman tipi (her-literal) | ✅ | ✅ **FAZ 3 YAPILDI** | taşındı |
| Dönüş tipi izleme (eksik #1) | ✅ | ✅ **FAZ 4 YAPILDI** | taşındı |
| Ölü işlev/yardımcı budaması | ✅ | ✅ **FAZ 5 YAPILDI** | taşındı |
| Optimizer (Faz 1) | ✅ | ✅ **AŞAMA 1 YAPILDI** | taşındı |
| `sözlük` tipi + f_sozluk_* (7) | ✅ | ❌ (paralel listelerle taklit) | **AŞAMA 2A** (roadmap Faz 7) |
| `kayıt` tipi + sema + metot | ✅ | ❌ | **AŞAMA 2B** (düşük öncelik — TancElf kaynak kullanmıyor) |
| Bitwise `& \| ^ << >>` | ✅ | ❌ (açık hata) | **AŞAMA 2C** (düşük öncelik) |
| Konumlamalı dosya G/Ç (f_dosya_ac_* 10, f_bellek_esle/coz) | ✅ | ❌ | **AŞAMA 2D** (test programları kullanmıyorsa opsiyonel) |
| Native eşzamanlılık (f_futex/f_iplik/f_kilit/f_atomik) | ✅ | ❌ | **AŞAMA 2E** (opsiyonel) |
| `dene/yakala`, `köprü`, paket yöneticisi, yorumlayıcı/VM | ❌/Go-only | ❌ | **taşınmaz** (kapsam dışı, BOOTSTRAP §11) |

Taşıma sırası önceliği: **AŞAMA 2A (sözlük)** en yüksek — derleyici içi
`islevTipKutu`/`adlarKutu` gibi paralel-liste yapılarının gerçek sözlüğe
taşınması uzun vadede kazandırır (roadmap §10 Faz 7). 2B-2E yalnız kullanıcı
programı özellikleridir; derleyici öz-kaynağı kullanmadığı sürece sabit nokta
riskleri düşük ama fayda da düşük → öncelik sıralaması kararı `kararlar
listesi.txt` K14.

---

## 3. Final GO REMOVAL Fazı (tüm 2A-2E bittikten sonra)

**Amaç:** go/gcc/as/ld/libc çağrılmayan, Go kaynakları ve go.mod olmayan,
yalnız TAN ikilisi + TAN kaynağı + script'lerden oluşan üretim.

**Adımlar:**
1. Go dosyalarının tamamını sil: `DerleElf.go Lexer.go Parser.go Optimize.go
   Hata.go Sayi.go SabitTam.go Yerlesik.go YerlesikFFI_*.go Yorumlayici.go
   VM.go Derleyici.go MainNative.go MainWasm.go Kopru.go Paket.go Paketle.go
   Modul.go Cikti.go go.mod` + refactor WIP (untracked Api.go/Ast.go/... — ayrı
   karar, kullanıcı onayı; kararlar listesi).
2. Git geçmişi korunur (dosyalar silinir ama tarihte kalır — gitignore'a
   gerek yok).
3. **Çapraz-kontrol araçlarının yeniden düzenlenmesi:** `TestArkaUc.sh elf` ve
   FarkTesti'nin Go-referans kolları `./tan` ikilisini kullanır — ya (a) prebuilt
   `tan` ikilisi "referans artifact" olarak korunur (kaynak değil), ya da (b)
   bu kollar kaldırılıp yerine TAN-içi karşılaştırma (TestOptimize,
   TestArkaUcGoSuzTemiz) birincil yapılır. **Karar: (b)** — Go-free başarı
   kriteriyle uyumlu (GercekProgramlar + TestOptimize + FarkTesti'nin Go'suz
   kolu yeter).
4. **Temiz ortam kanıtı:** `TestArkaUcGoSuzTemiz.sh` zaten go/gcc/as/ld tuzağı
   koyuyor; `PATH`'e sahte `go` konulduğu halde üretim GEÇER. Bu, Go dosyaları
   silindikten sonra da aynen koşulur + `ldd` boş + `grep -r go build` = 0.
5. **Başarı kriterleri (hepsi):**
   - `BootstrapGoSuz.sh` → gen1==gen2==gen3 (sabit nokta)
   - `TestArkaUcGoSuzTemiz.sh` → go/gcc/as/ld hiç çağrılmadı
   - `TestOptimize.sh` → 7/7
   - `GercekProgramlar.sh` → 43/43
   - `FarkTesti.sh ornekler/*.tan` → 13/13 (Go'suz kol)
   - Repoda `go.mod` + `*.go` yok
   - `./TancElf TancElf.tan gen1` temiz ortamda çalışır

**Riskler:** (1) TestArkaUc.sh'nin Go-referans kolları silinince gerçekleşen
yetenek kaybı (Go'ya karşı birebir değer paritesi denetimi) — 2A-2E'nin
kendileri Go'ya karşı doğrulanmış olduğundan kabul edilir; (2) refactor WIP
(untracked Go dosyaları) kullanıcının ayrı çalışması — final silmeden ÖNCE
kullanıcıya sorulur (STOP noktası).

---

## 4. İlgili Dosyalar

- BOOTSTRAP.md (bileşen envanteri §3-§6, Go bağımlılıkları §8)
- BOOTSTRAP_ROADMAP.md (§4 Faz 1 ✅, §12 takvim)
- BOOTSTRAP_STATUS.md (güncel doğrulanmış durum)
- `kararlar listesi.txt` (K14: Faz 7 kararı; K15: VETA)
- TestOptimize.sh, BootstrapGoSuz.sh, TestArkaUcGoSuzTemiz.sh, FarkTesti.sh,
  GercekProgramlar.sh, TestArkaUc.sh
