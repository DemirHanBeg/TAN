# TAN VS Code Eklentisi

TAN dili için LSP entegrasyonu. `tanlsp` sunucusuna bağlanır (sembol/tanı/hover/
git-tanıma).

## Kurulum
1. TAN toolchain'i derle: `bash kur.sh` → `tanlsp` binary üretir.
2. `tanlsp`'yi PATH'e ekle (ya da bu dizinde `npm install && npm run compile`).
3. Bu klasörü VS Code eklentisi olarak yükle (Extensions → Install from folder,
   ya da `vsce package`).

## Özellikler
- Sembol listesi (Outline)
- Canlı tanı (TAN001-006 hata/uyarı)
- Hover (işlev imzası)
- Tanıma git (Go to Definition)
