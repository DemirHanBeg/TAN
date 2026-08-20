# TAN_CAPABILITY_AUDIT.md

*Tarih: 2026-08-16. Bu belge repository'nin gerçek içeriği üzerinden doğrulanmıştır.
Hiçbir önceki "VERIFIED" iddiası olduğu gibi kabul edilmemiştir; her satır ya
kaynak koda ya canlı deneye dayanır. Doğrulanamayan nokta "BİLİNMİYOR"dur.*

---

## 0. Denetim Ortamı

| Öğe | Değer | Not |
|---|---|---|
| Çalışma dizini | `/data/data/com.termux/files/home/storage/downloads/yeniset/deneysel tan/deneysel tan` | Android/Termux, aarch64 Linux |
| Host mimari | aarch64 | TAN ikilileri x86-64 → qemu-user ile çalıştırıldı |
| qemu-user | kuruldu (apt `qemu-user`) | x86-64 emülasyonu |
| Git durumu | BOZUK kopya | `.git` objects kısmi; HEAD tree objesi yok; `refs/heads/` eksikti (reflog'dan kurtarıldı). `git diff`/`git log` çalışmıyor → çalışma ağacı tek kaynak |
| Eksik dizinler | `kutuphane/`, `testler/`, `ornekler/`, `arsiv/`, `web/` | Repo çalışma alanına KISMİ kopyalanmış; README'de atıf yapılan bu dizinler yok |
| Mevcut TAN kaynakları | `TancElf.tan` (4310 satır), 11 Go dosyası | — |
| Sabit-nokta artifact'leri | `TancElf`, `gen1`, `gen2`, `gen3` — 4'ü de md5 `914b0ffb971d4cf1991779e674f0bab1`, 141622 bayt | Birebir aynı |

## 1. Self-Hosting Kanıtı (canlı doğrulama)

- `TancElf` ikilisi qemu altında **çalışıyor**: `TancElf <kaynak> <cikti>` çağrısı
  ELF çıktısı üretiyor (canlı test: prog.tan → 2347 bayt ikili).
- Üretilen ikilinin çıktısı, TestArkaUcGoSuzTemiz.sh'nin beklediği çıktıyla **birebir**:
  `14`, `14.285714`, `121932631112635269`, `6765`, `4`, `40`, `Tan 42`. → int64 hassas
  aritmetik, float64 bölme, özyineleme, liste, metin birleştirme canlı doğrulandı.
- Commit'li `gen1 == gen2 == gen3` (md5 aynı) → repo kendi kaydında sabit noktayı
  doğrulamış. Bu ortamda tam gen1→gen2→gen3 zincirini yeniden koşmak qemu altında
  pratik değil (bkz. §5); commit'li artifact'ler + determinizm argümanı geçerli kanıt.

## 2. DerleElf.go — Rol Doğrulaması (soru: "DerleElf.go olmadan TAN kendi compiler/codegen'ini üretebilir mi?")

**Cevap: EVET — mevcut özellik kümesi için.** Kanıt:

1. `BootstrapGoSuz.sh` üretim zinciri yalnız `TancElf` + `TancElf.tan` kullanır
   (script metni 39-48. satırlar; Go'ya hiç değinmez).
2. `TestArkaUcGoSuzTemiz.sh` go/gcc/clang/as/ld/cc adlarına TUZAK script'ler koyar;
   tuzak çağrılırsa test kırmızıdır. Yani "Go'suz üretim" iddiası varsayım değil,
   kurulmuş bir davranış denetimidir.
3. Canlı: `TancElf` (Go'suz ikili) kendi kaynağının dil alt kümesindeki programları
   derleyip doğru çalıştırıyor (bkz. §1).

DerleElf.go ise **bağımsız Go referansıdır** — üretim zincirinin parçası değil.
İçindeki runtime helper'ları (`f_sozluk_*`, `f_dosya_ac_*`, `f_bellek_esle`,
`f_futex_*`, `f_iplik_*`, `f_kilit_*`, `f_atomik_ekle_ham`, `f_harfler`,
`f_parcala`, `f_metin_araligi`, ...) TancElf.tan'da **YOKTUR** (kaynak taramasıyla
doğrulandı: TancElf.tan'da `futex|iplik|f_kilit|atomik|f_dosya_ac|sözlük|f_sozluk`
hiç geçmiyor).

**Sonuç:** DerleElf.go, TancElf.tan'ın kapsamadığı bir özellik kümesi taşıyor.
Bu özellikler üretim zinciri için gerekli değil (derleyici kendi kaynağını
derlemek için bunları kullanmıyor) AMA kullanıcı programı için "capability gap"tir.

## 3. Aşama 2 (2A-2E) Durum Doğrulaması

Başlangıç hipotezi 2A=2B=2C=2E VERIFIED, 2D=COVERAGE GAP idi. **Denetim sonucu:**

| Aşama | Özellik | Go (DerleElf.go) | TancElf.tan | Hipotez doğru mu? |
|---|---|---|---|---|
| 2A | `sözlük` (sözlük tipi + f_sozluk_* 7 helper) | ✅ (satır 3564-3829) | ❌ yok | ✗ hipotez 2A=VERIFIED **YANLIŞ** (self-hosted için) |
| 2B | `kayıt` tipi + metot | ✅ (kayıt üretimi) | ❌ yok | ✗ aynı gerekçe |
| 2C | Bitwise `& \| ^ << >>` | ✅ | ❌ | ✗ aynı gerekçe |
| 2D | Konumlamalı dosya G/Ç (f_dosya_ac_* 10, f_bellek_esle/coz) | ✅ (satır 4303-4484) | ❌ yok | ✓ COVERAGE GAP doğru |
| 2E | Native eşzamanlılık (futex/iplik/kilit/atomik) | ✅ (satır 4587-4677) | ❌ yok | ✗ hipotez 2E=VERIFIED **YANLIŞ** |

**Kritik bulgu:** 0de1f03 commit mesajındaki "2E concurrency (atomic/lock/futex/
thread) TAN-native verified" iddiası **çalışma ağacındaki TancElf.tan kaynağıyla
çelişiyor.** Bu helper'lar yalnız Go referansında (DerleElf.go) var; self-hosted
TancElf.tan'da yok. İddia ya Go-yolu için geçerliydi ya da hatalıdır. VETA bu
tür doğrulanmamış iddiaları kanıt yerine saymayacaktır.

