// api_meta.go implements GET /api/meta. The endpoint returns the bits of
// project config the SPA needs at boot — the configured spec types and task
// stages, the configured startpage routes, the user's saved default route —
// in one round-trip so the shell doesn't have to fan out before rendering.
package cmd

import (
	"net/http"
)

// apiMetaResponse is the payload returned by GET /api/meta.
type apiMetaResponse struct {
	DefaultRoute     string            `json:"default_route"`
	Project          apiProjectMeta    `json:"project"`
	StartpageChoices []apiStartpageOpt `json:"startpage_choices"`
}

// apiProjectMeta is the project-config slice exposed to the SPA.
type apiProjectMeta struct {
	Name            string   `json:"name"`
	SpecTypes       []string `json:"spec_types"`
	TaskStages      []string `json:"task_stages"`
	DefaultPageSize int      `json:"default_page_size"`
}

// apiStartpageOpt mirrors StartpageChoice with JSON tags for the SPA. The
// SPA expects "title" rather than "label" so it can render a typed select.
type apiStartpageOpt struct {
	Title string `json:"title"`
	Route string `json:"route"`
}

// apiMetaHandler implements GET /api/meta. When the project is not yet
// initialized, the project sub-object is returned with empty arrays so the
// SPA shell can still render.
func apiMetaHandler(w http.ResponseWriter, _ *http.Request) {
	choices := make([]apiStartpageOpt, 0, len(StartpageChoices))
	for _, c := range StartpageChoices {
		choices = append(choices, apiStartpageOpt{Title: c.Label, Route: c.Route})
	}

	resp := apiMetaResponse{
		DefaultRoute:     readDefaultRoute(),
		StartpageChoices: choices,
		Project: apiProjectMeta{
			SpecTypes:       []string{},
			TaskStages:      []string{},
			DefaultPageSize: DefaultPageSize,
		},
	}

	if proj, err := LoadProjectConfig("."); err == nil && proj != nil {
		resp.Project.Name = proj.Dir
		if proj.SpecTypes != nil {
			resp.Project.SpecTypes = proj.SpecTypes
		}
		if proj.TaskStages != nil {
			resp.Project.TaskStages = proj.TaskStages
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
