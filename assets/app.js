// specd web UI runtime — theme, nav, htmx hooks.
// Loaded after beer.min.js and material-dynamic-colors.min.js.

(function () {
  // Set brand color via BeerCSS + Material Dynamic Colors.
  ui("theme", "#1c4bea");

  // Restore saved dark/light mode.
  var saved = localStorage.getItem("mode");
  if (saved) ui("mode", saved);

  // Toggle dark/light mode on click (desktop and mobile buttons).
  document.querySelectorAll("button:has(> i.page)").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var current = ui("mode");
      var next = current === "dark" ? "light" : "dark";
      ui("mode", next);
      localStorage.setItem("mode", next);
      document.querySelectorAll("button:has(> i.page) > i").forEach(function (i) {
        i.textContent = next === "dark" ? "light_mode" : "dark_mode";
      });
    });
  });

  // Sync icon text with current mode on page load.
  var mode = ui("mode");
  document.querySelectorAll("button:has(> i.page) > i").forEach(function (i) {
    i.textContent = mode === "dark" ? "light_mode" : "dark_mode";
  });

  // Desktop sidebar collapse/expand toggle.
  var leftNav = document.querySelector("nav.left");
  if (leftNav) {
    var savedNav = localStorage.getItem("nav-collapsed");
    var menuBtn = leftNav.querySelector("button[aria-label='Toggle sidebar']");
    var menuIcon = menuBtn ? menuBtn.querySelector("i") : null;
    if (savedNav === "true") {
      leftNav.classList.remove("max");
      if (menuBtn) menuBtn.setAttribute("aria-expanded", "false");
      if (menuIcon) menuIcon.textContent = "menu";
    }

    if (menuBtn) {
      menuBtn.addEventListener("click", function () {
        var collapsed = leftNav.classList.toggle("max") === false;
        localStorage.setItem("nav-collapsed", String(collapsed));
        menuBtn.setAttribute("aria-expanded", String(!collapsed));
        if (menuIcon) menuIcon.textContent = collapsed ? "menu" : "menu_open";
      });
    }
  }
})();
