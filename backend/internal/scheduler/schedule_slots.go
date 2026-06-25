package scheduler

import (
	"context"
	"log"
	"time"

	"health-checkup/backend/internal/seed"

	"gorm.io/gorm"
)

const futureSlotDays = 14

func StartScheduleSlotScheduler(ctx context.Context, db *gorm.DB) {
	runScheduleSlotEnsure(db, time.Now())
	go func() {
		for {
			timer := time.NewTimer(time.Until(nextDailyRun(time.Now())))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				runScheduleSlotEnsure(db, time.Now())
			}
		}
	}()
}

func runScheduleSlotEnsure(db *gorm.DB, now time.Time) {
	created, err := seed.EnsureFutureScheduleSlots(db, now, futureSlotDays)
	if err != nil {
		log.Printf("ensure future schedule slots failed: %v", err)
		return
	}
	log.Printf("future schedule slots ensured: created=%d days=%d", created, futureSlotDays)
}

func nextDailyRun(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 10, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
