package ui

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid/v5"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	routeutils "github.com/muety/wakapi/routes/utils"
)

// aliasGroup collects every original name that maps onto one alias, which is how the
// screen presents them: one row per alias, not one row per mapping.
type aliasGroup struct {
	Key    string   `json:"key"`
	Type   uint8    `json:"type"`
	Values []string `json:"values"`
}

func (h *SettingsHandler) loadAliasGroups(userId string) ([]aliasGroup, error) {
	aliases, err := h.aliasSrvc.GetByUser(userId)
	if err != nil {
		return nil, err
	}

	grouped := map[string]*aliasGroup{}
	for _, alias := range aliases {
		key := alias.Key + "\x00" + strconv.Itoa(int(alias.Type))
		if _, ok := grouped[key]; !ok {
			grouped[key] = &aliasGroup{Key: alias.Key, Type: alias.Type, Values: []string{}}
		}
		grouped[key].Values = append(grouped[key].Values, alias.Value)
	}

	groups := make([]aliasGroup, 0, len(grouped))
	for _, group := range grouped {
		sort.Strings(group.Values)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Type != groups[j].Type {
			return groups[i].Type < groups[j].Type
		}
		return groups[i].Key < groups[j].Key
	})
	return groups, nil
}

func (h *SettingsHandler) GetAliases(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	groups, err := h.loadAliasGroups(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load aliases", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load aliases")
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, groups)
}

type aliasRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  uint8  `json:"type"`
}

func (h *SettingsHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload aliasRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	alias := &models.Alias{
		UserID: user.ID,
		Key:    strings.TrimSpace(payload.Key),
		Value:  strings.TrimSpace(payload.Value),
		Type:   payload.Type,
	}
	if !alias.IsValid() {
		// The wildcard rule is the non-obvious one, so it is named rather than implied.
		respondError(w, r, http.StatusBadRequest, "invalid alias - both fields are required, and a wildcard pattern needs at least 3 other characters")
		return
	}

	if _, err := h.aliasSrvc.Create(alias); err != nil {
		conf.Log().Request(r).Error("failed to create alias", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusConflict, "could not create that alias - does it already exist?")
		return
	}

	respondMessage(w, r, http.StatusCreated, "Alias added.")
}

// DeleteAlias removes every mapping onto one alias at once, which is what the row in
// the ui represents.
func (h *SettingsHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	key := r.URL.Query().Get("key")
	aliasType, err := strconv.Atoi(r.URL.Query().Get("type"))
	if key == "" || err != nil {
		respondError(w, r, http.StatusBadRequest, "key and type are required")
		return
	}

	aliases, err := h.aliasSrvc.GetByUserAndKeyAndType(user.ID, key, uint8(aliasType))
	if err != nil || len(aliases) == 0 {
		respondError(w, r, http.StatusNotFound, "no such alias")
		return
	}
	if err := h.aliasSrvc.DeleteMulti(aliases); err != nil {
		conf.Log().Request(r).Error("failed to delete aliases", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not delete that alias")
		return
	}

	respondMessage(w, r, http.StatusOK, "Alias deleted.")
}

// labelGroup is one label with every project carrying it.
type labelGroup struct {
	Label    string   `json:"label"`
	Projects []string `json:"projects"`
}

func (h *SettingsHandler) loadLabelGroups(userId string) ([]labelGroup, error) {
	labelMap, err := h.projectLabelSrvc.GetByUserGroupedInverted(userId)
	if err != nil {
		return nil, err
	}

	groups := make([]labelGroup, 0, len(labelMap))
	for label, labels := range labelMap {
		projects := make([]string, 0, len(labels))
		for _, l := range labels {
			projects = append(projects, l.ProjectKey)
		}
		sort.Strings(projects)
		groups = append(groups, labelGroup{Label: label, Projects: projects})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Label < groups[j].Label })
	return groups, nil
}

func (h *SettingsHandler) GetLabels(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	groups, err := h.loadLabelGroups(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load labels", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load labels")
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, groups)
}

type labelRequest struct {
	Label    string   `json:"label"`
	Projects []string `json:"projects"`
}

// CreateLabels assigns one label to several projects in a single call - the bulk
// assignment the legacy screen offered.
func (h *SettingsHandler) CreateLabels(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload labelRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	label := strings.TrimSpace(payload.Label)
	if label == "" || len(payload.Projects) == 0 {
		respondError(w, r, http.StatusBadRequest, "a label and at least one project are required")
		return
	}

	for _, project := range payload.Projects {
		entry := &models.ProjectLabel{UserID: user.ID, ProjectKey: strings.TrimSpace(project), Label: label}
		if !entry.IsValid() {
			respondError(w, r, http.StatusBadRequest, "invalid project: "+project)
			return
		}
		if _, err := h.projectLabelSrvc.Create(entry); err != nil {
			conf.Log().Request(r).Error("failed to create project label", "userID", user.ID, "project", project, "error", err)
			respondError(w, r, http.StatusConflict, "could not label "+project+" - is it labelled already?")
			return
		}
	}

	respondMessage(w, r, http.StatusCreated, "Label assigned.")
}

func (h *SettingsHandler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	label := r.URL.Query().Get("label")
	project := r.URL.Query().Get("project")
	if label == "" || project == "" {
		respondError(w, r, http.StatusBadRequest, "label and project are required")
		return
	}

	labels, err := h.projectLabelSrvc.GetByUser(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load labels for deletion", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not delete that label")
		return
	}

	for _, l := range labels {
		if l.Label == label && l.ProjectKey == project {
			if err := h.projectLabelSrvc.Delete(l); err != nil {
				conf.Log().Request(r).Error("failed to delete label", "userID", user.ID, "error", err)
				respondError(w, r, http.StatusInternalServerError, "could not delete that label")
				return
			}
			respondMessage(w, r, http.StatusOK, "Label removed.")
			return
		}
	}
	respondError(w, r, http.StatusNotFound, "no such label")
}

