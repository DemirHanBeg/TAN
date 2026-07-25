#!/bin/bash
# GercekProgramlar — TestArkaUc.sh (elf regresyon) ve FarkTesti.sh
# (ornekler/) hicbirinin kapsamadigi gercek programlari ve testler/
# dizinini yorumlayiciyla kosup "HATA" cikip cikmadigini denetler.
#
# Neden ayri script: ekle() semantik hatasi (yerinde mutasyon -> yeni
# liste) bu programlarda kutuphane uzerinden 60+ yerde kirici cikti ama
# TestArkaUc.sh ELF sentetik testlerini, FarkTesti.sh ornekler/ dizinini
# sinar — ikisi de bu dosyalari hic calistirmadigi icin regresyon 5
# gercek programda bulununcaya kadar fark edilmedi.
#
# Kullanim: ./GercekProgramlar.sh
set -u

PROGRAMLAR="Kesim.tan Talay.tan Noral.tan Model.tan Tokenizer.tan Ornek.tan"

gecti=0
kaldi=0

kosVeDenetle() {
    local dosya="$1"
    local log="/tmp/gercekprog_$(basename "$dosya" .tan).log"
    ./tan "$dosya" > "$log" 2>&1
    local kod=$?
    if [ "$kod" -ne 0 ] || grep -qi "HATA" "$log"; then
        echo "  [HATA ] $dosya"
        tail -5 "$log" | sed 's/^/         /'
        kaldi=$((kaldi + 1))
    else
        echo "  [GECTI] $dosya"
        gecti=$((gecti + 1))
    fi
}

echo "=== Gercek programlar (yorumlayici) ==="
for f in $PROGRAMLAR; do
    kosVeDenetle "$f"
done

echo ""
echo "=== testler/ dizini (yorumlayici) ==="
for f in testler/*.tan; do
    kosVeDenetle "$f"
done

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
[ "$kaldi" -eq 0 ]
