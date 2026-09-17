'use strict';
// qklsp 客户端联调测试（Node 直跑，不依赖 vscode）：
//   node test/protocol.test.js
// 需要先构建语言服务器：go build -o /tmp/qklsp ./cmd/qklsp（或用 QKLSP=... 指定路径）

const fs = require('fs');
const os = require('os');
const path = require('path');
const { LspClient } = require('../src/protocol');

const bin = process.env.QKLSP || '/tmp/qklsp';
let failures = 0;

function check(name, ok, detail) {
  console.log(`${ok ? '✓' : '✗'} ${name}${ok ? '' : '  → ' + (detail || '')}`);
  if (!ok) failures++;
}

function waitFor(fn, timeoutMs, label) {
  return new Promise((resolve, reject) => {
    const t0 = Date.now();
    const tick = () => {
      const v = fn();
      if (v) return resolve(v);
      if (Date.now() - t0 > timeoutMs) return reject(new Error('等待超时: ' + label));
      setTimeout(tick, 20);
    };
    tick();
  });
}

async function main() {
  if (!fs.existsSync(bin)) {
    console.log(`跳过：未找到语言服务器 ${bin}（先 go build -o ${bin} ./cmd/qklsp）`);
    return;
  }
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qklsp-node-'));
  const file = path.join(dir, 'demo.qk');
  const src = [
    '// 翻倍。',
    'fn double(int n) int {',
    '    return n * 2;',
    '}',
    'fn main(IOStream io) void {',
    '    int unused = 1;',
    '    io.println(double(21));',
    '}',
    '',
  ].join('\n');
  fs.writeFileSync(file, src);
  const uri = 'file://' + file;

  let diag = null;
  const client = new LspClient(bin, [], {
    onExit: (err, code) => {
      if (err || (code && code !== 0)) console.log(`（服务端退出 code=${code} err=${err ? err.message : ''}）`);
    },
  }).start();
  client.on('textDocument/publishDiagnostics', (p) => {
    if (p.uri === uri) diag = p.diagnostics;
  });

  const init = await client.initialize('file://' + dir);
  check('initialize 返回能力', !!(init && init.capabilities && init.capabilities.definitionProvider), JSON.stringify(init).slice(0, 120));

  client.openDocument(uri, src);
  const diags = await waitFor(() => diag, 5000, 'publishDiagnostics');
  const codes = diags.map((d) => d.code).filter(Boolean);
  check('didOpen 收到诊断（含 QK101 未使用变量）', codes.includes('QK101'), JSON.stringify(diags));
  check('诊断带范围与消息', diags.every((d) => d.range && d.message), JSON.stringify(diags));

  const defRes = await client.request('textDocument/definition', {
    textDocument: { uri },
    position: { line: 6, character: 18 }, // 第 7 行 `double(21)` 的 double
  });
  check('定义跳转到第 2 行', !!defRes && defRes.range && defRes.range.start.line === 1, JSON.stringify(defRes));

  const comp = await client.request('textDocument/completion', {
    textDocument: { uri },
    position: { line: 6, character: 4 },
  });
  const labels = (comp.items || []).map((i) => i.label);
  check('补全含 double / io / while', ['double', 'io', 'while'].every((l) => labels.includes(l)), labels.slice(0, 12).join(','));

  const hover = await client.request('textDocument/hover', {
    textDocument: { uri },
    position: { line: 6, character: 18 },
  });
  check('悬停显示签名与文档', !!(hover && hover.contents && /fn double\(int n\) int/.test(hover.contents.value) && /翻倍/.test(hover.contents.value)), JSON.stringify(hover).slice(0, 120));

  // 修改后诊断应清空
  diag = undefined;
  const fixed = src.replace('    int unused = 1;\n', '');
  client.changeDocument(uri, fixed, 2);
  const after = await waitFor(() => diag, 5000, 'didChange 诊断');
  check('didChange 后诊断清空', after.length === 0, JSON.stringify(after));

  await client.stop();
  check('服务端优雅退出', !client.running);

  console.log(failures === 0 ? '\n全部通过' : `\n${failures} 项失败`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((err) => {
  console.error('测试异常:', err.message);
  process.exit(1);
});
