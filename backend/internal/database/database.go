package database

import (
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
	return backfillInstitutionPackages(db)
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
