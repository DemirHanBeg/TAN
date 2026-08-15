#!/bin/bash
# TestOptimize.sh — Faz 1 optimizer regresyonu.
#
# Kapsam (Optimize.go -> TancElf.tan taşınan kurallar):
#   1. SAYITAM x SAYITAM sabit katlama   (+, -, *, %, karşılaştırmalar)
#   2. "/" 2^53 (9007199254740992) sınırı — aşan sabit bölme DERLEME HATASI
#   3. Cebirsel sadeleştirme             (x+0, 0+x, x-0, x*1, 1*x, x*0, 0*x)
#   4. Sabit koşul / ölü dal eleme       (eğer 1/0, değilse eğer, iken 0)
#
# Notlar:
#   - Katlanan değerlerin DOĞRULUĞU Go referansıyla aynı olmalı; değerler
#     deterministik beklenen çıktılarla karşılaştırılır (Go'ya bağımlılık yok).
#   - 2^53 kuralı testi, derleyicide fold AKTİF ise ancak anlamlıdır — eski
#     (fold'suz) derleyici burada beklenen hatayı vermez, test [KALDI] olur.
#   - Derleyici olarak gen2 (güncel) varsa o, yoksa TancElf (tohum) kullanılır.
#
# Çıkış: 0 = geçti, 1 = kaldı, 2 = altyapı hatası (derleyici yok).

set -u
cd "$(dirname "$0")"

BASARISIZ=0
CALISMA=/tmp/tanoptimize
rm -rf "$CALISMA" 2>/dev/null
mkdir -p "$CALISMA"

if [ -x ./gen2 ]; then
    DERLEYICI=./gen2
    DERLEYICI_AD="gen2 (güncel self-hosted derleyici)"
elif [ -x ./TancElf ]; then
    DERLEYICI=./TancElf
    DERLEYICI_AD="TancElf (tohum)"
else
    echo "  [HATA] ./gen2 veya ./TancElf bulunamadı — TestOptimize için TancElf.tan derleyicisi gerekli."
    rm -rf "$CALISMA"
    exit 2
fi

# derle_ve_karsilastir <test-adı> <program> <beklenen-çıktı>
# programı derler, çalıştırır, çıktıyı beklenenle karşılaştırır.
derle_ve_karsilastir() {
    local AD="$1"
    local PROGRAM="$2"
    local BEKLENEN
    BEKLENEN=$(printf '%b' "$3")
    printf '%b' "$PROGRAM" > "$CALISMA/$AD.tan"
    if ! "$DERLEYICI" "$CALISMA/$AD.tan" "$CALISMA/$AD"_bin >/dev/null 2>&1; then
        echo "  [KALDI] $AD — derleme hatası"
        BASARISIZ=1
        return
    fi
    chmod +x "$CALISMA/$AD"_bin 2>/dev/null
    local CIKTI
    CIKTI=$("$CALISMA/$AD"_bin)
    if [ "$CIKTI" == "$BEKLENEN" ]; then
        echo "  [GECTI] $AD — çıktı beklenenle aynı"
    else
        echo "  [KALDI] $AD — beklenen:"
        echo "$BEKLENEN" | sed 's/^/     /'
        echo "  [KALDI] $AD — gelen:"
        echo "$CIKTI" | sed 's/^/     /'
        BASARISIZ=1
    fi
}

echo "=== FAZ 1 OPTIMIZER TESTI ==="
echo "Derleyici: $DERLEYICI_AD"
echo ""

echo "[1] Sabit katlama — aritmetik ve karşılaştırmalar"
derle_ve_karsilastir "katlama" 'yaz(2 + 3 * 4)\nyaz(10 - 4)\nyaz(6 / 3)\nyaz(7 / 2)\nyaz(100 % 7)\nyaz(5 > 3)\nyaz(2 * 3 * 4)\nyaz(1 + 2 + 3)\nyaz(0 - 5)\nyaz(2 * 3 + 4 * 5)\nyaz(100 - 5 - 3)\nyaz(2 == 2)\nyaz(2 != 2)\nyaz(7 >= 7)\nyaz(7 <= 6)\nyaz(7 < 8)\nyaz(3 <= 4)\nyaz(3 >= 4)' '14\n6\n2\n3.5\n2\n1\n24\n6\n-5\n26\n92\n1\n0\n1\n0\n1\n1\n0'

