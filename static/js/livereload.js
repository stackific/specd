// specd — Dev-mode live reload
// Connects to the SSE endpoint. When the server restarts (Air rebuild),
// the connection drops and the script reconnects + reloads the page.

(function () {
  "use strict";

  var source = new EventSource("/api/dev/livereload");

  source.addEventListener("reload", function () {
    window.location.reload();
  });

  source.onerror = function () {
    source.close();
    // Server went down (Air restart). Poll until it comes back.
    var interval = setInterval(function () {
      fetch("/api/dev/livereload", { method: "HEAD" })
        .then(function (resp) {
          if (resp.ok) {
            clearInterval(interval);
            window.location.reload();
          }
        })
        .catch(function () {
          // Still down, keep polling.
        });
    }, 500);
  };
})();
