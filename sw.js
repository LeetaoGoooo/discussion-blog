// Service Worker for caching and offline capabilities
const CACHE_NAME = 'blog-cache-v1';
const DATA_CACHE_NAME = 'blog-data-cache-v1';
const OFFLINE_URL = '/offline.html';

const urlsToCache = [
  '/',
  '/static/css',
  '/favicon',
  OFFLINE_URL
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('Opened cache');
        return cache.addAll(urlsToCache);
      })
  );
  // Force the waiting service worker to become the active service worker
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  const cacheWhitelist = [CACHE_NAME, DATA_CACHE_NAME];
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheWhitelist.indexOf(cacheName) === -1) {
              return caches.delete(cacheName);
            }
          })
        );
      })
  );
  // Claim clients for current service worker
  return self.clients.claim();
});

// Fetch event - implement different caching strategies
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Handle navigation requests (HTML pages)
  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request)
        .catch(() => {
          // If fetch fails, return offline page
          return caches.match(OFFLINE_URL);
        })
    );
    return;
  }
  
  // Strategy for API requests - Network First with cache fallback
  if (url.pathname.startsWith('/posts') || 
      url.pathname.startsWith('/category') || 
      url.pathname.startsWith('/post') ||
      url.pathname.startsWith('/tags') ||
      url.pathname.startsWith('/search')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // Clone the response because it's a stream that can only be consumed once
          const responseToCache = response.clone();
          
          // Cache successful responses
          if (response.status === 200) {
            caches.open(DATA_CACHE_NAME)
              .then((cache) => {
                cache.put(request, responseToCache);
              });
          }
          
          return response;
        })
        .catch(() => {
          // If network fails, try cache
          return caches.match(request);
        })
    );
    return;
  }
  
  // Strategy for static assets - Cache First
  if (url.pathname === '/static/css' || 
      url.pathname === '/favicon' || 
      url.pathname === '/' ||
      url.pathname === OFFLINE_URL) {
    event.respondWith(
      caches.match(request)
        .then((response) => {
          // Return cached response if found
          if (response) {
            return response;
          }
          
          // Clone the request because it's a stream that can only be consumed once
          const fetchRequest = request.clone();
          
          return fetch(fetchRequest)
            .then((response) => {
              // Check if we received a valid response
              if (!response || response.status !== 200 || response.type !== 'basic') {
                return response;
              }
              
              // Clone the response because it's a stream that can only be consumed once
              const responseToCache = response.clone();
              
              caches.open(CACHE_NAME)
                .then((cache) => {
                  cache.put(request, responseToCache);
                });
              
              return response;
            });
        })
    );
    return;
  }
  
  // For all other requests, try network first, then cache
  event.respondWith(
    fetch(request)
      .catch(() => {
        return caches.match(request);
      })
  );
});

// Handle background sync events
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-posts') {
    event.waitUntil(syncPosts());
  }
});

// Background sync function
function syncPosts() {
  // This is where you would implement syncing unsaved data
  // For example, sending queued requests when connection is restored
  return Promise.resolve();
}

// Handle push notifications
self.addEventListener('push', (event) => {
  const title = 'New Content Available';
  const options = {
    body: 'Check out the latest posts on the blog!',
    icon: '/favicon',
    badge: '/favicon'
  };
  
  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  event.waitUntil(
    clients.openWindow('/')
  );
});