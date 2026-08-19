package ui

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	routeutils "github.com/muety/wakapi/routes/utils"
	"github.com/muety/wakapi/services/imports"
	"github.com/muety/wakapi/utils"
)

type wakatimeRequest struct {
	// An empty ApiKey disconnects the relay.
	ApiKey string `json:"api_key"`
	ApiUrl string `json:"api_url"`
}

func (h *SettingsHandler) UpdateWakatimeIntegration(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload wakatimeRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	apiKey := strings.TrimSpace(payload.ApiKey)
	apiUrl := strings.TrimSpace(payload.ApiUrl)
	// Storing the default url would only be noise; empty means "wakatime.com".
	if apiUrl == conf.WakatimeApiUrl || apiKey == "" {
		apiUrl = ""
	}

	// Only check credentials when the relay is being switched on. Re-checking on every
	// save would make an unrelated settings change fail whenever WakaTime is down.
	if user.WakatimeApiKey == "" && apiKey != "" {
		if routeutils.ValidateWakatimeUrl(apiUrl) != nil || !h.validateWakatimeKey(apiKey, apiUrl) {
			respondError(w, r, http.StatusBadRequest, "could not reach WakaTime with those credentials - is the API key or the endpoint URL wrong?")
			return
		}
	}

	if _, err := h.userSrvc.SetWakatimeApiCredentials(user, apiKey, apiUrl); err != nil {
		conf.Log().Request(r).Error("failed to set wakatime credentials", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	if apiKey == "" {
		respondMessage(w, r, http.StatusOK, "Disconnected from WakaTime.")
		return
	}
	respondMessage(w, r, http.StatusOK, "Connected to WakaTime.")
}

func (h *SettingsHandler) validateWakatimeKey(apiKey, baseUrl string) bool {
	if baseUrl == "" {
		baseUrl = conf.WakatimeApiUrl
	}

	request, err := http.NewRequest(http.MethodGet, baseUrl+conf.WakatimeApiUserUrl, nil)
	if err != nil {
		return false
	}
	request.Header = http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(apiKey)))},
	}

	_, err = utils.RaiseForStatus(h.httpClient.Do(request))
	return err == nil
}

type wakatimeImportRequest struct {
	UseLegacyImporter bool `json:"use_legacy_importer"`
}

func (h *SettingsHandler) ImportWakatime(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	if !h.config.App.ImportEnabled {
		respondError(w, r, http.StatusForbidden, "imports are disabled on this server")
		return
	}
	if user.WakatimeApiKey == "" {
		respondError(w, r, http.StatusForbidden, "connect your WakaTime account first")
		return
	}

	var payload wakatimeImportRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	kvKeyLastImport := fmt.Sprintf("%s_%s", conf.KeyLastImport, user.ID)
	kvKeyLastImportSuccess := fmt.Sprintf("%s_%s", conf.KeyLastImportSuccess, user.ID)

	importer := imports.NewWakatimeImporter(user.WakatimeApiKey, payload.UseLegacyImporter)
	if err := importer.Validate(user); err != nil {
		respondError(w, r, http.StatusForbidden, fmt.Sprintf("cannot import: %v", err))
		return
	}

	if !h.config.IsDev() {
		lastImport, _ := time.Parse(time.RFC822, h.keyValueSrvc.MustGetString(kvKeyLastImport).Value)
		if time.Since(lastImport) < time.Duration(h.config.App.ImportBackoffMin)*time.Minute {
			respondError(w, r, http.StatusTooManyRequests,
				fmt.Sprintf("you can only request an import every %d minutes", h.config.App.ImportBackoffMin))
			return
		}

		lastSuccess, _ := time.Parse(time.RFC822, h.keyValueSrvc.MustGetString(kvKeyLastImportSuccess).Value)
		if time.Since(lastSuccess) < time.Duration(h.config.App.ImportMaxRate)*time.Hour {
			respondError(w, r, http.StatusTooManyRequests,
				fmt.Sprintf("your last import ran less than %d hours ago, please wait", h.config.App.ImportMaxRate))
			return
		}
	}

	go h.runWakatimeImport(user, importer, kvKeyLastImportSuccess)

	h.keyValueSrvc.PutString(&models.KeyStringValue{Key: kvKeyLastImport, Value: time.Now().Format(time.RFC822)})
	respondMessage(w, r, http.StatusAccepted, "Import started. This takes several minutes - check back later.")
}

