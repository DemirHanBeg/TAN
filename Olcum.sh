#!/bin/bash
# Tan olcum takimi — arka uclarin hiz ve boyut karsilastirmasi
set -u
[ -f ./tan ] || go build -o tan .

mkdir -p /tmp/tanolcum
cat > /tmp/tanolcum/agir.tan <<'TAN'
işlev fib(n)
    eğer n < 2 ise
        döndür n
    son
    döndür fib(n - 1) + fib(n - 2)
son
yaz(fib(27))
TAN

cat > /tmp/tanolcum/dongu.tan <<'TAN'
t = 0
i = 0
iken i < 20000000
    t = t + i
    i = i + 1
son
yaz(t)
TAN

cat > /tmp/tanolcum/asal.tan <<'TAN'
işlev asal_mi(n)
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
iken n < 300000
    eğer asal_mi(n) ise
        s = s + 1
    son
    n = n + 1
son
yaz(s)
TAN

olc() {
    local ad="$1"; local dosya="/tmp/tanolcum/$1.tan"
    echo "--- $ad ---"
    # yorumlayici/VM
    local t0=$(date +%s%N)
    local out=$(./tan "$dosya" 2>/dev/null)
    local t1=$(date +%s%N)
    printf "  %-14s %8s ms   sonuc=%s\n" "yorumlayici/VM" "$(( (t1-t0)/1000000 ))" "$out"

    for b in derle elf; do
        ./tan "$b" "$dosya" "/tmp/tanolcum/${ad}_$b" >/dev/null 2>&1 || { printf "  %-14s (desteklemiyor)\n" "$b"; continue; }
        t0=$(date +%s%N)
        out=$("/tmp/tanolcum/${ad}_$b" 2>/dev/null)
        t1=$(date +%s%N)
        local boy=$(stat -c%s "/tmp/tanolcum/${ad}_$b")
        printf "  %-14s %8s ms   sonuc=%s   binary=%s bayt\n" "$b" "$(( (t1-t0)/1000000 ))" "$out" "$boy"
    done
    echo ""
}

echo "=========================================="
echo " TAN OLCUM TAKIMI"
echo "=========================================="
echo ""
olc fib
olc dongu
olc asal
rm -rf /tmp/tanolcum
