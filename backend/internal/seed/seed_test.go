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
	assertMinCount(t, db, &models.PackageItem{}, "1 = ?", []any{1}, 20)

	assertDoctorDepartmentsAreUneven(t, db)
	assertEveryPackageHasItems(t, db)
	assertEveryDemoTimeHasSlots(t, db)
	assertNextTwoWeeksHaveCompleteSlotTimes(t, db)
	assertEveryInstitutionPackageCategoryHasDailyFutureAvailableSlot(t, db)
	assertSomeTimesHaveMultipleDoctors(t, db)
	assertSomeSlotsAreFull(t, db)
	assertNoSlotIsOverbooked(t, db)
	assertSeededIDCardsAreValid(t, db)
}

func assertEveryPackageHasItems(t *testing.T, db *gorm.DB) {
	t.Helper()
	var packages []models.CheckupPackage
	if err := db.Where("status = ?", "active").Find(&packages).Error; err != nil {
		t.Fatalf("query packages: %v", err)
	}
	for _, pkg := range packages {
		var count int64
		if err := db.Model(&models.PackageItem{}).Where("package_id = ?", pkg.ID).Count(&count).Error; err != nil {
			t.Fatalf("count package items for %s: %v", pkg.Name, err)
		}
		if count == 0 {
			t.Fatalf("expected package %q to have package item links", pkg.Name)
		}
	}
}

func TestEnsureFutureScheduleSlotsIsIdempotentAndRollsForward(t *testing.T) {
	db := openSeedTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("run seed: %v", err)
	}
	initialSlots := countRows(t, db, &models.ScheduleSlot{})
	bookedSlotID, bookedCount := firstBookedSlot(t, db)

	created, err := EnsureFutureScheduleSlots(db, time.Now(), 14)
	if err != nil {
		t.Fatalf("ensure future schedule slots: %v", err)
	}
	if created != 0 {
		t.Fatalf("expected idempotent ensure to create 0 slots, got %d", created)
	}
	assertCount(t, db, &models.ScheduleSlot{}, initialSlots)
	assertSlotBookedCount(t, db, bookedSlotID, bookedCount)

	tomorrow := time.Now().AddDate(0, 0, 1)
	created, err = EnsureFutureScheduleSlots(db, tomorrow, 14)
	if err != nil {
		t.Fatalf("ensure rolled future schedule slots: %v", err)
	}
	if created == 0 {
		t.Fatalf("expected rolled ensure to create slots for the new last day")
	}
	assertMaxScheduleSlotDate(t, db, dayStart(tomorrow).AddDate(0, 0, 13).Format("2006-01-02"))
	assertSlotBookedCount(t, db, bookedSlotID, bookedCount)
}

func TestEnsureFutureScheduleSlotsDoesNotRecreateDeletedSlot(t *testing.T) {
	db := openSeedTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("run seed: %v", err)
	}
	var slot models.ScheduleSlot
	if err := db.Where("booked_count = 0").First(&slot).Error; err != nil {
		t.Fatalf("find empty slot: %v", err)
	}
	if err := db.Model(&models.ScheduleSlot{}).Where("id = ?", slot.ID).Update("status", "deleted").Error; err != nil {
		t.Fatalf("mark slot deleted: %v", err)
	}
	if _, err := EnsureFutureScheduleSlots(db, time.Now(), 14); err != nil {
		t.Fatalf("ensure future schedule slots: %v", err)
	}
	var count int64
	if err := db.Model(&models.ScheduleSlot{}).
		Where("doctor_id = ? AND date = ? AND start_time = ?", slot.DoctorID, slot.Date, slot.StartTime).
		Count(&count).Error; err != nil {
		t.Fatalf("count exact slot: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected deleted generated slot not to be recreated, got %d matching rows", count)
	}
}

func TestEnsureFutureScheduleSlotsSkipsInstitutionWithoutScheduleTemplate(t *testing.T) {
	db := openSeedTestDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("run seed: %v", err)
	}
	institution := models.CheckupInstitution{Name: "未排班新机构", Address: "新机构路 1 号", Status: "active"}
	if err := db.Create(&institution).Error; err != nil {
		t.Fatalf("create institution: %v", err)
	}

	created, err := EnsureFutureScheduleSlots(db, time.Now().AddDate(0, 0, 1), 14)
	if err != nil {
		t.Fatalf("ensure future schedule slots: %v", err)
	}
	if created == 0 {
		t.Fatalf("expected rolling ensure to create slots for existing schedules")
	}

	var count int64
	if err := db.Model(&models.ScheduleSlot{}).Where("institution_id = ?", institution.ID).Count(&count).Error; err != nil {
		t.Fatalf("count new institution slots: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no slots for institution without schedule template, got %d", count)
	}
}

