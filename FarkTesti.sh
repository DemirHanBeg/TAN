#!/bin/bash
# FarkTesti — TancElf'in iki yolu AYNI bayti uretmeli.
#   A: yorumlayici ile kosan TancElf.tan
#   B: elf ile derlenmis TancElf (native)
# Sapma varsa arka uclar arasi anlam kaymasi vardir; erken yakala.
set -u
[ -f ./tan ] || go build -o tan .
./tan elf TancElf.tan /tmp/TancElfNative >/dev/null || { echo "TancElf derlenemedi"; exit 1; }

gecti=0; kaldi=0
for f in "$@"; do
    ./tan TancElf.tan "$f" /tmp/farkA >/dev/null 2>&1 || { echo "  [HATA] $f (yorumlayici)"; kaldi=$((kaldi+1)); continue; }
    /tmp/TancElfNative "$f" /tmp/farkB >/dev/null 2>&1 || { echo "  [HATA] $f (native)"; kaldi=$((kaldi+1)); continue; }
    if cmp -s /tmp/farkA /tmp/farkB; then
        echo "  [AYNI]   $f  ($(stat -c%s /tmp/farkA) bayt)"
        gecti=$((gecti+1))
    else
        echo "  [FARKLI] $f"
        kaldi=$((kaldi+1))
    fi
done
echo ""
echo "=== FARK TESTI: $gecti ayni, $kaldi sapma ==="
[ "$kaldi" -eq 0 ]
