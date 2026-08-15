#!/bin/bash
# TestPaket — "tan paket" komutunu sınar: manifest/kilit biçimi, SHA-256
# doğrulama, önbellek, yerel/git bağımlılıkları, kurulum ve "içe al" entegrasyonu.
# Kullanım: ./TestPaket.sh
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

# paketci: proje dizininde komut çalıştırır; stdout+stderr'i log'a yazar.
paketci() {
    (cd "$PROJE" && "$TAN" "$@") > /tmp/paket.log 2>&1
    return $?
}

TAN="$(pwd)/tan"
REPO="$(pwd)"
PROJE="/tmp/tan_paket_test"
KAYNAK="/tmp/tan_paket_kaynak"
ONBELLEK="/tmp/tan_paket_ontest"
export TAN_ONELLEK="$ONBELLEK"

rm -rf "$PROJE" "$KAYNAK" "$ONBELLEK"
mkdir -p "$PROJE" "$KAYNAK/Islemler"

cat > "$KAYNAK/Giris.tan" <<'EOF'
işlev selam()
    döndür "paket selam"
son
EOF
cat > "$KAYNAK/Islemler/Topla.tan" <<'EOF'
işlev topla(a, b)
    döndür a + b
son
EOF
cat > "$KAYNAK/Islemler/Carp.tan" <<'EOF'
işlev carp(a, b)
    döndür a * b
son
EOF

echo "=== tan paket ==="

# 1. Kullanım ve hata çıkış kodları
"$TAN" paket > /tmp/paket_usage.log 2>&1
denetle "argümansız yardım exit 0" "0" "$?"
grep -q "Tan paket yöneticisi" /tmp/paket_usage.log
denetle "yardım metni görünür" "0" "$?"
"$TAN" paket bilinmeyen > /tmp/paket_bad.log 2>&1
denetle "bilinmeyen komut exit 2" "2" "$?"
"$TAN" paket yeni > /tmp/paket_noarg.log 2>&1
denetle "eksik argüman exit 2" "2" "$?"

# 2. yeni: manifest oluşur
paketci paket yeni demo 0.1.0
denetle "yeni exit 0" "0" "$?"
test -f "$PROJE/tan.paket"
denetle "tan.paket oluşur" "0" "$?"
grep -q "paket demo" "$PROJE/tan.paket"
denetle "manifest 'paket demo' içerir" "0" "$?"
paketci paket yeni diger 0.2.0
denetle "yinelenen yeni exit 1" "1" "$?"

# 3. ekle (yerel dosya): kilit + özet
paketci paket ekle Giris "$KAYNAK/Giris.tan"
denetle "ekle dosya exit 0" "0" "$?"
test -f "$PROJE/tan.lock"
denetle "tan.lock oluşur" "0" "$?"
OZET=$(grep -A1 "bağımlılık Giris" "$PROJE/tan.lock" | tail -1 | awk '{print $2}')
echo "$OZET" | grep -qE "^[0-9a-f]{64}$"
denetle "kilit özeti 64 hex" "0" "$?"
paketci paket ekle Giris "$KAYNAK/Giris.tan"
denetle "yinelenen ekle exit 1" "1" "$?"

# 4. liste
paketci paket liste
denetle "liste exit 0" "0" "$?"
grep -q "Giris" /tmp/paket.log
denetle "liste Giris'i gösterir" "0" "$?"

# 5. indir + kur + doğrula
paketci paket indir
denetle "indir exit 0" "0" "$?"
paketci paket kur
denetle "kur exit 0" "0" "$?"
test -f "$PROJE/tan_moduller/Giris/Giris.tan"
denetle "kur tan_moduller/Giris/Giris.tan yazar" "0" "$?"
test -f "$PROJE/tan_moduller/Giris/tan.json"
denetle "kur tan.json yazar" "0" "$?"
paketci paket doğrula
denetle "doğrula exit 0" "0" "$?"
grep -q "her şey tutarlı" /tmp/paket.log
denetle "doğrula 'her şey tutarlı' der" "0" "$?"

# 6. dizin bağımlılığı
paketci paket ekle Islemler "$KAYNAK/Islemler"
denetle "ekle dizin exit 0" "0" "$?"
paketci paket indir
denetle "dizin indir exit 0" "0" "$?"
paketci paket kur
denetle "dizin kur exit 0" "0" "$?"
test -f "$PROJE/tan_moduller/Islemler/tan.json"
denetle "dizin kur tan.json (giriş) yazar" "0" "$?"
paketci paket doğrula
denetle "dizin sonrası doğrula exit 0" "0" "$?"

