#!/bin/bash
# Tan — tam dogrulama: self-hosting + regresyon
# Kademe 1 (Tan->C->gcc) ve Kademe 2 (Tan->asm->as/ld) ARŞİVLENDİ
# (bkz. arsiv/DerleC.go, arsiv/DerleAsm.go) — self-hosting ile
# (Kademe 3+4: sıfır dış araç) gereksiz kaldılar.
#
# GO ARTIK ZORUNLU DEĞİL. Bu script Go varsa çapraz-kontrol için kullanır
# (bkz. SURUM.md "Go'suz Bootstrap"), ama üretim zinciri artık
# BootstrapGoSuz.sh'e taşındı: commit'li `TancElf` ikilisi + TancElf.tan
# kaynağı yeterli, Go/gcc/clang/as/ld hiçbiri gerekmiyor. Temiz makine
# (Go kurulu olmayan) testi için doğrudan: ./BootstrapGoSuz.sh
set -e

echo "=== GO'SUZ ÜRETİM ZİNCİRİ (birincil yol) ==="
./BootstrapGoSuz.sh

if command -v go >/dev/null 2>&1; then
    echo ""
    echo "[çapraz-kontrol] Go bulundu — referans motoru derleyip karşılaştırılıyor"
    echo "(bu adım isteğe bağlıdır; Go yoksa script burada durmadan devam eder"
    echo " demek yerine zaten yukarıda tamamlandı)"
    go build -o tan .

    echo ""
    echo "=== KADEME 3+4: kendi assembler + kendi linker (Go referansı) ==="
    ./tan elf AsmTest.tan k34 && ./k34
    echo "boyut: $(stat -c%s k34) bayt (SIFIR dis arac)"
    file k34
    ldd k34 2>&1 || true

    echo ""
    echo "=== GERCEK PROGRAM: kesim optimizasyonu (yorumlayıcı) ==="
    ./tan Kesim.tan | tail -8

    echo ""
    echo "=== REGRESYON (Go referansına karşı) ==="
    ./TestArkaUc.sh elf
    ./FarkTesti.sh ornekler/*.tan
else
    echo ""
    echo "[bilgi] Go kurulu değil — çapraz-kontrol adımları atlandı."
    echo "Bu BEKLENEN ve KABUL EDİLEBİLİR bir durum: üretim zinciri Go'ya"
    echo "bağımlı değil, sadece EK bir bağımsız doğrulama katmanı Go'suz"
    echo "makinede çalışmıyor. gen2/gen3 sabit noktası zaten kanıtlandı."
fi
