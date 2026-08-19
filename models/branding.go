package models

import (
	"net/url"
	"regexp"
	"strings"
)

// BrandingKey is the key_string_values entry the admin-editable overrides live under.
// Same storage pattern as the imprint and newsbox texts.
const BrandingKey = "branding"

var (
	hexColorPattern = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
	// shields.io accepts both a hex triplet and a handful of named colors, so a badge
	// color is validated more loosely than the chart colors, which must be hex to be
	// interpolated.
	badgeColorPattern = regexp.MustCompile(`^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[a-z]+)$`)
)

// Branding is the instance's visual identity: the parts of the UI that name or colour
// this instance rather than describe a user's data.
//
// Every field is optional in transit - an empty value means "keep the configured
// default" rather than "blank it out", which is what lets an admin change a single
// item without restating the rest.
type Branding struct {
	InstanceName    string `json:"instance_name"`
	BadgeLabel      string `json:"badge_label"`
	BadgeColor      string `json:"badge_color"`
	ReadmeCardTitle string `json:"readme_card_title"`
	LogoUrl         string `json:"logo_url"`
	FooterText      string `json:"footer_text"`
	FooterUrl       string `json:"footer_url"`

	ActivityTitle         string `json:"activity_title"`
	ActivityLogoUrl       string `json:"activity_logo_url"`
	ActivityColorMinDark  string `json:"activity_color_min_dark"`
	ActivityColorMaxDark  string `json:"activity_color_max_dark"`
	ActivityColorMinLight string `json:"activity_color_min_light"`
	ActivityColorMaxLight string `json:"activity_color_max_light"`
}

// WithOverrides returns a copy of b with every non-empty field of o applied.
func (b Branding) WithOverrides(o *Branding) Branding {
	if o == nil {
		return b
	}
	apply := func(target *string, value string) {
		if v := strings.TrimSpace(value); v != "" {
			*target = v
		}
	}
	apply(&b.InstanceName, o.InstanceName)
	apply(&b.BadgeLabel, o.BadgeLabel)
	apply(&b.BadgeColor, o.BadgeColor)
	apply(&b.ReadmeCardTitle, o.ReadmeCardTitle)
	apply(&b.LogoUrl, o.LogoUrl)
	apply(&b.FooterText, o.FooterText)
	apply(&b.FooterUrl, o.FooterUrl)
	apply(&b.ActivityTitle, o.ActivityTitle)
	apply(&b.ActivityLogoUrl, o.ActivityLogoUrl)
	apply(&b.ActivityColorMinDark, o.ActivityColorMinDark)
	apply(&b.ActivityColorMaxDark, o.ActivityColorMaxDark)
	apply(&b.ActivityColorMinLight, o.ActivityColorMinLight)
	apply(&b.ActivityColorMaxLight, o.ActivityColorMaxLight)
	return b
}

// Sanitized drops values that would produce broken or unsafe output, so a bad admin
// entry degrades to the built-in default instead of corrupting every badge and chart
// on the instance. Returns the cleaned copy plus the names of the rejected fields.
func (b Branding) Sanitized() (Branding, []string) {
	var rejected []string

	normalizeHex := func(field string, value string, fallback string) string {
		if hexColorPattern.MatchString(value) {
			if !strings.HasPrefix(value, "#") {
				return "#" + value
			}
			return value
		}
		rejected = append(rejected, field)
		return fallback
	}

	b.ActivityColorMinDark = normalizeHex("activity_color_min_dark", b.ActivityColorMinDark, "#242B3A")
	b.ActivityColorMaxDark = normalizeHex("activity_color_max_dark", b.ActivityColorMaxDark, "#047857")
	b.ActivityColorMinLight = normalizeHex("activity_color_min_light", b.ActivityColorMinLight, "#DCE3E1")
	b.ActivityColorMaxLight = normalizeHex("activity_color_max_light", b.ActivityColorMaxLight, "#047857")

	if !badgeColorPattern.MatchString(b.BadgeColor) {
		rejected = append(rejected, "badge_color")
		b.BadgeColor = "2F855A"
	}

	// A logo or link that is not a plain http(s), data: or same-origin url would end up
	// in an src or href attribute, so anything else (javascript:, most notably) is dropped.
	b.LogoUrl = sanitizeAssetUrl("logo_url", b.LogoUrl, &rejected)
	b.ActivityLogoUrl = sanitizeAssetUrl("activity_logo_url", b.ActivityLogoUrl, &rejected)
	b.FooterUrl = sanitizeLinkUrl("footer_url", b.FooterUrl, &rejected)

	return b, rejected
}

func sanitizeAssetUrl(field, value string, rejected *[]string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "data:image/") || strings.HasPrefix(value, "/") {
		return value
	}
	return sanitizeLinkUrl(field, value, rejected)
}

func sanitizeLinkUrl(field, value string, rejected *[]string) string {
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		*rejected = append(*rejected, field)
		return ""
	}
	return value
}
