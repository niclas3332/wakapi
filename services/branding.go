package services

import (
	"encoding/json"
	"sync/atomic"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
)

// BrandingService resolves the instance's visual identity from two layers: the config
// file supplies the defaults, the database supplies an admin's overrides.
//
// The resolved value is held in an atomic pointer rather than a TTL cache, because it
// is read on every badge, chart and page render but written only from the admin screen.
// That makes reads free and updates immediate.
type BrandingService struct {
	config   *config.Config
	kvSrvc   IKeyValueService
	resolved atomic.Pointer[models.Branding]
	revision atomic.Uint64
}

// Revision changes whenever the branding does. Renderers that cache their output
// (badges, activity charts) mix it into their cache key, so an admin's change takes
// effect immediately instead of after the cache expires.
func (srv *BrandingService) Revision() uint64 {
	return srv.revision.Load()
}

func NewBrandingService(keyValueService IKeyValueService) *BrandingService {
	srv := &BrandingService{
		config: config.Get(),
		kvSrvc: keyValueService,
	}
	srv.reload()
	return srv
}

// Defaults returns the branding as configured in the config file, ignoring any
// database overrides. This is what an admin sees as the "reset to" value.
func (srv *BrandingService) Defaults() models.Branding {
	c := srv.config.Branding
	return models.Branding{
		InstanceName:          c.InstanceName,
		BadgeLabel:            c.BadgeLabel,
		BadgeColor:            c.BadgeColor,
		ReadmeCardTitle:       c.ReadmeCardTitle,
		LogoUrl:               c.LogoUrl,
		FooterText:            c.FooterText,
		FooterUrl:             c.FooterUrl,
		ActivityTitle:         c.ActivityTitle,
		ActivityLogoUrl:       c.ActivityLogoUrl,
		ActivityColorMinDark:  c.ActivityColorMinDark,
		ActivityColorMaxDark:  c.ActivityColorMaxDark,
		ActivityColorMinLight: c.ActivityColorMinLight,
		ActivityColorMaxLight: c.ActivityColorMaxLight,
	}
}

// Get returns the effective branding. It never fails and never returns nil - a
// misconfigured instance still has to render.
func (srv *BrandingService) Get() *models.Branding {
	if b := srv.resolved.Load(); b != nil {
		return b
	}
	defaults, _ := srv.Defaults().Sanitized()
	return &defaults
}

// GetOverrides returns only what an admin has stored, so the settings form can show
// which fields are actually customised rather than presenting defaults as edits.
func (srv *BrandingService) GetOverrides() *models.Branding {
	overrides := &models.Branding{}
	kv, err := srv.kvSrvc.GetString(models.BrandingKey)
	if err != nil || kv == nil || kv.Value == "" {
		return overrides
	}
	if err := json.Unmarshal([]byte(kv.Value), overrides); err != nil {
		config.Log().Error("failed to parse stored branding overrides", "error", err)
		return &models.Branding{}
	}
	return overrides
}

// Set stores the given overrides and applies them immediately. Rejected fields are
// reported back rather than silently dropped, so the admin learns what did not stick.
func (srv *BrandingService) Set(overrides *models.Branding) ([]string, error) {
	_, rejected := srv.Defaults().WithOverrides(overrides).Sanitized()

	payload, err := json.Marshal(overrides)
	if err != nil {
		return rejected, err
	}
	if err := srv.kvSrvc.PutString(&models.KeyStringValue{
		Key:   models.BrandingKey,
		Value: string(payload),
	}); err != nil {
		return rejected, err
	}

	srv.reload()
	return rejected, nil
}

func (srv *BrandingService) reload() {
	effective, rejected := srv.Defaults().WithOverrides(srv.GetOverrides()).Sanitized()
	if len(rejected) > 0 {
		config.Log().Warn("ignoring invalid branding overrides, falling back to defaults", "fields", rejected)
	}
	srv.resolved.Store(&effective)
	srv.revision.Add(1)
}
