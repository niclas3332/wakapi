package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/models"
)

func brandingServiceWith(apply func(*config.Config)) *BrandingService {
	cfg := config.Empty()
	cfg.Branding.InstanceName = "Wakapi"
	cfg.Branding.BadgeColor = "2F855A"
	cfg.Branding.ActivityColorMinDark = "242B3A"
	cfg.Branding.ActivityColorMaxDark = "047857"
	cfg.Branding.ActivityColorMinLight = "DCE3E1"
	cfg.Branding.ActivityColorMaxLight = "047857"
	apply(cfg)
	config.Set(cfg)

	return &BrandingService{config: cfg}
}

// A fork should not have to rename every surface away from an upstream domain: setting
// the instance name once is enough.
func TestBranding_NameBearingDefaultsFollowInstanceName(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {
		cfg.Branding.InstanceName = "niclas.xyz"
	})

	defaults := srv.Defaults()

	assert.Equal(t, "niclas.xyz", defaults.InstanceName)
	assert.Equal(t, "niclas.xyz", defaults.BadgeLabel)
	assert.Equal(t, "niclas.xyz Stats", defaults.ReadmeCardTitle)
	assert.Equal(t, "niclas.xyz", defaults.ActivityTitle)
}

func TestBranding_ExplicitConfigWinsOverDerivation(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {
		cfg.Branding.InstanceName = "niclas.xyz"
		cfg.Branding.BadgeLabel = "coding"
	})

	assert.Equal(t, "coding", srv.Defaults().BadgeLabel)
}

// Nothing may default to advertising the upstream project's domain.
func TestBranding_DefaultsDoNotMentionUpstreamDomain(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {})
	defaults := srv.Defaults()

	assert.NotContains(t, defaults.BadgeLabel, "wakapi.dev")
	assert.NotContains(t, defaults.ReadmeCardTitle, "wakapi.dev")
	assert.NotContains(t, defaults.ActivityTitle, "wakapi.dev")
	// An embedded chart must not pull an image from a foreign host unless asked to.
	assert.Empty(t, defaults.ActivityLogoUrl)
}

func TestBranding_MissingInstanceNameStillProducesAName(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {
		cfg.Branding.InstanceName = "  "
	})

	assert.Equal(t, "Wakapi", srv.Defaults().InstanceName)
	assert.Equal(t, "wakapi", srv.Defaults().BadgeLabel)
}

// Renaming the instance from the admin screen has to rename everything derived from
// it. Deriving before the overrides are merged would leave the old name behind.
func TestBranding_RenamingTheInstanceRenamesDerivedFields(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {})

	effective, rejected := srv.resolve(&models.Branding{InstanceName: "niclas.xyz"})

	assert.Empty(t, rejected)
	assert.Equal(t, "niclas.xyz", effective.BadgeLabel)
	assert.Equal(t, "niclas.xyz Stats", effective.ReadmeCardTitle)
	assert.Equal(t, "niclas.xyz", effective.ActivityTitle)
}

// A hex default written with a leading '#' is silently dropped by the config loader,
// because the value is parsed as yaml. Guard the resolved colours instead of trusting
// the tags: without this the chart falls back to hardcoded constants unnoticed.
func TestBranding_ConfiguredChartColorsSurviveLoading(t *testing.T) {
	srv := brandingServiceWith(func(cfg *config.Config) {})

	effective, rejected := srv.resolve(&models.Branding{})

	assert.Empty(t, rejected, "the configured chart colours must pass validation as loaded")
	assert.Equal(t, "#242B3A", effective.ActivityColorMinDark)
	assert.Equal(t, "#047857", effective.ActivityColorMaxDark)
}
