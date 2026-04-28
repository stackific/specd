// api_kb.go implements GET /api/kb (list) and GET /api/kb/{id} (detail with
// chunks). KB content is read-only via the API today — ingestion still
// happens out-of-band via direct DB inserts in the QA seed script.
package cmd

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// apiKBListItem is a single KB doc in the list response.
type apiKBListItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	SourceType string `json:"source_type"`
	Path       string `json:"path"`
	AddedAt    string `json:"added_at"`
	AddedBy    string `json:"added_by"`
}

// apiKBListResponse is the payload returned by GET /api/kb.
type apiKBListResponse struct {
	Items []apiKBListItem `json:"items"`
}

// apiKBListHandler implements GET /api/kb. Reads kb_docs ordered by id.
func apiKBListHandler(w http.ResponseWriter, _ *http.Request) {
	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api kb list: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	items, err := loadKBList(db)
	if err != nil {
		slog.Error("api kb list: load", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to list kb docs")
		return
	}
	writeJSON(w, http.StatusOK, apiKBListResponse{Items: items})
}

// loadKBList returns every KB doc, ordered by id, for the JSON list endpoint.
// Lives here (next to its only caller) rather than in the legacy detail page
// so the file stays focused on the per-doc detail handler.
func loadKBList(db *sql.DB) ([]apiKBListItem, error) {
	rows, err := db.Query(`
		SELECT id, title, summary, source_type, path, added_at, added_by
		FROM kb_docs
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := []apiKBListItem{}
	for rows.Next() {
		var it apiKBListItem
		var addedBy *string
		if err := rows.Scan(&it.ID, &it.Title, &it.Summary, &it.SourceType, &it.Path, &it.AddedAt, &addedBy); err != nil {
			return nil, err
		}
		if addedBy != nil {
			it.AddedBy = *addedBy
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// apiKBDetailHandler implements GET /api/kb/{id}.
func apiKBDetailHandler(w http.ResponseWriter, r *http.Request) {
	kbID := strings.ToUpper(r.PathValue("id"))
	if !strings.HasPrefix(kbID, IDPrefixKB) {
		writeJSONError(w, http.StatusBadRequest, "invalid kb id")
		return
	}

	data, err := loadKBDetailPage(kbID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			writeJSONError(w, http.StatusNotFound, "kb doc not found")
			return
		}
		slog.Error("api kb detail: load", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load kb doc")
		return
	}
	writeJSON(w, http.StatusOK, data)
}
