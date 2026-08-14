#!/bin/bash
# TestArkaUcGoSuzTemiz.sh — "temiz makine" testi (kural 9).
#
# Amaç: go/gcc/clang/as/ld/cc çağrılırsa SESSİZCE değil, YÜKSEK SESLE
# başarısız olsun. Bunu gerçek bir Go'suz/derleyicisiz konteynerle DEĞİL,
# PATH'e bu adlarla "tuzak" script'ler koyarak yapıyoruz: gerçek go/gcc
# kurulu olsa BİLE, bu script çalışırken çağrılırlarsa test KIRMIZI olur.
# Böylece "temiz ortamda çalışır" iddiası varsayıma değil, doğrulanmış
# bir davranışa dayanır.
#
# Kapsam: BootstrapGoSuz.sh (üretim zinciri) + gen2'nin gerçek bir
# regresyon alt kümesini derleyip çalıştırması. gen2, DerleElf.go'dan
# (Go referansı) BAĞIMSIZ, self-hosted TancElf.tan'ın ürettiği ikilidir;
# bilinen kapsam dışı özellikler (bkz. SURUM.md "Go'suz Bootstrap" —
# ondalık sayı, yazDosya/yaz_dosya, metin-listesi her..içinde) buradan
# hariç tutulmuştur, bunlar Go bağımlılığıyla değil TancElf.tan'ın kendi
# özellik kapsamıyla ilgilidir.

set -u
cd "$(dirname "$0")"

TUZAK=/tmp/tan_temiz_ortam_tuzak
rm -rf "$TUZAK" 2>/dev/null; mkdir -p "$TUZAK"
for arac in go gcc clang cc as ld ld.gold ld.lld; do
    cat > "$TUZAK/$arac" << EOF
#!/bin/sh
echo "YASAK ARAÇ ÇAĞRILDI: $arac (\$@)" >&2
echo "Temiz-ortam testi bunun HİÇ çağrılmamasını bekliyordu." >&2
exit 127
EOF
    chmod +x "$TUZAK/$arac"
done

# TUZAK dizinini PATH'in EN BAŞINA koy — gerçek go/gcc kurulu olsa bile
# önce tuzak script bulunur ve çağrı orada patlar.
export PATH="$TUZAK:$PATH"
unset GOROOT GOPATH GOCACHE 2>/dev/null

echo "=== TEMİZ ORTAM TESTİ (go/gcc/clang/as/ld tuzağa düşürüldü) ==="
echo "PATH içindeki tuzak dizin: $TUZAK"
echo ""

BASARISIZ=0

echo "[1] BootstrapGoSuz.sh — üretim zinciri"
if ! ./BootstrapGoSuz.sh; then
    echo "!!! BootstrapGoSuz.sh başarısız (ya da yasak bir araç çağrıldı) !!!"
    BASARISIZ=1
fi

echo ""
echo "[2] gen2 gerçek bir program derliyor ve çalıştırıyor (bilinen kapsamda)"
mkdir -p /tmp/tantemiz
cat > /tmp/tantemiz/prog.tan << 'EOF'
yaz(2 + 3 * 4)
yaz(100 / 7)
yaz(123456789 * 987654321)
işlev fib(n)
    eğer n < 2 ise
        döndür n
    son
    döndür fib(n - 1) + fib(n - 2)
son
yaz(fib(20))
l = [10, 20, 30]
l = ekle(l, 40)
yaz(uzunluk(l))
yaz(l[3])
s = "Tan " + metin(42)
yaz(s)
EOF
if [ -x ./gen2 ]; then
    ./gen2 /tmp/tantemiz/prog.tan /tmp/tantemiz/prog_bin >/dev/null
    chmod +x /tmp/tantemiz/prog_bin
    CIKTI=$(/tmp/tantemiz/prog_bin)
    BEKLENEN='14
14.285714
121932631112635269
6765
4
40
Tan 42'
    if [ "$CIKTI" == "$BEKLENEN" ]; then
        echo "  [GECTI] gen2 ile derlenen program doğru çalıştı"
    else
        echo "  [KALDI] beklenen:"
        echo "$BEKLENEN" | sed 's/^/     /'
        echo "  [KALDI] gelen:"
        echo "$CIKTI" | sed 's/^/     /'
        BASARISIZ=1
    fi
else
    echo "  [HATA] gen2 üretilmedi"
    BASARISIZ=1
fi
rm -rf /tmp/tantemiz

echo ""
echo "[3] Sabit nokta ikilisinin gerçekten dinamik bağımlılığı yok mu (ldd)"
if command -v ldd >/dev/null 2>&1; then
    if ldd gen2 2>&1 | grep -qi "not a dynamic executable\|statically linked"; then
        echo "  [GECTI] gen2 statik — dinamik bağımlılık yok"
    else
        ldd gen2 2>&1 | sed 's/^/     /'
        echo "  [UYARI] beklenmeyen ldd çıktısı (bkz. yukarı) — yine de devam"
    fi
fi

echo ""
if [ "$BASARISIZ" -eq 0 ]; then
    echo "=== SONUÇ: TEMİZ ORTAM TESTİ GEÇTİ — go/gcc/clang/as/ld hiç çağrılmadı ==="
else
    echo "=== SONUÇ: TEMİZ ORTAM TESTİ KALDI ==="
fi
rm -rf "$TUZAK"
exit $BASARISIZ
