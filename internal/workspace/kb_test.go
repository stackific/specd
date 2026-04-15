package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkParagraphs(t *testing.T) {
	// Build text with multiple paragraphs.
	var paras []string
	for i := 0; i < 5; i++ {
		paras = append(paras, strings.Repeat("word ", 120)) // ~600 chars each
	}
	text := strings.Join(paras, "\n\n")

	chunks := chunkParagraphs(text)
	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}

	// Each chunk should be within bounds.
	for _, c := range chunks {
		if c.CharStart < 0 || c.CharEnd < c.CharStart {
			t.Errorf("chunk %d: invalid offsets [%d, %d)", c.Position, c.CharStart, c.CharEnd)
		}
		if c.Text == "" {
			t.Errorf("chunk %d: empty text", c.Position)
		}
	}

	// Positions should be sequential.
	for i, c := range chunks {
		if c.Position != i {
			t.Errorf("chunk position %d, want %d", c.Position, i)
		}
	}
}

func TestChunkParagraphsSmallText(t *testing.T) {
	text := "A short paragraph."
	chunks := chunkParagraphs(text)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != text {
		t.Errorf("chunk text = %q", chunks[0].Text)
	}
}

func TestChunkParagraphsEmpty(t *testing.T) {
	chunks := chunkParagraphs("")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestChunkParagraphsLongParagraph(t *testing.T) {
	// Single paragraph exceeding hard cap.
	long := strings.Repeat("This is a sentence. ", 200)
	chunks := chunkParagraphs(long)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for long paragraph, got %d", len(chunks))
	}
}

func TestExtractHTMLText(t *testing.T) {
	html := `<html><head><title>Test</title><style>body{}</style></head>
		<body><h1>Hello</h1><p>World</p><script>alert(1)</script></body></html>`
	text, err := ExtractHTMLText(html)
	if err != nil {
		t.Fatalf("ExtractHTMLText: %v", err)
	}
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("text = %q, missing expected content", text)
	}
	if strings.Contains(text, "alert") {
		t.Error("text should not contain script content")
	}
	if strings.Contains(text, "body{}") {
		t.Error("text should not contain style content")
	}
}

func TestSanitizeHTML(t *testing.T) {
	raw := `<p>Hello</p><script>alert("xss")</script><img src="x" onerror="alert(1)">`
	clean := SanitizeHTML(raw)
	if strings.Contains(clean, "<script>") {
		t.Error("sanitized HTML should not contain script tags")
	}
	if strings.Contains(clean, "onerror") {
		t.Error("sanitized HTML should not contain event handlers")
	}
	if !strings.Contains(clean, "Hello") {
		t.Error("sanitized HTML should preserve safe content")
	}
}

func TestKBAddMarkdown(t *testing.T) {
	w := setupWorkspace(t)

	// Create a test markdown file.
	mdPath := filepath.Join(w.Root, "test-doc.md")
	content := "# Test Document\n\nThis is a test paragraph with enough content to be meaningful.\n\nSecond paragraph here."
	os.WriteFile(mdPath, []byte(content), 0o644)

	result, err := w.KBAdd(KBAddInput{
		Source: mdPath,
		Title:  "Test Document",
		Note:   "A test note",
	})
	if err != nil {
		t.Fatalf("KBAdd: %v", err)
	}

	if result.ID != "KB-1" {
		t.Errorf("id = %s", result.ID)
	}
	if result.SourceType != "md" {
		t.Errorf("source_type = %s", result.SourceType)
	}
	if result.ChunkCount == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify file was copied.
	absPath := filepath.Join(w.Root, result.Path)
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("copied file missing: %v", err)
	}
}

func TestKBAddPlainText(t *testing.T) {
	w := setupWorkspace(t)

	txtPath := filepath.Join(w.Root, "notes.txt")
	os.WriteFile(txtPath, []byte("Some plain text notes.\n\nAnother paragraph."), 0o644)

	result, err := w.KBAdd(KBAddInput{Source: txtPath})
	if err != nil {
		t.Fatalf("KBAdd txt: %v", err)
	}

	if result.SourceType != "txt" {
		t.Errorf("source_type = %s", result.SourceType)
	}
}

