# GÖREV: araclar/vscode-tan/ VS Code eklentisi (tanlsp'yi bağlar)

TAN LSP sunucusunu (tanlsp binary, stdio) VS Code'a bağlayan minimal eklenti.
SADECE şu dosyaları oluştur: araclar/vscode-tan/package.json,
araclar/vscode-tan/extension.js, araclar/vscode-tan/language-configuration.json

## package.json
- name: "tan-lsp", displayName: "TAN Language", version: "0.1.0"
- engines.vscode: "^1.75.0"
- activationEvents: ["onLanguage:tan"]
- main: "./extension.js"
- contributes.languages: [{id:"tan", extensions:[".tan"], configuration:"./language-configuration.json"}]
- dependencies: {"vscode-languageclient": "^8.0.0"}

## extension.js
vscode-languageclient kullan. activate(): LanguageClient oluştur, serverOptions =
{ command: "tanlsp", transport: stdio }, documentSelector = [{language:"tan"}].
client.start(). deactivate(): client.stop().

## language-configuration.json
comments.lineComment: "#", brackets: [["(",")"],["[","]"],["{","}"]].

Standart VS Code eklenti yapısı. Temiz, çalışan.
