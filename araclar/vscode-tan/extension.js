// extension.js — TAN LSP istemcisi (tanlsp'yi VS Code'a bağlar).
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

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
}

function deactivate() {
  if (!client) return undefined;
  return client.stop();
}

module.exports = { activate, deactivate };
