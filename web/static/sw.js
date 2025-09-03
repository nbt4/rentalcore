// RentalCore Professional - Enhanced Service Worker for PWA
const CACHE_NAME = 'rentalcore-professional-v3.1';
const STATIC_CACHE = 'rentalcore-static-v3.1';
const DYNAMIC_CACHE = 'rentalcore-dynamic-v3.1';
const OFFLINE_CACHE = 'rentalcore-offline-v3.1';

// Files to cache immediately
const STATIC_FILES = [
    '/static/css/app-new.css',
    '/static/js/app.js',
    '/static/images/icon-192.png',
    '/static/images/icon-512.png',
    '/',
    '/analytics',
    '/scan/select',
    '/manifest.json',
    'https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css',
    'https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.2/font/bootstrap-icons.css',
    'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap',
    'https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js',
    'https://cdn.jsdelivr.net/npm/chart.js',
    'https://unpkg.com/@zxing/library@0.20.0/umd/index.min.js'
];

// Critical offline pages
const OFFLINE_PAGES = [
    '/',
    '/analytics',
    '/jobs',
    '/devices',
    '/customers',
    '/scan/select'
];

// API endpoints to cache with network-first strategy
const API_ENDPOINTS = [
    '/api/v1/jobs',
    '/api/v1/devices', 
    '/api/v1/customers',
    '/analytics/revenue',
    '/analytics/equipment',
    '/search/global',
    '/search/suggestions'
];

// Install event - cache static files
self.addEventListener('install', (event) => {
    console.log('Service Worker: Installing...');
    
    event.waitUntil(
        caches.open(STATIC_CACHE)
            .then((cache) => {
                console.log('Service Worker: Caching static files');
                return cache.addAll(STATIC_FILES);
            })
            .then(() => {
                console.log('Service Worker: Static files cached');
                return self.skipWaiting();
            })
            .catch((error) => {
                console.error('Service Worker: Error caching static files', error);
            })
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
    console.log('Service Worker: Activating...');
    
    event.waitUntil(
        caches.keys()
            .then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => {
                        if (cacheName !== STATIC_CACHE && cacheName !== DYNAMIC_CACHE) {
                            console.log('Service Worker: Deleting old cache', cacheName);
                            return caches.delete(cacheName);
                        }
                    })
                );
            })
            .then(() => {
                console.log('Service Worker: Activated');
                return self.clients.claim();
            })
    );
});

// Fetch event - serve cached content and implement caching strategies
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);
    
    // Skip non-GET requests
    if (request.method !== 'GET') {
        return;
    }
    
    // Skip chrome-extension and other protocol requests
    if (!url.protocol.startsWith('http')) {
        return;
    }
    
    event.respondWith(
        handleRequest(request, url)
    );
});

async function handleRequest(request, url) {
    try {
        // Static files - Cache First strategy
        if (isStaticFile(url)) {
            return await cacheFirst(request, STATIC_CACHE);
        }
        
        // API endpoints - Network First strategy
        if (isApiEndpoint(url)) {
            return await networkFirst(request, DYNAMIC_CACHE);
        }
        
        // Scanner pages - Stale While Revalidate strategy
        if (isScannerPage(url)) {
            return await staleWhileRevalidate(request, DYNAMIC_CACHE);
        }
        
        // Other pages - Network First strategy
        return await networkFirst(request, DYNAMIC_CACHE);
        
    } catch (error) {
        console.error('Service Worker: Error handling request', error);
        
        // Fallback to cache if available
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        
        // Return a basic offline page if nothing else works
        return new Response('Offline - Please check your connection', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'text/plain' }
        });
    }
}

// Cache First strategy - good for static assets
async function cacheFirst(request, cacheName) {
    const cachedResponse = await caches.match(request);
    
    if (cachedResponse) {
        console.log('Service Worker: Serving from cache', request.url);
        return cachedResponse;
    }
    
    // Not in cache, fetch from network and cache it
    try {
        const networkResponse = await fetch(request);
        const cache = await caches.open(cacheName);
        
        // Only cache successful responses
        if (networkResponse.status === 200) {
            cache.put(request, networkResponse.clone());
        }
        
        return networkResponse;
    } catch (error) {
        console.error('Service Worker: Network request failed', error);
        throw error;
    }
}

// Network First strategy - good for API data
async function networkFirst(request, cacheName) {
    try {
        const networkResponse = await fetch(request);
        
        // Cache successful responses
        if (networkResponse.status === 200) {
            const cache = await caches.open(cacheName);
            cache.put(request, networkResponse.clone());
        }
        
        return networkResponse;
    } catch (error) {
        console.log('Service Worker: Network failed, trying cache', request.url);
        
        const cachedResponse = await caches.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
        
        throw error;
    }
}

// Stale While Revalidate strategy - good for scanner pages
async function staleWhileRevalidate(request, cacheName) {
    const cachedResponse = await caches.match(request);
    
    // Fetch from network in background
    const networkResponsePromise = fetch(request)
        .then((networkResponse) => {
            if (networkResponse.status === 200) {
                const cache = caches.open(cacheName);
                cache.then(c => c.put(request, networkResponse.clone()));
            }
            return networkResponse;
        })
        .catch((error) => {
            console.log('Service Worker: Background fetch failed', error);
        });
    
    // Return cached version immediately if available
    if (cachedResponse) {
        console.log('Service Worker: Serving stale content', request.url);
        return cachedResponse;
    }
    
    // If no cache, wait for network
    return networkResponsePromise;
}

// Helper functions
function isStaticFile(url) {
    return url.pathname.startsWith('/static/') || 
           url.hostname === 'cdn.jsdelivr.net' ||
           url.hostname === 'unpkg.com';
}

function isApiEndpoint(url) {
    return url.pathname.startsWith('/api/') ||
           API_ENDPOINTS.some(endpoint => url.pathname.startsWith(endpoint));
}

function isScannerPage(url) {
    return url.pathname.startsWith('/scan/') ||
           url.pathname === '/' ||
           url.pathname.startsWith('/jobs') ||
           url.pathname.startsWith('/devices');
}

// Background sync for offline form submissions
self.addEventListener('sync', (event) => {
    if (event.tag === 'background-sync') {
        console.log('Service Worker: Background sync triggered');
        event.waitUntil(doBackgroundSync());
    }
});

async function doBackgroundSync() {
    // Handle any pending offline form submissions
    // This would typically involve reading from IndexedDB and retrying failed requests
    console.log('Service Worker: Processing background sync');
}

// Push notifications (for future use)
self.addEventListener('push', (event) => {
    if (event.data) {
        const data = event.data.json();
        console.log('Service Worker: Push notification received', data);
        
        const options = {
            body: data.body,
            icon: '/static/images/icon-192.png',
            badge: '/static/images/icon-192.png',
            data: data.data
        };
        
        event.waitUntil(
            self.registration.showNotification(data.title, options)
        );
    }
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    
    if (event.notification.data && event.notification.data.url) {
        event.waitUntil(
            clients.openWindow(event.notification.data.url)
        );
    }
});