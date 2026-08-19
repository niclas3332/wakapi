package ui

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
)

type BootstrapHandler struct {
	config *conf.Config
}

func NewBootstrapHandler() *BootstrapHandler {
	return &BootstrapHandler{config: conf.Get()}
}

func (h *BootstrapHandler) RegisterRoutes(router chi.Router) {
	router.Get("/bootstrap", h.Get)
	router.Get("/user", h.GetUser)
}

// userResponse is a hand-written projection of models.User.
//
// models.User cannot be serialized directly: almost every field the settings screen
// edits carries `json:"-"`, and the ones that do serialize include secrets we must
// never hand out (password hash, reset token, stripe customer id, the WakaTime key).
type userResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	ApiKey         string    `json:"api_key"`
	Location       string    `json:"location"`
	StartOfWeek    int       `json:"start_of_week"`
	CreatedAt      time.Time `json:"created_at"`
	LastLoggedInAt time.Time `json:"last_logged_in_at"`
	AuthType       string    `json:"auth_type"`
	IsAdmin        bool      `json:"is_admin"`
	HasData        bool      `json:"has_data"`

	ReportsWeekly          bool `json:"reports_weekly"`
	PublicLeaderboard      bool `json:"public_leaderboard"`
	ExcludeUnknownProjects bool `json:"exclude_unknown_projects"`
	HeartbeatsTimeoutSec   int  `json:"heartbeats_timeout_sec"`

	ReadmeStatsBaseUrl string `json:"readme_stats_base_url"`

	Sharing sharingResponse `json:"sharing"`

	// The WakaTime key itself is never returned - only whether a relay is configured.
	WakatimeConnected bool   `json:"wakatime_connected"`
	WakatimeApiUrl    string `json:"wakatime_api_url"`

	HasActiveSubscription bool       `json:"has_active_subscription"`
	SubscribedUntil       *time.Time `json:"subscribed_until"`
}

type sharingResponse struct {
	MaxDays       int  `json:"max_days"`
	Projects      bool `json:"projects"`
	Languages     bool `json:"languages"`
	Editors       bool `json:"editors"`
	OperatingSys  bool `json:"operating_systems"`
	Machines      bool `json:"machines"`
	Labels        bool `json:"labels"`
	ActivityChart bool `json:"activity_chart"`
}

type bootstrapResponse struct {
	User *userResponse `json:"user"`
}

// Get returns everything the frontend needs about the signed-in user. The static
// server config is not repeated here - it is injected into the HTML shell, because
// the router needs it synchronously before the first request can even be made.
func (h *BootstrapHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, &bootstrapResponse{User: newUserResponse(user)})
}

func (h *BootstrapHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, newUserResponse(user))
}

func newUserResponse(user *models.User) *userResponse {
	var subscribedUntil *time.Time
	if user.SubscribedUntil != nil {
		t := user.SubscribedUntil.T()
		subscribedUntil = &t
	}

	return &userResponse{
		ID:             user.ID,
		Email:          user.Email,
		ApiKey:         user.ApiKey,
		Location:       user.Location,
		StartOfWeek:    user.StartOfWeek,
		CreatedAt:      user.CreatedAt.T(),
		LastLoggedInAt: user.LastLoggedInAt.T(),
		AuthType:       user.AuthType,
		IsAdmin:        user.IsAdmin,
		HasData:        user.HasData,

		ReportsWeekly:          user.ReportsWeekly,
		PublicLeaderboard:      user.PublicLeaderboard,
		ExcludeUnknownProjects: user.ExcludeUnknownProjects,
		HeartbeatsTimeoutSec:   user.HeartbeatsTimeoutSec,

		ReadmeStatsBaseUrl: user.ReadmeStatsBaseUrl,

		Sharing: sharingResponse{
			MaxDays:       user.ShareDataMaxDays,
			Projects:      user.ShareProjects,
			Languages:     user.ShareLanguages,
			Editors:       user.ShareEditors,
			OperatingSys:  user.ShareOSs,
			Machines:      user.ShareMachines,
			Labels:        user.ShareLabels,
			ActivityChart: user.ShareActivityChart,
		},

		WakatimeConnected: user.WakatimeApiKey != "",
		WakatimeApiUrl:    user.WakatimeApiUrl,

		HasActiveSubscription: user.HasActiveSubscription(),
		SubscribedUntil:       subscribedUntil,
	}
}
