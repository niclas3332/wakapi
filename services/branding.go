package services

import (
	"encoding/json"
	"strings"
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

// configured reads the config file's branding verbatim.
func (srv *BrandingService) configured() models.Branding {
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

// derive fills the name-bearing fields that were left empty from the instance name.
//
// This runs after overrides are merged, not before, so renaming the instance in the
// admin screen renames the badge, the readme card and the chart with it. A fork sets
// its identity in one place instead of renaming each surface away from an upstream
// domain it has nothing to do with.
func derive(b models.Branding) models.Branding {
	name := strings.TrimSpace(b.InstanceName)
	if name == "" {
		name = "Wakapi"
	}
	b.InstanceName = name

	if strings.TrimSpace(b.BadgeLabel) == "" {
		b.BadgeLabel = strings.ToLower(name)
	}
	if strings.TrimSpace(b.ReadmeCardTitle) == "" {
		b.ReadmeCardTitle = name + " Stats"
	}
	if strings.TrimSpace(b.ActivityTitle) == "" {
		b.ActivityTitle = name
	}
	return b
}

// Defaults is what an admin sees as the "reset to" value: the config file's branding
// with the derived names filled in, but without any stored override.
func (srv *BrandingService) Defaults() models.Branding {
	defaults, _ := derive(srv.configured()).Sanitized()
	return defaults
}

// Get returns the effective branding. It never fails and never returns nil - a
// misconfigured instance still has to render.
func (srv *BrandingService) Get() *models.Branding {
	if b := srv.resolved.Load(); b != nil {
		return b
	}
	defaults := srv.Defaults()
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
	_, rejected := srv.resolve(overrides)

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

// resolve applies the given overrides on top of the configured branding, derives the
// names that were left blank and validates the result.
func (srv *BrandingService) resolve(overrides *models.Branding) (models.Branding, []string) {
	return derive(srv.configured().WithOverrides(overrides)).Sanitized()
}

func (srv *BrandingService) reload() {
	effective, rejected := srv.resolve(srv.GetOverrides())
	if len(rejected) > 0 {
		config.Log().Warn("ignoring invalid branding overrides, falling back to defaults", "fields", rejected)
	}
	srv.resolved.Store(&effective)
	srv.revision.Add(1)
}