func TestKBAddHTML(t *testing.T) {
	w := setupWorkspace(t)

	htmlPath := filepath.Join(w.Root, "page.html")
	htmlContent := `<!DOCTYPE html><html><body><h1>Title</h1><p>Content paragraph.</p></body></html>`
	os.WriteFile(htmlPath, []byte(htmlContent), 0o644)

	result, err := w.KBAdd(KBAddInput{
		Source: htmlPath,
		Title:  "HTML Page",
	})
	if err != nil {
		t.Fatalf("KBAdd html: %v", err)
	}

	if result.SourceType != "html" {
		t.Errorf("source_type = %s", result.SourceType)
	}

	// Verify clean sidecar was created.
	docs, _ := w.KBList(KBListFilter{})
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].CleanPath == nil {
		t.Error("expected clean_path for HTML doc")
	}
}

func TestKBList(t *testing.T) {
	w := setupWorkspace(t)

	// Add two docs.
	md := filepath.Join(w.Root, "a.md")
	txt := filepath.Join(w.Root, "b.txt")
	os.WriteFile(md, []byte("Markdown content."), 0o644)
	os.WriteFile(txt, []byte("Text content."), 0o644)

	w.KBAdd(KBAddInput{Source: md, Title: "Doc A"})
	w.KBAdd(KBAddInput{Source: txt, Title: "Doc B"})

	// List all.
	all, err := w.KBList(KBListFilter{})
	if err != nil {
		t.Fatalf("KBList: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all len = %d", len(all))
	}

	// Filter by type.
	mds, err := w.KBList(KBListFilter{SourceType: "md"})
	if err != nil {
		t.Fatalf("KBList md: %v", err)
	}
	if len(mds) != 1 {
		t.Errorf("md len = %d", len(mds))
	}
}

func TestKBRead(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "read-test.md")
	os.WriteFile(md, []byte("# Read Test\n\nParagraph one.\n\nParagraph two."), 0o644)

	w.KBAdd(KBAddInput{Source: md, Title: "Read Test"})

	// Read all chunks.
	result, err := w.KBRead("KB-1", nil)
	if err != nil {
		t.Fatalf("KBRead: %v", err)
	}
	if result.Doc.Title != "Read Test" {
		t.Errorf("title = %q", result.Doc.Title)
	}
	if len(result.Chunks) == 0 {
		t.Error("expected chunks")
	}

	// Read single chunk.
	chunkPos := 0
	single, err := w.KBRead("KB-1", &chunkPos)
	if err != nil {
		t.Fatalf("KBRead chunk: %v", err)
	}
	if len(single.Chunks) != 1 {
		t.Errorf("single chunk len = %d", len(single.Chunks))
	}
}

func TestKBReadNotFound(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.KBRead("KB-999", nil)
	if err == nil {
		t.Fatal("expected error for missing KB doc")
	}
}

func TestKBRemove(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "remove-test.md")
	os.WriteFile(md, []byte("To be removed."), 0o644)

	result, _ := w.KBAdd(KBAddInput{Source: md, Title: "Remove Me"})

	// Verify it exists.
	docs, _ := w.KBList(KBListFilter{})
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc before remove, got %d", len(docs))
	}

	// Remove it.
	if err := w.KBRemove(result.ID); err != nil {
		t.Fatalf("KBRemove: %v", err)
	}

	// Verify it's gone.
	docs, _ = w.KBList(KBListFilter{})
	if len(docs) != 0 {
		t.Errorf("expected 0 docs after remove, got %d", len(docs))
	}

	// Verify file was removed from disk.
	absPath := filepath.Join(w.Root, result.Path)
	if _, err := os.Stat(absPath); err == nil {
		t.Error("file should have been removed from disk")
	}

	// Verify it's in trash.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM trash WHERE original_id = ?", result.ID).Scan(&count)
	if count != 1 {
		t.Errorf("trash count = %d, want 1", count)
	}
}

