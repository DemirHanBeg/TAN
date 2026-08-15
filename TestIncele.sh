#!/bin/bash
# TestIncele — "tan incele" komutunu sınar: ölçüm profili (satır/bayt,
# sembol sayıları, AST büyüklüğü, karmaşıklık, içe al listesi, dizin yinelemesi).
# Kullanım: ./TestIncele.sh
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
ISLER="/tmp/tan_incele_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan incele ==="

# 1. Kullanım ve hata çıkışları
"$TAN" incele > /tmp/inc_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan incele" /tmp/inc_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" incele --yardim > /tmp/inc_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" incele "$ISLER/yok.tan" > /tmp/inc_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"

# 2. Metin modu: satır/bayt ve sembol sayıları
cat > "$ISLER/ornek.tan" <<'EOF'
kayıt Hesap
    ad
    bakiye
    işlev yatir(bu, miktar)
        bu.bakiye = bu.bakiye + miktar
    son
son
işlev yeniHesap(ad)
    döndür Hesap{ad: ad, bakiye: 0}
son
h = yeniHesap("ali")
EOF
"$TAN" incele "$ISLER/ornek.tan" > /tmp/inc_ornek.log 2>&1
denetle "metin modu başlık" "1" "$(grep -c 'incele:.*ornek.tan' /tmp/inc_ornek.log)"
denetle "satır sayısı" "1" "$(grep -c 'satır: 11' /tmp/inc_ornek.log)"
denetle "işlev sayısı" "1" "$(grep -c 'işlev: 1' /tmp/inc_ornek.log)"
denetle "kayıt/metot/alan sayısı" "1" "$(grep -c 'kayıt: 1  metot: 1  alan: 2' /tmp/inc_ornek.log)"
denetle "değişken sayısı" "1" "$(grep -c 'değişken: 1' /tmp/inc_ornek.log)"
denetle "düğüm sayısı satırı" "1" "$(grep -c 'düğüm: ' /tmp/inc_ornek.log)"
denetle "en büyük gövde" "1" "$(grep -c 'en büyük gövde: Hesap.yatir' /tmp/inc_ornek.log)"

# 3. --json şeması
"$TAN" incele --json "$ISLER/ornek.tan" > /tmp/inc_json.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/inc_json.log'))
d = v[0]
assert d['dosya'] == '$ISLER/ornek.tan', d['dosya']
assert d['satirlar'] == 11 and d['bayt'] > 0
assert d['iceAl'] == [], d['iceAl']
o = d['ozet']
assert o['islev'] == 1 and o['kayit'] == 1 and o['metot'] == 1 and o['alan'] == 2 and o['degisken'] == 1, o
assert d['dugumSayisi'] > 0 and d['maksDerinlik'] > 0 and d['karmasiklik'] == 0
assert d['enBuyukGovde']['ad'] == 'Hesap.yatir' and d['enBuyukGovde']['dugumSayisi'] > 0
print('OK')
" > /tmp/inc_verify.log 2>&1
denetle "--json şeması" "OK" "$(cat /tmp/inc_verify.log)"

# 4. Karmaşıklık: eğer + her + iken = 3 karar noktası
cat > "$ISLER/karmasik.tan" <<'EOF'
işlev kar(kosul, liste)
    eğer kosul ise
        döndür 1
    son
    her x liste
        iken doğru ise
            yaz(x)
        son
    son
    döndür 0
son
EOF
"$TAN" incele --json "$ISLER/karmasik.tan" > /tmp/inc_karm.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/inc_karm.log'))
assert v[0]['karmasiklik'] == 3, v[0]['karmasiklik']
print('OK')
" > /tmp/inc_karmv.log 2>&1
denetle "karmaşıklık eğer+her+iken = 3" "OK" "$(cat /tmp/inc_karmv.log)"

# 5. içe al listesi
cat > "$ISLER/iceal.tan" <<'EOF'
içe al "kutuphane/Matematik.tan"
yaz(3)
EOF
"$TAN" incele --json "$ISLER/iceal.tan" > /tmp/inc_iceal.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/inc_iceal.log'))
d = v[0]
assert len(d['iceAl']) == 1, d['iceAl']
assert d['iceAl'][0]['dosya'] == 'kutuphane/Matematik.tan'
assert d['iceAl'][0]['satir'] == 1
print('OK')
" > /tmp/inc_icealv.log 2>&1
denetle "içe al listesi" "OK" "$(cat /tmp/inc_icealv.log)"

# 6. Dizin girişi: yinelenir, sıralı
mkdir -p "$ISLER/dizin"
printf 'yaz("b")\n' > "$ISLER/dizin/b.tan"
cp "$ISLER/ornek.tan" "$ISLER/dizin/a.tan"
"$TAN" incele --json "$ISLER/dizin" > /tmp/inc_dizin.log 2>&1
python3 -c "
import json, os
v = json.load(open('/tmp/inc_dizin.log'))
adlar = [os.path.basename(d['dosya']) for d in v]
assert adlar == ['a.tan', 'b.tan'], adlar
print('OK')
" > /tmp/inc_dizinv.log 2>&1
denetle "dizin yinelemesi sıralı" "OK" "$(cat /tmp/inc_dizinv.log)"

# 7. Sözdizimi hatası → exit 1
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" incele "$ISLER/bozuk.tan" > /tmp/inc_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"

# 8. Deterministik çıktı
"$TAN" incele --json "$ISLER/ornek.tan" > /tmp/inc_d1.log 2>&1
"$TAN" incele --json "$ISLER/ornek.tan" > /tmp/inc_d2.log 2>&1
if cmp -s /tmp/inc_d1.log /tmp/inc_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
