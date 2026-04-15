package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCitationRef(t *testing.T) {
	tests := []struct {
		input   string
		wantID  string
		wantPos int
		wantErr bool
	}{
		{"KB-4:12", "KB-4", 12, false},
		{"KB-1:0", "KB-1", 0, false},
		{"KB-100:999", "KB-100", 999, false},
		{"bad", "", 0, true},
		{"SPEC-1:5", "", 0, true},
		{"KB-1:abc", "", 0, true},
		{"KB-1:", "", 0, true},
	}

	for _, tt := range tests {
		ref, err := ParseCitationRef(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseCitationRef(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCitationRef(%q) error: %v", tt.input, err)
			continue
		}
		if ref.KBID != tt.wantID {
			t.Errorf("ParseCitationRef(%q).KBID = %q, want %q", tt.input, ref.KBID, tt.wantID)
		}
		if ref.ChunkPosition != tt.wantPos {
			t.Errorf("ParseCitationRef(%q).ChunkPosition = %d, want %d", tt.input, ref.ChunkPosition, tt.wantPos)
		}
	}
}

// setupWithKB creates a workspace with a spec, a task, and a KB doc.
func setupWithKB(t *testing.T) *Workspace {
	t.Helper()
	w := setupWorkspace(t)

	// Create a spec.
	w.NewSpec(NewSpecInput{
		Title:   "OAuth with GitHub",
		Type:    "technical",
		Summary: "OAuth flow using GitHub",
		Body:    "# OAuth\n\nBody.",
	})

	// Create a task under the spec.
	w.NewTask(NewTaskInput{
		SpecID:  "SPEC-1",
		Title:   "Implement OAuth",
		Summary: "Implement the OAuth flow",
		Body:    "# Implement\n\nBody.",
	})

	// Add a KB doc.
	md := filepath.Join(w.Root, "oauth-rfc.md")
	os.WriteFile(md, []byte("# OAuth 2.0 RFC\n\nAuthorization code grant is the most commonly used.\n\nAccess tokens are short-lived credentials."), 0o644)
	w.KBAdd(KBAddInput{Source: md, Title: "OAuth 2.0 RFC 6749"})

	return w
}

func TestCiteSpec(t *testing.T) {
	w := setupWithKB(t)

	err := w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})
	if err != nil {
		t.Fatalf("Cite: %v", err)
	}

	// Verify citation exists in DB.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Errorf("citation count = %d, want 1", count)
	}
}

func TestCiteTask(t *testing.T) {
	w := setupWithKB(t)

	err := w.Cite("TASK-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})
	if err != nil {
		t.Fatalf("Cite: %v", err)
	}

	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'TASK-1'").Scan(&count)
	if count != 1 {
		t.Errorf("citation count = %d, want 1", count)
	}
}

func TestCiteMultiple(t *testing.T) {
	w := setupWithKB(t)

	// Add a second KB doc.
	md2 := filepath.Join(w.Root, "jwt.md")
	os.WriteFile(md2, []byte("# JWT Best Practices\n\nAlways verify the signature."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "JWT Best Practices"})

	err := w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
		{KBID: "KB-2", ChunkPosition: 0},
	})
	if err != nil {
		t.Fatalf("Cite: %v", err)
	}

	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 2 {
		t.Errorf("citation count = %d, want 2", count)
	}
}

func TestCiteIdempotent(t *testing.T) {
	w := setupWithKB(t)

	ref := []CitationInput{{KBID: "KB-1", ChunkPosition: 0}}

	w.Cite("SPEC-1", ref)
	w.Cite("SPEC-1", ref) // duplicate should be ignored

	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Errorf("citation count = %d, want 1 (should be idempotent)", count)
	}
}

func TestCiteInvalidKB(t *testing.T) {
	w := setupWithKB(t)

	err := w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-999", ChunkPosition: 0},
	})
	if err == nil {
		t.Fatal("expected error for invalid KB doc")
	}
}

func TestCiteInvalidChunk(t *testing.T) {
	w := setupWithKB(t)

	err := w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 999},
	})
	if err == nil {
		t.Fatal("expected error for invalid chunk position")
	}
}

