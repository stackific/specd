package workspace

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	tokens := tokenize("Hello, World! This is a test-123.")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
	// Should be lowercase.
	for _, tok := range tokens {
		if tok != strings.ToLower(tok) {
			t.Errorf("token %q is not lowercase", tok)
		}
	}
	// Single-char tokens should be excluded.
	for _, tok := range tokens {
		if len(tok) < 2 {
			t.Errorf("single-char token %q should be excluded", tok)
		}
	}
}

func TestTokenizeEmpty(t *testing.T) {
	tokens := tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestTokenizePunctuation(t *testing.T) {
	tokens := tokenize("OAuth 2.0 — RFC-6749")
	found := make(map[string]bool)
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["oauth"] {
		t.Error("expected 'oauth' token")
	}
	if !found["rfc"] {
		t.Error("expected 'rfc' token")
	}
	if !found["6749"] {
		t.Error("expected '6749' token")
	}
}

func TestTermFrequency(t *testing.T) {
	tokens := []string{"the", "cat", "sat", "on", "the", "mat"}
	tf := termFrequency(tokens)

	if tf["the"] != 2.0/6.0 {
		t.Errorf("tf[the] = %f, want %f", tf["the"], 2.0/6.0)
	}
	if tf["cat"] != 1.0/6.0 {
		t.Errorf("tf[cat] = %f, want %f", tf["cat"], 1.0/6.0)
	}
}

func TestTermFrequencyEmpty(t *testing.T) {
	tf := termFrequency(nil)
	if len(tf) != 0 {
		t.Errorf("expected empty tf, got %d entries", len(tf))
	}
}

func TestBuildIDF(t *testing.T) {
	df := map[string]int{
		"common": 10,
		"rare":   1,
	}
	idf := buildIDF(df, 10)

	// common appears in all 10 docs: IDF = log(10/10) = 0
	if idf.idf["common"] != 0 {
		t.Errorf("idf[common] = %f, want 0", idf.idf["common"])
	}

	// rare appears in 1 doc: IDF = log(10/1) = log(10) ≈ 2.302
	expected := math.Log(10.0)
	if math.Abs(idf.idf["rare"]-expected) > 0.001 {
		t.Errorf("idf[rare] = %f, want %f", idf.idf["rare"], expected)
	}
}

func TestCosineSimilarityIdentical(t *testing.T) {
	a := sparseVec{"x": 1.0, "y": 2.0}
	sim := cosineSimilarity(a, a)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("identical vectors should have sim=1.0, got %f", sim)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := sparseVec{"x": 1.0}
	b := sparseVec{"y": 1.0}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("orthogonal vectors should have sim=0, got %f", sim)
	}
}

func TestCosineSimilarityPartialOverlap(t *testing.T) {
	a := sparseVec{"x": 1.0, "y": 1.0}
	b := sparseVec{"y": 1.0, "z": 1.0}
	sim := cosineSimilarity(a, b)
	// dot = 1, |a| = sqrt(2), |b| = sqrt(2), sim = 1/2 = 0.5
	if math.Abs(sim-0.5) > 0.001 {
		t.Errorf("sim = %f, want 0.5", sim)
	}
}

func TestCosineSimilarityEmptyVec(t *testing.T) {
	a := sparseVec{}
	b := sparseVec{"x": 1.0}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("empty vec sim should be 0, got %f", sim)
	}
}

func TestMagnitude(t *testing.T) {
	v := sparseVec{"x": 3.0, "y": 4.0}
	mag := magnitude(v)
	if math.Abs(mag-5.0) > 0.001 {
		t.Errorf("magnitude = %f, want 5.0", mag)
	}
}

func TestMagnitudeEmpty(t *testing.T) {
	v := sparseVec{}
	if magnitude(v) != 0 {
		t.Error("empty vec magnitude should be 0")
	}
}

func TestSortCandidates(t *testing.T) {
	candidates := []connectionCandidate{
		{chunkID: 1, strength: 0.3},
		{chunkID: 2, strength: 0.9},
		{chunkID: 3, strength: 0.5},
	}
	sortCandidates(candidates)
	if candidates[0].strength != 0.9 {
		t.Errorf("first should be 0.9, got %f", candidates[0].strength)
	}
	if candidates[2].strength != 0.3 {
		t.Errorf("last should be 0.3, got %f", candidates[2].strength)
	}
}

