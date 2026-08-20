# TAN — Proje Konvansiyonları

**%100 self-hosted, sıfır Go.** Tan dili artık tamamen kendi kendini derliyor:
`TancElf.tan` (Tan kaynağı) + `TancElf` (native x86-64 ELF ikilisi) üretim
zincirinin tamamıdır. Go referans motoru **2026-08-20'de tamamen kaldırıldı**
(11 .go dosyası, go.mod/sum, 3 Go binary silindi; git geçmişinde durur).

## Derleme & Çalıştırma (Go YOK)

- Derle: `./TancElf <kaynak>.tan <cikti>` → native ELF üretir, sıfır dış araç
  (go/gcc/clang/as/ld hiçbiri çağrılmaz).
- Self-derleme: `./TancElf TancElf.tan gen1` → yeni derleyici tohumu.
- WSL zorunlu (Linux ELF): işler `C:\Users\karab\Desktop\tanlar\elimizdekiler\tan`
  dizininden, WSL üzerinden yürür.

## Tooling (Tan'da yazılır, `araclar/`)

Go host'un sağladığı tüm tooling silindi; Tan'da yeniden yazılıyor:
- `araclar/simgeler.tan` — sembol listeleyici (işlev/değişken, --json)
- `araclar/denetle.tan` — blok dengesi linter'ı (--json)
- `araclar/bicimlendir.tan` — kanonik formatter (deterministik, idempotent)
- `kutuphane/tanoku.tan` — ortak tarama/tokenize yardımcıları (hepsi `içe al` eder)

Yeni araç yazarken: doğrulanmış yerleşikleri kullan (bkz. tanoku.tan başlığı),
`içe al "kutuphane/tanoku.tan"` ile ortak yardımcıları çek, `./TancElf` ile
derleyip WSL'de çalıştırarak doğrula.

## Hata Yönetimi (Tan)

Tan'ın kendi hata mekanizması: `dene`/`yakala` (codegen'de YOK, backlog) + derleme
zamanı `BAGLAMA HATASI` (linker, çözümlenemeyen etiket → exit(1)). Structured
diagnostics (TANxxxx kod + sebep + çözüm) yol haritasında (Kaldıraç 2).

## Test

`go test` YOK (hiç olmadı). Doğrulama shell script'leriyle (Go-free):
`TestArkaUcGoSuzTemiz.sh` (3 nesil sabit nokta + temiz ortam), `BootstrapGoSuz.sh`,
`KanitGoSuzTarihce.sh`. Yeni test önerirken bunlara ekle.

## Büyük Dosya Okuma Disiplini

`TancElf.tan` (~3550 satır) tek büyük dosya. Tam Read yapma — bkz.
`.claude/rules/elf-backend.md`. (Eski `DerleElf.go` Go referansı SİLİNDİ.)

## Not

Global `~/.claude/rules/ecc/golang/*.md` kuralları bu repoyla İLGİSİZDİR —
repoda Go yok. Bu CLAUDE.md geçerli.
