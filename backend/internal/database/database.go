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
	return db.AutoMigrate(
		&models.User{},
		&models.CheckupInstitution{},
		&models.CheckupPackage{},
		&models.ScheduleSlot{},
		&models.Appointment{},
		&models.WaitlistEntry{},
		&models.Report{},
		&models.MailLog{},
	)
}
