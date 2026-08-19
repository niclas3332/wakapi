package services

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	svg "github.com/ajstarks/svgo/float"
	"github.com/alitto/pond/v2"
	"github.com/duke-git/lancet/v2/condition"
	"github.com/duke-git/lancet/v2/datetime"
	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
	"github.com/patrickmn/go-cache"
)

const (
	gridRows    = 7
	cellWidth   = 20
	cellHeight  = 20
	cellSpacing = 3
	textDark    = "#D1D5DB"
	textLight   = "#37474F"

	// activityLevelCount is the number of shades an active day can take. Four is what
	// a contribution grid needs to read as a gradient while every step stays tellable
	// apart; more steps would land inside the noise of the colour ramp.
	activityLevelCount = 4

	// activityChartClass scopes the chart's css. Needed because the chart is also
	// inlined into the app, where an unscoped rule would leak into the whole document.
	activityChartClass = "wakapi-activity"
)

// ChartOptions describes which activity chart to render.
type ChartOptions struct {
	Interval *models.IntervalKey
	// Project limits the chart to a single project. Empty means all activity.
	Project         string
	DarkTheme       bool
	HideAttribution bool
	SkipCache       bool
}

type ActivityService struct {
	config         *config.Config
	cache          *cache.Cache
	summaryService ISummaryService
	brandingSrvc   IBrandingService
}

func NewActivityService(summaryService ISummaryService, brandingService IBrandingService) *ActivityService {
	return &ActivityService{
		config:         config.Get(),
		cache:          cache.New(6*time.Hour, 6*time.Hour),
		summaryService: summaryService,
		brandingSrvc:   brandingService,
	}
}

// GetChart generates an activity chart for a given user and the given time interval, similar to GitHub's contribution timeline. See https://github.com/muety/wakapi/issues/12.
// Please note: currently, only yearly charts ("last_12_months") are supported. However, we could fairly easily restructure this to support dynamic intervals.
func (s *ActivityService) GetChart(user *models.User, opts ChartOptions) (string, error) {
	cacheKey := fmt.Sprintf("chart_%s_%s_%s_%v_%v_%d", user.ID, (*opts.Interval)[0], opts.Project, opts.DarkTheme, opts.HideAttribution, s.brandingSrvc.Revision())
	if result, found := s.cache.Get(cacheKey); found && !opts.SkipCache {
		return result.(string), nil
	}

	switch opts.Interval {
	case models.IntervalPast12Months:
		chart, err := s.getChartPastYear(user, opts)
		if err == nil {
			s.cache.SetDefault(cacheKey, chart) // TODO: cache compressed?
		}
		return chart, err
	default:
		return "", errors.New("unsupported interval")
	}
}

