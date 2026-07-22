#!/bin/bash
# Tan — tam dogrulama: dort kademe + self-hosting + regresyon
set -e
echo "[tohum] Go ile motoru derle (tek seferlik)"
go build -o tan .

echo ""
echo "=== KADEME 1: Tan -> C -> gcc ==="
./tan derle AsmTest.tan k1 && ./k1
echo "boyut: $(stat -c%s k1) bayt (gcc+libc)"

echo ""
echo "=== KADEME 2: Tan -> asm -> as/ld ==="
./tan asm AsmTest.tan k2 && ./k2
echo "boyut: $(stat -c%s k2) bayt (libc YOK)"

echo ""
echo "=== KADEME 3+4: kendi assembler + kendi linker ==="
./tan elf AsmTest.tan k34 && ./k34
echo "boyut: $(stat -c%s k34) bayt (SIFIR dis arac)"
file k34
ldd k34 2>&1 || true

echo ""
echo "=== SELF-HOSTING: tanc2 (Tan ile yazildi) ==="
./tan derle Tanc2.tan tanc2
./tanc2 Ornek2.mt o2 && ./o2

echo ""
echo "=== GERCEK PROGRAM: kesim optimizasyonu ==="
./tan Kesim.tan | tail -8

echo ""
echo "=== REGRESYON ==="
./TestArkaUc.sh elf asm
