window.onload = function() {
    var socket = io();
    var lastId;
    socket.on('message', function(msg) {
        var json = JSON.parse(msg);
		if (json.news.length > 0 && lastId !== json.news[0].id) {
            for (var i = 4; i >= 0; i--) {
                if (document.getElementById(json.news[i].id) === null) {
                    var item = makeNode(json, i);
                    prepend(item);
                }
            }
            lastId = json.news[0].id
        }
        json, item = null;
    });
}

var prepend = function(firstElement) {
    var parent = document.getElementById('news-container');
    parent.insertBefore(firstElement, parent.firstChild);
    parent.removeChild(parent.lastChild)
}

var makeNode = function(json, i) {
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
    category.setAttribute("class", "category");
	category.innerHTML = json.news[i].category.categoryName;
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
