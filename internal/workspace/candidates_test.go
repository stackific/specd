package workspace

import "testing"

func TestCandidatesExcludesSelf(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth system", Type: "functional", Summary: "Authentication and authorization"})
	w.NewSpec(NewSpecInput{Title: "OAuth provider", Type: "functional", Summary: "OAuth authentication flow"})
	w.NewSpec(NewSpecInput{Title: "Billing", Type: "business", Summary: "Payment processing"})

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	for _, c := range result.Specs {
		if c.ID == "SPEC-1" {
			t.Error("candidates should exclude self")
		}
	}
}

func TestCandidatesExcludesLinked(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth system", Type: "functional", Summary: "Authentication"})
	w.NewSpec(NewSpecInput{Title: "Auth OAuth", Type: "functional", Summary: "OAuth authentication"})
	w.NewSpec(NewSpecInput{Title: "Auth tokens", Type: "functional", Summary: "Token authentication"})

	w.Link("SPEC-1", "SPEC-2")

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	for _, c := range result.Specs {
		if c.ID == "SPEC-2" {
			t.Error("candidates should exclude already-linked specs")
		}
	}
}

func TestCandidatesRankedByScore(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "User authentication system", Type: "functional", Summary: "Auth with users and sessions"})
	w.NewSpec(NewSpecInput{Title: "User session management", Type: "functional", Summary: "Session handling for authenticated users"})
	w.NewSpec(NewSpecInput{Title: "Logging infrastructure", Type: "functional", Summary: "Centralized log aggregation"})

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	if len(result.Specs) < 1 {
		t.Fatal("should find at least one candidate")
	}

	// SPEC-2 should score higher than SPEC-3 (more word overlap).
	if len(result.Specs) >= 2 {
		if result.Specs[0].ID != "SPEC-2" {
			t.Errorf("first candidate should be SPEC-2 (most similar), got %s", result.Specs[0].ID)
		}
	}
}

func TestCandidatesLimit(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "Auth A", Type: "functional", Summary: "Auth A"})
	w.NewSpec(NewSpecInput{Title: "Auth B", Type: "functional", Summary: "Auth B"})
	w.NewSpec(NewSpecInput{Title: "Auth C", Type: "functional", Summary: "Auth C"})
	w.NewSpec(NewSpecInput{Title: "Auth D", Type: "functional", Summary: "Auth D"})

	result, err := w.Candidates("SPEC-1", 2)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	if len(result.Specs) > 2 {
		t.Errorf("should respect limit, got %d", len(result.Specs))
	}
}

func TestTaskCandidates(t *testing.T) {
	w := setupWorkspace(t)

	w.NewSpec(NewSpecInput{Title: "S", Type: "functional", Summary: "S"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Design auth schema", Summary: "Database schema for auth"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Implement auth middleware", Summary: "Auth middleware for routes"})
	w.NewTask(NewTaskInput{SpecID: "SPEC-1", Title: "Write billing tests", Summary: "Unit tests for billing"})

	result, err := w.Candidates("TASK-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}

	if len(result.Tasks) < 1 {
		t.Fatal("should find at least one task candidate")
	}

	// TASK-2 has "auth" overlap, should rank higher than TASK-3.
	if result.Tasks[0].ID != "TASK-2" {
		t.Errorf("first task candidate should be TASK-2, got %s", result.Tasks[0].ID)
	}
}

func TestCandidatesInvalidID(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.Candidates("INVALID-1", 20)
	if err == nil {
		t.Fatal("expected error for invalid ID format")
	}
}

func TestCandidatesNoMatches(t *testing.T) {
	w := setupWorkspace(t)
	w.NewSpec(NewSpecInput{Title: "X", Type: "functional", Summary: "X"})

	result, err := w.Candidates("SPEC-1", 20)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(result.Specs) != 0 {
		t.Errorf("expected 0 spec candidates, got %d", len(result.Specs))
	}
}
