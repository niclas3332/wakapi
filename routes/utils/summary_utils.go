package utils

import (
	"net/http"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/models/types"
	"github.com/muety/wakapi/models/view"
	"github.com/muety/wakapi/services"
	wutils "github.com/muety/wakapi/utils"
)

const (
	// DailyStatsMinRangeDays and DailyStatsMaxRangeDays bound the range for which a
	// per-day timeline is worth computing: below the minimum there is nothing to plot,
	// above the maximum it becomes both slow and unreadable.
	DailyStatsMinRangeDays = 3
	DailyStatsMaxRangeDays = 31

	// HourlyBreakdownMaxDurations caps the hourly breakdown. Plotting excessively many
	// small segments freezes the browser and the chart is unreadable anyway.
	// See https://github.com/muety/wakapi/issues/871
	HourlyBreakdownMaxDurations = 200
)

func LoadUserSummary(ss services.ISummaryService, r *http.Request) (*models.Summary, error, int) {
	summaryParams, err := helpers.ParseSummaryParams(r)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}
	return LoadUserSummaryByParams(ss, summaryParams)
}

func LoadUserSummaryByParams(ss services.ISummaryService, params *models.SummaryParams) (*models.Summary, error, int) {
	var retrieveSummary types.SummaryRetriever = ss.Retrieve
	if params.Recompute {
		retrieveSummary = ss.Summarize
	}

	summary, err := ss.Aliased(
		params.From,
		params.To,
		params.User,
		retrieveSummary,
		params.Filters,
		nil,
		params.Recompute,
	)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	summary.User = params.User
	summary.FromTime = models.CustomTime(summary.FromTime.T().In(params.User.TZ()))
	summary.ToTime = models.CustomTime(summary.ToTime.T().In(params.User.TZ()))

	return summary, nil, http.StatusOK
}

func LoadUserSummaryWithoutFilter(ss services.ISummaryService, params *models.SummaryParams) (*models.Summary, error, int) {
	var retrieveSummary types.SummaryRetriever = ss.Retrieve
	if params.Recompute {
		retrieveSummary = ss.Summarize
	}

	summary, err := ss.Aliased(
		params.From,
		params.To,
		params.User,
		retrieveSummary,
		nil, // no filters
		nil,
		params.Recompute,
	)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	summary.User = params.User
	summary.FromTime = models.CustomTime(summary.FromTime.T().In(params.User.TZ()))
	summary.ToTime = models.CustomTime(summary.ToTime.T().In(params.User.TZ()))

	return summary, nil, http.StatusOK
}

// FetchSplitSummaries returns one aliased summary per calendar day covered by params,
// which is what the timeline chart is built from.
func FetchSplitSummaries(ss services.ISummaryService, params *models.SummaryParams) ([]*models.Summary, error) {
	intervals := wutils.SplitRangeByDays(params.From, params.To)
	summaries := make([]*models.Summary, 0, len(intervals))
	for _, interval := range intervals {
		curSummary, err := ss.Aliased(interval[0], interval[1], params.User, ss.Retrieve, params.Filters, nil, false)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, curSummary)
	}
	return summaries, nil
}

// ExtractAvailableFilters collects the filter values a summary offers, so the UI can
// populate its dropdowns. Fed with an unfiltered summary, it lists everything the user
// could filter by in the selected range.
func ExtractAvailableFilters(summary *models.Summary) view.AvailableFilters {
	names := func(items models.SummaryItems) []string {
		return slice.Map(items, func(_ int, item *models.SummaryItem) string { return item.Key })
	}
	return view.AvailableFilters{
		ProjectNames:  names(summary.Projects),
		LanguageNames: names(summary.Languages),
		MachineNames:  names(summary.Machines),
		LabelNames:    names(summary.Labels),
		CategoryNames: names(summary.Categories),
	}
}

// HourlyBreakdown is the hourly chart data together with the window it actually
// covers, which is at most the last 24 hours of the requested range.
type HourlyBreakdown struct {
	From      time.Time
	Items     view.HourlyBreakdownsViewModel
	Truncated bool
}

// BuildHourlyBreakdown assembles the per-project hourly breakdown from raw durations.
// Truncated reports that the range held too many segments to plot, in which case
// Items is empty - that is a display limitation, not an error.
func BuildHourlyBreakdown(ds services.IDurationService, as services.IAliasService, params *models.SummaryParams) (*HourlyBreakdown, error) {
	from := params.From
	if params.RangeDays() > 1 {
		from = params.To.Add(-24 * time.Hour)
	}

	result := &HourlyBreakdown{From: from, Items: view.HourlyBreakdownsViewModel{}}

	durations, err := ds.Get(from, params.To, params.User, params.Filters, nil, false)
	if err != nil {
		return result, err
	}

	if len(durations) > HourlyBreakdownMaxDurations {
		result.Truncated = true
		return result, nil
	}

	result.Items = view.NewHourlyBreakdownViewModel(view.NewHourlyBreakdownItems(durations, func(t uint8, k string) string {
		s, _ := as.GetAliasOrDefault(params.User.ID, t, k)
		return s
	}))
	return result, nil
}

func FilterColors(all map[string]string, haystack models.SummaryItems) map[string]string {
	subset := make(map[string]string)
	for _, item := range haystack {
		if c, ok := all[strings.ToLower(item.Key)]; ok {
			subset[strings.ToLower(item.Key)] = c
		}
	}
	return subset
}