func (h *SettingsHandler) runWakatimeImport(user *models.User, importer *imports.WakatimeImporter, kvKeyLastImportSuccess string) {
	start := time.Now()
	countBefore, _ := h.heartbeatSrvc.CountByUser(user)

	var (
		stream      <-chan *models.Heartbeat
		importError error
	)
	// After a first import, only fetch what is newer than the last imported heartbeat.
	if latest, err := h.heartbeatSrvc.GetLatestByOriginAndUser(imports.OriginWakatime, user); latest == nil || err != nil {
		stream, importError = importer.ImportAll(user)
	} else {
		stream, importError = importer.Import(user, latest.Time.T(), time.Now())
	}
	if importError != nil {
		conf.Log().Error("wakatime import failed", "userID", user.ID, "error", importError)
		return
	}

	h.keyValueSrvc.PutString(&models.KeyStringValue{Key: kvKeyLastImportSuccess, Value: time.Now().Format(time.RFC822)})

	batchSize := h.config.App.ImportBatchSize
	batch := make([]*models.Heartbeat, 0, batchSize)
	insert := func(batch []*models.Heartbeat) {
		if err := h.heartbeatSrvc.InsertBatch(batch); err != nil {
			conf.Log().Warn("failed to insert imported heartbeat, already existing?", "error", err)
		}
	}

	var count int
	for hb := range stream {
		count++
		batch = append(batch, hb)
		if len(batch) == batchSize {
			insert(batch)
			batch = make([]*models.Heartbeat, 0, batchSize)
		}
	}
	if len(batch) > 0 {
		insert(batch)
	}

	countAfter, _ := h.heartbeatSrvc.CountByUser(user)
	conf.Log().Info("downloaded heartbeats for user", "count", count, "userID", user.ID, "importedCount", countAfter-countBefore)

	if err := h.regenerateSummaries(user); err != nil {
		conf.Log().Error("failed to regenerate summaries after import", "userID", user.ID, "error", err)
	}

	if !user.HasData {
		user.HasData = true
		if _, err := h.userSrvc.Update(user); err != nil {
			conf.Log().Error("failed to set has_data flag", "userID", user.ID, "error", err)
		}
	}

	if user.Email != "" {
		if err := h.mailSrvc.SendImportNotification(user, time.Since(start), int(countAfter-countBefore)); err != nil {
			conf.Log().Error("failed to send import notification mail", "userID", user.ID, "error", err)
		}
	}
}

func (h *SettingsHandler) RegenerateSummaries(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	if !h.lockAggregation(user.ID) {
		respondError(w, r, http.StatusConflict, "a summary regeneration is already running, please wait for it to finish")
		return
	}

	go func(user *models.User) {
		defer h.unlockAggregation(user.ID)
		if err := h.regenerateSummaries(user); err != nil {
			conf.Log().Error("failed to regenerate summaries", "userID", user.ID, "error", err)
		}
	}(user)

	respondMessage(w, r, http.StatusAccepted, "Regenerating your summaries. This can take a couple of minutes.")
}

func (h *SettingsHandler) ClearData(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	conf.Log().Request(r).Info("user requested to delete all data", "userID", user.ID)

	go func(user *models.User) {
		if err := h.summarySrvc.DeleteByUser(user.ID); err != nil {
			conf.Log().Error("failed to clear summaries", "userID", user.ID, "error", err)
		}
		if err := h.durationSrvc.DeleteByUser(user); err != nil {
			conf.Log().Error("failed to clear durations", "userID", user.ID, "error", err)
		}
		if err := h.heartbeatSrvc.DeleteByUser(user); err != nil {
			conf.Log().Error("failed to clear heartbeats", "userID", user.ID, "error", err)
		}
	}(user)

	respondMessage(w, r, http.StatusAccepted, "Deleting your data. This may take a few seconds.")
}

type inviteResponse struct {
	Code string `json:"code"`
	Link string `json:"link"`
}

func (h *SettingsHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	if !h.config.Security.InviteCodes {
		respondError(w, r, http.StatusForbidden, "invites are disabled on this server")
		return
	}

	code := uuid.Must(uuid.NewV4()).String()[0:8]
	if err := h.keyValueSrvc.PutString(&models.KeyStringValue{
		Key:   fmt.Sprintf("%s_%s", conf.KeyInviteCode, code),
		Value: fmt.Sprintf("%s,%s", user.ID, time.Now().Format(time.RFC3339)),
	}); err != nil {
		conf.Log().Request(r).Error("failed to generate invite code", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not generate an invite code")
		return
	}

	helpers.RespondJSON(w, r, http.StatusCreated, &inviteResponse{
		Code: code,
		Link: fmt.Sprintf("%s/signup?invite=%s", h.config.Server.GetPublicUrl(), code),
	})
}
