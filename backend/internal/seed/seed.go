package seed

import (
	"fmt"
	"time"

	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/models"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	if err := reset(db); err != nil {
		return err
	}
	userPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	doctorPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	adminPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}

	users := []models.User{
		{Name: "张三", Phone: "U1001", Email: "huahua20414@foxmail.com", PasswordHash: userPassword, Role: "user", Status: "active", Gender: "男", Age: 30, EmailNotify: true, Bio: "关注年度体检和慢病风险管理。"},
		{Name: "李医生", Phone: "D1001", Email: "huahua20414@foxmail.com", PasswordHash: doctorPassword, Role: "doctor", Status: "active", EmployeeNo: "D1001", Department: "健康管理科", Title: "主治医师", EmailNotify: true},
		{Name: "王医生", Phone: "D1002", Email: "wangdoctor@example.com", PasswordHash: doctorPassword, Role: "doctor", Status: "active", EmployeeNo: "D1002", Department: "内科", Title: "副主任医师", EmailNotify: true},
		{Name: "赵医生", Phone: "D1003", Email: "pendingdoctor@example.com", PasswordHash: doctorPassword, Role: "doctor", Status: "pending", EmployeeNo: "D1003", Department: "影像科", Title: "住院医师", EmailNotify: true},
		{Name: "系统管理员", Phone: "A1001", Email: "huahua20414@foxmail.com", PasswordHash: adminPassword, Role: "admin", Status: "active", EmailNotify: false},
	}
	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	institutions := []models.CheckupInstitution{
		{Name: "熙心健康体检中心总院", Address: "沈阳市和平区健康路 88 号", Phone: "024-88880001", OpenHours: "周一至周六 08:00-17:00", Status: "active"},
		{Name: "熙心健康高新区分院", Address: "沈阳市浑南区创新路 19 号", Phone: "024-88880002", OpenHours: "周一至周五 08:30-16:30", Status: "active"},
		{Name: "熙心健康河西分院", Address: "沈阳市铁西区卫工街 26 号", Phone: "024-88880003", OpenHours: "周一至周六 08:30-16:30", Status: "active"},
	}
	for _, institution := range institutions {
		if err := db.Create(&institution).Error; err != nil {
			return err
		}
	}

	packages := []models.CheckupPackage{
		{Name: "基础入职体检", Description: "适合入职、入学等基础健康筛查。", Price: 199, Items: "一般检查、血常规、尿常规、肝功能、胸片", Status: "active"},
		{Name: "白领健康套餐", Description: "覆盖常见慢病风险和办公室人群重点指标。", Price: 399, Items: "一般检查、血常规、血脂、血糖、肝肾功能、心电图、腹部彩超", Status: "active"},
		{Name: "全面深度体检", Description: "适合年度全面健康评估。", Price: 899, Items: "基础项目、肿瘤标志物、甲状腺彩超、颈动脉彩超、骨密度", Status: "active"},
	}
	for _, pkg := range packages {
		if err := db.Create(&pkg).Error; err != nil {
			return err
		}
	}

	var doctorLi, doctorWang models.User
	var mainInstitution, branchInstitution models.CheckupInstitution
	if err := db.Where("phone = ?", "D1001").First(&doctorLi).Error; err != nil {
		return err
	}
	if err := db.Where("email = ?", "wangdoctor@example.com").First(&doctorWang).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "熙心健康体检中心总院").First(&mainInstitution).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "熙心健康高新区分院").First(&branchInstitution).Error; err != nil {
		return err
	}
	if err := seedSlots(db, []models.User{doctorLi, doctorWang}, []models.CheckupInstitution{mainInstitution, branchInstitution}); err != nil {
		return err
	}

	return seedCompletedReport(db)
}

func reset(db *gorm.DB) error {
	for _, model := range []any{
		&models.MailLog{},
		&models.Report{},
		&models.Appointment{},
		&models.WaitlistEntry{},
		&models.ScheduleSlot{},
		&models.CheckupPackage{},
		&models.CheckupInstitution{},
		&models.User{},
	} {
		if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCompletedReport(db *gorm.DB) error {
	var user, doctor models.User
	var pkg models.CheckupPackage
	var slot models.ScheduleSlot
	if err := db.Where("phone = ?", "U1001").First(&user).Error; err != nil {
		return err
	}
	if err := db.Where("phone = ?", "D1001").First(&doctor).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "白领健康套餐").First(&pkg).Error; err != nil {
		return err
	}
	if err := db.Where("doctor_id = ? AND date = ? AND start_time = ?", doctor.ID, "2026-06-05", "09:00").First(&slot).Error; err != nil {
		return err
	}
	appointment := models.Appointment{
		OrderNo:         "YY2026060509000001",
		UserID:          user.ID,
		DoctorID:        doctor.ID,
		InstitutionID:   slot.InstitutionID,
		SlotID:          slot.ID,
		PackageID:       pkg.ID,
		AppointmentType: "年度体检",
		Date:            slot.Date,
		Period:          slot.Period,
		StartTime:       slot.StartTime,
		EndTime:         slot.EndTime,
		Status:          "reported",
		Note:            "模拟数据：用户已完成体检。",
	}
	if err := db.Create(&appointment).Error; err != nil {
		return err
	}
	if err := db.Model(&slot).Update("booked_count", 1).Error; err != nil {
		return err
	}
	report := models.Report{
		ReportNo:       "BG2026060510300001",
		AppointmentID:  appointment.ID,
		UserID:         user.ID,
		DoctorID:       doctor.ID,
		Summary:        "血常规、肝肾功能、心电图等主要指标未见明显异常。",
		Conclusion:     "总体健康状况良好。",
		Recommendation: "保持规律作息，每周进行 3 次以上中等强度运动，半年后复查血脂。",
	}
	return db.Create(&report).Error
}

func seedSlots(db *gorm.DB, doctors []models.User, institutions []models.CheckupInstitution) error {
	days := []string{"2026-06-05", "2026-06-06", "2026-06-07", "2026-06-08", "2026-06-09"}
	starts := []string{"08:30", "09:00", "09:30", "10:00", "10:30", "14:00", "14:30", "15:00", "15:30", "16:00"}
	for _, institution := range institutions {
		for _, doctor := range doctors {
			for _, day := range days {
				for _, start := range starts {
					end, err := addMinutes(start, 30)
					if err != nil {
						return err
					}
					period := "上午"
					if start >= "12:00" {
						period = "下午"
					}
					slot := models.ScheduleSlot{
						DoctorID:      doctor.ID,
						InstitutionID: institution.ID,
						Date:          day,
						Period:        period,
						StartTime:     start,
						EndTime:       end,
						Capacity:      1,
						BookedCount:   0,
						Status:        "available",
					}
					if err := db.Create(&slot).Error; err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func addMinutes(value string, minutes int) (string, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", err
	}
	end := parsed.Add(time.Duration(minutes) * time.Minute)
	return fmt.Sprintf("%02d:%02d", end.Hour(), end.Minute()), nil
}
