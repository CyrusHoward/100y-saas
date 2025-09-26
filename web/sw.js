// Service Worker for 100y-saas
// Provides offline capability and progressive enhancement

const CACHE_NAME = '100y-saas-v1';
const STATIC_ASSETS = [
  '/web/index.html',
  '/web/styles.css',
  '/web/app.js',
  '/api/ping'
];

// Install event - cache static assets
self.addEventListener('install', event => {
  console.log('Service Worker: Installing...');
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        console.log('Service Worker: Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => self.skipWaiting()) // Activate immediately
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
  console.log('Service Worker: Activating...');
  
  event.waitUntil(
    caches.keys()
      .then(cacheNames => {
        return Promise.all(
          cacheNames.map(cacheName => {
            if (cacheName !== CACHE_NAME) {
              console.log('Service Worker: Deleting old cache', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => self.clients.claim()) // Take control of all clients
  );
});

// Fetch event - serve from cache or network
self.addEventListener('fetch', event => {
  // Only handle GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  const url = new URL(event.request.url);
  
  // Handle API requests with network-first strategy
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirstStrategy(event.request));
    return;
  }
  
  // Handle static assets with cache-first strategy
  if (STATIC_ASSETS.some(asset => url.pathname === asset)) {
    event.respondWith(cacheFirstStrategy(event.request));
    return;
  }
  
  // Handle navigation requests (HTML pages)
  if (event.request.mode === 'navigate') {
    event.respondWith(navigationStrategy(event.request));
    return;
  }
  
  // Default to network for everything else
  event.respondWith(fetch(event.request));
});

// Network-first strategy for API calls
async function networkFirstStrategy(request) {
  try {
    const networkResponse = await fetch(request);
    
    // Only cache successful API responses for certain endpoints
    if (networkResponse.ok && shouldCacheApiResponse(request)) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: Network failed, trying cache', error);
    
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Return offline page for failed API calls
    return new Response(JSON.stringify({
      success: false,
      error: 'You are offline. Please check your connection.',
      offline: true
    }), {
      status: 503,
      statusText: 'Service Unavailable',
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

// Cache-first strategy for static assets
async function cacheFirstStrategy(request) {
  const cachedResponse = await caches.match(request);
  
  if (cachedResponse) {
    return cachedResponse;
  }
  
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('Service Worker: Failed to fetch', request.url, error);
    throw error;
  }
}

// Navigation strategy for HTML pages
async function navigationStrategy(request) {
  try {
    return await fetch(request);
  } catch (error) {
    // Return cached index.html for offline navigation
    const cachedResponse = await caches.match('/web/index.html');
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Fallback offline page
    return new Response(`
      <!DOCTYPE html>
      <html>
        <head>
          <title>Offline - 100y SaaS</title>
          <meta charset="utf-8">
          <meta name="viewport" content="width=device-width, initial-scale=1">
          <style>
            body { 
              font-family: system-ui, sans-serif; 
              max-width: 720px; 
              margin: 2rem auto; 
              padding: 0 1rem; 
              text-align: center;
            }
            .offline-icon { font-size: 4rem; margin: 2rem 0; }
            .retry-btn { 
              background: #007bff; 
              color: white; 
              border: none; 
              padding: 0.75rem 1.5rem; 
              border-radius: 0.25rem; 
              cursor: pointer;
              margin-top: 1rem;
            }
            .retry-btn:hover { background: #0056b3; }
          </style>
        </head>
        <body>
          <div class="offline-icon">📱⚡</div>
          <h1>You're Offline</h1>
          <p>It looks like you're not connected to the internet.</p>
          <p>Some features may not be available until you reconnect.</p>
          <button class="retry-btn" onclick="window.location.reload()">
            Try Again
          </button>
        </body>
      </html>
    `, {
      headers: { 'Content-Type': 'text/html' }
    });
  }
}

// Determine which API responses should be cached
function shouldCacheApiResponse(request) {
  const url = new URL(request.url);
  
  // Cache these API endpoints for offline access
  const cacheableEndpoints = [
    '/api/ping',
    '/api/tenants',
    '/healthz'
  ];
  
  return cacheableEndpoints.some(endpoint => url.pathname === endpoint);
}

// Handle background sync (for when connection is restored)
self.addEventListener('sync', event => {
  console.log('Service Worker: Background sync', event.tag);
  
  if (event.tag === 'background-sync') {
    event.waitUntil(doBackgroundSync());
  }
});

async function doBackgroundSync() {
  // Clear any stale cache entries
  const cache = await caches.open(CACHE_NAME);
  const keys = await cache.keys();
  
  // Remove API cache entries older than 1 hour
  const oneHourAgo = Date.now() - (60 * 60 * 1000);
  
  for (const request of keys) {
    if (request.url.includes('/api/')) {
      const response = await cache.match(request);
      const cacheTime = response.headers.get('sw-cache-time');
      
      if (cacheTime && parseInt(cacheTime) < oneHourAgo) {
        await cache.delete(request);
        console.log('Service Worker: Removed stale cache entry', request.url);
      }
    }
  }
}

// Listen for messages from the main thread
self.addEventListener('message', event => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  
  if (event.data && event.data.type === 'CACHE_UPDATE') {
    // Update cache with new data
    caches.open(CACHE_NAME)
      .then(cache => cache.put(event.data.url, new Response(event.data.data)))
      .then(() => {
        event.ports[0].postMessage({ success: true });
      })
      .catch(error => {
        event.ports[0].postMessage({ success: false, error: error.message });
      });
  }
});
