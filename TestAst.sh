#!/bin/bash
# TestAst — "tan ast" komutunu sınar: ayrıştırma ağacı metin/JSON çıktısı,
# düğüm şeması, sözdizimi hatası ve deterministik sıra.
# Kullanım: ./TestAst.sh
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
ISLER="/tmp/tan_ast_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER"

echo "=== tan ast ==="

# 1. Kullanım ve hata çıkışları
"$TAN" ast > /tmp/ast_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan ast" /tmp/ast_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" ast --yardim > /tmp/ast_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" ast "$ISLER/yok.tan" > /tmp/ast_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"

# 2. Metin modu: yaz/atama/ikili/sayi/değişken etiketleri
cat > "$ISLER/temiz.tan" <<'EOF'
yaz("merhaba")
toplam = 2 + 3
yaz(toplam)
EOF
"$TAN" ast "$ISLER/temiz.tan" > /tmp/ast_temiz.log 2>&1
denetle "metin modu başlık" "1" "$(grep -c 'AST:.*temiz.tan' /tmp/ast_temiz.log)"
denetle "yaz düğümü" "2" "$(grep -c '^  yaz' /tmp/ast_temiz.log)"
denetle "metin sabiti" "1" "$(grep -c 'metin "merhaba"' /tmp/ast_temiz.log)"
denetle "atama düğümü" "1" "$(grep -c 'atama toplam =' /tmp/ast_temiz.log)"
denetle "ikili düğüm" "1" "$(grep -c 'ikili +' /tmp/ast_temiz.log)"
denetle "sayi sabiti" "2" "$(grep -c 'sayi 2$\|sayi 3$' /tmp/ast_temiz.log)"
denetle "değişken düğümü" "1" "$(grep -c 'degisken toplam' /tmp/ast_temiz.log)"

# 3. Kayıt ve işlev etiketleri metin modunda
cat > "$ISLER/kayit.tan" <<'EOF'
kayıt Kisi
    ad
    işlev selamla(bu)
        döndür "Merhaba"
    son
son
işlev toplam(a, b)
    döndür a + b
son
EOF
"$TAN" ast "$ISLER/kayit.tan" > /tmp/ast_kayit.log 2>&1
denetle "kayıt düğümü" "1" "$(grep -c 'kayitTanim Kisi' /tmp/ast_kayit.log)"
denetle "işlev etiketi" "1" "$(grep -c 'islev toplam(a, b)' /tmp/ast_kayit.log)"
denetle "metot etiketi" "1" "$(grep -c 'islev selamla(bu)' /tmp/ast_kayit.log)"
denetle "döndür düğümü" "2" "$(grep -c '^ *dondur' /tmp/ast_kayit.log)"

# 4. --json şeması: düğüm türleri, iç içe ikili, parametreler boş dizi
"$TAN" ast --json "$ISLER/temiz.tan" > /tmp/ast_json.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/ast_json.log'))
d = v[0]
assert d['dosya'] == '$ISLER/temiz.tan', d['dosya']
agac = d['agac']
assert agac[0]['tur'] == 'yaz' and agac[0]['deger']['tur'] == 'metin'
ikili = agac[1]['deger']
assert ikili['tur'] == 'ikili' and ikili['islec'] == '+', ikili
assert ikili['sol']['tur'] == 'sayi' and ikili['sol']['tam'] == 2
assert ikili['sag']['tur'] == 'sayi' and ikili['satir'] == 2 and ikili['sutun'] == 12
assert agac[2]['deger']['tur'] == 'degisken' and agac[2]['deger']['ad'] == 'toplam'
print('OK')
" > /tmp/ast_verify.log 2>&1
denetle "--json düğüm şeması" "OK" "$(cat /tmp/ast_verify.log)"

# 5. Kayıt/işlev JSON: alanlar, metotlar, parametreler (null değil, [])
"$TAN" ast --json "$ISLER/kayit.tan" > /tmp/ast_kayitj.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/ast_kayitj.log'))
agac = v[0]['agac']
k = agac[0]
assert k['tur'] == 'kayitTanim' and k['ad'] == 'Kisi', k
assert k['alanlar'] == ['ad'], k['alanlar']
assert len(k['metotlar']) == 1 and k['metotlar'][0]['ad'] == 'selamla'
assert k['metotlar'][0]['parametreler'] == ['bu']
i = agac[1]
assert i['tur'] == 'islev' and i['ad'] == 'toplam'
assert i['parametreler'] == ['a', 'b'] and len(i['govde']) == 1
assert i['govde'][0]['tur'] == 'dondur' and i['govde'][0]['deger']['islec'] == '+'
print('OK')
" > /tmp/ast_kayitjv.log 2>&1
denetle "--json kayıt/işlev şeması" "OK" "$(cat /tmp/ast_kayitjv.log)"

# 6. Sözdizimi hatası → exit 1, JSON çıktısı boş dizi
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" ast "$ISLER/bozuk.tan" > /tmp/ast_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"
grep -q "TAN2001" /tmp/ast_boz.log
denetle "hata kodunu gösterir" "0" "$?"
"$TAN" ast --json "$ISLER/bozuk.tan" > /tmp/ast_bozj.log 2>&1
denetle "bozuk dosyada --json exit 1" "1" "$?"

# 7. Çoklu dosya: dizi giriş sırası korunur
"$TAN" ast --json "$ISLER/temiz.tan" "$ISLER/kayit.tan" > /tmp/ast_multi.log 2>&1
python3 -c "
import json
v = json.load(open('/tmp/ast_multi.log'))
assert len(v) == 2
assert v[0]['dosya'] == '$ISLER/temiz.tan' and v[1]['dosya'] == '$ISLER/kayit.tan'
print('OK')
" > /tmp/ast_multiv.log 2>&1
denetle "çoklu dosya sırası" "OK" "$(cat /tmp/ast_multiv.log)"

# 8. Deterministik çıktı
"$TAN" ast --json "$ISLER/kayit.tan" > /tmp/ast_d1.log 2>&1
"$TAN" ast --json "$ISLER/kayit.tan" > /tmp/ast_d2.log 2>&1
if cmp -s /tmp/ast_d1.log /tmp/ast_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
