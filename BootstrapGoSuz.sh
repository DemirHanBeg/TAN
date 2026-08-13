#!/bin/bash
# BootstrapGoSuz.sh — TAN derleyicisini SIFIR Go ile yeniden üretir.
#
# Kullanılan TEK "dış" girdi: bu depoda commit'li duran `TancElf` — daha
# önceki bir turda (Go tohumuyla) üretilmiş, statik bağlı, native x86-64 ELF
# ikili. Bu script'in kanıtladığı şey: o ikili + TancElf.tan kaynağı yeterli;
# ne Go, ne gcc/clang, ne as/ld, ne libc gerekiyor.
#
#   TancElf (commit'li bootstrap artifact)
#        -> gen1 = TancElf TancElf.tan gen1
#        -> gen2 = gen1    TancElf.tan gen2
#        -> gen3 = gen2    TancElf.tan gen3
#        -> cmp gen2 gen3   (sabit nokta — SESSİZ ise birebir aynı)
#
# Go SADECE referans/çapraz-kontrol için TestArkaUc.sh ve FarkTesti.sh
# içinde kullanılabilir (varsa) — bu script'in KENDİSİ Go'ya hiç dokunmaz.
#
# Neden `TancElf` "tek zorunlu artifact": TancElf.tan'ın KENDİSİ bir
# derleyici kaynağıdır ama onu ilk kez makine koduna çevirecek BİR
# derleyiciye ihtiyaç var — bu klasik "T-diyagramı" bootstrap sorunu.
# `TancElf` o ilk adımı bir daha hiç Go'ya sormadan atlatmak için burada
# duruyor; deterministik/reprodüksiyonu KanitGoSuzTarihce.sh ile ayrıca
# kanıtlanmıştır (bkz. o script + SURUM.md "Go'suz Bootstrap" bölümü).

set -eu
cd "$(dirname "$0")"

if [ ! -x ./TancElf ]; then
    echo "HATA: ./TancElf bulunamadı ya da çalıştırılabilir değil."
    echo "Bu, Go'suz bootstrap için gereken TEK ikili artifact."
    exit 1
fi

echo "=== Go'suz bootstrap: TancElf -> gen1 -> gen2 -> gen3 ==="
echo "(dış araç: YOK — gcc/clang/as/ld/go hiçbiri çağrılmıyor)"
echo ""

echo "[1/3] TancElf, TancElf.tan'ı derliyor -> gen1"
./TancElf TancElf.tan gen1
chmod +x gen1

echo "[2/3] gen1, TancElf.tan'ı derliyor -> gen2"
./gen1 TancElf.tan gen2
chmod +x gen2

echo "[3/3] gen2, TancElf.tan'ı derliyor -> gen3"
./gen2 TancElf.tan gen3
chmod +x gen3

echo ""
if cmp -s gen2 gen3; then
    echo "SABİT NOKTA: gen2 == gen3 — Go'suz self-hosting doğrulandı."
else
    echo "UYARI: gen2 != gen3 — sabit nokta TUTMADI."
    echo "Bu, TancElf.tan'a EKLENEN son özelliğin kendi kendini derlerken"
    echo "kararlı olmadığı anlamına gelir (bkz. SURUM.md, 'parametre tipi"
    echo "çıkarımı' benzeri geçmiş örnekler). gen3'ü tekrar kendi üzerinde"
    echo "çalıştırıp (gen3 TancElf.tan gen4, cmp gen3 gen4) yakınsayıp"
    echo "yakınsamadığına bakın; yakınsamıyorsa gerçek bir self-hosting"
    echo "hatasıdır, TancElf.tan'da düzeltilmelidir."
    exit 1
fi

echo ""
echo "İsteğe bağlı doğrulama (mevcutsa Go referansına karşı):"
echo "  ./TestArkaUc.sh elf      # Go'nun kendi elf arka ucu, çapraz kontrol"
echo "  ./FarkTesti.sh ornekler/*.tan"
echo ""
echo "gen2 artık bu depodaki güncel TancElf.tan'ın Go'suz, kendi kendine"
echo "üretilmiş derleyicisidir. Yeni bir bootstrap tohumu olarak commit'lemek"
echo "için: cp gen2 TancElf"