func TestEnsureDoctorFutureScheduleSlotsUsesExistingInstitutionTemplates(t *testing.T) {
	db := openSeedTestDB(t)
	institution := models.CheckupInstitution{ID: 1, Name: "已有排班机构", Address: "健康路 1 号", Status: "active"}
	newInstitution := models.CheckupInstitution{ID: 2, Name: "未排班新机构", Address: "新机构路 1 号", Status: "active"}
	activeDoctor := models.User{ID: 10, Name: "已有医生", Email: "doctor1@example.com", Phone: "D001", Role: "doctor", Status: "active", PasswordHash: "hash"}
	newDoctor := models.User{ID: 11, Name: "新医生", Email: "doctor2@example.com", Phone: "D002", Role: "doctor", Status: "active", PasswordHash: "hash", Specialties: "入职体检"}
	pkg := models.CheckupPackage{ID: 20, Name: "入职基础体检", Category: "入职体检", Price: 199, Status: "active"}
	templateDate := dayStart(time.Now()).Format("2006-01-02")
	templateSlot := models.ScheduleSlot{ID: 30, DoctorID: activeDoctor.ID, InstitutionID: institution.ID, Date: templateDate, Period: "上午", Category: "入职体检", StartTime: "09:00", EndTime: "09:30", Capacity: 1, Status: "available"}
	for _, row := range []any{&institution, &newInstitution, &activeDoctor, &newDoctor, &pkg, &templateSlot} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create row %#v: %v", row, err)
		}
	}

	created, err := EnsureDoctorFutureScheduleSlots(db, newDoctor.ID, []string{"入职体检"}, time.Now(), 14)
	if err != nil {
		t.Fatalf("ensure doctor future schedule slots: %v", err)
	}
	if created == 0 {
		t.Fatalf("expected slots to be created for new doctor from existing templates")
	}
	assertMinCount(t, db, &models.ScheduleSlot{}, "doctor_id = ? AND institution_id = ?", []any{newDoctor.ID, institution.ID}, 1)

	var newInstitutionSlots int64
	if err := db.Model(&models.ScheduleSlot{}).Where("doctor_id = ? AND institution_id = ?", newDoctor.ID, newInstitution.ID).Count(&newInstitutionSlots).Error; err != nil {
		t.Fatalf("count new institution doctor slots: %v", err)
	}
	if newInstitutionSlots != 0 {
		t.Fatalf("expected no slots for new institution without template, got %d", newInstitutionSlots)
	}
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

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return count
}

func assertCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	got := countRows(t, db, model)
	if got != want {
		t.Fatalf("expected %d rows for %T, got %d", want, model, got)
	}
}

func firstBookedSlot(t *testing.T, db *gorm.DB) (uint, int) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.Where("booked_count > 0").First(&slot).Error; err != nil {
		t.Fatalf("find booked slot: %v", err)
	}
	return slot.ID, slot.BookedCount
}

func assertSlotBookedCount(t *testing.T, db *gorm.DB, slotID uint, want int) {
	t.Helper()
	var slot models.ScheduleSlot
	if err := db.First(&slot, slotID).Error; err != nil {
		t.Fatalf("find slot %d: %v", slotID, err)
	}
	if slot.BookedCount != want {
		t.Fatalf("expected slot %d booked_count %d, got %d", slotID, want, slot.BookedCount)
	}
}

func assertMaxScheduleSlotDate(t *testing.T, db *gorm.DB, want string) {
	t.Helper()
	var maxDate string
	if err := db.Model(&models.ScheduleSlot{}).Select("MAX(date)").Scan(&maxDate).Error; err != nil {
		t.Fatalf("query max slot date: %v", err)
	}
	if maxDate != want {
		t.Fatalf("expected max schedule slot date %s, got %s", want, maxDate)
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

func assertEveryInstitutionPackageCategoryHasDailyFutureAvailableSlot(t *testing.T, db *gorm.DB) {
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
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	for offset := 0; offset < 14; offset++ {
		date := today.AddDate(0, 0, offset).Format("2006-01-02")
		for _, institution := range institutions {
			for _, category := range categories {
				var count int64
				err := db.Model(&models.ScheduleSlot{}).
					Where("institution_id = ? AND category = ? AND date = ? AND status = ? AND booked_count < capacity", institution.ID, category, date, "available").
					Count(&count).Error
				if err != nil {
					t.Fatalf("count future slots for %s/%s/%s: %v", date, institution.Name, category, err)
				}
				if count == 0 {
					t.Fatalf("expected daily future available slot for date %q, institution %q and category %q", date, institution.Name, category)
				}
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
