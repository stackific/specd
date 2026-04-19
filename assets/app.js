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

  // ── Board filter persistence ────────────────────────────
  // Saves the selected spec filter to localStorage so it survives
  // navigation away and back. The early-redirect script in <head>
  // restores it before first paint. Invalidates if the spec was deleted.
  function initBoardFilter() {
    var sel = document.getElementById("board-spec-filter");
    if (!sel) return;

    // Invalidate: if localStorage has a spec that no longer exists
    // in the dropdown (deleted), clear it so next visit shows all.
    var saved = localStorage.getItem("sd-board-spec");
    if (saved) {
      var found = false;
      for (var i = 0; i < sel.options.length; i++) {
        if (sel.options[i].value === saved) { found = true; break; }
      }
      if (!found) localStorage.removeItem("sd-board-spec");
    }

    // Persist on change.
    sel.addEventListener("change", function () {
      if (sel.value) {
        localStorage.setItem("sd-board-spec", sel.value);
      } else {
        localStorage.removeItem("sd-board-spec");
      }
    });
  }
  initBoardFilter();

  // ── Markdown body rendering ──────────────────────────────
  renderMarkdownBodies();

  // ── Kanban drag-and-drop ─────────────────────────────────
  initKanbanDnD();

  // Re-init after htmx swaps (board reload, detail reload).
  document.body.addEventListener("htmx:afterSwap", function () {
    initBoardFilter();
    initKanbanDnD();
    renderMarkdownBodies();
  });

  function initKanbanDnD() {
    var board = document.getElementById("kanban-board");
    if (!board) return;

    var draggedCard = null;
    var draggedTaskID = null;
    var sourceStatus = null;
    var placeholder = null;
    // Track the resolved drop target so drop uses the same index the user saw.
    var dropTarget = null; // { status, position }

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
        clearPlaceholder();
        draggedCard = null;
        draggedTaskID = null;
        sourceStatus = null;
        dropTarget = null;
      });
    });

    board.querySelectorAll(".sd-col-body").forEach(function (colBody) {
      colBody.addEventListener("dragover", function (e) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";

        // Get visible cards (exclude the dragged one).
        var cards = Array.from(
          colBody.querySelectorAll(".sd-card:not(.is-dragging)")
        );
        var afterCard = cardAfterCursor(cards, e.clientY);

        // Compute position index before any DOM changes.
        var position = afterCard ? cards.indexOf(afterCard) : cards.length;
        var targetStatus = colBody.dataset.status;

        // Skip if this is the same slot the card is already in.
        // position is an index in the filtered list (without dragged card),
        // srcIdx is the index in the full list (with dragged card).
        // Only position === srcIdx is a true no-op in the filtered list.
        if (targetStatus === sourceStatus) {
          var srcCards = Array.from(
            colBody.querySelectorAll(".sd-card")
          );
          var srcIdx = srcCards.indexOf(draggedCard);
          if (position === srcIdx) {
            clearPlaceholder();
            dropTarget = null;
            return;
          }
        }

        // Only update DOM if insertion point actually changed.
        var ref = afterCard ? afterCard.dataset.taskId : "_end";
        if (dropTarget && dropTarget.ref === ref && dropTarget.status === targetStatus) {
          return;
        }

        clearPlaceholder();
        placeholder = document.createElement("div");
        placeholder.className = "sd-drop-placeholder";
        if (afterCard) {
          colBody.insertBefore(placeholder, afterCard);
        } else {
          colBody.appendChild(placeholder);
        }

        dropTarget = { status: targetStatus, position: position, ref: ref };
      });

      colBody.addEventListener("dragleave", function (e) {
        if (!colBody.contains(e.relatedTarget)) {
          clearPlaceholder();
          dropTarget = null;
        }
      });

      colBody.addEventListener("drop", function (e) {
        e.preventDefault();
        clearPlaceholder();

        if (!draggedTaskID || !dropTarget) return;

        var url = dropTarget.status !== sourceStatus
          ? "/api/board/move"
          : "/api/board/reorder";
        var payload = dropTarget.status !== sourceStatus
          ? { task_id: draggedTaskID, status: dropTarget.status, position: dropTarget.position }
          : { task_id: draggedTaskID, position: dropTarget.position };

        dropTarget = null;

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

    // Find the card whose top-center is just below the cursor.
    function cardAfterCursor(cards, y) {
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

    function clearPlaceholder() {
      if (placeholder && placeholder.parentNode) {
        placeholder.remove();
      }
      placeholder = null;
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
