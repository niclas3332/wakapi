package ui

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
)

const (
	projectsDefaultPageSize = 24
)

type ProjectsHandler struct {
	config        *conf.Config
	projectSrvc   services.IProjectService
	heartbeatSrvc services.IHeartbeatService
	aliasSrvc     services.IAliasService
}

func NewProjectsHandler(projectService services.IProjectService, heartbeatService services.IHeartbeatService, aliasService services.IAliasService) *ProjectsHandler {
	return &ProjectsHandler{
		config:        conf.Get(),
		projectSrvc:   projectService,
		heartbeatSrvc: heartbeatService,
		aliasSrvc:     aliasService,
	}
}

func (h *ProjectsHandler) RegisterRoutes(router chi.Router) {
	router.Get("/projects", h.Get)
	router.Get("/projects/names", h.GetNames)
}

type projectItem struct {
	Project     string    `json:"project"`
	TopLanguage string    `json:"top_language"`
	Count       int64     `json:"count"`
	First       time.Time `json:"first"`
	Last        time.Time `json:"last"`
}

// Get lists the user's projects with their heartbeat counts and activity span.
func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	pageParams := utils.ParsePageParamsWithDefault(r, 1, projectsDefaultPageSize)
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// The upper bound is today's start, matching the legacy page: project stats are
	// served from a cache that the housekeeping job warms on day boundaries.
	stats, err := h.projectSrvc.GetUserProjectStats(user, time.Time{}, utils.BeginOfToday(time.Local), query, pageParams, false)
	if err != nil {
		conf.Log().Request(r).Error("error while fetching project stats", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]*projectItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, &projectItem{
			Project:     s.Project,
			TopLanguage: s.TopLanguage,
			Count:       s.Count,
			First:       s.First.T(),
			Last:        s.Last.T(),
		})
	}

	helpers.RespondJSON(w, r, http.StatusOK, newPagedResponse(items, pageParams))
}

type projectNamesResponse struct {
	Projects []string `json:"projects"`
}

// GetNames returns the plain project name list used to populate the alias and label
// pickers in the settings screen. Unlike Get it is alias-resolved and unpaginated.
func (h *ProjectsHandler) GetNames(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	projects, err := routeutils.GetEffectiveProjectsList(user, h.heartbeatSrvc, h.aliasSrvc)
	if err != nil {
		conf.Log().Request(r).Error("error while fetching project names", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []string{}
	}

	helpers.RespondJSON(w, r, http.StatusOK, &projectNamesResponse{Projects: projects})
}