func TestKBSearch(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "search-test.md")
	os.WriteFile(md, []byte("# Authentication\n\nOAuth flow using GitHub as identity provider.\n\nJWT tokens for session management."), 0o644)

	w.KBAdd(KBAddInput{Source: md, Title: "Auth Guide"})

	results, err := w.KBSearch("OAuth GitHub", 10)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected search results")
	}
}

func TestKBSearchNoResults(t *testing.T) {
	w := setupWorkspace(t)

	results, err := w.KBSearch("nonexistent query xyz", 10)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestKBAddMultipleIDs(t *testing.T) {
	w := setupWorkspace(t)

	for i := 0; i < 3; i++ {
		f := filepath.Join(w.Root, "doc"+string(rune('a'+i))+".md")
		os.WriteFile(f, []byte("Content "+string(rune('a'+i))), 0o644)
		r, err := w.KBAdd(KBAddInput{Source: f})
		if err != nil {
			t.Fatalf("KBAdd %d: %v", i, err)
		}
		expected := "KB-" + string(rune('1'+i))
		if r.ID != expected {
			t.Errorf("id = %s, want %s", r.ID, expected)
		}
	}
}

func TestDetectSourceType(t *testing.T) {
	tests := map[string]string{
		"file.md":       "md",
		"file.markdown": "md",
		"file.html":     "html",
		"file.htm":      "html",
		"file.pdf":      "pdf",
		"file.txt":      "txt",
		"file.log":      "txt",
		"file.csv":      "txt",
	}
	for path, want := range tests {
		got := detectSourceType(path)
		if got != want {
			t.Errorf("detectSourceType(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestTitleFromFilename(t *testing.T) {
	tests := map[string]string{
		"oauth-rfc6749.pdf":     "oauth rfc6749",
		"my_notes.md":           "my notes",
		"simple.txt":            "simple",
		"github-apps-docs.html": "github apps docs",
	}
	for filename, want := range tests {
		got := titleFromFilename(filename)
		if got != want {
			t.Errorf("titleFromFilename(%q) = %q, want %q", filename, got, want)
		}
	}
}

// --- Chunking edge cases ---

func TestChunkParagraphsMergSmallParagraphs(t *testing.T) {
	// Multiple small paragraphs should be merged into one chunk.
	text := "Short one.\n\nShort two.\n\nShort three."
	chunks := chunkParagraphs(text)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 merged chunk, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0].Text, "Short one.") || !strings.Contains(chunks[0].Text, "Short three.") {
		t.Errorf("merged chunk missing paragraphs: %q", chunks[0].Text)
	}
}

func TestChunkParagraphsOffsets(t *testing.T) {
	// Verify CharStart/CharEnd don't overlap and cover the text.
	para1 := strings.Repeat("A", 600)
	para2 := strings.Repeat("B", 600)
	text := para1 + "\n\n" + para2
	chunks := chunkParagraphs(text)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Chunks should not overlap.
	for i := 1; i < len(chunks); i++ {
		if chunks[i].CharStart < chunks[i-1].CharEnd {
			t.Errorf("chunk %d starts at %d but previous ends at %d — overlap",
				i, chunks[i].CharStart, chunks[i-1].CharEnd)
		}
	}
}

func TestChunkParagraphsTargetRange(t *testing.T) {
	// Each chunk (except possibly the last) should be >= chunkTargetMin.
	var paras []string
	for i := 0; i < 20; i++ {
		paras = append(paras, strings.Repeat("word ", 60)) // ~300 chars each
	}
	text := strings.Join(paras, "\n\n")
	chunks := chunkParagraphs(text)

	for i, c := range chunks {
		runeLen := len([]rune(c.Text))
		if i < len(chunks)-1 && runeLen < chunkTargetMin {
			t.Errorf("chunk %d has %d runes, below target min %d", i, runeLen, chunkTargetMin)
		}
		if runeLen > chunkHardCap {
			t.Errorf("chunk %d has %d runes, above hard cap %d", i, runeLen, chunkHardCap)
		}
	}
}

func TestChunkParagraphsWhitespaceOnly(t *testing.T) {
	chunks := chunkParagraphs("   \n\n   \n\n   ")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for whitespace-only, got %d", len(chunks))
	}
}

func TestSplitParagraphs(t *testing.T) {
	text := "First paragraph.\n\nSecond paragraph.\n\n\n\nThird after extra newlines."
	paras := splitParagraphs(text)
	if len(paras) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d: %v", len(paras), paras)
	}
	if paras[0] != "First paragraph." {
		t.Errorf("para[0] = %q", paras[0])
	}
	if paras[2] != "Third after extra newlines." {
		t.Errorf("para[2] = %q", paras[2])
	}
}

