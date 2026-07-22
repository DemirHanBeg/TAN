# Sürüm Notları

Tan **semver** kullanır: BÜYÜK.KÜÇÜK.YAMA

- **BÜYÜK** — geriye uyumsuz dil/API değişikliği
- **KÜÇÜK** — geriye uyumlu yeni özellik
- **YAMA** — geriye uyumlu hata düzeltmesi

`1.0.0` öncesinde KÜÇÜK sürümler uyumsuzluk içerebilir.

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
