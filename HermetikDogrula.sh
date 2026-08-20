#!/bin/bash
# HermetikDogrula.sh — deterministik/hermetic build doğrulaması (Kaldıraç 3,
# madde 42/43 reproducibility). Verilen .tan dosyalarını N kez derler, çıktının
# BAYT-BİREBİR aynı olduğunu (belirlenimci) doğrular. Ayrıca self-host sabit
# noktasını kontrol eder. go/gcc/as/ld kullanmaz.
# Kullanım: bash HermetikDogrula.sh [dosya.tan ...]   (argsız: tüm araçlar + derleyici)
set -e
cd "$(dirname "$0")"

TUR=3   # kaç kez derle
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

hedefler="$*"
if [ -z "$hedefler" ]; then
    hedefler="TancElf.tan tan.tan araclar/simgeler.tan araclar/denetle.tan araclar/bicimlendir.tan kutuphane/sha256.tan"
fi

hata=0
for f in $hedefler; do
    ilk=""
    ayni=1
    for i in $(seq 1 $TUR); do
        ./TancElf "$f" "$TMP/out" >/dev/null 2>&1
        h=$(sha256sum "$TMP/out" | awk '{print $1}')
        if [ -z "$ilk" ]; then
            ilk="$h"
        elif [ "$h" != "$ilk" ]; then
            ayni=0
        fi
    done
    if [ "$ayni" -eq 1 ]; then
        echo "  [BELIRLENIMCI] $f  ($TUR derleme = $ilk)"
    else
        echo "  [BELIRSIZ] $f — derlemeler farklı!"
        hata=1
    fi
done

echo ""
echo "--- self-host sabit nokta ---"
./TancElf TancElf.tan "$TMP/g1" >/dev/null 2>&1 && chmod +x "$TMP/g1"
"$TMP/g1" TancElf.tan "$TMP/g2" >/dev/null 2>&1
if [ "$(sha256sum "$TMP/g1" | awk '{print $1}')" = "$(sha256sum "$TMP/g2" | awk '{print $1}')" ]; then
    echo "  [OK] g1==g2 sabit nokta"
else
    echo "  [KIRIK] sabit nokta"
    hata=1
fi

echo ""
if [ "$hata" -eq 0 ]; then
    echo "=== HERMETIK: TÜM BUILD'LER BELIRLENIMCI (go/gcc/as/ld yok) ==="
else
    echo "=== HERMETIK DOGRULAMA BASARISIZ ==="
    exit 1
fi
