#!/bin/bash
# TestBagimlilik — "tan bagimlilik" komutunu sınar: "içe al" çözümlemesi,
# transitif grafik, ters bağımlılıklar, döngü tespiti, dizin girişi ve --json.
# Kullanım: ./TestBagimlilik.sh
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
ISLER="/tmp/tan_bagimlilik_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER/alt"

echo "=== tan bagimlilik ==="

# 1. Kullanım ve hata çıkışları
"$TAN" bagimlilik > /tmp/bag_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan bagimlilik" /tmp/bag_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" bagimlilik --yardim > /tmp/bag_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
"$TAN" bagimlilik "$ISLER/yok.tan" > /tmp/bag_yok.log 2>&1
denetle "olmayan giriş exit 1" "1" "$?"

# 2. Boş dosya (içe al yok)
printf 'yaz(1)\n' > "$ISLER/yalniz.tan"
"$TAN" bagimlilik "$ISLER/yalniz.tan" > /tmp/bag_yalniz.log 2>&1
denetle "içe al yok bildirilir" "1" "$(grep -c 'içe al yok' /tmp/bag_yalniz.log)"
denetle "ters ve döngü için yok (2)" "2" "$(grep -c '^  yok$' /tmp/bag_yalniz.log)"

# 3. Tek içe al: çözülür, ters liste hedefi gösterir
printf 'işlev kare(x)\n\tdöndür x * x\nson\n' > "$ISLER/islem.tan"
printf 'içe al "islem"\nyaz kare(3)\n' > "$ISLER/ana.tan"
"$TAN" bagimlilik "$ISLER/ana.tan" > /tmp/bag_tek.log 2>&1
denetle "içe al çözülür" "1" "$(grep -c '→.*islem.tan' /tmp/bag_tek.log)"
denetle "ters listede islem.tan" "1" "$(grep -c 'islem.tan  ←  .*ana.tan' /tmp/bag_tek.log)"

# 4. Transitif: ust → orta → taban (üçü de grafikte)
printf 'işlev tabanFonksiyonu()\n\tdöndür 1\nson\n' > "$ISLER/taban.tan"
printf 'içe al "taban"\nişlev ortaFonksiyonu()\n\tdöndür tabanFonksiyonu() + 1\nson\n' > "$ISLER/orta.tan"
printf 'içe al "orta"\nyaz ortaFonksiyonu()\n' > "$ISLER/ust.tan"
"$TAN" bagimlilik --json "$ISLER/ust.tan" > /tmp/bag_trans.log 2>&1
python3 -c "
import json, os
v = json.load(open('/tmp/bag_trans.log'))
adlar = [os.path.basename(d['dosya']) for d in v['dosyalar']]
assert sorted(adlar) == ['orta.tan', 'taban.tan', 'ust.tan'], adlar
print('OK')
" > /tmp/bag_transv.log 2>&1
denetle "transitif grafik (5 dosya)" "OK" "$(cat /tmp/bag_transv.log)"
python3 -c "
import json, os
v = json.load(open('/tmp/bag_trans.log'))
ters = {os.path.basename(t['dosya']): [os.path.basename(k) for k in t['kullananlar']] for t in v['ters']}
assert 'taban.tan' in ters and 'orta.tan' in ters['taban.tan'], ters
assert 'orta.tan' in ters and 'ust.tan' in ters['orta.tan'], ters
print('OK')
" > /tmp/bag_tersv.log 2>&1
denetle "ters bağımlılık doğru" "OK" "$(cat /tmp/bag_tersv.log)"

# 5. Döngü tespiti: a ↔ b
printf 'içe al "b"\nyaz(1)\n' > "$ISLER/a.tan"
printf 'içe al "a"\nyaz(2)\n' > "$ISLER/b.tan"
"$TAN" bagimlilik --json "$ISLER/a.tan" > /tmp/bag_dongu.log 2>&1
python3 -c "
import json, os
v = json.load(open('/tmp/bag_dongu.log'))
donguler = [[os.path.basename(x) for x in d] for d in v['donguler']]
assert any(sorted(d) == ['a.tan', 'b.tan'] for d in donguler), donguler
print('OK')
" > /tmp/bag_donguv.log 2>&1
denetle "a-b döngüsü bulunur" "OK" "$(cat /tmp/bag_donguv.log)"

# 6. Çözülemeyen içe al: BULUNAMADI, çıkış 0
printf 'içe al "YokBoyleBirModul"\nyaz(1)\n' > "$ISLER/belirsiz.tan"
"$TAN" bagimlilik "$ISLER/belirsiz.tan" > /tmp/bag_bulun.log 2>&1
denetle "bulunamayan bildirilir" "1" "$(grep -c 'BULUNAMADI' /tmp/bag_bulun.log)"
denetle "bulunamayan exit 0 (bulgu)" "0" "$?"

# 7. Dizin girişi: yinelenir, tüm .tan dosyaları tarama
"$TAN" bagimlilik --json "$ISLER" > /tmp/bag_dizin.log 2>&1
python3 -c "
import json, os
v = json.load(open('/tmp/bag_dizin.log'))
adlar = sorted(os.path.basename(d['dosya']) for d in v['dosyalar'])
# dizindeki tüm .tan dosyaları grafikte olmalı (hatalı dosya yok)
beklenen = sorted(f for f in os.listdir('$ISLER') if f.endswith('.tan'))
assert adlar == beklenen, (adlar, beklenen)
print('OK')
" > /tmp/bag_dizinv.log 2>&1
denetle "dizin taraması tüm dosyaları kapsar" "OK" "$(cat /tmp/bag_dizinv.log)"

# 8. --json şeması: donguler [] dizisi, cozuldu bayrağı
python3 -c "
import json
v = json.load(open('/tmp/bag_dongu.log'))
assert isinstance(v['donguler'], list) and isinstance(v['ters'], list)
for d in v['dosyalar']:
    for b in d['bagimliliklar']:
        assert 'cozuldu' in b and isinstance(b['cozuldu'], bool)
        assert 'hedef' in b
print('OK')
" > /tmp/bag_sema.log 2>&1
denetle "--json şeması" "OK" "$(cat /tmp/bag_sema.log)"

# 9. Sözdizimi hatası olan dosya → hata kaydı, exit 1
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" bagimlilik "$ISLER/bozuk.tan" > /tmp/bag_boz.log 2>&1
denetle "bozuk dosya exit 1" "1" "$?"

# 10. Deterministik çıktı
"$TAN" bagimlilik --json "$ISLER/ust.tan" > /tmp/bag_d1.log 2>&1
"$TAN" bagimlilik --json "$ISLER/ust.tan" > /tmp/bag_d2.log 2>&1
if cmp -s /tmp/bag_d1.log /tmp/bag_d2.log; then S=0; else S=1; fi
denetle "--json deterministik" "0" "$S"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
