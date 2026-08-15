#!/bin/bash
# TestDerle — "tan derle" komutunu sınar: varsayılan/-o çıktı adı, ELF derleme,
# çalıştırılabilirlik, hata çıkışları, "içe al" ve paket entegrasyonu.
# Kullanım: ./TestDerle.sh
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
ISLER="/tmp/tan_derle_isler"
rm -rf "$ISLER"
mkdir -p "$ISLER/alt"

echo "=== tan derle ==="

# 1. Kullanım ve hata çıkışları
"$TAN" derle > /tmp/derle_usage.log 2>&1
denetle "argümansız exit 2" "2" "$?"
grep -q "Kullanım: tan derle" /tmp/derle_usage.log
denetle "argümansız kullanım mesajı" "0" "$?"
"$TAN" derle --yardim > /tmp/derle_help.log 2>&1
denetle "--yardim exit 0" "0" "$?"
grep -q "ELF arka ucu" /tmp/derle_help.log
denetle "--yardim metni" "0" "$?"
"$TAN" derle selam.tan --bilinmeyen > /tmp/derle_opt.log 2>&1
denetle "bilinmeyen seçenek exit 2" "2" "$?"

# 2. Varsayılan çıktı adı: <ad> (uzantısız), çalıştırılabilir
printf 'yaz(6 * 7)\n' > "$ISLER/selam.tan"
(cd "$ISLER" && "$TAN" derle selam.tan) > /tmp/derle_1.log 2>&1
denetle "derle varsayılan exit 0" "0" "$?"
grep -q "derlendi: selam" /tmp/derle_1.log
denetle "çıktı adı 'selam' bildirilir" "0" "$?"
test -x "$ISLER/selam"
denetle "çıktı çalıştırılabilir" "0" "$?"
CIKTI=$("$ISLER/selam" 2>/dev/null)
denetle "çalıştırılan çıktı doğru" "42" "$CIKTI"

# 3. -o ve --cikti
(cd "$ISLER" && "$TAN" derle selam.tan -o uygulama) > /tmp/derle_o.log 2>&1
denetle "-o exit 0" "0" "$?"
grep -q "derlendi: uygulama" /tmp/derle_o.log
denetle "-o çıktı adı bildirilir" "0" "$?"
CIKTI=$("$ISLER/uygulama" 2>/dev/null)
denetle "-o çıktısı doğru" "42" "$CIKTI"
(cd "$ISLER" && "$TAN" derle selam.tan --cikti diger_uygulama) > /tmp/derle_ci.log 2>&1
denetle "--cikti exit 0" "0" "$?"
CIKTI=$("$ISLER/diger_uygulama" 2>/dev/null)
denetle "--cikti çıktısı doğru" "42" "$CIKTI"
"$TAN" derle "$ISLER/selam.tan" -o > /tmp/derle_oe.log 2>&1
denetle "-o değersiz exit 2" "2" "$?"

# 4. Hata yolları
"$TAN" derle "$ISLER/yok.tan" > /tmp/derle_yok.log 2>&1
denetle "olmayan dosya exit 1" "1" "$?"
grep -q "Dosya okunamadı" /tmp/derle_yok.log
denetle "olmayan dosya tanısı" "0" "$?"
printf '(1 + 2) = 5\n' > "$ISLER/bozuk.tan"
"$TAN" derle "$ISLER/bozuk.tan" > /tmp/derle_boz.log 2>&1
denetle "sözdizimi hatası exit 1" "1" "$?"
grep -q "TAN2001" /tmp/derle_boz.log
denetle "sözdizimi TAN2001 tanısı" "0" "$?"

# 5. içe al: modül derleme zamanında genişletilir
printf 'işlev kare(x)\n\tdöndür x * x\nson\n' > "$ISLER/islem.tan"
printf 'içe al "islem"\nyaz kare(9)\n' > "$ISLER/ana.tan"
(cd "$ISLER" && "$TAN" derle ana.tan) > /tmp/derle_ic.log 2>&1
denetle "içe al derle exit 0" "0" "$?"
CIKTI=$("$ISLER/ana" 2>/dev/null)
denetle "içe al çıktısı doğru" "81" "$CIKTI"

# 6. Alt dizinden gelen kaynağın çıktısı çalışma dizinine yazılır
printf 'yaz(100 / 4)\n' > "$ISLER/alt/bolme.tan"
(cd "$ISLER" && "$TAN" derle alt/bolme.tan) > /tmp/derle_alt.log 2>&1
denetle "alt dizin derle exit 0" "0" "$?"
grep -q "derlendi: bolme" /tmp/derle_alt.log
denetle "çıktı adı dosyadan türetilir" "0" "$?"
CIKTI=$("$ISLER/bolme" 2>/dev/null)
denetle "alt dizin çıktısı doğru" "25" "$CIKTI"

# 7. Paket entegrasyonu: tan paket kurulan modül derlenir
export TAN_ONELLEK="$ISLER/onbellek"
PROJE="$ISLER/proje"
mkdir -p "$PROJE" "$ISLER/kaynaklar"
printf 'işlev selam()\n\tdöndür "paket selam"\nson\n' > "$ISLER/kaynaklar/Giris.tan"
(cd "$PROJE" && "$TAN" paket yeni demo 0.1.0 && "$TAN" paket ekle Giris "$ISLER/kaynaklar/Giris.tan" && "$TAN" paket indir && "$TAN" paket kur) > /tmp/derle_pkt.log 2>&1
denetle "paket kurulumu exit 0" "0" "$?"
printf 'içe al "Giris"\nyaz selam()\n' > "$PROJE/ana.tan"
(cd "$PROJE" && "$TAN" derle ana.tan) > /tmp/derle_pkt2.log 2>&1
denetle "paket modülü derle exit 0" "0" "$?"
CIKTI=$("$PROJE/ana" 2>/dev/null)
denetle "paket modülü çıktısı doğru" "paket selam" "$CIKTI"

# 8. Yorumlayıcı ve derlenmiş program aynı sonucu üretir
printf 't = 0\ni = 1\niken i <= 100\n\tt = t + i\n\ti = i + 1\nson\nyaz(t)\n' > "$ISLER/toplam.tan"
(cd "$ISLER" && "$TAN" derle toplam.tan) > /tmp/derle_top.log 2>&1
denetle "döngü derle exit 0" "0" "$?"
CIKTI=$("$ISLER/toplam" 2>/dev/null)
denetle "derlenmiş döngü çıktısı" "5050" "$CIKTI"

echo ""
echo "=== SONUC: $gecti gecti, $kaldi kaldi ==="
rm -rf "$ISLER"
[ "$kaldi" -eq 0 ]
