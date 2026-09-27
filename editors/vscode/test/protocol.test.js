'use strict';
// End-to-end test for the qklsp client (run directly with Node, no vscode dependency):
//   node test/protocol.test.js
// Requires a built language server: go build -o /tmp/qklsp ./cmd/qklsp (or point QKLSP=... at one)

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
      if (Date.now() - t0 > timeoutMs) return reject(new Error('timed out waiting for: ' + label));
      setTimeout(tick, 20);
    };
    tick();
  });
}

async function main() {
  if (!fs.existsSync(bin)) {
    console.log(`Skipped: language server not found at ${bin} (build it first: go build -o ${bin} ./cmd/qklsp)`);
    return;
  }
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'qklsp-node-'));
  const file = path.join(dir, 'demo.qk');
  const src = [
    '// Doubles.',
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
      if (err || (code && code !== 0)) console.log(`(server exited code=${code} err=${err ? err.message : ''})`);
    },
  }).start();
  client.on('textDocument/publishDiagnostics', (p) => {
    if (p.uri === uri) diag = p.diagnostics;
  });

  const init = await client.initialize('file://' + dir);
  check('initialize returns capabilities', !!(init && init.capabilities && init.capabilities.definitionProvider), JSON.stringify(init).slice(0, 120));

  client.openDocument(uri, src);
  const diags = await waitFor(() => diag, 5000, 'publishDiagnostics');
  const codes = diags.map((d) => d.code).filter(Boolean);
  check('didOpen receives diagnostics (including QK101 unused variable)', codes.includes('QK101'), JSON.stringify(diags));
  check('diagnostics carry a range and a message', diags.every((d) => d.range && d.message), JSON.stringify(diags));

  const defRes = await client.request('textDocument/definition', {
    textDocument: { uri },
    position: { line: 6, character: 18 }, // the `double` in `double(21)` on line 7
  });
  check('definition jumps to line 2', !!defRes && defRes.range && defRes.range.start.line === 1, JSON.stringify(defRes));

  const comp = await client.request('textDocument/completion', {
    textDocument: { uri },
    position: { line: 6, character: 4 },
  });
  const labels = (comp.items || []).map((i) => i.label);
  check('completion includes double / io / while', ['double', 'io', 'while'].every((l) => labels.includes(l)), labels.slice(0, 12).join(','));

  const hover = await client.request('textDocument/hover', {
    textDocument: { uri },
    position: { line: 6, character: 18 },
  });
  check('hover shows signature and doc', !!(hover && hover.contents && /fn double\(int n\) int/.test(hover.contents.value) && /Doubles\./.test(hover.contents.value)), JSON.stringify(hover).slice(0, 120));

  // Diagnostics should be cleared after the edit
  diag = undefined;
  const fixed = src.replace('    int unused = 1;\n', '');
  client.changeDocument(uri, fixed, 2);
  const after = await waitFor(() => diag, 5000, 'didChange diagnostics');
  check('diagnostics cleared after didChange', after.length === 0, JSON.stringify(after));

  await client.stop();
  check('server shuts down gracefully', !client.running);

  console.log(failures === 0 ? '\nall passed' : `\n${failures} check(s) failed`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((err) => {
  console.error('test error:', err.message);
  process.exit(1);
});
