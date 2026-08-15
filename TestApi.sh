#!/bin/bash
# TestApi — "tan api" komutunu sınar: tür çıkarımlı işlev/metot imzaları,
# kayıt alan tipleri, üst-seviye değişken tipleri, --json ve determinizm.
# Kullanım: ./TestApi.sh
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
ISLER="/tmp/tan_api_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan api ==="

# 1. Kullanım ve hata çıkışları
"$TAN" api > /tmp/api_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan api" /tmp/api_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" api --yardim > /tmp/api_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" api "$ISLER/yok.tan" > /tmp/api_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"

# 2. Dönüş tipleri çıkarımı: tam sayı, metin, kayıt
cat > "$ISLER/islevler.tan" <<'EOF'
işlev ikiKat(x)
    döndür x * 2
son
işlev selamVer(ad)
    döndür "Merhaba " + ad
son
EOF
"$TAN" api "$ISLER/islevler.tan" > /tmp/api_islev.log 2>&1
denetle "tam sayı dönüşü çıkarılır" "1" "$(grep -c 'ikiKat(x: bilinmeyen) → tam sayı' /tmp/api_islev.log)"
denetle "metin dönüşü çıkarılır" "1" "$(grep -c 'selamVer(ad: bilinmeyen) → metin' /tmp/api_islev.log)"

# 3. Parametre tipleri çağrı yerinden öğrenilir
cat > "$ISLER/cagri.tan" <<'EOF'
işlev birleştir(a, b)
    döndür a + b
son
işlev goster(metin, sayi, oran)
    yaz birleştir(metin, "!")
    döndür sayi + oran
son
yaz birleştir("selam", " dünya")
yaz goster("x", 5, 2.5)
EOF
"$TAN" api --json "$ISLER/cagri.tan" > /tmp/api_cagri.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/api_cagri.log'))
d = v[0]
islev = {i['ad']: i for i in d['islevler']}
b = islev['birleştir']
# birleştir('selam', ' dünya') üst seviyede ve 'a+b' metin çıkarımıyla
assert b['donusTipi'] == 'metin', b
# a: 'selam' → metin; b: ' dünya' → metin
tipler = {p['ad']: p['tip'] for p in b['parametreler']}
assert tipler['a'] == 'metin' and tipler['b'] == 'metin', tipler
g = islev['goster']
gt = {p['ad']: p['tip'] for p in g['parametreler']}
# goster('x', 5, 2.5): metin, tam sayı, ondalık
assert gt['metin'] == 'metin' and gt['sayi'] == 'bilinmeyen' and gt['oran'] == 'ondalık', gt
print('OK')
" > /tmp/api_cagriv.log 2>&1
denetle "parametre tipi öğrenimi" "OK" "$(cat /tmp/api_cagriv.log)"

# 4. Kayıt: alan tipleri, metot alıcısı ve metot dönüş tipi
cat > "$ISLER/kayit.tan" <<'EOF'
kayıt Kisi
    ad
    yas
    işlev selamla(bu)
        döndür "Merhaba " + bu.ad
    son
son
işlev yeniKisi(ad, yas)
    döndür Kisi{ad: ad, yas: yas}
son
EOF
"$TAN" api --json "$ISLER/kayit.tan" > /tmp/api_kayit.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/api_kayit.log'))
d = v[0]
assert len(d['kayitlar']) == 1
k = d['kayitlar'][0]
assert k['ad'] == 'Kisi'
assert k['alanlar'] == [{'ad': 'ad', 'tip': 'tam sayı'}, {'ad': 'yas', 'tip': 'tam sayı'}], k['alanlar']
m = k['metotlar'][0]
assert m['ad'] == 'selamla'
assert m['parametreler'] == [{'ad': 'bu', 'tip': 'kayıt<Kisi>'}], m['parametreler']
assert m['donusTipi'] == 'metin', m['donusTipi']
print('OK')
" > /tmp/api_kayitv.log 2>&1
denetle "kayıt API'si" "OK" "$(cat /tmp/api_kayitv.log)"

# 5. Üst-seviye değişken tipleri
printf 'kMetin = "merhaba"\nkListe = [1, 2, 3]\n' > "$ISLER/degisken.tan"
"$TAN" api --json "$ISLER/degisken.tan" > /tmp/api_degisken.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/api_degisken.log'))
d = v[0]
deg = {x['ad']: x['tip'] for x in d['degiskenler']}
assert deg['kMetin'] == 'metin', deg
assert deg['kListe'] == 'liste<tam sayı>', deg
print('OK')
" > /tmp/api_degiskenv.log 2>&1
denetle "değişken tipi çıkarımı" "OK" "$(cat /tmp/api_degiskenv.log)"

# 6. Boş dosya: boş listeler, geçerli JSON
printf '# yorum\n' > "$ISLER/bos.tan"
"$TAN" api --json "$ISLER/bos.tan" > /tmp/api_bos.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/api_bos.log'))
d = v[0]
assert d['islevler'] == [] and d['kayitlar'] == [] and d['degiskenler'] == []
print('OK')
" > /tmp/api_bosv.log 2>&1
denetle "boş dosya API'si" "OK" "$(cat /tmp/api_bosv.log)"

# 7. Sözdizimi hatası → exit 1
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" api "$ISLER/bozuk.tan" > /tmp/api_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"

# 8. Deterministik çıktı
"$TAN" api --json "$ISLER/kayit.tan" > /tmp/api_d1.log 2>&1
"$TAN" api --json "$ISLER/kayit.tan" > /tmp/api_d2.log 2>&1
if cmp -s /tmp/api_d1.log /tmp/api_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
