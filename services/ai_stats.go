package services

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/repositories"
)

// AIStatsService turns raw heartbeats into the AI figures the insights screens read.
//
// Aggregation happens in Go rather than in SQL on purpose. The heartbeat time column is
// an INTEGER of epoch milliseconds on sqlite and a timestamp elsewhere, so a GROUP BY
// DATE(time) would mean one query per dialect and would silently produce nonsense on
// sqlite. Day boundaries also depend on the user's own time zone, which the database
// does not know. Reading rows and grouping them here is the only version that is
// correct on every dialect wakapi supports.
type AIStatsService struct {
	config       *config.Config
	repository   *repositories.AIStatsRepository
	heartbeatSrv IHeartbeatService
	kvSrvc       IKeyValueService

	prices atomic.Pointer[models.AIPriceTable]
}

func NewAIStatsService(repository *repositories.AIStatsRepository, heartbeatService IHeartbeatService, keyValueService IKeyValueService) *AIStatsService {
	srv := &AIStatsService{
		config:       config.Get(),
		repository:   repository,
		heartbeatSrv: heartbeatService,
		kvSrvc:       keyValueService,
	}
	srv.reloadPrices()
	return srv
}

// Prices returns the effective price table. Never nil: an instance with no stored table
// still prices with the built-in defaults.
func (srv *AIStatsService) Prices() models.AIPriceTable {
	if table := srv.prices.Load(); table != nil {
		return *table
	}
	return models.DefaultAIPrices()
}

// SetPrices stores an operator's price table.
func (srv *AIStatsService) SetPrices(table models.AIPriceTable) error {
	payload, err := json.Marshal(table)
	if err != nil {
		return err
	}
	if err := srv.kvSrvc.PutString(&models.KeyStringValue{Key: models.AIPricesKey, Value: string(payload)}); err != nil {
		return err
	}
	srv.reloadPrices()
	return nil
}

// HasStoredPrices reports whether an operator has edited the table, so the settings
// screen can say whether it is showing defaults or their own numbers.
func (srv *AIStatsService) HasStoredPrices() bool {
	kv, err := srv.kvSrvc.GetString(models.AIPricesKey)
	return err == nil && kv != nil && kv.Value != ""
}

func (srv *AIStatsService) reloadPrices() {
	table := models.DefaultAIPrices()

	if kv, err := srv.kvSrvc.GetString(models.AIPricesKey); err == nil && kv != nil && kv.Value != "" {
		stored := models.AIPriceTable{}
		if err := json.Unmarshal([]byte(kv.Value), &stored); err != nil {
			config.Log().Error("failed to parse stored ai price table, using defaults", "error", err)
		} else {
			table = stored
		}
	}

	srv.prices.Store(&table)
}

// dayBounds returns the start of the day a moment falls into, in the user's time zone,
// plus the start of the next one.
func dayBounds(user *models.User, at time.Time) (time.Time, time.Time) {
	local := at.In(user.TZ())
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, user.TZ())
	return start, start.AddDate(0, 0, 1)
}

// AggregateDay recomputes one calendar day for one user and replaces what was stored.
func (srv *AIStatsService) AggregateDay(user *models.User, day time.Time) error {
	from, to := dayBounds(user, day)

	stats, sessions, err := srv.computeDay(user, from, to)
	if err != nil {
		return err
	}
	return srv.repository.ReplaceDay(user.ID, from, stats, sessions)
}

// computeDay reads a day's heartbeats and folds them into rows. It is also what the
// live path uses for the current day, which is why it does not touch the database.
func (srv *AIStatsService) computeDay(user *models.User, from, to time.Time) ([]*models.AIStatsDaily, []*models.AISessionDaily, error) {
	heartbeats, err := srv.heartbeatSrv.StreamAllWithinRaw(from, to, user)
	if err != nil {
		return nil, nil, err
	}

	statsByKey := map[models.AIStatsKey]*models.AIStatsDaily{}
	sessionsById := map[string]*models.AISessionDaily{}

	for heartbeat := range heartbeats {
		isAI := heartbeat.IsAI()

		var model, version string
		if isAI {
			model, version = heartbeat.AIModel()
		}

		key := models.AIStatsKey{Date: from, Project: heartbeat.Project, Model: model, ModelVersion: version}
		row, ok := statsByKey[key]
		if !ok {
			row = &models.AIStatsDaily{
				UserID:       user.ID,
				Date:         models.CustomTime(from),
				Project:      heartbeat.Project,
				Model:        model,
				ModelVersion: version,
			}
			statsByKey[key] = row
		}

		row.HeartbeatCount++
		models.AddLineDelta(&row.HumanLinesAdded, &row.HumanLinesRemoved, heartbeat.HumanLineChanges)
		row.LineAdditions += heartbeat.LineAdditions
		row.LineDeletions += heartbeat.LineDeletions

		if !isAI {
			continue
		}

		models.AddLineDelta(&row.AILinesAdded, &row.AILinesRemoved, heartbeat.AILineChanges)
		row.AIInputTokens += int64(heartbeat.AIInputTokens)
		row.AICachedInputTokens += int64(heartbeat.AICachedInputTokens)
		row.AIOutputTokens += int64(heartbeat.AIOutputTokens)

		// A prompt is counted where its length is reported; a zero length carries no
		// evidence that a prompt happened at all.
		if heartbeat.AIPromptLength > 0 {
			row.PromptCount++
			row.PromptLengthTotal += int64(heartbeat.AIPromptLength)
		}

		if heartbeat.AISession != "" {
			if _, seen := sessionsById[heartbeat.AISession]; !seen {
				sessionsById[heartbeat.AISession] = &models.AISessionDaily{
					UserID:    user.ID,
					Date:      models.CustomTime(from),
					SessionID: heartbeat.AISession,
					Project:   heartbeat.Project,
					Model:     model,
				}
			}
		}
	}

	stats := make([]*models.AIStatsDaily, 0, len(statsByKey))
	for _, row := range statsByKey {
		if row.IsEmpty() {
			continue
		}
		stats = append(stats, row)
	}

	sessions := make([]*models.AISessionDaily, 0, len(sessionsById))
	for _, session := range sessionsById {
		sessions = append(sessions, session)
	}

	return stats, sessions, nil
}

