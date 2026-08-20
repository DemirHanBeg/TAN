# Tan

**A programming language with Turkish keywords that compiles to native x86-64 binaries with zero external tools — its own assembler, its own linker, and now written entirely in itself.**

*Türkçe anahtar kelimeli, kendi assembler'ı ve kendi linker'ı olan, sıfır dış bağımlılıkla native binary üreten, %100 kendi kendini derleyen programlama dili.*

```
$ ./TancElf program.tan cikti
TancElf: program.tan -> cikti  (1048 bayt)

$ ldd cikti
        not a dynamic executable
```

---

## Self-hosting: Tan compiles Tan — ZERO Go

**`TancElf.tan` — the compiler itself, written in Tan — compiles itself and reproduces itself byte-for-byte, with no Go anywhere.** As of **2026-08-20 the Go reference engine was fully removed** (all `.go` files, `go.mod/sum`, and the Go-built binaries deleted; they remain in git history only).

```bash
./TancElf TancElf.tan gen1   # committed native seed compiles the self-hosted compiler
./gen1    TancElf.tan gen2   # gen1 compiles it again
./gen2    TancElf.tan gen3   # gen2 compiles it again
cmp gen2 gen3                 # silent — byte-identical, fixed point reached
```

or simply `./BootstrapGoSuz.sh`, which does exactly this and checks the result.
`TestArkaUcGoSuzTemiz.sh` runs it in a clean environment and asserts that
**go/gcc/clang/as/ld are never invoked**.

`TancElf` — a small (~146 KB), statically-linked, native x86-64 ELF binary
committed in this repo — is the one bootstrap artifact the chain needs, and it
is itself reproducible from git history using nothing but earlier native `Tan`
binaries (`KanitGoSuzTarihce.sh`).

At the machine-code level, Tan writes the REX prefixes, ModRM bytes, RIP-relative
addressing, label fixups, the ELF64 header and the program header by hand. There
is no `printf` — integer-to-string conversion is hand-written machine code and
output goes through a raw `write` syscall. Strings and lists are backed by a
hand-written `brk`-based bump allocator, no libc.

---

## Quick start (Go-free — the only path now)

```bash
git clone https://github.com/DemirHanBeg/TAN.git
cd TAN
./BootstrapGoSuz.sh        # TancElf -> gen1 -> gen2 -> gen3, fixed point, zero Go

./gen2 AsmTest.tan out     # gen2 is a full native Tan compiler — use it directly
./out
```

Verify, including the self-hosting fixed point and clean environment:

```bash
./TestArkaUcGoSuzTemiz.sh       # clean-room: no go/gcc/clang/as/ld touched
./KanitGoSuzTarihce.sh          # independent proof: rebuilds the history, zero Go
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

## Developer tooling (written in Tan, `araclar/`)

Self-hosted tooling — each compiles with `./TancElf` and emits `--json`:

```bash
./TancElf araclar/simgeler.tan simgeler && ./simgeler dosya.tan --json   # symbols
./TancElf araclar/denetle.tan  denetle  && ./denetle  dosya.tan --json   # block-balance lint
./TancElf araclar/bicimlendir.tan bic   && ./bic      dosya.tan          # canonical formatter
```

`kutuphane/tanoku.tan` holds the shared tokenizer/scan helpers all tools
`içe al`. This is the foundation for the coming LSP.

---

## Real example: cutting-stock optimizer

`Kesim.tan` — a working production tool. Given stock bars and a cut list, it minimizes waste (First Fit Decreasing), accounts for saw kerf, and **verifies its own output** before you cut anything.

```
 Cubuk | Parcalar (mm)                        |  Fire | Doluluk
     3 | 1850+1850+1850+430                   |     8 |  %99.86
     7 | 1200+950+950+950+950+950             |    32 |  %99.46

Kullanilan stok : 11 cubuk = 66000 mm    Verim: %89.63
 >>> PLAN GECERLI. Kesime hazir.
```

---

## Number system

Tan distinguishes `int64` (exact) from `float64`:

```
123456789 * 987654321
  correct answer : 121932631112635269
  float64 result : 121932631112635260   ← wrong
  Tan            : 121932631112635269   ← exact
```

Rules: `int OP int → int`, `int OP float → float`, `int / int → int if divisible, else float`.

---

## Honest limitations

**The compiler handles:** int64/float64 arithmetic, string and list variables (hand-written heap allocator), comparisons, `ve/veya/değil`, `eğer/değilse`, `iken`, `her...içinde`, `dur/devam`, recursive functions, file I/O (`oku`/`yazBaytlar`/`dosyaAc/Oku/Yaz/Kapat`), `içe al` (module import — static expansion at compile time), and is complete enough to compile itself.

**Open items:**
- `dene/yakala` (try/catch) — parsed by the lexer, not yet in code generation (clear compile-time error, not silent wrong output).
- Full toolchain still being rebuilt in Tan (Go host was removed): LSP, test runner, structured diagnostics, package manager, API/dependency query. See the leverage roadmap.
- x86-64 Linux only. Other architectures need a new backend.
- No DWARF debug info (`gdb` sees no symbols).
- Naive code generator — no register allocation / DCE. Correct, not fast.

**What never goes away:** the x86-64 instruction set and the Linux syscall ABI. Not dependencies — the language being spoken.

---

## Repository layout

```
TancElf.tan         the self-hosting compiler, written entirely in Tan —
                    compiles itself, byte-identical fixed point proven
TancElf             committed native seed binary (~146 KB, static)
gen1 / gen2 / gen3  bootstrap generations (fixed-point artifacts)
araclar/            developer tooling, written in Tan (simgeler, denetle,
                    bicimlendir; more coming)
kutuphane/          standard library + tooling modules, written in Tan
                    (tanoku.tan = shared tokenizer/scan helpers)
Tanc.tan, Tanc2.tan, TancAsm.tan   earlier Tan-written compiler attempts
Kesim.tan           cutting-stock optimizer (real tool)
Talay.tan           freight index scoring pipeline
Noral.tan           neural network with backpropagation
testler/            test programs
BootstrapGoSuz.sh   self-hosting chain, zero Go
TestArkaUcGoSuzTemiz.sh   clean-room: asserts no go/gcc/clang/as/ld
KanitGoSuzTarihce.sh      rebuilds compiler history from git, zero Go
```

---

## License

MIT — see [LICENSE](LICENSE).

Contributions welcome. Highest-value right now: the developer toolchain being
rebuilt in Tan (LSP, test runner, structured diagnostics, package manager),
`dene/yakala` code generation, or an ARM64 backend.
