---
paths:
  - "DerleElf.go"
  - "TancElf.tan"
---

# ELF Backend — Büyük Dosya Okuma Disiplini

Bu iki dosya self-hosting migration'ın çekirdeği (DerleElf.go: 5998 satır Go orijinali,
TancElf.tan: 3552 satır Tan portu). Tam Read yapma.

- Önce fonksiyon indeksi çıkar: `grep -n "^func " DerleElf.go` (Go) veya karşılık gelen
  Tan syntax pattern'i TancElf.tan için.
- Hedef fonksiyon/case bloğunu bul, offset/limit ile sadece o aralığı + birkaç satır
  bağlam oku.
- Port işlerinde (Go → Tan taşıma) iki tarafı da aynı şekilde hedefle, ikisini birden
  tam okuma.

Not: `paths:` frontmatter şu an Claude Code'da bug'lı (anthropics/claude-code#16299,
açık) — session başında koşulsuz yükleniyor, sadece DerleElf.go/TancElf.tan'a
dokunulduğunda değil. Bug düzelene kadar bu dosya her TAN session'ında yüklenecek
(küçük olduğu için sorun değil), ama içindeki paths: hedefleme henüz gerçek anlamda
çalışmıyor.
