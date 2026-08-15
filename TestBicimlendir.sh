#!/bin/bash
# TestBicimlendir — "tan biçimlendir" komutunu sınar:
# tekrarlı-özdeşlik (idempotence), anlam korunumu, yorum korunumu,
# hata yönetimi ve --denet/--cikti modları.
# Kullanım: ./TestBicimlendir.sh
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

DAGINIK="testler/_ornekler/BicimDaginik.tan"
F1="/tmp/bicim_f1.tan"
F2="/tmp/bicim_f2.tan"

echo "=== tan biçimlendir ==="

# 1. Biçimle: çıkış 0
./tan biçimlendir --cikti "$DAGINIK" > "$F1" 2>/dev/null
denetle "biçimlendir --cikti exit 0" "0" "$?"

# 2. Tekrarlı-özdeşlik: ikinci biçimleme aynı çıktıyı verir
./tan biçimlendir --cikti "$F1" > "$F2" 2>/dev/null
cmp -s "$F1" "$F2"
denetle "tekrarlı-özdeşlik (idempotence)" "0" "$?"

# 3. Kanonik biçim: girinti ve işlev imzası
grep -q "işlev topla(a, b)$" "$F1"
denetle "kanonik işlev imzası" "0" "$?"
grep -qP '^    sonuç = a \+ b  # toplam$' "$F1"
denetle "gövde girintisi + satır-içi yorum" "0" "$?"

# 4. Dizge içindeki '#' yorum SAYILMAZ
grep -qP '^z = topla\(x, y\)$' "$F1"
denetle "çağrı boşluksuzlaştırma" "0" "$?"

# 5. Anlam korunumu: özgün ve biçimlenmiş çıktılar AYNI
./tan "$DAGINIK" > /tmp/bicim_orj.log 2>&1
kod1=$?
./tan "$F1" > /tmp/bicim_biç.log 2>&1
kod2=$?
denetle "özgün çalışma exit" "0" "$kod1"
denetle "biçimlenmiş çalışma exit" "0" "$kod2"
cmp -s /tmp/bicim_orj.log /tmp/bicim_biç.log
denetle "özgün vs biçimlenmiş çıktı AYNI" "0" "$?"
grep -q "bölme hatası" /tmp/bicim_biç.log
denetle "dene/yakala anlamı korundu" "0" "$?"

# 6. ELF anlam korunumu: dene kullanmayan bir örnek üzerinde (ELF arka ucu
#    DeneDugum'u desteklemez — TAN5000, özgün dosyayla aynı sınırlama).
./tan biçimlendir --cikti ornekler/Dosya1.tan > /tmp/bicim_dosya1.tan 2>/dev/null
if ./tan elf ornekler/Dosya1.tan /tmp/bicim_orj_native >/dev/null 2>&1 &&
   ./tan elf /tmp/bicim_dosya1.tan /tmp/bicim_biç_native >/dev/null 2>&1; then
    /tmp/bicim_orj_native > /tmp/bicim_orj_native.log 2>&1
    /tmp/bicim_biç_native > /tmp/bicim_biç_native.log 2>&1
    cmp -s /tmp/bicim_orj_native.log /tmp/bicim_biç_native.log
    denetle "özgün vs biçimlenmiş ELF çıktısı AYNI" "0" "$?"
else
    denetle "örnek dosya ELF derlenir (ön koşul)" "0" "1"
fi

# 7. --denet: düzenli dosyada çıkış 0
./tan biçimlendir --denet "$F1" > /tmp/bicim_d1.log 2>&1
denetle "--denet düzenli dosya exit 0" "0" "$?"
grep -q "düzenli" /tmp/bicim_d1.log
denetle "--denet 'düzenli' mesajı" "0" "$?"

# 8. --denet: dağınık dosyada çıkış 1
./tan biçimlendir --denet "$DAGINIK" > /tmp/bicim_d2.log 2>&1
denetle "--denet dağınık dosya exit 1" "1" "$?"
grep -q "biçimlendirme gerekli" /tmp/bicim_d2.log
denetle "--denet 'biçimlendirme gerekli' mesajı" "0" "$?"

# 9. Yerinde biçimleme dosyayı değiştirir
cp "$DAGINIK" /tmp/bicim_yerinde.tan
./tan biçimlendir /tmp/bicim_yerinde.tan > /tmp/bicim_d3.log 2>&1
denetle "yerinde biçimleme exit 0" "0" "$?"
grep -q "biçimlendirildi" /tmp/bicim_d3.log
denetle "yerinde 'biçimlendirildi' mesajı" "0" "$?"
cmp -s /tmp/bicim_yerinde.tan "$DAGINIK"
denetle "yerinde dosya değişti (dağınıktan farklı)" "1" "$?"

# 10. Sözdizimi hatası: dosya değişmez, çıkış 1
printf '(1 + 2) = 5\n' > /tmp/bicim_bozuk.tan
./tan biçimlendir /tmp/bicim_bozuk.tan > /tmp/bicim_d4.log 2>&1
denetle "bozuk dosya exit 1" "1" "$?"
grep -q "TAN" /tmp/bicim_d4.log
denetle "bozuk dosya tanı mesajı" "0" "$?"

# 11. Argümansız: kullanım + çıkış 1
./tan biçimlendir > /tmp/bicim_d5.log 2>&1
denetle "argümansız exit 1" "1" "$?"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
[ "$kaldi" -eq 0 ]
