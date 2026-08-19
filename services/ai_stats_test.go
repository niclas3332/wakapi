package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
)

func aiTestUser() *models.User {
	return &models.User{ID: "alice", Location: "UTC"}
}

func heartbeatStream(heartbeats []*models.Heartbeat) chan *models.Heartbeat {
	channel := make(chan *models.Heartbeat, len(heartbeats))
	for _, heartbeat := range heartbeats {
		channel <- heartbeat
	}
	close(channel)
	return channel
}

// computeDay is the whole aggregation, so it is what the tests exercise; the database
// layer around it only stores what comes out.
func computeTestDay(t *testing.T, user *models.User, day time.Time, heartbeats []*models.Heartbeat) ([]*models.AIStatsDaily, []*models.AISessionDaily) {
	t.Helper()
	config.Set(config.Empty())

	from, to := dayBounds(user, day)

	heartbeatMock := new(mocks.HeartbeatServiceMock)
	heartbeatMock.On("StreamAllWithinRaw", from, to, user).Return(heartbeatStream(heartbeats), nil)

	srv := &AIStatsService{config: config.Get(), heartbeatSrv: heartbeatMock}
	stats, sessions, err := srv.computeDay(user, from, to)
	require.NoError(t, err)
	return stats, sessions
}

func aiHeartbeat(at time.Time, project, editor, agent string) *models.Heartbeat {
	return &models.Heartbeat{
		UserID:    "alice",
		Time:      models.CustomTime(at),
		Project:   project,
		Category:  models.AICategory,
		Editor:    editor,
		UserAgent: agent,
	}
}

const opusAgent = "wakatime/v2.23.0 (darwin) go1.26.5 opus/5 claude-code/2.1.220 claude-code-wakatime/4.1.0"

func TestAIStats_SplitsAIAndHumanWork(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	ai := aiHeartbeat(day, "wakapi", "Opus", opusAgent)
	ai.AILineChanges = 120
	ai.AIInputTokens = 1000
	ai.AICachedInputTokens = 50000
	ai.AIOutputTokens = 300

	human := &models.Heartbeat{
		UserID: "alice", Time: models.CustomTime(day), Project: "wakapi",
		Category: "coding", Editor: "PhpStorm", HumanLineChanges: 40,
	}

	stats, _ := computeTestDay(t, user, day, []*models.Heartbeat{ai, human})

	require.Len(t, stats, 2, "ai and human work belong to separate rows")

	var aiRow, humanRow *models.AIStatsDaily
	for _, row := range stats {
		if row.Model == "" {
			humanRow = row
		} else {
			aiRow = row
		}
	}

	require.NotNil(t, aiRow)
	assert.Equal(t, "opus", aiRow.Model)
	assert.Equal(t, "5", aiRow.ModelVersion)
	assert.Equal(t, 120, aiRow.AILinesAdded)
	assert.Equal(t, int64(50000), aiRow.AICachedInputTokens)

	require.NotNil(t, humanRow)
	assert.Equal(t, 40, humanRow.HumanLinesAdded)
	assert.Zero(t, humanRow.AIInputTokens)
}

// Line changes arrive as signed deltas. Summing them would let one truncated file wipe
// out a month of writing; splitting by sign keeps both the net and the volume.
func TestAIStats_NegativeLineChangesBecomeRemovals(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)

	shrunk := &models.Heartbeat{
		UserID: "alice", Time: models.CustomTime(day), Project: "wakapi",
		Category: "coding", HumanLineChanges: -34582,
	}
	grew := &models.Heartbeat{
		UserID: "alice", Time: models.CustomTime(day), Project: "wakapi",
		Category: "coding", HumanLineChanges: 50,
	}

	stats, _ := computeTestDay(t, user, day, []*models.Heartbeat{shrunk, grew})

	require.Len(t, stats, 1)
	assert.Equal(t, 50, stats[0].HumanLinesAdded)
	assert.Equal(t, 34582, stats[0].HumanLinesRemoved)
}

