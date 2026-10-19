import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { readFileSync } from 'node:fs'
import { createHash } from 'node:crypto'

export default defineConfig({
  plugins: [react(), {
    name: 'shell-pwa',
    apply: 'build',
    generateBundle(_, bundle) {
      const assets = Object.keys(bundle).filter(name => /\.(js|css)$/.test(name)).map(name => '/' + name)
      const shell = ['/index.html', '/manifest.webmanifest', '/favicon.svg', '/icon-192.png', '/icon-512.png', ...assets]
      const hash = createHash('sha256').update(assets.join('|'))
      for (const path of ['./index.html', './src/service-worker.js', ...shell.filter(path => !path.startsWith('/assets/') && path !== '/index.html').map(path => './public' + path)]) hash.update(readFileSync(new URL(path, import.meta.url)))
      const version = hash.digest('hex').slice(0, 16)
      this.emitFile({ type: 'asset', fileName: 'sw.js', source: `const CACHE='mwl-shell-${version}';\nconst SHELL=${JSON.stringify(shell)};\n` + readFileSync(new URL('./src/service-worker.js', import.meta.url), 'utf8') })
    },
  }],
  server: {
    proxy: { '/api': 'http://127.0.0.1:8080' },
  },
})
