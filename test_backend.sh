#!/bin/bash
# Tan arka uc regresyon testi
# Beklenen degerler ACIK yazilir. Yorumlayici referans DEGIL:
# yorumlayici float64 kullanir, native arka uclar int64 —
# buyuk tam sayilarda native arka uc daha dogrudur.
#
# Kullanim: ./test_backend.sh [backend...]    (varsayilan: elf asm)

set -u
BACKENDS="${*:-elf asm}"
GECTI=0
KALDI=0
mkdir -p /tmp/tantests

test_et() {
    local ad="$1"; local kod="$2"; local beklenen="$3"
    printf '%s\n' "$kod" > /tmp/tantests/$ad.tan
    for b in $BACKENDS; do
        local bin="/tmp/tantests/${ad}_${b}"
        if ! ./tan "$b" /tmp/tantests/$ad.tan "$bin" >/dev/null 2>&1; then
            echo "  [HATA ] $ad / $b  -> derlenemedi"
            KALDI=$((KALDI+1)); continue
        fi
        local cikti; cikti=$("$bin" 2>/dev/null)
        if [ "$cikti" == "$beklenen" ]; then
            echo "  [GECTI] $ad / $b"
            GECTI=$((GECTI+1))
        else
            echo "  [KALDI] $ad / $b"
            echo "     beklenen: $(printf '%s' "$beklenen" | tr '\n' '|')"
            echo "     gelen   : $(printf '%s' "$cikti" | tr '\n' '|')"
            KALDI=$((KALDI+1))
        fi
    done
}

echo "=== Tan arka uc regresyon testi  (backendler: $BACKENDS) ==="
echo ""

test_et aritmetik 'yaz(2 + 3 * 4)
yaz((2 + 3) * 4)
yaz(100 / 7)
yaz(100 % 7)
yaz(0 - 5 + 3)
yaz(2 * 3 * 4 * 5)' '14
20
14
2
-2
120'

test_et karsilastirma 'yaz(5 > 3)
yaz(5 < 3)
yaz(5 == 5)
yaz(5 != 5)
yaz(5 >= 5)
yaz(5 <= 4)' '1
0
1
0
1
0'

test_et mantik 'yaz(1 ve 1)
yaz(1 ve 0)
yaz(0 veya 1)
yaz(0 veya 0)
yaz(7 > 3 ve 2 < 1)
yaz(7 > 3 veya 2 < 1)' '1
0
1
0
0
1'

test_et kosul 'x = 10
eğer x > 5 ise
    yaz(1)
değilse
    yaz(2)
son
eğer x > 100 ise
    yaz(3)
değilse
    yaz(4)
son' '1
4'

test_et dongu 't = 0
i = 1
iken i <= 1000
    t = t + i
    i = i + 1
son
yaz(t)' '500500'

test_et ozyineleme 'işlev f(n)
    eğer n <= 1 ise
        döndür 1
    son
    döndür n * f(n - 1)
son
yaz(f(10))
işlev fib(n)
    eğer n < 2 ise
        döndür n
    son
    döndür fib(n - 1) + fib(n - 2)
son
yaz(fib(20))' '3628800
6765'

test_et ic_ice 't = 0
i = 1
iken i <= 50
    j = 1
    iken j <= 50
        t = t + i * j
        j = j + 1
    son
    i = i + 1
son
yaz(t)
k = 0
iken 1 == 1
    k = k + 1
    eğer k > 99 ise
        dur
    son
son
yaz(k)' '1625625
100'

test_et buyuk_sayi 'yaz(1000000 * 1000000)
yaz(123456789 * 987654321)
yaz(0 - 1000000000000000)' '1000000000000
121932631112635269
-1000000000000000'

test_et alti_parametre 'işlev alti(a, b, c, d, e, f)
    döndür a * 100000 + b * 10000 + c * 1000 + d * 100 + e * 10 + f
son
yaz(alti(1, 2, 3, 4, 5, 6))' '123456'

test_et obeb 'işlev obeb(a, b)
    iken b != 0
        t = b
        b = a % b
        a = t
    son
    döndür a
son
yaz(obeb(1071, 462))
yaz(obeb(123456, 7890))' '21
6'

test_et asal_sayaci 'işlev asal_mi(n)
    eğer n < 2 ise
        döndür 0
    son
    i = 2
    iken i * i <= n
        eğer n % i == 0 ise
            döndür 0
        son
        i = i + 1
    son
    döndür 1
son
s = 0
n = 2
iken n < 1000
    eğer asal_mi(n) ise
        s = s + 1
    son
    n = n + 1
son
yaz(s)' '168'

test_et metin_sabiti 'yaz("merhaba")
yaz("Tan kendi ayaginda")
yaz(42)' 'merhaba
Tan kendi ayaginda
42'

echo ""
echo "=== SONUC: $GECTI gecti, $KALDI kaldi ==="
rm -rf /tmp/tantests
[ "$KALDI" -eq 0 ]
