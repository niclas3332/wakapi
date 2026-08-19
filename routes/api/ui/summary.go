package ui

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/view"
	su "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
)

type SummaryHandler struct {
	config        *conf.Config
	summarySrvc   services.ISummaryService
	durationSrvc  services.IDurationService
	aliasSrvc     services.IAliasService
	heartbeatSrvc services.IHeartbeatService
}

func NewSummaryHandler(
	summaryService services.ISummaryService,
	durationService services.IDurationService,
	aliasService services.IAliasService,
	heartbeatService services.IHeartbeatService,
) *SummaryHandler {
	return &SummaryHandler{
		config:        conf.Get(),
		summarySrvc:   summaryService,
		durationSrvc:  durationService,
		aliasSrvc:     aliasService,
		heartbeatSrvc: heartbeatService,
	}
}

func (h *SummaryHandler) RegisterRoutes(router chi.Router) {
	router.Get("/summary", h.Get)
	router.Get("/summary/timeline", h.GetTimeline)
	router.Get("/summary/hourly", h.GetHourly)
}

type availableFilters struct {
	Projects   []string `json:"projects"`
	Languages  []string `json:"languages"`
	Machines   []string `json:"machines"`
	Labels     []string `json:"labels"`
	Categories []string `json:"categories"`
}

type summaryItemDTO struct {
	Key     string  `json:"key"`
	Seconds float64 `json:"seconds"`
}

// summaryDTO mirrors models.Summary but emits seconds and never a null collection,
// so the client needs neither unit conversion nor null checks.
type summaryDTO struct {
	From             time.Time        `json:"from"`
	To               time.Time        `json:"to"`
	Projects         []summaryItemDTO `json:"projects"`
	Languages        []summaryItemDTO `json:"languages"`
	Editors          []summaryItemDTO `json:"editors"`
	OperatingSystems []summaryItemDTO `json:"operating_systems"`
	Machines         []summaryItemDTO `json:"machines"`
	Labels           []summaryItemDTO `json:"labels"`
	Branches         []summaryItemDTO `json:"branches"`
	Entities         []summaryItemDTO `json:"entities"`
	Categories       []summaryItemDTO `json:"categories"`
}

func newSummaryItems(items models.SummaryItems) []summaryItemDTO {
	out := make([]summaryItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, summaryItemDTO{Key: item.Key, Seconds: secondsFromSummaryItem(item.Total)})
	}
	return out
}

func newSummaryDTO(summary *models.Summary) *summaryDTO {
	return &summaryDTO{
		From:             summary.FromTime.T(),
		To:               summary.ToTime.T(),
		Projects:         newSummaryItems(summary.Projects),
		Languages:        newSummaryItems(summary.Languages),
		Editors:          newSummaryItems(summary.Editors),
		OperatingSystems: newSummaryItems(summary.OperatingSystems),
		Machines:         newSummaryItems(summary.Machines),
		Labels:           newSummaryItems(summary.Labels),
		Branches:         newSummaryItems(summary.Branches),
		Entities:         newSummaryItems(summary.Entities),
		Categories:       newSummaryItems(summary.Categories),
	}
}

type summaryResponse struct {
	Summary *summaryDTO `json:"summary"`
	// AvailableFilters comes from an unfiltered summary of the same range, so the
	// dropdowns keep offering every option even while a filter is active.
	AvailableFilters    availableFilters  `json:"available_filters"`
	EditorColors        map[string]string `json:"editor_colors"`
	LanguageColors      map[string]string `json:"language_colors"`
	OSColors            map[string]string `json:"os_colors"`
	UserFirstData       time.Time         `json:"user_first_data"`
	DataRetentionMonths int               `json:"data_retention_months"`
}

// Get returns the summary for the requested range plus everything the dashboard
// needs to render its filters and colour its charts.
func (h *SummaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	summary, err, status := su.LoadUserSummaryByParams(h.summarySrvc, params)
	if err != nil {
		conf.Log().Request(r).Error("failed to load summary", "error", err)
		respondError(w, r, status, err.Error())
		return
	}

	unfiltered, err, status := su.LoadUserSummaryWithoutFilter(h.summarySrvc, params)
	if err != nil {
		conf.Log().Request(r).Error("failed to load unfiltered summary", "error", err)
		respondError(w, r, status, err.Error())
		return
	}
	filters := su.ExtractAvailableFilters(unfiltered)

	firstData, err := h.heartbeatSrvc.GetFirstByUser(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load user's first heartbeat", "user", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &summaryResponse{
		Summary: newSummaryDTO(summary),
		AvailableFilters: availableFilters{
			Projects:   filters.ProjectNames,
			Languages:  filters.LanguageNames,
			Machines:   filters.MachineNames,
			Labels:     filters.LabelNames,
			Categories: filters.CategoryNames,
		},
		EditorColors:        su.FilterColors(h.config.App.GetEditorColors(), summary.Editors),
		LanguageColors:      su.FilterColors(h.config.App.GetLanguageColors(), summary.Languages),
		OSColors:            su.FilterColors(h.config.App.GetOSColors(), summary.OperatingSystems),
		UserFirstData:       firstData,
		DataRetentionMonths: h.config.App.DataRetentionMonths,
	})
}

