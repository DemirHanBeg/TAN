#!/bin/bash
# TestTest — "tan test" komutunu ve bekle/bekleEsit yerleşiklerini sınar.
# Kullanım: ./TestTest.sh
set -u

cd "$(dirname "$0")"

gecti=0
kaldi=0

denetle() {
    local aciklama="$1"
    local beklenen="$2"
    local gercek="$3"
    if [ "$beklenen" = "$gercek" ]; then
        echo "  [GECTI] $aciklama"
        gecti=$((gecti + 1))
    else
        echo "  [HATA ] $aciklama"
        echo "         beklenen: $beklenen"
        echo "         gercek : $gercek"
        kaldi=$((kaldi + 1))
    fi
}

echo "=== tan test komutu ==="

# 1. Varsayılan keşif: exit 0, SONUÇ satırı
./tan test > /tmp/tantest1.log 2>&1
denetle "varsayılan keşif exit 0" "0" "$?"
grep -q "SONUÇ" /tmp/tantest1.log && grep -q "gecti" /tmp/tantest1.log
denetle "varsayılan keşif SONUÇ satırı" "0" "$?"

# 2. Bilerek başarısız test: exit 1 + HATA + kaldi
./tan test testler/_ornekler/Kirmizi_test.tan > /tmp/tantest2.log 2>&1
kod=$?
denetle "başarısız test exit 1" "1" "$kod"
grep -q "HATA" /tmp/tantest2.log
denetle "başarısız test HATA raporu" "0" "$?"
grep -q "kaldi" /tmp/tantest2.log
denetle "başarısız test SONUÇ'te kaldi" "0" "$?"

# 3. --liste: çalıştırmadan listeler, kategori gösterir
./tan test --liste testler/birim/Hesap_test.tan > /tmp/tantest3.log 2>&1
kod=$?
denetle "--liste exit 0" "0" "$kod"
grep -q "\[birim\] testler/birim/Hesap_test.tan" /tmp/tantest3.log
denetle "--liste kategori gösterimi" "0" "$?"
grep -q "SONUÇ" /tmp/tantest3.log
denetle "--liste SONUÇ basmaz" "1" "$?"

# 4. # tür: yorumu dizin varsayılanını ezer
./tan test --liste testler/_ornekler/TurEzme_test.tan > /tmp/tantest4.log 2>&1
grep -q "\[regresyon\]" /tmp/tantest4.log
denetle "# tür: yorumu dizini ezer" "0" "$?"

# 5. --json: geçerli alanlar, exit 0
./tan test --json testler/birim/Hesap_test.tan testler/birim/Liste_test.tan > /tmp/tantest5.log 2>&1
denetle "--json exit 0" "0" "$?"
grep -q '"gecti":true' /tmp/tantest5.log
denetle "--json gecti:true alanı" "0" "$?"
grep -q '"dosya":' /tmp/tantest5.log
denetle "--json dosya alanı" "0" "$?"

# 6. Var olmayan dosya: exit 2
./tan test yok_boyle.tan > /tmp/tantest6.log 2>&1
denetle "olmayan dosya exit 2" "2" "$?"

# 7. Tek dosya: exit 0 + GECTI raporu
./tan test testler/birim/Hesap_test.tan > /tmp/tantest7.log 2>&1
kod=$?
denetle "tek dosya exit 0" "0" "$kod"
grep -q "GECTI" /tmp/tantest7.log
denetle "tek dosya GECTI raporu" "0" "$?"

# 8. Başarısız testte program çıktısı raporlanır
printf 'yaz "on-arka-cikti"\nbekle(1 == 2)\n' > /tmp/tantest_ariza.tan
./tan test /tmp/tantest_ariza.tan > /tmp/tantest8.log 2>&1
denetle "çıktı raporlu arıza exit 1" "1" "$?"
grep -q "on-arka-cikti" /tmp/tantest8.log
denetle "başarısız test çıktısı gösterilir" "0" "$?"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
[ "$kaldi" -eq 0 ]