func TestFindSentenceBoundary(t *testing.T) {
	text := "First sentence. Second sentence. Third sentence."
	cut := findSentenceBoundary(text, 35)
	// Should cut after "Second sentence." (at position 32 or 33).
	piece := text[:cut]
	if !strings.HasSuffix(strings.TrimSpace(piece), ".") {
		t.Errorf("expected cut at sentence boundary, got %q", piece)
	}
}

func TestChunkMarkdownPreservesContent(t *testing.T) {
	content := "# Heading\n\nParagraph with **bold** and *italic*.\n\nAnother paragraph."
	chunks := ChunkMarkdown(content)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	// All original text should appear across chunks.
	var combined string
	for _, c := range chunks {
		combined += c.Text
	}
	if !strings.Contains(combined, "**bold**") {
		t.Error("markdown formatting lost in chunks")
	}
}

func TestChunkPlainTextPreservesContent(t *testing.T) {
	content := "Line one.\n\nLine two.\n\nLine three."
	chunks := ChunkPlainText(content)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	var combined string
	for _, c := range chunks {
		combined += c.Text
	}
	if !strings.Contains(combined, "Line one.") || !strings.Contains(combined, "Line three.") {
		t.Error("content lost in chunks")
	}
}

// --- HTML extraction edge cases ---

func TestExtractHTMLTextNested(t *testing.T) {
	html := `<div><p>Outer <span>inner <b>bold</b></span> text</p></div>`
	text, err := ExtractHTMLText(html)
	if err != nil {
		t.Fatalf("ExtractHTMLText: %v", err)
	}
	if !strings.Contains(text, "Outer") || !strings.Contains(text, "bold") || !strings.Contains(text, "text") {
		t.Errorf("nested extraction failed: %q", text)
	}
}

func TestExtractHTMLTextNoScript(t *testing.T) {
	html := `<body><noscript>No JS</noscript><template>Template</template><p>Visible</p></body>`
	text, err := ExtractHTMLText(html)
	if err != nil {
		t.Fatalf("ExtractHTMLText: %v", err)
	}
	if strings.Contains(text, "No JS") {
		t.Error("noscript content should be excluded")
	}
	if strings.Contains(text, "Template") {
		t.Error("template content should be excluded")
	}
	if !strings.Contains(text, "Visible") {
		t.Error("visible content missing")
	}
}

func TestChunkHTMLWithParagraphs(t *testing.T) {
	// HTML with enough content to produce multiple chunks.
	var paras []string
	for i := 0; i < 10; i++ {
		paras = append(paras, "<p>"+strings.Repeat("Lorem ipsum dolor sit amet. ", 30)+"</p>")
	}
	html := "<html><body>" + strings.Join(paras, "") + "</body></html>"
	chunks, err := ChunkHTML(html)
	if err != nil {
		t.Fatalf("ChunkHTML: %v", err)
	}
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks from large HTML, got %d", len(chunks))
	}
}

