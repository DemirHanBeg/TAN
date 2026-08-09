# ELF Backend — Büyük Dosya Okuma Disiplini

Bu iki dosya self-hosting migration'ın çekirdeği (DerleElf.go: 5998 satır Go orijinali,
TancElf.tan: 3552 satır Tan portu). Tam Read yapma.

- Önce fonksiyon indeksi çıkar: `grep -n "^func " DerleElf.go` (Go) veya karşılık gelen
  Tan syntax pattern'i TancElf.tan için.
- Hedef fonksiyon/case bloğunu bul, offset/limit ile sadece o aralığı + birkaç satır
  bağlam oku.
- Port işlerinde (Go → Tan taşıma) iki tarafı da aynı şekilde hedefle, ikisini birden
  tam okuma.

Not: `paths:` frontmatter bilinçli olarak yok. Repo zaten %100 Go/Tan (TAN, TS/CSS
gibi karışık bir stack değil), path scoping burada hiçbir kazanç sağlamazdı — üstüne
`paths:` mekanizması Claude Code'da bug'lı (anthropics/claude-code#16299 ve ilişkili
issue'lar, hepsi açık). Bu dosya küçük (< 30 satır), her TAN session'ında koşulsuz
yüklenmesi sorun değil.