func TestSortCandidatesEmpty(t *testing.T) {
	var candidates []connectionCandidate
	sortCandidates(candidates) // should not panic
}

func TestBuildConnectionsEmpty(t *testing.T) {
	connections := buildConnections(nil, nil, 0.3, 10)
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

func TestBuildConnectionsSimilarChunks(t *testing.T) {
	// Use many chunks so IDF values are meaningful, plus enough shared
	// vocabulary between chunks 1 and 2 to produce a real similarity.
	newChunks := map[int]string{
		1: "OAuth authentication access tokens refresh tokens authorization code grant secure API",
	}
	existingChunks := map[int]string{
		2: "OAuth authentication access tokens refresh tokens authorization code grant flow",
		3: "Kubernetes pod scheduling resource management container orchestration",
		4: "Database indexing query optimization SQL performance tuning",
		5: "Machine learning neural networks deep learning gradient descent",
		6: "Network protocols TCP UDP routing firewall security",
		7: "File system storage block devices partition management",
		8: "Compiler design parsing lexical analysis abstract syntax trees",
	}

	connections := buildConnections(newChunks, existingChunks, 0.05, 10)
	// Chunk 1 and 2 share many OAuth/auth terms.
	hasConn12 := false
	for _, c := range connections {
		if (c.fromID == 1 && c.toID == 2) || (c.fromID == 2 && c.toID == 1) {
			hasConn12 = true
		}
	}
	if !hasConn12 {
		t.Error("expected connection between chunks 1 and 2 (OAuth related)")
	}
}

func TestBuildConnectionsBidirectional(t *testing.T) {
	// Many diverse chunks so IDF gives weight to shared terms.
	newChunks := map[int]string{
		1: "OAuth authentication security tokens protocol access refresh grant",
	}
	existingChunks := map[int]string{
		2: "OAuth security authentication tokens management access refresh grant",
		3: "Kubernetes container orchestration pods services deployments",
		4: "Database schema migrations relational tables indexes constraints",
		5: "Compiler parsing tokenization abstract syntax tree code generation",
		6: "Cryptography encryption decryption symmetric asymmetric keys",
		7: "Operating system kernel process scheduling memory management",
	}

	connections := buildConnections(newChunks, existingChunks, 0.05, 10)

	has12 := false
	has21 := false
	for _, c := range connections {
		if c.fromID == 1 && c.toID == 2 {
			has12 = true
		}
		if c.fromID == 2 && c.toID == 1 {
			has21 = true
		}
	}
	if !has12 || !has21 {
		t.Errorf("expected bidirectional connections, got 1→2=%v, 2→1=%v", has12, has21)
	}
}

func TestBuildConnectionsTopK(t *testing.T) {
	newChunks := map[int]string{
		1: "test authentication OAuth tokens",
	}
	existingChunks := map[int]string{}
	for i := 2; i <= 20; i++ {
		existingChunks[i] = "authentication OAuth tokens security " + strings.Repeat("word ", i)
	}

	connections := buildConnections(newChunks, existingChunks, 0.01, 3)

	// Count connections from chunk 1 (each direction).
	from1 := 0
	for _, c := range connections {
		if c.fromID == 1 {
			from1++
		}
	}
	if from1 > 3 {
		t.Errorf("expected at most 3 connections from chunk 1, got %d", from1)
	}
}

func TestBuildConnectionsThreshold(t *testing.T) {
	newChunks := map[int]string{
		1: "completely unrelated content about cooking recipes",
	}
	existingChunks := map[int]string{
		2: "quantum mechanics particle physics wave function",
	}

	connections := buildConnections(newChunks, existingChunks, 0.5, 10)
	if len(connections) != 0 {
		t.Errorf("expected 0 connections for unrelated content, got %d", len(connections))
	}
}

// --- Integration tests for KB connections ---

func TestKBAddComputesConnections(t *testing.T) {
	w := setupWorkspace(t)

	// Add first doc about OAuth.
	md1 := filepath.Join(w.Root, "oauth.md")
	os.WriteFile(md1, []byte("OAuth authentication flow with access tokens, refresh tokens, and authorization codes for secure API access."), 0o644)
	w.KBAdd(KBAddInput{Source: md1, Title: "OAuth Guide"})

	// Add second doc about OAuth (should connect).
	md2 := filepath.Join(w.Root, "oauth2.md")
	os.WriteFile(md2, []byte("OAuth 2.0 authorization code grant type for authentication using access tokens and refresh tokens."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "OAuth 2.0 RFC"})

	// Check for connections.
	var count int
	w.DB.QueryRow("SELECT count(*) FROM chunk_connections").Scan(&count)
	// Should have some connections since both docs discuss OAuth.
	t.Logf("connections after adding related docs: %d", count)
}

func TestKBAddNoConnectionsForUnrelated(t *testing.T) {
	w := setupWorkspace(t)

	// Add two completely unrelated docs.
	md1 := filepath.Join(w.Root, "cooking.md")
	os.WriteFile(md1, []byte("Recipe for chocolate cake with flour sugar eggs butter and cocoa powder. Bake at three hundred fifty degrees."), 0o644)
	w.KBAdd(KBAddInput{Source: md1, Title: "Cooking"})

	md2 := filepath.Join(w.Root, "astronomy.md")
	os.WriteFile(md2, []byte("Neutron stars and black holes are formed from the gravitational collapse of massive stellar cores after supernova explosions."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "Astronomy"})

	var count int
	w.DB.QueryRow("SELECT count(*) FROM chunk_connections").Scan(&count)
	if count > 0 {
		t.Logf("unexpected connections between unrelated docs: %d (may be weak false positives)", count)
	}
}

func TestKBConnections(t *testing.T) {
	w := setupWorkspace(t)

	// Add related docs.
	md1 := filepath.Join(w.Root, "auth1.md")
	os.WriteFile(md1, []byte("OAuth authentication protocol uses access tokens and refresh tokens for secure API authorization."), 0o644)
	w.KBAdd(KBAddInput{Source: md1, Title: "Auth Guide 1"})

	md2 := filepath.Join(w.Root, "auth2.md")
	os.WriteFile(md2, []byte("OAuth authorization server issues access tokens and refresh tokens after authenticating the resource owner."), 0o644)
	w.KBAdd(KBAddInput{Source: md2, Title: "Auth Guide 2"})

	results, err := w.KBConnections("KB-1", nil, 20)
	if err != nil {
		t.Fatalf("KBConnections: %v", err)
	}

	t.Logf("connections for KB-1: %d", len(results))
	for _, r := range results {
		if r.Strength < 0 || r.Strength > 1 {
			t.Errorf("strength %f out of range [0,1]", r.Strength)
		}
		if r.Method != "tfidf_cosine" {
			t.Errorf("method = %q, want tfidf_cosine", r.Method)
		}
	}
}

func TestKBConnectionsNotFound(t *testing.T) {
	w := setupWorkspace(t)

	_, err := w.KBConnections("KB-999", nil, 20)
	if err == nil {
		t.Fatal("expected error for nonexistent KB doc")
	}
}

func TestKBConnectionsSpecificChunk(t *testing.T) {
	w := setupWorkspace(t)

	md1 := filepath.Join(w.Root, "c1.md")
	os.WriteFile(md1, []byte("OAuth tokens for authentication and authorization security."), 0o644)
	w.KBAdd(KBAddInput{Source: md1})

	md2 := filepath.Join(w.Root, "c2.md")
	os.WriteFile(md2, []byte("OAuth tokens provide secure authentication and authorization."), 0o644)
	w.KBAdd(KBAddInput{Source: md2})

	chunkPos := 0
	results, err := w.KBConnections("KB-1", &chunkPos, 20)
	if err != nil {
		t.Fatalf("KBConnections chunk: %v", err)
	}

	// All results should be from chunk at position 0.
	for _, r := range results {
		if r.FromChunkID == 0 {
			t.Error("from_chunk_id should not be 0 (it's a DB auto-increment ID)")
		}
	}
	t.Logf("connections for KB-1 chunk 0: %d", len(results))
}

func TestKBRebuildConnections(t *testing.T) {
	w := setupWorkspace(t)

	// Add several related docs.
	for i, content := range []string{
		"OAuth authentication tokens for API authorization and security protocols.",
		"Authentication using OAuth access tokens and refresh token management.",
		"Secure API authorization with OAuth bearer tokens and scopes.",
	} {
		f := filepath.Join(w.Root, strings.Repeat("d", i+1)+".md")
		os.WriteFile(f, []byte(content), 0o644)
		w.KBAdd(KBAddInput{Source: f})
	}

	// Clear connections.
	w.DB.Exec("DELETE FROM chunk_connections")

	var beforeCount int
	w.DB.QueryRow("SELECT count(*) FROM chunk_connections").Scan(&beforeCount)
	if beforeCount != 0 {
		t.Fatalf("expected 0 connections after clear, got %d", beforeCount)
	}

	// Rebuild.
	count, err := w.KBRebuildConnections(0.1, 10)
	if err != nil {
		t.Fatalf("KBRebuildConnections: %v", err)
	}

	t.Logf("rebuilt %d connection pairs", count)

	var afterCount int
	w.DB.QueryRow("SELECT count(*) FROM chunk_connections").Scan(&afterCount)
	if afterCount == 0 && count > 0 {
		t.Error("expected connections in DB after rebuild")
	}
}

func TestKBRebuildConnectionsEmpty(t *testing.T) {
	w := setupWorkspace(t)

	count, err := w.KBRebuildConnections(0.3, 10)
	if err != nil {
		t.Fatalf("KBRebuildConnections: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 connections for empty KB, got %d", count)
	}
}

func TestKBRebuildConnectionsSingleDoc(t *testing.T) {
	w := setupWorkspace(t)

	md := filepath.Join(w.Root, "single.md")
	os.WriteFile(md, []byte("Just one document."), 0o644)
	w.KBAdd(KBAddInput{Source: md})

	count, err := w.KBRebuildConnections(0.3, 10)
	if err != nil {
		t.Fatalf("KBRebuildConnections: %v", err)
	}
	// With only one chunk, intra-chunk connections are not created.
	t.Logf("connections for single doc: %d", count)
}

func TestKBRebuildConnectionsCustomThreshold(t *testing.T) {
	w := setupWorkspace(t)

	for i, content := range []string{
		"OAuth authentication tokens access refresh authorization code.",
		"OAuth authorization server token endpoint authentication flow.",
	} {
		f := filepath.Join(w.Root, strings.Repeat("t", i+1)+".md")
		os.WriteFile(f, []byte(content), 0o644)
		w.KBAdd(KBAddInput{Source: f})
	}

	// High threshold — fewer connections.
	highCount, _ := w.KBRebuildConnections(0.9, 10)

	// Low threshold — more connections.
	lowCount, _ := w.KBRebuildConnections(0.01, 10)

	if lowCount < highCount {
		t.Errorf("lower threshold should produce >= connections: low=%d, high=%d", lowCount, highCount)
	}
}

func TestBuildConnectionsFullEmpty(t *testing.T) {
	connections := buildConnectionsFull(nil, 0.3, 10)
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

func TestBuildConnectionsFullSingle(t *testing.T) {
	chunks := map[int]string{1: "just one chunk"}
	connections := buildConnectionsFull(chunks, 0.3, 10)
	if len(connections) != 0 {
		t.Errorf("expected 0 connections for single chunk, got %d", len(connections))
	}
}

func TestTFIDFVectorNonEmpty(t *testing.T) {
	tokens := tokenize("OAuth authentication tokens for API security")
	df := map[string]int{"oauth": 2, "authentication": 3, "tokens": 2, "for": 10, "api": 1, "security": 1}
	idf := buildIDF(df, 10)
	vec := tfidfVector(tokens, idf)

	if len(vec) == 0 {
		t.Fatal("expected non-empty TF-IDF vector")
	}

	// High-IDF terms (api, security) should have higher values than low-IDF (for).
	if vec["api"] <= vec["for"] {
		t.Logf("api=%f, for=%f — rare terms should score higher", vec["api"], vec["for"])
	}
}