func (s *ActivityService) getChartPastYear(user *models.User, opts ChartOptions) (string, error) {
	err, from, to := helpers.ResolveIntervalTZ(models.IntervalPast12Months, user.TZ(), user.StartOfWeekDay())
	// TODO: I am not sure if we have to handle startOfWeekDay here, but it seems like we do not need to.
	from = datetime.BeginOfWeek(from, time.Monday)
	if err != nil {
		return "", err
	}

	// A single-project filter that selects only its own entity type can be served from
	// the pre-generated daily summaries, so a project chart costs about the same as the
	// overall one instead of recomputing a year of heartbeats.
	var filters *models.Filters
	if opts.Project != "" {
		filters = models.NewFiltersWith(models.SummaryProject, opts.Project).WithSelectFilteredOnly()
	}

	intervals := utils.SplitRangeByDays(from, to)
	summaries := make([]*models.Summary, len(intervals))

	wp := pond.NewPool(utils.HalfCPUs())
	mut := sync.RWMutex{}

	// fetch summaries
	for i, interval := range intervals {
		i := i // https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
		interval := interval

		wp.Submit(func() {
			summary, err := s.summaryService.Aliased(interval[0], interval[1], user, s.summaryService.Retrieve, filters, nil, false)
			if err != nil {
				config.Log().Warn("failed to retrieve summary for activity chart", "userID", user.ID, "from", from, "to", to)
				summary = models.NewEmptySummary()
				summary.FromTime = models.CustomTime(from)
				summary.ToTime = models.CustomTime(to)
				summary.UserID = user.ID
				summary.User = user
			}

			mut.Lock()
			summaries[i] = summary
			mut.Unlock()
		})
	}

	wp.StopAndWait()

	branding := s.brandingSrvc.Get()

	totals := make([]time.Duration, len(summaries))
	for i, summary := range summaries {
		totals[i] = summary.TotalTime()
	}
	levels := activityLevels(totals)

	var (
		colorRGBAMin         = utils.HexToRGBA(condition.Ternary[bool, string](opts.DarkTheme, branding.ActivityColorMinDark, branding.ActivityColorMinLight))
		colorRGBAMax         = utils.HexToRGBA(condition.Ternary[bool, string](opts.DarkTheme, branding.ActivityColorMaxDark, branding.ActivityColorMaxLight))
		colorText            = condition.Ternary[bool, string](opts.DarkTheme, textDark, textLight)
		gridCols             = math.Ceil(float64(len(summaries)) / float64(gridRows))
		w            float64 = gridCols*cellWidth + gridCols*cellSpacing
		h            float64 = gridRows*cellHeight + 25 + 24 + 5 + 5 + gridRows*cellSpacing
	)

	// regenerate svg
	buf := &bytes.Buffer{}

	canvas := svg.New(buf)
	canvas.Start(w, h)
	// Every rule is scoped to the chart's own group. A <style> element inside an svg is
	// document-scoped once the svg is inlined into a page, so bare "text" and "rect"
	// selectors would repaint every other chart on that page.
	canvas.Style("text/css",
		fmt.Sprintf(".%s text { font-family: 'Source Sans 3', Roboto, Helvetica, Arial, sans-serif; font-size: 0.9rem; font-weight: 500; fill: %s; }", activityChartClass, colorText),
		fmt.Sprintf(".%s rect { fill-opacity: 1; rx: 3px; ry: 3px; }", activityChartClass),
		fmt.Sprintf(".%s rect:hover { filter: brightness(0.9) }", activityChartClass),
	)
	canvas.Group(fmt.Sprintf(`class="%s"`, activityChartClass))

	heading := fmt.Sprintf("%s to %s", helpers.FormatDateHuman(summaries[0].FromTime.T()), helpers.FormatDateHuman(summaries[len(summaries)-1].ToTime.T()))
	if opts.Project != "" {
		heading = fmt.Sprintf("%s · %s", opts.Project, heading)
	}
	canvas.Text(0, 15, heading)

	for i, summary := range summaries {
		total := summary.TotalTime()
		fillColor := utils.RGBAToHex(utils.FadeColors(colorRGBAMin, colorRGBAMax, float64(levels[i])/float64(activityLevelCount)))

		canvas.Group()
		canvas.Title(fmt.Sprintf("%s on %s", helpers.FmtWakatimeDuration(total), helpers.FormatDateHuman(summary.FromTime.T())))
		canvas.Rect(float64(i/gridRows)*(cellWidth+cellSpacing), 25+float64((i%gridRows)*(cellHeight+cellSpacing)), cellWidth, cellHeight, fmt.Sprintf("fill: %s", fillColor))
		canvas.Gend()
	}

	if !opts.HideAttribution && branding.ActivityLogoUrl != "" {
		canvas.Group()
		canvas.Title(branding.ActivityTitle)
		canvas.Image(w-60, h-24, 60, 24, branding.ActivityLogoUrl)
		canvas.Gend()
	}

	canvas.Gend()
	canvas.End()

	return buf.String(), nil
}

// activityLevels maps each day's total onto a shading level from 0 (no activity) to
// activityLevelCount (the busiest band).
//
// Scaling linearly against the single largest day - which is what this used to do -
// makes the chart unreadable as soon as one outlier exists: against a 16 hour day, a
// full 8 hour workday lands at half brightness and a normal two hour evening is
// indistinguishable from nothing. Cutting at quartiles of the days actually worked
// keeps the shading a statement about where a day ranks among the user's own days.
func activityLevels(totals []time.Duration) []int {
	active := make([]time.Duration, 0, len(totals))
	for _, total := range totals {
		if total > 0 {
			active = append(active, total)
		}
	}

	levels := make([]int, len(totals))
	if len(active) == 0 {
		return levels
	}
	sort.Slice(active, func(i, j int) bool { return active[i] < active[j] })

	// One cut per level boundary, e.g. the 25th, 50th and 75th percentile for four levels.
	cuts := make([]time.Duration, 0, activityLevelCount-1)
	for i := 1; i < activityLevelCount; i++ {
		cuts = append(cuts, quantile(active, float64(i)/float64(activityLevelCount)))
	}

	for i, total := range totals {
		if total <= 0 {
			continue
		}
		level := 1
		for _, cut := range cuts {
			if total > cut {
				level++
			}
		}
		levels[i] = level
	}
	return levels
}

// quantile returns the q-th quantile of an ascending slice, interpolating between the
// two neighbouring samples.
func quantile(ascending []time.Duration, q float64) time.Duration {
	if len(ascending) == 1 {
		return ascending[0]
	}
	pos := q * float64(len(ascending)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return ascending[lower]
	}
	weight := pos - float64(lower)
	return ascending[lower] + time.Duration(weight*float64(ascending[upper]-ascending[lower]))
}
