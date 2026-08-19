package v1

import (
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
)

// https://shields.io/endpoint

type BadgeData struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// NewBadgeDataFrom renders a summary as badge data. Label and color are the instance's
// branding defaults; a caller may still override them per request (?label=, ?color=).
func NewBadgeDataFrom(summary *models.Summary, branding *models.Branding) *BadgeData {
	return &BadgeData{
		SchemaVersion: 1,
		Label:         branding.BadgeLabel,
		Message:       helpers.FmtWakatimeDuration(summary.TotalTime()),
		Color:         branding.BadgeColor,
	}
}