type languageMappingDTO struct {
	ID        uint   `json:"id"`
	Extension string `json:"extension"`
	Language  string `json:"language"`
}

func (h *SettingsHandler) loadLanguageMappings(userId string) ([]languageMappingDTO, error) {
	mappings, err := h.languageMappingSrvc.GetByUser(userId)
	if err != nil {
		return nil, err
	}
	out := make([]languageMappingDTO, 0, len(mappings))
	for _, mapping := range mappings {
		out = append(out, languageMappingDTO{ID: mapping.ID, Extension: mapping.Extension, Language: mapping.Language})
	}
	return out, nil
}

func (h *SettingsHandler) GetLanguageMappings(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	mappings, err := h.loadLanguageMappings(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load language mappings", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load language mappings")
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, mappings)
}

type languageMappingRequest struct {
	Extension string `json:"extension"`
	Language  string `json:"language"`
}

func (h *SettingsHandler) CreateLanguageMapping(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload languageMappingRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	// People type ".rb" as readily as "rb"; both mean the same mapping.
	extension := strings.TrimPrefix(strings.TrimSpace(payload.Extension), ".")
	mapping := &models.LanguageMapping{
		UserID:    user.ID,
		Extension: extension,
		Language:  strings.TrimSpace(payload.Language),
	}
	if !mapping.IsValid() {
		respondError(w, r, http.StatusBadRequest, "both an extension and a language are required")
		return
	}

	if _, err := h.languageMappingSrvc.Create(mapping); err != nil {
		respondError(w, r, http.StatusConflict, "a mapping for that extension already exists")
		return
	}

	respondMessage(w, r, http.StatusCreated, "Mapping added.")
}

func (h *SettingsHandler) DeleteLanguageMapping(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid mapping id")
		return
	}

	mapping, err := h.languageMappingSrvc.GetById(uint(id))
	if err != nil || mapping == nil {
		respondError(w, r, http.StatusNotFound, "no such mapping")
		return
	}
	if mapping.UserID != user.ID {
		// Deliberately the same answer as "not found": whether someone else's mapping
		// exists is not this user's business.
		respondError(w, r, http.StatusNotFound, "no such mapping")
		return
	}

	if err := h.languageMappingSrvc.Delete(mapping); err != nil {
		conf.Log().Request(r).Error("failed to delete language mapping", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not delete that mapping")
		return
	}

	respondMessage(w, r, http.StatusOK, "Mapping deleted.")
}

type apiKeyDTO struct {
	Label    string `json:"label"`
	ApiKey   string `json:"api_key"`
	ReadOnly bool   `json:"read_only"`
	// Primary marks the account's main key, which can be regenerated but not deleted.
	Primary bool `json:"primary"`
}

func (h *SettingsHandler) loadApiKeys(user *models.User) ([]apiKeyDTO, error) {
	keys := []apiKeyDTO{{Label: "Main API Key", ApiKey: user.ApiKey, Primary: true}}

	additional, err := h.apiKeySrvc.GetByUser(user.ID)
	if err != nil {
		return nil, err
	}
	for _, key := range additional {
		keys = append(keys, apiKeyDTO{Label: key.Label, ApiKey: key.ApiKey, ReadOnly: key.ReadOnly})
	}
	return keys, nil
}

func (h *SettingsHandler) GetApiKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	keys, err := h.loadApiKeys(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load api keys", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load api keys")
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, keys)
}

type apiKeyRequest struct {
	Label    string `json:"label"`
	ReadOnly bool   `json:"read_only"`
}

func (h *SettingsHandler) CreateApiKey(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	var payload apiKeyRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	label := strings.TrimSpace(payload.Label)
	if label == "" {
		respondError(w, r, http.StatusBadRequest, "give the key a name so you can tell it apart later")
		return
	}

	key := &models.ApiKey{
		User:     user,
		Label:    label,
		ApiKey:   uuid.Must(uuid.NewV4()).String(),
		ReadOnly: payload.ReadOnly,
	}
	if _, err := h.apiKeySrvc.Create(key); err != nil {
		conf.Log().Request(r).Error("failed to create api key", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, conf.ErrInternalServerError)
		return
	}

	helpers.RespondJSON(w, r, http.StatusCreated, &apiKeyDTO{
		Label:    key.Label,
		ApiKey:   key.ApiKey,
		ReadOnly: key.ReadOnly,
	})
}

func (h *SettingsHandler) DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}

	value := chi.URLParam(r, "key")
	if value == user.ApiKey {
		respondError(w, r, http.StatusBadRequest, "the main API key can only be regenerated, not deleted")
		return
	}

	keys, err := h.apiKeySrvc.GetByUser(user.ID)
	if err != nil {
		conf.Log().Request(r).Error("failed to load api keys for deletion", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not delete that key")
		return
	}

	for _, key := range keys {
		if key.ApiKey == value {
			if err := h.apiKeySrvc.Delete(key); err != nil {
				conf.Log().Request(r).Error("failed to delete api key", "userID", user.ID, "error", err)
				respondError(w, r, http.StatusInternalServerError, "could not delete that key")
				return
			}
			respondMessage(w, r, http.StatusOK, "API key deleted.")
			return
		}
	}
	respondError(w, r, http.StatusNotFound, "no such API key")
}

func (h *SettingsHandler) loadProjectNames(user *models.User) ([]string, error) {
	projects, err := routeutils.GetEffectiveProjectsList(user, h.heartbeatSrvc, h.aliasSrvc)
	if err != nil {
		return nil, err
	}
	if projects == nil {
		projects = []string{}
	}
	return projects, nil
}
