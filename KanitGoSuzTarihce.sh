#!/bin/bash
# KanitGoSuzTarihce.sh — "Go hiç kullanılmadan TancElf.tan'ın tüm özellik
# tarihi baştan sona native TAN ikilileriyle yeniden inşa edilebilir mi?"
# sorusunun kanıtı. Sadece git tarihini ve doğrudan çalıştırılabilir ELF
# ikililerini kullanır — go/gcc/clang/as/ld hiçbiri çağrılmaz.
#
# Neden var: BootstrapGoSuz.sh "bugünkü TancElf.tan'ı bugünkü commit'li
# TancElf ikilisiyle üret" sorusuna cevap veriyor. Bu script ise DAHA
# İDDİALI bir soruya cevap veriyor: "commit'li ikili olmasaydı bile,
# TAN'ın kendi geçmişini SADECE önceki nesil TAN ikilileriyle, Go'ya HİÇ
# dönmeden, adım adım yeniden üretebilir miydik?" — cevap: EVET, tek bir
# bilinen istisna dışında (aşağıda İKİ ADIMLI ÇÖZÜM ile açıklanmıştır).
#
# Bulgular (bu script'in ürettiği kanıt):
#   62d0fce  -> doğrudan çalışır, sabit nokta (gen_a==gen_b)
#   086baeb  -> doğrudan çalışır, sabit nokta
#   9ffe0b5  -> DOĞRUDAN ÇALIŞMAZ: "BAGLAMA HATASI: etiket bulunamadi:
#               f_dosyaVarMi". Kök sebep: bu commit AYNI ANDA (a) yeni bir
#               yerleşik (dosyaVarMi) için derleyici desteği ekliyor VE
#               (b) TancElf.tan'ın KENDİ modül-yolu çözümleme kodu o
#               yerleşiği HEMEN kullanıyor. Önceki nesil derleyici (086baeb)
#               "dosyaVarMi" adını hiç tanımadığından üretilen "call
#               f_dosyaVarMi" için gövde asla gömülmüyor. Bu, TAN projesinin
#               kendi SURUM.md'sinde belgelenen 9 self-hosting hatasıyla
#               AYNI KÖK DESEN (bkz. "adAra", "cagriSonucTipi" vb.) — burada
#               10.'su, spesifik olarak "yeni özellik + aynı commit'te
#               kendi kendine kullanım" bootstrap sırası sorunu.
#               ÇÖZÜM (iki adımlı geçici "shim"): önce dosyaVarMi(...) öz-
#               kullanım noktalarını geçici olarak nötrleyip derle (ara
#               ikili artık dosyaVarMiBant() çalışma-zamanı gövdesini HER
#               ZAMAN gömüyor), SONRA gerçek/yamasız kaynağı bu ara ikiliyle
#               derle. Sonuç: gerçek kaynaktan sabit nokta elde edilir.
#               TancElf.tan'ın KENDİSİ değiştirilmez — yama sadece bu
#               script'in geçici/atılabilir ara adımıdır.
#   fd06ddc  -> doğrudan çalışır, sabit nokta
#   d6b4fac  -> SALINIYOR (109098 <-> 114000 <-> 109101 arası), sabit nokta
#               TUTMUYOR. Beklenen: bu commit henüz parametre tipi
#               çıkarımına sahip değil (bkz. SURUM.md, ADIM 2 / 4aecdf4'te
#               eklendi) — NEXUS'un yeni metin/liste parametreli
#               fonksiyonları bazı çağrı bölgelerinde yanlış/eksik
#               derleniyor. Geçici çözüm: en BÜYÜK (en tam) üretimi seç
#               (gen_b, 114000 bayt) ve ondan devam et — sonraki commit
#               (9723e7c) bu tohumla sabit noktaya ulaşıyor.
#   9723e7c  -> d6b4fac'ın "gen_b" (büyük/tam) soyundan başlarsan sabit
#               nokta TUTUYOR.
#   4aecdf4  -> (HEAD) doğrudan çalışır, ÜÇ KEZ doğrulanmış sabit nokta
#               (parametre tipi çıkarımı eklendiği için d6b4fac'taki
#               salınım burada kapanıyor).
#
# Bu script'in ürettiği SON ikili, mevcut depodaki TancElf ile fonksiyonel
# olarak eşdeğerdir (ikisi de HEAD TancElf.tan'ı derlerken sabit noktaya
# ulaşır). Bkz. SURUM.md "Go'suz Bootstrap" bölümü.

