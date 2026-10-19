// SHELL contém apenas arquivos públicos emitidos pelo build. Nunca armazenar /api.
self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(SHELL)).then(() => self.skipWaiting()))
})
self.addEventListener('activate', event => {
  event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(key => key.startsWith('mwl-shell-') && key !== CACHE).map(key => caches.delete(key)))).then(() => self.clients.claim()))
})
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url)
  if (event.request.method !== 'GET' || url.origin !== self.location.origin || url.pathname === '/api' || url.pathname.startsWith('/api/')) return
  if (event.request.mode === 'navigate') {
    event.respondWith(fetch(event.request).catch(async () => {
      const shell = await (await caches.open(CACHE)).match('/index.html')
      // Servidores podem redirecionar /index.html para /. Navegações não aceitam
      // a flag redirected da resposta armazenada; devolvemos o shell público.
      return new Response(shell.body, { status: shell.status, headers: shell.headers })
    }))
  } else if (SHELL.includes(url.pathname)) {
    event.respondWith(caches.open(CACHE).then(async cache => (await cache.match(url.pathname)) || fetch(event.request)))
  }
})
