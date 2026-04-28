// loaders_specs.go contains the spec list/detail data loaders that back the
// JSON API in cmd/api.go. The HTML rendering wrappers were removed in the
// SPA migration; only the shared business logic remains here.
package cmd

import (
	"database/sql"
	"fmt"
)

// Spec page view mode identifiers used in the ?view= query string.
const (
	SpecsViewGrouped = "grouped" // default: per-type bordered tables
	SpecsViewCards   = "cards"   // per-type card grid
	SpecsViewFlat    = "flat"    // single table with a Type column
)

// SpecsTypeAll is the sentinel value for "no type filter" in the ?type= query
// string. Anything else must match a configured spec type slug.
const SpecsTypeAll = "all"

// specsPageDefaultSize is the default page size for the specs page.
const specsPageDefaultSize = 20

// specsPageMaxSize caps user-supplied page_size.
const specsPageMaxSize = 100

// validSpecsViews enumerates the accepted view modes.
var validSpecsViews = map[string]bool{
	SpecsViewGrouped: true,
	SpecsViewCards:   true,
	SpecsViewFlat:    true,
}

// SpecsPageData is the view model returned by loadSpecsPage and serialized
// by /api/specs.
type SpecsPageData struct {
	View       string
	Type       string         // active type filter slug, or SpecsTypeAll
	Types      []string       // configured spec type slugs offered as filter options
	Items      []ListSpecItem // current page, ordered for the active view
	Groups     []SpecsGroup   // populated for grouped views; nil for flat
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
}

// SpecsGroup is one type-grouped slice of specs in the current page.
// Total holds the per-type count across all pages so the count badge stays
// stable when only a subset of items appears on the current page.
type SpecsGroup struct {
	Type  string
	Items []ListSpecItem
	Total int
}

// configuredSpecTypes returns the project's configured spec type slugs, or an
// empty slice if the project config can't be read.
func configuredSpecTypes() []string {
	proj, err := LoadProjectConfig(".")
	if err != nil || proj == nil {
		return nil
	}
	return proj.SpecTypes
}

// isAllowedSpecType reports whether t is the "all" sentinel or one of the
// configured type slugs.
func isAllowedSpecType(t string, types []string) bool {
	if t == SpecsTypeAll {
		return true
	}
	for _, s := range types {
		if s == t {
			return true
		}
	}
	return false
}

// loadSpecsPage queries one page of specs, ordered so that grouped views fall
// out naturally (type, then position, then id), populates pagination metadata,
// and groups items by type when needed. data.Type filters the result set; the
// SpecsTypeAll sentinel disables the filter.
func loadSpecsPage(data *SpecsPageData) error {
	db, _, err := OpenProjectDB()
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM specs WHERE ?1 = ?2 OR type = ?1`,
		data.Type, SpecsTypeAll,
	).Scan(&total); err != nil {
		return fmt.Errorf("counting specs: %w", err)
	}
	data.TotalCount = total
	if total == 0 {
		data.TotalPages = 0
		return nil
	}

	data.TotalPages = (total + data.PageSize - 1) / data.PageSize
	if data.Page > data.TotalPages {
		data.Page = data.TotalPages
	}
	offset := (data.Page - 1) * data.PageSize

	rows, err := db.Query(`
		SELECT id, title, type, summary, position, created_at, updated_at
		FROM specs
		WHERE ?1 = ?2 OR type = ?1
		ORDER BY type, position, id
		LIMIT ?3 OFFSET ?4
	`, data.Type, SpecsTypeAll, data.PageSize, offset)
	if err != nil {
		return fmt.Errorf("listing specs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var s ListSpecItem
		if err := rows.Scan(&s.ID, &s.Title, &s.Type, &s.Summary, &s.Position, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return fmt.Errorf("scanning spec: %w", err)
		}
		data.Items = append(data.Items, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating specs: %w", err)
	}

	if data.View != SpecsViewFlat {
		typeTotals, err := loadSpecTypeTotals(db, data.Type)
		if err != nil {
			return err
		}
		data.Groups = groupSpecsByType(data.Items, typeTotals)
	}

	return nil
}

// groupSpecsByType buckets a contiguous slice (already sorted by type) into
// per-type groups while preserving order. typeTotals supplies the project-wide
// count per type so each group's badge reflects the true total, not just the
// items rendered on the current page.
func groupSpecsByType(items []ListSpecItem, typeTotals map[string]int) []SpecsGroup {
	if len(items) == 0 {
		return nil
	}
	var groups []SpecsGroup
	cur := SpecsGroup{Type: items[0].Type, Items: []ListSpecItem{items[0]}, Total: typeTotals[items[0].Type]}
	for _, it := range items[1:] {
		if it.Type == cur.Type {
			cur.Items = append(cur.Items, it)
			continue
		}
		groups = append(groups, cur)
		cur = SpecsGroup{Type: it.Type, Items: []ListSpecItem{it}, Total: typeTotals[it.Type]}
	}
	groups = append(groups, cur)
	return groups
}

// loadSpecTypeTotals returns a map of spec type → total count, optionally
// constrained by the active type filter. Pass SpecsTypeAll to disable the
// filter.
func loadSpecTypeTotals(db *sql.DB, typeFilter string) (map[string]int, error) {
	rows, err := db.Query(
		`SELECT type, COUNT(*) FROM specs WHERE ?1 = ?2 OR type = ?1 GROUP BY type`,
		typeFilter, SpecsTypeAll,
	)
	if err != nil {
		return nil, fmt.Errorf("counting specs by type: %w", err)
	}
	defer func() { _ = rows.Close() }()

	totals := map[string]int{}
	for rows.Next() {
		var t string
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			return nil, fmt.Errorf("scanning type total: %w", err)
		}
		totals[t] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating type totals: %w", err)
	}
	return totals, nil
}
