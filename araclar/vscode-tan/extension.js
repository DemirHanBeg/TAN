// extension.js — TAN LSP istemcisi (tanlsp'yi VS Code'a bağlar).
const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

// workspace/symbol: tanlsp sunucusu tek-dosya (documentSymbol) ötesine
// geçmiyor — dizin listeleme için yeni bir TancElf syscall builtin'i
// gerektirir (self-hosting'e risk). Bunun yerine VS Code'un kendi dosya
// taramasını (findFiles) kullanıp her .tan dosyası için VAR OLAN
// documentSymbol'ü çağırıyoruz — sunucuya hiç dokunmadan, workspace
// çapında "Go to Symbol" çalışır hale gelir.
function kayitliIsimEslesiyorMu(ad, sorgu) {
  if (!sorgu) return true;
  return ad.toLowerCase().includes(sorgu.toLowerCase());
}

async function workspaceSembolleriGetir(sorgu) {
  const dosyalar = await vscode.workspace.findFiles('**/*.tan', '**/{node_modules,.git}/**');
  const sonuclar = [];
  for (const uri of dosyalar) {
    let semboller;
    try {
      semboller = await vscode.commands.executeCommand('vscode.executeDocumentSymbolProvider', uri);
    } catch {
      continue;
    }
    if (!semboller) continue;
    for (const s of semboller) {
      if (kayitliIsimEslesiyorMu(s.name, sorgu)) {
        sonuclar.push(new vscode.SymbolInformation(s.name, s.kind, '', new vscode.Location(uri, s.range)));
      }
    }
  }
  return sonuclar;
}

function activate(context) {
  // tanlsp binary PATH'te ya da ayarla değiştirilebilir olmalı.
  const serverOptions = {
    command: 'tanlsp',
    transport: TransportKind.stdio,
  };
  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'tan' }],
  };
  client = new LanguageClient('tanLsp', 'TAN Language Server', serverOptions, clientOptions);
  client.start();

  context.subscriptions.push(
    vscode.languages.registerWorkspaceSymbolProvider({
      provideWorkspaceSymbols: (query) => workspaceSembolleriGetir(query),
    })
  );
}

function deactivate() {
  if (!client) return undefined;
  return client.stop();
}

module.exports = { activate, deactivate };
