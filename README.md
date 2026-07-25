# Tan

**A programming language with Turkish keywords that compiles to native x86-64 binaries with zero external tools — its own assembler, its own linker.**

*Türkçe anahtar kelimeli, kendi assembler'ı ve kendi linker'ı olan, sıfır dış bağımlılıkla native binary üreten programlama dili.*

```
$ tan elf program.tan cikti
ELF doğrudan yazıldı: cikti (1048 bayt)
Kullanılan dış araç: YOK (as/ld/gcc/libc hiçbiri)

$ ldd cikti
        not a dynamic executable
```

---

## Self-hosting: Tan compiles Tan

**`TancElf.tan` — the compiler itself, written in Tan — compiles itself and reproduces itself byte-for-byte.**

```bash
./tan elf TancElf.tan gen1   # Go seed compiles the self-hosted compiler
./gen1 TancElf.tan gen2      # gen1 compiles it again
./gen2 TancElf.tan gen3      # gen2 compiles it again
cmp gen2 gen3                 # silent — byte-identical, fixed point reached
```

Go was used to build the first seed (`gen1`). After that, **Go's role in producing the compiler is retired** — `gen2` and every generation after it are produced entirely by Tan compiling Tan, zero external tools, zero Go. Go source (`DerleElf.go`, the interpreter, the VM) stays in the tree as an independent reference: `FarkTesti.sh` cross-checks `gen`'s output against Go's own `tan elf` output on every change, so a bug introduced into the self-hosted compiler can still be caught against an independent implementation (see `SURUM.md` 0.5.0 for the nine self-hosting bugs this caught).

The two external-tool-dependent legacy backends (`tan derle` → C → gcc, `tan asm` → x86-64 asm → as/ld) are archived in `arsiv/` — superseded by `tan elf`, which needs nothing.

At the machine-code level, Tan writes the REX prefixes, ModRM bytes, RIP-relative addressing, label fixups, the ELF64 header and the program header by hand. There is no `printf` — integer-to-string conversion is hand-written machine code and output goes through a raw `write` syscall. Strings and lists are backed by a hand-written `brk`-based bump allocator, no libc.

---

## Quick start

```bash
git clone https://github.com/DemirHanBeg/TAN.git
cd TAN
go build -o tan .          # Go seed — only needed once, to bootstrap gen1

./tan Ornek.tan             # run with the interpreter
./tan elf AsmTest.tan out   # compile to a native binary, zero external tools
./out
```

Verify everything, including the self-hosting fixed point:

```bash
./Bootstrap.sh          # elf backend + self-hosting chain + regression
./TestArkaUc.sh elf   # 20 backend regression tests
./FarkTesti.sh ornekler/*.tan   # interpreter vs native cross-check
```

---

## Hello, Tan

```tan
yaz("Merhaba Tan")

işlev faktoriyel(n)
    eğer n <= 1 ise
        döndür 1
    son
    döndür n * faktoriyel(n - 1)
son

yaz(faktoriyel(20))     # 2432902008176640000 — exact, int64
```

Keywords: `işlev` (function), `döndür` (return), `eğer/değilse/son` (if/else/end), `iken` (while), `her ... içinde` (for each), `dur/devam` (break/continue), `yaz` (print), `içe al` (import), `dene/yakala` (try/catch).

---

## Real example: cutting-stock optimizer

`Kesim.tan` — a working production tool. Given stock bars and a cut list, it minimizes waste (First Fit Decreasing), accounts for saw kerf, and **verifies its own output** before you cut anything.

```
 Cubuk | Parcalar (mm)                        |  Fire | Doluluk
     3 | 1850+1850+1850+430                   |     8 |  %99.86
     7 | 1200+950+950+950+950+950             |    32 |  %99.46

Kullanilan stok : 11 cubuk = 66000 mm    Verim: %89.63

 OZ-DENETIM
   2400 mm : istenen 4 / planda 4   TAMAM
 DENGE DENETIMI: TAMAM — 59160 + 6657 + 183 = 66000
 >>> PLAN GECERLI. Kesime hazir.
```

---

## Number system

Tan distinguishes `int64` (exact) from `float64`. This matters:

```
123456789 * 987654321
  correct answer : 121932631112635269
  float64 result : 121932631112635260   ← wrong
  Tan            : 121932631112635269   ← exact
```

Rules: `int OP int → int`, `int OP float → float`, `int / int → int if divisible, else float`.

---

## Honest limitations

This is the part most projects hide. Read it before you judge.

**The `elf` backend handles:** int64 and float64 arithmetic, string and list variables (with a hand-written heap allocator), comparisons, `ve/veya/değil`, `eğer/değilse`, `iken`, `her...içinde`, `dur/devam`, functions with recursion, file I/O (`oku`/`yazBaytlar`), and is complete enough to compile itself (see Self-hosting above).

**It does NOT yet handle:** `içe al` (module imports — interpreter-only for now) and `dene/yakala` (try/catch — parsed by the lexer, not yet implemented in code generation; both raise a clear compile-time error rather than silently producing wrong output).

**Other open items:**
- x86-64 Linux only. Other architectures would need a new backend.
- No DWARF debug info, so `gdb` sees no symbols.
- The code generator is naive — no register allocation, no dead code elimination, no loop unrolling. Correct, not fast.
- WASM build exists but has not been tested in a real browser.
- The archived `derle`/`asm` backends (`arsiv/`) are frozen at whatever feature set they had when archived — not maintained further.

**What never goes away:** the x86-64 instruction set and the Linux syscall ABI. Those are not dependencies; they are the language being spoken.

---

## Repository layout

```
*.go                Go seed engine: lexer, parser, interpreter, bytecode VM,
                    Sayi.go (number system), DerleElf.go (elf backend —
                    still the reference/cross-check for TancElf.tan)
arsiv/              archived: DerleC.go, DerleAsm.go — the two external-
                    tool-dependent backends, superseded by tan elf
TancElf.tan         the self-hosting compiler, written entirely in Tan —
                    compiles itself, byte-identical fixed point proven
kutuphane/          31 standard library modules, written in Tan
Tanc.tan, Tanc2.tan, TancAsm.tan   earlier Tan-written compiler attempts
Kesim.tan           cutting-stock optimizer (real tool)
Talay.tan           freight index scoring pipeline
Noral.tan           neural network with backpropagation
testler/            test programs
TestArkaUc.sh     20 backend regression tests
FarkTesti.sh        interpreter-vs-native cross-check
Bootstrap.sh        elf backend + self-hosting chain + regression
web/                browser REPL (build tan.wasm separately)
```

---

## Building the WASM REPL

```bash
GOOS=js GOARCH=wasm go build -o web/tan.wasm .
```

---

## License

MIT — see [LICENSE](LICENSE).

Contributions welcome. The most useful ones right now: `içe al` (module import) support in the self-hosted compiler, `dene/yakala` (try/catch) code generation, the remaining handful of built-ins (`taban`, `tavan`, `sözlük`, `harfler`, `parçala`) in `TancElf.tan`'s own runtime, or an ARM64 backend.