// AggregateRange recomputes every complete day in [from, to), skipping the day that is
// still in progress - that one is recomputed live on every read instead.
func (srv *AIStatsService) AggregateRange(user *models.User, from, to time.Time) error {
	today, _ := dayBounds(user, time.Now())

	day, _ := dayBounds(user, from)
	for day.Before(to) && day.Before(today) {
		if err := srv.AggregateDay(user, day); err != nil {
			return err
		}
		day = day.AddDate(0, 0, 1)
	}
	return nil
}

// AggregateSince recomputes from the last stored day onwards, which is what the nightly
// job wants. With nothing stored it starts at the user's first heartbeat.
func (srv *AIStatsService) AggregateSince(user *models.User) error {
	last, err := srv.repository.GetLastDate(user.ID)
	if err != nil {
		return err
	}

	from := last
	if from.IsZero() {
		first, err := srv.heartbeatSrv.GetFirstByUser(user)
		if err != nil {
			return err
		}
		if first.IsZero() {
			return nil
		}
		from = first
	}

	return srv.AggregateRange(user, from, time.Now())
}

// Retrieve returns the rows covering [from, to), taking the completed days from the
// pre-aggregated table and computing the current day live on top - the same shape
// SummaryService uses, and the reason a dashboard is both fast and current.
func (srv *AIStatsService) Retrieve(user *models.User, from, to time.Time) ([]*models.AIStatsDaily, []*models.AISessionDaily, error) {
	today, _ := dayBounds(user, time.Now())

	storedTo := to
	if storedTo.After(today) {
		storedTo = today
	}

	stats, err := srv.repository.GetStatsWithin(user, from, storedTo)
	if err != nil {
		return nil, nil, err
	}
	sessions, err := srv.repository.GetSessionsWithin(user, from, storedTo)
	if err != nil {
		return nil, nil, err
	}

	if to.After(today) {
		liveFrom, liveTo := today, today.AddDate(0, 0, 1)
		liveStats, liveSessions, err := srv.computeDay(user, liveFrom, liveTo)
		if err != nil {
			return nil, nil, err
		}
		stats = append(stats, liveStats...)
		sessions = append(sessions, liveSessions...)
	}

	return stats, sessions, nil
}

func (srv *AIStatsService) DeleteByUser(userId string) error {
	return srv.repository.DeleteByUser(userId)
}

// EarliestCachedTokenDay reports the first day whose stored rows carry cache tokens.
//
// Before the column existed the cli's cached-input figure was decoded and thrown away,
// so anything older is missing the bulk of its tokens and its cost is understated. The
// number cannot be recovered - the raw payload was never stored - so the screens say so
// rather than presenting a total that quietly excludes most of it.
func (srv *AIStatsService) EarliestCachedTokenDay(user *models.User) (time.Time, error) {
	stats, err := srv.repository.GetStatsWithin(user, time.Time{}, time.Now().AddDate(0, 0, 1))
	if err != nil {
		return time.Time{}, err
	}

	var earliest time.Time
	for _, row := range stats {
		if row.AICachedInputTokens == 0 {
			continue
		}
		if earliest.IsZero() || row.Date.T().Before(earliest) {
			earliest = row.Date.T()
		}
	}
	return earliest, nil
}

// StartOfDay exposes the day boundary rule to callers that need it, so nobody
// reimplements "midnight in the user's zone" slightly differently.
func StartOfDay(user *models.User, at time.Time) time.Time {
	start, _ := dayBounds(user, at)
	return start
}
