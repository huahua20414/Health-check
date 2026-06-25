package seed

import (
	"fmt"
	"time"

	"health-checkup/backend/internal/models"

	"gorm.io/gorm"
)

const shortcutEmail = "huahua20414@foxmail.com"

func EnsureFutureScheduleSlots(db *gorm.DB, now time.Time, days int) (int, error) {
	if days <= 0 {
		days = 14
	}
	var doctors []models.User
	if err := db.Where("role = ? AND status = ?", "doctor", "active").Find(&doctors).Error; err != nil {
		return 0, err
	}
	var institutions []models.CheckupInstitution
	if err := db.Where("status = ?", "active").Find(&institutions).Error; err != nil {
		return 0, err
	}
	var packages []models.CheckupPackage
	if err := db.Where("status = ?", "active").Find(&packages).Error; err != nil {
		return 0, err
	}
	if len(doctors) == 0 || len(institutions) == 0 || len(packages) == 0 {
		return 0, nil
	}
	slots := futureScheduleSlots(now, days, doctors, institutions, packages)
	created := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, slot := range slots {
			exists, err := scheduleSlotExists(tx, slot)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return created, nil
}

func Run(db *gorm.DB) error {
	if err := reset(db); err != nil {
		return err
	}

	admin := models.User{
		Name: "系统管理员", Phone: "A1001", Email: shortcutEmail, Role: "admin", Status: "active", EmailNotify: false,
	}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}

	permissions := defaultRolePermissions()
	if err := db.Create(&permissions).Error; err != nil {
		return err
	}

	settings := defaultSystemSettings()
	if err := db.Create(&settings).Error; err != nil {
		return err
	}

	institutions := []models.CheckupInstitution{
		{Name: "熙心健康体检中心总院", Address: "沈阳市和平区健康路 88 号", Phone: "024-88880001", OpenHours: "周一至周六 08:00-17:00", Status: "active"},
		{Name: "熙心健康高新区分院", Address: "沈阳市浑南区创新路 19 号", Phone: "024-88880002", OpenHours: "周一至周五 08:30-16:30", Status: "active"},
		{Name: "熙心健康河西分院", Address: "沈阳市铁西区卫工街 26 号", Phone: "024-88880003", OpenHours: "周一至周六 08:30-16:30", Status: "active"},
	}
	if err := db.Create(&institutions).Error; err != nil {
		return err
	}

	packages := []models.CheckupPackage{
		{Name: "入职基础体检", Category: "入职体检", Description: "适合入职、入学、资格审查等基础健康筛查。", Price: 199, Items: "一般检查、血常规、尿常规、肝功能、胸片", Status: "active"},
		{Name: "慢病风险筛查", Category: "慢病筛查", Description: "覆盖血糖、血脂、肝肾功能和心血管慢病风险。", Price: 399, Items: "一般检查、血常规、血脂、血糖、肝肾功能、心电图", Status: "active"},
		{Name: "年度综合体检", Category: "年度综合", Description: "适合年度全面健康评估，多科室协同完成。", Price: 899, Items: "基础项目、肿瘤标志物、甲状腺彩超、颈动脉彩超、骨密度", Status: "active"},
		{Name: "影像专项检查", Category: "影像专项", Description: "适合关注肺部、腹部、甲状腺、乳腺等影像检查人群。", Price: 499, Items: "胸部影像、腹部彩超、甲状腺彩超、乳腺彩超", Status: "active"},
		{Name: "女性专项体检", Category: "女性专项", Description: "面向女性健康管理，覆盖乳腺、甲状腺和妇科基础筛查。", Price: 599, Items: "妇科基础检查、乳腺彩超、甲状腺彩超、血常规", Status: "active"},
		{Name: "老年健康评估", Category: "老年体检", Description: "关注老年慢病、骨密度、心脑血管和生活方式风险。", Price: 699, Items: "血糖血脂、肝肾功能、心电图、骨密度、颈动脉彩超", Status: "active"},
	}
	if err := db.Create(&packages).Error; err != nil {
		return err
	}

	items := []models.CheckupItem{
		{Name: "一般检查", Category: "基础检查", Department: "健康管理科", Price: 20, DurationMin: 10, Description: "身高、体重、血压、内外科基础问诊。", Status: "active"},
		{Name: "血常规", Category: "检验", Department: "检验科", Price: 35, DurationMin: 10, Description: "血液细胞基础指标检查。", Status: "active"},
		{Name: "尿常规", Category: "检验", Department: "检验科", Price: 20, DurationMin: 10, Description: "尿液基础指标检查。", Status: "active"},
		{Name: "肝肾功能", Category: "检验", Department: "检验科", Price: 80, DurationMin: 15, Description: "肝功能、肾功能相关指标。", Status: "active"},
		{Name: "心电图", Category: "功能检查", Department: "心电科", Price: 45, DurationMin: 15, Description: "静息心电图检查。", Status: "active"},
		{Name: "腹部彩超", Category: "影像检查", Department: "影像科", Price: 120, DurationMin: 20, Description: "肝胆胰脾肾影像检查。", Status: "active"},
	}
	if err := db.Create(&items).Error; err != nil {
		return err
	}
	if err := seedPackageItems(db, packages, items); err != nil {
		return err
	}

	coupons := []models.Coupon{
		{Name: "新客体检立减", Code: "NEW50", Type: "amount", Value: 50, MinAmount: 199, Status: "active", Description: "新用户预约体检可用，结算页展示活动价。"},
		{Name: "年度综合九折", Code: "YEAR10", Type: "percent", Value: 10, MinAmount: 500, Status: "active", Description: "年度综合类套餐活动优惠。"},
	}
	if err := db.Create(&coupons).Error; err != nil {
		return err
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

	if err := seedDemoBusinessData(db, now, institutions, packages, coupons); err != nil {
		return err
	}

	return nil
}

func seedDemoBusinessData(db *gorm.DB, now time.Time, institutions []models.CheckupInstitution, packages []models.CheckupPackage, coupons []models.Coupon) error {
	users := demoUsers()
	if err := db.Create(&users).Error; err != nil {
		return err
	}
	doctors := demoDoctors()
	if err := db.Create(&doctors).Error; err != nil {
		return err
	}
	if err := seedFamilyMembers(db, users); err != nil {
		return err
	}

	slots := demoScheduleSlots(now, doctors, institutions, packages)
	if err := db.Create(&slots).Error; err != nil {
		return err
	}
	if err := seedAppointments(db, users, packages, coupons, slots, now); err != nil {
		return err
	}
	if err := seedWaitlist(db, users, institutions, packages, slots, now); err != nil {
		return err
	}
	return seedEngagement(db, users, packages)
}

func demoUsers() []models.User {
	names := []string{
		"李明", "王芳", "张伟", "刘洋", "陈静", "赵磊", "孙悦", "周敏",
		"吴昊", "郑洁", "冯晨", "蒋欣", "韩雪", "马骏", "朱琳", "胡宁",
		"郭佳", "何雨", "罗成", "高娜", "宋凯", "谢婷", "唐宇", "邓璐",
		"曹阳", "袁媛", "潘杰", "杜娟", "程远", "魏然", "苏晴", "梁峰",
	}
	users := []models.User{{
		Name: "快捷用户", Phone: "U1001", Email: shortcutEmail, Role: "user", Status: "active",
		Gender: "男", Age: 29, IDCard: demoIDCard("210102", 1997, 5, 12, 100), EmailNotify: false,
	}}
	for i, name := range names {
		index := i + 1
		gender := "男"
		if index%2 == 0 {
			gender = "女"
		}
		status := "active"
		if index == 29 {
			status = "disabled"
		}
		users = append(users, models.User{
			Name:        name,
			Phone:       fmt.Sprintf("1390001%04d", index),
			Role:        "user",
			Status:      status,
			Gender:      gender,
			Age:         24 + index%33,
			IDCard:      demoIDCard("210102", 1970+index%25, index%12+1, index%27+1, index),
			Email:       fmt.Sprintf("demo.user%02d@example.com", index),
			Bio:         "用于本地演示的体检客户档案。",
			EmailNotify: index%4 != 0,
		})
	}
	return users
}

func demoDoctors() []models.User {
	definitions := []struct {
		name        string
		department  string
		title       string
		specialties string
		status      string
	}{
		{"林主任", "健康管理科", "主任医师", "年度综合评估、慢病风险管理", "active"},
		{"许医生", "健康管理科", "副主任医师", "入职体检、报告解读", "active"},
		{"邱医生", "健康管理科", "主治医师", "老年健康评估、生活方式干预", "active"},
		{"沈医生", "健康管理科", "主治医师", "团检流程、基础筛查", "active"},
		{"丁医生", "健康管理科", "医师", "个人体检、复查建议", "active"},
		{"叶医生", "健康管理科", "医师", "慢病随访、健康咨询", "active"},
		{"范医生", "检验科", "副主任技师", "血液检验、生化检验", "active"},
		{"任医生", "检验科", "主管技师", "肝肾功能、血糖血脂", "active"},
		{"姚医生", "检验科", "技师", "尿常规、血常规", "active"},
		{"夏医生", "检验科", "技师", "检验质量控制", "active"},
		{"顾医生", "影像科", "主任医师", "腹部彩超、甲状腺彩超", "active"},
		{"石医生", "影像科", "副主任医师", "胸部影像、乳腺彩超", "active"},
		{"陆医生", "影像科", "主治医师", "血管超声、骨密度评估", "active"},
		{"白医生", "心电科", "主治医师", "静息心电图、心血管风险", "active"},
		{"常医生", "心电科", "医师", "心电图复核、运动建议", "active"},
		{"孟医生", "内科", "副主任医师", "内科问诊、慢病联合评估", "active"},
		{"方医生", "妇科", "主治医师", "女性专项筛查、乳腺评估", "active"},
		{"康医生", "眼科", "医师", "视力筛查、眼底评估", "active"},
		{"试用医生甲", "健康管理科", "医师", "待审核医生资料", "pending"},
		{"试用医生乙", "影像科", "医师", "待审核影像医生资料", "pending"},
	}
	doctors := []models.User{{
		Name: "快捷医生", Phone: "D1001", Email: shortcutEmail, Role: "doctor", Status: "active",
		Gender: "女", Age: 36, EmployeeNo: "DOC1001", Department: "健康管理科", Title: "主治医师",
		Specialties: "年度综合评估、报告解读", EmailNotify: false,
	}}
	for i, item := range definitions {
		index := i + 1
		doctors = append(doctors, models.User{
			Name:        item.name,
			Phone:       fmt.Sprintf("D%04d", index),
			Role:        "doctor",
			Status:      item.status,
			Gender:      doctorGender(index),
			Age:         30 + index%22,
			Email:       fmt.Sprintf("demo.doctor%02d@example.com", index),
			EmployeeNo:  fmt.Sprintf("DOC%04d", index),
			Department:  item.department,
			Title:       item.title,
			Specialties: item.specialties,
			EmailNotify: true,
		})
	}
	return doctors
}

func doctorGender(index int) string {
	if index%3 == 0 {
		return "女"
	}
	return "男"
}

func demoIDCard(areaCode string, year, month, day, sequence int) string {
	base := fmt.Sprintf("%s%04d%02d%02d%03d", areaCode, year, month, day, sequence%1000)
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i, weight := range weights {
		sum += int(base[i]-'0') * weight
	}
	return base + string(checkCodes[sum%11])
}

func seedFamilyMembers(db *gorm.DB, users []models.User) error {
	members := make([]models.FamilyMember, 0, 18)
	relations := []string{"父亲", "母亲", "配偶", "子女"}
	for i := 0; i < 18 && i < len(users); i++ {
		index := i + 1
		members = append(members, models.FamilyMember{
			UserID:   users[i].ID,
			Name:     fmt.Sprintf("%s家属", users[i].Name),
			Relation: relations[i%len(relations)],
			Gender:   []string{"男", "女"}[i%2],
			Age:      18 + index%45,
			IDCard:   demoIDCard("210103", 1960+index%35, index%12+1, index%27+1, index+100),
			Phone:    fmt.Sprintf("1380002%04d", index),
			Status:   "active",
		})
	}
	return db.Create(&members).Error
}

func seedPackageItems(db *gorm.DB, packages []models.CheckupPackage, items []models.CheckupItem) error {
	itemByName := make(map[string]models.CheckupItem, len(items))
	for _, item := range items {
		itemByName[item.Name] = item
	}
	definitions := map[string][]string{
		"入职基础体检": {"一般检查", "血常规", "尿常规", "肝肾功能"},
		"慢病风险筛查": {"一般检查", "血常规", "肝肾功能", "心电图"},
		"年度综合体检": {"一般检查", "血常规", "尿常规", "肝肾功能", "心电图", "腹部彩超"},
		"影像专项检查": {"一般检查", "腹部彩超", "心电图"},
		"女性专项体检": {"一般检查", "血常规", "尿常规", "腹部彩超"},
		"老年健康评估": {"一般检查", "血常规", "肝肾功能", "心电图", "腹部彩超"},
	}
	var links []models.PackageItem
	for _, pkg := range packages {
		names := definitions[pkg.Name]
		if len(names) == 0 {
			names = []string{"一般检查", "血常规", "肝肾功能"}
		}
		for index, name := range names {
			item, ok := itemByName[name]
			if !ok {
				continue
			}
			links = append(links, models.PackageItem{
				PackageID: pkg.ID,
				ItemID:    item.ID,
				SortOrder: index + 1,
				Required:  index < 3,
			})
		}
	}
	return db.Create(&links).Error
}

func demoScheduleSlots(now time.Time, doctors []models.User, institutions []models.CheckupInstitution, packages []models.CheckupPackage) []models.ScheduleSlot {
	return futureScheduleSlots(now, 14, filterActiveDoctors(doctors), institutions, packages)
}

func futureScheduleSlots(now time.Time, days int, activeDoctors []models.User, institutions []models.CheckupInstitution, packages []models.CheckupPackage) []models.ScheduleSlot {
	categories := packageCategories(packages)
	rules := demoDoctorShiftRules(activeDoctors, institutions, categories)
	startDate := dayStart(now)
	slots := make([]models.ScheduleSlot, 0, len(rules)*days*15)
	usedDoctorTimes := make(map[string]bool)
	for offset := 0; offset < days; offset++ {
		day := startDate.AddDate(0, 0, offset)
		date := day.Format("2006-01-02")
		weekday := int(day.Weekday())
		for _, rule := range rules {
			if !rule.Workdays[weekday] {
				continue
			}
			for _, start := range shiftTimes(rule.Shift) {
				key := fmt.Sprintf("%d|%s|%s", rule.Doctor.ID, date, start)
				if usedDoctorTimes[key] {
					continue
				}
				usedDoctorTimes[key] = true
				slots = append(slots, models.ScheduleSlot{
					DoctorID:      rule.Doctor.ID,
					InstitutionID: rule.Institution.ID,
					Date:          date,
					Period:        periodForStart(start),
					Category:      rule.Category,
					StartTime:     start,
					EndTime:       addHalfHour(start),
					Capacity:      1,
					BookedCount:   0,
					Status:        "available",
				})
			}
		}
	}
	return appendFutureCoverageSlots(slots, activeDoctors, institutions, categories, startDate, days, usedDoctorTimes)
}

type doctorShiftRule struct {
	Doctor      models.User
	Institution models.CheckupInstitution
	Category    string
	Shift       string
	Workdays    map[int]bool
}

func demoDoctorShiftRules(doctors []models.User, institutions []models.CheckupInstitution, categories []string) []doctorShiftRule {
	rules := make([]doctorShiftRule, 0, len(doctors)*2)
	categoryWeights := map[string]int{
		"健康管理科": 0,
		"内科":    1,
		"检验科":   1,
		"影像科":   3,
		"妇科":    4,
		"老年医学科": 5,
		"心电科":   2,
	}
	for i, doctor := range doctors {
		categoryIndex, ok := categoryWeights[doctor.Department]
		if !ok {
			categoryIndex = i
		}
		category := categories[categoryIndex%len(categories)]
		institution := institutions[(i+i/3)%len(institutions)]
		rules = append(rules, doctorShiftRule{
			Doctor:      doctor,
			Institution: institution,
			Category:    category,
			Shift:       "上午",
			Workdays:    workdaySet((i+1)%7, (i+3)%7, (i+5)%7),
		})
		if i%3 != 1 {
			rules = append(rules, doctorShiftRule{
				Doctor:      doctor,
				Institution: institutions[(i+1)%len(institutions)],
				Category:    categories[(categoryIndex+i+2)%len(categories)],
				Shift:       "下午",
				Workdays:    workdaySet((i+2)%7, (i+4)%7),
			})
		}
	}
	return rules
}

func workdaySet(days ...int) map[int]bool {
	result := make(map[int]bool, len(days))
	for _, day := range days {
		result[day] = true
	}
	return result
}

func shiftTimes(shift string) []string {
	if shift == "下午" {
		return []string{"13:30", "14:00", "14:30", "15:00", "15:30", "16:00", "16:30"}
	}
	return []string{"08:00", "08:30", "09:00", "09:30", "10:00", "10:30", "11:00", "11:30"}
}

func filterActiveDoctors(doctors []models.User) []models.User {
	active := make([]models.User, 0, len(doctors))
	for _, doctor := range doctors {
		if doctor.Status == "active" {
			active = append(active, doctor)
		}
	}
	return active
}

func packageCategories(packages []models.CheckupPackage) []string {
	seen := map[string]bool{}
	var categories []string
	for _, pkg := range packages {
		if !seen[pkg.Category] {
			seen[pkg.Category] = true
			categories = append(categories, pkg.Category)
		}
	}
	return categories
}

func appendFutureCoverageSlots(slots []models.ScheduleSlot, doctors []models.User, institutions []models.CheckupInstitution, categories []string, startDate time.Time, days int, usedDoctorTimes map[string]bool) []models.ScheduleSlot {
	allTimes := []string{"08:00", "08:30", "09:00", "09:30", "10:00", "10:30", "11:00", "11:30", "13:30", "14:00", "14:30", "15:00", "15:30", "16:00", "16:30"}
	exists := make(map[string]bool, len(slots))
	for _, slot := range slots {
		exists[fmt.Sprintf("%s|%s", slot.Date, slot.StartTime)] = true
	}
	cursor := 0
	for offset := 0; offset < days; offset++ {
		date := startDate.AddDate(0, 0, offset).Format("2006-01-02")
		for timeIndex, start := range allTimes {
			if exists[fmt.Sprintf("%s|%s", date, start)] {
				continue
			}
			doctor := nextAvailableDoctor(doctors, &cursor, usedDoctorTimes, date, start)
			slots = append(slots, models.ScheduleSlot{
				DoctorID:      doctor.ID,
				InstitutionID: institutions[(offset+timeIndex)%len(institutions)].ID,
				Date:          date,
				Period:        periodForStart(start),
				Category:      categories[(offset+timeIndex)%len(categories)],
				StartTime:     start,
				EndTime:       addHalfHour(start),
				Capacity:      1,
				BookedCount:   0,
				Status:        "available",
			})
		}
	}
	coverageTimes := []string{"10:00", "11:00", "13:30", "15:00", "16:00", "16:30"}
	for institutionIndex, institution := range institutions {
		for categoryIndex, category := range categories {
			start := coverageTimes[(institutionIndex+categoryIndex)%len(coverageTimes)]
			offset := 5 + institutionIndex + categoryIndex/len(coverageTimes)
			if offset >= days {
				offset = days - 1
			}
			date := startDate.AddDate(0, 0, offset).Format("2006-01-02")
			doctor := nextAvailableDoctor(doctors, &cursor, usedDoctorTimes, date, start)
			slots = append(slots, models.ScheduleSlot{
				DoctorID:      doctor.ID,
				InstitutionID: institution.ID,
				Date:          date,
				Period:        periodForStart(start),
				Category:      category,
				StartTime:     start,
				EndTime:       addHalfHour(start),
				Capacity:      1,
				BookedCount:   0,
				Status:        "available",
			})
		}
	}
	return slots
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func scheduleSlotExists(db *gorm.DB, slot models.ScheduleSlot) (bool, error) {
	var count int64
	err := db.Model(&models.ScheduleSlot{}).
		Where("doctor_id = ? AND date = ? AND start_time = ?", slot.DoctorID, slot.Date, slot.StartTime).
		Count(&count).Error
	if err != nil || count > 0 {
		return count > 0, err
	}
	err = db.Model(&models.ScheduleSlot{}).
		Where("doctor_id = ? AND date = ? AND status <> ? AND start_time < ? AND end_time > ?", slot.DoctorID, slot.Date, "deleted", slot.EndTime, slot.StartTime).
		Count(&count).Error
	return count > 0, err
}

func nextAvailableDoctor(doctors []models.User, cursor *int, usedDoctorTimes map[string]bool, date, start string) models.User {
	for attempts := 0; attempts < len(doctors)*2; attempts++ {
		doctor := doctors[*cursor%len(doctors)]
		*cursor += 3
		key := fmt.Sprintf("%d|%s|%s", doctor.ID, date, start)
		if usedDoctorTimes[key] {
			continue
		}
		usedDoctorTimes[key] = true
		return doctor
	}
	doctor := doctors[*cursor%len(doctors)]
	*cursor++
	return doctor
}

func periodForStart(start string) string {
	if start >= "12:00" {
		return "下午"
	}
	return "上午"
}

func addHalfHour(start string) string {
	parsed, err := time.Parse("15:04", start)
	if err != nil {
		return start
	}
	return parsed.Add(30 * time.Minute).Format("15:04")
}

func seedAppointments(db *gorm.DB, users []models.User, packages []models.CheckupPackage, coupons []models.Coupon, slots []models.ScheduleSlot, now time.Time) error {
	packageByCategory := packagesByCategory(packages)
	activeUsers := filterActiveUsers(users)
	appointmentIndex := 0
	reportIndex := 0
	reviewIndex := 0
	for i := range slots {
		slot := &slots[i]
		if !shouldBookSlot(*slot, now) || appointmentIndex >= 86 {
			continue
		}
		pkg := packageByCategory[slot.Category]
		user := activeUsers[appointmentIndex%len(activeUsers)]
		status := appointmentStatusForSlot(*slot, now, appointmentIndex)
		paymentStatus := "unpaid"
		if appointmentIndex%3 != 0 {
			paymentStatus = "paid"
		}
		invoiceStatus := "none"
		invoiceTitle := ""
		invoiceTaxNo := ""
		if appointmentIndex%7 == 0 {
			invoiceStatus = "requested"
			invoiceTitle = user.Name
			invoiceTaxNo = fmt.Sprintf("TAX%08d", appointmentIndex+1)
		}
		couponID := uint(0)
		discount := 0.0
		if len(coupons) > 0 && appointmentIndex%5 == 0 && pkg.Price >= coupons[0].MinAmount {
			couponID = coupons[0].ID
			discount = coupons[0].Value
		}
		appointment := models.Appointment{
			OrderNo:         fmt.Sprintf("HCSEED%06d", appointmentIndex+1),
			UserID:          user.ID,
			DoctorID:        slot.DoctorID,
			InstitutionID:   slot.InstitutionID,
			SlotID:          slot.ID,
			PackageID:       pkg.ID,
			CouponID:        couponID,
			AppointmentType: []string{"个人体检", "复查体检", "入职体检", "家人体检"}[appointmentIndex%4],
			Category:        pkg.Category,
			Date:            slot.Date,
			Period:          slot.Period,
			StartTime:       slot.StartTime,
			EndTime:         slot.EndTime,
			Status:          status,
			Note:            appointmentNote(appointmentIndex),
			PaymentStatus:   paymentStatus,
			OriginalAmount:  pkg.Price,
			DiscountAmount:  discount,
			PayableAmount:   pkg.Price - discount,
			InvoiceTitle:    invoiceTitle,
			InvoiceTaxNo:    invoiceTaxNo,
			InvoiceStatus:   invoiceStatus,
		}
		if err := db.Create(&appointment).Error; err != nil {
			return err
		}
		if err := seedAppointmentItems(db, appointment, appointmentIndex); err != nil {
			return err
		}
		if err := db.Model(slot).Update("booked_count", 1).Error; err != nil {
			return err
		}
		slot.BookedCount = 1
		if status == "reported" || (status == "checked" && reportIndex%3 == 0) {
			reportIndex++
			if err := seedReport(db, appointment, reportIndex); err != nil {
				return err
			}
		}
		if (status == "reported" || status == "checked") && reviewIndex < 18 && appointmentIndex%2 == 0 {
			reviewIndex++
			if err := seedReview(db, appointment, reviewIndex); err != nil {
				return err
			}
		}
		appointmentIndex++
	}
	return nil
}

func shouldBookSlot(slot models.ScheduleSlot, now time.Time) bool {
	slotDate, err := time.Parse("2006-01-02", slot.Date)
	if err == nil && !slotDate.After(now) {
		return true
	}
	if slot.StartTime == "09:30" || slot.StartTime == "14:00" {
		return true
	}
	if slot.StartTime == "08:30" && slot.Category == "入职体检" {
		return true
	}
	return slot.StartTime == "10:30" && slot.DoctorID%2 == 0
}

func appointmentStatusForSlot(slot models.ScheduleSlot, now time.Time, index int) string {
	slotDate, err := time.Parse("2006-01-02", slot.Date)
	if err == nil && slotDate.Before(now) {
		if index%3 == 0 {
			return "reported"
		}
		return "checked"
	}
	if index%11 == 0 {
		return "checked"
	}
	return "booked"
}

func appointmentNote(index int) string {
	notes := []string{
		"希望尽量安排靠前时段。",
		"近期睡眠一般，想重点关注慢病风险。",
		"单位团检成员，需开具发票。",
		"为家人预约，请协助现场引导。",
		"复查血脂和肝功能指标。",
		"",
	}
	return notes[index%len(notes)]
}

func seedAppointmentItems(db *gorm.DB, appointment models.Appointment, index int) error {
	var links []models.PackageItem
	if err := db.Preload("Item").Where("package_id = ?", appointment.PackageID).Order("sort_order asc, id asc").Find(&links).Error; err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}
	items := make([]models.AppointmentItem, 0, len(links))
	for _, link := range links {
		if !link.Required && (index+int(link.ID))%3 != 0 {
			continue
		}
		price := 0.0
		if !link.Required {
			price = link.Item.Price
		}
		items = append(items, models.AppointmentItem{
			AppointmentID: appointment.ID,
			PackageItemID: link.ID,
			ItemID:        link.ItemID,
			Required:      link.Required,
			Price:         price,
		})
	}
	return db.Create(&items).Error
}

func packagesByCategory(packages []models.CheckupPackage) map[string]models.CheckupPackage {
	result := make(map[string]models.CheckupPackage, len(packages))
	for _, pkg := range packages {
		if _, ok := result[pkg.Category]; !ok {
			result[pkg.Category] = pkg
		}
	}
	return result
}

func filterActiveUsers(users []models.User) []models.User {
	active := make([]models.User, 0, len(users))
	for _, user := range users {
		if user.Status == "active" {
			active = append(active, user)
		}
	}
	return active
}

func seedReport(db *gorm.DB, appointment models.Appointment, index int) error {
	report := models.Report{
		ReportNo:       fmt.Sprintf("RPTSEED%06d", index),
		AppointmentID:  appointment.ID,
		UserID:         appointment.UserID,
		DoctorID:       appointment.DoctorID,
		Summary:        "本次体检基础项目已完成，主要指标用于演示报告流程。",
		Conclusion:     []string{"总体指标稳定", "建议复查血脂", "注意血压和体重管理"}[index%3],
		Recommendation: []string{"保持规律作息，三个月后复查。", "清淡饮食并增加有氧运动。", "如有不适请及时到专科门诊就诊。"}[index%3],
	}
	return db.Create(&report).Error
}

func seedReview(db *gorm.DB, appointment models.Appointment, index int) error {
	review := models.ServiceReview{
		UserID:        appointment.UserID,
		AppointmentID: appointment.ID,
		PackageID:     appointment.PackageID,
		InstitutionID: appointment.InstitutionID,
		DoctorID:      appointment.DoctorID,
		Rating:        3 + index%3,
		Content:       []string{"流程清楚，等候时间可以再短一些。", "医生解释比较耐心，报告内容清楚。", "预约和现场引导都比较顺畅。"}[index%3],
		Status:        "published",
	}
	if index%4 == 0 {
		review.Reply = "感谢反馈，我们会继续优化现场排队和引导。"
	}
	return db.Create(&review).Error
}

func seedWaitlist(db *gorm.DB, users []models.User, institutions []models.CheckupInstitution, packages []models.CheckupPackage, slots []models.ScheduleSlot, now time.Time) error {
	activeUsers := filterActiveUsers(users)
	waitingSlots := selectFullSlots(slots, 10)
	entries := make([]models.WaitlistEntry, 0, len(waitingSlots)+4)
	for i, slot := range waitingSlots {
		pkg := packageForCategory(packages, slot.Category)
		entries = append(entries, models.WaitlistEntry{
			UserID:          activeUsers[(i+11)%len(activeUsers)].ID,
			PackageID:       pkg.ID,
			InstitutionID:   slot.InstitutionID,
			AppointmentType: "个人体检",
			Category:        slot.Category,
			Date:            slot.Date,
			Period:          slot.Period,
			StartTime:       slot.StartTime,
			EndTime:         slot.EndTime,
			Note:            "热门时段已满，加入候补队列。",
			Status:          "waiting",
		})
	}
	for i := 0; i < 4; i++ {
		pkg := packages[(i+2)%len(packages)]
		entries = append(entries, models.WaitlistEntry{
			UserID:          activeUsers[(i+21)%len(activeUsers)].ID,
			PackageID:       pkg.ID,
			InstitutionID:   institutions[i%len(institutions)].ID,
			AppointmentType: "家人体检",
			Category:        pkg.Category,
			Date:            now.AddDate(0, 0, i+3).Format("2006-01-02"),
			Period:          []string{"上午", "下午"}[i%2],
			Note:            "可接受系统自动递补。",
			Status:          []string{"waiting", "canceled", "waiting", "promoted"}[i],
		})
	}
	return db.Create(&entries).Error
}

func selectFullSlots(slots []models.ScheduleSlot, limit int) []models.ScheduleSlot {
	selected := make([]models.ScheduleSlot, 0, limit)
	for _, slot := range slots {
		if slot.BookedCount >= slot.Capacity && (slot.StartTime == "09:30" || slot.StartTime == "14:00") {
			selected = append(selected, slot)
			if len(selected) == limit {
				break
			}
		}
	}
	return selected
}

func packageForCategory(packages []models.CheckupPackage, category string) models.CheckupPackage {
	for _, pkg := range packages {
		if pkg.Category == category {
			return pkg
		}
	}
	return packages[0]
}

func seedEngagement(db *gorm.DB, users []models.User, packages []models.CheckupPackage) error {
	var favorites []models.PackageFavorite
	var histories []models.PackageBrowseHistory
	var tickets []models.SupportTicket
	activeUsers := filterActiveUsers(users)
	for i, user := range activeUsers {
		if i%2 == 0 {
			favorites = append(favorites, models.PackageFavorite{UserID: user.ID, PackageID: packages[i%len(packages)].ID})
		}
		histories = append(histories, models.PackageBrowseHistory{
			UserID:    user.ID,
			PackageID: packages[(i+1)%len(packages)].ID,
			ViewCount: 1 + i%7,
			ViewedAt:  time.Now().Add(-time.Duration(i%96) * time.Hour),
		})
		if i < 8 {
			tickets = append(tickets, models.SupportTicket{
				UserID:  user.ID,
				Subject: []string{"预约时间咨询", "发票信息修改", "报告解读咨询", "候补递补问题"}[i%4],
				Content: "本条工单用于本地演示客服处理流程。",
				Reply:   []string{"", "已收到，工作人员会协助处理。"}[i%2],
				Status:  []string{"open", "replied", "closed", "open"}[i%4],
			})
		}
	}
	if err := db.Create(&favorites).Error; err != nil {
		return err
	}
	if err := db.Create(&histories).Error; err != nil {
		return err
	}
	if err := db.Create(&tickets).Error; err != nil {
		return err
	}
	return nil
}

func reset(db *gorm.DB) error {
	for _, model := range []any{
		&models.SystemAnnouncement{},
		&models.SupportTicket{},
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
		&models.AppointmentItem{},
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
		"appointment:pay":           "模拟预约支付",
		"appointment:invoice":       "维护预约发票信息",
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
		"admin:notification:manage": "管理消息通知",
		"admin:system:manage":       "管理系统设置和日志",
		"admin:data:exchange":       "导入导出业务数据",
		"admin:permission:manage":   "管理角色权限",
	}
	roles := map[string][]string{
		"user": {
			"appointment:create", "appointment:reschedule", "appointment:cancel", "review:create",
			"appointment:pay", "appointment:invoice", "favorite:manage", "family:manage", "report:view",
		},
		"doctor": {"doctor:appointment:update", "report:create", "customer:view"},
		"admin": {
			"admin:user:manage", "admin:doctor:review", "admin:package:manage", "admin:resource:manage",
			"admin:operation:manage", "admin:notification:manage", "admin:system:manage", "admin:data:exchange", "admin:permission:manage",
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
