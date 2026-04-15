package workspace

import (
	"fmt"
	"testing"
)

func TestSearchSpecsBM25(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "User authentication", Type: "technical", Summary: "Login and session management"})
	w.NewSpec(NewSpecInput{Title: "Payment processing", Type: "business", Summary: "Stripe integration for billing"})
	w.NewSpec(NewSpecInput{Title: "OAuth provider", Type: "technical", Summary: "OAuth authentication flow"})

	results, err := w.Search("authentication", "spec", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Specs) < 1 {
		t.Fatal("should find at least one spec")
	}

	// Both auth-related specs should appear.
	ids := map[string]bool{}
	for _, r := range results.Specs {
		ids[r.ID] = true
		if r.MatchType != "bm25" && r.MatchType != "trigram" {
			t.Errorf("unexpected match type: %s", r.MatchType)
		}
	}
	if !ids["SPEC-1"] {
		t.Error("SPEC-1 should be in results")
	}
}

func TestSearchTasksBM25(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "technical", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Design database schema", Summary: "Schema for users and sessions"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Write unit tests", Summary: "Tests for auth middleware"})

	results, err := w.Search("schema", "task", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Tasks) < 1 {
		t.Fatal("should find at least one task")
	}
	if results.Tasks[0].ID != "TASK-1" {
		t.Errorf("first result should be TASK-1, got %s", results.Tasks[0].ID)
	}
}

func TestSearchKindFilter(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth system", Type: "technical", Summary: "Authentication"})
	w.NewSpec(NewSpecInput{Title: "S2", Type: "technical", Summary: "S2"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-2", Title: "Auth task", Summary: "Authentication task"})

	// Search only specs.
	results, err := w.Search("auth", "spec", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Tasks) != 0 {
		t.Error("should not return tasks when kind=spec")
	}

	// Search only tasks.
	results, err = w.Search("auth", "task", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Specs) != 0 {
		t.Error("should not return specs when kind=task")
	}
}

func TestSearchAll(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth system", Type: "technical", Summary: "Authentication and authorization"})
	w.NewSpec(NewSpecInput{Title: "S2", Type: "technical", Summary: "S2"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-2", Title: "Auth middleware", Summary: "Auth middleware implementation"})

	results, err := w.Search("auth", "all", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Specs) < 1 {
		t.Error("should find spec results")
	}
	if len(results.Tasks) < 1 {
		t.Error("should find task results")
	}
}

func TestSearchLimit(t *testing.T) {
	w := setupWorkspace(t)

	for i := 0; i < 5; i++ {
		w.NewSpec(NewSpecInput{
			Title:   fmt.Sprintf("Auth spec %d", i),
			Type:    "technical",
			Summary: "Authentication related",
		})
	}

	results, err := w.Search("auth", "spec", 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Specs) > 2 {
		t.Errorf("should respect limit, got %d", len(results.Specs))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	w := setupWorkspace(t)

	results, err := w.Search("", "all", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results.Specs) != 0 || len(results.Tasks) != 0 || len(results.KB) != 0 {
		t.Error("empty query should return no results")
	}
}

func TestSearchTrigramFallback(t *testing.T) {
	w := setupWorkspace(t)

	// Create a spec with a unique substring that FTS5 porter stemming might not match well.
	w.NewSpec(NewSpecInput{Title: "The xyzfoo handler", Type: "technical", Summary: "Handles xyzfoo requests"})

	// Trigram search for a substring.
	results, err := w.Search("xyzfoo", "spec", 20)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Specs) < 1 {
		t.Fatal("should find the spec via BM25 or trigram")
	}
}

func TestSearchWithFTSOperators(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "OAuth Authentication", Type: "technical", Summary: "OAuth auth flow"})
	w.NewSpec(NewSpecInput{Title: "Database Schema", Type: "technical", Summary: "DB design"})

	results, err := w.Search("OAuth AND Authentication", "spec", 10)
	if err != nil {
		t.Fatalf("Search AND: %v", err)
	}
	if len(results.Specs) == 0 {
		t.Error("expected results for AND query")
	}
}

func TestSearchQuotedPhrase(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "OAuth Flow Design", Type: "technical", Summary: "OAuth flow"})

	results, err := w.Search(`"OAuth Flow"`, "spec", 10)
	if err != nil {
		t.Fatalf("Search quoted: %v", err)
	}
	if len(results.Specs) == 0 {
		t.Error("expected results for quoted phrase")
	}
}

func TestSearchKBEmpty(t *testing.T) {
	w := setupWorkspace(t)

	results, err := w.Search("anything", "kb", 10)
	if err != nil {
		t.Fatalf("Search empty KB: %v", err)
	}
	if len(results.KB) != 0 {
		t.Errorf("expected 0 KB results, got %d", len(results.KB))
	}
}
