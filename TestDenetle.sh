#!/bin/bash
# TestDenetle — "tan denetle" komutunu sınar: 10 kuralın tümü, temiz
# dosyalar, çıkış kodları, --json ve kütüphane/program ayrımı.
# Kullanım: ./TestDenetle.sh
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

KIRLI="testler/_ornekler/DenetKirli.tan"

echo "=== tan denetle ==="

# 1. Kirli fikstür: 10 kuralın TÜMÜ tetiklenir, özet doğru
./tan denetle "$KIRLI" > /tmp/denet_fix.log 2>&1
denetle "kirli dosya exit 1" "1" "$?"
for kod in TAN9101 TAN9102 TAN9103 TAN9104 TAN9105 TAN9106 TAN9107 TAN9108 TAN9109 TAN9110; do
    grep -q "$kod" /tmp/denet_fix.log
    denetle "kural $kod tetiklenir" "0" "$?"
done
grep -q "10 uyarı, 1 bilgi" /tmp/denet_fix.log
denetle "özet 10 uyarı, 1 bilgi" "0" "$?"
grep -q "^${KIRLI}:3: uyarı TAN9101" /tmp/denet_fix.log
denetle "GOLGE_YERLESIK satırı doğru (3)" "0" "$?"
grep -qE "liste|metin|kod" /tmp/denet_fix.log
denetle "uyarı satır bağlamı (kaynak satırı) görünür" "0" "$?"

# 2. Temiz dosya: 0/0, çıkış 0
./tan denetle ornekler/Dosya1.tan > /tmp/denet_clean.log 2>&1
denetle "temiz dosya exit 0" "0" "$?"
grep -q "0 uyarı, 0 bilgi" /tmp/denet_clean.log
denetle "temiz özet 0/0" "0" "$?"

# 3. --json: geçerli JSON dizisi, tüm raporlar içinde
./tan denetle --json "$KIRLI" > /tmp/denet.json 2>&1
denetle "--json exit 1 (sorun var)" "1" "$?"
if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import json
r = json.load(open('/tmp/denet.json'))
assert isinstance(r, list) and len(r) >= 11, len(r)
kodlar = {d['kod'] for d in r}
assert 'TAN9101' in kodlar and 'TAN9110' in kodlar, kodlar
assert all('onem' in d and 'mesaj' in d and 'satir' in d for d in r)
print('ok')
" > /tmp/denet_json_chk.log 2>&1
    denetle "--json geçerli JSON + tüm alanlar" "0" "$?"
else
    head -c1 /tmp/denet.json | grep -q "\["
    denetle "--json '[' ile başlar (python3 yok)" "0" "$?"
fi

# 4. Sözdizimi hatası: çıkış 1, stderr'de TAN tanısı
printf '(1 + 2) = 5\n' > /tmp/denet_bozuk.tan
./tan denetle /tmp/denet_bozuk.tan > /tmp/denet_boz.log 2>&1
denetle "bozuk dosya exit 1" "1" "$?"
grep -q "TAN2001" /tmp/denet_boz.log
denetle "bozuk dosya TAN2001 tanısı" "0" "$?"

# 5. Argümansız: kullanım + çıkış 2
./tan denetle > /tmp/denet_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan denetle" /tmp/denet_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"

# 6. Döngü değişkeni: kullanılmayan bildirilir, kullanılan bildirilmez
printf 'i = 0\niken i < 3\n\ther j [1, 2]\n\t\ti = i + 1\n\tson\nson\nyaz i\n' > /tmp/denet_her.tan
./tan denetle /tmp/denet_her.tan > /tmp/denet_her.log 2>&1
grep -q "TAN9102.*'j'" /tmp/denet_her.log
denetle "kullanılmayan döngü değişkeni bildirilir" "0" "$?"
printf 'i = 0\niken i < 3\n\ther j [1, 2]\n\t\ti = i + j\n\tson\nson\nyaz i\n' > /tmp/denet_her2.tan
./tan denetle /tmp/denet_her2.tan > /tmp/denet_her2.log 2>&1
grep -q "TAN9102" /tmp/denet_her2.log
denetle "kullanılan döngü değişkeni bildirilmez" "1" "$?"

# 7. TEK_KULLANIM: programda ölü işlev bildirilir, kütüphanede bildirilmez
printf 'işlev ölü(x)\n\tdöndür x\nson\nişlev canli(y)\n\tdöndür y + 1\nson\nyaz canli(1)\n' > /tmp/denet_tek.tan
./tan denetle /tmp/denet_tek.tan > /tmp/denet_tek.log 2>&1
grep -q "TAN9110.*'ölü'" /tmp/denet_tek.log
denetle "programda ölü işlev bildirilir" "0" "$?"
./tan denetle kutuphane/Bicim.tan > /tmp/denet_lib.log 2>&1
grep -q "TAN9110" /tmp/denet_lib.log
denetle "kütüphanede TEK_KULLANIM bildirilmez" "1" "$?"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
[ "$kaldi" -eq 0 ]
