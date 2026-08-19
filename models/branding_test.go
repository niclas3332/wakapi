package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func defaultTestBranding() Branding {
	return Branding{
		InstanceName:          "Wakapi",
		BadgeLabel:            "wakapi.dev",
		BadgeColor:            "2F855A",
		ReadmeCardTitle:       "Wakapi.dev Stats",
		FooterText:            "a wakapi fork",
		FooterUrl:             "https://github.com/niclas3332/wakapi",
		ActivityTitle:         "Wakapi.dev",
		ActivityLogoUrl:       "https://wakapi.dev/assets/images/logo-gh.svg",
		ActivityColorMinDark:  "#242B3A",
		ActivityColorMaxDark:  "#047857",
		ActivityColorMinLight: "#DCE3E1",
		ActivityColorMaxLight: "#047857",
	}
}

func TestBranding_OverridesOnlyReplaceGivenFields(t *testing.T) {
	result := defaultTestBranding().WithOverrides(&Branding{BadgeLabel: "niclas.xyz"})

	assert.Equal(t, "niclas.xyz", result.BadgeLabel)
	// Everything the admin did not touch keeps its configured value rather than blanking out.
	assert.Equal(t, "2F855A", result.BadgeColor)
	assert.Equal(t, "Wakapi", result.InstanceName)
}

func TestBranding_BlankOverrideKeepsDefault(t *testing.T) {
	result := defaultTestBranding().WithOverrides(&Branding{BadgeLabel: "   "})
	assert.Equal(t, "wakapi.dev", result.BadgeLabel)
}

func TestBranding_SanitizedNormalizesHexColors(t *testing.T) {
	input := defaultTestBranding()
	input.ActivityColorMaxDark = "ff0000" // no leading hash

	result, rejected := input.Sanitized()

	assert.Empty(t, rejected)
	assert.Equal(t, "#ff0000", result.ActivityColorMaxDark)
}

func TestBranding_SanitizedRejectsUnusableColors(t *testing.T) {
	input := defaultTestBranding()
	input.ActivityColorMinDark = "rebeccapurple" // not interpolatable

	result, rejected := input.Sanitized()

	assert.Contains(t, rejected, "activity_color_min_dark")
	assert.Equal(t, "#242B3A", result.ActivityColorMinDark)
}

func TestBranding_SanitizedAcceptsNamedBadgeColor(t *testing.T) {
	input := defaultTestBranding()
	input.BadgeColor = "blueviolet" // shields.io understands named colors

	result, rejected := input.Sanitized()

	assert.Empty(t, rejected)
	assert.Equal(t, "blueviolet", result.BadgeColor)
}

func TestBranding_SanitizedDropsScriptUrls(t *testing.T) {
	input := defaultTestBranding()
	input.LogoUrl = "javascript:alert(1)"
	input.FooterUrl = "javascript:alert(1)"

	result, rejected := input.Sanitized()

	assert.Contains(t, rejected, "logo_url")
	assert.Contains(t, rejected, "footer_url")
	assert.Empty(t, result.LogoUrl)
	assert.Empty(t, result.FooterUrl)
}

func TestBranding_SanitizedKeepsDataAndRelativeLogos(t *testing.T) {
	input := defaultTestBranding()
	input.LogoUrl = "data:image/svg+xml;base64,PHN2Zy8+"

	result, rejected := input.Sanitized()
	assert.Empty(t, rejected)
	assert.Equal(t, input.LogoUrl, result.LogoUrl)

	input.LogoUrl = "/assets/images/logo.svg"
	result, rejected = input.Sanitized()
	assert.Empty(t, rejected)
	assert.Equal(t, "/assets/images/logo.svg", result.LogoUrl)
}
