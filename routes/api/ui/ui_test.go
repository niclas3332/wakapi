package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/utils"
)

func init() {
	config.Set(config.Empty())
}

// newRequest builds a request carrying the shared-data container the principal
// helpers rely on. Passing a nil user simulates an unauthenticated request.
func newRequest(target string, user *models.User) *http.Request {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r = r.WithContext(context.WithValue(r.Context(), config.KeySharedData, config.NewSharedData()))
	if user != nil {
		routeutils.SetPrincipal(r, user)
	}
	return r
}

func testUser() *models.User {
	return &models.User{ID: "alice", ApiKey: "key", Location: "UTC"}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestBootstrap_RequiresPrincipal(t *testing.T) {
	sut := NewBootstrapHandler()

	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/bootstrap", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// The user DTO must never carry secrets. models.User holds a password hash, reset
// token and the WakaTime api key, none of which may reach the browser.
func TestBootstrap_DoesNotLeakSecrets(t *testing.T) {
	user := testUser()
	user.Password = "hashed-password"
	user.ResetToken = "reset-token"
	user.WakatimeApiKey = "wakatime-secret"
	user.StripeCustomerId = "cus_123"

	sut := NewBootstrapHandler()
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/bootstrap", user))

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.NotContains(t, body, "hashed-password")
	assert.NotContains(t, body, "reset-token")
	assert.NotContains(t, body, "wakatime-secret")
	assert.NotContains(t, body, "cus_123")

	// but it does report that a relay is configured
	resp := decode[bootstrapResponse](t, rec)
	assert.True(t, resp.User.WakatimeConnected)
	assert.Equal(t, "alice", resp.User.ID)
}

func TestLeaderboard_RejectsUnsupportedAggregation(t *testing.T) {
	leaderboardMock := new(mocks.LeaderboardServiceMock)
	leaderboardMock.On("GetDefaultScope").Return(models.IntervalPast7Days)

	sut := NewLeaderboardHandler(leaderboardMock)
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/leaderboard?by=editor", testUser()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	// no leaderboard was fetched for an aggregation we do not support
	leaderboardMock.AssertNotCalled(t, "GetAggregatedByInterval", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLeaderboard_MarksOwnEntry(t *testing.T) {
	user := testUser()
	board := models.Leaderboard{
		{LeaderboardItem: models.LeaderboardItem{UserID: "bob", Total: 2 * time.Hour}, Rank: 1},
		{LeaderboardItem: models.LeaderboardItem{UserID: "alice", Total: 1 * time.Hour}, Rank: 2},
	}

	leaderboardMock := new(mocks.LeaderboardServiceMock)
	leaderboardMock.On("GetDefaultScope").Return(models.IntervalPast7Days)
	leaderboardMock.On("GetByInterval", mock.Anything, mock.Anything, true).Return(board, nil)

	sut := NewLeaderboardHandler(leaderboardMock)
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/leaderboard", user))

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decode[leaderboardResponse](t, rec)
	assert.Len(t, resp.Items, 2)
	assert.False(t, resp.Items[0].IsMe)
	assert.True(t, resp.Items[1].IsMe)
}

// Entries without any tracked time are dropped, matching the legacy page.
func TestLeaderboard_DropsEmptyEntries(t *testing.T) {
	board := models.Leaderboard{
		{LeaderboardItem: models.LeaderboardItem{UserID: "bob", Total: time.Hour}, Rank: 1},
		{LeaderboardItem: models.LeaderboardItem{UserID: "carol", Total: 0}, Rank: 2},
	}

	leaderboardMock := new(mocks.LeaderboardServiceMock)
	leaderboardMock.On("GetDefaultScope").Return(models.IntervalPast7Days)
	leaderboardMock.On("GetByInterval", mock.Anything, mock.Anything, true).Return(board, nil)

	sut := NewLeaderboardHandler(leaderboardMock)
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/leaderboard", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decode[leaderboardResponse](t, rec)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "bob", resp.Items[0].UserID)
}

func TestProjects_PassesSearchAndPaging(t *testing.T) {
	user := testUser()
	stats := []*models.ProjectStats{
		{Project: "wakapi", TopLanguage: "Go", Count: 42, First: models.CustomTime(time.Unix(1000, 0)), Last: models.CustomTime(time.Unix(2000, 0))},
	}

	projectMock := new(mocks.ProjectServiceMock)
	pagingMatches := mock.MatchedBy(func(p *utils.PageParams) bool {
		return p != nil && p.Page == 1 && p.PageSize == 1
	})
	projectMock.On("GetUserProjectStats", user, mock.Anything, mock.Anything, "waka", pagingMatches, false).Return(stats, nil)

	sut := NewProjectsHandler(projectMock, nil, nil)
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/projects?page=1&page_size=1&q=waka", user))

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decode[pagedResponse[projectItem]](t, rec)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "wakapi", resp.Items[0].Project)
	assert.Equal(t, int64(42), resp.Items[0].Count)
	assert.Equal(t, 1, resp.PageSize)
	// a full page means there may be another one
	assert.True(t, resp.HasMore)
	projectMock.AssertExpectations(t)
}

func TestProjects_RequiresPrincipal(t *testing.T) {
	sut := NewProjectsHandler(new(mocks.ProjectServiceMock), nil, nil)
	rec := httptest.NewRecorder()
	sut.Get(rec, newRequest("/api/ui/projects", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// A range outside the supported window is answered with available=false rather than
// an error, and must not trigger any summary computation.
func TestTimeline_OutOfRangeIsNotAnError(t *testing.T) {
	summaryMock := new(mocks.SummaryServiceMock)

	sut := NewSummaryHandler(summaryMock, nil, nil, nil)
	rec := httptest.NewRecorder()
	sut.GetTimeline(rec, newRequest("/api/ui/summary/timeline?interval=today", testUser()))

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decode[timelineResponse](t, rec)
	assert.False(t, resp.Available)
	assert.Empty(t, resp.Items)
	assert.Equal(t, routeutils.DailyStatsMinRangeDays, resp.MinRangeDays)
	summaryMock.AssertNotCalled(t, "Aliased", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSummary_RequiresPrincipal(t *testing.T) {
	sut := NewSummaryHandler(new(mocks.SummaryServiceMock), nil, nil, nil)

	for name, handler := range map[string]http.HandlerFunc{
		"summary":  sut.Get,
		"timeline": sut.GetTimeline,
		"hourly":   sut.GetHourly,
	} {
		rec := httptest.NewRecorder()
		handler(rec, newRequest("/api/ui/summary", nil))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s must require a principal", name)
	}
}
