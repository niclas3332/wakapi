package models

import (
	"strings"
	"time"
)

// AIStatsDaily is one day's AI activity for a user, split by project and model.
//
// It exists because the raw signal lives on individual heartbeats, which are far too
// many to aggregate on every page load: a year of activity is hundreds of thousands of
// rows. This table is written by the nightly aggregation job and backfilled once, and
// the current day is recomputed live on top of it - the same shape SummaryService uses.
//
// Column widths are deliberately below the varchar(191) the rest of the schema uses,
// so the composite unique index stays well inside InnoDB's 3072 byte key limit.
type AIStatsDaily struct {
	ID     uint   `gorm:"primaryKey"`
	UserID string `gorm:"not null; size:191; uniqueIndex:idx_ai_stats_daily_key,priority:1; index:idx_ai_stats_daily_user_date,priority:1"`
	// Date is midnight in the user's own time zone, which is where their day starts.
	Date         CustomTime `gorm:"not null; uniqueIndex:idx_ai_stats_daily_key,priority:2; index:idx_ai_stats_daily_user_date,priority:2"`
	Project      string     `gorm:"size:191; uniqueIndex:idx_ai_stats_daily_key,priority:3"`
	Model        string     `gorm:"size:64; uniqueIndex:idx_ai_stats_daily_key,priority:4"`
	ModelVersion string     `gorm:"size:32; uniqueIndex:idx_ai_stats_daily_key,priority:5"`

	// Line changes arrive as signed deltas - negative when a file shrank - so they are
	// split by sign on the way in. Storing only the sum would make "how much did you
	// write" collapse to a negative number the first time a large file is truncated,
	// which is what the raw data does: one heartbeat in this database carries -34582.
	// Net is added minus removed, volume is added plus removed; both stay available.
	AILinesAdded      int `gorm:"not null; default:0"`
	AILinesRemoved    int `gorm:"not null; default:0"`
	HumanLinesAdded   int `gorm:"not null; default:0"`
	HumanLinesRemoved int `gorm:"not null; default:0"`
	// From the heartbeat's own line_additions / line_deletions columns, which most
	// plugins never populate. Kept separate from the derived numbers above so an
	// instance whose plugins do send them is not mixed up with one that does not.
	LineAdditions int `gorm:"not null; default:0"`
	LineDeletions int `gorm:"not null; default:0"`

	AIInputTokens       int64 `gorm:"not null; default:0"`
	AICachedInputTokens int64 `gorm:"not null; default:0"`
	AIOutputTokens      int64 `gorm:"not null; default:0"`

	PromptCount       int   `gorm:"not null; default:0"`
	PromptLengthTotal int64 `gorm:"not null; default:0"`
	HeartbeatCount    int   `gorm:"not null; default:0"`
}

// IsEmpty reports whether a row carries no information worth storing. Days that only
// produced ordinary heartbeats are skipped, so this table stays about AI activity and
// line changes rather than duplicating the summaries.
func (s *AIStatsDaily) IsEmpty() bool {
	return s.AILinesAdded == 0 && s.AILinesRemoved == 0 &&
		s.HumanLinesAdded == 0 && s.HumanLinesRemoved == 0 &&
		s.LineAdditions == 0 && s.LineDeletions == 0 &&
		s.AIInputTokens == 0 && s.AICachedInputTokens == 0 && s.AIOutputTokens == 0 &&
		s.PromptCount == 0 && s.PromptLengthTotal == 0
}

// AddLineDelta files a signed line-change delta under added or removed.
func AddLineDelta(added, removed *int, delta int) {
	if delta > 0 {
		*added += delta
	} else {
		*removed += -delta
	}
}

func (AIStatsDaily) TableName() string {
	return "ai_stats_daily"
}

