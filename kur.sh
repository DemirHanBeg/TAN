#!/bin/bash
# kur.sh — TAN toolchain'i derleyip kurar (Kaldıraç 5: tek binary).
# tan.tan → 'tan' native binary (tüm araçlar: simgeler/api/bagimlilik/ast/
# denetle/bicimlendir/kilit/dogrula/imzala/sürüm). Sıfır Go.
# Kullanım: bash kur.sh [hedef-dizin]   (varsayılan: mevcut dizin)
set -e
cd "$(dirname "$0")"

HEDEF="${1:-.}"

echo "TAN toolchain derleniyor (self-hosted, sıfır Go)..."
./TancElf tan.tan "$HEDEF/tan"
chmod +x "$HEDEF/tan"

echo "LSP sunucusu derleniyor..."
./TancElf araclar/lsp.tan "$HEDEF/tanlsp"
chmod +x "$HEDEF/tanlsp"

echo "Kuruldu: $HEDEF/tan  +  $HEDEF/tanlsp (LSP)"
echo ""
"$HEDEF/tan" sürüm
echo ""
echo "Deneyin:  $HEDEF/tan denetle <dosya.tan>"
