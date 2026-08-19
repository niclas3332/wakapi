package ui

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/services"
)

// SettingsHandler serves everything the settings screen reads and writes.
//
// It is one handler rather than eight because the legacy screen was one form with an
// "action" field, and keeping the split identical would only invent boundaries that
// nothing else respects. What changed is the shape: each former action is now its own
// route with its own method, so a client can tell "not allowed" from "does not exist"
// from "conflicting" without reading prose out of an html page.
type SettingsHandler struct {
	config              *conf.Config
	userSrvc            services.IUserService
	heartbeatSrvc       services.IHeartbeatService
	durationSrvc        services.IDurationService
	summarySrvc         services.ISummaryService
	aliasSrvc           services.IAliasService
	aggregationSrvc     services.IAggregationService
	languageMappingSrvc services.ILanguageMappingService
	projectLabelSrvc    services.IProjectLabelService
	keyValueSrvc        services.IKeyValueService
	mailSrvc            services.IMailService
	apiKeySrvc          services.IApiKeyService
	webAuthnSrvc        services.IWebAuthnService
	brandingSrvc        services.IBrandingService
	httpClient          *http.Client

	// aggregationLocks keeps one user from starting a second regeneration while the
	// first is still running. The legacy handler used a plain map written from
	// goroutines, which is a data race; sync.Map is the same guard without it.
	aggregationLocks sync.Map
}

func NewSettingsHandler(
	userService services.IUserService,
	heartbeatService services.IHeartbeatService,
	durationService services.IDurationService,
	summaryService services.ISummaryService,
	aliasService services.IAliasService,
	aggregationService services.IAggregationService,
	languageMappingService services.ILanguageMappingService,
	projectLabelService services.IProjectLabelService,
	keyValueService services.IKeyValueService,
	mailService services.IMailService,
	apiKeyService services.IApiKeyService,
	webAuthnService services.IWebAuthnService,
	brandingService services.IBrandingService,
) *SettingsHandler {
	return &SettingsHandler{
		config:              conf.Get(),
		userSrvc:            userService,
		heartbeatSrvc:       heartbeatService,
		durationSrvc:        durationService,
		summarySrvc:         summaryService,
		aliasSrvc:           aliasService,
		aggregationSrvc:     aggregationService,
		languageMappingSrvc: languageMappingService,
		projectLabelSrvc:    projectLabelService,
		keyValueSrvc:        keyValueService,
		mailSrvc:            mailService,
		apiKeySrvc:          apiKeyService,
		webAuthnSrvc:        webAuthnService,
		brandingSrvc:        brandingService,
		httpClient:          &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *SettingsHandler) RegisterRoutes(router chi.Router) {
	router.Get("/settings", h.GetSettings)

	router.Patch("/user/profile", h.UpdateProfile)
	router.Post("/user/password", h.ChangePassword)
	router.Post("/user/username", h.ChangeUsername)
	router.Post("/user/api-key/reset", h.ResetApiKey)
	router.Put("/user/sharing", h.UpdateSharing)
	router.Put("/user/leaderboard-opt-in", h.UpdateLeaderboardOptIn)
	router.Put("/user/exclude-unknown-projects", h.UpdateExcludeUnknownProjects)
	router.Put("/user/heartbeats-timeout", h.UpdateHeartbeatsTimeout)
	router.Put("/user/readme-stats-base-url", h.UpdateReadmeStatsBaseUrl)
	router.Delete("/user", h.DeleteAccount)

	router.Get("/aliases", h.GetAliases)
	router.Post("/aliases", h.CreateAlias)
	router.Delete("/aliases", h.DeleteAlias)

	router.Get("/labels", h.GetLabels)
	router.Post("/labels", h.CreateLabels)
	router.Delete("/labels", h.DeleteLabel)

	router.Get("/language-mappings", h.GetLanguageMappings)
	router.Post("/language-mappings", h.CreateLanguageMapping)
	router.Delete("/language-mappings/{id}", h.DeleteLanguageMapping)

	router.Get("/api-keys", h.GetApiKeys)
	router.Post("/api-keys", h.CreateApiKey)
	router.Delete("/api-keys/{key}", h.DeleteApiKey)

	router.Put("/integrations/wakatime", h.UpdateWakatimeIntegration)
	router.Post("/integrations/wakatime/import", h.ImportWakatime)

	router.Post("/data/regenerate", h.RegenerateSummaries)
	router.Post("/data/clear", h.ClearData)

	router.Post("/invites", h.CreateInvite)

	router.Get("/webauthn/registration-options", h.GetWebAuthnRegistrationOptions)
	router.Get("/webauthn/credentials", h.GetWebAuthnCredentials)
	router.Post("/webauthn/credentials", h.CreateWebAuthnCredential)
	router.Delete("/webauthn/credentials/{name}", h.DeleteWebAuthnCredential)

	router.Get("/admin/branding", h.GetBranding)
	router.Put("/admin/branding", h.UpdateBranding)
}

// decodeBody reads a JSON request body, answering with 400 on malformed input.
// Unknown fields are rejected: silently ignoring a misspelled field would make a
// setting look saved when it was never read.
func decodeBody[T any](w http.ResponseWriter, r *http.Request, target *T) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body: "+err.Error())
		return false
	}
	return true
}

