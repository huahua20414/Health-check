package seed

import (
	"health-checkup/backend/internal/models"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	users := []models.User{
		{Name: "张三", Phone: "13800000001", Role: "user"},
		{Name: "李医生", Phone: "13900000001", Role: "doctor"},
	}
	for _, user := range users {
		if err := db.FirstOrCreate(&user, models.User{Phone: user.Phone}).Error; err != nil {
			return err
		}
	}

	packages := []models.CheckupPackage{
		{Name: "基础入职体检", Description: "适合入职、入学等基础健康筛查。", Price: 199, Items: "一般检查、血常规、尿常规、肝功能、胸片"},
		{Name: "白领健康套餐", Description: "覆盖常见慢病风险和办公室人群重点指标。", Price: 399, Items: "一般检查、血常规、血脂、血糖、肝肾功能、心电图、腹部彩超"},
		{Name: "全面深度体检", Description: "适合年度全面健康评估。", Price: 899, Items: "基础项目、肿瘤标志物、甲状腺彩超、颈动脉彩超、骨密度"},
	}
	for _, pkg := range packages {
		if err := db.FirstOrCreate(&pkg, models.CheckupPackage{Name: pkg.Name}).Error; err != nil {
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

	appointment := models.Appointment{
		UserID:    user.ID,
		PackageID: pkg.ID,
		Date:      "2026-06-05",
		Period:    "上午",
		Status:    "reported",
		Note:      "模拟数据：用户已完成体检。",
	}
	if err := db.Where(models.Appointment{UserID: user.ID, PackageID: pkg.ID, Date: appointment.Date}).FirstOrCreate(&appointment).Error; err != nil {
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