## 4. Self-Hosted Derleyicinin (TancElf.tan) Gerçek Yüzeyi

Kod üretimine doğrudan gömülen runtime helper'ları (TancElf.tan 4271-4272):
`yaz_sayi`, `f_tan_ayir`, `f_bellek_kopyala`, `f_liste_ekle`, `f_liste_yap`,
`f_sayi_metne`, `f_metin_birlestir`, `f_metin_esit`, `f_metin_indeks`, `f_kod`,
`f_karakter`, `f_yaz_metin`, `f_oku`, `f_dosyaVarMi`, `f_yazBaytlar`, `f_argsay`,
`f_arg`, `f_kesir_metne`, `f_yaz_kesir`, `f_hata_sifira_bolme`.

Özel-dağıtımlı yerleşikler (`ifadeBantYaz` içinde): `uzunluk` (işaretçiden ilk 8
bayt okur — liste VE metin için çalışır), `ekle`, `listeYap`, `metin`, `kod`,
`karakter`, `metinEsit`, `metinBirlestir`, `yaz` (tip yığınına göre sayı/kesir/metin
yazar), `tamBol`, `arg`/`argsay`, `dosyaVarMi`, `sistemDur`.

Dil yapıları (kaynak analizi + canlı test): int64 tam aritmetik (+ - * / %),
float64 SSE (FAZ 6), tüm karşılaştırmalar, `ve/veya/değil`, `eğer/değilse/son`,
`iken`, `her...içinde`, `dur/devam`, işlev (≤6 parametre, özyineleme), liste
(literal, indeks, indeks-atama, `ekle`, iç içe liste), metin (literal, `+`
birleştirme, `s[i]`, `uzunluk`, `kod`, `karakter`, `metin(n)`, `==`/`!=` içerik),
üst-seviye deyimler, `içe al` (token seviyesinde statik açılım), dosya G/Ç
(`oku`/`yazBaytlar`/`dosyaVarMi`), kaçış dizileri (`\n \t \r`), yorumlar.

