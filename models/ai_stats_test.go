package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The model name comes from the editor column and its version is read back out of the
// user agent. No list of known names is involved, so a model released tomorrow parses
// like any other. These user agents are copied from real rows in the database.
func TestHeartbeat_AIModelReadsNameAndVersion(t *testing.T) {
	cases := []struct {
		name      string
		editor    string
		userAgent string
		model     string
		version   string
	}{
		{
			name:      "current opus",
			editor:    "Opus",
			userAgent: "wakatime/v2.23.0 (darwin-25.5.0-arm64) go1.26.5 opus/5 claude-code/2.1.220 claude-code-wakatime/4.1.0",
			model:     "opus",
			version:   "5",
		},
		{
			name:      "dashed version",
			editor:    "Opus",
			userAgent: "wakatime/v2.22.2 (darwin-25.5.0-arm64) go1.26.5 opus/4-8 claude-code/2.1.215 claude-code-wakatime/4.1.0",
			model:     "opus",
			version:   "4.8",
		},
		{
			name:      "build date dropped",
			editor:    "Haiku",
			userAgent: "wakatime/v2.20.5 (darwin) go1.26.4 haiku/4-5-20251001 claude-code/2.1.161",
			model:     "haiku",
			version:   "4.5",
		},
		{
			name:      "codex suffix kept",
			editor:    "Gpt",
			userAgent: "wakatime/v2.20.5 (darwin) go1.26.4 gpt/5-3-codex codex-cli/1.0",
			model:     "gpt",
			version:   "5.3-codex",
		},
		{
			name:      "a model never seen before still parses",
			editor:    "Nebula",
			userAgent: "wakatime/v2.30.0 (darwin) go1.28.0 nebula/2-1 some-tool/1.0",
			model:     "nebula",
			version:   "2.1",
		},
		{
			// A plugin that reports ai coding without naming a model leaves the ide in
			// the editor column. It is reported as-is and simply carries no price.
			name:      "no model in the agent",
			editor:    "Phpstorm",
			userAgent: "wakatime/v2.22.2 (darwin) go1.26.5 phpstorm/2026.2 phpstorm-wakatime/16.1.2",
			model:     "phpstorm",
			version:   "2026.2",
		},
		{
			name:      "version missing from the agent",
			editor:    "Sonnet",
			userAgent: "wakatime/v2.20.5 (darwin) go1.26.4 claude-code/2.1.161",
			model:     "sonnet",
			version:   "",
		},
		{name: "no editor at all", editor: "", userAgent: "", model: "", version: ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			model, version := (&Heartbeat{Editor: testCase.editor, UserAgent: testCase.userAgent}).AIModel()
			assert.Equal(t, testCase.model, model)
			assert.Equal(t, testCase.version, version)
		})
	}
}

func TestNormalizeModelVersion(t *testing.T) {
	assert.Equal(t, "4.8", NormalizeModelVersion("4-8"))
	assert.Equal(t, "3.1-pro", NormalizeModelVersion("3-1-pro"))
	assert.Equal(t, "5.6-terra", NormalizeModelVersion("5-6-terra"))
	// A build date is not part of what anyone prices by.
	assert.Equal(t, "4.5", NormalizeModelVersion("4-5-20251001"))
	assert.Equal(t, "auto", NormalizeModelVersion("auto"))
	assert.Equal(t, "", NormalizeModelVersion(""))
}

// The category is the discriminator, so a normal editor is never mistaken for a model.
func TestHeartbeat_IsAIUsesCategoryNotEditor(t *testing.T) {
	assert.True(t, (&Heartbeat{Category: "ai coding", Editor: "Opus"}).IsAI())
	assert.True(t, (&Heartbeat{Category: "AI Coding"}).IsAI())
	assert.False(t, (&Heartbeat{Category: "coding", Editor: "PhpStorm"}).IsAI())
	// Even an editor that looks like a model name is not AI activity without the category.
	assert.False(t, (&Heartbeat{Category: "coding", Editor: "Opus"}).IsAI())
	assert.False(t, (&Heartbeat{}).IsAI())
}
