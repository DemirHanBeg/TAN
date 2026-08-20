#!/bin/bash
# DerleTani.sh — derleyici hatalarını AI-native yapısal tanıya çevirir (Kaldıraç 2).
# denetle statik lint yapar; bu, DERLEYİCİNİN kendi hatalarını (tanımsız işlev/
# etiket = BAGLAMA HATASI, geçersiz deyim = DERLEME HATASI) aynı {tani:[...]}
# şemasına bağlar. Editör/AI hem lint hem derleyici hatasını tek formatta alır.
# Kullanım: bash DerleTani.sh <dosya.tan>
cd "$(dirname "$0")"

f="$1"
if [ -z "$f" ]; then
    echo '{"tani":[{"kod":"TAN000","onem":"hata","satir":0,"mesaj":"dosya verilmedi","sebep":"kullanım: DerleTani.sh <dosya.tan>","cozum":"bir .tan dosyası ver"}]}'
    exit 1
fi

TMP="$(mktemp)"
out=$(./TancElf "$f" "$TMP" 2>&1)
rm -f "$TMP"

if echo "$out" | grep -q "BAGLAMA HATASI"; then
    sym=$(echo "$out" | grep "BAGLAMA HATASI" | head -1 | sed 's/.*bulunamadi: *//; s/^f_//')
    printf '{"tani":[{"kod":"TAN005","onem":"hata","satir":0,"mesaj":"tanımsız işlev/etiket: %s","sebep":"çağrılan %s hiçbir yerde tanımlı değil ve yerleşik değil","cozum":"işlevi tanımla ya da doğru içe al ekle"}]}\n' "$sym" "$sym"
elif echo "$out" | grep -q "DERLEME HATASI"; then
    msg=$(echo "$out" | grep "DERLEME HATASI" | head -1 | sed 's/DERLEME HATASI: *//')
    printf '{"tani":[{"kod":"TAN006","onem":"hata","satir":0,"mesaj":"derleme hatası: %s","sebep":"derleyici bu deyimi/yapıyı çözemedi","cozum":"sözdizimini düzelt"}]}\n' "$msg"
else
    echo '{"tani":[]}'
fi
