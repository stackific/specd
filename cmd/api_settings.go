// api_settings.go implements UI-state mutation endpoints under
// /api/settings/*. Today the only setting is the configured startpage
// route, persisted in the meta KV table. Add new settings here as
// individual handlers — the table-driven approach the legacy form pages
// used isn't worth the indirection at this scale.
package cmd

import (
	"log/slog"
	"net/http"
)

// apiSetDefaultRouteRequest is the body of POST /api/settings/default-route.
type apiSetDefaultRouteRequest struct {
	DefaultRoute string `json:"default_route"`
}

// apiSetDefaultRouteHandler implements POST /api/settings/default-route. The
// posted route is validated against StartpageChoices via lookupStartpageRoute
// so user input never reaches the meta table raw.
func apiSetDefaultRouteHandler(w http.ResponseWriter, r *http.Request) {
	body, err := decodeJSON[apiSetDefaultRouteRequest](w, r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	canonical, ok := lookupStartpageRoute(body.DefaultRoute)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid default_route")
		return
	}

	db, _, err := OpenProjectDB()
	if err != nil {
		slog.Error("api settings: db", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	defer func() { _ = db.Close() }()

	if err := WriteMeta(db, MetaDefaultRoute, canonical); err != nil {
		slog.Error("api settings: write", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save")
		return
	}

	slog.Info("api.settings.default_route", "route", canonical)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"default_route": canonical,
	})
}
