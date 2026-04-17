// specd web UI runtime — theme, nav, htmx hooks.
// Loaded after beer.min.js and material-dynamic-colors.min.js.

(function () {
  // Allow htmx to swap content on 422 (validation error) responses.
  // Dialog forms return 422 with the re-rendered form partial on error.
  document.body.addEventListener("htmx:beforeSwap", function (e) {
    if (e.detail.xhr.status === 422) {
      e.detail.shouldSwap = true;
      e.detail.isError = false;
    }
  });

  // KB add form: validate that either file or URL is provided.
  document.body.addEventListener("htmx:configRequest", function (e) {
    var form = e.detail.elt;
    if (form.id !== "kb-add-form") return;
    var fileInput = form.querySelector("[name=file]");
    var urlInput = form.querySelector("[name=url]");
    if (!fileInput.files.length && !urlInput.value.trim()) {
      e.preventDefault();
      urlInput.setCustomValidity("Please select a file or enter a URL");
      form.reportValidity();
    } else {
      urlInput.setCustomValidity("");
    }
  });

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
    // Default collapsed; only expand if user explicitly saved expanded.
    if (savedNav === "false") {
      leftNav.classList.add("max");
      if (menuBtn) menuBtn.setAttribute("aria-expanded", "true");
      if (menuIcon) menuIcon.textContent = "menu_open";
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

  // ── Auto-submit selects ─────────────────────────────────
  document.body.addEventListener("change", function (e) {
    if (e.target.classList.contains("sd-auto-submit")) {
      e.target.closest("form").submit();
    }
  });

  // ── Markdown body rendering ──────────────────────────────
  renderMarkdownBodies();

  // ── Kanban drag-and-drop ─────────────────────────────────
  initKanbanDnD();

  // Re-init after htmx swaps (board reload, detail reload).
  document.body.addEventListener("htmx:afterSwap", function () {
    initKanbanDnD();
    renderMarkdownBodies();
  });

  function initKanbanDnD() {
    var board = document.getElementById("kanban-board");
    if (!board) return;

    var draggedCard = null;
    var draggedTaskID = null;
    var sourceStatus = null;

    // Attach drag events to all cards.
    board.querySelectorAll(".sd-card[draggable]").forEach(function (card) {
      card.addEventListener("dragstart", function (e) {
        draggedCard = card;
        draggedTaskID = card.dataset.taskId;
        sourceStatus = card.dataset.status;
        card.classList.add("is-dragging");
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", draggedTaskID);
      });

      card.addEventListener("dragend", function () {
        card.classList.remove("is-dragging");
        clearDropHighlights();
        removePlaceholders();
        draggedCard = null;
        draggedTaskID = null;
        sourceStatus = null;
      });
    });

    // Attach drop zone events to column bodies.
    board.querySelectorAll(".sd-col-body").forEach(function (colBody) {
      colBody.addEventListener("dragover", function (e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        colBody.classList.add("is-drag-over");

        // Show placeholder between cards.
        var afterCard = getCardAfterCursor(colBody, e.clientY);
        removePlaceholders();
        var placeholder = document.createElement("div");
        placeholder.className = "sd-drop-placeholder";
        if (afterCard) {
          colBody.insertBefore(placeholder, afterCard);
        } else {
          colBody.appendChild(placeholder);
        }
      });

      colBody.addEventListener("dragleave", function (e) {
        // Only remove highlight if leaving the column entirely.
        if (!colBody.contains(e.relatedTarget)) {
          colBody.classList.remove("is-drag-over");
          removePlaceholders();
        }
      });

      colBody.addEventListener("drop", function (e) {
        e.preventDefault();
        colBody.classList.remove("is-drag-over");
        removePlaceholders();

        if (!draggedTaskID) return;

        var targetStatus = colBody.dataset.status;
        var afterCard = getCardAfterCursor(colBody, e.clientY);

        // Compute the target position index.
        var cards = Array.from(colBody.querySelectorAll(".sd-card:not(.is-dragging)"));
        var targetPos = afterCard ? cards.indexOf(afterCard) : cards.length;

        var url = targetStatus !== sourceStatus
          ? "/api/board/move"
          : "/api/board/reorder";
        var payload = targetStatus !== sourceStatus
          ? { task_id: draggedTaskID, status: targetStatus }
          : { task_id: draggedTaskID, position: targetPos };

        fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        })
          .then(function (resp) {
            if (!resp.ok) throw new Error(resp.statusText);
            return resp.text();
          })
          .then(function (html) {
            var container = document.getElementById("board-content");
            if (container) {
              container.outerHTML = html;
              initKanbanDnD();
              renderMarkdownBodies();
            }
          })
          .catch(function (err) {
            var snackbar = document.createElement("div");
            snackbar.className = "snackbar error active";
            snackbar.setAttribute("role", "alert");
            snackbar.innerHTML = "<i aria-hidden='true'>error</i><span>" +
              err.message + "</span>";
            document.body.appendChild(snackbar);
            setTimeout(function () { snackbar.remove(); }, 4000);
          });
      });
    });

    function getCardAfterCursor(colBody, y) {
      var cards = Array.from(
        colBody.querySelectorAll(".sd-card:not(.is-dragging)")
      );
      var closest = null;
      var closestOffset = Number.NEGATIVE_INFINITY;
      cards.forEach(function (card) {
        var box = card.getBoundingClientRect();
        var offset = y - box.top - box.height / 2;
        if (offset < 0 && offset > closestOffset) {
          closestOffset = offset;
          closest = card;
        }
      });
      return closest;
    }

    function clearDropHighlights() {
      board.querySelectorAll(".sd-col-body.is-drag-over").forEach(function (el) {
        el.classList.remove("is-drag-over");
      });
    }

    function removePlaceholders() {
      board.querySelectorAll(".sd-drop-placeholder").forEach(function (el) {
        el.remove();
      });
    }
  }

  // ── Markdown body rendering ─────────────────────────────
  // Renders [data-md-body] elements using marked.js.
  function renderMarkdownBodies() {
    if (typeof marked === "undefined") return;
    document.querySelectorAll("[data-md-body]").forEach(function (el) {
      if (el.dataset.mdRendered) return;
      var raw = el.getAttribute("data-md-body");
      if (!raw) {
        el.innerHTML = "<p>No content.</p>";
      } else {
        try {
          el.innerHTML = marked.parse(raw);
        } catch (e) {
          el.innerHTML = "<pre>" + raw.replace(/</g, "&lt;") + "</pre>";
        }
      }
      el.dataset.mdRendered = "1";
    });
  }
})();
