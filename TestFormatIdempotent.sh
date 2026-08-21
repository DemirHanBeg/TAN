#!/bin/bash
# TestFormatIdempotent.sh — bicimlendir'in idempotent olduğunu doğrular.
#
# Amaç: format(format(kaynak)) == format(kaynak) invariant'ı — formatter
# ikinci kez çalıştırıldığında ÇIKTIYI DEĞİŞTİRMEMELİ. Bozulursa formatter
# kararsızdır (editör kaydet-format döngüsünde dosya sürekli değişir) ve
# codegen pipeline'ında altta yatan bir sorunun belirtisi olabilir.
#
# Kapsam: ornekler/, kutuphane/, araclar/ altındaki tüm .tan dosyaları.

set -u
cd "$(dirname "$0")"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

./TancElf tan.tan "$TMP/tan" >/dev/null

FAIL=0
SAYI=0
for f in ornekler/*.tan kutuphane/*.tan araclar/*.tan; do
    [ -f "$f" ] || continue
    SAYI=$((SAYI + 1))
    "$TMP/tan" bicimlendir "$f" > "$TMP/f1.tan" 2>/dev/null
    "$TMP/tan" bicimlendir "$TMP/f1.tan" > "$TMP/f2.tan" 2>/dev/null
    if ! cmp -s "$TMP/f1.tan" "$TMP/f2.tan"; then
        echo "  [KALDI] $f — format(format(x)) != format(x)"
        FAIL=1
    fi
done

if [ "$FAIL" -eq 0 ]; then
    echo "=== TestFormatIdempotent: $SAYI dosya, HEPSİ İDEMPOTENT ==="
    exit 0
else
    echo "=== TestFormatIdempotent: BAŞARISIZ ==="
    exit 1
fi
