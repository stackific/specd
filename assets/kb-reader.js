// KB reader — renders markdown, plain text, HTML, and PDF documents
// with chunk highlighting and navigation. Loaded on the kb-detail page.

(function () {
  var reader = document.getElementById("kb-reader");
  if (!reader) return;

  var docId = reader.dataset.docId;
  var sourceType = reader.dataset.sourceType;
  var totalChunks = parseInt(reader.dataset.totalChunks, 10) || 0;
  var initialChunk = parseInt(reader.dataset.activeChunk, 10);
  var contentEl = document.getElementById("kr-content");
  var loadingEl = document.getElementById("kr-loading");
  var posEl = document.getElementById("kr-current");
  var prevBtn = document.getElementById("kr-prev");
  var nextBtn = document.getElementById("kr-next");
  var toggleBtn = document.getElementById("kr-toggle-chunks");
  var highlightBtn = document.getElementById("kr-highlight-toggle");
  var sidebar = document.getElementById("kr-sidebar");
  var chunkListEl = document.getElementById("kr-chunk-list");

  // Parse embedded chunk data.
  var chunksDataEl = document.getElementById("kr-chunks-data");
  var chunks = [];
  if (chunksDataEl) {
    try {
      chunks = JSON.parse(chunksDataEl.textContent);
    } catch (e) {
      console.error("Failed to parse chunk data:", e);
    }
  }

  var currentChunk = -1;
  var highlightEnabled = true;
  var renderedContainer = null; // the DOM element holding rendered content

  // ── Initialization ─────────────────────────────────────────

  if (sourceType === "md") {
    renderMarkdown();
  } else if (sourceType === "txt") {
    renderPlainText();
  } else if (sourceType === "html") {
    renderHTML();
  } else if (sourceType === "pdf") {
    renderPDF();
  } else {
    hideLoading();
    contentEl.innerHTML =
      '<p class="p-3">Rendering for <strong>' +
      escapeHTML(sourceType) +
      "</strong> documents is not yet supported in the reader.</p>";
  }

  // ── Event listeners ────────────────────────────────────────

  if (prevBtn) {
    prevBtn.addEventListener("click", function () {
      if (currentChunk > 0) navigateToChunk(currentChunk - 1);
    });
  }
  if (nextBtn) {
    nextBtn.addEventListener("click", function () {
      if (currentChunk < totalChunks - 1) navigateToChunk(currentChunk + 1);
    });
  }
  // Keyboard shortcuts for chunk navigation.
  document.addEventListener("keydown", function (e) {
    // Skip if user is typing in an input/textarea.
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") return;
    if (e.key === "ArrowLeft" && currentChunk > 0) {
      e.preventDefault();
      navigateToChunk(currentChunk - 1);
    } else if (e.key === "ArrowRight" && currentChunk < totalChunks - 1) {
      e.preventDefault();
      navigateToChunk(currentChunk + 1);
    }
  });

  if (toggleBtn && sidebar) {
    toggleBtn.addEventListener("click", function () {
      var hidden = sidebar.hidden;
      sidebar.hidden = !hidden;
      toggleBtn.setAttribute("aria-expanded", String(hidden));
    });
  }
  if (highlightBtn) {
    highlightBtn.addEventListener("click", function () {
      highlightEnabled = !highlightEnabled;
      highlightBtn.setAttribute("aria-pressed", String(highlightEnabled));
      if (highlightEnabled && currentChunk >= 0) {
        applyHighlight(currentChunk);
      } else {
        clearHighlights();
      }
    });
  }

  // Sidebar chunk item clicks.
  if (chunkListEl) {
    chunkListEl.addEventListener("click", function (e) {
      var btn = e.target.closest(".kr-chunk-item");
      if (!btn) return;
      var pos = parseInt(btn.dataset.chunkPos, 10);
      if (!isNaN(pos)) navigateToChunk(pos);
    });
  }

  // ── Markdown renderer ──────────────────────────────────────

  function renderMarkdown() {
    fetch("/api/kb/" + docId + "/raw")
      .then(function (r) {
        if (!r.ok) throw new Error("Failed to fetch");
        return r.text();
      })
      .then(function (raw) {
        hideLoading();
        // Use marked.js (loaded as module, available on window).
        var html;
        if (typeof marked !== "undefined" && marked.parse) {
          html = marked.parse(raw);
        } else if (
          typeof window.marked !== "undefined" &&
          window.marked.parse
        ) {
          html = window.marked.parse(raw);
        } else {
          // Fallback: simple pre-formatted display.
          html = "<pre>" + escapeHTML(raw) + "</pre>";
        }
        var div = document.createElement("div");
        div.className = "kr-rendered";
        div.innerHTML = html;
        contentEl.appendChild(div);
        renderedContainer = div;

        // Navigate to initial chunk if specified.
        if (initialChunk >= 0 && initialChunk < totalChunks) {
          navigateToChunk(initialChunk);
        } else if (totalChunks > 0) {
          navigateToChunk(0);
        }
      })
      .catch(function (err) {
        hideLoading();
        contentEl.innerHTML =
          '<p class="p-3">Error loading document: ' +
          escapeHTML(err.message) +
          "</p>";
      });
  }

  // ── Plain text renderer ────────────────────────────────────

  function renderPlainText() {
    fetch("/api/kb/" + docId + "/raw")
      .then(function (r) {
        if (!r.ok) throw new Error("Failed to fetch");
        return r.text();
      })
      .then(function (raw) {
        hideLoading();
        var div = document.createElement("div");
        div.className = "kr-plaintext";
        div.textContent = raw;
        contentEl.appendChild(div);
        renderedContainer = div;

        if (initialChunk >= 0 && initialChunk < totalChunks) {
          navigateToChunk(initialChunk);
        } else if (totalChunks > 0) {
          navigateToChunk(0);
        }
      })
      .catch(function (err) {
        hideLoading();
        contentEl.innerHTML =
          '<p class="p-3">Error loading document: ' +
          escapeHTML(err.message) +
          "</p>";
      });
  }

  // ── HTML renderer (sandboxed iframe) ────────────────────────

  var iframeDoc = null; // non-null when sourceType === "html"

  function renderHTML() {
    fetch("/api/kb/" + docId + "/raw")
      .then(function (r) {
        if (!r.ok) throw new Error("Failed to fetch");
        return r.text();
      })
      .then(function (raw) {
        hideLoading();

        // Create a sandboxed iframe: allow-same-origin lets us reach
        // into the document for highlighting; no allow-scripts prevents
        // any script execution inside the sanitized HTML.
        var iframe = document.createElement("iframe");
        iframe.className = "kr-html-frame";
        iframe.setAttribute("sandbox", "allow-same-origin");
        iframe.setAttribute("title", "KB document content");
        contentEl.appendChild(iframe);

        // Write the sanitized HTML into the iframe.
        var doc = iframe.contentDocument || iframe.contentWindow.document;
        doc.open();
        doc.write(raw);
        doc.close();
        iframeDoc = doc;

        // Auto-size the iframe to fit its content after layout.
        requestAnimationFrame(function () {
          resizeIframe(iframe);
          // Second pass for images/fonts that load asynchronously.
          setTimeout(function () { resizeIframe(iframe); }, 200);
        });

        // Observe dynamic resizing (images loading, etc.).
        if (typeof ResizeObserver !== "undefined") {
          var ro = new ResizeObserver(function () {
            resizeIframe(iframe);
          });
          ro.observe(doc.body);
        }

        // Navigate to initial chunk.
        if (initialChunk >= 0 && initialChunk < totalChunks) {
          navigateToChunk(initialChunk);
        } else if (totalChunks > 0) {
          navigateToChunk(0);
        }
      })
      .catch(function (err) {
        hideLoading();
        contentEl.innerHTML =
          '<p class="p-3">Error loading document: ' +
          escapeHTML(err.message) +
          "</p>";
      });
  }

  // Resize iframe height to match its content (no scrollbar inside frame).
  function resizeIframe(iframe) {
    var doc = iframe.contentDocument;
    if (!doc || !doc.body) return;
    iframe.style.height = doc.body.scrollHeight + 32 + "px";
  }

  // ── PDF renderer (PDF.js) ─────────────────────────────────

  var pdfDoc = null;         // PDFDocumentProxy from PDF.js
  var pdfRenderedPages = {};  // page number → { container, textLayerDiv }
  var pdfScale = 1.5;
  var pdfContainer = null;   // wrapper div for all pages

  function renderPDF() {
    import("/assets/pdf.min.mjs")
      .then(function () {
        var pdfjsLib = window.pdfjsLib || globalThis.pdfjsLib;
        if (!pdfjsLib) {
          throw new Error("PDF.js library failed to load");
        }

        // Configure worker.
        pdfjsLib.GlobalWorkerOptions.workerSrc = "/assets/pdf.worker.min.mjs";

        // Fetch PDF bytes and load the document.
        return fetch("/api/kb/" + docId + "/raw")
          .then(function (r) {
            if (!r.ok) throw new Error("Failed to fetch PDF");
            return r.arrayBuffer();
          })
          .then(function (data) {
            return pdfjsLib.getDocument({ data: data }).promise;
          });
      })
      .then(function (pdf) {
        pdfDoc = pdf;
        hideLoading();

        // Create the pages container.
        pdfContainer = document.createElement("div");
        pdfContainer.className = "kr-pdf-container";
        contentEl.appendChild(pdfContainer);
        renderedContainer = pdfContainer;

        // Determine which page to render first based on initial chunk.
        var startPage = 1;
        if (initialChunk >= 0 && initialChunk < chunks.length && chunks[initialChunk].page) {
          startPage = chunks[initialChunk].page;
        }

        // Render all pages for smooth scrolling.
        return renderAllPages(pdf).then(function () {
          // Navigate to initial chunk.
          if (initialChunk >= 0 && initialChunk < totalChunks) {
            navigateToChunk(initialChunk);
          } else if (totalChunks > 0) {
            navigateToChunk(0);
          }
        });
      })
      .catch(function (err) {
        hideLoading();
        contentEl.innerHTML =
          '<p class="p-3">Error loading PDF: ' +
          escapeHTML(err.message) + "</p>";
      });
  }

  // Render all pages of the PDF document sequentially.
  function renderAllPages(pdf) {
    var chain = Promise.resolve();
    for (var i = 1; i <= pdf.numPages; i++) {
      (function (pageNum) {
        chain = chain.then(function () {
          return renderPDFPage(pdf, pageNum);
        });
      })(i);
    }
    return chain;
  }

  // Render a single PDF page: canvas + text layer.
  function renderPDFPage(pdf, pageNum) {
    if (pdfRenderedPages[pageNum]) return Promise.resolve();

    return pdf.getPage(pageNum).then(function (page) {
      var viewport = page.getViewport({ scale: pdfScale });

      // Page wrapper.
      var pageDiv = document.createElement("div");
      pageDiv.className = "kr-pdf-page";
      pageDiv.setAttribute("data-page", pageNum);

      // Page number label.
      var pageLabel = document.createElement("div");
      pageLabel.className = "kr-pdf-page-label";
      pageLabel.textContent = "Page " + pageNum;
      pageDiv.appendChild(pageLabel);

      // Canvas for the rendered page image.
      var canvasWrapper = document.createElement("div");
      canvasWrapper.className = "kr-pdf-canvas-wrap";
      canvasWrapper.style.width = viewport.width + "px";
      canvasWrapper.style.height = viewport.height + "px";
      canvasWrapper.style.position = "relative";

      var canvas = document.createElement("canvas");
      canvas.width = viewport.width;
      canvas.height = viewport.height;
      canvasWrapper.appendChild(canvas);

      // Text layer div — overlaid on top of the canvas for
      // selection and highlighting.
      var textLayerDiv = document.createElement("div");
      textLayerDiv.className = "kr-pdf-text-layer";
      textLayerDiv.style.width = viewport.width + "px";
      textLayerDiv.style.height = viewport.height + "px";
      canvasWrapper.appendChild(textLayerDiv);

      pageDiv.appendChild(canvasWrapper);
      pdfContainer.appendChild(pageDiv);

      // Render the canvas.
      var ctx = canvas.getContext("2d");
      var renderTask = page.render({
        canvasContext: ctx,
        viewport: viewport,
      });

      return renderTask.promise.then(function () {
        // Render the text layer using PDF.js TextLayer API.
        return page.getTextContent();
      }).then(function (textContent) {
        var pdfjsLib = window.pdfjsLib || globalThis.pdfjsLib;
        if (pdfjsLib && pdfjsLib.TextLayer) {
          var textLayer = new pdfjsLib.TextLayer({
            textContentSource: textContent,
            container: textLayerDiv,
            viewport: viewport,
          });
          return textLayer.render().then(function () {
            pdfRenderedPages[pageNum] = {
              container: pageDiv,
              textLayerDiv: textLayerDiv,
            };
          });
        }

        // Fallback: manually place text spans if TextLayer class
        // is not available (older PDF.js versions).
        renderTextLayerFallback(textContent, textLayerDiv, viewport);
        pdfRenderedPages[pageNum] = {
          container: pageDiv,
          textLayerDiv: textLayerDiv,
        };
      });
    });
  }

  // Fallback text layer renderer for older PDF.js versions.
  function renderTextLayerFallback(textContent, container, viewport) {
    textContent.items.forEach(function (item) {
      if (!item.str) return;
      var tx = item.transform;
      var span = document.createElement("span");
      span.textContent = item.str;
      span.style.position = "absolute";
      span.style.left = tx[4] + "px";
      span.style.top = (viewport.height - tx[5] - item.height) + "px";
      span.style.fontSize = Math.abs(tx[0]) + "px";
      span.style.fontFamily = "sans-serif";
      container.appendChild(span);
    });
  }

  // ── Chunk navigation ───────────────────────────────────────

  function navigateToChunk(pos) {
    if (pos < 0 || pos >= totalChunks) return;
    currentChunk = pos;
    updateNavState();
    updateSidebarActive(pos);

    if (highlightEnabled) {
      applyHighlight(pos);
    }
  }

  function updateNavState() {
    if (posEl) posEl.textContent = currentChunk + 1;
    if (prevBtn) prevBtn.disabled = currentChunk <= 0;
    if (nextBtn) nextBtn.disabled = currentChunk >= totalChunks - 1;
  }

  function updateSidebarActive(pos) {
    if (!chunkListEl) return;
    chunkListEl.querySelectorAll(".kr-chunk-item").forEach(function (btn) {
      btn.classList.toggle(
        "is-active",
        parseInt(btn.dataset.chunkPos, 10) === pos
      );
    });
  }

  // ── Highlighting ───────────────────────────────────────────

  function clearHighlights() {
    // Clear highlights in parent document, iframe, and PDF text layers.
    var roots = [contentEl];
    if (iframeDoc && iframeDoc.body) roots.push(iframeDoc.body);
    // Include all rendered PDF text layers.
    Object.keys(pdfRenderedPages).forEach(function (key) {
      var p = pdfRenderedPages[key];
      if (p && p.textLayerDiv) roots.push(p.textLayerDiv);
    });
    roots.forEach(function (root) {
      if (!root) return;
      root.querySelectorAll(".kr-highlight").forEach(function (mark) {
        var parent = mark.parentNode;
        while (mark.firstChild) {
          parent.insertBefore(mark.firstChild, mark);
        }
        parent.removeChild(mark);
        parent.normalize();
      });
    });
  }

  function applyHighlight(pos) {
    clearHighlights();
    if (pos < 0 || pos >= chunks.length) return;

    var chunk = chunks[pos];
    if (!chunk) return;

    var mark;
    if (sourceType === "html") {
      // Highlight inside the iframe's DOM using prefix match.
      if (!iframeDoc || !iframeDoc.body) return;
      injectHighlightStyle(iframeDoc);
      var prefix = chunk.text.slice(0, 80).trim();
      if (prefix) {
        mark = highlightPrefixInDOM(iframeDoc.body, prefix);
      }
      if (mark) {
        // Scroll the iframe itself into view in the parent, then
        // scroll the highlight into view inside the iframe.
        var iframe = contentEl.querySelector(".kr-html-frame");
        if (iframe) iframe.scrollIntoView({ behavior: "smooth", block: "start" });
        mark.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    } else if (sourceType === "pdf") {
      // PDF: find the chunk's page, search its text layer.
      var pageNum = chunk.page || 1;
      var pageData = pdfRenderedPages[pageNum];
      if (!pageData) return;

      var prefix = chunk.text.slice(0, 80).trim();
      if (prefix) {
        mark = highlightPrefixInDOM(pageData.textLayerDiv, prefix);
      }
      if (mark) {
        // Scroll the page container into view, then the highlight.
        pageData.container.scrollIntoView({ behavior: "smooth", block: "start" });
        setTimeout(function () {
          mark.scrollIntoView({ behavior: "smooth", block: "center" });
        }, 300);
      } else {
        // If text match fails, at least scroll to the page.
        pageData.container.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    } else if (sourceType === "txt") {
      if (!renderedContainer) return;
      mark = highlightByCharOffset(renderedContainer, chunk.char_start, chunk.char_end);
      if (mark) mark.scrollIntoView({ behavior: "smooth", block: "center" });
    } else {
      // Markdown: prefix match on rendered DOM text.
      // Strip markdown syntax since the DOM contains rendered text
      // (e.g., "# Heading" becomes "Heading" in the DOM).
      if (!renderedContainer) return;
      // Use only the first meaningful line — multi-line prefixes span
      // different DOM elements and won't match a single text node.
      var stripped = stripMarkdown(chunk.text);
      var firstLine = stripped.split("\n").filter(function (l) { return l.trim(); })[0] || "";
      var prefix = firstLine.slice(0, 80).trim();
      if (prefix) {
        mark = highlightPrefixInDOM(renderedContainer, prefix);
      }
      if (mark) mark.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }

  // Inject the highlight CSS class into the iframe document so
  // <mark class="kr-highlight"> is styled inside the sandboxed frame.
  var iframeStyleInjected = false;
  function injectHighlightStyle(doc) {
    if (iframeStyleInjected) return;
    iframeStyleInjected = true;
    var style = doc.createElement("style");
    style.textContent =
      ".kr-highlight { background: #e8def8; border-left: 3px solid #6750a4; " +
      "padding: 2px 4px; scroll-margin-top: 100px; border-radius: 4px; }";
    (doc.head || doc.documentElement).appendChild(style);
  }

  // Highlight by character offsets in a plain text container.
  function highlightByCharOffset(root, charStart, charEnd) {
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    var charPos = 0;
    var startNode = null;
    var startOffset = 0;
    var endNode = null;
    var endOffset = 0;

    while (walker.nextNode()) {
      var node = walker.currentNode;
      var nodeLen = node.textContent.length;
      var nodeEnd = charPos + nodeLen;

      if (!startNode && charStart < nodeEnd) {
        startNode = node;
        startOffset = charStart - charPos;
      }
      if (charEnd <= nodeEnd) {
        endNode = node;
        endOffset = charEnd - charPos;
        break;
      }
      charPos = nodeEnd;
    }

    if (!startNode || !endNode) return null;

    try {
      var range = document.createRange();
      range.setStart(startNode, startOffset);
      range.setEnd(endNode, endOffset);
      var mark = document.createElement("mark");
      mark.className = "kr-highlight";
      range.surroundContents(mark);
      return mark;
    } catch (e) {
      // surroundContents fails if range spans multiple elements;
      // fall back to wrapping individual text nodes.
      return highlightRangeFallback(startNode, startOffset, endNode, endOffset);
    }
  }

  // Fallback: wrap text nodes individually when range spans elements.
  function highlightRangeFallback(startNode, startOffset, endNode, endOffset) {
    var firstMark = null;
    var walker = document.createTreeWalker(
      startNode.parentNode.closest(".kr-plaintext, .kr-rendered") || startNode.parentNode,
      NodeFilter.SHOW_TEXT
    );
    var inRange = false;

    while (walker.nextNode()) {
      var node = walker.currentNode;
      if (node === startNode) inRange = true;
      if (!inRange) continue;

      var from = node === startNode ? startOffset : 0;
      var to = node === endNode ? endOffset : node.textContent.length;

      if (from < to) {
        var range = document.createRange();
        range.setStart(node, from);
        range.setEnd(node, to);
        var mark = document.createElement("mark");
        mark.className = "kr-highlight";
        try {
          range.surroundContents(mark);
          if (!firstMark) firstMark = mark;
        } catch (e) {
          // Skip nodes that can't be wrapped.
        }
      }

      if (node === endNode) break;
    }
    return firstMark;
  }

  // Highlight first occurrence of prefix text in rendered DOM.
  function highlightPrefixInDOM(root, prefix) {
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      var node = walker.currentNode;
      var idx = node.textContent.indexOf(prefix);
      if (idx < 0) continue;

      try {
        var range = document.createRange();
        range.setStart(node, idx);
        range.setEnd(
          node,
          Math.min(idx + prefix.length, node.textContent.length)
        );
        var mark = document.createElement("mark");
        mark.className = "kr-highlight";
        range.surroundContents(mark);
        return mark;
      } catch (e) {
        // If surroundContents fails (cross-element), try next node.
        continue;
      }
    }

    // Fallback: try concatenated text search across nodes.
    return highlightConcatenated(root, prefix);
  }

  // Search across concatenated text nodes for the prefix.
  function highlightConcatenated(root, prefix) {
    var textNodes = [];
    var fullText = "";
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      var node = walker.currentNode;
      textNodes.push({ node: node, start: fullText.length });
      fullText += node.textContent;
    }

    var idx = fullText.indexOf(prefix);
    if (idx < 0) return null;

    // Find the first text node containing the start of the match.
    for (var i = 0; i < textNodes.length; i++) {
      var tn = textNodes[i];
      var tnEnd = tn.start + tn.node.textContent.length;
      if (idx < tnEnd) {
        var localIdx = idx - tn.start;
        var endIdx = Math.min(localIdx + prefix.length, tn.node.textContent.length);
        try {
          var range = document.createRange();
          range.setStart(tn.node, localIdx);
          range.setEnd(tn.node, endIdx);
          var mark = document.createElement("mark");
          mark.className = "kr-highlight";
          range.surroundContents(mark);
          return mark;
        } catch (e) {
          return null;
        }
      }
    }
    return null;
  }

  // ── Helpers ────────────────────────────────────────────────

  function hideLoading() {
    if (loadingEl) loadingEl.style.display = "none";
  }

  function escapeHTML(s) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(s));
    return div.innerHTML;
  }

  // Strip common markdown syntax so the text matches the rendered DOM.
  // Handles headings (#), bold/italic (*/_), inline code (`), links, images.
  function stripMarkdown(s) {
    return s
      .replace(/^#{1,6}\s+/gm, "")        // heading markers
      .replace(/\*\*\*(.+?)\*\*\*/g, "$1") // bold italic
      .replace(/\*\*(.+?)\*\*/g, "$1")     // bold
      .replace(/__(.+?)__/g, "$1")          // bold alt
      .replace(/\*(.+?)\*/g, "$1")          // italic
      .replace(/_(.+?)_/g, "$1")            // italic alt
      .replace(/~~(.+?)~~/g, "$1")          // strikethrough
      .replace(/`([^`]+)`/g, "$1")          // inline code
      .replace(/```[\s\S]*?```/g, "")       // fenced code blocks
      .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1") // images
      .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")  // links
      .replace(/^\s*[-*+]\s+/gm, "")       // unordered list markers
      .replace(/^\s*\d+\.\s+/gm, "")       // ordered list markers
      .replace(/^>\s?/gm, "")              // blockquote markers
      .replace(/^\|.*\|$/gm, "")           // table rows
      .replace(/^[-:|]+$/gm, "")           // table separators
      .replace(/\[\^[^\]]+\]/g, "");       // footnote references
  }
})();
