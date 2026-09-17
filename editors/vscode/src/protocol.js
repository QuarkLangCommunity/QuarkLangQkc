'use strict';
// QuarkLang LSP 客户端（零 npm 依赖）：JSON-RPC over stdio 的收发与生命周期。
// 本文件不依赖 vscode，可单独用 node 测试（见 test/protocol.test.js）。

const cp = require('child_process');

class LspClient {
  /**
   * @param {string} command  qklsp 可执行文件
   * @param {string[]} args   额外参数（如 -L dir）
   */
  constructor(command, args = [], opts = {}) {
    this.command = command;
    this.args = args;
    this._id = 0;
    this._pending = new Map();
    this._handlers = new Map(); // method → [fn]
    this._buf = Buffer.alloc(0);
    this._proc = null;
    this._trace = !!opts.trace;
    this._onTrace = opts.onTrace || (() => {});
    this._onExit = opts.onExit || (() => {});
  }

  start() {
    this._proc = cp.spawn(this.command, this.args, { stdio: ['pipe', 'pipe', 'pipe'] });
    this._proc.stdout.on('data', (chunk) => this._onData(chunk));
    this._proc.stderr.on('data', (chunk) => this._onTrace('stderr: ' + chunk.toString()));
    this._proc.on('error', (err) => this._onExit(err));
    this._proc.on('exit', (code, signal) => this._onExit(null, code, signal));
    return this;
  }

  get running() {
    return !!this._proc && this._proc.exitCode === null;
  }

  _onData(chunk) {
    this._buf = Buffer.concat([this._buf, chunk]);
    for (;;) {
      const sep = this._buf.indexOf('\r\n\r\n');
      if (sep < 0) return;
      const header = this._buf.slice(0, sep).toString('utf8');
      const m = /Content-Length:\s*(\d+)/i.exec(header);
      if (!m) {
        this._buf = this._buf.slice(sep + 4);
        continue;
      }
      const len = parseInt(m[1], 10);
      const bodyStart = sep + 4;
      if (this._buf.length < bodyStart + len) return;
      const body = this._buf.slice(bodyStart, bodyStart + len).toString('utf8');
      this._buf = this._buf.slice(bodyStart + len);
      let msg;
      try {
        msg = JSON.parse(body);
      } catch (e) {
        this._onTrace('解析报文失败: ' + e.message);
        continue;
      }
      this._dispatch(msg);
    }
  }

  _dispatch(msg) {
    if (this._trace) this._onTrace('收到: ' + JSON.stringify(msg).slice(0, 400));
    if (msg.id !== undefined && this._pending.has(msg.id)) {
      const { resolve, reject } = this._pending.get(msg.id);
      this._pending.delete(msg.id);
      if (msg.error) reject(new Error(msg.error.message || 'LSP 错误'));
      else resolve(msg.result);
      return;
    }
    if (msg.method) {
      const fns = this._handlers.get(msg.method) || [];
      for (const fn of fns) fn(msg.params);
    }
  }

  _write(obj) {
    if (!this.running) return;
    const data = Buffer.from(JSON.stringify(obj), 'utf8');
    const head = Buffer.from(`Content-Length: ${data.length}\r\n\r\n`, 'ascii');
    if (this._trace) this._onTrace('发送: ' + JSON.stringify(obj).slice(0, 400));
    this._proc.stdin.write(Buffer.concat([head, data]));
  }

  /** 发通知（无响应）。 */
  notify(method, params) {
    this._write({ jsonrpc: '2.0', method, params });
  }

  /** 发请求，返回 Promise。 */
  request(method, params) {
    const id = ++this._id;
    return new Promise((resolve, reject) => {
      this._pending.set(id, { resolve, reject });
      this._write({ jsonrpc: '2.0', id, method, params });
      setTimeout(() => {
        if (this._pending.has(id)) {
          this._pending.delete(id);
          reject(new Error(`${method} 超时`));
        }
      }, 10000);
    });
  }

  /** 注册服务端通知处理。 */
  on(method, fn) {
    if (!this._handlers.has(method)) this._handlers.set(method, []);
    this._handlers.get(method).push(fn);
    return this;
  }

  initialize(rootUri) {
    return this.request('initialize', {
      processId: process.pid,
      rootUri: rootUri || null,
      capabilities: {
        textDocument: {
          synchronization: { dynamicRegistration: false },
          publishDiagnostics: { relatedInformation: true },
        },
      },
      clientInfo: { name: 'quarklang-vscode' },
    }).then((res) => {
      this.notify('initialized', {});
      return res;
    });
  }

  openDocument(uri, text, languageId = 'quarklang', version = 1) {
    this.notify('textDocument/didOpen', {
      textDocument: { uri, languageId, version, text },
    });
  }

  changeDocument(uri, text, version = 2) {
    this.notify('textDocument/didChange', {
      textDocument: { uri, version },
      contentChanges: [{ text }],
    });
  }

  closeDocument(uri) {
    this.notify('textDocument/didClose', { textDocument: { uri } });
  }

  /** 优雅退出：shutdown → exit → 必要时 kill。 */
  async stop() {
    if (!this.running) return;
    try {
      await this.request('shutdown', null);
    } catch (e) {
      /* 忽略：进程可能已退出 */
    }
    this.notify('exit', null);
    await new Promise((resolve) => {
      const t = setTimeout(() => {
        if (this.running) this._proc.kill();
        resolve();
      }, 1000);
      this._proc.once('exit', () => {
        clearTimeout(t);
        resolve();
      });
    });
  }
}

module.exports = { LspClient };
