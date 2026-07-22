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

## What makes it different / Farkı ne

Most toy languages stop at "it compiles to C." Tan goes four stages down:

| Stage | Command | What Tan does itself | External tools | Binary size |
|-------|---------|----------------------|----------------|-------------|
| 1 | `tan derle` | expression tree → C | gcc + libc | 16 600 B |
| 2 | `tan asm` | + register allocation, stack frames, calling convention | as + ld | 9 088 B |
| 3+4 | `tan elf` | + **machine code bytes, ELF header, linking** | **none** | **1 048 B** |

At stage 3+4 Tan writes the REX prefixes, ModRM bytes, RIP-relative addressing, label fixups, the ELF64 header and the program header by hand. There is no `printf` — integer-to-string conversion is hand-written machine code and output goes through a raw `write` syscall.

**Self-hosting:** the compiler is also written in Tan (`Tanc2.tan`) — its own lexer, its own shunting-yard operator precedence, emitting x86-64 assembly. Go is used exactly once, as a seed, then it leaves the chain.

---

## Quick start

```bash
git clone https://github.com/karaefendii/tan.git
cd tan
go build -o tan .          # Go is only needed to build the engine (seed)

./tan Ornek.tan            # run with the interpreter
./tan elf AsmTest.tan out  # compile to a native binary, no external tools
./out
```

Verify everything:

```bash
./Bootstrap.sh        # all four stages + self-hosting + regression tests
./TestArkaUc.sh     # 24 backend regression tests
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

**The `elf` and `asm` backends handle:** int64 arithmetic, variables, comparisons, `ve/veya/değil`, `eğer/değilse`, `iken`, `dur/devam`, functions with recursion (up to 6 parameters), `yaz` for numbers and string literals.

**They do NOT handle:** floating point, string variables, lists, dictionaries, `her...içinde`, file or network I/O. Floating point needs SSE encoding; strings and lists need a heap allocator written against `brk`/`mmap`, since there is no libc. Attempting these raises a clear error rather than silently producing wrong output.

All of the above **does work** on the C backend (`tan derle`) and in the interpreter.

**Other open items:**
- `Tanc2.tan` (the Tan-written compiler) does not yet compile function definitions — it handles assignment, `yaz`, `eger`, `iken`, and full arithmetic with precedence.
- The bytecode VM does not cover the entire language; it falls back to the interpreter.
- x86-64 Linux only. Other architectures would need a new backend (the C path is portable and gets ARM/RISC-V for free).
- No DWARF debug info, so `gdb` sees no symbols.
- The code generator is naive — no register allocation, no dead code elimination, no loop unrolling. Correct, not fast.
- WASM build exists but has not been tested in a real browser.

**What never goes away:** the x86-64 instruction set and the Linux syscall ABI. Those are not dependencies; they are the language being spoken.

---

## Why keep the C backend

Deliberate. Three backends consume the same AST. Day-to-day language and library work happens on the C path — it is fast, portable, and libc is already there. The ELF path stands as the independence proof and for minimal deployment. Keeping both means development speed is not traded for independence.

---

## Repository layout

```
*.go                engine: lexer, parser, interpreter, bytecode VM,
                    Sayi.go (number system), and three backends:
                    DerleC.go, DerleAsm.go, DerleElf.go
kutuphane/          31 standard library modules, written in Tan
Tanc.tan            compiler written in Tan (emits C)
TancAsm.tan        compiler written in Tan (emits assembly)
Tanc2.tan           + comparison operators, eger, iken, nested blocks
Kesim.tan           cutting-stock optimizer (real tool)
Talay.tan           freight index scoring pipeline
Noral.tan           neural network with backpropagation
testler/            test programs
TestArkaUc.sh     24 backend regression tests
Bootstrap.sh        full verification
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

Contributions welcome. The most useful ones right now: SSE floating-point in the ELF backend, a `brk`/`mmap` heap allocator for strings and lists, function definitions in `Tanc2.tan`, or an ARM64 backend.
