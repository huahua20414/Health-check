package seed

import (
	"time"

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

	for _, permission := range defaultRolePermissions() {
		if err := db.Create(&permission).Error; err != nil {
			return err
		}
	}

	for _, setting := range defaultSystemSettings() {
		if err := db.Create(&setting).Error; err != nil {
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

	items := []models.CheckupItem{
		{Name: "一般检查", Category: "基础检查", Department: "健康管理科", Price: 20, DurationMin: 10, Description: "身高、体重、血压、内外科基础问诊。", Status: "active"},
		{Name: "血常规", Category: "检验", Department: "检验科", Price: 35, DurationMin: 10, Description: "血液细胞基础指标检查。", Status: "active"},
		{Name: "尿常规", Category: "检验", Department: "检验科", Price: 20, DurationMin: 10, Description: "尿液基础指标检查。", Status: "active"},
		{Name: "肝肾功能", Category: "检验", Department: "检验科", Price: 80, DurationMin: 15, Description: "肝功能、肾功能相关指标。", Status: "active"},
		{Name: "心电图", Category: "功能检查", Department: "心电科", Price: 45, DurationMin: 15, Description: "静息心电图检查。", Status: "active"},
		{Name: "腹部彩超", Category: "影像检查", Department: "影像科", Price: 120, DurationMin: 20, Description: "肝胆胰脾肾影像检查。", Status: "active"},
	}
	for _, item := range items {
		if err := db.Create(&item).Error; err != nil {
			return err
		}
	}

	coupons := []models.Coupon{
		{Name: "新客体检立减", Code: "NEW50", Type: "amount", Value: 50, MinAmount: 199, Status: "active", Description: "新用户预约体检可用，结算页展示活动价。"},
		{Name: "年度综合九折", Code: "YEAR10", Type: "percent", Value: 10, MinAmount: 500, Status: "active", Description: "年度综合类套餐活动优惠。"},
	}
	for _, coupon := range coupons {
		if err := db.Create(&coupon).Error; err != nil {
			return err
		}
	}

	now := time.Now()
	announcement := models.SystemAnnouncement{
		Title:       "体检服务预约须知",
		Content:     "请按预约时间携带有效证件到达体检机构，部分抽血项目建议空腹。",
		Audience:    "all",
		Status:      "published",
		PublishedAt: &now,
	}
	if err := db.Create(&announcement).Error; err != nil {
		return err
	}

	return nil
}

func reset(db *gorm.DB) error {
	for _, model := range []any{
		&models.SystemAnnouncement{},
		&models.SystemSetting{},
		&models.Notification{},
		&models.ServiceReview{},
		&models.PackageBrowseHistory{},
		&models.PackageFavorite{},
		&models.PackageItem{},
		&models.CheckupItem{},
		&models.Coupon{},
		&models.MailLog{},
		&models.OperationLog{},
		&models.LoginLog{},
		&models.RolePermission{},
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

func defaultSystemSettings() []models.SystemSetting {
	return []models.SystemSetting{
		{Key: "appointment.reminder_hours", Value: "24", ValueType: "number", Group: "appointment", Label: "体检前提醒小时数", Description: "预约体检前提前多少小时生成提醒。", Status: "active"},
		{Key: "appointment.allow_reschedule_hours", Value: "12", ValueType: "number", Group: "appointment", Label: "改期截止小时数", Description: "距预约开始不足该小时数时不建议改期。", Status: "active"},
		{Key: "notification.in_app_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "站内信通知", Description: "预约、候补和报告生成时发送站内信。", Status: "active"},
		{Key: "notification.sms_mock_enabled", Value: "true", ValueType: "boolean", Group: "notification", Label: "短信模拟通知", Description: "用站内信记录模拟短信触达结果。", Status: "active"},
		{Key: "security.login_code_required", Value: "true", ValueType: "boolean", Group: "security", Label: "登录验证码", Description: "正式环境登录是否要求邮箱验证码。", Status: "active"},
		{Key: "service.customer_service_url", Value: "https://example.com/support", ValueType: "string", Group: "service", Label: "在线客服入口", Description: "用户端客服入口跳转地址。", Status: "active"},
		{Key: "service.customer_service_hours", Value: "08:30-18:00", ValueType: "string", Group: "service", Label: "客服服务时间", Description: "用户端展示的在线客服服务时间。", Status: "active"},
		{Key: "service.faq", Value: `[{"question":"体检前需要注意什么？","answer":"前一天清淡饮食，部分抽血项目建议空腹；请携带有效证件并提前 15 分钟到达。"},{"question":"可以为家人预约吗？","answer":"可以。先在家庭成员中维护家人档案，提交预约时选择对应成员。"},{"question":"预约成功后会有什么提醒？","answer":"系统会生成站内信，并模拟短信通知；邮件通知按 SMTP 配置实际发送。"}]`, ValueType: "json", Group: "service", Label: "常见问题 FAQ", Description: "用户端 FAQ 列表，JSON 数组格式，字段为 question 和 answer。", Status: "active"},
	}
}

func defaultRolePermissions() []models.RolePermission {
	definitions := map[string]string{
		"appointment:create":        "创建体检预约",
		"appointment:reschedule":    "预约改期",
		"appointment:cancel":        "取消预约",
		"review:create":             "评价体检服务",
		"favorite:manage":           "收藏套餐",
		"family:manage":             "管理家庭成员",
		"report:view":               "查看体检报告",
		"doctor:appointment:update": "处理预约状态",
		"report:create":             "生成体检报告",
		"customer:view":             "查看客户档案",
		"admin:user:manage":         "管理用户状态",
		"admin:doctor:review":       "审核医生账号",
		"admin:package:manage":      "管理体检套餐",
		"admin:resource:manage":     "管理项目和排班",
		"admin:operation:manage":    "管理优惠券、评价和公告",
		"admin:system:manage":       "管理系统设置和日志",
		"admin:data:exchange":       "导入导出业务数据",
		"admin:permission:manage":   "管理角色权限",
	}
	roles := map[string][]string{
		"user": {
			"appointment:create", "appointment:reschedule", "appointment:cancel", "review:create",
			"favorite:manage", "family:manage", "report:view",
		},
		"doctor": {"doctor:appointment:update", "report:create", "customer:view"},
		"admin": {
			"admin:user:manage", "admin:doctor:review", "admin:package:manage", "admin:resource:manage",
			"admin:operation:manage", "admin:system:manage", "admin:data:exchange", "admin:permission:manage",
		},
	}
	var permissions []models.RolePermission
	for role, codes := range roles {
		for _, code := range codes {
			permissions = append(permissions, models.RolePermission{Role: role, Permission: code, Description: definitions[code], Enabled: true})
		}
	}
	return permissions
}
