# GÖREV: araclar/lsp_test_client.py yaz (Python LSP test istemcisi)

TAN LSP sunucusunu (./tanlsp binary, stdio üzerinden LSP protokolü) test eden
Python 3 istemcisi. SADECE bu dosyayı oluştur.

## Protokol: LSP over stdio
Her mesaj: `Content-Length: N\r\n\r\n` + N byte UTF-8 JSON gövde.

## İstemci ne yapmalı
1. subprocess ile ./tanlsp başlat (stdin/stdout pipe).
2. Şu mesajları SIRAYLA gönder, her birinden sonra yanıtı oku+yazdır:
   a. initialize (id=1, params={"capabilities":{}}) → sunucu capabilities döndürür
   b. initialized (notification, method="initialized", params={})
   c. textDocument/documentSymbol (id=2, params={"textDocument":{"uri":"file:///DOSYA"}})
      DOSYA = sys.argv[1] (test edilecek .tan dosyası)
   d. shutdown (id=3) → null
   e. exit (notification)
3. Her yanıtı Content-Length çerçevesinden ayrıştır, JSON parse et, güzel yazdır.
4. Sunucu çıkışını bekle.

## Yardımcılar
- gonder(proc, mesaj_dict): JSON encode, Content-Length header ekle, stdin'e yaz+flush.
- oku(proc): stdout'tan Content-Length header oku, N byte gövde oku, JSON parse et döndür.

## Kullanım
python3 araclar/lsp_test_client.py test.tan

Sadece standart kütüphane (subprocess, json, sys). Temiz, çalışan Python 3.
