package ui

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
)

// insightsTopN is how many entries each ranking carries. Enough to see the shape of a
// year without turning the response into a data dump.
const insightsTopN = 10

type InsightsHandler struct {
	config        *conf.Config
	summarySrvc   services.ISummaryService
	heartbeatSrvc services.IHeartbeatService
	aiStatsSrvc   services.IAIStatsService
}

func NewInsightsHandler(summaryService services.ISummaryService, heartbeatService services.IHeartbeatService, aiStatsService services.IAIStatsService) *InsightsHandler {
	return &InsightsHandler{
		config:        conf.Get(),
		summarySrvc:   summaryService,
		heartbeatSrvc: heartbeatService,
		aiStatsSrvc:   aiStatsService,
	}
}

func (h *InsightsHandler) RegisterRoutes(router chi.Router) {
	router.Get("/insights", h.GetInsights)
	router.Get("/wrapped/years", h.GetWrappedYears)
	router.Get("/wrapped/{year}", h.GetWrapped)
}

type dayTotalDTO struct {
	Date    time.Time `json:"date"`
	Seconds float64   `json:"seconds"`
}

type weekdayDTO struct {
	// Weekday is 0 for Sunday, matching time.Weekday and JavaScript's getDay().
	Weekday      int     `json:"weekday"`
	TotalSeconds float64 `json:"total_seconds"`
	// AverageSeconds divides by the number of that weekday in range, including the ones
	// with no activity - otherwise "average Sunday" would only average the Sundays that
	// were worked, which is a different and much larger number.
	AverageSeconds float64 `json:"average_seconds"`
	Days           int     `json:"days"`
}

type streakDTO struct {
	Current int        `json:"current"`
	Longest int        `json:"longest"`
	Start   *time.Time `json:"longest_start"`
	End     *time.Time `json:"longest_end"`
}

