package workspace

import "fmt"

// CandidateSpec is a spec returned as a link candidate.
type CandidateSpec struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Score float64 `json:"score"`
}

// CandidateTask is a task returned as a link candidate.
type CandidateTask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Score float64 `json:"score"`
}

// CandidatesResult holds candidate specs and tasks for linking.
type CandidatesResult struct {
	Specs []CandidateSpec `json:"specs"`
	Tasks []CandidateTask `json:"tasks"`
}

// Candidates finds specs or tasks related to the given ID by searching
// title, summary, and body text. Excludes the item itself and any
// already-linked items. Uses simple word overlap scoring until FTS5
// hybrid search is wired in Phase 5.
func (w *Workspace) Candidates(id string, limit int) (*CandidatesResult, error) {
	if limit <= 0 {
		limit = 20
	}

	if isSpec(id) {
		return w.specCandidates(id, limit)
	}
	if isTask(id) {
		return w.taskCandidates(id, limit)
	}
	return nil, fmt.Errorf("invalid ID format: %s", id)
}

func (w *Workspace) specCandidates(specID string, limit int) (*CandidatesResult, error) {
	spec, err := w.ReadSpec(specID)
	if err != nil {
		return nil, err
	}

	// Build search terms from the spec's text.
	terms := extractTerms(spec.Title + " " + spec.Summary + " " + spec.Body)
	if len(terms) == 0 {
		return &CandidatesResult{}, nil
	}

	// Get already-linked spec IDs to exclude.
	linked, _ := w.getSpecLinks(specID)
	exclude := map[string]bool{specID: true}
	for _, l := range linked {
		exclude[l] = true
	}

	// Score all other specs by word overlap.
	allSpecs, err := w.ListSpecs(ListSpecsFilter{})
	if err != nil {
		return nil, err
	}

	var candidates []CandidateSpec
	for _, s := range allSpecs {
		if exclude[s.ID] {
			continue
		}
		score := wordOverlap(terms, s.Title+" "+s.Summary+" "+s.Body)
		if score > 0 {
			candidates = append(candidates, CandidateSpec{ID: s.ID, Title: s.Title, Score: score})
		}
	}

	sortCandidateSpecs(candidates)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return &CandidatesResult{Specs: candidates}, nil
}

func (w *Workspace) taskCandidates(taskID string, limit int) (*CandidatesResult, error) {
	task, err := w.ReadTask(taskID)
	if err != nil {
		return nil, err
	}

	terms := extractTerms(task.Title + " " + task.Summary + " " + task.Body)
	if len(terms) == 0 {
		return &CandidatesResult{}, nil
	}

	linked, _ := w.getTaskLinks(taskID)
	exclude := map[string]bool{taskID: true}
	for _, l := range linked {
		exclude[l] = true
	}

	allTasks, err := w.ListTasks(ListTasksFilter{})
	if err != nil {
		return nil, err
	}

	var candidates []CandidateTask
	for _, t := range allTasks {
		if exclude[t.ID] {
			continue
		}
		score := wordOverlap(terms, t.Title+" "+t.Summary+" "+t.Body)
		if score > 0 {
			candidates = append(candidates, CandidateTask{ID: t.ID, Title: t.Title, Score: score})
		}
	}

	sortCandidateTasks(candidates)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return &CandidatesResult{Tasks: candidates}, nil
}

// extractTerms splits text into lowercase unique words, filtering short ones.
func extractTerms(text string) map[string]bool {
	terms := map[string]bool{}
	for _, word := range splitWords(text) {
		if len(word) >= 3 {
			terms[word] = true
		}
	}
	return terms
}

// wordOverlap returns the fraction of source terms found in the target text.
func wordOverlap(sourceTerms map[string]bool, target string) float64 {
	if len(sourceTerms) == 0 {
		return 0
	}
	targetWords := map[string]bool{}
	for _, w := range splitWords(target) {
		targetWords[w] = true
	}
	matches := 0
	for term := range sourceTerms {
		if targetWords[term] {
			matches++
		}
	}
	return float64(matches) / float64(len(sourceTerms))
}

// splitWords lowercases and splits on non-alphanumeric boundaries.
func splitWords(s string) []string {
	var words []string
	var current []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			current = append(current, c)
		} else if c >= 'A' && c <= 'Z' {
			current = append(current, c+32) // toLower
		} else {
			if len(current) > 0 {
				words = append(words, string(current))
				current = current[:0]
			}
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// sortCandidateSpecs sorts by score descending.
func sortCandidateSpecs(c []CandidateSpec) {
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && c[j].Score > c[j-1].Score; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}

// sortCandidateTasks sorts by score descending.
func sortCandidateTasks(c []CandidateTask) {
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && c[j].Score > c[j-1].Score; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}
