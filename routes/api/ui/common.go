// Package ui serves the JSON API consumed by the React frontend.
//
// It is deliberately separate from /api/v1 and /api/compat/wakatime/v1: those are
// public contracts that editor plugins depend on, while everything here is internal
// and shaped for one specific UI. It may change shape whenever the frontend does.
package ui

import (
	"net/http"

	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
)

// pagedResponse is the shape of every list endpoint.
//
// There is no total count: neither the project nor the leaderboard service can
// produce one today (see the pagination TODOs in routes/projects.go and
// routes/leaderboard.go), so the UI does "load more" rather than numbered pages.
// HasMore is a heuristic - a full page might still be the last one, in which case
// the next request simply comes back empty.
type pagedResponse[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	HasMore  bool `json:"has_more"`
}

func newPagedResponse[T any](items []T, params *utils.PageParams) *pagedResponse[T] {
	if items == nil {
		items = []T{}
	}
	return &pagedResponse[T]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		HasMore:  params.PageSize > 0 && len(items) >= params.PageSize,
	}
}

// requirePrincipal resolves the authenticated user. The auth middleware normally
// guarantees one, so a miss here means a wiring mistake rather than a client error.
func requirePrincipal(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user := middlewares.GetPrincipal(r)
	if user == nil {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	return user, true
}

type errorResponse struct {
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, r *http.Request, status int, message string) {
	helpers.RespondJSON(w, r, status, &errorResponse{Error: message})
}
