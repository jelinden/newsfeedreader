window.onload = function() {
  if (location.pathname === "/fi" || location.pathname === "/en") {
    var lang = (location.pathname === "/fi") ? "fi" : "en";
    var socket = io();
    var lastId;
    socket.on('message', function(msg) {
      var json = JSON.parse(msg);
      if (json.news.length > 0 && lastId !== json.news[0].id) {
        for (var i = 4; i >= 0; i--) {
          if (document.getElementById(json.news[i].id) === null) {
            var item = makeNode(json, i, lang);
            prepend(item);
          }
        }
        lastId = json.news[0].id
      }
      json, item = null;
    });
  }
}

var prepend = function(firstElement) {
  var parent = document.getElementById('news-container');
  parent.insertBefore(firstElement, parent.firstChild);
  parent.removeChild(parent.lastChild)
}

var makeNode = function(json, i, lang) {
  var linkCatName = json.news[i].category.categoryName.toLowerCase();
  var catName = (lang === "en") ?
    json.news[i].category.categoryEnName :
    json.news[i].category.categoryName;

  var item = document.createElement("div");
  item.setAttribute("class", "item new");
  var source = document.createElement("div");
  source.setAttribute("class", "source");
  source.innerHTML = json.news[i].rssSource;
  var date = document.createElement("div");
  date.setAttribute("class", "date");
  date.innerHTML = moment(json.news[i].pubDate).format("DD.MM. HH:mm");
  var link = document.createElement("div");
  link.setAttribute("class", "link");
  var category = document.createElement("div");
  var categoryLink = document.createElement("a");
  categoryLink.setAttribute("href", "/" + lang + "/category/" + linkCatName + "/0")
  category.setAttribute("class", "category");
  categoryLink.innerHTML = catName;
  category.appendChild(categoryLink);
  var a = document.createElement("a");
  a.setAttribute("id", json.news[i].id);
  a.setAttribute("href", json.news[i].rssLink);
  a.setAttribute("target", "_blank");
  a.innerHTML = json.news[i].rssTitle;
  link.appendChild(a);
  item.appendChild(date);
  item.appendChild(source);
  item.appendChild(category);
  item.appendChild(link);
  return item;
}

var xmlhttp = new XMLHttpRequest();

function saveClick(id) {
  xmlhttp.open("GET", "/api/click/" + id, true);
  xmlhttp.send();
}

document.addEventListener('DOMContentLoaded', function() {
  var item = document.getElementsByClassName("itemClick");
  for (var i = 0; i < item.length; i++) {
    item[i].addEventListener("click", function() {
      saveClick(this.id);
    });
  }
});

(function(window, document) {
  var layout = document.getElementById('layout'),
    menu = document.getElementById('menu'),
    menuLink = document.getElementById('menuLink');

  function toggleClass(element, className) {
    var classes = element.className.split(/\s+/),
      length = classes.length;

    for (var i = 0; i < length; i++) {
      if (classes[i] === className) {
        classes.splice(i, 1);
        break;
      }
    }
    if (length === classes.length) {
      classes.push(className);
    }

    element.className = classes.join(' ');
  }

  menuLink.onclick = function(e) {
    var active = 'active';
    e.preventDefault();
    toggleClass(layout, active);
    toggleClass(menu, active);
    toggleClass(menuLink, active);
  };
}(this, this.document));

if (navigator.serviceWorker.controller) {
  console.log('[PWA Builder] active service worker found, no need to register')
} else {
  navigator.serviceWorker.register('serviceworker.js', {
    scope: '/'
  }).then(function(reg) {
    console.log('Service worker has been registered for scope:'+ reg.scope);
  });
}
