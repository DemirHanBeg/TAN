# TAN — Proje Konvansiyonları

Self-hosting derleyici projesi (Tan dili). Repo'nun büyük kısmı Go (host derleyici) +
kendi kendini derleyen Tan portu (TancElf.tan).

## Hata Yönetimi

Bu repo global `golang/coding-style.md`'deki `fmt.Errorf("%w", err)` idiomunu KULLANMIYOR.
Kendi hata mekanizması var: `Hata.go` içindeki `TanHata` struct + `firlat(satir, bicim, args...)`,
panic/recover tabanlı. Yeni Go kodu yazarken bu deseni izle, stdlib error-wrapping ekleme.

## Test

`_test.go` / `go test` yok. Doğrulama shell script'leriyle yapılıyor:
`Olcum.sh`, `TestArkaUc.sh`, `FarkTesti.sh`, `KanitGoSuzTarihce.sh`. Yeni test önerirken
bu script'lere ekle, `go test` altyapısı kurma (proje bunu kullanmıyor).

## Büyük Dosyalar

`DerleElf.go` (5998 satır) ve `TancElf.tan` (3552 satır) için okuma disiplini:
bkz. `.claude/rules/elf-backend.md`.

## Not

Global `~/.claude/rules/ecc/golang/*.md` kuralları (varsa yüklendiyse) bu dosyayla çelişirse
bu CLAUDE.md geçerli — repo'nun gerçek konvansiyonu burada.
