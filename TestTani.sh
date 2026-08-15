#!/bin/bash
# TestTani — "tan tanı" komutunu sınar: kural motoru üzerinde tanı raporları,
# ayrıştırma hatası kaydı, özet sayıları, --json şeması, dizin yinelemesi.
# Kullanım: ./TestTani.sh
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

TAN="$(pwd)/tan"
ISLER="/tmp/tan_tani_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan tanı ==="

# 1. Kullanım ve hata çıkışları
"$TAN" tanı > /tmp/tani_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan tanı" /tmp/tani_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" tanı --yardim > /tmp/tani_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" tanı "$ISLER/yok.tan" > /tmp/tani_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"

# 2. Temiz dosya: exit 0, sıfır özet
cat > "$ISLER/temiz.tan" <<'EOF'
yaz("merhaba")
toplam = 2 + 3
yaz(toplam)
EOF
"$TAN" tanı "$ISLER/temiz.tan" > /tmp/tani_temiz.log 2>&1
denetle "temiz dosya exit 0" "0" "$?"
denetle "sıfır özet" "1" "$(grep -c '0 hata, 0 uyarı, 0 bilgi' /tmp/tani_temiz.log)"

# 3. Uyarılı dosya: exit 1, kurallar tetiklenir, özet uyarı sayar
cat > "$ISLER/uyarili.tan" <<'EOF'
liste = [1, 2, 3]
x = 5
s = 99999999999999999999

işlev yalniz(c)
	döndür c * 2
son

deger = 0.1
esitMi = deger == 0.5

eğer doğru ise
	yaz liste
değilse
	yaz x
son

sonuc = yalniz(3) + yalniz(4)
yaz sonuc
EOF
"$TAN" tanı "$ISLER/uyarili.tan" > /tmp/tani_uyarili.log 2>&1
denetle "uyarılı dosya exit 1" "1" "$?"
denetle "taşma uyarısı TAN9106" "1" "$(grep -c 'TAN9106' /tmp/tani_uyarili.log)"
denetle "gölgeleme uyarısı TAN9101" "1" "$(grep -c 'TAN9101' /tmp/tani_uyarili.log)"
denetle "ondalık == uyarısı TAN9109" "1" "$(grep -c 'TAN9109' /tmp/tani_uyarili.log)"
denetle "sabit koşul uyarısı TAN9105" "1" "$(grep -c 'TAN9105' /tmp/tani_uyarili.log)"
denetle "kullanılmayan değişken uyarısı (TAN9102)" "2" "$(grep -c 'TAN9102' /tmp/tani_uyarili.log)"
denetle "uyarı özeti 6" "1" "$(grep -c '0 hata, 6 uyarı, 0 bilgi' /tmp/tani_uyarili.log)"

# 4. Sözdizimi hatası: exit 1, tanı kaydı TAN2001
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" tanı "$ISLER/bozuk.tan" > /tmp/tani_bozuk.log 2>&1
denetle "bozuk dosya exit 1" "1" "$?"
denetle "bozuk dosya hata özeti" "1" "$(grep -c '1 hata, 0 uyarı, 0 bilgi' /tmp/tani_bozuk.log)"

# 5. --json şeması: hata/raporlar/ozet
"$TAN" tanı --json "$ISLER/uyarili.tan" > /tmp/tani_json.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/tani_json.log'))
d = v[0]
assert d['dosya'] == '$ISLER/uyarili.tan', d['dosya']
assert 'hata' not in d, d.get('hata')
assert isinstance(d['raporlar'], list) and len(d['raporlar']) > 0
assert d['ozet'] == {'hata': 0, 'uyari': 6, 'bilgi': 0}, d['ozet']
kodlar = [r['kod'] for r in d['raporlar']]
assert 'TAN9106' in kodlar and 'TAN9101' in kodlar
for r in d['raporlar']:
    assert r['onem'] == 'uyarı' and 'dosya' in r and 'mesaj' in r
print('OK')
" > /tmp/tani_verify.log 2>&1
denetle "--json uyarılı şema" "OK" "$(cat /tmp/tani_verify.log)"

# 6. --json bozuk: hata alanı Diyagnostik şemasında
"$TAN" tanı --json "$ISLER/bozuk.tan" > /tmp/tani_bozj.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/tani_bozj.log'))
d = v[0]
h = d['hata']
assert h is not None, d
assert h['kod'] == 'TAN2001' and h['onem'] == 'hata'
assert h['dosya'] == '$ISLER/bozuk.tan'
assert h['satir'] == 1 and h['sutun'] == 9
assert d['raporlar'] == [] and d['ozet']['hata'] == 1
print('OK')
" > /tmp/tani_bozjv.log 2>&1
denetle "--json bozuk şema" "OK" "$(cat /tmp/tani_bozjv.log)"

# 7. Dizin girişi: yinelenir, hata/uyarılı dosyalar exit 1'i tetikler
mkdir -p "$ISLER/dizin"
cp "$ISLER/temiz.tan" "$ISLER/dizin/temiz.tan"
cp "$ISLER/uyarili.tan" "$ISLER/dizin/uyarili.tan"
cp "$ISLER/bozuk.tan" "$ISLER/dizin/bozuk.tan"
"$TAN" tanı --json "$ISLER/dizin" > /tmp/tani_dizin.log 2>&1
denetle "dizin exit 1 (hata/uyarı var)" "1" "$?"
python3 -c "
import json, os
v = json.load(open('/tmp/tani_dizin.log'))
adlar = sorted(os.path.basename(d['dosya']) for d in v)
assert adlar == ['bozuk.tan', 'temiz.tan', 'uyarili.tan'], adlar
ozet = {os.path.basename(d['dosya']): d['ozet'] for d in v}
assert ozet['temiz.tan'] == {'hata': 0, 'uyari': 0, 'bilgi': 0}
assert ozet['bozuk.tan']['hata'] == 1 and ozet['uyarili.tan']['uyari'] == 6
print('OK')
" > /tmp/tani_dizinv.log 2>&1
denetle "dizin JSON sıralı ve özetler" "OK" "$(cat /tmp/tani_dizinv.log)"

# 8. Deterministik çıktı
"$TAN" tanı --json "$ISLER/uyarili.tan" > /tmp/tani_d1.log 2>&1
"$TAN" tanı --json "$ISLER/uyarili.tan" > /tmp/tani_d2.log 2>&1
if cmp -s /tmp/tani_d1.log /tmp/tani_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
