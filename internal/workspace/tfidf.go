// Package workspace — tfidf.go implements TF-IDF vector computation and
// cosine similarity for building statistical chunk-to-chunk connections
// in the knowledge base. Vectors are sparse (only terms present in the
// chunk are stored), and candidate pruning skips pairs that share no
// high-IDF terms.
package workspace

import (
	"math"
	"strings"
	"unicode"
)

// sparseVec is a sparse TF-IDF vector keyed by term.
type sparseVec map[string]float64

// tokenize splits text into lowercase alphanumeric tokens.
func tokenize(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) >= 2 { // skip single-char tokens
			tokens = append(tokens, w)
		}
	}
	return tokens
}

// termFrequency computes normalized term frequencies for a token list.
func termFrequency(tokens []string) map[string]float64 {
	counts := make(map[string]int, len(tokens))
	for _, t := range tokens {
		counts[t]++
	}
	tf := make(map[string]float64, len(counts))
	total := float64(len(tokens))
	if total == 0 {
		return tf
	}
	for term, count := range counts {
		tf[term] = float64(count) / total
	}
	return tf
}

// idfTable holds inverse document frequency values for terms across a corpus.
type idfTable struct {
	idf    map[string]float64
	docCount int
}

// buildIDF computes IDF from document frequency counts and total document count.
// IDF(t) = log(N / df(t)) where N is total docs and df(t) is docs containing t.
func buildIDF(df map[string]int, totalDocs int) *idfTable {
	idf := make(map[string]float64, len(df))
	n := float64(totalDocs)
	for term, freq := range df {
		idf[term] = math.Log(n / float64(freq))
	}
	return &idfTable{idf: idf, docCount: totalDocs}
}

// tfidfVector computes a sparse TF-IDF vector for a token list given an IDF table.
func tfidfVector(tokens []string, idf *idfTable) sparseVec {
	tf := termFrequency(tokens)
	vec := make(sparseVec, len(tf))
	for term, tfVal := range tf {
		idfVal, ok := idf.idf[term]
		if !ok {
			continue
		}
		vec[term] = tfVal * idfVal
	}
	return vec
}

// cosineSimilarity computes the cosine similarity between two sparse vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b sparseVec) float64 {
	// Dot product — iterate over the smaller vector.
	if len(a) > len(b) {
		a, b = b, a
	}

	var dot float64
	for term, aVal := range a {
		if bVal, ok := b[term]; ok {
			dot += aVal * bVal
		}
	}

	if dot == 0 {
		return 0
	}

	magA := magnitude(a)
	magB := magnitude(b)
	if magA == 0 || magB == 0 {
		return 0
	}

	return dot / (magA * magB)
}

// magnitude computes the L2 norm of a sparse vector.
func magnitude(v sparseVec) float64 {
	var sum float64
	for _, val := range v {
		sum += val * val
	}
	return math.Sqrt(sum)
}

// connectionCandidate holds a chunk ID and its similarity score.
type connectionCandidate struct {
	chunkID  int
	strength float64
}

// buildConnections computes TF-IDF cosine connections for a set of new chunks
// against existing chunks. Returns pairs that exceed the threshold, capped at
// topK per chunk.
//
// Parameters:
//   - newChunks: map of chunk DB ID -> chunk text for newly added chunks
//   - existingChunks: map of chunk DB ID -> chunk text for all existing chunks
//   - threshold: minimum cosine similarity to create a connection (default 0.3)
//   - topK: max connections per chunk (default 10)
func buildConnections(newChunks, existingChunks map[int]string, threshold float64, topK int) []chunkConnection {
	if len(newChunks) == 0 || len(existingChunks) == 0 {
		return nil
	}

	// Tokenize all chunks.
	allTokens := make(map[int][]string)
	df := make(map[string]int)       // document frequency
	termDocs := make(map[string]map[int]bool) // which docs contain each term

	for id, text := range existingChunks {
		tokens := tokenize(text)
		allTokens[id] = tokens
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				seen[t] = true
				df[t]++
				if termDocs[t] == nil {
					termDocs[t] = make(map[int]bool)
				}
				termDocs[t][id] = true
			}
		}
	}

	for id, text := range newChunks {
		if _, exists := allTokens[id]; exists {
			continue // already in existing
		}
		tokens := tokenize(text)
		allTokens[id] = tokens
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				seen[t] = true
				df[t]++
				if termDocs[t] == nil {
					termDocs[t] = make(map[int]bool)
				}
				termDocs[t][id] = true
			}
		}
	}

	totalDocs := len(allTokens)
	idf := buildIDF(df, totalDocs)

	// Compute TF-IDF vectors for new chunks.
	newVecs := make(map[int]sparseVec, len(newChunks))
	for id := range newChunks {
		newVecs[id] = tfidfVector(allTokens[id], idf)
	}

	// Compute TF-IDF vectors for existing chunks (only those that share
	// at least one term with a new chunk — candidate pruning).
	// For small corpora (< 20 docs), skip IDF pruning and compare all.
	candidateIDs := make(map[int]bool)
	if totalDocs < 20 {
		for id := range existingChunks {
			candidateIDs[id] = true
		}
	} else {
		for id := range newChunks {
			for term := range newVecs[id] {
				idfVal := idf.idf[term]
				// Only consider high-IDF terms for pruning.
				if idfVal < 1.0 {
					continue
				}
				for existID := range termDocs[term] {
					if _, isNew := newChunks[existID]; !isNew {
						candidateIDs[existID] = true
					}
				}
			}
		}
	}

	existVecs := make(map[int]sparseVec, len(candidateIDs))
	for id := range candidateIDs {
		existVecs[id] = tfidfVector(allTokens[id], idf)
	}

	// Compute similarities.
	var connections []chunkConnection
	for newID, newVec := range newVecs {
		var candidates []connectionCandidate

		for existID, existVec := range existVecs {
			if existID == newID {
				continue
			}
			sim := cosineSimilarity(newVec, existVec)
			if sim >= threshold {
				candidates = append(candidates, connectionCandidate{
					chunkID:  existID,
					strength: sim,
				})
			}
		}

		// Also check new-to-new connections.
		for otherID, otherVec := range newVecs {
			if otherID <= newID { // avoid duplicates
				continue
			}
			sim := cosineSimilarity(newVec, otherVec)
			if sim >= threshold {
				candidates = append(candidates, connectionCandidate{
					chunkID:  otherID,
					strength: sim,
				})
			}
		}

		// Sort by strength descending, cap at topK.
		sortCandidates(candidates)
		if len(candidates) > topK {
			candidates = candidates[:topK]
		}

		for _, c := range candidates {
			// Bidirectional: A→B and B→A.
			connections = append(connections,
				chunkConnection{fromID: newID, toID: c.chunkID, strength: c.strength},
				chunkConnection{fromID: c.chunkID, toID: newID, strength: c.strength},
			)
		}
	}

	return connections
}

// chunkConnection represents a connection between two chunks.
type chunkConnection struct {
	fromID   int
	toID     int
	strength float64
}

// sortCandidates sorts candidates by strength descending using insertion sort
// (typically small N).
func sortCandidates(candidates []connectionCandidate) {
	for i := 1; i < len(candidates); i++ {
		key := candidates[i]
		j := i - 1
		for j >= 0 && candidates[j].strength < key.strength {
			candidates[j+1] = candidates[j]
			j--
		}
		candidates[j+1] = key
	}
}