func TestSanitizeHTMLPreservesLinks(t *testing.T) {
	raw := `<p>Click <a href="https://example.com">here</a></p>`
	clean := SanitizeHTML(raw)
	if !strings.Contains(clean, "href") {
		t.Error("sanitizer should preserve links")
	}
	if !strings.Contains(clean, "here") {
		t.Error("sanitizer should preserve link text")
	}
}

func TestSanitizeHTMLRemovesIframes(t *testing.T) {
	raw := `<p>Safe</p><iframe src="https://evil.com"></iframe>`
	clean := SanitizeHTML(raw)
	if strings.Contains(clean, "iframe") {
		t.Error("sanitizer should remove iframes")
	}
}

// --- KB operations edge cases ---

func TestKBAddNoteStored(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "noted.md")
	os.WriteFile(md, []byte("Content here."), 0o644)

	w.KBAdd(KBAddInput{Source: md, Title: "Noted Doc", Note: "Important reference"})

	result, err := w.KBRead("KB-1", nil)
	if err != nil {
		t.Fatalf("KBRead: %v", err)
	}
	if result.Doc.Note == nil || *result.Doc.Note != "Important reference" {
		t.Errorf("note = %v, want %q", result.Doc.Note, "Important reference")
	}
}

func TestKBAddNoNoteIsNil(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "nonoted.md")
	os.WriteFile(md, []byte("No note."), 0o644)

	w.KBAdd(KBAddInput{Source: md, Title: "No Note"})

	result, err := w.KBRead("KB-1", nil)
	if err != nil {
		t.Fatalf("KBRead: %v", err)
	}
	if result.Doc.Note != nil {
		t.Errorf("note should be nil, got %q", *result.Doc.Note)
	}
}

func TestKBAddTitleFromFilename(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "my-great-doc.md")
	os.WriteFile(md, []byte("Content."), 0o644)

	// No explicit title — should derive from filename.
	w.KBAdd(KBAddInput{Source: md})

	docs, _ := w.KBList(KBListFilter{})
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Title != "my great doc" {
		t.Errorf("title = %q, want %q", docs[0].Title, "my great doc")
	}
}