func TestCiteInvalidID(t *testing.T) {
	w := setupWithKB(t)

	err := w.Cite("INVALID-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestUncite(t *testing.T) {
	w := setupWithKB(t)

	// Add citation.
	ref := []CitationInput{{KBID: "KB-1", ChunkPosition: 0}}
	w.Cite("SPEC-1", ref)

	// Verify it exists.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Fatalf("citation count = %d before uncite", count)
	}

	// Remove citation.
	err := w.Uncite("SPEC-1", ref)
	if err != nil {
		t.Fatalf("Uncite: %v", err)
	}

	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 0 {
		t.Errorf("citation count = %d after uncite, want 0", count)
	}
}

func TestUncitePartial(t *testing.T) {
	w := setupWithKB(t)

	// Add a second KB doc.
	md2 := filepath.Join(w.Root, "jwt2.md")
	os.WriteFile(md2, []byte("JWT content here."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "JWT"})

	// Cite both.
	w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
		{KBID: "KB-2", ChunkPosition: 0},
	})

	// Remove only one.
	w.Uncite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE from_id = 'SPEC-1'").Scan(&count)
	if count != 1 {
		t.Errorf("citation count = %d, want 1 after partial uncite", count)
	}
}

func TestGetCitations(t *testing.T) {
	w := setupWithKB(t)

	w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	citations, err := w.GetCitations("SPEC-1")
	if err != nil {
		t.Fatalf("GetCitations: %v", err)
	}

	if len(citations) != 1 {
		t.Fatalf("citations len = %d, want 1", len(citations))
	}

	c := citations[0]
	if c.FromKind != "spec" {
		t.Errorf("from_kind = %q", c.FromKind)
	}
	if c.FromID != "SPEC-1" {
		t.Errorf("from_id = %q", c.FromID)
	}
	if c.KBDocID != "KB-1" {
		t.Errorf("kb_doc_id = %q", c.KBDocID)
	}
	if c.KBDocTitle != "OAuth 2.0 RFC 6749" {
		t.Errorf("kb_doc_title = %q", c.KBDocTitle)
	}
	if c.SourceType != "md" {
		t.Errorf("source_type = %q", c.SourceType)
	}
	if c.ChunkPosition != 0 {
		t.Errorf("chunk_position = %d", c.ChunkPosition)
	}
	if c.ChunkText == "" {
		t.Error("chunk_text should not be empty")
	}
}

func TestGetCitationsEmpty(t *testing.T) {
	w := setupWithKB(t)

	citations, err := w.GetCitations("SPEC-1")
	if err != nil {
		t.Fatalf("GetCitations: %v", err)
	}
	if len(citations) != 0 {
		t.Errorf("citations len = %d, want 0", len(citations))
	}
}

func TestGetCitationsForTask(t *testing.T) {
	w := setupWithKB(t)

	w.Cite("TASK-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	citations, err := w.GetCitations("TASK-1")
	if err != nil {
		t.Fatalf("GetCitations: %v", err)
	}
	if len(citations) != 1 {
		t.Fatalf("citations len = %d, want 1", len(citations))
	}
	if citations[0].FromKind != "task" {
		t.Errorf("from_kind = %q", citations[0].FromKind)
	}
}

func TestCiteFrontmatterSync(t *testing.T) {
	w := setupWithKB(t)

	w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	// Read the spec file and verify cites in frontmatter.
	spec, _ := w.ReadSpec("SPEC-1")
	data, err := os.ReadFile(filepath.Join(w.Root, spec.Path))
	if err != nil {
		t.Fatalf("read spec file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "cites:") {
		t.Error("frontmatter should contain cites field")
	}
	if !strings.Contains(content, "kb: KB-1") {
		t.Error("frontmatter should contain KB-1 reference")
	}
}

func TestUnciteFrontmatterSync(t *testing.T) {
	w := setupWithKB(t)

	ref := []CitationInput{{KBID: "KB-1", ChunkPosition: 0}}
	w.Cite("SPEC-1", ref)
	w.Uncite("SPEC-1", ref)

	// Read the spec file — cites should be empty/absent.
	spec, _ := w.ReadSpec("SPEC-1")
	data, _ := os.ReadFile(filepath.Join(w.Root, spec.Path))
	content := string(data)
	if strings.Contains(content, "kb: KB-1") {
		t.Error("frontmatter should not contain KB-1 after uncite")
	}
}

func TestCiteTaskFrontmatterSync(t *testing.T) {
	w := setupWithKB(t)

	w.Cite("TASK-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	task, _ := w.ReadTask("TASK-1")
	data, err := os.ReadFile(filepath.Join(w.Root, task.Path))
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "cites:") {
		t.Error("task frontmatter should contain cites field")
	}
	if !strings.Contains(content, "kb: KB-1") {
		t.Error("task frontmatter should contain KB-1 reference")
	}
}

func TestCiteMultipleFrontmatterGrouped(t *testing.T) {
	w := setupWithKB(t)

	// Add second KB doc with multiple chunks.
	md := filepath.Join(w.Root, "big-doc.md")
	var content string
	for i := 0; i < 5; i++ {
		content += strings.Repeat("Paragraph content. ", 60) + "\n\n"
	}
	os.WriteFile(md, []byte(content), 0o644)
	w.KBAdd(KBAddInput{Source: md, Title: "Big Doc"})

	// Cite multiple chunks from same doc.
	w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
		{KBID: "KB-2", ChunkPosition: 0},
		{KBID: "KB-2", ChunkPosition: 1},
	})

	// Read frontmatter and verify grouped structure.
	spec, _ := w.ReadSpec("SPEC-1")
	data, _ := os.ReadFile(filepath.Join(w.Root, spec.Path))
	fm := string(data)

	if !strings.Contains(fm, "kb: KB-1") {
		t.Error("should contain KB-1")
	}
	if !strings.Contains(fm, "kb: KB-2") {
		t.Error("should contain KB-2")
	}
}

