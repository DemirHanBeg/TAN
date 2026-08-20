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

## Developer toolchain (written in Tan, single binary)

A unified `tan` toolchain — compile once with `bash kur.sh`, then:

```bash
tan simgeler   dosya.tan --json    # symbols (functions/variables)
tan api        dosya.tan --json    # public function signatures
tan bagimlilik dosya.tan --json    # içe al dependency graph
tan ast        dosya.tan --json    # structural parse tree
tan denetle    dosya.tan --json    # linter + AI-native diagnostics (TANxxxx + reason + fix)
tan bicimlendir dosya.tan          # canonical formatter (deterministic, idempotent)
tan kilit      dosya.tan           # SHA-256 content-addressed lockfile (transitive)
tan dogrula    dosya.kilit         # integrity check against lockfile
tan imzala     dosya.tan <key>     # HMAC-SHA256 signature
tan imzadogrula dosya.tan <key> <sig>   # verify signature
tan sürüm                          # toolchain info
```

All self-hosted (zero Go), all `--json` for AI/editor consumption.
`kutuphane/tanoku.tan` = shared tokenizer/scan helpers; `kutuphane/sha256.tan` =
SHA-256 + HMAC (real crypto, matches `sha256sum`). Unit tests: `bash TestAraclar.sh`.
Reproducibility: `bash HermetikDogrula.sh` (proves byte-deterministic builds).

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

**The compiler handles:** int64/float64 arithmetic, bitwise & shift (`& | ^ << >>`), string and list variables (hand-written heap allocator), comparisons, `ve/veya/değil`, `eğer/değilse`, `iken`, `her...içinde`, `dur/devam`, recursive functions, file I/O (`oku`/`yazBaytlar`/`dosyaAc/Oku/Yaz/Kapat`), `içe al` (module import — static expansion at compile time), and is complete enough to compile itself. Bitwise support enables real crypto in pure Tan — see `kutuphane/sha256.tan` (SHA-256 + HMAC, matches `sha256sum`).

**Open items:**
- `dene/yakala` (try/catch) — parsed by the lexer, not yet in code generation (clear compile-time error, not silent wrong output).
- Toolchain built in Tan (Go host removed): the query/lint/format/lockfile/signing commands are done; a persistent **LSP protocol server** (needs a stdin builtin + JSON-RPC loop), a debugger, and a remote package registry are the remaining larger subsystems.
- Weak static return-type inference: a function returning text may need `metinBirlestir("", f(x))` at the call site to print as text.
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
