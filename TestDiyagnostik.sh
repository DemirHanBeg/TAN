#!/bin/bash
# TestDiyagnostik — diyagnostik sisteminin sözleşmesini sınar.
#
# Her fixture: hata üretir (exit != 0), stderr'de TANxxxx kodu + konum
# (dosya:satir:sutun) + kaynak bağlamı + imleç (^) + öneri barındırır.
# TAN_DIYAGNOSTIK_JSON=1 ile tek satır makine-okunur JSON üretir.
# iyi.tan ise hatasız çalışmalı (exit 0, stderr boş).
#
# Kullanım: ./TestDiyagnostik.sh
set -u

DIZIN="testler/diyagnostik"
gecti=0
kaldi=0

# go veya /usr/local/go/bin/go ile derle (repo konvansiyonu: go test yok).
if [ ! -f ./tan ]; then
    if ! command -v go >/dev/null 2>&1; then
        PATH="/usr/local/go/bin:$PATH" go build -o tan . || { echo "derlenemedi"; exit 1; }
    else
        go build -o tan . || { echo "derlenemedi"; exit 1; }
    fi
fi

# denetle <fixture> <TANxxxx> <satir:sutun>
denetle() {
    local f="$1" kod="$2" konum="$3"
    local log="/tmp/td_$(basename "$f" .tan).log"
    ./tan "$DIZIN/$f" > /dev/null 2> "$log"
    local kodExit=$?
    local mesaj=""

    if [ "$kodExit" -eq 0 ]; then
        mesaj="exit 0 bekleniyordu"
    elif ! grep -q "$kod" "$log"; then
        mesaj="'$kod' bulunamadı"
    elif ! grep -q ":$konum:" "$log"; then
        mesaj="konum ':$konum:' bulunamadı"
    elif ! grep -q '^ *\^' "$log"; then
        mesaj="imleç satırı (^) bulunamadı"
    elif ! grep -q "öneri:" "$log"; then
        mesaj="öneri bulunamadı"
    fi

    if [ -z "$mesaj" ]; then
        echo "  [GECTI] $f ($kod, $konum)"
        gecti=$((gecti + 1))
    else
        echo "  [HATA ] $f -> $mesaj"
        head -2 "$log" | sed 's/^/         /'
        kaldi=$((kaldi + 1))
    fi
}

# denetleJson <fixture> <TANxxxx> <satir> <sutun>
denetleJson() {
    local f="$1" kod="$2" satir="$3" sutun="$4"
    local log="/tmp/td_json_$(basename "$f" .tan).log"
    TAN_DIYAGNOSTIK_JSON=1 ./tan "$DIZIN/$f" > /dev/null 2> "$log"
    local mesaj=""

    if ! head -1 "$log" | grep -q '^{'; then
        mesaj="JSON başlamıyor"
    elif ! grep -q "\"kod\":\"$kod\"" "$log"; then
        mesaj="JSON'da kod yok"
    elif ! grep -q "\"satir\":$satir," "$log"; then
        mesaj="JSON'da satir yok"
    elif ! grep -q "\"sutun\":$sutun" "$log"; then
        mesaj="JSON'da sutun yok"
    elif [ "$(wc -l < "$log")" -ne 1 ]; then
        mesaj="JSON tek satır değil ($(wc -l < "$log") satır)"
    fi

    if [ -z "$mesaj" ]; then
        echo "  [GECTI] $f (JSON: $kod, $satir:$sutun)"
        gecti=$((gecti + 1))
    else
        echo "  [HATA ] $f (JSON) -> $mesaj"
        head -1 "$log" | cut -c1-160
        kaldi=$((kaldi + 1))
    fi
}

echo "=== Diyagnostik: hata siteleri (insan çıktısı) ==="
denetle "bilinmeyen_karakter.tan" TAN1001 "1:6"
denetle "ayrilmis_kelime.tan"      TAN2002 "1:9"
denetle "tanimsiz.tan"             TAN4001 "2:5"
denetle "sifira.tan"               TAN4008 "2:10"
denetle "indeks.tan"               TAN4005 "2:10"
denetle "tip_uyusmazligi.tan"      TAN4007 "1:15"
denetle "modul.tan"                TAN7001 "1:5"

echo ""
echo "=== Diyagnostik: JSON çıktısı ==="
denetleJson "bilinmeyen_karakter.tan" TAN1001 1 6
denetleJson "tanimsiz.tan"            TAN4001 2 5
denetleJson "sifira.tan"              TAN4008 2 10
denetleJson "modul.tan"               TAN7001 1 5

echo ""
echo "=== Diyagnostik: hatasız program ==="
if ./tan "$DIZIN/iyi.tan" > /tmp/td_iyi.out 2> /tmp/td_iyi.log; then
    if [ -s /tmp/td_iyi.log ]; then
        echo "  [HATA ] iyi.tan stderr'de çıktı üretti"
        kaldi=$((kaldi + 1))
    elif grep -q "merhaba diyagnostik" /tmp/td_iyi.out; then
        echo "  [GECTI] iyi.tan (çıktı: merhaba diyagnostik)"
        gecti=$((gecti + 1))
    else
        echo "  [HATA ] iyi.tan beklenen çıktıyı üretmedi"
        kaldi=$((kaldi + 1))
    fi
else
    echo "  [HATA ] iyi.tan exit 0 bekleniyordu"
    kaldi=$((kaldi + 1))
fi

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
[ "$kaldi" -eq 0 ]