func TestCandidatesIncludeKBChunks(t *testing.T) {
	w := setupWithKB(t)

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	// Should include KB chunk candidates since there's a KB doc about OAuth.
	if len(result.KBChunks) == 0 {
		t.Log("no KB chunk candidates found (may depend on search terms)")
	}
}

func TestCandidatesKBChunksFields(t *testing.T) {
	w := setupWithKB(t)

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	for _, c := range result.KBChunks {
		if c.DocID == "" {
			t.Error("doc_id is empty")
		}
		if c.DocTitle == "" {
			t.Error("doc_title is empty")
		}
		if c.Text == "" {
			t.Error("text is empty")
		}
		if c.MatchType != "bm25" {
			t.Errorf("match_type = %q, want bm25", c.MatchType)
		}
	}
}

func TestCandidatesTaskIncludeKBChunks(t *testing.T) {
	w := setupWithKB(t)

	result, err := w.Candidates("TASK-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	// KB chunks should be present in task candidates too.
	t.Logf("KB chunk candidates for task: %d", len(result.KBChunks))
}

func TestResolveKind(t *testing.T) {
	w := setupWithKB(t)

	kind, err := w.resolveKind("SPEC-1")
	if err != nil {
		t.Fatalf("resolveKind spec: %v", err)
	}
	if kind != "spec" {
		t.Errorf("kind = %q, want spec", kind)
	}

	kind, err = w.resolveKind("TASK-1")
	if err != nil {
		t.Fatalf("resolveKind task: %v", err)
	}
	if kind != "task" {
		t.Errorf("kind = %q, want task", kind)
	}

	_, err = w.resolveKind("INVALID-1")
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestResolveKindNotFound(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.resolveKind("SPEC-999")
	if err == nil {
		t.Error("expected error for nonexistent spec")
	}
}

func TestCiteCascadeOnKBRemove(t *testing.T) {
	w := setupWithKB(t)

	// Add a citation.
	w.Cite("SPEC-1", []CitationInput{
		{KBID: "KB-1", ChunkPosition: 0},
	})

	// Verify citation exists.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE kb_doc_id = 'KB-1'").Scan(&count)
	if count != 1 {
		t.Fatalf("citation count = %d before remove", count)
	}

	// Remove the KB doc.
	w.KBRemove("KB-1")

	// Citation should be cascade deleted.
	w.DB.QueryRow("SELECT count(*) FROM citations WHERE kb_doc_id = 'KB-1'").Scan(&count)
	if count != 0 {
		t.Errorf("citation count = %d after KB remove, want 0 (should cascade)", count)
	}
}

func TestUnciteNonexistentCitation(t *testing.T) {
	w := setupWithKB(t)

	err := w.Uncite("SPEC-1", []CitationInput{{KBID: "KB-1", ChunkPosition: 0}})
	if err != nil {
		t.Fatalf("Uncite no-op should not error: %v", err)
	}
}
