// specd — Kanban drag/drop wiring.
// Pure HTML5 drag-and-drop. After a successful move the server returns the
// re-rendered board partial, which htmx swaps into #kanban-root. We re-bind
// listeners on every htmx swap.

(function () {
  "use strict";

  var ROOT_ID = "kanban-root";
  var DRAG_KEY = "application/x-specd-task";
  var COLLAPSE_KEY = "specd-kanban-collapsed";

  var dragging = null; // currently-dragged card element
  var placeholder = null;

  // Read the active board filter from the kanban-root's data-filter attribute,
  // which is rendered server-side from the URL query string.
  function currentFilter() {
    var root = document.getElementById(ROOT_ID);
    return (root && root.dataset.filter) || "all";
  }

  function loadCollapsed() {
    try {
      var raw = localStorage.getItem(COLLAPSE_KEY);
      if (!raw) return new Set();
      return new Set(raw.split(",").filter(Boolean));
    } catch (e) {
      return new Set();
    }
  }

  function saveCollapsed(set) {
    try {
      localStorage.setItem(COLLAPSE_KEY, [...set].join(","));
    } catch (e) {
      // localStorage unavailable — ignore
    }
  }

  function applyCollapsedState() {
    var collapsed = loadCollapsed();
    document.querySelectorAll(".kanban-column").forEach(function (col) {
      var status = col.dataset.status;
      var isCollapsed = collapsed.has(status);
      col.classList.toggle("collapsed", isCollapsed);
      var btn = col.querySelector(".kanban-collapse-btn");
      if (btn) btn.setAttribute("aria-expanded", String(!isCollapsed));
    });
  }

  function toggleCollapse(col) {
    var collapsed = loadCollapsed();
    var status = col.dataset.status;
    if (col.classList.toggle("collapsed")) {
      collapsed.add(status);
    } else {
      collapsed.delete(status);
    }
    saveCollapsed(collapsed);
    var btn = col.querySelector(".kanban-collapse-btn");
    if (btn) btn.setAttribute("aria-expanded", String(!col.classList.contains("collapsed")));
  }

  function expandColumn(col) {
    if (!col.classList.contains("collapsed")) return;
    var collapsed = loadCollapsed();
    collapsed.delete(col.dataset.status);
    saveCollapsed(collapsed);
    col.classList.remove("collapsed");
    var btn = col.querySelector(".kanban-collapse-btn");
    if (btn) btn.setAttribute("aria-expanded", "true");
  }

  function makePlaceholder() {
    var el = document.createElement("div");
    el.className = "kanban-placeholder";
    return el;
  }

  function bind() {
    var root = document.getElementById(ROOT_ID);
    if (!root) return;

    applyCollapsedState();

    root.querySelectorAll(".kanban-collapse-btn").forEach(function (btn) {
      btn.addEventListener("click", function (evt) {
        evt.preventDefault();
        var col = btn.closest(".kanban-column");
        if (col) toggleCollapse(col);
      });
    });

    root.querySelectorAll(".kanban-column").forEach(function (col) {
      // Auto-expand a collapsed column when a card is dragged onto it.
      col.addEventListener("dragenter", function () {
        if (dragging) expandColumn(col);
      });
    });

    root.querySelectorAll(".kanban-card").forEach(function (card) {
      card.addEventListener("dragstart", onDragStart);
      card.addEventListener("dragend", onDragEnd);
    });

    root.querySelectorAll(".kanban-cards").forEach(function (col) {
      col.addEventListener("dragover", onDragOver);
      col.addEventListener("dragleave", onDragLeave);
      col.addEventListener("drop", onDrop);
    });
  }

  function onDragStart(evt) {
    dragging = evt.currentTarget;
    placeholder = makePlaceholder();
    try {
      evt.dataTransfer.setData(DRAG_KEY, dragging.dataset.taskId);
      evt.dataTransfer.effectAllowed = "move";
    } catch (e) {
      // some browsers throw on custom mime types
    }
    requestAnimationFrame(function () {
      if (dragging) dragging.classList.add("dragging");
    });
  }

  function onDragEnd() {
    if (dragging) dragging.classList.remove("dragging");
    if (placeholder && placeholder.parentNode) {
      placeholder.parentNode.removeChild(placeholder);
    }
    dragging = null;
    placeholder = null;
    document.querySelectorAll(".kanban-cards.drop-target").forEach(function (el) {
      el.classList.remove("drop-target");
    });
  }

  function onDragOver(evt) {
    if (!dragging) return;
    evt.preventDefault();
    evt.dataTransfer.dropEffect = "move";
    var col = evt.currentTarget;
    col.classList.add("drop-target");

    var afterEl = cardAfter(col, evt.clientY);
    if (!placeholder) placeholder = makePlaceholder();
    if (afterEl == null) {
      col.appendChild(placeholder);
    } else {
      col.insertBefore(placeholder, afterEl);
    }
  }

  function onDragLeave(evt) {
    var col = evt.currentTarget;
    // only clear when leaving the column entirely
    if (!col.contains(evt.relatedTarget)) {
      col.classList.remove("drop-target");
    }
  }

  function onDrop(evt) {
    if (!dragging) return;
    evt.preventDefault();

    var col = evt.currentTarget;
    var status = col.dataset.status;
    var taskId = dragging.dataset.taskId;

    // Compute the index where the placeholder sits among siblings,
    // ignoring the placeholder itself and the dragged card.
    var index = 0;
    var cards = col.querySelectorAll(".kanban-card, .kanban-placeholder");
    for (var i = 0; i < cards.length; i++) {
      if (cards[i] === placeholder) break;
      if (cards[i] === dragging) continue;
      index++;
    }

    // Move locally first for snappy feel; server response will replace anyway.
    if (placeholder && placeholder.parentNode) {
      placeholder.parentNode.insertBefore(dragging, placeholder);
      placeholder.parentNode.removeChild(placeholder);
      placeholder = null;
    }
    col.classList.remove("drop-target");

    if (typeof htmx === "undefined") return;
    htmx.ajax("POST", "/api/tasks/move", {
      target: "#" + ROOT_ID,
      swap: "innerHTML",
      values: {
        id: taskId,
        status: status,
        position: String(index),
        filter: currentFilter(),
      },
    });
  }

  function cardAfter(col, y) {
    var cards = col.querySelectorAll(".kanban-card:not(.dragging)");
    for (var i = 0; i < cards.length; i++) {
      var rect = cards[i].getBoundingClientRect();
      if (y < rect.top + rect.height / 2) return cards[i];
    }
    return null;
  }

  document.body.addEventListener("htmx:afterSwap", function (evt) {
    if (evt.target && (evt.target.id === ROOT_ID || evt.target.closest && evt.target.closest("#" + ROOT_ID))) {
      bind();
    }
  });

  document.addEventListener("DOMContentLoaded", bind);
})();
