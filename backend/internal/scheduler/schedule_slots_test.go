package scheduler

import (
	"testing"
	"time"
)

func TestNextDailyRunUsesNextTenPastMidnight(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	before := time.Date(2026, 6, 25, 0, 9, 0, 0, loc)
	wantSameDay := time.Date(2026, 6, 25, 0, 10, 0, 0, loc)
	if got := nextDailyRun(before); !got.Equal(wantSameDay) {
		t.Fatalf("expected same-day run %s, got %s", wantSameDay, got)
	}

	after := time.Date(2026, 6, 25, 0, 10, 0, 0, loc)
	wantNextDay := time.Date(2026, 6, 26, 0, 10, 0, 0, loc)
	if got := nextDailyRun(after); !got.Equal(wantNextDay) {
		t.Fatalf("expected next-day run %s, got %s", wantNextDay, got)
	}
}
