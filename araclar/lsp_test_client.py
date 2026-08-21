#!/usr/bin/env python3
"""lsp_test_client.py — TAN LSP sunucusu (tanlsp) için test istemcisi.

stdio üzerinden LSP protokolü ile initialize + documentSymbol + didOpen
(publishDiagnostics) + shutdown + exit gönderir, yanıtları doğrular/yazdırır.

Kullanım:
    ./TancElf araclar/lsp.tan tanlsp
    python3 araclar/lsp_test_client.py <dosya.tan> [./tanlsp]
"""
import subprocess, json, sys, os


def frame(msg: dict) -> bytes:
    body = json.dumps(msg).encode("utf-8")
    return b"Content-Length: " + str(len(body)).encode() + b"\r\n\r\n" + body


def parse_frames(data: bytes):
    i = 0
    while i < len(data):
        j = data.find(b"\r\n\r\n", i)
        if j < 0:
            break
        header = data[i:j].decode("utf-8")
        n = int(header.split("Content-Length:")[1].strip())
        body = data[j + 4:j + 4 + n]
        yield json.loads(body)
        i = j + 4 + n


def main():
    if len(sys.argv) < 2:
        print("Kullanım: lsp_test_client.py <dosya.tan> [tanlsp-yolu]")
        sys.exit(1)
    dosya = os.path.abspath(sys.argv[1])
    sunucu = sys.argv[2] if len(sys.argv) > 2 else "./tanlsp"
    satir = int(sys.argv[3]) if len(sys.argv) > 3 else 0
    kolon = int(sys.argv[4]) if len(sys.argv) > 4 else 0
    uri = "file://" + dosya

    msgs = [
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"capabilities": {}}},
        {"jsonrpc": "2.0", "method": "initialized", "params": {}},
        {"jsonrpc": "2.0", "id": 2, "method": "textDocument/documentSymbol",
         "params": {"textDocument": {"uri": uri}}},
        {"jsonrpc": "2.0", "method": "textDocument/didOpen",
         "params": {"textDocument": {"uri": uri}}},
        {"jsonrpc": "2.0", "id": 4, "method": "textDocument/formatting",
         "params": {"textDocument": {"uri": uri}}},
        {"jsonrpc": "2.0", "id": 5, "method": "textDocument/references",
         "params": {"textDocument": {"uri": uri}, "position": {"line": satir, "character": kolon}}},
        {"jsonrpc": "2.0", "id": 6, "method": "textDocument/completion",
         "params": {"textDocument": {"uri": uri}, "position": {"line": satir, "character": kolon}}},
        {"jsonrpc": "2.0", "id": 3, "method": "shutdown"},
        {"jsonrpc": "2.0", "method": "exit"},
    ]
    inp = b"".join(frame(m) for m in msgs)
    out = subprocess.run([sunucu], input=inp, capture_output=True, timeout=15).stdout

    for obj in parse_frames(out):
        if "id" in obj and obj.get("id") == 1:
            caps = obj["result"]["capabilities"]
            print(f"[initialize] capabilities: {list(caps.keys())}")
        elif obj.get("id") == 2:
            syms = obj["result"]
            print(f"[documentSymbol] {len(syms)} sembol:")
            for s in syms:
                print(f"    {s['name']}  (kind={s['kind']}, satır {s['range']['start']['line']})")
        elif obj.get("method") == "textDocument/publishDiagnostics":
            diags = obj["params"]["diagnostics"]
            print(f"[diagnostics] {len(diags)} tanı:")
            for d in diags:
                sev = {1: "hata", 2: "uyarı", 3: "bilgi"}.get(d["severity"], "?")
                print(f"    {d['code']} [{sev}] satır {d['range']['start']['line']}: {d['message']}")
        elif obj.get("id") == 4:
            edits = obj["result"]
            if edits:
                yeni = edits[0]["newText"]
                print(f"[formatting] {len(edits)} TextEdit, {len(yeni.splitlines())} satır")
                print(yeni[:200])
            else:
                print("[formatting] sonuç yok")
        elif obj.get("id") == 5:
            locs = obj["result"]
            print(f"[references] {len(locs)} konum:")
            for l in locs[:5]:
                print(f"    satır {l['range']['start']['line']}, kolon {l['range']['start']['character']}")
        elif obj.get("id") == 6:
            items = obj["result"]
            print(f"[completion] {len(items)} öğe: {[i['label'] for i in items[:8]]}...")
        elif obj.get("id") == 3:
            print("[shutdown] ok")


if __name__ == "__main__":
    main()
