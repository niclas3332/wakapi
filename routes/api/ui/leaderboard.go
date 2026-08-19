package ui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
)

const leaderboardDefaultPageSize = 100

// allowedAggregations mirrors the legacy page: language is the only dimension the
// leaderboard service aggregates by today.
var allowedAggregations = map[string]uint8{
	"language": models.SummaryLanguage,
}

type LeaderboardHandler struct {
	config             *conf.Config
	leaderboardService services.ILeaderboardService
}

func NewLeaderboardHandler(leaderboardService services.ILeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{
		config:             conf.Get(),
		leaderboardService: leaderboardService,
	}
}

func (h *LeaderboardHandler) RegisterRoutes(router chi.Router) {
	router.Get("/leaderboard", h.Get)
}

type leaderboardItem struct {
	Rank   uint   `json:"rank"`
	UserID string `json:"user_id"`
	// Seconds, like every other duration this api emits. Leaderboard totals are
	// real nanosecond durations, unlike summary item totals.
	Seconds   float64  `json:"seconds"`
	Key       *string  `json:"key"`
	IsMe      bool     `json:"is_me"`
	Languages []string `json:"languages,omitempty"`
}

type leaderboardResponse struct {
	Items []*leaderboardItem `json:"items"`
	// By is the aggregation dimension, empty for the overall ranking.
	By  string `json:"by"`
	Key string `json:"key"`
	// TopKeys are the selectable values when aggregating, e.g. the top languages.
	TopKeys       []string `json:"top_keys"`
	IntervalLabel string   `json:"interval_label"`
	Page          int      `json:"page"`
	PageSize      int      `json:"page_size"`
	HasMore       bool     `json:"has_more"`
}

// Get serves the leaderboard, either overall or aggregated by a dimension.
// Authentication is optional here when the instance allows public leaderboards, so
// there may be no principal.
func (h *LeaderboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middlewares.GetPrincipal(r)
	byParam := strings.ToLower(r.URL.Query().Get("by"))
	keyParam := strings.ToLower(r.URL.Query().Get("key"))
	pageParams := utils.ParsePageParamsWithDefault(r, 1, leaderboardDefaultPageSize)
	scope := h.leaderboardService.GetDefaultScope()

	var leaderboard models.Leaderboard
	var userLanguages map[string][]string
	var topKeys []string

	if byParam == "" {
		var err error
		leaderboard, err = h.leaderboardService.GetByInterval(scope, pageParams, true)
		if err != nil {
			conf.Log().Request(r).Error("error while fetching general leaderboard items", "error", err)
			respondError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		h.appendOwnRank(r, &leaderboard, user, pageParams)
	} else {
		by, ok := allowedAggregations[byParam]
		if !ok {
			respondError(w, r, http.StatusBadRequest, fmt.Sprintf("unsupported aggregation '%s'", byParam))
			return
		}

		var err error
		leaderboard, err = h.leaderboardService.GetAggregatedByInterval(scope, &by, pageParams, true)
		if err != nil {
			conf.Log().Request(r).Error("error while fetching aggregated leaderboard items", "error", err)
			respondError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		h.appendOwnAggregatedRank(r, &leaderboard, user, &by, pageParams)

		userLeaderboards := slice.GroupWith[*models.LeaderboardItemRanked, string](leaderboard, func(item *models.LeaderboardItemRanked) string {
			return item.UserID
		})
		userLanguages = map[string][]string{}
		for u, items := range userLeaderboards {
			userLanguages[u] = models.Leaderboard(items).TopKeysByUser(models.SummaryLanguage, u)
		}

		topKeys = leaderboard.TopKeys(by)
		if len(topKeys) > 0 {
			if keyParam == "" {
				keyParam = topKeys[0]
			}
			keyParam = strings.ToLower(keyParam)
			leaderboard = leaderboard.TopByKey(by, keyParam)
		}
	}

	leaderboard.FilterEmpty()

	items := make([]*leaderboardItem, 0, len(leaderboard))
	for _, item := range leaderboard {
		items = append(items, &leaderboardItem{
			Rank:      item.Rank,
			UserID:    item.UserID,
			Seconds:   secondsFromDuration(item.Total),
			Key:       item.Key,
			IsMe:      user != nil && item.UserID == user.ID,
			Languages: userLanguages[item.UserID],
		})
	}

	helpers.RespondJSON(w, r, http.StatusOK, &leaderboardResponse{
		Items:         items,
		By:            byParam,
		Key:           keyParam,
		TopKeys:       topKeys,
		IntervalLabel: scope.GetHumanReadable(),
		Page:          pageParams.Page,
		PageSize:      pageParams.PageSize,
		HasMore:       pageParams.PageSize > 0 && len(leaderboard) >= pageParams.PageSize,
	})
}

// appendOwnRank makes sure the current user sees their own position even when it
// falls outside the requested page. Only worth doing when the board is paginated.
func (h *LeaderboardHandler) appendOwnRank(r *http.Request, leaderboard *models.Leaderboard, user *models.User, pageParams *utils.PageParams) {
	if user == nil || leaderboard.HasUser(user.ID) {
		return
	}
	count, err := h.leaderboardService.CountUsers(true)
	if err != nil || count <= int64(pageParams.PageSize) {
		return
	}
	if l, err := h.leaderboardService.GetByIntervalAndUser(h.leaderboardService.GetDefaultScope(), user.ID, true); err == nil && len(l) > 0 {
		*leaderboard = append(*leaderboard, l[0])
	} else if err != nil {
		conf.Log().Request(r).Error("error while fetching own leaderboard rank", "error", err)
	}
}

func (h *LeaderboardHandler) appendOwnAggregatedRank(r *http.Request, leaderboard *models.Leaderboard, user *models.User, by *uint8, pageParams *utils.PageParams) {
	if user == nil {
		return
	}
	count, err := h.leaderboardService.CountUsers(true)
	if err != nil || count <= int64(pageParams.PageSize) {
		return
	}
	if l, err := h.leaderboardService.GetAggregatedByIntervalAndUser(h.leaderboardService.GetDefaultScope(), user.ID, by, true); err == nil {
		leaderboard.AddMany(l)
	} else {
		conf.Log().Request(r).Error("error while fetching own aggregated user leaderboard", "error", err)
	}
}
