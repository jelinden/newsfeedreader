self.addEventListener('install', e => {
 e.waitUntil(
   caches.open('uutispuro-pwa-cache').then(cache => {
     return cache.addAll([
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
        '/public/js/socket.io-1.3.7.js',
        '/public/js/moment.2.10.6.js',
        '/public/css/uutispuro-1501267239.min.css',
        '/public/js/uutispuro-1501267239.min.js',
        '/public/img/manifest.json'
     ]);
   })
 );
});