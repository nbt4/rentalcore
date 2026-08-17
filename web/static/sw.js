const CACHE_NAME = 'rentalcore-app-shell-v1'
const APP_SHELL = [
  '/manifest.json',
  '/static/images/app-icons/icon-192.png',
  '/static/images/app-icons/icon-512.png',
  '/static/images/app-icons/icon-maskable-512.png',
]

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)))
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((names) =>
      Promise.all(names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name))),
    ),
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET' || request.mode === 'navigate') return

  const url = new URL(request.url)
  if (url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return

  const cacheableDestinations = new Set(['font', 'image', 'script', 'style'])
  if (!cacheableDestinations.has(request.destination) && url.pathname !== '/manifest.json') return

  event.respondWith(
    caches.match(request).then((cached) => {
      if (cached) return cached
      return fetch(request).then((response) => {
        if (response.ok) {
          void caches.open(CACHE_NAME).then((cache) => cache.put(request, response.clone()))
        }
        return response
      })
    }),
  )
})
