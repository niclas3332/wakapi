package ui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/mocks"
	"github.com/muety/wakapi/models"
	routeutils "github.com/muety/wakapi/routes/utils"
)

// newJsonRequest builds an authenticated request with a JSON body, mirroring what the
// frontend sends.
func newJsonRequest(method, target, body string, user *models.User) *http.Request {
	r := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), config.KeySharedData, config.NewSharedData()))
	if user != nil {
		routeutils.SetPrincipal(r, user)
	}
	return r
}

// withUrlParam attaches a chi route parameter, which handlers read via chi.URLParam.
func withUrlParam(r *http.Request, key, value string) *http.Request {
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

func settingsHandler() *SettingsHandler {
	return &SettingsHandler{config: config.Get()}
}

// A misspelled field must not be swallowed: silently ignoring it would report a
// setting as saved when the server never read it.
func TestSettings_UnknownFieldIsRejected(t *testing.T) {
	sut := settingsHandler()
	rec := httptest.NewRecorder()

	sut.UpdateSharing(rec, newJsonRequest(http.MethodPut, "/api/ui/user/sharing", `{"projekts":true}`, testUser()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSettings_MutationsRequirePrincipal(t *testing.T) {
	sut := settingsHandler()

	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"profile":  sut.UpdateProfile,
		"password": sut.ChangePassword,
		"sharing":  sut.UpdateSharing,
		"invite":   sut.CreateInvite,
		"branding": sut.UpdateBranding,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			call(rec, newJsonRequest(http.MethodPost, "/api/ui/x", `{}`, nil))
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestSettings_HeartbeatsTimeoutIsBounded(t *testing.T) {
	userMock := new(mocks.UserServiceMock)
	userMock.On("FlushCache").Return()
	sut := &SettingsHandler{config: config.Get(), userSrvc: userMock}

	for _, minutes := range []int{0, 61, -5} {
		rec := httptest.NewRecorder()
		body := `{"minutes":` + strconv.Itoa(minutes) + `}`
		sut.UpdateHeartbeatsTimeout(rec, newJsonRequest(http.MethodPut, "/api/ui/user/heartbeats-timeout", body, testUser()))
		assert.Equal(t, http.StatusBadRequest, rec.Code, "minutes=%d must be rejected", minutes)
	}

	// A value inside the range is stored as seconds.
	user := testUser()
	userMock.On("Update", user).Return(user, nil)
	rec := httptest.NewRecorder()
	sut.UpdateHeartbeatsTimeout(rec, newJsonRequest(http.MethodPut, "/api/ui/user/heartbeats-timeout", `{"minutes":15}`, user))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 900, user.HeartbeatsTimeoutSec)
}

// The main api key is the account's identity for every editor plugin, so it may be
// regenerated but never deleted - deleting it would lock every plugin out silently.
func TestSettings_MainApiKeyCannotBeDeleted(t *testing.T) {
	user := testUser()
	sut := settingsHandler()

	rec := httptest.NewRecorder()
	r := withUrlParam(newJsonRequest(http.MethodDelete, "/api/ui/api-keys/key", "", user), "key", user.ApiKey)
	r = r.WithContext(context.WithValue(r.Context(), config.KeySharedData, config.NewSharedData()))
	routeutils.SetPrincipal(r, user)
	sut.DeleteApiKey(rec, r)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "regenerated")
}

// Someone else's language mapping is answered with 404 rather than 403: whether it
// exists is not this user's business.
func TestSettings_ForeignLanguageMappingIsNotFound(t *testing.T) {
	mappingMock := new(mocks.LanguageMappingServiceMock)
	mappingMock.On("GetById", uint(7)).Return(&models.LanguageMapping{ID: 7, UserID: "bob"}, nil)

	sut := &SettingsHandler{config: config.Get(), languageMappingSrvc: mappingMock}
	rec := httptest.NewRecorder()

	r := withUrlParam(newJsonRequest(http.MethodDelete, "/api/ui/language-mappings/7", "", nil), "id", "7")
	r = r.WithContext(context.WithValue(r.Context(), config.KeySharedData, config.NewSharedData()))
	routeutils.SetPrincipal(r, testUser())
	sut.DeleteLanguageMapping(rec, r)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	mappingMock.AssertNotCalled(t, "Delete", mock.Anything)
}

func TestSettings_BrandingRequiresAdmin(t *testing.T) {
	sut := settingsHandler()

	rec := httptest.NewRecorder()
	sut.GetBranding(rec, newJsonRequest(http.MethodGet, "/api/ui/admin/branding", "", testUser()))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	admin := testUser()
	admin.IsAdmin = true
	branding := &models.Branding{InstanceName: "Wakapi"}
	sut.brandingSrvc = mocks.NewStaticBrandingServiceMock(branding)

	rec = httptest.NewRecorder()
	sut.GetBranding(rec, newJsonRequest(http.MethodGet, "/api/ui/admin/branding", "", admin))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Wakapi", decode[brandingResponse](t, rec).Effective.InstanceName)
}

// Bulk labelling is the behaviour the legacy screen had: one label, many projects.
func TestSettings_LabelsAreAssignedInBulk(t *testing.T) {
	user := testUser()
	labelMock := new(mocks.ProjectLabelServiceMock)
	labelMock.On("Create", mock.AnythingOfType("*models.ProjectLabel")).Return(&models.ProjectLabel{}, nil)

	sut := &SettingsHandler{config: config.Get(), projectLabelSrvc: labelMock}
	rec := httptest.NewRecorder()
	sut.CreateLabels(rec, newJsonRequest(http.MethodPost, "/api/ui/labels", `{"label":"work","projects":["a","b","c"]}`, user))

	assert.Equal(t, http.StatusCreated, rec.Code)
	labelMock.AssertNumberOfCalls(t, "Create", 3)
}

func TestSettings_LabelNeedsAtLeastOneProject(t *testing.T) {
	sut := settingsHandler()
	rec := httptest.NewRecorder()
	sut.CreateLabels(rec, newJsonRequest(http.MethodPost, "/api/ui/labels", `{"label":"work","projects":[]}`, testUser()))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// A leading dot is how people write extensions, and means the same mapping.
func TestSettings_LanguageMappingExtensionIsNormalized(t *testing.T) {
	var captured *models.LanguageMapping
	mappingMock := new(mocks.LanguageMappingServiceMock)
	mappingMock.On("Create", mock.AnythingOfType("*models.LanguageMapping")).
		Run(func(args mock.Arguments) { captured = args.Get(0).(*models.LanguageMapping) }).
		Return(&models.LanguageMapping{}, nil)

	sut := &SettingsHandler{config: config.Get(), languageMappingSrvc: mappingMock}
	rec := httptest.NewRecorder()
	sut.CreateLanguageMapping(rec, newJsonRequest(http.MethodPost, "/api/ui/language-mappings", `{"extension":".rb","language":"Ruby"}`, testUser()))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "rb", captured.Extension)
}

// Only one regeneration per user may run at a time; a second request is a conflict,
// not a silent no-op.
func TestSettings_RegenerationIsLockedPerUser(t *testing.T) {
	sut := settingsHandler()
	user := testUser()

	assert.True(t, sut.lockAggregation(user.ID))

	rec := httptest.NewRecorder()
	sut.RegenerateSummaries(rec, newJsonRequest(http.MethodPost, "/api/ui/data/regenerate", "", user))
	assert.Equal(t, http.StatusConflict, rec.Code)

	sut.unlockAggregation(user.ID)
	// A different user is unaffected by another user's lock.
	assert.True(t, sut.lockAggregation("bob"))
}