type timelineProjectDTO struct {
	Name    string  `json:"name"`
	Seconds float64 `json:"seconds"`
}

type timelineDayDTO struct {
	Date     time.Time            `json:"date"`
	Projects []timelineProjectDTO `json:"projects"`
}

type timelineResponse struct {
	Items []timelineDayDTO `json:"items"`
	// Available reports whether the range qualifies for a per-day timeline at all.
	// False is a normal answer, not an error.
	Available    bool `json:"available"`
	MinRangeDays int  `json:"min_range_days"`
	MaxRangeDays int  `json:"max_range_days"`
}

func newTimelineDays(summaries []*models.Summary) []timelineDayDTO {
	days := make([]timelineDayDTO, 0, len(summaries))
	for _, summary := range summaries {
		projects := make([]timelineProjectDTO, 0, len(summary.Projects))
		for _, project := range summary.Projects {
			projects = append(projects, timelineProjectDTO{
				Name:    project.Key,
				Seconds: secondsFromSummaryItem(project.Total),
			})
		}
		days = append(days, timelineDayDTO{Date: summary.FromTime.T(), Projects: projects})
	}
	return days
}

// GetTimeline returns one entry per day for the requested range, which only makes
// sense for ranges of a few days up to a month.
func (h *SummaryHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePrincipal(w, r); !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response := &timelineResponse{
		Items:        []timelineDayDTO{},
		MinRangeDays: su.DailyStatsMinRangeDays,
		MaxRangeDays: su.DailyStatsMaxRangeDays,
	}

	rangeDays := params.RangeDays()
	if rangeDays < su.DailyStatsMinRangeDays || rangeDays > su.DailyStatsMaxRangeDays {
		helpers.RespondJSON(w, r, http.StatusOK, response)
		return
	}

	summaries, err := su.FetchSplitSummaries(h.summarySrvc, params)
	if err != nil {
		conf.Log().Request(r).Error("failed to load timeline stats", "error", err)
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.Available = true
	response.Items = newTimelineDays(summaries)
	helpers.RespondJSON(w, r, http.StatusOK, response)
}

type hourlySegmentDTO struct {
	FromTime time.Time `json:"from_time"`
	Seconds  float64   `json:"seconds"`
	Entity   string    `json:"entity"`
}

type hourlyProjectDTO struct {
	Project string             `json:"project"`
	Items   []hourlySegmentDTO `json:"items"`
}

type hourlyResponse struct {
	From  time.Time          `json:"from"`
	Items []hourlyProjectDTO `json:"items"`
	// Truncated means the range held too many segments to plot. The chart is hidden
	// in that case; it is a display limit, not a failure.
	Truncated    bool `json:"truncated"`
	MaxDurations int  `json:"max_durations"`
}

func newHourlyProjects(breakdown view.HourlyBreakdownsViewModel) []hourlyProjectDTO {
	projects := make([]hourlyProjectDTO, 0, len(breakdown))
	for _, project := range breakdown {
		segments := make([]hourlySegmentDTO, 0, len(project.Items))
		for _, item := range project.Items {
			segments = append(segments, hourlySegmentDTO{
				FromTime: item.FromTime,
				Seconds:  secondsFromDuration(item.Duration),
				Entity:   item.Entity,
			})
		}
		projects = append(projects, hourlyProjectDTO{Project: project.Project, Items: segments})
	}
	return projects
}

// GetHourly returns the per-project hourly breakdown, covering at most the last 24
// hours of the requested range.
func (h *SummaryHandler) GetHourly(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePrincipal(w, r); !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	breakdown, err := su.BuildHourlyBreakdown(h.durationSrvc, h.aliasSrvc, params)
	if err != nil {
		conf.Log().Request(r).Error("failed to load hourly breakdown stats", "error", err)
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &hourlyResponse{
		From:         breakdown.From,
		Items:        newHourlyProjects(breakdown.Items),
		Truncated:    breakdown.Truncated,
		MaxDurations: su.HourlyBreakdownMaxDurations,
	})
}
