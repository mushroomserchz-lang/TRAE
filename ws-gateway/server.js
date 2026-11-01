/**
 * WebSocket ↔ VNC(TCP) 网关（websockify 的最小实现）
 * - GET /healthz            健康检查
 * - WS  /websockify         WebSocket 连接桥接到 VNC_TARGET
 * 环境变量：
 *   PORT         监听端口（默认 8000）
 *   VNC_TARGET   VNC 服务地址，host:port（默认 127.0.0.1:5900）
 *   WS_PATH      WebSocket 路径（默认 /websockify，留空则接受任意路径）
 *   BASIC_AUTH   可选，user:pass，启用 HTTP Basic 保护
 */
const http = require('http');
const net = require('net');
const crypto = require('crypto');

const PORT = Number(process.env.PORT || 8000);
const TARGET = process.env.VNC_TARGET || '127.0.0.1:5900';
const WS_PATH = (process.env.WS_PATH === undefined ? '/websockify' : process.env.WS_PATH);
const AUTH = process.env.BASIC_AUTH || '';

const [TARGET_HOST, TARGET_PORT] = (() => {
  const i = TARGET.lastIndexOf(':');
  return [TARGET.slice(0, i), Number(TARGET.slice(i + 1))];
})();

function authorized(req) {
  if (!AUTH) return true;
  const h = req.headers.authorization || '';
  if (!h.startsWith('Basic ')) return false;
  const got = Buffer.from(h.slice(6), 'base64').toString();
  const want = AUTH;
  const a = Buffer.from(got), b = Buffer.from(want);
  return a.length === b.length && crypto.timingSafeEqual(a, b);
}

const server = http.createServer((req, res) => {
  if (!authorized(req)) {
    res.writeHead(401, { 'WWW-Authenticate': 'Basic realm="ws-gateway"' });
    return res.end('unauthorized');
  }
  if (req.url === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    return res.end(JSON.stringify({ ok: true, target: TARGET, ws_path: WS_PATH || '(any)' }));
  }
  res.writeHead(200, { 'Content-Type': 'text/plain; charset=utf-8' });
  res.end([
    'VNC WebSocket gateway is running.',
    'WebSocket URL: wss://<this-host>' + (WS_PATH || '/websockify'),
    'Backend target: ' + TARGET,
  ].join('\n'));
});

server.on('upgrade', (req, socket) => {
  if (!authorized(req)) { socket.destroy(); return; }
  if (WS_PATH && req.url.split('?')[0] !== WS_PATH) { socket.destroy(); return; }
  const key = req.headers['sec-websocket-key'];
  if (!key) { socket.destroy(); return; }

  const accept = crypto.createHash('sha1')
    .update(key + '258EAFA5-E914-47DA-95CA-C5AB0DC85B11')
    .digest('base64');

  socket.write(
    'HTTP/1.1 101 Switching Protocols\r\n' +
    'Upgrade: websocket\r\nConnection: Upgrade\r\n' +
    'Sec-WebSocket-Accept: ' + accept + '\r\n\r\n'
  );

  const tcp = net.connect(TARGET_PORT, TARGET_HOST);
  tcp.on('error', () => cleanup());
  tcp.on('close', () => cleanup());
  socket.on('error', () => cleanup());
  socket.on('close', () => cleanup());

  let closed = false;
  function cleanup() {
    if (closed) return;
    closed = true;
    try { tcp.destroy(); } catch {}
    try { socket.destroy(); } catch {}
  }

  // TCP -> WS
  tcp.on('data', (buf) => {
    if (!closed) socket.write(encodeFrame(0x2, buf));
  });

  // WS -> TCP（支持分片累积与续航）
  let pending = Buffer.alloc(0);
  socket.on('data', (chunk) => {
    pending = Buffer.concat([pending, chunk]);
    while (true) {
      const parsed = parseFrame(pending);
      if (!parsed) break;
      pending = pending.slice(parsed.size);
      const { opcode, payload, fin } = parsed;
      if (opcode === 0x8) { cleanup(); return; }
      if (opcode === 0x9) { socket.write(encodeFrame(0xA, payload)); continue; }
      if (opcode === 0xA) continue;
      if (opcode === 0x1 || opcode === 0x2 || opcode === 0x0) {
        if (payload.length) tcp.write(payload);
      }
    }
  });
});

function encodeFrame(opcode, payload) {
  const len = payload.length;
  let header;
  if (len < 126) {
    header = Buffer.from([0x80 | opcode, len]);
  } else if (len < 65536) {
    header = Buffer.alloc(4);
    header[0] = 0x80 | opcode; header[1] = 126; header.writeUInt16BE(len, 2);
  } else {
    header = Buffer.alloc(10);
    header[0] = 0x80 | opcode; header[1] = 127;
    header.writeUInt32BE(0, 2); header.writeUInt32BE(len, 6);
  }
  return Buffer.concat([header, payload]);
}

function parseFrame(buf) {
  if (buf.length < 2) return null;
  const fin = (buf[0] & 0x80) !== 0;
  const opcode = buf[0] & 0x0f;
  const masked = (buf[1] & 0x80) !== 0;
  let len = buf[1] & 0x7f;
  let offset = 2;
  if (len === 126) {
    if (buf.length < 4) return null;
    len = buf.readUInt16BE(2); offset = 4;
  } else if (len === 127) {
    if (buf.length < 10) return null;
    len = Number(buf.readBigUInt64BE(2)); offset = 10;
  }
  const maskLen = masked ? 4 : 0;
  if (buf.length < offset + maskLen + len) return null;
  const mask = masked ? buf.slice(offset, offset + 4) : null;
  offset += maskLen;
  const raw = buf.slice(offset, offset + len);
  const payload = masked ? Buffer.from(raw.map((b, i) => b ^ mask[i & 3])) : raw;
  return { fin, opcode, payload, size: offset + len };
}

server.listen(PORT, '0.0.0.0', () => {
  console.log('gateway listening on 0.0.0.0:' + PORT + ' -> ' + TARGET);
});