// A session spanning two projects is one session, so it is stored once. Counting it
// from the per-project stats rows would inflate the number.
func TestAIStats_SessionsAreDeduplicatedAcrossProjects(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)

	first := aiHeartbeat(day, "wakapi", "Opus", opusAgent)
	first.AISession = "session-1"
	first.AILineChanges = 10
	second := aiHeartbeat(day, "other-project", "Opus", opusAgent)
	second.AISession = "session-1"
	second.AILineChanges = 5

	stats, sessions := computeTestDay(t, user, day, []*models.Heartbeat{first, second})

	assert.Len(t, stats, 2, "two projects, two stats rows")
	assert.Len(t, sessions, 1, "but one session")
	assert.Equal(t, "session-1", sessions[0].SessionID)
}

// A day of ordinary coding with no line data carries nothing this table is for, and the
// summaries already cover the time. Storing a row per project per day anyway would
// double the database for no answer.
func TestAIStats_DaysWithoutAnythingToRecordAreSkipped(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 11, 0, 0, 0, time.UTC)

	plain := &models.Heartbeat{
		UserID: "alice", Time: models.CustomTime(day), Project: "wakapi", Category: "coding",
	}

	stats, sessions := computeTestDay(t, user, day, []*models.Heartbeat{plain, plain})

	assert.Empty(t, stats)
	assert.Empty(t, sessions)
}

// Prompts are counted where a length is reported: a zero length is no evidence that a
// prompt happened.
func TestAIStats_PromptsAreCountedByReportedLength(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 16, 0, 0, 0, time.UTC)

	withPrompt := aiHeartbeat(day, "wakapi", "Opus", opusAgent)
	withPrompt.AIPromptLength = 320
	withPrompt.AILineChanges = 4
	withoutPrompt := aiHeartbeat(day, "wakapi", "Opus", opusAgent)
	withoutPrompt.AILineChanges = 2

	stats, _ := computeTestDay(t, user, day, []*models.Heartbeat{withPrompt, withoutPrompt})

	require.Len(t, stats, 1)
	assert.Equal(t, 1, stats[0].PromptCount)
	assert.Equal(t, int64(320), stats[0].PromptLengthTotal)
	assert.Equal(t, 2, stats[0].HeartbeatCount)
}

// Two versions of the same model are priced differently, so they must not share a row.
func TestAIStats_ModelVersionsAreKeptApart(t *testing.T) {
	user := aiTestUser()
	day := time.Date(2026, 8, 18, 13, 0, 0, 0, time.UTC)

	newer := aiHeartbeat(day, "wakapi", "Opus", opusAgent)
	newer.AIInputTokens = 100
	older := aiHeartbeat(day, "wakapi", "Opus", "wakatime/v2.22.2 (darwin) go1.26.5 opus/4-8 claude-code/2.1.215")
	older.AIInputTokens = 200

	stats, _ := computeTestDay(t, user, day, []*models.Heartbeat{newer, older})

	require.Len(t, stats, 2)
	versions := map[string]int64{}
	for _, row := range stats {
		versions[row.ModelVersion] = row.AIInputTokens
	}
	assert.Equal(t, int64(100), versions["5"])
	assert.Equal(t, int64(200), versions["4.8"])
}

// The day boundary is midnight where the user is, not where the server is.
func TestAIStats_DayBoundsFollowTheUsersTimeZone(t *testing.T) {
	user := &models.User{ID: "alice", Location: "Europe/Berlin"}

	// 00:30 Berlin on 19 August is still 22:30 UTC on the 18th.
	at := time.Date(2026, 8, 18, 22, 30, 0, 0, time.UTC)
	from, to := dayBounds(user, at)

	assert.Equal(t, 2026, from.Year())
	assert.Equal(t, time.August, from.Month())
	assert.Equal(t, 19, from.Day())
	assert.Equal(t, 0, from.Hour())
	assert.Equal(t, 24*time.Hour, to.Sub(from))
}
