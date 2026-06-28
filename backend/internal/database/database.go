package database

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"health-checkup/backend/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil && sqlDB.Ping() == nil {
				return db, nil
			}
		}
		time.Sleep(time.Second)
	}
	return nil, err
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.CheckupInstitution{},
		&models.CheckupPackage{},
		&models.InstitutionPackage{},
		&models.CheckupItem{},
		&models.PackageItem{},
		&models.Coupon{},
		&models.FamilyMember{},
		&models.PackageFavorite{},
		&models.PackageBrowseHistory{},
		&models.ScheduleTemplate{},
		&models.ScheduleSlot{},
		&models.Appointment{},
		&models.AppointmentItem{},
		&models.WaitlistEntry{},
		&models.Report{},
		&models.ServiceReview{},
		&models.MailLog{},
		&models.LoginLog{},
		&models.OperationLog{},
		&models.RolePermission{},
		&models.Notification{},
		&models.SystemAnnouncement{},
		&models.SupportTicket{},
		&models.SystemSetting{},
	); err != nil {
		return err
	}
	if err := backfillScheduleTemplates(db); err != nil {
		return err
	}
	return backfillInstitutionPackages(db)
}

type scheduleTemplateBackfillRow struct {
	DoctorID      uint
	InstitutionID uint
	Category      string
	Weekdays      []int
	StartTimes    []string
	Capacity      int
	Status        string
}

func backfillScheduleTemplates(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.ScheduleTemplate{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var slots []models.ScheduleSlot
	if err := db.Where("status <> ?", "deleted").Order("doctor_id asc, institution_id asc, category asc, date asc, start_time asc").Find(&slots).Error; err != nil {
		return err
	}
	if len(slots) == 0 {
		return nil
	}
	grouped := map[string]*scheduleTemplateBackfillRow{}
	for _, slot := range slots {
		slotDate, err := time.Parse("2006-01-02", slot.Date)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("%d|%d|%s", slot.DoctorID, slot.InstitutionID, slot.Category)
		row := grouped[key]
		if row == nil {
			row = &scheduleTemplateBackfillRow{
				DoctorID:      slot.DoctorID,
				InstitutionID: slot.InstitutionID,
				Category:      slot.Category,
				Capacity:      slot.Capacity,
				Status:        slot.Status,
			}
			grouped[key] = row
		}
		row.Weekdays = appendUniqueInt(row.Weekdays, int(slotDate.Weekday()))
		row.StartTimes = appendUniqueString(row.StartTimes, slot.StartTime)
		if slot.Capacity > row.Capacity {
			row.Capacity = slot.Capacity
		}
		if row.Status != "available" && slot.Status == "available" {
			row.Status = "available"
		}
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range grouped {
			sort.Ints(row.Weekdays)
			sort.Strings(row.StartTimes)
			weekdaysJSON, err := json.Marshal(row.Weekdays)
			if err != nil {
				return err
			}
			startTimesJSON, err := json.Marshal(row.StartTimes)
			if err != nil {
				return err
			}
			template := models.ScheduleTemplate{
				DoctorID:       row.DoctorID,
				InstitutionID:  row.InstitutionID,
				Category:       row.Category,
				WeekdaysText:   string(weekdaysJSON),
				StartTimesText: string(startTimesJSON),
				Capacity:       row.Capacity,
				Status:         row.Status,
			}
			if err := tx.Create(&template).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ScheduleSlot{}).
				Where("doctor_id = ? AND institution_id = ? AND category = ?", row.DoctorID, row.InstitutionID, row.Category).
				Update("template_id", template.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillInstitutionPackages(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.InstitutionPackage{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var links []models.InstitutionPackage
	if err := db.Table("schedule_slots").
		Select("DISTINCT schedule_slots.institution_id, checkup_packages.id AS package_id").
		Joins("JOIN checkup_institutions ON checkup_institutions.id = schedule_slots.institution_id").
		Joins("JOIN checkup_packages ON checkup_packages.category = schedule_slots.category").
		Where("schedule_slots.status <> ? AND checkup_institutions.status = ? AND checkup_packages.status = ?", "deleted", "active", "active").
		Scan(&links).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, link := range links {
			if link.InstitutionID == 0 || link.PackageID == 0 {
				continue
			}
			if err := tx.Where(models.InstitutionPackage{InstitutionID: link.InstitutionID, PackageID: link.PackageID}).
				FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func appendUniqueInt(values []int, target int) []int {
	for _, value := range values {
		if value == target {
			return values
		}
	}
	return append(values, target)
}

func appendUniqueString(values []string, target string) []string {
	for _, value := range values {
		if value == target {
			return values
		}
	}
	return append(values, target)
}