func TestKBAddSourceNotFound(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.KBAdd(KBAddInput{Source: "/nonexistent/file.md"})
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestKBListEmpty(t *testing.T) {
	w := setupWorkspace(t)

	docs, err := w.KBList(KBListFilter{})
	if err != nil {
		t.Fatalf("KBList: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestKBListFilterNoMatch(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "only-md.md")
	os.WriteFile(md, []byte("Markdown."), 0o644)
	w.KBAdd(KBAddInput{Source: md})

	docs, err := w.KBList(KBListFilter{SourceType: "pdf"})
	if err != nil {
		t.Fatalf("KBList: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 pdf docs, got %d", len(docs))
	}
}

func TestKBReadChunkNotFound(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "chunktest.md")
	os.WriteFile(md, []byte("Short content."), 0o644)
	w.KBAdd(KBAddInput{Source: md})

	badPos := 999
	_, err := w.KBRead("KB-1", &badPos)
	if err == nil {
		t.Fatal("expected error for nonexistent chunk position")
	}
}

func TestKBRemoveNotFound(t *testing.T) {
	w := setupWorkspace(t)

	err := w.KBRemove("KB-999")
	if err == nil {
		t.Fatal("expected error for removing nonexistent KB doc")
	}
}

func TestKBRemoveHTMLCleansSidecar(t *testing.T) {
	w := setupWorkspace(t)

	htmlPath := filepath.Join(w.Root, "removable.html")
	os.WriteFile(htmlPath, []byte("<p>Remove me</p>"), 0o644)

	result, _ := w.KBAdd(KBAddInput{Source: htmlPath, Title: "Removable HTML"})

	// Verify clean sidecar exists.
	docs, _ := w.KBList(KBListFilter{})
	if docs[0].CleanPath == nil {
		t.Fatal("expected clean_path")
	}
	cleanAbs := filepath.Join(w.Root, *docs[0].CleanPath)
	if _, err := os.Stat(cleanAbs); err != nil {
		t.Fatalf("clean sidecar should exist: %v", err)
	}

	// Remove.
	w.KBRemove(result.ID)

	// Verify both files removed.
	if _, err := os.Stat(filepath.Join(w.Root, result.Path)); err == nil {
		t.Error("main file should be removed")
	}
	if _, err := os.Stat(cleanAbs); err == nil {
		t.Error("clean sidecar should be removed")
	}
}

func TestKBRemoveCascadesChunks(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "cascade.md")
	os.WriteFile(md, []byte("Chunk content here.\n\nMore content."), 0o644)
	result, _ := w.KBAdd(KBAddInput{Source: md})

	// Verify chunks exist.
	var chunkCount int
	w.DB.QueryRow("SELECT count(*) FROM kb_chunks WHERE doc_id = ?", result.ID).Scan(&chunkCount)
	if chunkCount == 0 {
		t.Fatal("expected chunks before remove")
	}

	w.KBRemove(result.ID)

	// Verify chunks are gone.
	w.DB.QueryRow("SELECT count(*) FROM kb_chunks WHERE doc_id = ?", result.ID).Scan(&chunkCount)
	if chunkCount != 0 {
		t.Errorf("expected 0 chunks after remove, got %d", chunkCount)
	}
}

func TestKBSearchMultipleDocs(t *testing.T) {
	w := setupWorkspace(t)

	// Add two docs with distinct content.
	md1 := filepath.Join(w.Root, "auth.md")
	os.WriteFile(md1, []byte("OAuth authentication flow with access tokens and refresh tokens."), 0o644)
	w.KBAdd(KBAddInput{Source: md1, Title: "Auth Guide"})

	md2 := filepath.Join(w.Root, "deploy.md")
	os.WriteFile(md2, []byte("Kubernetes deployment strategies including rolling updates and blue green."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "Deploy Guide"})

	// Search for auth-related content.
	results, err := w.KBSearch("OAuth tokens", 10)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'OAuth tokens'")
	}
	// First result should be from the auth doc.
	if results[0].DocID != "KB-1" {
		t.Errorf("top result doc_id = %s, want KB-1", results[0].DocID)
	}
}