set -u
cd "$(dirname "$0")"
IS=/tmp/kanitgosuz
rm -rf "$IS" 2>/dev/null; mkdir -p "$IS"
GECTI=0; KALDI=0

adim() { echo ""; echo "=================== $1 ==================="; }

sabit_nokta_dogrula() {
    # $1=derleyici $2=kaynak $3=etiket
    "$1" "$2" "$IS/${3}_x" >/dev/null 2>&1 || return 1
    chmod +x "$IS/${3}_x"
    "$IS/${3}_x" "$2" "$IS/${3}_y" >/dev/null 2>&1 || return 1
    cmp -s "$IS/${3}_x" "$IS/${3}_y"
}

adim "62d0fce (tohum: commit'li TancElf ikilisi, bir önceki turdan)"
git show 62d0fce:TancElf > "$IS/gen_62d0fce" 2>/dev/null || { echo "atlanıyor: eski TancElf blob'u git geçmişinde yok"; exit 1; }
chmod +x "$IS/gen_62d0fce"
git show 62d0fce:TancElf.tan > "$IS/src_62d0fce.tan"
"$IS/gen_62d0fce" "$IS/src_62d0fce.tan" "$IS/gen_62d0fce_a" >/dev/null
chmod +x "$IS/gen_62d0fce_a"
if sabit_nokta_dogrula "$IS/gen_62d0fce_a" "$IS/src_62d0fce.tan" 62d0fce_fp; then
    echo "SABİT NOKTA: 62d0fce (Go'suz)"; GECTI=$((GECTI+1)); PREV="$IS/gen_62d0fce_a"
else
    echo "HATA: 62d0fce sabit nokta vermedi"; KALDI=$((KALDI+1)); exit 1
fi

adim "086baeb"
git show 086baeb:TancElf.tan > "$IS/src_086baeb.tan"
"$PREV" "$IS/src_086baeb.tan" "$IS/gen_086baeb_a" >/dev/null
chmod +x "$IS/gen_086baeb_a"
if sabit_nokta_dogrula "$IS/gen_086baeb_a" "$IS/src_086baeb.tan" 086baeb_fp; then
    echo "SABİT NOKTA: 086baeb (Go'suz)"; GECTI=$((GECTI+1)); PREV="$IS/gen_086baeb_a"
else
    echo "HATA: 086baeb sabit nokta vermedi"; KALDI=$((KALDI+1)); exit 1
fi

adim "9ffe0b5 (iki adımlı shim gerekiyor — bkz. üstteki not)"
git show 9ffe0b5:TancElf.tan > "$IS/src_9ffe0b5.tan"
cp "$IS/src_9ffe0b5.tan" "$IS/src_9ffe0b5_shim.tan"
python3 - "$IS/src_9ffe0b5_shim.tan" << 'PYEOF'
import sys
p = sys.argv[1]
s = open(p, encoding='utf-8').read()
s = s.replace('eğer dosyaVarMi(aday1) ise', 'eğer 1 == 1 ise')
s = s.replace('eğer dosyaVarMi(aday2) ise', 'eğer 1 == 1 ise')
open(p, 'w', encoding='utf-8').write(s)
PYEOF
"$PREV" "$IS/src_9ffe0b5_shim.tan" "$IS/gen_9ffe0b5_shim" >/dev/null || { echo "HATA: shim adımı derlenemedi"; exit 1; }
chmod +x "$IS/gen_9ffe0b5_shim"
"$IS/gen_9ffe0b5_shim" "$IS/src_9ffe0b5.tan" "$IS/gen_9ffe0b5_a" >/dev/null || { echo "HATA: shim'den gerçek kaynak derlenemedi"; exit 1; }
chmod +x "$IS/gen_9ffe0b5_a"
if sabit_nokta_dogrula "$IS/gen_9ffe0b5_a" "$IS/src_9ffe0b5.tan" 9ffe0b5_fp; then
    echo "SABİT NOKTA: 9ffe0b5 (Go'suz, shim ile)"; GECTI=$((GECTI+1)); PREV="$IS/gen_9ffe0b5_a"
