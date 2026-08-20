#!/bin/bash
# TestAraclar.sh — TAN tooling birim testleri (Go-free, self-verification).
# TancElf ile test dosyalarını derler, çalıştırır, sonucu raporlar.
# Kullanım: bash TestAraclar.sh
set -e
cd "$(dirname "$0")"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

hata=0
for t in testler/*_testleri.tan; do
    echo "### $t"
    ./TancElf "$t" "$TMP/bin" >/dev/null
    chmod +x "$TMP/bin"
    if ! "$TMP/bin"; then
        hata=1
    fi
    # "TEST KALDI" çıktısı varsa başarısızlık say
    if "$TMP/bin" | grep -q "TEST KALDI"; then
        hata=1
    fi
    echo ""
done

if [ "$hata" -eq 0 ]; then
    echo "=== TUM ARAC TESTLERI GECTI ==="
else
    echo "=== BAZI TESTLER KALDI ==="
    exit 1
fi
