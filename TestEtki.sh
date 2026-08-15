#!/bin/bash
# TestEtki — "tan etki" komutunu sınar: statik çağrı grafiği, işlev/metot
# sorguları, alıcı çözümlemesi (kayıt sabiti ve "bu"), bulunamayan sembol,
# --json şeması ve deterministik sıra.
# Kullanım: ./TestEtki.sh
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
ISLER="/tmp/tan_etki_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan etki ==="

# 1. Kullanım ve hata çıkışları
"$TAN" etki > /tmp/etki_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan etki" /tmp/etki_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" etki --yardim > /tmp/etki_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" etki "$ISLER/yok.tan" f > /tmp/etki_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"
"$TAN" etki "$ISLER/yok.tan" > /tmp/etki_yok2.log 2>&1
denetle "tek argüman exit 2" "2" "$?"

# 2. Fikstür: kayıt metotları ("bu" alıcı çözümü) + işlevler + program
cat > "$ISLER/etki.tan" <<'EOF'
kayıt Hesap
    ad
    bakiye
    işlev yatir(bu, miktar)
        bu.bakiye = bu.bakiye + miktar
        döndür bu
    son
    işlev ciftYatir(bu, miktar)
        bu.yatir(miktar)
        bu.yatir(miktar)
    son
    işlev rapor(bu)
        yaz(bu.ad)
    son
son
işlev yeniHesap(ad)
    döndür Hesap{ad: ad, bakiye: 0}
son
işlev cevir(kaynak, miktar)
    kaynak.yatir(miktar)
son
işlev selamla()
    Hesap{ad: "x", bakiye: 1}.rapor()
son
h = yeniHesap("ali")
h.yatir(5)
h.rapor()
EOF

# 3. İşlev sorgusu: çağıranlar (program dahil) ve çağırdıkları
"$TAN" etki "$ISLER/etki.tan" yeniHesap > /tmp/etki_f.log 2>&1
denetle "işlev sorgusu exit 0" "0" "$?"
denetle "çağıran program" "1" "$(grep -c '<ana> (program)' /tmp/etki_f.log)"
denetle "çağrılan kayıt" "1" "$(grep -c 'Hesap (kayit)' /tmp/etki_f.log)"
denetle "konum bilgisi" "1" "$(grep -c 'sembol: yeniHesap (işlev)' /tmp/etki_f.log)"

# 4. Çözümlenmiş kayıt.metot sorgusu: yalnız "bu" alıcılı çağrılar
"$TAN" etki --json "$ISLER/etki.tan" Hesap.yatir > /tmp/etki_m.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/etki_m.log'))
d = v[0]
assert d['sembol'] == 'Hesap.yatir' and d['tur'] == 'metot'
assert d['kayit'] == 'Hesap' and d['satir'] > 0
cagiranlar = [c['ad'] for c in d['cagiranlar']]
assert cagiranlar == ['Hesap.ciftYatir'], cagiranlar
assert d['cagirdikleri'] == []
print('OK')
" > /tmp/etki_mv.log 2>&1
denetle "kayıt.metot çözümlü şema" "OK" "$(cat /tmp/etki_mv.log)"

# 5. Yalın metot adı: bilinmeyen alıcılı çağrılar da eşleşir
"$TAN" etki "$ISLER/etki.tan" yatir > /tmp/etki_n.log 2>&1
denetle "yalın metot çağıran cevir" "1" "$(grep -c 'cevir (islev)' /tmp/etki_n.log)"
denetle "yalın metot çağıran ciftYatir" "1" "$(grep -c 'Hesap.ciftYatir (metot)' /tmp/etki_n.log)"
denetle "yalın metot çağıran program" "1" "$(grep -c '<ana> (program)' /tmp/etki_n.log)"

# 6. Kayıt sabiti alıcılı metot çağrısı (Hesap{...}.rapor): çözümlenmiş
#    sorguda yalnız kayıt sabiti alıcılı çağrı eşleşir; değişken alıcı (<ana>
#    programındaki h.rapor) bilinmeyen kayıttan dolayı eşleşmez.
"$TAN" etki --json "$ISLER/etki.tan" Hesap.rapor > /tmp/etki_r.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/etki_r.log'))
d = v[0]
cagiranlar = [c['ad'] for c in d['cagiranlar']]
assert 'selamla' in cagiranlar, cagiranlar
assert '<ana>' not in cagiranlar, cagiranlar
print('OK')
" > /tmp/etki_rv.log 2>&1
denetle "kayıt sabiti alıcı çözümü" "OK" "$(cat /tmp/etki_rv.log)"

# 7. Bulunamayan sembol → exit 1
"$TAN" etki "$ISLER/etki.tan" yokSembol > /tmp/etki_nf.log 2>&1
denetle "bulunamayan sembol exit 1" "1" "$?"
grep -q "bulunamadı" /tmp/etki_nf.log
denetle "bulunamadı mesajı" "0" "$?"

# 8. Sözdizimi hatası → exit 1
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" etki "$ISLER/bozuk.tan" f > /tmp/etki_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"

# 9. Deterministik çıktı
"$TAN" etki --json "$ISLER/etki.tan" yatir > /tmp/etki_d1.log 2>&1
"$TAN" etki --json "$ISLER/etki.tan" yatir > /tmp/etki_d2.log 2>&1
if cmp -s /tmp/etki_d1.log /tmp/etki_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
