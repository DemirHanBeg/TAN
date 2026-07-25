#!/bin/bash
# Tan — tam dogrulama: self-hosting + regresyon
# Kademe 1 (Tan->C->gcc) ve Kademe 2 (Tan->asm->as/ld) ARŞİVLENDİ
# (bkz. arsiv/DerleC.go, arsiv/DerleAsm.go) — self-hosting ile
# (Kademe 3+4: sıfır dış araç) gereksiz kaldılar.
set -e
echo "[tohum] Go ile motoru derle (referans/çapraz-kontrol için, tek seferlik)"
go build -o tan .

echo ""
echo "=== KADEME 3+4: kendi assembler + kendi linker (Go tohum) ==="
./tan elf AsmTest.tan k34 && ./k34
echo "boyut: $(stat -c%s k34) bayt (SIFIR dis arac)"
file k34
ldd k34 2>&1 || true

echo ""
echo "=== SELF-HOSTING: TancElf.tan kendi kendini derliyor ==="
./tan elf TancElf.tan gen1
chmod +x gen1
./gen1 TancElf.tan gen2
chmod +x gen2
./gen2 TancElf.tan gen3
chmod +x gen3
cmp gen2 gen3 && echo "SABİT NOKTA: gen2 == gen3 (self-hosting kanıtlandı)"

echo ""
echo "=== GERCEK PROGRAM: kesim optimizasyonu (yorumlayıcı) ==="
./tan Kesim.tan | tail -8

echo ""
echo "=== REGRESYON ==="
./TestArkaUc.sh elf
./FarkTesti.sh ornekler/*.tan
