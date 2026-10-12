'use strict';
// Design preview only. This allowlist deliberately excludes all business data.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../../..');
const design = 'docs/ui/chat-creation-design-2026-10-09/';
const resources = new Map([
  ['/', [design + 'index-v2.html', 'text/html; charset=utf-8']],
  ['/prototype-v2.css', [design + 'prototype-v2.css', 'text/css; charset=utf-8']],
  ['/prototype-v2.js', [design + 'prototype-v2.js', 'text/javascript; charset=utf-8']],
  ['/previous/', [design + 'index.html', 'text/html; charset=utf-8']],
  ['/prototype.css', [design + 'prototype.css', 'text/css; charset=utf-8']],
  ['/prototype.js', [design + 'prototype.js', 'text/javascript; charset=utf-8']],
  ...['source-han-sans-sc-regular.woff2', 'source-han-sans-sc-medium.woff2', 'inter-latin-variable.woff2'].map(name => ['/frontend/public/fonts/' + name, ['frontend/public/fonts/' + name, 'font/woff2']]),
  ...['creator-studio-poster.jpg', 'creator-studio-last.jpg'].map(name => ['/frontend/public/brand/auth/' + name, ['frontend/public/brand/auth/' + name, 'image/jpeg']]),
  ['/frontend/public/brand/auth/creator-studio-loop.mp4', ['frontend/public/brand/auth/creator-studio-loop.mp4', 'video/mp4']],
]);
const server = http.createServer((req, res) => {
  const entry = resources.get(new URL(req.url, 'http://127.0.0.1').pathname);
  if (!entry || !['GET', 'HEAD'].includes(req.method)) { res.writeHead(404); res.end(); return; }
  const file = path.join(root, entry[0]);
  const size = fs.statSync(file).size;
  const range = req.headers.range?.match(/^bytes=(\d+)-(\d*)$/);
  const start = range ? Number(range[1]) : 0;
  const end = range?.[2] ? Math.min(Number(range[2]), size - 1) : size - 1;
  if (start > end || start >= size) { res.writeHead(416, { 'Content-Range': 'bytes */' + size }); res.end(); return; }
  const headers = { 'Content-Type': entry[1], 'Content-Length': end - start + 1, 'Cache-Control': 'no-store', 'Accept-Ranges': 'bytes', 'X-Content-Type-Options': 'nosniff' };
  if (range) headers['Content-Range'] = `bytes ${start}-${end}/${size}`;
  res.writeHead(range ? 206 : 200, headers);
  if (req.method === 'HEAD') res.end(); else fs.createReadStream(file, { start, end }).pipe(res);
});
server.listen(4199, '127.0.0.1', () => console.log('Sub2Video design-only preview: http://127.0.0.1:4199/'));