type insightsResponse struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`

	TotalSeconds  float64      `json:"total_seconds"`
	DailyAverage  float64      `json:"daily_average"`
	ActiveDays    int          `json:"active_days"`
	DayCount      int          `json:"day_count"`
	AISeconds     float64      `json:"ai_seconds"`
	HumanSeconds  float64      `json:"human_seconds"`
	MostActiveDay *dayTotalDTO `json:"most_active_day"`

	Daily          []dayTotalDTO     `json:"daily"`
	Weekdays       []weekdayDTO      `json:"weekdays"`
	Streak         streakDTO         `json:"streak"`
	Projects       []summaryItemDTO  `json:"projects"`
	Languages      []summaryItemDTO  `json:"languages"`
	Editors        []summaryItemDTO  `json:"editors"`
	OperatingSys   []summaryItemDTO  `json:"operating_systems"`
	Machines       []summaryItemDTO  `json:"machines"`
	Categories     []summaryItemDTO  `json:"categories"`
	LanguageColors map[string]string `json:"language_colors"`
	EditorColors   map[string]string `json:"editor_colors"`
	OSColors       map[string]string `json:"os_colors"`
}

func (h *InsightsHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	params, err := helpers.ParseSummaryParams(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.buildInsights(user, params.From, params.To)
	if err != nil {
		conf.Log().Request(r).Error("failed to build insights", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load insights")
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, response)
}

// buildInsights reads the range twice, on purpose.
//
// The headline figures and the rankings come from one merged summary - the same call
// the dashboard makes - so the two screens can never disagree about how much time a
// range holds. The per-day summaries supply the shape: which days were worked, how long
// each was, where the streaks are.
//
// The two do not add up to the same number, and that is upstream behaviour rather than
// something introduced here: a merged retrieval skips intra-day gaps for speed (the
// `precise` flag in SummaryService.getMissingIntervals is always false), so it lands a
// few percent below the sum of the days. Each individual day matches wakapi's own
// per-day api exactly. Taking the total from the merged call keeps this screen
// consistent with every other one rather than quietly reporting a different number.
func (h *InsightsHandler) buildInsights(user *models.User, from, to time.Time) (*insightsResponse, error) {
	// "All time" resolves to the unix epoch, which would mean twenty thousand empty
	// days before the account existed - and twenty thousand summary lookups to prove
	// they are empty. A user's all time starts at their first heartbeat.
	if first, err := h.heartbeatSrvc.GetFirstByUser(user); err == nil && !first.IsZero() && first.After(from) {
		from = first
	}

	intervals := utils.SplitRangeByDays(from, to)
	summaries := make([]*models.Summary, len(intervals))

	pool := pond.NewPool(utils.HalfCPUs())
	var mutex sync.Mutex
	var firstErr error

	for i, interval := range intervals {
		i, interval := i, interval
		pool.Submit(func() {
			summary, err := h.summarySrvc.Aliased(interval[0], interval[1], user, h.summarySrvc.Retrieve, nil, nil, false)
			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			summaries[i] = summary
		})
	}
	pool.StopAndWait()

	if firstErr != nil {
		return nil, firstErr
	}

	response := &insightsResponse{
		From:           from,
		To:             to,
		DayCount:       len(intervals),
		Daily:          make([]dayTotalDTO, 0, len(intervals)),
		Weekdays:       make([]weekdayDTO, 7),
		Projects:       []summaryItemDTO{},
		Languages:      []summaryItemDTO{},
		Editors:        []summaryItemDTO{},
		OperatingSys:   []summaryItemDTO{},
		Machines:       []summaryItemDTO{},
		Categories:     []summaryItemDTO{},
		LanguageColors: map[string]string{},
		EditorColors:   map[string]string{},
		OSColors:       map[string]string{},
	}
	for weekday := range response.Weekdays {
		response.Weekdays[weekday].Weekday = weekday
	}

	var activeDates []time.Time

	for i, summary := range summaries {
		date := intervals[i][0]
		var seconds float64
		if summary != nil {
			// Summary.TotalTime multiplies its items by time.Second, so unlike a bare
			// SummaryItem.Total this really is a duration.
			seconds = secondsFromDuration(summary.TotalTime())
		}

		response.Daily = append(response.Daily, dayTotalDTO{Date: date, Seconds: seconds})

		weekday := int(date.Weekday())
		response.Weekdays[weekday].TotalSeconds += seconds
		response.Weekdays[weekday].Days++

		if seconds > 0 {
			response.ActiveDays++
			activeDates = append(activeDates, date)
			if response.MostActiveDay == nil || seconds > response.MostActiveDay.Seconds {
				response.MostActiveDay = &dayTotalDTO{Date: date, Seconds: seconds}
			}
		}
	}

	// Headline figures and rankings from the merged summary, so this screen agrees with
	// the dashboard.
	merged, err := h.summarySrvc.Aliased(from, to, user, h.summarySrvc.Retrieve, nil, nil, false)
	if err != nil {
		return nil, err
	}

	response.TotalSeconds = secondsFromDuration(merged.TotalTime())
	response.Projects = topItems(itemTotals(merged.Projects), insightsTopN)
	response.Languages = topItems(itemTotals(merged.Languages), insightsTopN)
	response.Editors = topItems(itemTotals(merged.Editors), insightsTopN)
	response.OperatingSys = topItems(itemTotals(merged.OperatingSystems), insightsTopN)
	response.Machines = topItems(itemTotals(merged.Machines), insightsTopN)
	response.Categories = topItems(itemTotals(merged.Categories), insightsTopN)

	categories := itemTotals(merged.Categories)
	response.AISeconds = categories[models.AICategory]
	response.HumanSeconds = categories["coding"]

	for i := range response.Weekdays {
		if response.Weekdays[i].Days > 0 {
			response.Weekdays[i].AverageSeconds = response.Weekdays[i].TotalSeconds / float64(response.Weekdays[i].Days)
		}
	}
	if response.DayCount > 0 {
		response.DailyAverage = response.TotalSeconds / float64(response.DayCount)
	}

	response.LanguageColors = colorsFor(h.config.App.GetLanguageColors(), response.Languages)
	response.EditorColors = colorsFor(h.config.App.GetEditorColors(), response.Editors)
	response.OSColors = colorsFor(h.config.App.GetOSColors(), response.OperatingSys)

	response.Streak = computeStreak(activeDates, to)

	return response, nil
}

// itemTotals reads a summary's items into a keyed map of seconds. Summary item totals
// are a second count despite their time.Duration type.
func itemTotals(items models.SummaryItems) map[string]float64 {
	totals := make(map[string]float64, len(items))
	for _, item := range items {
		totals[item.Key] += secondsFromSummaryItem(item.Total)
	}
	return totals
}

func topItems(totals map[string]float64, limit int) []summaryItemDTO {
	items := make([]summaryItemDTO, 0, len(totals))
	for key, seconds := range totals {
		if seconds <= 0 {
			continue
		}
		items = append(items, summaryItemDTO{Key: key, Seconds: seconds})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Seconds > items[j].Seconds })
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func colorsFor(palette map[string]string, items []summaryItemDTO) map[string]string {
	colors := make(map[string]string, len(items))
	for _, item := range items {
		if color, ok := palette[item.Key]; ok {
			colors[item.Key] = color
		}
	}
	return colors
}

// computeStreak measures runs of consecutive days with activity.
//
// The current streak counts back from the last day in range, and a streak that has
// today still ahead of it is not broken: nobody has coded at 00:05 yet.
func computeStreak(activeDates []time.Time, to time.Time) streakDTO {
	if len(activeDates) == 0 {
		return streakDTO{}
	}

	sort.Slice(activeDates, func(i, j int) bool { return activeDates[i].Before(activeDates[j]) })

	sameDay := func(a, b time.Time) bool {
		return a.Year() == b.Year() && a.YearDay() == b.YearDay()
	}

	var streak streakDTO
	runLength := 1
	runStart := activeDates[0]

	record := func(start, end time.Time, length int) {
		if length > streak.Longest {
			streak.Longest = length
			s, e := start, end
			streak.Start, streak.End = &s, &e
		}
	}

	for i := 1; i < len(activeDates); i++ {
		previous, current := activeDates[i-1], activeDates[i]
		if sameDay(previous.AddDate(0, 0, 1), current) {
			runLength++
			continue
		}
		record(runStart, previous, runLength)
		runStart, runLength = current, 1
	}
	record(runStart, activeDates[len(activeDates)-1], runLength)

	// The current streak only counts if it reaches the end of the range: yesterday is
	// still a live streak today, anything older has been broken.
	last := activeDates[len(activeDates)-1]
	end := to.AddDate(0, 0, -1)
	if sameDay(last, end) || sameDay(last, to) || sameDay(last.AddDate(0, 0, 1), end) {
		streak.Current = runLength
	}

	return streak
}

type wrappedYearsResponse struct {
	Years []int `json:"years"`
}

// GetWrappedYears lists the years that actually hold data, so the picker never offers a
// year that turns out to be empty.
func (h *InsightsHandler) GetWrappedYears(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	first, err := h.heartbeatSrvc.GetFirstByUser(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load first heartbeat", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load your history")
		return
	}

	years := []int{}
	if !first.IsZero() {
		for year := first.In(user.TZ()).Year(); year <= time.Now().In(user.TZ()).Year(); year++ {
			years = append(years, year)
		}
	}
	// Newest first: that is the one people want.
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	helpers.RespondJSON(w, r, http.StatusOK, &wrappedYearsResponse{Years: years})
}

type wrappedResponse struct {
	Year int `json:"year"`
	*insightsResponse
}

// GetWrapped is one calendar year of the same figures, bounded to the year rather than
// to a rolling range.
func (h *InsightsHandler) GetWrapped(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	year, err := parseYear(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "not a valid year")
		return
	}

	location := user.TZ()
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, location)
	to := from.AddDate(1, 0, 0)
	// A year still in progress ends now, so its daily average is not divided by days
	// that have not happened.
	if now := time.Now().In(location); to.After(now) {
		to = now
	}
	if from.After(time.Now()) {
		respondError(w, r, http.StatusBadRequest, "that year has not started yet")
		return
	}

	insights, err := h.buildInsights(user, from, to)
	if err != nil {
		conf.Log().Request(r).Error("failed to build wrapped", "userID", user.ID, "year", year, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load your year")
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &wrappedResponse{Year: year, insightsResponse: insights})
}

func parseYear(raw string) (int, error) {
	parsed, err := time.Parse("2006", raw)
	if err != nil {
		return 0, err
	}
	return parsed.Year(), nil
}
