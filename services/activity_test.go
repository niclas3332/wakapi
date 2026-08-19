package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestActivityLevels_NoActivityStaysAtZero(t *testing.T) {
	levels := activityLevels([]time.Duration{0, 0, 0})
	assert.Equal(t, []int{0, 0, 0}, levels)
}

func TestActivityLevels_EmptyDaysNeverGetAShade(t *testing.T) {
	levels := activityLevels([]time.Duration{0, time.Hour, 0, 8 * time.Hour})
	assert.Equal(t, 0, levels[0])
	assert.Equal(t, 0, levels[2])
	assert.Greater(t, levels[1], 0)
	assert.Greater(t, levels[3], 0)
}

// A single very long day used to flatten the whole chart, because the scale was linear
// against the maximum. Ranking against the days actually worked keeps ordinary days legible.
func TestActivityLevels_OutlierDoesNotFlattenOrdinaryDays(t *testing.T) {
	totals := []time.Duration{
		30 * time.Minute,
		time.Hour,
		2 * time.Hour,
		3 * time.Hour,
		4 * time.Hour,
		6 * time.Hour,
		8 * time.Hour,
		16*time.Hour + 30*time.Minute, // the outlier
	}

	levels := activityLevels(totals)

	// The 8 hour day sits in the top band despite the 16:30 outlier existing.
	assert.Equal(t, activityLevelCount, levels[6])
	assert.Equal(t, activityLevelCount, levels[7])
	// And the shortest day is still visibly the lightest active shade.
	assert.Equal(t, 1, levels[0])
}

func TestActivityLevels_AreMonotonic(t *testing.T) {
	totals := []time.Duration{
		10 * time.Minute, 20 * time.Minute, time.Hour, 90 * time.Minute,
		2 * time.Hour, 5 * time.Hour, 9 * time.Hour, 12 * time.Hour,
	}

	levels := activityLevels(totals)

	for i := 1; i < len(levels); i++ {
		assert.GreaterOrEqual(t, levels[i], levels[i-1], "a longer day must never be lighter than a shorter one")
	}
	assert.LessOrEqual(t, levels[len(levels)-1], activityLevelCount)
}

func TestActivityLevels_SingleActiveDayIsFullyShaded(t *testing.T) {
	levels := activityLevels([]time.Duration{0, 45 * time.Minute, 0})
	assert.Equal(t, 1, levels[1])
}
