# ELF Backend — Büyük Dosya Okuma Disiplini

`TancElf.tan` (~3550 satır) self-hosting derleyicinin tamamıdır — Tan kaynağını
doğrudan x86-64 ELF'e derler, sıfır dış araç. **Tam Read yapma.**

(Not: Eski Go referansı `DerleElf.go` 2026-08-20'de SİLİNDİ — repo artık %100
self-hosted, sıfır Go. Sadece git geçmişinde durur.)

- Önce işlev indeksi çıkar: `grep -nE "^işlev |^kayıt " TancElf.tan`.
- Hedef işlev/bölümü bul, offset/limit ile sadece o aralığı + birkaç satır
  bağlam oku.
- Aynısı `araclar/*.tan` ve `kutuphane/tanoku.tan` tooling dosyaları için de
  geçerli (ama onlar küçük, gerekirse tam okunabilir).

Not: `paths:` frontmatter bilinçli yok. Repo %100 Tan (tek dil), path scoping
kazanç sağlamaz. Bu dosya küçük, her TAN session'ında koşulsuz yüklenir.
