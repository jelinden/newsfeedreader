(function () {
	"use strict";

	var wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
	var THEME_KEY = "uutispuro-theme";
	var themeToggle = null;
	var lastId = null;
	var reconnectTimer = null;

	function pad2(n) {
		return (n < 10 ? "0" : "") + n;
	}

	// Replaces moment().format("DD.MM. HH:mm") without the ~50kb dependency.
	function formatDate(value) {
		var d = new Date(value);
		if (isNaN(d.getTime())) {
			return "";
		}
		return pad2(d.getDate()) + "." + pad2(d.getMonth() + 1) + ". " + pad2(d.getHours()) + ":" + pad2(d.getMinutes());
	}

	function makeNode(item, lang) {
		var slug = item.category.categoryName.toLowerCase();
		var categoryName = lang === "en" ? item.category.categoryEnName : item.category.categoryName;

		var el = document.createElement("div");
		el.className = "item new";

		var date = document.createElement("div");
		date.className = "date";
		date.textContent = formatDate(item.pubDate);

		var source = document.createElement("div");
		source.className = "source";
		source.textContent = item.rssSource;

		var category = document.createElement("div");
		category.className = "category";
		var categoryLink = document.createElement("a");
		categoryLink.href = "/" + lang + "/category/" + slug + "/0";
		categoryLink.textContent = categoryName;
		category.appendChild(categoryLink);

		var link = document.createElement("div");
		link.className = "link";
		var a = document.createElement("a");
		a.className = "itemClick";
		a.id = item.id;
		a.href = item.rssLink;
		a.target = "_blank";
		a.rel = "noopener noreferrer";
		a.textContent = item.rssTitle;
		link.appendChild(a);

		el.appendChild(date);
		el.appendChild(source);
		el.appendChild(category);
		el.appendChild(link);
		return el;
	}

	// Keep the last page full: drop the oldest item for every new one.
	function prepend(el) {
		var container = document.getElementById("news-container");
		if (!container) {
			return;
		}
		container.insertBefore(el, container.firstChild);
		if (container.children.length > 1) {
			container.removeChild(container.lastElementChild);
		}
	}

	function handleMessage(data, lang) {
		var json;
		try {
			json = JSON.parse(data);
		} catch (e) {
			console.log("ws: ignoring malformed payload", e);
			return;
		}
		var news = json && json.news;
		if (!news || news.length === 0 || news[0].id === lastId) {
			return;
		}
		for (var i = Math.min(news.length - 1, 4); i >= 0; i--) {
			if (document.getElementById(news[i].id) === null) {
				prepend(makeNode(news[i], lang));
			}
		}
		lastId = news[0].id;
	}

	// The live feed only runs on the front pages.
	function startWS(lang) {
		var url = wsProtocol + "://" + window.location.hostname + ":" + window.location.port + "/ws/" + lang;
		var socket = new WebSocket(url, wsProtocol);

		socket.onmessage = function (msg) {
			if (msg.data) {
				handleMessage(msg.data, lang);
			}
		};
		socket.onopen = function () {
			console.log("ws socket open");
			if (reconnectTimer) {
				clearTimeout(reconnectTimer);
				reconnectTimer = null;
			}
		};
		socket.onclose = function () {
			console.log("ws socket closed");
			if (!reconnectTimer) {
				reconnectTimer = setTimeout(function () {
					reconnectTimer = null;
					if (document.visibilityState !== "hidden") {
						startWS(lang);
					}
				}, 15000);
			}
		};
		socket.onerror = function () {
			console.log("ws socket error");
			socket.close();
		};
	}

	// Click beacons. navigator.sendBeacon survives the tab being closed before
	// the request finishes, which a plain XHR did not.
	function saveClick(id) {
		var url = "/api/click/" + encodeURIComponent(id);
		if (!navigator.sendBeacon || !navigator.sendBeacon(url)) {
			var img = new Image();
			img.src = url;
		}
	}

	function trackClicks() {
		var items = document.getElementsByClassName("itemClick");
		for (var i = 0; i < items.length; i++) {
			items[i].addEventListener("click", function () {
				saveClick(this.id);
			});
		}
	}

	// Off canvas navigation on narrow screens.
	function startMenu() {
		var layout = document.getElementById("layout");
		var menu = document.getElementById("menu");
		var toggle = document.getElementById("menuLink");
		var scrim = document.querySelector(".sideScrim");
		if (!layout || !menu || !toggle) {
			return;
		}
		function close() {
			layout.classList.remove("active");
		}
		toggle.addEventListener("click", function (e) {
			e.preventDefault();
			layout.classList.toggle("active");
		});
		var closeButton = menu.querySelector(".menuClose");
		if (closeButton) {
			closeButton.addEventListener("click", function (e) {
				e.preventDefault();
				close();
			});
		}
		if (scrim) {
			scrim.addEventListener("click", function (e) {
				e.preventDefault();
				close();
			});
		}
		document.addEventListener("keydown", function (e) {
			if (e.key === "Escape") {
				close();
			}
		});
	}

	// Light or dark, remembered per browser. The theme_script partial in the
	// document head applies the stored value before the stylesheet paints.
	// persist=false while simply mirroring the operating system, otherwise the
	// first visit would store a choice and stop following further OS changes.
	function applyTheme(theme, persist) {
		document.documentElement.setAttribute("data-theme", theme);
		if (persist) {
			try {
				localStorage.setItem(THEME_KEY, theme);
			} catch (e) {}
		}
		var meta = document.querySelector('meta[name="theme-color"]');
		if (meta) {
			meta.setAttribute("content", theme === "dark" ? "#0f1214" : "#f4f5f7");
		}
		if (themeToggle) {
			themeToggle.setAttribute("aria-pressed", theme === "dark" ? "true" : "false");
		}
	}

	function startThemeToggle() {
		themeToggle = document.getElementById("themeToggle");
		if (!themeToggle) {
			return;
		}
		applyTheme(document.documentElement.getAttribute("data-theme") || "light", false);
		themeToggle.addEventListener("click", function () {
			applyTheme(document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark", true);
		});
		if (!window.matchMedia) {
			return;
		}
		// Keep following the operating system until an explicit choice is stored.
		var query = window.matchMedia("(prefers-color-scheme: dark)");
		var onChange = function (event) {
			var stored = null;
			try {
				stored = localStorage.getItem(THEME_KEY);
			} catch (e) {}
			if (!stored) {
				applyTheme(event.matches ? "dark" : "light", false);
			}
		};
		if (query.addEventListener) {
			query.addEventListener("change", onChange);
		} else if (query.addListener) {
			query.addListener(onChange);
		}
	}

	// Marks the single menu entry that matches the current url closest, so the
	// sidebar shows where you are instead of lighting up every prefix.
	function markCurrentSection() {
		var normalize = function (path) {
			return path.replace(/\/\d+\/?$/, "").replace(/\/+$/, "");
		};
		var current = normalize(location.pathname);
		var links = document.querySelectorAll("#menu a");
		var best = null;
		var bestLength = -1;
		for (var i = 0; i < links.length; i++) {
			var href = normalize(links[i].getAttribute("href") || "");
			if (href === "") {
				continue;
			}
			if ((current === href || current.indexOf(href + "/") === 0) && href.length > bestLength) {
				best = links[i];
				bestLength = href.length;
			}
		}
		if (best) {
			best.parentNode.classList.add("isCurrent");
		}
	}

	function startShortcuts() {
		document.addEventListener("keydown", function (e) {
			var tag = document.activeElement && document.activeElement.tagName;
			if (e.key === "/" && tag !== "INPUT" && tag !== "TEXTAREA") {
				var field = document.getElementById("q");
				if (field) {
					e.preventDefault();
					field.focus();
				}
			}
		});
	}

	window.addEventListener("load", function () {
		var lang = location.pathname === "/fi" || location.pathname === "/en" ? location.pathname.slice(1) : null;
		if (lang) {
			startWS(lang);
		}
	});

	document.addEventListener("DOMContentLoaded", function () {
		trackClicks();
		startThemeToggle();
		startMenu();
		markCurrentSection();
		startShortcuts();
	});
})();
