package models

import (
	"sort"
	"strconv"
	"strings"
)

// AIPricesKey is the key_string_values entry the admin-editable price table lives under.
const AIPricesKey = "ai_model_prices"

// AIModelPrice is what one million tokens cost, in whatever currency the operator
// entered. Wakapi does not convert; it only multiplies.
type AIModelPrice struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
}

// AIPriceTable maps model -> version -> price, mirroring how vendors publish them.
type AIPriceTable map[string]map[string]AIModelPrice

// AICost is a computed cost, split so the UI can show where it came from.
type AICost struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
	Total       float64 `json:"total"`
	// Priced is false when no price is configured for the model, in which case the
	// figures are zero. Reporting that plainly beats implying the work was free.
	Priced bool `json:"priced"`
}

// Lookup finds the price for a model and version, following the same rules WakaTime
// documents at https://wakatime.com/faq#ai-model-cost:
//
//   - prices are per model and version;
//   - if the version cannot be determined, the model's most recent configured price is
//     used;
//   - if no cached-input price is configured, the normal input price applies - no
//     discount is assumed on the operator's behalf.
func (t AIPriceTable) Lookup(model, version string) (AIModelPrice, bool) {
	versions, ok := t[strings.ToLower(model)]
	if !ok || len(versions) == 0 {
		return AIModelPrice{}, false
	}

	price, ok := versions[version]
	if !ok {
		price, ok = versions[newestVersion(versions)]
		if !ok {
			return AIModelPrice{}, false
		}
	}

	if price.CachedInput == 0 {
		price.CachedInput = price.Input
	}
	return price, true
}

// Cost prices a bundle of tokens.
func (t AIPriceTable) Cost(model, version string, inputTokens, cachedInputTokens, outputTokens int64) AICost {
	price, ok := t.Lookup(model, version)
	if !ok {
		return AICost{}
	}

	const perMillion = 1_000_000.0
	cost := AICost{
		Input:       float64(inputTokens) / perMillion * price.Input,
		CachedInput: float64(cachedInputTokens) / perMillion * price.CachedInput,
		Output:      float64(outputTokens) / perMillion * price.Output,
		Priced:      true,
	}
	cost.Total = cost.Input + cost.CachedInput + cost.Output
	return cost
}

// newestVersion picks the highest version key, comparing numeric segments as numbers so
// "10" sorts above "9" rather than below it.
func newestVersion(versions map[string]AIModelPrice) string {
	keys := make([]string, 0, len(versions))
	for key := range versions {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return compareVersions(keys[i], keys[j]) > 0 })
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	for i := 0; i < len(partsA) || i < len(partsB); i++ {
		var segA, segB string
		if i < len(partsA) {
			segA = partsA[i]
		}
		if i < len(partsB) {
			segB = partsB[i]
		}

		numA, errA := strconv.Atoi(strings.SplitN(segA, "-", 2)[0])
		numB, errB := strconv.Atoi(strings.SplitN(segB, "-", 2)[0])
		if errA == nil && errB == nil && numA != numB {
			if numA > numB {
				return 1
			}
			return -1
		}
		if cmp := strings.Compare(segA, segB); cmp != 0 {
			return cmp
		}
	}
	return 0
}

// DefaultAIPrices is the price table wakapi starts with, taken from WakaTime's own
// published list (https://wakatime.com/faq#ai-model-cost, read 2026-08-19). It is only
// a starting point: an operator edits it under Settings -> Admin, and once stored the
// database wins so an edit is never overwritten on restart.
//
// Prices are per million tokens.
func DefaultAIPrices() AIPriceTable {
	// Several agent wrappers bill at the sonnet 4.6 rate; naming the rate once keeps
	// them from drifting apart when it changes.
	sonnetRate := AIModelPrice{Input: 3.00, CachedInput: 0.30, Output: 15.00}

	table := AIPriceTable{
		"opus": {
			"5":   {Input: 5.00, CachedInput: 0.50, Output: 25.00},
			"4.8": {Input: 5.00, CachedInput: 0.50, Output: 25.00},
		},
		"sonnet": {
			"5":   {Input: 2.00, CachedInput: 0.20, Output: 10.00},
			"4.6": sonnetRate,
			"4.5": sonnetRate,
		},
		"haiku": {
			"4.5": {Input: 1.00, CachedInput: 0.10, Output: 5.00},
		},
		"fable": {
			"5": {Input: 10.00, CachedInput: 1.00, Output: 50.00},
		},
		"gpt": {
			"5.6":       {Input: 5.00, CachedInput: 0.50, Output: 30.00},
			"5.5":       {Input: 5.00, CachedInput: 0.50, Output: 30.00},
			"5.4":       {Input: 2.50, CachedInput: 0.25, Output: 15.00},
			"5.3-codex": {Input: 1.75, CachedInput: 0.17, Output: 14.00},
			"5.6-luna":  {Input: 1.00, CachedInput: 0.10, Output: 6.00},
			"5.6-terra": {Input: 2.50, CachedInput: 0.25, Output: 15.00},
		},
		"grok": {
			"4.5": {Input: 2.00, CachedInput: 0.30, Output: 6.00},
		},
		"gemini": {
			"3.1-pro":   {Input: 2.00, CachedInput: 0.20, Output: 12.00},
			"3.5-flash": {Input: 1.50, CachedInput: 0.15, Output: 9.00},
			"3.6-flash": {Input: 1.50, CachedInput: 0.15, Output: 7.50},
		},
		"cursor": {
			"auto": {Input: 1.25, CachedInput: 0.12, Output: 6.00},
		},
		"copilot": {
			"5.3-codex": {Input: 1.75, CachedInput: 0.17, Output: 14.00},
		},
		"qwen": {
			"3.7-max": {Input: 2.50, CachedInput: 0.50, Output: 7.50},
		},
		"amp": {
			"5.6-sol": {Input: 5.00, CachedInput: 0.50, Output: 30.00},
		},
	}

	for _, wrapper := range []string{
		"cline", "cody", "continue", "goose", "kiro", "opencode", "pi", "qoder", "roo",
		"roo code", "windsurf",
	} {
		table[wrapper] = map[string]AIModelPrice{"4.6": sonnetRate}
	}

	return table
}