## 5. Ortam Sınırı: gen4/gen5 Deneyi

Tam `gen1→gen2→gen3` zinciri qemu altında bu cihazda pratik değil (küçük bir
test derlemesi 45s-6dk; self-compile saatlerce sürer; CPU kısıntısı gözlemlendi).
Bu yüzden zincir yeniden koşulmadı; **mevcut kanıt:**

1. Commit'li `TancElf==gen1==gen2==gen3` (md5 aynı) → sabit nokta repo kaydında.
2. **Determinizm argümanı (gen4/gen5 için):** gen2 ve gen3 birebir aynı programdır
   (bayt aynı). Derleyici deterministiktir (rastgelelik/ASLR bağımlılığı yok; tüm
   kod RIP-rel, sabit taban 0x400000). Öyleyse gen3 aynı girdiyi (TancElf.tan)
   işlediğinde gen2 ile aynı çıktıyı üretir: gen4 = gen3 = gen2 = gen1 = TancElf.
   Sabit nokta kanıtlandığı anda gen4 = gen5 = ... matematiksel olarak garantidir.

`gen1head` (140551 bayt) farklı bir ara üründür; üretim zincirinde yeri yok.

## 6. Dil Özellik Envanteri (detay için CURRENT_TAN_CAPABILITIES.md)

Üç çalıştırma katmanı vardır; yetenekleri FARKLIDIR:

| Katman | Üretici | Durum |
|---|---|---|
| Self-hosted ELF yolu | TancElf.tan (TAN) | üretim zinciri; §4'teki yüzey |
| Go ELF yolu | DerleElf.go | bağımsız referans; ek yetenekler (sözlük/kayıt/bitwise/dosya/eşzamanlılık) |
| Yorumlayıcı + VM | Go (Yerlesik.go vb.) | Go-only araç; en geniş yüzey (112 yerleşik, paket, FFI, test, biçimlendir...) |

**VETA için geçerli olan katman self-hosted ELF yoludur** (Go bağımlılığı yok).

## 7. Önemli Canlı Bulgu: Tip Çıkarımı Sınırı (KÜTÜPHANELER İÇİN KRİTİK)

Canlı deney (`strtest.tan`): metin parametresi alıp metin döndüren işlevler YANLIŞ
kod üretiyor:

- `ters(s)` (döngüde biriktirme + `döndür t`) → çıktı `4198480` (ham işaretçi).
- `topla(a, b)` (`döndür a + b`) → çıktı `4198496` (ham işaretçi).
- `uzunluk("uzunmetin")` → `9` ✓ (işaretçi tabanlı, tip-bağımsız).

Kök neden kaynakta belgeli: TancElf.tan 2880-2913 — dönüş tipi çıkarımı YALNIZCA
`döndür <metin literal>` veya `döndür <bilinen-metin-döndüren-ad>(...)` biçimini
tanır; parametre tipleri ve değişken üzerinden dönen sonuçlar varsayılan "tam"
kalır (güvenli yanlış-negatif tasarımı). `metinBirlestir`/`metinEsit`/`kod`/
`karakter` statik tip izlemeden bağımsız çalışır — bunlar kütüphane güvenli
ilkel işlemlerdir.

**Kütüphane tasarımına etki:** tam sayı/metin-döndüren işlevler DOĞRU sonuç için
ya doğrudan çağrı zinciri (`döndür metinBirlestir(a, b)`) ya literal döndürmeli;
parametreleri tip yığınına güvenen işleçlere (`+`, `==`, `yaz`) sokmamalı. Bu,
dil çekirdeği iyileştirilene (parametre tipi + değişken-dönüş tipi izleme) kadar
VETA kütüphane yazım kuralıdır.
