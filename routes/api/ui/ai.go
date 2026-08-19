package ui

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	su "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services"
)

// topFilesLimit caps the on-demand file breakdown. File cardinality is unbounded, so
// this is a display limit rather than a data one, and the response says it applied.
const topFilesLimit = 25

type AIHandler struct {
	config        *conf.Config
	aiStatsSrvc   services.IAIStatsService
	summarySrvc   services.ISummaryService
	heartbeatSrvc services.IHeartbeatService
}

func NewAIHandler(aiStatsService services.IAIStatsService, summaryService services.ISummaryService, heartbeatService services.IHeartbeatService) *AIHandler {
	return &AIHandler{
		config:        conf.Get(),
		aiStatsSrvc:   aiStatsService,
		summarySrvc:   summaryService,
		heartbeatSrvc: heartbeatService,
	}
}

func (h *AIHandler) RegisterRoutes(router chi.Router) {
	router.Get("/ai/overview", h.GetOverview)
	router.Get("/ai/files", h.GetFiles)
	router.Get("/admin/ai-prices", h.GetPrices)
	router.Put("/admin/ai-prices", h.UpdatePrices)
}

type aiTokensDTO struct {
	Input       int64 `json:"input"`
	CachedInput int64 `json:"cached_input"`
	Output      int64 `json:"output"`
	Total       int64 `json:"total"`
}

type aiLinesDTO struct {
	AIAdded      int `json:"ai_added"`
	AIRemoved    int `json:"ai_removed"`
	HumanAdded   int `json:"human_added"`
	HumanRemoved int `json:"human_removed"`
}

type aiTotalsDTO struct {
	Lines             aiLinesDTO    `json:"lines"`
	Tokens            aiTokensDTO   `json:"tokens"`
	Cost              models.AICost `json:"cost"`
	PromptCount       int           `json:"prompt_count"`
	PromptLengthTotal int64         `json:"prompt_length_total"`
	SessionCount      int           `json:"session_count"`
	HeartbeatCount    int           `json:"heartbeat_count"`
	// UnpricedTokens are tokens whose model has no entry in the price table, so they
	// contribute nothing to the cost. Some plugins report AI usage without naming a
	// model - the editor lands in that field instead - and without this the total would
	// look complete while quietly excluding them.
	UnpricedTokens int64 `json:"unpriced_tokens"`

	// TokensTracked and CostTracked cover only the days on which cache tokens were
	// actually recorded, and they are the figures worth showing.
	//
	// Before wakapi stored ai_cached_input_tokens, plugins of that era folded cache
	// reads into ai_input_tokens instead. Those days therefore price cache reads at the
	// full input rate - roughly ten times what they cost - so the plain total is an
	// upper bound for them, not a measurement. This split lets the screen headline a
	// number that means something and label the rest for what it is.
	TokensTracked aiTokensDTO   `json:"tokens_tracked"`
	CostTracked   models.AICost `json:"cost_tracked"`
}

type aiModelDTO struct {
	Model   string        `json:"model"`
	Version string        `json:"version"`
	Lines   aiLinesDTO    `json:"lines"`
	Tokens  aiTokensDTO   `json:"tokens"`
	Cost    models.AICost `json:"cost"`
	// Seconds is not available per model - the summaries key on the editor, which for
	// AI heartbeats is the model name but is not split by version.
	PromptCount  int `json:"prompt_count"`
	SessionCount int `json:"session_count"`
}

type aiDayDTO struct {
	Date   time.Time     `json:"date"`
	Lines  aiLinesDTO    `json:"lines"`
	Tokens aiTokensDTO   `json:"tokens"`
	Cost   models.AICost `json:"cost"`
	// CacheTracked is true when this day recorded cache tokens separately. On days
	// where it is false the input figure may already include them, so the cost is an
	// upper bound.
	CacheTracked bool `json:"cache_tracked"`
}

type aiProjectDTO struct {
	Project string        `json:"project"`
	Lines   aiLinesDTO    `json:"lines"`
	Tokens  aiTokensDTO   `json:"tokens"`
	Cost    models.AICost `json:"cost"`
}

