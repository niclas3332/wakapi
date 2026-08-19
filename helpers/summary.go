package helpers

import (
	"errors"
	"net/http"
	"time"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
)

// summaryCacheGrid is the resolution the summary range is snapped to.
//
// Open-ended ranges end at time.Now(), so without this every request carries a
// nanosecond-unique upper bound: the summary cache demands second precision to store
// anything at all and a matching key to serve it, meaning a dashboard load recomputed
// from scratch every single time. Snapping both ends onto a shared grid gives the
// cache a stable key for the length of one grid step.
//
// The upper bound is rounded up rather than down, so the range never stops short of
// now and no recorded activity can fall outside it. The cost is that a summary may be
// served up to one grid step old.
const summaryCacheGrid = 30 * time.Second

func ParseSummaryParams(r *http.Request) (*models.SummaryParams, error) {
	user := extractUser(r)
	params := r.URL.Query()

	var err error
	var from, to time.Time

	if interval := params.Get("interval"); interval != "" {
		err, from, to = ResolveIntervalRawTZ(interval, user.TZ(), user.StartOfWeekDay())
		if err != nil {
			// Silently answering an unknown interval with an empty range makes a typo
			// look like "you tracked nothing", which is worse than saying so.
			return nil, err
		}
	} else if start := params.Get("start"); start != "" {
		err, from, to = ResolveIntervalRawTZ(start, user.TZ(), user.StartOfWeekDay())
		if err != nil {
			return nil, err
		}
	} else {
		from, err = ParseDateTimeTZ(params.Get("from"), user.TZ())
		if err != nil {
			return nil, errors.New("missing or invalid 'from' parameter")
		}

		to, err = ParseDateTimeTZ(params.Get("to"), user.TZ())
		if err != nil {
			return nil, errors.New("missing or invalid 'to' parameter")
		}
	}

	recompute := params.Get("recompute") != "" && params.Get("recompute") != "false"

	filters := ParseSummaryFilters(r)

	return &models.SummaryParams{
		From:      snapDown(from),
		To:        snapUp(to),
		User:      user,
		Recompute: recompute,
		Filters:   filters,
	}, nil
}

func snapDown(t time.Time) time.Time {
	return t.Truncate(summaryCacheGrid)
}

func snapUp(t time.Time) time.Time {
	if snapped := t.Truncate(summaryCacheGrid); snapped.Equal(t) {
		return t
	} else {
		return snapped.Add(summaryCacheGrid)
	}
}

func ParseSummaryFilters(r *http.Request) *models.Filters {
	filters := &models.Filters{}
	if q := r.URL.Query().Get("project"); q != "" {
		filters.With(models.SummaryProject, q)
	}
	if q := r.URL.Query().Get("language"); q != "" {
		filters.With(models.SummaryLanguage, q)
	}
	if q := r.URL.Query().Get("editor"); q != "" {
		filters.With(models.SummaryEditor, q)
	}
	if q := r.URL.Query().Get("machine"); q != "" {
		filters.With(models.SummaryMachine, q)
	}
	if q := r.URL.Query().Get("operating_system"); q != "" {
		filters.With(models.SummaryOS, q)
	}
	if q := r.URL.Query().Get("label"); q != "" {
		filters.With(models.SummaryLabel, q)
	}
	if q := r.URL.Query().Get("branch"); q != "" {
		filters.With(models.SummaryBranch, q)
	}
	if q := r.URL.Query().Get("entity"); q != "" {
		filters.With(models.SummaryBranch, q)
	}
	if q := r.URL.Query().Get("category"); q != "" {
		filters.With(models.SummaryCategory, q)
	}
	return filters
}

func extractUser(r *http.Request) *models.User {
	sharedData := r.Context().Value(config.KeySharedData)
	if sharedData == nil {
		config.Log().Error("request shared data not set while retrieving principal")
		return nil
	}
	val := sharedData.(*config.SharedData).MustGet(config.MiddlewareKeyPrincipal)
	if val == nil {
		return nil
	}
	return val.(*models.User)
}
