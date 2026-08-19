package ui

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
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
	summarySrvc   services.ISummaryService
}

func NewProjectsHandler(projectService services.IProjectService, heartbeatService services.IHeartbeatService, aliasService services.IAliasService, summaryService services.ISummaryService) *ProjectsHandler {
	return &ProjectsHandler{
		config:        conf.Get(),
		projectSrvc:   projectService,
		heartbeatSrvc: heartbeatService,
		aliasSrvc:     aliasService,
		summarySrvc:   summaryService,
	}
}

func (h *ProjectsHandler) RegisterRoutes(router chi.Router) {
	router.Get("/projects", h.Get)
	router.Get("/projects/names", h.GetNames)
}

type projectItem struct {
	Project     string `json:"project"`
	TopLanguage string `json:"top_language"`
	Count       int64  `json:"count"`
	// Seconds is the all-time total coded on this project. A heartbeat count means
	// nothing to a reader, so this is what the list actually shows.
	Seconds float64   `json:"seconds"`
	First   time.Time `json:"first"`
	Last    time.Time `json:"last"`
}

// Get lists the user's projects with their total time and activity span.
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

	// One unfiltered all-time summary covers every project on the page. It reads the
	// pre-generated daily summaries, so this is a single cheap lookup rather than a
	// per-project query.
	totals := h.allTimeSecondsByProject(r, user)

	items := make([]*projectItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, &projectItem{
			Project:     s.Project,
			TopLanguage: s.TopLanguage,
			Count:       s.Count,
			Seconds:     totals[s.Project],
			First:       s.First.T(),
			Last:        s.Last.T(),
		})
	}

	helpers.RespondJSON(w, r, http.StatusOK, newPagedResponse(items, pageParams))
}

// allTimeSecondsByProject maps every project to its total tracked seconds. A failure
// here costs the durations but not the list, so it degrades to an empty map.
func (h *ProjectsHandler) allTimeSecondsByProject(r *http.Request, user *models.User) map[string]float64 {
	err, from, to := helpers.ResolveIntervalTZ(models.IntervalAny, user.TZ(), user.StartOfWeekDay())
	if err != nil {
		conf.Log().Request(r).Error("failed to resolve all-time interval", "userID", user.ID, "error", err)
		return map[string]float64{}
	}

	summary, err := h.summarySrvc.Aliased(from, to, user, h.summarySrvc.Retrieve, nil, nil, false)
	if err != nil {
		conf.Log().Request(r).Error("failed to load all-time summary for project durations", "userID", user.ID, "error", err)
		return map[string]float64{}
	}

	totals := make(map[string]float64, len(summary.Projects))
	for _, item := range summary.Projects {
		totals[item.Key] = secondsFromSummaryItem(item.Total)
	}
	return totals
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