# 7. git bağımlılığı: sıfır özet kaydı, ağ olmadan doğrulanamaz
paketci paket ekle UzakDep "git@ornek:dep.git"
denetle "ekle git exit 0" "0" "$?"
grep -A1 "bağımlılık UzakDep" "$PROJE/tan.lock" | grep -q "0000000000000000"
denetle "git sıfır özetle kilitlenir" "0" "$?"
paketci paket liste
grep -q "UzakDep" /tmp/paket.log
denetle "liste UzakDep'i gösterir" "0" "$?"
paketci paket indir
denetle "git indir (ağ yok) exit 1" "1" "$?"
grep -q "önbellek" /tmp/paket.log
denetle "git hata mesajı 'önbellek' ipucu içerir" "0" "$?"

# 8. git doğrulama: yerel kopya ile önbelleğe alınır
paketci paket önbellek UzakDep "$KAYNAK/Islemler"
denetle "önbellek exit 0" "0" "$?"
paketci paket indir
denetle "önbellek sonrası indir exit 0" "0" "$?"
paketci paket kur
denetle "önbellek sonrası kur exit 0" "0" "$?"
paketci paket doğrula
denetle "önbellek sonrası doğrula exit 0" "0" "$?"

# 9. Kurcalama: doğrula ve indir reddeder
cp "$KAYNAK/Giris.tan" "$KAYNAK/Giris.yedek"
echo "# bozuldu" >> "$KAYNAK/Giris.tan"
paketci paket doğrula
denetle "kurcalanmış kaynak doğrula exit 1" "1" "$?"
paketci paket indir
denetle "kurcalanmış kaynak indir exit 1" "1" "$?"
mv "$KAYNAK/Giris.yedek" "$KAYNAK/Giris.tan"
paketci paket doğrula
denetle "geri yükleme sonrası doğrula exit 0" "0" "$?"

# 10. Önbellek kurcalaması: doğrula yakalar, indir iyileştirir
echo "# k" >> "$ONBELLEK/$OZET.tan"
paketci paket doğrula
denetle "kurcalanmış önbellek doğrula exit 1" "1" "$?"
paketci paket indir
denetle "indir iyileştirme exit 0" "0" "$?"
paketci paket doğrula
denetle "iyileştirme sonrası doğrula exit 0" "0" "$?"

# 11. Bozuk manifest: ayrıştırma hatası
cp "$PROJE/tan.paket" "$PROJE/tan.paket.yedek"
printf 'paket demo\nsürüm 0.1.0\nbilinmeyen alan x\n' > "$PROJE/tan.paket"
paketci paket liste
denetle "bozuk manifest exit 1" "1" "$?"
grep -q "bilinmeyen anahtar" /tmp/paket.log
denetle "bozuk manifest tanısı" "0" "$?"
paketci paket yeni diger 0.2.0
denetle "bozuk manifest sonrası yeni exit 1" "1" "$?"
mv "$PROJE/tan.paket.yedek" "$PROJE/tan.paket"

# 12. sil
paketci paket sil Islemler
denetle "sil exit 0" "0" "$?"
paketci paket liste
grep -q "Islemler" /tmp/paket.log
denetle "sil manifestten kaldırır" "1" "$?"
test ! -d "$PROJE/tan_moduller/Islemler"
denetle "sil kurulumu temizler" "0" "$?"
paketci paket doğrula
denetle "sil sonrası doğrula exit 0" "0" "$?"

# 13. --json: geçerli JSON dizisi
paketci paket liste --json
if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import json
r = json.load(open('/tmp/paket.log'))
assert isinstance(r, list), type(r)
assert any(d['ad'] == 'Giris' and d['ozet'] for d in r), r
assert all('durum' in d and 'kaynak' in d for d in r)
print('ok')
" > /tmp/paket_json_chk.log 2>&1
    denetle "--json geçerli JSON + alanlar" "0" "$?"
else
    head -c1 /tmp/paket.log | grep -q "\["
    denetle "--json '[' ile başlar (python3 yok)" "0" "$?"
fi

# 14. içe al entegrasyonu: kurulan paket modül adıyla kullanılır
printf 'içe al "Giris"\nyaz selam()\n' > "$PROJE/ana.tan"
(cd "$PROJE" && "$TAN" ana.tan) > /tmp/paket_ic.log 2>&1
denetle "içe al entegrasyonu exit 0" "0" "$?"
grep -q "paket selam" /tmp/paket_ic.log
denetle "kurulan modülden çağrı çalışır" "0" "$?"

# 15. Kurulmamış modül adı bulunamaz (modulAra hata yolu)
printf 'içe al "YokBoyle"\n' > "$PROJE/eksik.tan"
(cd "$PROJE" && "$TAN" eksik.tan) > /tmp/paket_eksik.log 2>&1
denetle "kurulmamış modül exit 1" "1" "$?"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$PROJE" "$KAYNAK" "$ONBELLEK"
[ "$kaldi" -eq 0 ]
