package ui

import (
	"net/http"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
)

// brandingResponse shows all three layers at once: what is in effect, what an admin
// has overridden, and what the config file would fall back to. Without the last two a
// form cannot tell "this is customised" from "this is the default".
type brandingResponse struct {
	Effective models.Branding `json:"effective"`
	Overrides models.Branding `json:"overrides"`
	Defaults  models.Branding `json:"defaults"`
}

func (h *SettingsHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}

	helpers.RespondJSON(w, r, http.StatusOK, &brandingResponse{
		Effective: *h.brandingSrvc.Get(),
		Overrides: *h.brandingSrvc.GetOverrides(),
		Defaults:  h.brandingSrvc.Defaults(),
	})
}

type brandingUpdateResponse struct {
	Message string `json:"message"`
	// Rejected names the fields that failed validation and fell back to the default,
	// so the admin is told rather than left wondering why nothing changed.
	Rejected  []string        `json:"rejected"`
	Effective models.Branding `json:"effective"`
}

func (h *SettingsHandler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}

	var payload models.Branding
	if !decodeBody(w, r, &payload) {
		return
	}

	rejected, err := h.brandingSrvc.Set(&payload)
	if err != nil {
		conf.Log().Request(r).Error("failed to store branding", "error", err)
		respondError(w, r, http.StatusInternalServerError, "could not save the branding")
		return
	}

	message := "Branding saved. Reload to see it everywhere."
	if len(rejected) > 0 {
		message = "Branding saved, but some values were rejected and kept their default."
	}

	helpers.RespondJSON(w, r, http.StatusOK, &brandingUpdateResponse{
		Message:   message,
		Rejected:  rejected,
		Effective: *h.brandingSrvc.Get(),
	})
}

// requireAdmin gates the instance-wide settings. A non-admin gets 403 rather than 404:
// the route's existence is not a secret, only the ability to use it.
func requireAdmin(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := requirePrincipal(w, r)
	if !ok {
		return nil, false
	}
	if !user.IsAdmin {
		respondError(w, r, http.StatusForbidden, "only an administrator can change instance settings")
		return nil, false
	}
	return user, true
}
