package ui

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	routeutils "github.com/muety/wakapi/routes/utils"
)

type webauthnCredentialDTO struct {
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func (h *SettingsHandler) loadWebAuthnCredentials(user *models.User) ([]webauthnCredentialDTO, error) {
	if h.config.Security.DisableWebAuthn {
		return []webauthnCredentialDTO{}, nil
	}

	credentials, err := h.webAuthnSrvc.GetCredentialsByUser(user)
	if err != nil {
		return nil, err
	}

	out := make([]webauthnCredentialDTO, 0, len(credentials))
	for _, credential := range credentials {
		dto := webauthnCredentialDTO{Name: credential.Name, CreatedAt: credential.CreatedAt}
		if credential.LastUsedAt.Valid {
			t := credential.LastUsedAt.Time
			dto.LastUsedAt = &t
		}
		out = append(out, dto)
	}
	return out, nil
}

func (h *SettingsHandler) GetWebAuthnCredentials(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	credentials, err := h.loadWebAuthnCredentials(user)
	if err != nil {
		conf.Log().Request(r).Error("failed to load webauthn credentials", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load passkeys")
		return
	}
	helpers.RespondJSON(w, r, http.StatusOK, credentials)
}

// GetWebAuthnRegistrationOptions starts a registration ceremony. The challenge is kept
// server-side in a session, so the browser cannot choose it.
func (h *SettingsHandler) GetWebAuthnRegistrationOptions(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	if !h.checkWebAuthnAllowed(w, r, user) {
		return
	}

	if err := h.webAuthnSrvc.LoadCredentialIntoUser(user); err != nil {
		conf.Log().Request(r).Error("failed to load webauthn credentials", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load passkeys")
		return
	}

	options, session, err := conf.WebAuthn.BeginRegistration(user, webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
		RequireResidentKey: protocol.ResidentKeyRequired(), // for usernameless login
	}))
	if err != nil {
		conf.Log().Request(r).Error("failed to begin webauthn registration", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not start passkey registration")
		return
	}

	if err := routeutils.SetWebAuthnSession(session, r, w); err != nil {
		conf.Log().Request(r).Error("failed to store webauthn session", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not start passkey registration")
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, options)
}

type webauthnCredentialRequest struct {
	Name string `json:"name"`
	// Credential is the raw PublicKeyCredential the browser produced, forwarded verbatim.
	Credential string `json:"credential"`
}

func (h *SettingsHandler) CreateWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	if !h.checkWebAuthnAllowed(w, r, user) {
		return
	}

	var payload webauthnCredentialRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		respondError(w, r, http.StatusBadRequest, "give the passkey a name")
		return
	}

	if err := h.webAuthnSrvc.LoadCredentialIntoUser(user); err != nil {
		conf.Log().Request(r).Error("failed to load webauthn credentials", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "failed to load passkeys")
		return
	}
	for _, c := range user.Credentials {
		if c.Name == name {
			respondError(w, r, http.StatusConflict, "you already have a passkey with that name")
			return
		}
	}

	sessionData, err := routeutils.GetWebAuthnSession(r)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "your passkey registration expired, please try again")
		return
	}

	// FinishRegistration cannot be used here: the client sends only the credential, not
	// the whole request wrapper the library expects.
	pcc, err := protocol.ParseCredentialCreationResponseBytes([]byte(payload.Credential))
	if err != nil {
		conf.Log().Request(r).Error("failed to parse webauthn registration response", "error", err)
		respondError(w, r, http.StatusBadRequest, "could not read the passkey your browser produced")
		return
	}

	credential, err := conf.WebAuthn.CreateCredential(user, *sessionData, pcc)
	if err != nil {
		conf.Log().Request(r).Error("failed to create webauthn credential", "error", err)
		respondError(w, r, http.StatusBadRequest, "could not verify the passkey")
		return
	}

	if _, err := h.webAuthnSrvc.CreateCredential(credential, user, name); err != nil {
		conf.Log().Request(r).Error("failed to store webauthn credential", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not store the passkey")
		return
	}

	respondMessage(w, r, http.StatusCreated, "Passkey added.")
}

func (h *SettingsHandler) DeleteWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return
	}
	if !h.checkWebAuthnAllowed(w, r, user) {
		return
	}

	name := chi.URLParam(r, "name")
	credential, err := h.webAuthnSrvc.GetCredentialByUserAndName(user, name)
	if err != nil || credential == nil {
		respondError(w, r, http.StatusNotFound, "no such passkey")
		return
	}

	if err := h.webAuthnSrvc.DeleteCredential(credential); err != nil {
		conf.Log().Request(r).Error("failed to delete webauthn credential", "userID", user.ID, "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not delete that passkey")
		return
	}

	respondMessage(w, r, http.StatusOK, "Passkey removed.")
}

// checkWebAuthnAllowed answers the two conditions that rule passkeys out entirely, so
// each handler does not repeat them.
func (h *SettingsHandler) checkWebAuthnAllowed(w http.ResponseWriter, r *http.Request, user *models.User) bool {
	if h.config.Security.DisableWebAuthn {
		respondError(w, r, http.StatusForbidden, "passkeys are disabled on this server")
		return false
	}
	if user.AuthType != "local" {
		respondError(w, r, http.StatusBadRequest, "passkeys are only available for local accounts")
		return false
	}
	return true
}
