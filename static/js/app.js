// specd — Client-side logic
// Theme management, navigation state, htmx configuration.

(function () {
  "use strict";

  var THEME_KEY = "specd-theme";
  var SEED_COLOR = "#1c4bea";

  // -- Theme ------------------------------------------------------

  function initTheme() {
    ui("theme", SEED_COLOR);
    var saved = localStorage.getItem(THEME_KEY);
    if (saved === "light" || saved === "dark") {
      ui("mode", saved);
    } else {
      var prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      ui("mode", prefersDark ? "dark" : "light");
    }
    syncThemeRadios();
  }

  // Pre-check the theme radio on the settings page if present.
  function syncThemeRadios() {
    var current = currentTheme();
    document.querySelectorAll('input[name="specd-theme"]').forEach(function (el) {
      el.checked = el.value === current;
    });
  }

  // Apply a theme value ("light" | "dark") and persist it.
  function setTheme(value) {
    if (value !== "light" && value !== "dark") return;
    ui("mode", value);
    localStorage.setItem(THEME_KEY, value);
  }

  // Return the active theme ("light" | "dark") from the body class.
  function currentTheme() {
    return document.body.classList.contains("dark") ? "dark" : "light";
  }

  // -- Navigation active state ------------------------------------

  function syncNavActive() {
    var path = window.location.pathname;
    document.querySelectorAll("[data-nav-link]").forEach(function (el) {
      var href = el.getAttribute("href");
      if (path.startsWith(href)) {
        el.classList.add("active");
      } else {
        el.classList.remove("active");
      }
    });
  }

  // -- htmx configuration ----------------------------------------

  if (typeof htmx !== "undefined") {
    // Disable history cache to avoid localStorage bloat.
    htmx.config.historyCacheSize = 0;

    // After every htmx swap, update nav active state and re-init theme icons.
    document.body.addEventListener("htmx:afterSwap", function () {
      syncNavActive();
      syncThemeRadios();
    });
  }

  // -- Sidebar toggle ---------------------------------------------

  var SIDEBAR_KEY = "specd-sidebar";

  function toggleSidebar() {
    var nav = document.querySelector("nav.left.l");
    if (!nav) return;
    var expanded = nav.classList.toggle("max");
    var btn = nav.querySelector("[aria-label='Toggle sidebar']");
    if (btn) {
      btn.setAttribute("aria-expanded", String(expanded));
      var icon = btn.querySelector("i");
      if (icon) icon.textContent = expanded ? "menu_open" : "menu";
    }
    localStorage.setItem(SIDEBAR_KEY, expanded ? "expanded" : "collapsed");
  }

  function restoreSidebar() {
    var saved = localStorage.getItem(SIDEBAR_KEY);
    if (saved === "expanded") {
      var nav = document.querySelector("nav.left.l");
      if (!nav) return;
      nav.classList.add("max");
      var btn = nav.querySelector("[aria-label='Toggle sidebar']");
      if (btn) {
        btn.setAttribute("aria-expanded", "true");
        var icon = btn.querySelector("i");
        if (icon) icon.textContent = "menu_open";
      }
    }
  }

  // -- Init -------------------------------------------------------

  // Expose toggles for onclick handlers in templates.
  window.setTheme = setTheme;
  window.toggleSidebar = toggleSidebar;

  document.addEventListener("DOMContentLoaded", function () {
    initTheme();
    syncNavActive();
    restoreSidebar();
  });
})();
