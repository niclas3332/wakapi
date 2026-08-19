package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIPrices_LookupByModelAndVersion(t *testing.T) {
	price, ok := DefaultAIPrices().Lookup("opus", "5")

	require.True(t, ok)
	assert.Equal(t, 5.00, price.Input)
	assert.Equal(t, 0.50, price.CachedInput)
	assert.Equal(t, 25.00, price.Output)
}

// An undeterminable version falls back to the model's most recent price rather than
// dropping the cost entirely.
func TestAIPrices_UnknownVersionUsesNewest(t *testing.T) {
	price, ok := DefaultAIPrices().Lookup("sonnet", "")

	require.True(t, ok)
	assert.Equal(t, 2.00, price.Input, "sonnet 5 is the newest configured version")
}

func TestAIPrices_NewestVersionComparesNumerically(t *testing.T) {
	table := AIPriceTable{"model": {
		"9":  {Input: 9},
		"10": {Input: 10},
		"2":  {Input: 2},
	}}

	price, ok := table.Lookup("model", "does-not-exist")

	require.True(t, ok)
	assert.Equal(t, 10.0, price.Input, "10 must sort above 9, not below it")
}

// No cached price configured means the normal input price applies. Assuming a discount
// on the operator's behalf would silently understate the cost.
func TestAIPrices_MissingCachedPriceFallsBackToInput(t *testing.T) {
	table := AIPriceTable{"model": {"1": {Input: 4, Output: 8}}}

	price, ok := table.Lookup("model", "1")

	require.True(t, ok)
	assert.Equal(t, 4.0, price.CachedInput)
}

func TestAIPrices_UnknownModelIsNotPriced(t *testing.T) {
	cost := DefaultAIPrices().Cost("phpstorm", "2026.2", 1_000_000, 0, 0)

	assert.False(t, cost.Priced)
	assert.Zero(t, cost.Total)
}

// The arithmetic is WakaTime's: tokens/1e6 * price, summed over the three kinds.
// Checked against a real week of data - 128.7M cached, 3.02M input, 0.54M output on
// opus - where WakaTime reported $93.13 for a mix that is 99.6% opus.
func TestAIPrices_CostMatchesPublishedFormula(t *testing.T) {
	cost := DefaultAIPrices().Cost("opus", "5", 3_020_000, 128_700_000, 540_000)

	assert.True(t, cost.Priced)
	assert.InDelta(t, 15.10, cost.Input, 0.01)
	assert.InDelta(t, 64.35, cost.CachedInput, 0.01)
	assert.InDelta(t, 13.50, cost.Output, 0.01)
	assert.InDelta(t, 92.95, cost.Total, 0.01)
}

// Cache reads are the bulk of the tokens, so dropping them - which is what wakapi did
// before the column existed - understates the cost by more than an order of magnitude.
func TestAIPrices_IgnoringCachedTokensUnderstatesCost(t *testing.T) {
	table := DefaultAIPrices()

	withCache := table.Cost("opus", "5", 3_020_000, 128_700_000, 540_000)
	withoutCache := table.Cost("opus", "5", 3_020_000, 0, 540_000)

	assert.Greater(t, withCache.Total, withoutCache.Total*3)
}

func TestAIPrices_SonnetWrappersShareTheSameRate(t *testing.T) {
	table := DefaultAIPrices()

	direct, ok := table.Lookup("sonnet", "4.6")
	require.True(t, ok)

	for _, wrapper := range []string{"cline", "windsurf", "roo code"} {
		price, ok := table.Lookup(wrapper, "4.6")
		require.True(t, ok, wrapper)
		assert.Equal(t, direct, price, wrapper)
	}
}