type aiOverviewResponse struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`

	Totals    aiTotalsDTO    `json:"totals"`
	ByModel   []aiModelDTO   `json:"by_model"`
	ByDay     []aiDayDTO     `json:"by_day"`
	ByProject []aiProjectDTO `json:"by_project"`

	// Time split comes from the summaries, where "ai coding" is its own category.
	AISeconds    float64 `json:"ai_seconds"`
	HumanSeconds float64 `json:"human_seconds"`

	// CachedTokensSince is the first day cache tokens were recorded at all. Everything
	// before it is missing the bulk of its tokens, so its cost is understated and the
	// screen has to say so. Null means none were ever recorded.
	CachedTokensSince *time.Time `json:"cached_tokens_since"`
	// PricesConfigured is false while the built-in default table is in use.
	PricesConfigured bool `json:"prices_configured"`
}

func (h *AIHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	stats, sessions, err := h.aiStatsSrvc.Retrieve(user, params.From, params.To)
	if err != nil {
		conf.Log().Request(r).Error("failed to load ai stats", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load AI statistics")
		return
	}

	prices := h.aiStatsSrvc.Prices()
	response := &aiOverviewResponse{
		From:             params.From,
		To:               params.To,
		PricesConfigured: h.aiStatsSrvc.HasStoredPrices(),
		ByModel:          []aiModelDTO{},
		ByDay:            []aiDayDTO{},
		ByProject:        []aiProjectDTO{},
	}

	buildAggregates(response, stats, sessions, prices)

	// The AI/human time split is a summary question, not a heartbeat one: the category
	// totals already carry it and are pre-aggregated.
	if summary, err, _ := su.LoadUserSummaryByParams(h.summarySrvc, params); err == nil {
		for _, category := range summary.Categories {
			switch strings.ToLower(category.Key) {
			case models.AICategory:
				response.AISeconds = secondsFromSummaryItem(category.Total)
			case "coding":
				response.HumanSeconds = secondsFromSummaryItem(category.Total)
			}
		}
	} else {
		conf.Log().Request(r).Error("failed to load summary for ai time split", "userID", user.ID, "error", err)
	}

	if since, err := h.aiStatsSrvc.EarliestCachedTokenDay(user); err == nil && !since.IsZero() {
		response.CachedTokensSince = &since
	}

	helpers.RespondJSON(w, r, http.StatusOK, response)
}

// buildAggregates folds the daily rows into the four views the screen shows. Grouping
// happens here rather than in SQL so one read of the range answers all of them.
func buildAggregates(response *aiOverviewResponse, stats []*models.AIStatsDaily, sessions []*models.AISessionDaily, prices models.AIPriceTable) {
	type modelKey struct{ model, version string }

	byModel := map[modelKey]*aiModelDTO{}
	byDay := map[time.Time]*aiDayDTO{}
	byProject := map[string]*aiProjectDTO{}

	addLines := func(target *aiLinesDTO, row *models.AIStatsDaily) {
		target.AIAdded += row.AILinesAdded
		target.AIRemoved += row.AILinesRemoved
		target.HumanAdded += row.HumanLinesAdded
		target.HumanRemoved += row.HumanLinesRemoved
	}
	addTokens := func(target *aiTokensDTO, row *models.AIStatsDaily) {
		target.Input += row.AIInputTokens
		target.CachedInput += row.AICachedInputTokens
		target.Output += row.AIOutputTokens
		target.Total = target.Input + target.CachedInput + target.Output
	}
	addCost := func(target *models.AICost, cost models.AICost) {
		target.Input += cost.Input
		target.CachedInput += cost.CachedInput
		target.Output += cost.Output
		target.Total += cost.Total
		target.Priced = target.Priced || cost.Priced
	}

	for _, row := range stats {
		// Cost is priced per row, because the rate depends on the model and version the
		// row belongs to. Summing first and pricing once would use one model's rate for
		// everything.
		cost := prices.Cost(row.Model, row.ModelVersion, row.AIInputTokens, row.AICachedInputTokens, row.AIOutputTokens)

		addLines(&response.Totals.Lines, row)
		addTokens(&response.Totals.Tokens, row)
		addCost(&response.Totals.Cost, cost)
		response.Totals.PromptCount += row.PromptCount
		response.Totals.PromptLengthTotal += row.PromptLengthTotal
		response.Totals.HeartbeatCount += row.HeartbeatCount

		rowTokens := row.AIInputTokens + row.AICachedInputTokens + row.AIOutputTokens
		if !cost.Priced {
			response.Totals.UnpricedTokens += rowTokens
		}
		if row.AICachedInputTokens > 0 {
			addTokens(&response.Totals.TokensTracked, row)
			addCost(&response.Totals.CostTracked, cost)
		}

		// Rows with no model are the human side of the split and do not belong in a
		// model breakdown.
		if row.Model != "" {
			key := modelKey{row.Model, row.ModelVersion}
			entry, ok := byModel[key]
			if !ok {
				entry = &aiModelDTO{Model: row.Model, Version: row.ModelVersion}
				byModel[key] = entry
			}
			addLines(&entry.Lines, row)
			addTokens(&entry.Tokens, row)
			addCost(&entry.Cost, cost)
			entry.PromptCount += row.PromptCount
		}

		date := row.Date.T()
		day, ok := byDay[date]
		if !ok {
			day = &aiDayDTO{Date: date}
			byDay[date] = day
		}
		addLines(&day.Lines, row)
		addTokens(&day.Tokens, row)
		addCost(&day.Cost, cost)
		day.CacheTracked = day.CacheTracked || row.AICachedInputTokens > 0

		project, ok := byProject[row.Project]
		if !ok {
			project = &aiProjectDTO{Project: row.Project}
			byProject[row.Project] = project
		}
		addLines(&project.Lines, row)
		addTokens(&project.Tokens, row)
		addCost(&project.Cost, cost)
	}

	// Sessions are counted from their own table so one spanning two projects counts once.
	seen := map[string]bool{}
	sessionsByModel := map[string]int{}
	for _, session := range sessions {
		if seen[session.SessionID] {
			continue
		}
		seen[session.SessionID] = true
		response.Totals.SessionCount++
		sessionsByModel[session.Model]++
	}

	for key, entry := range byModel {
		entry.SessionCount = sessionsByModel[key.model]
		response.ByModel = append(response.ByModel, *entry)
	}
	sort.Slice(response.ByModel, func(i, j int) bool {
		return response.ByModel[i].Tokens.Total > response.ByModel[j].Tokens.Total
	})

	for _, day := range byDay {
		response.ByDay = append(response.ByDay, *day)
	}
	sort.Slice(response.ByDay, func(i, j int) bool {
		return response.ByDay[i].Date.Before(response.ByDay[j].Date)
	})

	for _, project := range byProject {
		response.ByProject = append(response.ByProject, *project)
	}
	sort.Slice(response.ByProject, func(i, j int) bool {
		left, right := response.ByProject[i], response.ByProject[j]
		return left.Lines.AIAdded+left.Lines.HumanAdded > right.Lines.AIAdded+right.Lines.HumanAdded
	})
}

type aiFileDTO struct {
	Entity   string `json:"entity"`
	Project  string `json:"project"`
	AILines  int    `json:"ai_lines"`
	Human    int    `json:"human_lines"`
	Prompts  int    `json:"prompts"`
	Modified int    `json:"heartbeats"`
}

type aiFilesResponse struct {
	Items []aiFileDTO `json:"items"`
	Limit int         `json:"limit"`
	// Truncated says the list was cut off, so the screen can say so instead of
	// presenting a top 25 as if it were everything.
	Truncated bool `json:"truncated"`
}

// GetFiles answers the per-file breakdown on demand.
//
// It is not pre-aggregated: file cardinality is far higher than project cardinality, so
// a daily table keyed on entity would dwarf everything else in the database for a view
// that is looked at occasionally.
func (h *AIHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// "human" lists what was edited by hand, "ai" what a model touched.
	wantHuman := r.URL.Query().Get("kind") == "human"

	heartbeats, err := h.heartbeatSrvc.StreamAllWithinRaw(params.From, params.To, user)
	if err != nil {
		conf.Log().Request(r).Error("failed to stream heartbeats for file breakdown", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load files")
		return
	}

	byEntity := map[string]*aiFileDTO{}
	for heartbeat := range heartbeats {
		if heartbeat.Entity == "" {
			continue
		}
		isAI := heartbeat.IsAI()
		if isAI == wantHuman {
			continue
		}

		entry, ok := byEntity[heartbeat.Entity]
		if !ok {
			entry = &aiFileDTO{Entity: heartbeat.Entity, Project: heartbeat.Project}
			byEntity[heartbeat.Entity] = entry
		}
		entry.Modified++
		if isAI {
			entry.AILines += heartbeat.AILineChanges
			if heartbeat.AIPromptLength > 0 {
				entry.Prompts++
			}
		}
		entry.Human += heartbeat.HumanLineChanges
	}

	items := make([]aiFileDTO, 0, len(byEntity))
	for _, entry := range byEntity {
		items = append(items, *entry)
	}
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if wantHuman {
			if left.Human != right.Human {
				return abs(left.Human) > abs(right.Human)
			}
		} else if left.AILines != right.AILines {
			return left.AILines > right.AILines
		}
		return left.Modified > right.Modified
	})

	truncated := len(items) > topFilesLimit
	if truncated {
		items = items[:topFilesLimit]
	}

	helpers.RespondJSON(w, r, http.StatusOK, &aiFilesResponse{
		Items:     items,
		Limit:     topFilesLimit,
		Truncated: truncated,
	})
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

type aiPricesResponse struct {
	Prices models.AIPriceTable `json:"prices"`
	// Defaults lets an admin see what they are diverging from and reset to it.
	Defaults   models.AIPriceTable `json:"defaults"`
	Customized bool                `json:"customized"`
}

func (h *AIHandler) GetPrices(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &aiPricesResponse{
		Prices:     h.aiStatsSrvc.Prices(),
		Defaults:   models.DefaultAIPrices(),
		Customized: h.aiStatsSrvc.HasStoredPrices(),
	})
}

func (h *AIHandler) UpdatePrices(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}

	var table models.AIPriceTable
	if !decodeBody(w, r, &table) {
		return
	}
	if len(table) == 0 {
		respondError(w, r, http.StatusBadRequest, "the price table must not be empty - reset to the defaults instead")
		return
	}

	if err := h.aiStatsSrvc.SetPrices(table); err != nil {
		conf.Log().Request(r).Error("failed to store ai price table", "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not save the price table")
		return
	}

	respondMessage(w, r, http.StatusOK, "Price table saved.")
}
