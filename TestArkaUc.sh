#!/bin/bash
# Tan arka uc regresyon testi
# Beklenen degerler ACIK yazilir. Yorumlayici referans DEGIL:
# yorumlayici float64 kullanir, native arka uclar int64 —
# buyuk tam sayilarda native arka uc daha dogrudur.
#
# Kullanim: ./TestArkaUc.sh [backend...]    (varsayilan: elf asm)

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

test_et metin_yigin 'a = "Tan"
b = a + " dili"
yaz(b)
yaz(uzunluk(b))
s = ""
i = 0
iken i < 5
    s = s + "ab"
    i = i + 1
son
yaz(s)
yaz(uzunluk(s))' 'Tan dili
8
ababababab
10'

test_et metin_cevirme 'yaz("sayi: " + metin(42))
yaz("negatif: " + metin(0 - 17))
n = 20
f = 1
k = 1
iken k <= n
    f = f * k
    k = k + 1
son
yaz(metin(n) + "! = " + metin(f))' 'sayi: 42
negatif: -17
20! = 2432902008176640000'

test_et liste 'l = [10, 20, 30]
yaz(uzunluk(l))
yaz(l[0])
l = ekle(l, 40)
yaz(l[3])
t = 0
her x l içinde
    t = t + x
son
yaz(t)
b = []
i = 0
iken i < 5
    b = ekle(b, i * i)
    i = i + 1
son
her v b içinde
    yaz(v)
son' '3
10
40
100
0
1
4
9
16'

test_et metin_liste 'm = ["bir", "iki", "uc"]
her s m içinde
    yaz(s)
son
s = "Tan"
yaz(s[0])
yaz(kod(s))
yaz(karakter(65))
yaz(uzunluk(s))' 'bir
iki
uc
T
84
A
3'

test_et ondalik 'yaz(3.14)
yaz(1.5 + 2.5)
yaz(10.0 / 4.0)
yaz(2.5 * 4.0)
yaz(0.0 - 1.75)
yaz(1 + 0.5)
yaz(3.0 > 2.0)
yaz(3.0 < 2.0)
x = 1.5
yaz(x * 2.0)
yaz("pi " + metin(3.14159))' '3.14
4
2.5
10
-1.75
1.5
1
0
3
pi 3.14159'

test_et dosya 'yaz_dosya("/tmp/tantest_io.txt", "Tan " + metin(7))
yaz(oku("/tmp/tantest_io.txt"))
yaz(uzunluk(oku("/tmp/tantest_io.txt")))' 'Tan 7
5'

test_et islev_tip 'işlev selam(ad)
    döndür "merhaba " + ad
son
işlev liste_yap(n)
    l = []
    i = 0
    iken i < n
        l = ekle(l, i * 2)
        i = i + 1
    son
    döndür l
son
işlev topla(l)
    t = 0
    her x l içinde
        t = t + x
    son
    döndür t
son
işlev yarim(x)
    döndür x / 2.0
son
yaz(selam("Demir"))
l = liste_yap(5)
yaz(uzunluk(l))
yaz(topla(l))
yaz(yarim(7.0))' 'merhaba Demir
5
20
3.5'

test_et karisik 'işlev tersine(s)
    r = ""
    i = 0
    iken i < uzunluk(s)
        r = s[i] + r
        i = i + 1
    son
    döndür r
son
yaz(tersine("abcdef"))
isimler = ["ali", "veli"]
her i isimler içinde
    yaz("merhaba " + i)
son
yaz(1.5 * 2.0 + 0.25)' 'fedcba
merhaba ali
merhaba veli
3.25'

echo ""
echo "=== SONUC: $GECTI gecti, $KALDI kaldi ==="
rm -rf /tmp/tantests
[ "$KALDI" -eq 0 ]
