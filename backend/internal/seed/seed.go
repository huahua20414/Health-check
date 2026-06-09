package seed

import (
	"fmt"
	"strings"
	"time"

	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/models"

	"gorm.io/gorm"
)

type doctorSeed struct {
	Name        string
	Email       string
	EmployeeNo  string
	Department  string
	Title       string
	Status      string
	Specialties []string
}

func doctorSeeds() []doctorSeed {
	return []doctorSeed{
		{Name: "李医生", Email: "huahua20414@foxmail.com", EmployeeNo: "D1001", Department: "健康管理科", Title: "主治医师", Status: "active", Specialties: []string{"入职体检", "年度综合"}},
		{Name: "王医生", Email: "wangdoctor@example.com", EmployeeNo: "D1002", Department: "内科", Title: "副主任医师", Status: "active", Specialties: []string{"慢病筛查", "老年体检", "年度综合"}},
		{Name: "赵医生", Email: "zhaodoctor@example.com", EmployeeNo: "D1003", Department: "影像科", Title: "主治医师", Status: "active", Specialties: []string{"影像专项", "女性专项"}},
		{Name: "陈医生", Email: "chendoctor@example.com", EmployeeNo: "D1004", Department: "健康管理科", Title: "副主任医师", Status: "active", Specialties: []string{"入职体检", "年度综合"}},
		{Name: "刘医生", Email: "liudoctor@example.com", EmployeeNo: "D1005", Department: "健康管理科", Title: "主治医师", Status: "active", Specialties: []string{"入职体检"}},
		{Name: "周医生", Email: "zhoudoctor@example.com", EmployeeNo: "D1006", Department: "检验科", Title: "主管检验师", Status: "active", Specialties: []string{"入职体检"}},
		{Name: "吴医生", Email: "wudoctor@example.com", EmployeeNo: "D1007", Department: "心电科", Title: "主治医师", Status: "active", Specialties: []string{"入职体检"}},
		{Name: "孙医生", Email: "sundoctor@example.com", EmployeeNo: "D1008", Department: "健康管理科", Title: "住院医师", Status: "active", Specialties: []string{"入职体检"}},
		{Name: "郑医生", Email: "zhengdoctor@example.com", EmployeeNo: "D1009", Department: "内科", Title: "主任医师", Status: "active", Specialties: []string{"慢病筛查", "年度综合"}},
		{Name: "钱医生", Email: "qiandoctor@example.com", EmployeeNo: "D1010", Department: "内科", Title: "副主任医师", Status: "active", Specialties: []string{"慢病筛查", "老年体检"}},
		{Name: "冯医生", Email: "fengdoctor@example.com", EmployeeNo: "D1011", Department: "内科", Title: "主治医师", Status: "active", Specialties: []string{"慢病筛查"}},
		{Name: "曹医生", Email: "caodoctor@example.com", EmployeeNo: "D1012", Department: "老年医学科", Title: "副主任医师", Status: "active", Specialties: []string{"慢病筛查", "老年体检"}},
		{Name: "何医生", Email: "hedoctor@example.com", EmployeeNo: "D1013", Department: "内科", Title: "主治医师", Status: "active", Specialties: []string{"慢病筛查"}},
		{Name: "高医生", Email: "gaodoctor@example.com", EmployeeNo: "D1014", Department: "老年医学科", Title: "主任医师", Status: "active", Specialties: []string{"慢病筛查", "老年体检"}},
		{Name: "罗医生", Email: "luodoctor@example.com", EmployeeNo: "D1015", Department: "内科", Title: "副主任医师", Status: "active", Specialties: []string{"慢病筛查", "年度综合"}},
		{Name: "梁医生", Email: "liangdoctor@example.com", EmployeeNo: "D1016", Department: "健康管理科", Title: "主任医师", Status: "active", Specialties: []string{"年度综合"}},
		{Name: "宋医生", Email: "songdoctor@example.com", EmployeeNo: "D1017", Department: "检验科", Title: "主管检验师", Status: "active", Specialties: []string{"年度综合"}},
		{Name: "唐医生", Email: "tangdoctor@example.com", EmployeeNo: "D1018", Department: "心电科", Title: "副主任医师", Status: "active", Specialties: []string{"年度综合"}},
		{Name: "韩医生", Email: "handoctor@example.com", EmployeeNo: "D1019", Department: "健康管理科", Title: "主治医师", Status: "active", Specialties: []string{"年度综合"}},
		{Name: "马医生", Email: "madoctor@example.com", EmployeeNo: "D1020", Department: "内科", Title: "主治医师", Status: "active", Specialties: []string{"年度综合"}},
		{Name: "朱医生", Email: "zhudoctor@example.com", EmployeeNo: "D1021", Department: "影像科", Title: "副主任医师", Status: "active", Specialties: []string{"影像专项"}},
		{Name: "胡医生", Email: "hudoctor@example.com", EmployeeNo: "D1022", Department: "影像科", Title: "主治医师", Status: "active", Specialties: []string{"影像专项"}},
		{Name: "林医生", Email: "lindoctor@example.com", EmployeeNo: "D1023", Department: "影像科", Title: "主治医师", Status: "active", Specialties: []string{"影像专项"}},
		{Name: "郭医生", Email: "guodoctor@example.com", EmployeeNo: "D1024", Department: "影像科", Title: "主任医师", Status: "active", Specialties: []string{"影像专项"}},
		{Name: "蔡医生", Email: "caidoctor@example.com", EmployeeNo: "D1025", Department: "影像科", Title: "副主任医师", Status: "active", Specialties: []string{"影像专项"}},
		{Name: "袁医生", Email: "yuandoctor@example.com", EmployeeNo: "D1026", Department: "影像科", Title: "主治医师", Status: "active", Specialties: []string{"影像专项", "女性专项"}},
		{Name: "邓医生", Email: "dengdoctor@example.com", EmployeeNo: "D1027", Department: "妇科", Title: "主任医师", Status: "active", Specialties: []string{"女性专项"}},
		{Name: "许医生", Email: "xudoctor@example.com", EmployeeNo: "D1028", Department: "妇科", Title: "副主任医师", Status: "active", Specialties: []string{"女性专项"}},
		{Name: "姚医生", Email: "yaodoctor@example.com", EmployeeNo: "D1029", Department: "妇科", Title: "主治医师", Status: "active", Specialties: []string{"女性专项"}},
		{Name: "潘医生", Email: "pandoctor@example.com", EmployeeNo: "D1030", Department: "老年医学科", Title: "副主任医师", Status: "active", Specialties: []string{"老年体检"}},
		{Name: "蒋医生", Email: "jiangdoctor@example.com", EmployeeNo: "D1031", Department: "老年医学科", Title: "主治医师", Status: "active", Specialties: []string{"老年体检"}},
		{Name: "汪医生", Email: "wang2doctor@example.com", EmployeeNo: "D1032", Department: "老年医学科", Title: "主治医师", Status: "active", Specialties: []string{"老年体检"}},
		{Name: "丁医生", Email: "dingdoctor@example.com", EmployeeNo: "D1033", Department: "老年医学科", Title: "主任医师", Status: "active", Specialties: []string{"老年体检"}},
		{Name: "叶医生", Email: "yedoctor@example.com", EmployeeNo: "D1034", Department: "老年医学科", Title: "副主任医师", Status: "active", Specialties: []string{"老年体检"}},
		{Name: "待审核医生", Email: "pendingdoctor@example.com", EmployeeNo: "D1035", Department: "心电科", Title: "住院医师", Status: "pending", Specialties: []string{"年度综合"}},
	}
}

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
	}
	for _, doctor := range doctorSeeds() {
		users = append(users, models.User{
			Name:         doctor.Name,
			Phone:        doctor.EmployeeNo,
			Email:        doctor.Email,
			PasswordHash: doctorPassword,
			Role:         "doctor",
			Status:       doctor.Status,
			EmployeeNo:   doctor.EmployeeNo,
			Department:   doctor.Department,
			Title:        doctor.Title,
			Specialties:  strings.Join(doctor.Specialties, ","),
			EmailNotify:  true,
		})
	}
	users = append(users, models.User{
		Name: "系统管理员", Phone: "A1001", Email: "huahua20414@foxmail.com", PasswordHash: adminPassword, Role: "admin", Status: "active", EmailNotify: false,
	})
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

	var activeDoctors []models.User
	var mainInstitution, branchInstitution models.CheckupInstitution
	if err := db.Where("role = ? AND status = ?", "doctor", "active").Order("employee_no asc").Find(&activeDoctors).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "熙心健康体检中心总院").First(&mainInstitution).Error; err != nil {
		return err
	}
	if err := db.Where("name = ?", "熙心健康高新区分院").First(&branchInstitution).Error; err != nil {
		return err
	}
	if err := seedSlots(db, activeDoctors, []models.CheckupInstitution{mainInstitution, branchInstitution}); err != nil {
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
	if err := db.Where("name = ?", "年度综合体检").First(&pkg).Error; err != nil {
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
		Category:        pkg.Category,
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
			categories := splitSpecialties(doctor.Specialties)
			if len(categories) == 0 {
				continue
			}
			for _, day := range days {
				for index, start := range starts {
					category := categories[index%len(categories)]
					if category == "" {
						continue
					}
					if err := createSlot(db, doctor, institution, day, start, category); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func createSlot(db *gorm.DB, doctor models.User, institution models.CheckupInstitution, day, start, category string) error {
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
		Category:      category,
		StartTime:     start,
		EndTime:       end,
		Capacity:      1,
		BookedCount:   0,
		Status:        "available",
	}
	return db.Create(&slot).Error
}

func splitSpecialties(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func addMinutes(value string, minutes int) (string, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", err
	}
	end := parsed.Add(time.Duration(minutes) * time.Minute)
	return fmt.Sprintf("%02d:%02d", end.Hour(), end.Minute()), nil
}