echo ""
echo "[2] Büyük sabitler — 2^53 sınırında katlama ve 2^53 üstü runtime"
derle_ve_karsilastir "buyuk" 'yaz(4503599627370496 + 4503599627370496)\nyaz(9007199254740992 * 2)\nyaz(123456789 * 987654321)\nyaz(9007199254740992 / 2)\nyaz(9007199254740992 / 1)\nyaz(9007199254740993 * 1)' '9007199254740992\n18014398509481984\n121932631112635269\n4503599627370496\n9007199254740992\n9007199254740993'

echo ""
echo "[3] Cebirsel sadeleştirme"
derle_ve_karsilastir "cebirsel" 'x = 7\nyaz(x + 0)\nyaz(0 + x)\nyaz(x - 0)\nyaz(x * 1)\nyaz(1 * x)\nyaz(x * 0)\nyaz(0 * x)\ns = "merhaba"\nyaz(s + "")' '7\n7\n7\n7\n7\n0\n0\nmerhaba'

echo ""
echo "[4] Sabit koşul / ölü dal eleme"
derle_ve_karsilastir "oludall" 'eğer 1 ise\n    yaz(111)\ndeğilse\n    yaz(222)\nson\neğer 0 ise\n    yaz(333)\ndeğilse eğer 1 ise\n    yaz(444)\ndeğilse\n    yaz(555)\nson\niken 0\n    yaz(666)\nson\neğer 0 ise\n    yaz(777)\ndeğilse eğer 0 ise\n    yaz(888)\ndeğilse\n    yaz(999)\nson' '111\n444\n999'

echo ""
echo "[5] Fold + ölü dal döngü içinde (kontrol akışı bütünlüğü)"
derle_ve_karsilastir "dongu" 'i = 0\niken i < 3\n    i = i + 1\n    eğer 2 * 3 == 6 ise\n        yaz(i * 10)\n    son\nson\ni = 0\niken i < 3\n    i = i + 1\n    eğer 0 ise\n        yaz(99)\n    değilse\n        yaz(i)\n    son\nson' '10\n20\n30\n1\n2\n3'

echo ""
echo "[6] 2^53 sınırını aşan sabit '/' — DERLEME HATASI (exit 1)"
printf 'yaz(9007199254740993 / 1)\n' > "$CALISMA/sinir.tan"
if "$DERLEYICI" "$CALISMA/sinir.tan" "$CALISMA/sinir"_bin >"$CALISMA/sinir_out" 2>&1; then
    echo "  [KALDI] 2^53 sabit bölme HATA vermedi (fold aktif mi?)"
    BASARISIZ=1
else
    if grep -q "2^53" "$CALISMA/sinir_out"; then
        echo "  [GECTI] 2^53 sabit bölme derleme hatası + '2^53' mesajı (exit 1)"
    else
        echo "  [KALDI] 2^53 hatası verildi ama mesajda '2^53' yok:"
        cat "$CALISMA/sinir_out" | sed 's/^/     /'
        BASARISIZ=1
    fi
fi

echo ""
echo "[7] Sıfıra bölme runtime'a bırakılıyor (exit 4)"
printf 'yaz(5 %% 0)\n' > "$CALISMA/sifir.tan"
if "$DERLEYICI" "$CALISMA/sifir.tan" "$CALISMA/sifir"_bin >/dev/null 2>&1; then
    chmod +x "$CALISMA/sifir"_bin 2>/dev/null
    "$CALISMA/sifir"_bin >"$CALISMA/sifir_run" 2>&1
    SIFIR_EXIT=$?
    if [ "$SIFIR_EXIT" -eq 4 ] && grep -q "sıfıra bölme" "$CALISMA/sifir_run"; then
        echo "  [GECTI] 5 % 0 -> çalışma anı hatası, exit 4"
    else
        echo "  [KALDI] 5 % 0 -> exit $SIFIR_EXIT (4 bekleniyordu)"
        BASARISIZ=1
    fi
else
    echo "  [KALDI] 5 % 0 derlenemedi (derleme aşamasında katlanmamalı)"
    BASARISIZ=1
fi

echo ""
if [ "$BASARISIZ" -eq 0 ]; then
    echo "=== SONUÇ: FAZ 1 OPTIMIZER TESTI GEÇTİ ==="
else
    echo "=== SONUÇ: FAZ 1 OPTIMIZER TESTI KALDI ==="
fi
rm -rf "$CALISMA"
exit $BASARISIZ
