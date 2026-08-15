#!/bin/bash
# TestSimgeler — "tan simgeler" komutunu sınar: işlev/kayıt/alan/metot/değişken
# envanteri, konum bilgisi, görünürlük, --json şeması ve deterministik sıra.
# Kullanım: ./TestSimgeler.sh
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
ISLER="/tmp/tan_simgeler_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan simgeler ==="

# 1. Kullanım ve hata çıkışları
"$TAN" simgeler > /tmp/sim_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan simgeler" /tmp/sim_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" simgeler --yardim > /tmp/sim_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" simgeler "$ISLER/yok.tan" > /tmp/sim_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"

# 2. İşlev, değişken ve kayıt (alan+metot) taraması
cat > "$ISLER/ornek.tan" <<'EOF'
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
kYas = 30
kAd = "Ali"
EOF
"$TAN" simgeler "$ISLER/ornek.tan" > /tmp/sim_ornek.log 2>&1
denetle "işlev bulunur" "1" "$(grep -c 'yeniKisi(ad, yas)' /tmp/sim_ornek.log)"
denetle "kayıt bulunur" "1" "$(grep -c 'kayıt.*Kisi' /tmp/sim_ornek.log)"
denetle "alan bulunur" "1" "$(grep -c 'alan.* ad' /tmp/sim_ornek.log)"
denetle "metot bulunur" "1" "$(grep -c 'metot.*selamla(bu)' /tmp/sim_ornek.log)"
denetle "değişken bulunur" "1" "$(grep -c 'değişken.*kYas' /tmp/sim_ornek.log)"
denetle "görünürlük işaretlenir (5 dışa simge)" "5" "$(grep -c '(dışa)' /tmp/sim_ornek.log)"

# 3. Konum doğruluğu (işlev adının satır:sütunu) ve --json şeması
"$TAN" simgeler --json "$ISLER/ornek.tan" > /tmp/sim_json.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/sim_json.log'))
d = v[0]
assert d['dosya'] == '$ISLER/ornek.tan'
def bul(tur, ad):
    for s in d['simgeler']:
        if s['tur'] == tur and s['ad'] == ad:
            return s
        for a in s.get('alanlar', []):
            if a['tur'] == tur and a['ad'] == ad:
                return a
        for m in s.get('metotlar', []):
            if m['tur'] == tur and m['ad'] == ad:
                return m
    return None
k = bul('kayit', 'Kisi')
assert k is not None and k['satir'] == 1 and k['sutun'] == 7
m = bul('metot', 'selamla')
assert m is not None and m['parametreler'] == ['bu'] and m['kayit'] == 'Kisi'
a = bul('alan', 'yas')
assert a is not None and a['kayit'] == 'Kisi'
i = bul('islev', 'yeniKisi')
assert i is not None and i['parametreler'] == ['ad', 'yas'] and i['satir'] == 8
v_ = bul('degisken', 'kAd')
assert v_ is not None and v_['satir'] == 12 and v_['gorunurluk'] == 'dışa'
print('OK')
" > /tmp/sim_verify.log 2>&1
denetle "--json şeması doğrulaması" "OK" "$(cat /tmp/sim_verify.log)"

# 4. Özet sayıları
python3 -c "
import json
v = json.load(open('/tmp/sim_json.log'))
o = v[0]['ozet']
assert o == {'toplam': 7, 'islev': 1, 'kayit': 1, 'metot': 1, 'alan': 2, 'degisken': 2}, o
print('OK')
" > /tmp/sim_ozet.log 2>&1
denetle "özet sayıları" "OK" "$(cat /tmp/sim_ozet.log)"

# 5. Boş dosya: simge yok, özet 0
printf '# yorum satırı\n' > "$ISLER/bos.tan"
"$TAN" simgeler --json "$ISLER/bos.tan" > /tmp/sim_bos.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/sim_bos.log'))
assert v[0]['simgeler'] == [] and v[0]['ozet']['toplam'] == 0
print('OK')
" > /tmp/sim_bosv.log 2>&1
denetle "boş dosya özet 0" "OK" "$(cat /tmp/sim_bosv.log)"

# 6. Çoklu dosya: dizi, giriş sırası korunur
"$TAN" simgeler --json "$ISLER/bos.tan" "$ISLER/ornek.tan" > /tmp/sim_multi.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/sim_multi.log'))
assert len(v) == 2
assert v[0]['dosya'] == '$ISLER/bos.tan' and v[1]['dosya'] == '$ISLER/ornek.tan'
print('OK')
" > /tmp/sim_multiv.log 2>&1
denetle "çoklu dosya sırası" "OK" "$(cat /tmp/sim_multiv.log)"

# 7. Sözdizimi hatası → exit 1
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" simgeler "$ISLER/bozuk.tan" > /tmp/sim_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"

# 8. Deterministik çıktı (iki çalıştırma aynı)
"$TAN" simgeler --json "$ISLER/ornek.tan" > /tmp/sim_d1.log 2>&1
"$TAN" simgeler --json "$ISLER/ornek.tan" > /tmp/sim_d2.log 2>&1
if cmp -s /tmp/sim_d1.log /tmp/sim_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