func TestKBSearchLimit(t *testing.T) {
	w := setupWorkspace(t)

	// Add several docs all containing "important".
	for i := 0; i < 5; i++ {
		f := filepath.Join(w.Root, strings.Repeat("x", i+1)+".md")
		os.WriteFile(f, []byte("This is an important document about important things."), 0o644)
		w.KBAdd(KBAddInput{Source: f})
	}

	results, err := w.KBSearch("important", 2)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestKBSearchEmptyQuery(t *testing.T) {
	w := setupWorkspace(t)

	results, err := w.KBSearch("", 10)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestKBSearchResultFields(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "fields.md")
	os.WriteFile(md, []byte("Specific searchable content about database indexing and query optimization."), 0o644)
	w.KBAdd(KBAddInput{Source: md, Title: "DB Indexing"})

	results, err := w.KBSearch("database indexing", 10)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	r := results[0]
	if r.DocID == "" {
		t.Error("doc_id is empty")
	}
	if r.DocTitle == "" {
		t.Error("doc_title is empty")
	}
	if r.Text == "" {
		t.Error("text is empty")
	}
	if r.MatchType != "bm25" && r.MatchType != "trigram" {
		t.Errorf("match_type = %q, want bm25 or trigram", r.MatchType)
	}
	if r.Score < 0 {
		t.Errorf("score = %f, expected non-negative", r.Score)
	}
}

func TestTypeFromContentType(t *testing.T) {
	tests := map[string]string{
		"text/html; charset=utf-8":    "html",
		"application/pdf":             "pdf",
		"text/markdown":               "md",
		"text/plain":                  "txt",
		"application/octet-stream":    "txt",
		"text/html":                   "html",
	}
	for ct, want := range tests {
		got := typeFromContentType(ct)
		if got != want {
			t.Errorf("typeFromContentType(%q) = %q, want %q", ct, got, want)
		}
	}
}

func TestExtForType(t *testing.T) {
	tests := map[string]string{
		"md":   ".md",
		"html": ".html",
		"pdf":  ".pdf",
		"txt":  ".txt",
	}
	for srcType, want := range tests {
		got := extForType(srcType)
		if got != want {
			t.Errorf("extForType(%q) = %q, want %q", srcType, got, want)
		}
	}
}

func TestKBAddCopiesFileContent(t *testing.T) {
	w := setupWorkspace(t)

	content := "Original content that should be preserved exactly."
	md := filepath.Join(w.Root, "preserve.md")
	os.WriteFile(md, []byte(content), 0o644)

	result, _ := w.KBAdd(KBAddInput{Source: md, Title: "Preserve"})

	// Read the copied file and verify content matches.
	copied, err := os.ReadFile(filepath.Join(w.Root, result.Path))
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(copied) != content {
		t.Errorf("copied content = %q, want %q", string(copied), content)
	}
}

func TestKBAddHTMLSidecarIsSanitized(t *testing.T) {
	w := setupWorkspace(t)

	htmlPath := filepath.Join(w.Root, "xss.html")
	os.WriteFile(htmlPath, []byte(`<p>Safe</p><script>alert("xss")</script>`), 0o644)

	w.KBAdd(KBAddInput{Source: htmlPath, Title: "XSS Test"})

	docs, _ := w.KBList(KBListFilter{})
	if docs[0].CleanPath == nil {
		t.Fatal("expected clean_path")
	}

	// Read the sidecar and verify script is removed.
	clean, _ := os.ReadFile(filepath.Join(w.Root, *docs[0].CleanPath))
	if strings.Contains(string(clean), "<script>") {
		t.Error("clean sidecar should not contain script tags")
	}
	if !strings.Contains(string(clean), "Safe") {
		t.Error("clean sidecar should preserve safe content")
	}
}

func TestKBAddContentHashStored(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "hashed.md")
	os.WriteFile(md, []byte("Hash me."), 0o644)
	w.KBAdd(KBAddInput{Source: md})

	docs, _ := w.KBList(KBListFilter{})
	if docs[0].ContentHash == "" {
		t.Error("content_hash should not be empty")
	}
}

func TestKBSearchOnEmptyKB(t *testing.T) {
	w := setupWorkspace(t)

	results, err := w.KBSearch("anything", 10)
	if err != nil {
		t.Fatalf("KBSearch empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty KB, got %d", len(results))
	}
}

func TestKBReadAllChunksOrdered(t *testing.T) {
	w := setupWorkspace(t)

	// Create doc with enough content for multiple chunks.
	var paras []string
	for i := 0; i < 10; i++ {
		paras = append(paras, strings.Repeat("Paragraph content here. ", 30))
	}
	md := filepath.Join(w.Root, "ordered.md")
	os.WriteFile(md, []byte(strings.Join(paras, "\n\n")), 0o644)
	w.KBAdd(KBAddInput{Source: md})

	result, err := w.KBRead("KB-1", nil)
	if err != nil {
		t.Fatalf("KBRead: %v", err)
	}

	// Verify chunks are in order.
	for i, c := range result.Chunks {
		if c.Position != i {
			t.Errorf("chunk %d has position %d", i, c.Position)
		}
	}

	// Verify positions are sequential.
	for i := 1; i < len(result.Chunks); i++ {
		if result.Chunks[i].CharStart < result.Chunks[i-1].CharEnd {
			t.Errorf("chunk %d overlaps with %d", i, i-1)
		}
	}
}
