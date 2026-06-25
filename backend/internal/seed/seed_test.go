package seed

import (
	"testing"
	"time"

	"health-checkup/backend/internal/database"
	"health-checkup/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunSeedsRichDemoBusinessData(t *testing.T) {
	db := openSeedTestDB(t)

	if err := Run(db); err != nil {
		t.Fatalf("run seed: %v", err)
	}

	assertMinCount(t, db, &models.User{}, "role = ?", []any{"user"}, 30)
	assertMinCount(t, db, &models.User{}, "role = ? AND status = ?", []any{"doctor", "active"}, 18)
	assertMinCount(t, db, &models.User{}, "role = ? AND status = ?", []any{"doctor", "pending"}, 2)
	assertMinCount(t, db, &models.ScheduleSlot{}, "status = ?", []any{"available"}, 130)
	assertMinCount(t, db, &models.Appointment{}, "status <> ?", []any{"canceled"}, 60)
	assertMinCount(t, db, &models.WaitlistEntry{}, "status = ?", []any{"waiting"}, 10)

	assertDoctorDepartmentsAreUneven(t, db)
	assertEveryDemoTimeHasSlots(t, db)
	assertNextTwoWeeksHaveCompleteSlotTimes(t, db)
	assertEveryInstitutionPackageCategoryHasFutureAvailableSlot(t, db)
	assertSomeTimesHaveMultipleDoctors(t, db)
	assertSomeSlotsAreFull(t, db)
	assertNoSlotIsOverbooked(t, db)
	assertSeededIDCardsAreValid(t, db)
}

func assertNextTwoWeeksHaveCompleteSlotTimes(t *testing.T, db *gorm.DB) {
	t.Helper()
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	for offset := 0; offset < 14; offset++ {
		date := today.AddDate(0, 0, offset).Format("2006-01-02")
		for _, startTime := range []string{"08:00", "08:30", "09:00", "09:30", "10:00", "10:30", "11:00", "11:30", "13:30", "14:00", "14:30", "15:00", "15:30", "16:00", "16:30"} {
			var count int64
			if err := db.Model(&models.ScheduleSlot{}).Where("date = ? AND start_time = ?", date, startTime).Count(&count).Error; err != nil {
				t.Fatalf("count slots for %s %s: %v", date, startTime, err)
			}
			if count == 0 {
				t.Fatalf("expected at least one slot for %s %s", date, startTime)
			}
		}
	}
}

func openSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func assertMinCount(t *testing.T, db *gorm.DB, model any, condition string, args []any, min int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(condition, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count < min {
		t.Fatalf("expected at least %d rows for %T, got %d", min, model, count)
	}
}

func assertDoctorDepartmentsAreUneven(t *testing.T, db *gorm.DB) {
	t.Helper()
	type departmentCount struct {
		Department string
		Count      int64
	}
	var rows []departmentCount
	if err := db.Model(&models.User{}).
		Select("department, COUNT(*) AS count").
		Where("role = ? AND status = ?", "doctor", "active").
		Group("department").
		Order("count desc").
		Scan(&rows).Error; err != nil {
		t.Fatalf("count departments: %v", err)
	}
	if len(rows) < 5 {
		t.Fatalf("expected at least 5 active doctor departments, got %d", len(rows))
	}
	if rows[0].Count <= rows[len(rows)-1].Count {
		t.Fatalf("expected uneven department distribution, got %#v", rows)
	}
}

func assertEveryDemoTimeHasSlots(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, startTime := range []string{"08:00", "08:30", "09:00", "09:30", "10:00", "10:30", "11:00", "11:30", "13:30", "14:00", "14:30", "15:00", "15:30", "16:00", "16:30"} {
		var count int64
		if err := db.Model(&models.ScheduleSlot{}).Where("start_time = ?", startTime).Count(&count).Error; err != nil {
			t.Fatalf("count slots for %s: %v", startTime, err)
		}
		if count == 0 {
			t.Fatalf("expected seeded slots for %s", startTime)
		}
	}
}

func assertEveryInstitutionPackageCategoryHasFutureAvailableSlot(t *testing.T, db *gorm.DB) {
	t.Helper()
	var institutions []models.CheckupInstitution
	if err := db.Where("status = ?", "active").Find(&institutions).Error; err != nil {
		t.Fatalf("query institutions: %v", err)
	}
	var categories []string
	if err := db.Model(&models.CheckupPackage{}).
		Distinct("category").
		Where("status = ?", "active").
		Pluck("category", &categories).Error; err != nil {
		t.Fatalf("query package categories: %v", err)
	}
	for _, institution := range institutions {
		for _, category := range categories {
			var count int64
			err := db.Model(&models.ScheduleSlot{}).
				Where("institution_id = ? AND category = ? AND date >= DATE('now') AND status = ? AND booked_count < capacity", institution.ID, category, "available").
				Count(&count).Error
			if err != nil {
				t.Fatalf("count future slots for %s/%s: %v", institution.Name, category, err)
			}
			if count == 0 {
				t.Fatalf("expected future available slot for institution %q and category %q", institution.Name, category)
			}
		}
	}
}

func assertSomeTimesHaveMultipleDoctors(t *testing.T, db *gorm.DB) {
	t.Helper()
	type groupedSlot struct {
		Date      string
		StartTime string
		Doctors   int64
	}
	var row groupedSlot
	err := db.Model(&models.ScheduleSlot{}).
		Select("date, start_time, COUNT(DISTINCT doctor_id) AS doctors").
		Group("date, start_time").
		Having("COUNT(DISTINCT doctor_id) > 1").
		Order("doctors desc").
		First(&row).Error
	if err != nil {
		t.Fatalf("expected at least one time with multiple doctors: %v", err)
	}
}

func assertSomeSlotsAreFull(t *testing.T, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.Model(&models.ScheduleSlot{}).Where("booked_count = capacity AND capacity > 0").Count(&count).Error; err != nil {
		t.Fatalf("count full slots: %v", err)
	}
	if count < 20 {
		t.Fatalf("expected at least 20 full slots, got %d", count)
	}
}

func assertNoSlotIsOverbooked(t *testing.T, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.Model(&models.ScheduleSlot{}).Where("booked_count > capacity").Count(&count).Error; err != nil {
		t.Fatalf("count overbooked slots: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no overbooked slots, got %d", count)
	}
}

func assertSeededIDCardsAreValid(t *testing.T, db *gorm.DB) {
	t.Helper()
	var users []models.User
	if err := db.Where("role = ? AND id_card <> ?", "user", "").Find(&users).Error; err != nil {
		t.Fatalf("query users with id cards: %v", err)
	}
	for _, user := range users {
		if !validSeedIDCard(user.IDCard) {
			t.Fatalf("expected valid user id card for %s, got %s", user.Email, user.IDCard)
		}
	}
	var members []models.FamilyMember
	if err := db.Where("id_card <> ?", "").Find(&members).Error; err != nil {
		t.Fatalf("query family members with id cards: %v", err)
	}
	for _, member := range members {
		if !validSeedIDCard(member.IDCard) {
			t.Fatalf("expected valid family member id card for %s, got %s", member.Name, member.IDCard)
		}
	}
}

func validSeedIDCard(idCard string) bool {
	if len(idCard) != 18 {
		return false
	}
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i, weight := range weights {
		if idCard[i] < '0' || idCard[i] > '9' {
			return false
		}
		sum += int(idCard[i]-'0') * weight
	}
	return idCard[17] == checkCodes[sum%11]
}
