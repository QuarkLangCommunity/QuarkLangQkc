'use strict';
// QuarkLang VS Code 扩展：语法高亮（TextMate，见 syntaxes/）+ qklsp 语言服务
// （诊断 / 跳转定义 / 补全 / 悬停 / 文档符号）。
//
// 依赖只有 vscode 与 Node 内置模块——不引入 vscode-languageclient，免 npm 安装。

const vscode = require('vscode');
const path = require('path');
const { LspClient } = require('./protocol');

/** @type {LspClient|null} */
let client = null;
let output = null;
/** @type {vscode.DiagnosticCollection} */
let diagnostics = null;

function toUri(doc) {
  return doc.uri.toString();
}

function toRange(r) {
  return new vscode.Range(r.start.line, r.start.character, r.end.line, r.end.character);
}

function toSeverity(sev) {
  switch (sev) {
    case 1:
      return vscode.DiagnosticSeverity.Error;
    case 2:
      return vscode.DiagnosticSeverity.Warning;
    case 3:
      return vscode.DiagnosticSeverity.Information;
    default:
      return vscode.DiagnosticSeverity.Hint;
  }
}

function publishDiagnostics(params) {
  const uri = vscode.Uri.parse(params.uri);
  const items = (params.diagnostics || []).map((d) => {
    const diag = new vscode.Diagnostic(toRange(d.range), d.message, toSeverity(d.severity));
    diag.source = d.source || 'qklsp';
    if (d.code) diag.code = d.code;
    return diag;
  });
  diagnostics.set(uri, items);
}

function activate(context) {
  output = vscode.window.createOutputChannel('QuarkLang');
  diagnostics = vscode.languages.createDiagnosticCollection('quarklang');
  context.subscriptions.push(output, diagnostics);

  const cfg = vscode.workspace.getConfiguration('quarklang');
  const serverPath = cfg.get('serverPath') || 'qklsp';
  const libDirs = cfg.get('libDirs') || [];
  const args = [];
  for (const dir of libDirs) args.push('-L', dir);

  client = new LspClient(serverPath, args, {
    trace: cfg.get('trace.server') === true,
    onTrace: (line) => output.appendLine(line),
    onExit: (err, code, signal) => {
      if (err) output.appendLine(`qklsp 启动失败: ${err.message}`);
      else if (code) output.appendLine(`qklsp 退出: code=${code} signal=${signal || ''}`);
    },
  }).start();

  client.on('textDocument/publishDiagnostics', publishDiagnostics);

  const rootUri = vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length
    ? vscode.workspace.workspaceFolders[0].uri.toString()
    : null;

  client.initialize(rootUri).catch((err) => {
    vscode.window.showWarningMessage(
      `QuarkLang 语言服务未就绪（${err.message}）。请先构建 qklsp：go build -o qklsp ./cmd/qklsp，` +
      '并在设置 quarklang.serverPath 里指向它。'
    );
  });

  // 已打开的文档同步给服务端
  for (const doc of vscode.workspace.textDocuments) {
    if (doc.languageId === 'quarklang') client.openDocument(toUri(doc), doc.getText());
  }

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((doc) => {
      if (doc.languageId === 'quarklang') client.openDocument(toUri(doc), doc.getText());
    }),
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (e.document.languageId === 'quarklang') client.changeDocument(toUri(e.document), e.document.getText());
    }),
    vscode.workspace.onDidCloseTextDocument((doc) => {
      if (doc.languageId === 'quarklang') {
        client.closeDocument(toUri(doc));
        diagnostics.delete(doc.uri);
      }
    })
  );

  const docParams = (doc) => ({ textDocument: { uri: toUri(doc) } });
  const posParams = (doc, pos) => ({
    textDocument: { uri: toUri(doc) },
    position: { line: pos.line, character: pos.character },
  });

  context.subscriptions.push(
    vscode.languages.registerDefinitionProvider('quarklang', {
      async provideDefinition(doc, pos) {
        const res = await client.request('textDocument/definition', posParams(doc, pos));
        if (!res) return null;
        const list = Array.isArray(res) ? res : [res];
        return list.map((loc) => new vscode.Location(vscode.Uri.parse(loc.uri), toRange(loc.range)));
      },
    }),
    vscode.languages.registerCompletionItemProvider(
      'quarklang',
      {
        async provideCompletionItems(doc, pos) {
          const res = await client.request('textDocument/completion', posParams(doc, pos));
          const items = (res && res.items) || [];
          return items.map((it) => {
            const item = new vscode.CompletionItem(it.label, toCompletionKind(it.kind));
            if (it.detail) item.detail = it.detail;
            return item;
          });
        },
      },
      '.',
      ':'
    ),
    vscode.languages.registerHoverProvider('quarklang', {
      async provideHover(doc, pos) {
        const res = await client.request('textDocument/hover', posParams(doc, pos));
        if (!res || !res.contents) return null;
        const md = new vscode.MarkdownString(
          typeof res.contents === 'string' ? res.contents : res.contents.value
        );
        return new vscode.Hover(md, res.range ? toRange(res.range) : undefined);
      },
    }),
    vscode.languages.registerDocumentSymbolProvider('quarklang', {
      async provideDocumentSymbols(doc) {
        const res = await client.request('textDocument/documentSymbol', docParams(doc));
        return (res || []).map((s) => {
          const sym = new vscode.SymbolInformation(
            s.name,
            toSymbolKind(s.kind),
            '',
            new vscode.Location(vscode.Uri.parse(doc.uri.toString()), toRange(s.range))
          );
          return sym;
        });
      },
    })
  );
}

function toCompletionKind(kind) {
  const K = vscode.CompletionItemKind;
  switch (kind) {
    case 2:
      return K.Method;
    case 3:
      return K.Function;
    case 6:
      return K.Variable;
    case 8:
      return K.Interface;
    case 9:
      return K.Module;
    case 14:
      return K.Keyword;
    case 22:
      return K.Struct;
    case 25:
      return K.TypeParameter;
    default:
      return K.Text;
  }
}

function toSymbolKind(kind) {
  const K = vscode.SymbolKind;
  switch (kind) {
    case 3:
      return K.Function;
    case 8:
      return K.Interface;
    case 9:
      return K.Namespace;
    case 22:
      return K.Struct;
    case 25:
      return K.TypeParameter;
    default:
      return K.Object;
  }
}

async function deactivate() {
  if (client) await client.stop();
  client = null;
}

module.exports = { activate, deactivate };