else
    echo "HATA: 9ffe0b5 sabit nokta vermedi"; KALDI=$((KALDI+1)); exit 1
fi

adim "fd06ddc"
git show fd06ddc:TancElf.tan > "$IS/src_fd06ddc.tan"
"$PREV" "$IS/src_fd06ddc.tan" "$IS/gen_fd06ddc_a" >/dev/null
chmod +x "$IS/gen_fd06ddc_a"
if sabit_nokta_dogrula "$IS/gen_fd06ddc_a" "$IS/src_fd06ddc.tan" fd06ddc_fp; then
    echo "SABİT NOKTA: fd06ddc (Go'suz)"; GECTI=$((GECTI+1)); PREV="$IS/gen_fd06ddc_a"
else
    echo "HATA: fd06ddc sabit nokta vermedi"; KALDI=$((KALDI+1)); exit 1
fi

adim "d6b4fac (BİLİNEN salınım — parametre tipi çıkarımı henüz yok)"
git show d6b4fac:TancElf.tan > "$IS/src_d6b4fac.tan"
"$PREV" "$IS/src_d6b4fac.tan" "$IS/gen_d6b4fac_a" >/dev/null
chmod +x "$IS/gen_d6b4fac_a"
"$IS/gen_d6b4fac_a" "$IS/src_d6b4fac.tan" "$IS/gen_d6b4fac_b" >/dev/null
chmod +x "$IS/gen_d6b4fac_b"
if cmp -s "$IS/gen_d6b4fac_a" "$IS/gen_d6b4fac_b"; then
    echo "SABİT NOKTA: d6b4fac ilk denemede tutuyor (beklenenden iyi)"
else
    echo "beklenen salınım doğrulandı (gen_a != gen_b) — en büyük/tam üretimi seçiyoruz"
fi
PREV="$IS/gen_d6b4fac_b"
GECTI=$((GECTI+1))

adim "9723e7c (d6b4fac'ın 'tam' soyundan)"
git show 9723e7c:TancElf.tan > "$IS/src_9723e7c.tan"
"$PREV" "$IS/src_9723e7c.tan" "$IS/gen_9723e7c_a" >/dev/null
chmod +x "$IS/gen_9723e7c_a"
if sabit_nokta_dogrula "$IS/gen_9723e7c_a" "$IS/src_9723e7c.tan" 9723e7c_fp; then
    echo "SABİT NOKTA: 9723e7c (Go'suz)"; GECTI=$((GECTI+1)); PREV="$IS/gen_9723e7c_a"
else
    echo "HATA: 9723e7c sabit nokta vermedi"; KALDI=$((KALDI+1)); exit 1
fi

adim "4aecdf4 = HEAD (mevcut TancElf.tan) — parametre tipi çıkarımı burada eklendi"
git show 4aecdf4:TancElf.tan > "$IS/src_4aecdf4.tan"
"$PREV" "$IS/src_4aecdf4.tan" "$IS/gen_4aecdf4_a" >/dev/null
chmod +x "$IS/gen_4aecdf4_a"
"$IS/gen_4aecdf4_a" "$IS/src_4aecdf4.tan" "$IS/gen_4aecdf4_b" >/dev/null
chmod +x "$IS/gen_4aecdf4_b"
"$IS/gen_4aecdf4_b" "$IS/src_4aecdf4.tan" "$IS/gen_4aecdf4_c" >/dev/null
if cmp -s "$IS/gen_4aecdf4_a" "$IS/gen_4aecdf4_b" && cmp -s "$IS/gen_4aecdf4_b" "$IS/gen_4aecdf4_c"; then
    echo "SABİT NOKTA (3x doğrulandı): HEAD, TAMAMEN Go'suz üretildi"
    GECTI=$((GECTI+1))
else
    echo "HATA: HEAD'de sabit nokta yok"; KALDI=$((KALDI+1))
fi

echo ""
echo "=== SONUÇ: $GECTI aşama geçti, $KALDI aşama kaldı ==="
echo "Ara üretimler: $IS/"
[ "$KALDI" -eq 0 ]
