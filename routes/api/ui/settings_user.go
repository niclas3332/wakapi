package ui

import (
	"net/http"
	"slices"
	"strings"
	"time"

	datastructure "github.com/duke-git/lancet/v2/datastructure/set"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
)

type profileRequest struct {
	Email             string `json:"email"`
	Location          string `json:"location"`
	StartOfWeek       int    `json:"start_of_week"`
	ReportsWeekly     bool   `json:"reports_weekly"`
	PublicLeaderboard bool   `json:"public_leaderboard"`
}

func (h *SettingsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload profileRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	update := models.UserDataUpdate{
		Email:             strings.TrimSpace(payload.Email),
		Location:          payload.Location,
		StartOfWeek:       payload.StartOfWeek,
		ReportsWeekly:     payload.ReportsWeekly,
		PublicLeaderboard: payload.PublicLeaderboard,
	}
	if !update.IsValid() {
		respondError(w, r, http.StatusBadRequest, "invalid profile data - is the e-mail address or time zone valid?")
		return
	}
	// An active subscription is tied to an address we can reach, so it cannot be dropped.
	if update.Email == "" && user.HasActiveSubscription() {
		respondError(w, r, http.StatusBadRequest, "cannot unset your e-mail address while a subscription is active")
		return
	}

	user.Email = update.Email
	user.Location = update.Location
	user.StartOfWeek = update.StartOfWeek
	user.ReportsWeekly = update.ReportsWeekly
	user.PublicLeaderboard = update.PublicLeaderboard

	if _, err := h.userSrvc.Update(user); err != nil {
		if strings.Contains(err.Error(), "email address already in use") {
			respondError(w, r, http.StatusConflict, "that e-mail address is already taken")
			return
		}
		conf.Log().Request(r).Error("failed to update user profile", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	respondMessage(w, r, http.StatusOK, "Profile saved.")
}

type passwordRequest struct {
	PasswordOld    string `json:"password_old"`
	PasswordNew    string `json:"password_new"`
	PasswordRepeat string `json:"password_repeat"`
}

func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	if user.AuthType != "local" {
		respondError(w, r, http.StatusBadRequest, "your password is managed by your identity provider")
		return
	}

	var payload passwordRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	if !utils.ComparePassword(user.Password, payload.PasswordOld, h.config.Security.PasswordSalt) {
		respondError(w, r, http.StatusUnauthorized, "your current password is not correct")
		return
	}

	credentials := models.CredentialsReset{
		PasswordOld:    payload.PasswordOld,
		PasswordNew:    payload.PasswordNew,
		PasswordRepeat: payload.PasswordRepeat,
	}
	if !credentials.IsValid() {
		respondError(w, r, http.StatusBadRequest, "the new password must be at least 6 characters and both entries must match")
		return
	}

	hash, err := utils.HashPassword(credentials.PasswordNew, h.config.Security.PasswordSalt)
	if err != nil {
		conf.Log().Request(r).Error("failed to hash password", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	user.Password = hash

	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to change password", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	respondMessage(w, r, http.StatusOK, "Password changed.")
}

type usernameRequest struct {
	Username string `json:"username"`
}

// usernameResponse tells the client whether the session survived. Changing the user id
// invalidates a local session, because the auth cookie names the old id.
type usernameResponse struct {
	Message         string `json:"message"`
	ReloginRequired bool   `json:"relogin_required"`
}

func (h *SettingsHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload usernameRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	username := strings.TrimSpace(payload.Username)
	if !models.ValidateUsername(username) || username == user.ID {
		respondError(w, r, http.StatusBadRequest, "pick a different username without spaces")
		return
	}
	if existing, _ := h.userSrvc.GetUserById(username); existing != nil {
		respondError(w, r, http.StatusConflict, "that username is already taken")
		return
	}

	if _, err := h.userSrvc.ChangeUserId(user, username); err != nil {
		conf.Log().Request(r).Error("failed to change username", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	// An OIDC session is keyed on the provider's subject, not on the local id, so it
	// stays valid. A local session does not.
	if slices.Contains(h.config.Security.ListOidcProviders(), user.AuthType) {
		helpers.RespondJSON(w, r, http.StatusOK, &usernameResponse{
			Message:         "Username changed to " + username + ".",
			ReloginRequired: false,
		})
		return
	}

	http.SetCookie(w, h.config.GetClearCookie(models.AuthCookieKey))
	helpers.RespondJSON(w, r, http.StatusOK, &usernameResponse{
		Message:         "Username changed to " + username + ". Please log in again.",
		ReloginRequired: true,
	})
}

type apiKeyResetResponse struct {
	Message string `json:"message"`
	ApiKey  string `json:"api_key"`
}

func (h *SettingsHandler) ResetApiKey(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	if _, err := h.userSrvc.ResetApiKey(user); err != nil {
		conf.Log().Request(r).Error("failed to reset api key", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &apiKeyResetResponse{
		Message: "Your API key was regenerated. Update it in every editor plugin.",
		ApiKey:  user.ApiKey,
	})
}

type sharingRequest struct {
	MaxDays       int  `json:"max_days"`
	Projects      bool `json:"projects"`
	Languages     bool `json:"languages"`
	Editors       bool `json:"editors"`
	OperatingSys  bool `json:"operating_systems"`
	Machines      bool `json:"machines"`
	Labels        bool `json:"labels"`
	ActivityChart bool `json:"activity_chart"`
}

func (h *SettingsHandler) UpdateSharing(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	var payload sharingRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	user.ShareDataMaxDays = payload.MaxDays
	user.ShareProjects = payload.Projects
	user.ShareLanguages = payload.Languages
	user.ShareEditors = payload.Editors
	user.ShareOSs = payload.OperatingSys
	user.ShareMachines = payload.Machines
	user.ShareLabels = payload.Labels
	user.ShareActivityChart = payload.ActivityChart

	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to update sharing settings", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	// Flushed only after a successful write - a rejected request changed nothing, so
	// throwing the cache away would be pure cost.
	h.userSrvc.FlushUserCache(user.ID)

	respondMessage(w, r, http.StatusOK, "Sharing settings saved.")
}

type enabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *SettingsHandler) UpdateLeaderboardOptIn(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	var payload enabledRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	user.PublicLeaderboard = payload.Enabled
	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to update leaderboard opt-in", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	h.userSrvc.FlushCache()

	respondMessage(w, r, http.StatusOK, "Leaderboard setting saved.")
}

// UpdateExcludeUnknownProjects also regenerates summaries, because the flag changes how
// past heartbeats are counted, not just future ones.
func (h *SettingsHandler) UpdateExcludeUnknownProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	var payload enabledRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	if !h.lockAggregation(user.ID) {
		respondError(w, r, http.StatusConflict, "a summary regeneration is already running, please wait for it to finish")
		return
	}

	user.ExcludeUnknownProjects = payload.Enabled
	if _, err := h.userSrvc.Update(user); err != nil {
		h.unlockAggregation(user.ID)
		conf.Log().Request(r).Error("failed to update exclude-unknown-projects", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	h.userSrvc.FlushCache()

	go func(user *models.User) {
		defer h.unlockAggregation(user.ID)
		if err := h.regenerateSummaries(user); err != nil {
			conf.Log().Error("failed to regenerate summaries after settings change", "userID", user.ID, "error", err)
		}
	}(user)

	respondMessage(w, r, http.StatusAccepted, "Saved. Your summaries are being regenerated, which may take a while.")
}

type heartbeatsTimeoutRequest struct {
	Minutes int `json:"minutes"`
}

func (h *SettingsHandler) UpdateHeartbeatsTimeout(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	var payload heartbeatsTimeoutRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	timeout := time.Duration(payload.Minutes) * time.Minute
	if timeout < models.MinHeartbeatsTimeout || timeout > models.MaxHeartbeatsTimeout {
		respondError(w, r, http.StatusBadRequest, "the timeout must be between 1 and 60 minutes")
		return
	}
	user.HeartbeatsTimeoutSec = int(timeout.Seconds())

	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to update heartbeats timeout", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	h.userSrvc.FlushCache()

	respondMessage(w, r, http.StatusOK, "Saved. Regenerate your summaries to apply this to existing data.")
}

type readmeStatsRequest struct {
	BaseUrl string `json:"base_url"`
}

func (h *SettingsHandler) UpdateReadmeStatsBaseUrl(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	var payload readmeStatsRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	user.ReadmeStatsBaseUrl = strings.TrimSpace(payload.BaseUrl)
	if _, err := h.userSrvc.Update(user); err != nil {
		conf.Log().Request(r).Error("failed to update readme stats base url", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}
	h.userSrvc.FlushUserCache(user.ID)

	respondMessage(w, r, http.StatusOK, "Saved.")
}

func (h *SettingsHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	go func(user *models.User) {
		conf.Log().Info("deleting user", "userID", user.ID)
		if err := h.userSrvc.Delete(user); err != nil {
			conf.Log().Error("failed to delete user", "userID", user.ID, "error", err)
			return
		}
		conf.Log().Info("successfully deleted user", "userID", user.ID)
	}(user)

	// The session is cleared here rather than left to the client: once deletion is
	// under way the cookie points at an account that is about to stop existing.
	http.SetCookie(w, h.config.GetClearCookie(models.AuthCookieKey))
	respondMessage(w, r, http.StatusAccepted, "Your account is being deleted. Sorry to see you go.")
}

func (h *SettingsHandler) regenerateSummaries(user *models.User) error {
	conf.Log().Info("clearing summaries and durations for user", "userID", user.ID)

	if err := h.summarySrvc.DeleteByUser(user.ID); err != nil {
		return err
	}
	// This regenerates durations as well.
	return h.aggregationSrvc.AggregateSummaries(datastructure.New(user.ID))
}