// messageResponse is what every mutation answers with. The frontend shows it verbatim,
// so the text is written for a person, not for a log.
type messageResponse struct {
	Message string `json:"message"`
}

func respondMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	helpers.RespondJSON(w, r, status, &messageResponse{Message: message})
}

func (h *SettingsHandler) lockAggregation(userId string) bool {
	_, running := h.aggregationLocks.LoadOrStore(userId, true)
	return !running
}

func (h *SettingsHandler) unlockAggregation(userId string) {
	h.aggregationLocks.Delete(userId)
}

// settingsResponse is the single payload the settings screen boots from. One request
// instead of eight, because every tab needs its data the moment the screen opens.
type settingsResponse struct {
	User             *userResponse           `json:"user"`
	Aliases          []aliasGroup            `json:"aliases"`
	Labels           []labelGroup            `json:"labels"`
	LanguageMappings []languageMappingDTO    `json:"language_mappings"`
	ApiKeys          []apiKeyDTO             `json:"api_keys"`
	Projects         []string                `json:"projects"`
	WebauthnCreds    []webauthnCredentialDTO `json:"webauthn_credentials"`

	// ReadmeCardTitle is branded per instance and already carries the sharing window,
	// so the client does not have to reconstruct it.
	ReadmeCardTitle   string `json:"readme_card_title"`
	SubscriptionPrice string `json:"subscription_price"`
	SupportContact    string `json:"support_contact"`
	WakatimeApiUrl    string `json:"wakatime_api_url"`
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	aliases, err := h.loadAliasGroups(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load aliases", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load aliases")
		return
	}

	labels, err := h.loadLabelGroups(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load labels", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load labels")
		return
	}

	mappings, err := h.loadLanguageMappings(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load language mappings", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load language mappings")
		return
	}

	apiKeys, err := h.loadApiKeys(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load api keys", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load api keys")
		return
	}

	projects, err := h.loadProjectNames(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load project names", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load projects")
		return
	}

	credentials, err := h.loadWebAuthnCredentials(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load webauthn credentials", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load passkeys")
		return
	}

	var subscriptionPrice string
	if h.config.Subscriptions.Enabled {
		subscriptionPrice = h.config.Subscriptions.StandardPrice
	}

	helpers.RespondJSON(w, r, http.StatusOK, &settingsResponse{
		User:              newUserResponse(user),
		Aliases:           aliases,
		Labels:            labels,
		LanguageMappings:  mappings,
		ApiKeys:           apiKeys,
		Projects:          projects,
		WebauthnCreds:     credentials,
		ReadmeCardTitle:   h.readmeCardTitle(user),
		SubscriptionPrice: subscriptionPrice,
		SupportContact:    h.config.App.SupportContact,
		WakatimeApiUrl:    conf.WakatimeApiUrl,
	})
}

// readmeCardTitle mirrors what the legacy settings page built: the instance's branded
// title plus the range the user actually shares, since the card cannot show more.
func (h *SettingsHandler) readmeCardTitle(user *models.User) string {
	title := h.brandingSrvc.Get().ReadmeCardTitle
	if err, maxRange := helpers.ResolveMaximumRange(user.ShareDataMaxDays); err == nil {
		title += " (" + maxRange.GetHumanReadable() + ")"
	}
	return title
}