// AISessionDaily records that a session was seen on a day.
//
// Sessions are kept apart from AIStatsDaily because one session can span several
// projects: counting them from the stats rows would count such a session once per
// project, and "how many sessions did I have" would drift upwards over time.
type AISessionDaily struct {
	ID        uint       `gorm:"primaryKey"`
	UserID    string     `gorm:"not null; size:191; uniqueIndex:idx_ai_sessions_daily_key,priority:1; index:idx_ai_sessions_daily_user_date,priority:1"`
	Date      CustomTime `gorm:"not null; uniqueIndex:idx_ai_sessions_daily_key,priority:2; index:idx_ai_sessions_daily_user_date,priority:2"`
	SessionID string     `gorm:"size:128; uniqueIndex:idx_ai_sessions_daily_key,priority:3"`
	Project   string     `gorm:"size:191"`
	Model     string     `gorm:"size:64"`
}

func (AISessionDaily) TableName() string {
	return "ai_sessions_daily"
}

// AICategory is the discriminator for AI activity. It is the category the plugins
// themselves report, which is what WakaTime keys on too - not a list of known model
// names, which would go stale the moment a new model ships.
const AICategory = "ai coding"

// IsAI reports whether a heartbeat describes AI-assisted work.
func (h *Heartbeat) IsAI() bool {
	return strings.EqualFold(h.Category, AICategory)
}

// AIModel returns the model name and version behind an AI heartbeat.
//
// The name is the editor column. For AI heartbeats that is the model, because the user
// agent is positional and the ingestion parser already picks that position out:
// "wakatime/v2.23.0 (darwin) go1.26.5 opus/5 claude-code/2.1.220" stores "Opus". No
// list of known model names is involved, so a model shipped tomorrow is handled like
// any other.
//
// The version is not stored anywhere, so it is read back out of the user agent by
// matching the same token: "opus/5" gives "5", "opus/4-8" gives "4.8". Normalising it
// this way makes it a price table key; a build date suffix is dropped, so
// "haiku/4-5-20251001" gives "4.5".
//
// Known limitation: some plugins report category "ai coding" without naming a model at
// all (a JetBrains IDE sending its own AI usage, say). For those the editor is the IDE,
// so that is what shows up as the model. It carries no price, which is the honest
// answer - the heartbeat genuinely does not say which model ran.
func (h *Heartbeat) AIModel() (model string, version string) {
	model = strings.ToLower(strings.TrimSpace(h.Editor))
	if model == "" {
		return "", ""
	}
	return model, versionFromUserAgent(h.UserAgent, model)
}

// versionFromUserAgent finds the "<name>/<version>" token belonging to the given name.
func versionFromUserAgent(userAgent, name string) string {
	for _, field := range strings.Fields(userAgent) {
		token, rest, found := strings.Cut(field, "/")
		if !found {
			continue
		}
		if strings.ToLower(token) == name {
			return NormalizeModelVersion(rest)
		}
	}
	return ""
}

// NormalizeModelVersion turns a user agent's version token into a price table key.
func NormalizeModelVersion(raw string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(raw)), "-")

	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		// A build date is not part of the version people price by, and it is the only
		// 8-digit run that appears here.
		if len(part) == 8 && isAllDigits(part) {
			continue
		}
		if part == "" {
			continue
		}
		kept = append(kept, part)
	}
	if len(kept) == 0 {
		return ""
	}

	// Numeric segments join with dots ("4-8" -> "4.8"); a trailing word stays a suffix
	// ("5-3-codex" -> "5.3-codex"), which is how the price table names them.
	var numeric []string
	var suffix []string
	for _, part := range kept {
		if isAllDigits(part) && len(suffix) == 0 {
			numeric = append(numeric, part)
		} else {
			suffix = append(suffix, part)
		}
	}

	version := strings.Join(numeric, ".")
	if len(suffix) > 0 {
		joined := strings.Join(suffix, "-")
		if version == "" {
			return joined
		}
		return version + "-" + joined
	}
	return version
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// AIStatsKey identifies one AIStatsDaily row.
type AIStatsKey struct {
	Date         time.Time
	Project      string
	Model        string
	ModelVersion string
}
