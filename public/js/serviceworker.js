var CACHE = 'pwauutispuro-cache';
var precacheFiles = [
      '/en',
      '/fi',
      '/en/category/talous/0',
      '/fi/category/talous/0',
      '/en/category/digi/0',
      '/fi/category/digi/0',
      '/en/category/pelit/0',
      '/fi/category/pelit/0',
      '/en/category/terveys/0',
      '/fi/category/terveys/0',
      '/en/category/kotimaa/0',
      '/fi/category/kotimaa/0',
      '/en/category/ulkomaat/0',
      '/fi/category/ulkomaat/0',
      '/en/category/kulttuuri/0',
      '/fi/category/kulttuuri/0',
      '/en/category/urheilu/0',
      '/fi/category/urheilu/0',
      '/en/category/viihde/0',
      '/fi/category/viihde/0',
      '/en/category/elokuvat/0',
      '/fi/category/elokuvat/0',
      '/en/category/tiede/0',
      '/fi/category/tiede/0',
      '/en/category/ruoka/0',
      '/fi/category/ruoka/0',
      '/en/category/matkustus/0',
      '/fi/category/matkustus/0',
      '/en/category/asuminen/0',
      '/fi/category/asuminen/0',
      '/en/category/naisetjamuoti/0',
      '/fi/category/naisetjamuoti/0',
      '/en/category/blogs/0',
      '/fi/category/blogs/0',
      '/serviceworker.js',
      '/public/js/socket.io-1.3.7.js',
      '/public/js/moment.2.10.6.js',
      '/public/css/uutispuro-1511114584.min.css',
      '/public/js/uutispuro-1511114584.min.js',
      '/public/img/manifest.json',
      '/socket.io',
      '/socket.io/'
    ];

self.addEventListener('install', function(evt) {
  console.log('The service worker is being installed.');
  evt.waitUntil(precache().then(function() {
    console.log('[ServiceWorker] Skip waiting on install');
    return self.skipWaiting();
  })
  );
});

//allow sw to control of current page
self.addEventListener('activate', function(event) {
console.log('[ServiceWorker] Claiming clients for current page');
      return self.clients.claim();
});

self.addEventListener('fetch', function(evt) {
  console.log('The service worker is serving the asset: '+ evt.request.url);
  evt.respondWith(fromCache(evt.request).catch(fromServer(evt.request)));
  evt.waitUntil(update(evt.request));
});

function precache() {
  return caches.open(CACHE).then(function (cache) {
    return cache.addAll(precacheFiles);
  });
}

function fromCache(request) {
  //we pull files from the cache first thing so we can show them fast
  return caches.open(CACHE).then(function (cache) {
    return cache.match(request).then(function (matching) {
      return matching || Promise.reject('no-match');
    });
  });
}

function update(request) {
  //this is where we call the server to get the newest version of the 
  //file to use the next time we show view
  return caches.open(CACHE).then(function (cache) {
    return fetch(request).then(function (response) {
      return cache.put(request, response);
    });
  });
}

function fromServer(request) {
  //this is the fallback if it is not in the cache to go to the server and get it
  return fetch(request).then(function(response){ return response; })
}
