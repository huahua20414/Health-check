package seed

import (
	"fmt"
	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/models"
	"time"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	userPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	doctorPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	adminPassword, err := auth.HashPassword("admin123")
	if err != nil {
		return err
	}
	users := []models.User{
		{Name: "张三", Phone: "13800000001", Email: "huahua20414@foxmail.com", PasswordHash: userPassword, Role: "user", Status: "active", Gender: "男", Age: 30, EmailNotify: true, Bio: "关注年度体检和慢病风险管理。"},
		{Name: "李医生", Phone: "13900000001", Email: "huahua20414@foxmail.com", PasswordHash: doctorPassword, Role: "doctor", Status: "active", EmployeeNo: "D1001", Department: "健康管理科", Title: "主治医师", EmailNotify: true},
		{Name: "王医生", Phone: "13900000002", Email: "huahua20414@foxmail.com", PasswordHash: doctorPassword, Role: "doctor", Status: "pending", EmployeeNo: "D1002", Department: "内科", Title: "住院医师", EmailNotify: true},
		{Name: "系统管理员", Phone: "13700000001", PasswordHash: adminPassword, Role: "admin", Status: "active"},
	}
	for _, user := range users {
		if err := db.Where(models.User{Phone: user.Phone}).Assign(user).FirstOrCreate(&user).Error; err != nil {
			return err
		}
	}

	packages := []models.CheckupPackage{
		{Name: "基础入职体检", Description: "适合入职、入学等基础健康筛查。", Price: 199, Items: "一般检查、血常规、尿常规、肝功能、胸片", Status: "active"},
		{Name: "白领健康套餐", Description: "覆盖常见慢病风险和办公室人群重点指标。", Price: 399, Items: "一般检查、血常规、血脂、血糖、肝肾功能、心电图、腹部彩超", Status: "active"},
		{Name: "全面深度体检", Description: "适合年度全面健康评估。", Price: 899, Items: "基础项目、肿瘤标志物、甲状腺彩超、颈动脉彩超、骨密度", Status: "active"},
	}
	for _, pkg := range packages {
		if err := db.Where(models.CheckupPackage{Name: pkg.Name}).Assign(pkg).FirstOrCreate(&pkg).Error; err != nil {
			return err
		}
	}

	var user models.User
	var doctor models.User
	var pkg models.CheckupPackage
	if err := db.Where("phone = ?", "13800000001").First(&user).Error; err != nil {
		return err
	}
	if err := db.Where("phone = ?", "13900000001").First(&doctor).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "白领健康套餐").First(&pkg).Error; err != nil {
		return err
	}

	if err := seedSlots(db, doctor.ID); err != nil {
		return err
	}

	var slot models.ScheduleSlot
	if err := db.Where("doctor_id = ? AND date = ? AND start_time = ?", doctor.ID, "2026-06-05", "09:00").First(&slot).Error; err != nil {
		return err
	}

	appointment := models.Appointment{
		UserID:    user.ID,
		DoctorID:  doctor.ID,
		SlotID:    slot.ID,
		PackageID: pkg.ID,
		Date:      "2026-06-05",
		Period:    "上午",
		StartTime: "09:00",
		EndTime:   "09:30",
		Status:    "reported",
		Note:      "模拟数据：用户已完成体检。",
	}
	if err := db.Where(models.Appointment{UserID: user.ID, PackageID: pkg.ID, Date: appointment.Date}).FirstOrCreate(&appointment).Error; err != nil {
		return err
	}
	if err := db.Model(&slot).Update("booked_count", 1).Error; err != nil {
		return err
	}

	report := models.Report{
		AppointmentID:  appointment.ID,
		UserID:         user.ID,
		DoctorID:       doctor.ID,
		Summary:        "血常规、肝肾功能、心电图等主要指标未见明显异常。",
		Conclusion:     "总体健康状况良好。",
		Recommendation: "保持规律作息，每周进行 3 次以上中等强度运动，半年后复查血脂。",
	}
	return db.Where(models.Report{AppointmentID: appointment.ID}).FirstOrCreate(&report).Error
}

func seedSlots(db *gorm.DB, doctorID uint) error {
	days := []string{"2026-06-05", "2026-06-06", "2026-06-07", "2026-06-08", "2026-06-09"}
	starts := []string{"08:30", "09:00", "09:30", "10:00", "10:30", "14:00", "14:30", "15:00", "15:30", "16:00"}
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
				DoctorID:    doctorID,
				Date:        day,
				Period:      period,
				StartTime:   start,
				EndTime:     end,
				Capacity:    1,
				BookedCount: 0,
				Status:      "available",
			}
			if err := db.Where(models.ScheduleSlot{DoctorID: doctorID, Date: day, StartTime: start}).FirstOrCreate(&slot).Error; err != nil {
				return err
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
	return fmt.Sprintf("%02d:%02d", parsed.Add(time.Duration(minutes)*time.Minute).Hour(), parsed.Add(time.Duration(minutes)*time.Minute).Minute()), nil
}
