package seed

import (
	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/models"

	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	if err := reset(db); err != nil {
		return err
	}

	adminPassword, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}

	admin := models.User{
		Name: "系统管理员", Phone: "A1001", Email: "huahua20414@foxmail.com", PasswordHash: adminPassword, Role: "admin", Status: "active", EmailNotify: false,
	}
	if err := db.Create(&admin).Error; err != nil {
		return err
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
		{Name: "入职基础体检", Category: "入职体检", Description: "适合入职、入学、资格审查等基础健康筛查。", Price: 199, Items: "一般检查、血常规、尿常规、肝功能、胸片", Status: "active"},
		{Name: "慢病风险筛查", Category: "慢病筛查", Description: "覆盖血糖、血脂、肝肾功能和心血管慢病风险。", Price: 399, Items: "一般检查、血常规、血脂、血糖、肝肾功能、心电图", Status: "active"},
		{Name: "年度综合体检", Category: "年度综合", Description: "适合年度全面健康评估，多科室协同完成。", Price: 899, Items: "基础项目、肿瘤标志物、甲状腺彩超、颈动脉彩超、骨密度", Status: "active"},
		{Name: "影像专项检查", Category: "影像专项", Description: "适合关注肺部、腹部、甲状腺、乳腺等影像检查人群。", Price: 499, Items: "胸部影像、腹部彩超、甲状腺彩超、乳腺彩超", Status: "active"},
		{Name: "女性专项体检", Category: "女性专项", Description: "面向女性健康管理，覆盖乳腺、甲状腺和妇科基础筛查。", Price: 599, Items: "妇科基础检查、乳腺彩超、甲状腺彩超、血常规", Status: "active"},
		{Name: "老年健康评估", Category: "老年体检", Description: "关注老年慢病、骨密度、心脑血管和生活方式风险。", Price: 699, Items: "血糖血脂、肝肾功能、心电图、骨密度、颈动脉彩超", Status: "active"},
	}
	for _, pkg := range packages {
		if err := db.Create(&pkg).Error; err != nil {
			return err
		}
	}

	return nil
}

func reset(db *gorm.DB) error {
	for _, model := range []any{
		&models.Notification{},
		&models.PackageBrowseHistory{},
		&models.PackageFavorite{},
		&models.MailLog{},
		&models.Report{},
		&models.Appointment{},
		&models.WaitlistEntry{},
		&models.ScheduleSlot{},
		&models.FamilyMember{},
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
