package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/config"
	"health-checkup/backend/internal/mail"
	"health-checkup/backend/internal/middleware"
	"health-checkup/backend/internal/models"
	"health-checkup/backend/internal/seed"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	redis  *redis.Client
	config config.Config
	mailer mail.Sender
}

func NewRouter(db *gorm.DB, redisClient *redis.Client, cfg config.Config) *gin.Engine {
	handler := &Handler{db: db, redis: redisClient, config: cfg, mailer: mail.NewSender(cfg)}
	router := gin.New()
	router.Use(gin.Logger(), middleware.RequestID(), middleware.UnifiedJSONResponse(), middleware.Recovery())
	router.Use(cors.Default())
	router.Use(middleware.IPRateLimit(120, time.Minute))

	api := router.Group("/api")
	api.GET("/health", handler.health)
	api.GET("/packages", handler.packages)
	api.GET("/packages/popular", handler.popularPackages)
	api.GET("/packages/recommended", handler.recommendedPackages)
	api.GET("/coupons/active", handler.activeCoupons)
	api.GET("/announcements/active", handler.activeAnnouncements)
	api.GET("/institutions", handler.institutions)

	authGroup := api.Group("/auth")
	authGroup.Use(middleware.IPRateLimit(20, time.Minute))
	authGroup.POST("/email-code", handler.sendAuthEmailCode)
	authGroup.POST("/register/user", handler.registerUser)
	authGroup.POST("/register/doctor", handler.registerDoctor)
	authGroup.POST("/login", handler.login)

	protected := api.Group("")
	protected.Use(handler.authRequired())
	protected.POST("/auth/logout", handler.logout)
	protected.GET("/auth/me", handler.me)
	protected.PATCH("/profile", handler.updateProfile)
	protected.POST("/profile/email-code", handler.sendEmailCode)
	protected.PATCH("/profile/email", handler.updateEmail)
	protected.GET("/appointments", handler.appointments)
	protected.POST("/appointments", handler.requireRole("user"), handler.createAppointment)
	protected.PATCH("/appointments/:id/cancel", handler.requireRole("user"), handler.cancelAppointment)
	protected.PATCH("/appointments/:id/reschedule", handler.requireRole("user"), handler.rescheduleAppointment)
	protected.GET("/schedule/slots", handler.scheduleSlots)
	protected.POST("/schedule/slots", handler.requireRole("admin"), handler.createScheduleSlot)
	protected.PATCH("/schedule/slots/:id", handler.requireRole("admin"), handler.updateScheduleSlot)
	protected.DELETE("/schedule/slots/:id", handler.requireRole("admin"), handler.archiveScheduleSlot)
	protected.GET("/waitlist", handler.requireRole("user"), handler.waitlist)
	protected.PATCH("/appointments/:id/status", handler.requireRole("doctor", "admin"), handler.updateAppointmentStatus)
	protected.GET("/reports", handler.reports)
	protected.POST("/reports", handler.requireRole("doctor"), handler.createReport)
	protected.GET("/reviews", handler.reviews)
	protected.POST("/reviews", handler.requireRole("user"), handler.createReview)
	protected.PATCH("/reviews/:id/reply", handler.requireRole("admin"), handler.replyReview)
	protected.GET("/family-members", handler.requireRole("user"), handler.familyMembers)
	protected.POST("/family-members", handler.requireRole("user"), handler.createFamilyMember)
	protected.PATCH("/family-members/:id", handler.requireRole("user"), handler.updateFamilyMember)
	protected.DELETE("/family-members/:id", handler.requireRole("user"), handler.deleteFamilyMember)
	protected.GET("/package-favorites", handler.requireRole("user"), handler.packageFavorites)
	protected.POST("/package-favorites/:id", handler.requireRole("user"), handler.favoritePackage)
	protected.DELETE("/package-favorites/:id", handler.requireRole("user"), handler.unfavoritePackage)
	protected.POST("/packages/:id/browse", handler.requireRole("user"), handler.recordPackageBrowse)
	protected.GET("/package-browses", handler.requireRole("user"), handler.packageBrowses)
	protected.GET("/notifications", handler.notifications)
	protected.PATCH("/notifications/:id/read", handler.markNotificationRead)
	protected.GET("/permissions/me", handler.myPermissions)
	protected.GET("/admin/dashboard", handler.requireRole("admin"), handler.adminDashboard)
	protected.GET("/coupons", handler.requireRole("admin"), handler.coupons)
	protected.POST("/coupons", handler.requireRole("admin"), handler.createCoupon)
	protected.PATCH("/coupons/:id", handler.requireRole("admin"), handler.updateCoupon)
	protected.DELETE("/coupons/:id", handler.requireRole("admin"), handler.archiveCoupon)
	protected.GET("/announcements", handler.requireRole("admin"), handler.announcements)
	protected.POST("/announcements", handler.requireRole("admin"), handler.createAnnouncement)
	protected.PATCH("/announcements/:id", handler.requireRole("admin"), handler.updateAnnouncement)
	protected.DELETE("/announcements/:id", handler.requireRole("admin"), handler.archiveAnnouncement)
	protected.POST("/packages", handler.requireRole("admin"), handler.createPackage)
	protected.GET("/packages/export", handler.requireRole("admin"), handler.exportPackages)
	protected.POST("/packages/import", handler.requireRole("admin"), handler.importPackages)
	protected.PATCH("/packages/:id", handler.requireRole("admin"), handler.updatePackage)
	protected.DELETE("/packages/:id", handler.requireRole("admin"), handler.archivePackage)
	protected.GET("/checkup-items", handler.requireRole("admin"), handler.checkupItems)
	protected.POST("/checkup-items", handler.requireRole("admin"), handler.createCheckupItem)
	protected.PATCH("/checkup-items/:id", handler.requireRole("admin"), handler.updateCheckupItem)
	protected.DELETE("/checkup-items/:id", handler.requireRole("admin"), handler.archiveCheckupItem)
	protected.GET("/package-items", handler.requireRole("admin"), handler.packageItems)
	protected.POST("/package-items", handler.requireRole("admin"), handler.upsertPackageItem)
	protected.DELETE("/package-items/:id", handler.requireRole("admin"), handler.deletePackageItem)
	protected.GET("/users", handler.requireRole("doctor", "admin"), handler.users)
	protected.PATCH("/users/:id/status", handler.requireRole("admin"), handler.updateUserStatus)
	protected.PATCH("/users/:id/doctor-profile", handler.requireRole("admin"), handler.updateDoctorProfile)
	protected.GET("/mail-logs", handler.requireRole("admin"), handler.mailLogs)
	protected.GET("/login-logs", handler.requireRole("admin"), handler.loginLogs)
	protected.GET("/operation-logs", handler.requireRole("admin"), handler.operationLogs)
	protected.GET("/role-permissions", handler.requireRole("admin"), handler.rolePermissions)
	protected.PATCH("/role-permissions/:id", handler.requireRole("admin"), handler.updateRolePermission)
	protected.GET("/system-settings", handler.requireRole("admin"), handler.systemSettings)
	protected.PATCH("/system-settings/:id", handler.requireRole("admin"), handler.updateSystemSetting)
	protected.POST("/seed", handler.requireRole("admin"), handler.seed)

	return router
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) sendAuthEmailCode(c *gin.Context) {
	var req emailCodeRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if exists, err := h.redis.Exists(c.Request.Context(), authEmailCodeCooldownKey(email)).Result(); err == nil && exists > 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "email code requests are too frequent"})
		return
	}
	code, err := generateEmailCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate email code failed"})
		return
	}
	if err := h.redis.Set(c.Request.Context(), authEmailCodeKey(email), code, 10*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save email code failed"})
		return
	}
	body := "您好：\n\n您的登录/注册验证码是：" + code + "\n验证码 10 分钟内有效。"
	sendErr := h.mailer.Send(email, "熙心体检验证码", body)
	h.recordMail(0, email, "熙心体检验证码", "登录/注册验证码邮件，正文已脱敏。", sendErr)
	if sendErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "send email code failed"})
		return
	}
	h.redis.Set(c.Request.Context(), authEmailCodeCooldownKey(email), "1", time.Minute)
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handler) registerUser(c *gin.Context) {
	var req registerUserRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.verifyAuthEmailCode(c, email, req.Code) {
		return
	}
	if h.emailExists(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := models.User{
		Name:         strings.TrimSpace(req.Name),
		Phone:        syntheticPhone(email),
		PasswordHash: passwordHash,
		Role:         "user",
		Status:       "active",
		Gender:       req.Gender,
		Age:          req.Age,
		IDCard:       req.IDCard,
		Email:        email,
		EmailNotify:  true,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists or invalid user data"})
		return
	}
	h.redis.Del(c.Request.Context(), authEmailCodeKey(email))
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) registerDoctor(c *gin.Context) {
	var req registerDoctorRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.verifyAuthEmailCode(c, email, req.Code) {
		return
	}
	if h.emailExists(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := models.User{
		Name:         strings.TrimSpace(req.Name),
		Phone:        syntheticPhone(email),
		PasswordHash: passwordHash,
		Role:         "doctor",
		Status:       "pending",
		Email:        email,
		EmployeeNo:   req.EmployeeNo,
		Department:   req.Department,
		Title:        req.Title,
		EmailNotify:  true,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists or invalid doctor data"})
		return
	}
	h.redis.Del(c.Request.Context(), authEmailCodeKey(email))
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !bind(c, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !h.config.DevAuthEnabled && !h.verifyAuthEmailCode(c, email, req.Code) {
		return
	}
	var candidates []models.User
	query := h.db.Where("email = ?", email)
	if h.config.DevAuthEnabled && req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}
	if err := query.Find(&candidates).Error; err != nil || len(candidates) == 0 {
		h.recordLogin(c, 0, email, req.Role, "failed", "account not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email, password or code"})
		return
	}
	var user models.User
	for _, candidate := range candidates {
		if auth.CheckPassword(candidate.PasswordHash, req.Password) {
			user = candidate
			break
		}
	}
	if user.ID == 0 {
		h.recordLogin(c, 0, email, req.Role, "failed", "password mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email, password or code"})
		return
	}
	if user.Status != "active" {
		h.recordLogin(c, user.ID, user.Email, user.Role, "blocked", user.Status)
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not active", "status": user.Status})
		return
	}
	token, err := auth.IssueToken(c.Request.Context(), h.redis, h.config.JWTSecret, time.Duration(h.config.TokenHours)*time.Hour, user)
	if err != nil {
		h.recordLogin(c, user.ID, user.Email, user.Role, "failed", "issue token failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	if req.Code != "" {
		h.redis.Del(c.Request.Context(), authEmailCodeKey(email))
	}
	h.recordLogin(c, user.ID, user.Email, user.Role, "success", "")
	c.JSON(http.StatusOK, gin.H{"accessToken": token, "user": user})
}

func (h *Handler) logout(c *gin.Context) {
	claims := currentClaims(c)
	if claims != nil {
		h.redis.Del(c.Request.Context(), auth.SessionKey(claims.SessionID))
	}
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

func (h *Handler) me(c *gin.Context) {
	c.JSON(http.StatusOK, currentUser(c))
}

func (h *Handler) updateProfile(c *gin.Context) {
	var req profileRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	updates := map[string]any{
		"name":         req.Name,
		"gender":       req.Gender,
		"age":          req.Age,
		"id_card":      req.IDCard,
		"avatar_url":   req.AvatarURL,
		"bio":          req.Bio,
		"email_notify": req.EmailNotify,
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", current.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.First(&current, current.ID)
	c.JSON(http.StatusOK, current)
}

func (h *Handler) sendEmailCode(c *gin.Context) {
	var req emailCodeRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	if exists, err := h.redis.Exists(c.Request.Context(), emailCodeCooldownKey(current.ID)).Result(); err == nil && exists > 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "email code requests are too frequent"})
		return
	}
	code, err := generateEmailCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate email code failed"})
		return
	}
	key := emailCodeKey(current.ID)
	if err := h.redis.Set(c.Request.Context(), key, req.Email+"|"+code, 10*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save email code failed"})
		return
	}
	body := "您好，" + current.Name + "：\n\n您的邮箱变更验证码是：" + code + "\n验证码 10 分钟内有效。"
	sendErr := h.mailer.Send(req.Email, "邮箱变更验证码", body)
	h.recordMail(current.ID, req.Email, "邮箱变更验证码", "邮箱变更验证码邮件，正文已脱敏。", sendErr)
	if sendErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "send email code failed"})
		return
	}
	h.redis.Set(c.Request.Context(), emailCodeCooldownKey(current.ID), "1", time.Minute)
	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handler) updateEmail(c *gin.Context) {
	var req updateEmailRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	value, err := h.redis.Get(c.Request.Context(), emailCodeKey(current.ID)).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email code expired"})
		return
	}
	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 || parts[0] != req.Email || parts[1] != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email code"})
		return
	}
	var existing models.User
	if err := h.db.Where("email = ? AND id <> ?", req.Email, current.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", current.ID).Update("email", req.Email).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.redis.Del(c.Request.Context(), emailCodeKey(current.ID))
	h.db.First(&current, current.ID)
	c.JSON(http.StatusOK, current)
}

func (h *Handler) packages(c *gin.Context) {
	var packages []models.CheckupPackage
	query := h.db.Model(&models.CheckupPackage{}).Order("price asc")
	if c.GetHeader("Authorization") == "" {
		query = query.Where("status = ?", "active")
	} else if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", "deleted")
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &packages)
		return
	}
	if err := query.Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, packages)
}

func (h *Handler) popularPackages(c *gin.Context) {
	var packages []models.CheckupPackage
	query := h.db.Model(&models.CheckupPackage{}).
		Select("checkup_packages.*, COUNT(appointments.id) AS booking_count").
		Joins("LEFT JOIN appointments ON appointments.package_id = checkup_packages.id AND appointments.status <> ?", "canceled").
		Where("checkup_packages.status = ?", "active").
		Group("checkup_packages.id").
		Order("booking_count DESC, checkup_packages.price ASC").
		Limit(6)
	if err := query.Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, packages)
}

func (h *Handler) recommendedPackages(c *gin.Context) {
	var packages []models.CheckupPackage
	query := h.db.Model(&models.CheckupPackage{}).Where("status = ?", "active").Order("price asc").Limit(6)
	if user, ok := c.Get("user"); ok {
		current, _ := user.(models.User)
		if current.ID != 0 && current.Age >= 55 {
			query = h.db.Model(&models.CheckupPackage{}).
				Where("status = ? AND (category = ? OR name LIKE ?)", "active", "老年体检", "%老年%").
				Order("price asc").
				Limit(6)
		}
	}
	if err := query.Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, packages)
}

func (h *Handler) institutions(c *gin.Context) {
	var institutions []models.CheckupInstitution
	query := h.db.Order("id asc")
	if c.GetHeader("Authorization") == "" {
		query = query.Where("status = ?", "active")
	}
	if err := query.Find(&institutions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, institutions)
}

func (h *Handler) activeCoupons(c *gin.Context) {
	var coupons []models.Coupon
	today := time.Now().Format("2006-01-02")
	query := h.db.Where("status = ?", "active").
		Where("(start_date = '' OR start_date <= ?) AND (end_date = '' OR end_date >= ?)", today, today).
		Order("value desc, created_at desc").
		Limit(20)
	if packageID := c.Query("packageId"); packageID != "" {
		query = query.Where("package_id = 0 OR package_id = ?", packageID)
	}
	if err := query.Find(&coupons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, coupons)
}

func (h *Handler) activeAnnouncements(c *gin.Context) {
	var announcements []models.SystemAnnouncement
	query := h.db.Where("status = ?", "published").Order("published_at desc, created_at desc").Limit(10)
	if audience := c.Query("audience"); audience != "" {
		query = query.Where("audience = ? OR audience = ?", audience, "all")
	}
	if err := query.Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, announcements)
}

func (h *Handler) appointments(c *gin.Context) {
	current := currentUser(c)
	var appointments []models.Appointment
	query := h.db.Model(&models.Appointment{}).Preload("User").Preload("FamilyMember").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").Preload("Report").Order("created_at desc")
	if current.Role == "user" {
		query = query.Where("user_id = ?", current.ID)
	} else if current.Role == "doctor" {
		query = query.Where("doctor_id = ?", current.ID)
	} else if userID := c.Query("userId"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Joins("LEFT JOIN users appointment_users ON appointment_users.id = appointments.user_id").
			Joins("LEFT JOIN checkup_packages appointment_packages ON appointment_packages.id = appointments.package_id").
			Where("appointment_users.name LIKE ? OR appointment_packages.name LIKE ? OR appointments.order_no LIKE ? OR appointments.date LIKE ?", pattern, pattern, pattern, pattern)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &appointments)
		return
	}
	if err := query.Find(&appointments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, appointments)
}

func (h *Handler) createAppointment(c *gin.Context) {
	var req appointmentRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	var pkg models.CheckupPackage
	if err := h.db.First(&pkg, req.PackageID).Error; err != nil || pkg.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package is unavailable"})
		return
	}
	var institution models.CheckupInstitution
	if err := h.db.First(&institution, req.InstitutionID).Error; err != nil || institution.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "institution is unavailable"})
		return
	}
	if req.FamilyMemberID != 0 && !h.familyMemberBelongsTo(current.ID, req.FamilyMemberID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "family member not found"})
		return
	}
	appointmentType := normalizeStatus(req.AppointmentType, "个人体检")
	var result any
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var slot models.ScheduleSlot
		query := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("status = ? AND booked_count < capacity AND institution_id = ? AND category = ?", "available", req.InstitutionID, pkg.Category)
		if req.SlotID != 0 {
			query = query.Where("id = ?", req.SlotID)
		} else {
			query = query.Where("date = ? AND period = ?", req.Date, req.Period).Order("start_time asc")
		}
		err := query.First(&slot).Error
		if err != nil {
			waitDate := req.Date
			waitPeriod := req.Period
			waitStartTime := ""
			waitEndTime := ""
			waitInstitutionID := req.InstitutionID
			if req.SlotID != 0 {
				var requestedSlot models.ScheduleSlot
				if lookupErr := tx.First(&requestedSlot, req.SlotID).Error; lookupErr == nil {
					waitDate = requestedSlot.Date
					waitPeriod = requestedSlot.Period
					waitStartTime = requestedSlot.StartTime
					waitEndTime = requestedSlot.EndTime
					waitInstitutionID = requestedSlot.InstitutionID
				}
			}
			wait := models.WaitlistEntry{
				UserID:          current.ID,
				PackageID:       req.PackageID,
				InstitutionID:   waitInstitutionID,
				AppointmentType: appointmentType,
				Category:        pkg.Category,
				Date:            waitDate,
				Period:          waitPeriod,
				StartTime:       waitStartTime,
				EndTime:         waitEndTime,
				Note:            req.Note,
				Status:          "waiting",
			}
			var existingWait models.WaitlistEntry
			duplicateQuery := tx.Preload("Institution").Preload("Package").
				Where("user_id = ? AND package_id = ? AND institution_id = ? AND category = ? AND date = ? AND period = ? AND status = ?", current.ID, req.PackageID, waitInstitutionID, pkg.Category, waitDate, waitPeriod, "waiting")
			if waitStartTime != "" {
				duplicateQuery = duplicateQuery.Where("start_time = ?", waitStartTime)
			}
			if existingErr := duplicateQuery.First(&existingWait).Error; existingErr == nil {
				result = gin.H{"type": "waitlist", "waitlist": existingWait}
				return nil
			}
			if createErr := tx.Create(&wait).Error; createErr != nil {
				return createErr
			}
			tx.Preload("Institution").Preload("Package").First(&wait, wait.ID)
			result = gin.H{"type": "waitlist", "waitlist": wait}
			return nil
		}
		appointment := models.Appointment{
			OrderNo:         generateOrderNo(),
			UserID:          current.ID,
			FamilyMemberID:  req.FamilyMemberID,
			DoctorID:        slot.DoctorID,
			InstitutionID:   slot.InstitutionID,
			SlotID:          slot.ID,
			PackageID:       req.PackageID,
			AppointmentType: appointmentType,
			Category:        pkg.Category,
			Date:            slot.Date,
			Period:          slot.Period,
			StartTime:       slot.StartTime,
			EndTime:         slot.EndTime,
			Status:          "booked",
			Note:            req.Note,
			PaymentStatus:   normalizeStatus(req.PaymentStatus, "unpaid"),
			InvoiceTitle:    req.InvoiceTitle,
			InvoiceTaxNo:    req.InvoiceTaxNo,
		}
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		if err := tx.Model(&slot).Update("booked_count", slot.BookedCount+1).Error; err != nil {
			return err
		}
		tx.Preload("User").Preload("FamilyMember").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").First(&appointment, appointment.ID)
		result = gin.H{"type": "appointment", "appointment": appointment}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payload, ok := result.(gin.H); ok && payload["type"] == "appointment" {
		if appointment, ok := payload["appointment"].(models.Appointment); ok {
			h.sendAppointmentMail(appointment, "体检预约成功")
			h.createAppointmentNotifications(appointment, "appointment_success", "预约成功", "您的体检预约已成功，系统已模拟发送短信提醒。")
		}
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) cancelAppointment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment id"})
		return
	}
	current := currentUser(c)
	var appointment models.Appointment
	if err := h.db.Where("id = ? AND user_id = ?", id, current.ID).First(&appointment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}
	if appointment.Status != "booked" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only booked appointments can be canceled"})
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&appointment).Update("status", "canceled").Error; err != nil {
			return err
		}
		if appointment.SlotID != 0 {
			if err := tx.Model(&models.ScheduleSlot{}).Where("id = ? AND booked_count > 0", appointment.SlotID).Update("booked_count", gorm.Expr("booked_count - 1")).Error; err != nil {
				return err
			}
			return h.promoteWaitlist(tx, appointment.SlotID)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "canceled"})
}

func (h *Handler) rescheduleAppointment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment id"})
		return
	}
	var req rescheduleRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	var appointment models.Appointment
	if err := h.db.Where("id = ? AND user_id = ?", id, current.ID).First(&appointment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}
	if appointment.Status != "booked" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only booked appointments can be rescheduled"})
		return
	}
	var pkg models.CheckupPackage
	if err := h.db.First(&pkg, appointment.PackageID).Error; err != nil || pkg.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package is unavailable"})
		return
	}
	var institution models.CheckupInstitution
	if err := h.db.First(&institution, req.InstitutionID).Error; err != nil || institution.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "institution is unavailable"})
		return
	}
	var updated models.Appointment
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var slot models.ScheduleSlot
		query := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("status = ? AND booked_count < capacity AND institution_id = ? AND category = ?", "available", req.InstitutionID, pkg.Category)
		if req.SlotID != 0 {
			query = query.Where("id = ?", req.SlotID)
		} else {
			query = query.Where("date = ? AND period = ?", req.Date, req.Period).Order("start_time asc")
		}
		if err := query.First(&slot).Error; err != nil {
			return err
		}
		if appointment.SlotID == slot.ID {
			return tx.Preload("User").Preload("FamilyMember").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").First(&updated, appointment.ID).Error
		}
		if appointment.SlotID != 0 {
			if err := tx.Model(&models.ScheduleSlot{}).Where("id = ? AND booked_count > 0", appointment.SlotID).Update("booked_count", gorm.Expr("booked_count - 1")).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.ScheduleSlot{}).Where("id = ?", slot.ID).Update("booked_count", slot.BookedCount+1).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"doctor_id":      slot.DoctorID,
			"institution_id": slot.InstitutionID,
			"slot_id":        slot.ID,
			"date":           slot.Date,
			"period":         slot.Period,
			"start_time":     slot.StartTime,
			"end_time":       slot.EndTime,
		}
		if strings.TrimSpace(req.Note) != "" {
			updates["note"] = req.Note
		}
		if err := tx.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Preload("User").Preload("FamilyMember").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").First(&updated, appointment.ID).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no available slot for reschedule"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.createAppointmentNotifications(updated, "appointment_rescheduled", "预约已改期", "您的体检预约时间已更新，请按新的时间到检。")
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) updateAppointmentStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment id"})
		return
	}
	var req statusRequest
	if !bind(c, &req) {
		return
	}
	if req.Status != "booked" && req.Status != "checked" && req.Status != "reported" && req.Status != "canceled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment status"})
		return
	}
	if err := h.db.Model(&models.Appointment{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

func (h *Handler) reports(c *gin.Context) {
	current := currentUser(c)
	var reports []models.Report
	query := h.db.Model(&models.Report{}).Preload("Appointment.Institution").Preload("Appointment.Package").Preload("User").Preload("Doctor").Order("created_at desc")
	if current.Role == "user" {
		query = query.Where("user_id = ?", current.ID)
	} else if current.Role == "doctor" {
		query = query.Where("doctor_id = ?", current.ID)
	} else if userID := c.Query("userId"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &reports)
		return
	}
	if err := query.Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reports)
}

func (h *Handler) createReport(c *gin.Context) {
	var req reportRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	var appointment models.Appointment
	if err := h.db.First(&appointment, req.AppointmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}
	if appointment.Status != "checked" && appointment.Status != "reported" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report can only be created after checkup is completed"})
		return
	}
	report := models.Report{
		ReportNo:       generateReportNo(),
		AppointmentID:  appointment.ID,
		UserID:         appointment.UserID,
		DoctorID:       current.ID,
		Summary:        req.Summary,
		Conclusion:     req.Conclusion,
		Recommendation: req.Recommendation,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(models.Report{AppointmentID: appointment.ID}).Assign(report).FirstOrCreate(&report).Error; err != nil {
			return err
		}
		return tx.Model(&models.Appointment{}).Where("id = ?", appointment.ID).Update("status", "reported").Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("Appointment.Institution").Preload("Appointment.Package").Preload("User").Preload("Doctor").First(&report, report.ID)
	h.sendReportMail(report)
	c.JSON(http.StatusCreated, report)
}

func (h *Handler) scheduleSlots(c *gin.Context) {
	var slots []models.ScheduleSlot
	query := h.db.Preload("Doctor").Preload("Institution").Order("date asc, start_time asc, doctor_id asc")
	if institutionID := c.Query("institutionId"); institutionID != "" {
		query = query.Where("institution_id = ?", institutionID)
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if date := c.Query("date"); date != "" {
		query = query.Where("date = ?", date)
	}
	if period := c.Query("period"); period != "" {
		query = query.Where("period = ?", period)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", "deleted")
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &slots)
		return
	}
	if err := query.Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slots)
}

func (h *Handler) createScheduleSlot(c *gin.Context) {
	var req scheduleSlotRequest
	if !bind(c, &req) {
		return
	}
	slot, ok := h.buildScheduleSlot(c, req, 0)
	if !ok {
		return
	}
	if err := h.db.Create(&slot).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("Doctor").Preload("Institution").First(&slot, slot.ID)
	h.recordOperation(c, "create", "schedule_slot", strconv.Itoa(int(slot.ID)), "success", fmt.Sprintf("%s %s %s", slot.Date, slot.StartTime, slot.Category))
	c.JSON(http.StatusCreated, slot)
}

func (h *Handler) updateScheduleSlot(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule slot id"})
		return
	}
	var existing models.ScheduleSlot
	if err := h.db.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule slot not found"})
		return
	}
	var req scheduleSlotRequest
	if !bind(c, &req) {
		return
	}
	slot, ok := h.buildScheduleSlot(c, req, existing.BookedCount)
	if !ok {
		return
	}
	updates := map[string]any{
		"doctor_id":      slot.DoctorID,
		"institution_id": slot.InstitutionID,
		"date":           slot.Date,
		"period":         slot.Period,
		"category":       slot.Category,
		"start_time":     slot.StartTime,
		"end_time":       slot.EndTime,
		"capacity":       slot.Capacity,
		"booked_count":   slot.BookedCount,
		"status":         slot.Status,
	}
	if err := h.db.Model(&models.ScheduleSlot{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var updated models.ScheduleSlot
	h.db.Preload("Doctor").Preload("Institution").First(&updated, id)
	h.recordOperation(c, "update", "schedule_slot", strconv.Itoa(id), "success", fmt.Sprintf("%s %s", updated.Date, updated.StartTime))
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) archiveScheduleSlot(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "invalid schedule slot id")
	if !ok {
		return
	}
	if err := h.archiveByID(&models.ScheduleSlot{}, id, "schedule_slot"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "archive", "schedule_slot", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) waitlist(c *gin.Context) {
	current := currentUser(c)
	var entries []models.WaitlistEntry
	query := h.db.Model(&models.WaitlistEntry{}).Preload("Institution").Preload("Package").Where("user_id = ?", current.ID).Order("created_at desc")
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &entries)
		return
	}
	if err := query.Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) reviews(c *gin.Context) {
	current := currentUser(c)
	var reviews []models.ServiceReview
	query := h.db.Model(&models.ServiceReview{}).
		Preload("User").Preload("Appointment").Preload("Package").Preload("Institution").Preload("Doctor").
		Order("created_at desc")
	if current.Role == "user" {
		query = query.Where("user_id = ?", current.ID)
	} else if current.Role != "admin" {
		query = query.Where("doctor_id = ?", current.ID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &reviews)
		return
	}
	if err := query.Limit(100).Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reviews)
}

func (h *Handler) createReview(c *gin.Context) {
	var req reviewRequest
	if !bind(c, &req) {
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be between 1 and 5"})
		return
	}
	current := currentUser(c)
	var appointment models.Appointment
	if err := h.db.Where("id = ? AND user_id = ?", req.AppointmentID, current.ID).First(&appointment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}
	if appointment.Status != "reported" && appointment.Status != "checked" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "review can only be created after checkup"})
		return
	}
	review := models.ServiceReview{
		UserID:        current.ID,
		AppointmentID: appointment.ID,
		PackageID:     appointment.PackageID,
		InstitutionID: appointment.InstitutionID,
		DoctorID:      appointment.DoctorID,
		Rating:        req.Rating,
		Content:       strings.TrimSpace(req.Content),
		Status:        "published",
	}
	if err := h.db.Create(&review).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appointment has already been reviewed"})
		return
	}
	h.db.Preload("User").Preload("Appointment").Preload("Package").Preload("Institution").Preload("Doctor").First(&review, review.ID)
	c.JSON(http.StatusCreated, review)
}

func (h *Handler) replyReview(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	var req reviewReplyRequest
	if !bind(c, &req) {
		return
	}
	updates := map[string]any{
		"reply":  strings.TrimSpace(req.Reply),
		"status": normalizeStatus(req.Status, "published"),
	}
	if updates["status"] != "published" && updates["status"] != "hidden" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review status"})
		return
	}
	if err := h.db.Model(&models.ServiceReview{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var review models.ServiceReview
	h.db.Preload("User").Preload("Appointment").Preload("Package").Preload("Institution").Preload("Doctor").First(&review, id)
	c.JSON(http.StatusOK, review)
}

func (h *Handler) familyMembers(c *gin.Context) {
	current := currentUser(c)
	var members []models.FamilyMember
	if err := h.db.Where("user_id = ?", current.ID).Order("created_at desc").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, members)
}

func (h *Handler) createFamilyMember(c *gin.Context) {
	var req familyMemberRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	member := models.FamilyMember{
		UserID:   current.ID,
		Name:     strings.TrimSpace(req.Name),
		Relation: strings.TrimSpace(req.Relation),
		Gender:   req.Gender,
		Age:      req.Age,
		IDCard:   req.IDCard,
		Phone:    req.Phone,
	}
	if err := h.db.Create(&member).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, member)
}

func (h *Handler) updateFamilyMember(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid family member id"})
		return
	}
	var req familyMemberRequest
	if !bind(c, &req) {
		return
	}
	current := currentUser(c)
	updates := map[string]any{
		"name":     strings.TrimSpace(req.Name),
		"relation": strings.TrimSpace(req.Relation),
		"gender":   req.Gender,
		"age":      req.Age,
		"id_card":  req.IDCard,
		"phone":    req.Phone,
	}
	result := h.db.Model(&models.FamilyMember{}).Where("id = ? AND user_id = ?", id, current.ID).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "family member not found"})
		return
	}
	var member models.FamilyMember
	h.db.First(&member, id)
	c.JSON(http.StatusOK, member)
}

func (h *Handler) deleteFamilyMember(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid family member id"})
		return
	}
	current := currentUser(c)
	result := h.db.Where("id = ? AND user_id = ?", id, current.ID).Delete(&models.FamilyMember{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "family member not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) packageFavorites(c *gin.Context) {
	current := currentUser(c)
	var favorites []models.PackageFavorite
	if err := h.db.Preload("Package").Where("user_id = ?", current.ID).Order("created_at desc").Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, favorites)
}

func (h *Handler) favoritePackage(c *gin.Context) {
	packageID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package id"})
		return
	}
	current := currentUser(c)
	var pkg models.CheckupPackage
	if err := h.db.Where("id = ? AND status = ?", packageID, "active").First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	favorite := models.PackageFavorite{UserID: current.ID, PackageID: uint(packageID)}
	if err := h.db.Where(favorite).FirstOrCreate(&favorite).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("Package").First(&favorite, favorite.ID)
	c.JSON(http.StatusOK, favorite)
}

func (h *Handler) unfavoritePackage(c *gin.Context) {
	packageID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package id"})
		return
	}
	current := currentUser(c)
	if err := h.db.Where("user_id = ? AND package_id = ?", current.ID, packageID).Delete(&models.PackageFavorite{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"packageId": packageID, "status": "deleted"})
}

func (h *Handler) recordPackageBrowse(c *gin.Context) {
	packageID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package id"})
		return
	}
	current := currentUser(c)
	var pkg models.CheckupPackage
	if err := h.db.Where("id = ? AND status = ?", packageID, "active").First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	now := time.Now()
	var history models.PackageBrowseHistory
	if err := h.db.Where("user_id = ? AND package_id = ?", current.ID, packageID).First(&history).Error; err != nil {
		history = models.PackageBrowseHistory{UserID: current.ID, PackageID: uint(packageID), ViewCount: 1, ViewedAt: now}
		if err := h.db.Create(&history).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if err := h.db.Model(&history).Updates(map[string]any{"view_count": gorm.Expr("view_count + 1"), "viewed_at": now}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("Package").First(&history, history.ID)
	c.JSON(http.StatusOK, history)
}

func (h *Handler) packageBrowses(c *gin.Context) {
	current := currentUser(c)
	var histories []models.PackageBrowseHistory
	if err := h.db.Preload("Package").Where("user_id = ?", current.ID).Order("viewed_at desc").Limit(10).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, histories)
}

func (h *Handler) notifications(c *gin.Context) {
	current := currentUser(c)
	var notifications []models.Notification
	query := h.db.Where("user_id = ?", current.ID).Order("created_at desc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &notifications)
		return
	}
	if err := query.Limit(50).Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, notifications)
}

func (h *Handler) markNotificationRead(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}
	current := currentUser(c)
	now := time.Now()
	result := h.db.Model(&models.Notification{}).Where("id = ? AND user_id = ?", id, current.ID).Updates(map[string]any{"status": "read", "read_at": &now})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "read"})
}

func (h *Handler) mailLogs(c *gin.Context) {
	var logs []models.MailLog
	query := h.db.Model(&models.MailLog{}).Order("created_at desc")
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &logs)
		return
	}
	if err := query.Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) loginLogs(c *gin.Context) {
	var logs []models.LoginLog
	query := h.db.Model(&models.LoginLog{}).Order("created_at desc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("email LIKE ? OR ip LIKE ? OR role LIKE ?", pattern, pattern, pattern)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &logs)
		return
	}
	if err := query.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) operationLogs(c *gin.Context) {
	var logs []models.OperationLog
	query := h.db.Model(&models.OperationLog{}).Order("created_at desc")
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if resource := c.Query("resource"); resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("user_name LIKE ? OR action LIKE ? OR resource LIKE ? OR detail LIKE ?", pattern, pattern, pattern, pattern)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &logs)
		return
	}
	if err := query.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) myPermissions(c *gin.Context) {
	current := currentUser(c)
	permissions := h.permissionsForRole(current.Role)
	c.JSON(http.StatusOK, gin.H{"role": current.Role, "permissions": permissions})
}

func (h *Handler) rolePermissions(c *gin.Context) {
	var permissions []models.RolePermission
	query := h.db.Model(&models.RolePermission{}).Order("role asc, permission asc")
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if err := query.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permissions)
}

func (h *Handler) updateRolePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}
	var req rolePermissionRequest
	if !bind(c, &req) {
		return
	}
	if err := h.db.Model(&models.RolePermission{}).Where("id = ?", id).Update("enabled", req.Enabled).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var permission models.RolePermission
	h.db.First(&permission, id)
	h.recordOperation(c, "update", "role_permission", strconv.Itoa(id), "success", fmt.Sprintf("%s:%s=%t", permission.Role, permission.Permission, permission.Enabled))
	c.JSON(http.StatusOK, permission)
}

func (h *Handler) systemSettings(c *gin.Context) {
	var settings []models.SystemSetting
	query := h.db.Model(&models.SystemSetting{}).Order("`group` asc, id asc")
	if group := c.Query("group"); group != "" {
		query = query.Where("`group` = ?", group)
	}
	if err := query.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) updateSystemSetting(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid setting id"})
		return
	}
	var req systemSettingRequest
	if !bind(c, &req) {
		return
	}
	value := strings.TrimSpace(req.Value)
	if err := validateSettingValue(req.ValueType, value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]any{
		"value":       value,
		"value_type":  normalizeStatus(req.ValueType, "string"),
		"description": req.Description,
		"status":      normalizeStatus(req.Status, "active"),
	}
	if strings.TrimSpace(req.Label) != "" {
		updates["label"] = strings.TrimSpace(req.Label)
	}
	if err := h.db.Model(&models.SystemSetting{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var setting models.SystemSetting
	h.db.First(&setting, id)
	h.recordOperation(c, "update", "system_setting", strconv.Itoa(id), "success", fmt.Sprintf("%s=%s", setting.Key, setting.Value))
	c.JSON(http.StatusOK, setting)
}

func (h *Handler) adminDashboard(c *gin.Context) {
	var userCount, doctorCount, appointmentCount, reportCount, reviewCount int64
	h.db.Model(&models.User{}).Where("role = ?", "user").Count(&userCount)
	h.db.Model(&models.User{}).Where("role = ?", "doctor").Count(&doctorCount)
	h.db.Model(&models.Appointment{}).Count(&appointmentCount)
	h.db.Model(&models.Report{}).Count(&reportCount)
	h.db.Model(&models.ServiceReview{}).Count(&reviewCount)

	type row struct {
		Label string  `json:"label"`
		Count int64   `json:"count"`
		Total float64 `json:"total,omitempty"`
	}
	var appointmentTrend []row
	h.db.Model(&models.Appointment{}).
		Select("date AS label, COUNT(*) AS count").
		Group("date").
		Order("date asc").
		Limit(14).
		Scan(&appointmentTrend)
	var packageSales []row
	h.db.Model(&models.Appointment{}).
		Select("checkup_packages.name AS label, COUNT(appointments.id) AS count, SUM(checkup_packages.price) AS total").
		Joins("LEFT JOIN checkup_packages ON checkup_packages.id = appointments.package_id").
		Where("appointments.status <> ?", "canceled").
		Group("checkup_packages.id, checkup_packages.name").
		Order("count desc").
		Limit(10).
		Scan(&packageSales)
	var userGrowth []row
	h.db.Model(&models.User{}).
		Select("DATE(created_at) AS label, COUNT(*) AS count").
		Where("role = ?", "user").
		Group("DATE(created_at)").
		Order("label asc").
		Limit(14).
		Scan(&userGrowth)
	var averageRating struct {
		Average float64 `json:"average"`
	}
	h.db.Model(&models.ServiceReview{}).Select("AVG(rating) AS average").Scan(&averageRating)
	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"users":         userCount,
			"doctors":       doctorCount,
			"appointments":  appointmentCount,
			"reports":       reportCount,
			"reviews":       reviewCount,
			"averageRating": averageRating.Average,
		},
		"appointmentTrend": appointmentTrend,
		"packageSales":     packageSales,
		"userGrowth":       userGrowth,
	})
}

func (h *Handler) coupons(c *gin.Context) {
	var coupons []models.Coupon
	query := h.db.Model(&models.Coupon{}).Order("created_at desc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", "deleted")
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &coupons)
		return
	}
	if err := query.Find(&coupons).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, coupons)
}

func (h *Handler) createCoupon(c *gin.Context) {
	var req couponRequest
	if !bind(c, &req) {
		return
	}
	coupon := models.Coupon{
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.ToUpper(strings.TrimSpace(req.Code)),
		Type:        normalizeStatus(req.Type, "amount"),
		Value:       req.Value,
		MinAmount:   req.MinAmount,
		PackageID:   req.PackageID,
		Status:      normalizeStatus(req.Status, "active"),
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Description: req.Description,
	}
	if err := h.db.Create(&coupon).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "create", "coupon", strconv.Itoa(int(coupon.ID)), "success", coupon.Code)
	c.JSON(http.StatusCreated, coupon)
}

func (h *Handler) updateCoupon(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coupon id"})
		return
	}
	var req couponRequest
	if !bind(c, &req) {
		return
	}
	updates := map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"code":        strings.ToUpper(strings.TrimSpace(req.Code)),
		"type":        normalizeStatus(req.Type, "amount"),
		"value":       req.Value,
		"min_amount":  req.MinAmount,
		"package_id":  req.PackageID,
		"status":      normalizeStatus(req.Status, "active"),
		"start_date":  req.StartDate,
		"end_date":    req.EndDate,
		"description": req.Description,
	}
	if err := h.db.Model(&models.Coupon{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var coupon models.Coupon
	h.db.First(&coupon, id)
	h.recordOperation(c, "update", "coupon", strconv.Itoa(id), "success", coupon.Code)
	c.JSON(http.StatusOK, coupon)
}

func (h *Handler) archiveCoupon(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "invalid coupon id")
	if !ok {
		return
	}
	if err := h.archiveByID(&models.Coupon{}, id, "coupon"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "archive", "coupon", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) announcements(c *gin.Context) {
	var announcements []models.SystemAnnouncement
	query := h.db.Model(&models.SystemAnnouncement{}).Order("created_at desc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", "deleted")
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &announcements)
		return
	}
	if err := query.Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, announcements)
}

func (h *Handler) createAnnouncement(c *gin.Context) {
	var req announcementRequest
	if !bind(c, &req) {
		return
	}
	announcement := models.SystemAnnouncement{
		Title:    strings.TrimSpace(req.Title),
		Content:  strings.TrimSpace(req.Content),
		Audience: normalizeStatus(req.Audience, "all"),
		Status:   normalizeStatus(req.Status, "draft"),
	}
	if announcement.Status == "published" {
		now := time.Now()
		announcement.PublishedAt = &now
	}
	if err := h.db.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "create", "announcement", strconv.Itoa(int(announcement.ID)), "success", announcement.Title)
	c.JSON(http.StatusCreated, announcement)
}

func (h *Handler) updateAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement id"})
		return
	}
	var req announcementRequest
	if !bind(c, &req) {
		return
	}
	status := normalizeStatus(req.Status, "draft")
	updates := map[string]any{
		"title":    strings.TrimSpace(req.Title),
		"content":  strings.TrimSpace(req.Content),
		"audience": normalizeStatus(req.Audience, "all"),
		"status":   status,
	}
	if status == "published" {
		now := time.Now()
		updates["published_at"] = &now
	}
	if err := h.db.Model(&models.SystemAnnouncement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var announcement models.SystemAnnouncement
	h.db.First(&announcement, id)
	h.recordOperation(c, "update", "announcement", strconv.Itoa(id), "success", announcement.Title)
	c.JSON(http.StatusOK, announcement)
}

func (h *Handler) archiveAnnouncement(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "invalid announcement id")
	if !ok {
		return
	}
	if err := h.archiveByID(&models.SystemAnnouncement{}, id, "announcement"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "archive", "announcement", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) createPackage(c *gin.Context) {
	var req packageRequest
	if !bind(c, &req) {
		return
	}
	pkg := models.CheckupPackage{
		Name:        req.Name,
		Category:    normalizeStatus(req.Category, "综合体检"),
		Description: req.Description,
		Price:       req.Price,
		Items:       req.Items,
		Status:      normalizeStatus(req.Status, "active"),
	}
	if err := h.db.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "create", "package", strconv.Itoa(int(pkg.ID)), "success", pkg.Name)
	c.JSON(http.StatusCreated, pkg)
}

func (h *Handler) updatePackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package id"})
		return
	}
	var req packageRequest
	if !bind(c, &req) {
		return
	}
	updates := map[string]any{
		"name":        req.Name,
		"category":    normalizeStatus(req.Category, "综合体检"),
		"description": req.Description,
		"price":       req.Price,
		"items":       req.Items,
		"status":      normalizeStatus(req.Status, "active"),
	}
	if err := h.db.Model(&models.CheckupPackage{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var pkg models.CheckupPackage
	h.db.First(&pkg, id)
	h.recordOperation(c, "update", "package", strconv.Itoa(id), "success", pkg.Name)
	c.JSON(http.StatusOK, pkg)
}

func (h *Handler) archivePackage(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "invalid package id")
	if !ok {
		return
	}
	if err := h.archiveByID(&models.CheckupPackage{}, id, "package"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "archive", "package", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) exportPackages(c *gin.Context) {
	var packages []models.CheckupPackage
	if err := h.db.Order("id asc").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="packages.csv"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"name", "category", "description", "price", "items", "status"})
	for _, pkg := range packages {
		_ = writer.Write([]string{pkg.Name, pkg.Category, pkg.Description, fmt.Sprintf("%.2f", pkg.Price), pkg.Items, pkg.Status})
	}
	writer.Flush()
	h.recordOperation(c, "export", "package", "", "success", fmt.Sprintf("%d packages", len(packages)))
}

func (h *Handler) importPackages(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "csv file is required"})
		return
	}
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "open csv file failed"})
		return
	}
	defer opened.Close()
	reader := csv.NewReader(opened)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read csv header failed"})
		return
	}
	index := map[string]int{}
	for i, name := range header {
		index[strings.ToLower(strings.TrimSpace(name))] = i
	}
	required := []string{"name", "category", "price"}
	for _, field := range required {
		if _, ok := index[field]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing csv column: " + field})
			return
		}
	}
	imported := 0
	updated := 0
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			name := csvValue(record, index, "name")
			if name == "" {
				continue
			}
			price, err := strconv.ParseFloat(csvValue(record, index, "price"), 64)
			if err != nil {
				return fmt.Errorf("invalid price for %s", name)
			}
			pkg := models.CheckupPackage{
				Name:        name,
				Category:    normalizeStatus(csvValue(record, index, "category"), "综合体检"),
				Description: csvValue(record, index, "description"),
				Price:       price,
				Items:       csvValue(record, index, "items"),
				Status:      normalizeStatus(csvValue(record, index, "status"), "active"),
			}
			var existing models.CheckupPackage
			err = tx.Where("name = ?", name).First(&existing).Error
			if err == nil {
				if err := tx.Model(&existing).Updates(pkg).Error; err != nil {
					return err
				}
				updated++
				continue
			}
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err := tx.Create(&pkg).Error; err != nil {
				return err
			}
			imported++
		}
		return nil
	}); err != nil {
		h.recordOperation(c, "import", "package", "", "failed", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	detail := fmt.Sprintf("created=%d updated=%d", imported, updated)
	h.recordOperation(c, "import", "package", "", "success", detail)
	c.JSON(http.StatusOK, gin.H{"created": imported, "updated": updated})
}

func (h *Handler) checkupItems(c *gin.Context) {
	var items []models.CheckupItem
	query := h.db.Model(&models.CheckupItem{}).Order("category asc, name asc")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", "deleted")
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR category LIKE ? OR department LIKE ?", pattern, pattern, pattern)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &items)
		return
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createCheckupItem(c *gin.Context) {
	var req checkupItemRequest
	if !bind(c, &req) {
		return
	}
	item := models.CheckupItem{
		Name:        strings.TrimSpace(req.Name),
		Category:    strings.TrimSpace(req.Category),
		Department:  strings.TrimSpace(req.Department),
		Price:       req.Price,
		DurationMin: req.DurationMin,
		Description: req.Description,
		Status:      normalizeStatus(req.Status, "active"),
	}
	if item.DurationMin <= 0 {
		item.DurationMin = 10
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "create", "checkup_item", strconv.Itoa(int(item.ID)), "success", item.Name)
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateCheckupItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkup item id"})
		return
	}
	var req checkupItemRequest
	if !bind(c, &req) {
		return
	}
	duration := req.DurationMin
	if duration <= 0 {
		duration = 10
	}
	updates := map[string]any{
		"name":         strings.TrimSpace(req.Name),
		"category":     strings.TrimSpace(req.Category),
		"department":   strings.TrimSpace(req.Department),
		"price":        req.Price,
		"duration_min": duration,
		"description":  req.Description,
		"status":       normalizeStatus(req.Status, "active"),
	}
	if err := h.db.Model(&models.CheckupItem{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.CheckupItem
	h.db.First(&item, id)
	h.recordOperation(c, "update", "checkup_item", strconv.Itoa(id), "success", item.Name)
	c.JSON(http.StatusOK, item)
}

func (h *Handler) archiveCheckupItem(c *gin.Context) {
	id, ok := parseIDParam(c, "id", "invalid checkup item id")
	if !ok {
		return
	}
	if err := h.archiveByID(&models.CheckupItem{}, id, "checkup_item"); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "archive", "checkup_item", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) packageItems(c *gin.Context) {
	var items []models.PackageItem
	query := h.db.Model(&models.PackageItem{}).Preload("Package").Preload("Item").Order("package_id asc, sort_order asc, id asc")
	if packageID := c.Query("packageId"); packageID != "" {
		query = query.Where("package_id = ?", packageID)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &items)
		return
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) upsertPackageItem(c *gin.Context) {
	var req packageItemRequest
	if !bind(c, &req) {
		return
	}
	link := models.PackageItem{
		PackageID: req.PackageID,
		ItemID:    req.ItemID,
		SortOrder: req.SortOrder,
		Required:  req.Required,
	}
	if err := h.db.Where("package_id = ? AND item_id = ?", req.PackageID, req.ItemID).Assign(link).FirstOrCreate(&link).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("Package").Preload("Item").First(&link, link.ID)
	h.recordOperation(c, "upsert", "package_item", strconv.Itoa(int(link.ID)), "success", fmt.Sprintf("package=%d item=%d", req.PackageID, req.ItemID))
	c.JSON(http.StatusOK, link)
}

func (h *Handler) deletePackageItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package item id"})
		return
	}
	result := h.db.Delete(&models.PackageItem{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "package item not found"})
		return
	}
	h.recordOperation(c, "delete", "package_item", strconv.Itoa(id), "success", "")
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "deleted"})
}

func (h *Handler) users(c *gin.Context) {
	var users []models.User
	query := h.db.Model(&models.User{}).Order("created_at desc")
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR email LIKE ? OR employee_no LIKE ? OR department LIKE ?", pattern, pattern, pattern, pattern)
	}
	if page, pageSize, ok := paginationParams(c); ok {
		respondPaginated(c, query, page, pageSize, &users)
		return
	}
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) updateUserStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req statusRequest
	if !bind(c, &req) {
		return
	}
	if req.Status != "active" && req.Status != "pending" && req.Status != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user status"})
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "update_status", "user", strconv.Itoa(id), "success", req.Status)
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

func (h *Handler) updateDoctorProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req doctorProfileRequest
	if !bind(c, &req) {
		return
	}
	department := strings.TrimSpace(req.Department)
	specialties := strings.TrimSpace(req.Specialties)
	if department == "" || specialties == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department and specialties are required"})
		return
	}
	updates := map[string]any{
		"department":  department,
		"specialties": specialties,
	}
	if strings.TrimSpace(req.Title) != "" {
		updates["title"] = strings.TrimSpace(req.Title)
	}
	categories := splitCSV(specialties)
	if len(categories) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "specialties are required"})
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.User{}).Where("id = ? AND role = ?", id, "doctor").Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return syncDoctorAvailableSlots(tx, uint(id), categories)
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "doctor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "doctor not found"})
		return
	}
	h.recordOperation(c, "update_profile", "doctor", strconv.Itoa(id), "success", user.Name)
	c.JSON(http.StatusOK, user)
}

func (h *Handler) seed(c *gin.Context) {
	if err := seed.Run(h.db); err != nil {
		h.recordOperation(c, "reset", "seed", "", "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.recordOperation(c, "reset", "seed", "", "success", "seed data rebuilt")
	c.JSON(http.StatusOK, gin.H{"status": "seeded"})
}

func (h *Handler) promoteWaitlist(tx *gorm.DB, slotID uint) error {
	var slot models.ScheduleSlot
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&slot, slotID).Error; err != nil {
		return err
	}
	if slot.BookedCount >= slot.Capacity || slot.Status != "available" {
		return nil
	}
	var wait models.WaitlistEntry
	if err := tx.Where("date = ? AND period = ? AND institution_id = ? AND category = ? AND status = ?", slot.Date, slot.Period, slot.InstitutionID, slot.Category, "waiting").
		Where("(start_time = ? OR start_time = '')", slot.StartTime).
		Order("created_at asc").
		First(&wait).Error; err != nil {
		return nil
	}
	appointment := models.Appointment{
		OrderNo:         generateOrderNo(),
		UserID:          wait.UserID,
		DoctorID:        slot.DoctorID,
		InstitutionID:   slot.InstitutionID,
		SlotID:          slot.ID,
		PackageID:       wait.PackageID,
		AppointmentType: wait.AppointmentType,
		Category:        wait.Category,
		Date:            slot.Date,
		Period:          slot.Period,
		StartTime:       slot.StartTime,
		EndTime:         slot.EndTime,
		Status:          "booked",
		Note:            wait.Note,
	}
	if err := tx.Create(&appointment).Error; err != nil {
		return err
	}
	if err := tx.Model(&slot).Update("booked_count", slot.BookedCount+1).Error; err != nil {
		return err
	}
	if err := tx.Model(&wait).Update("status", "promoted").Error; err != nil {
		return err
	}
	tx.Preload("User").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").First(&appointment, appointment.ID)
	go h.sendAppointmentMail(appointment, "候补预约成功")
	return nil
}

func (h *Handler) sendAppointmentMail(appointment models.Appointment, subject string) {
	if appointment.User.Email == "" || !appointment.User.EmailNotify {
		return
	}
	body := renderAppointmentHTML(appointment)
	h.recordMail(appointment.UserID, appointment.User.Email, subject, body, h.mailer.SendHTML(appointment.User.Email, subject, body))
}

func (h *Handler) sendReportMail(report models.Report) {
	if report.User.Email == "" || !report.User.EmailNotify {
		return
	}
	body := renderReportHTML(report)
	h.recordMail(report.UserID, report.User.Email, "体检报告已生成", body, h.mailer.SendHTML(report.User.Email, "体检报告已生成", body))
}

func (h *Handler) recordMail(userID uint, to, subject, body string, err error) {
	status := "sent"
	errText := ""
	if err != nil {
		status = "failed"
		errText = err.Error()
	}
	h.db.Create(&models.MailLog{UserID: userID, To: to, Subject: subject, Body: body, Status: status, Error: errText})
}

func (h *Handler) recordLogin(c *gin.Context, userID uint, email, role, status, reason string) {
	h.db.Create(&models.LoginLog{
		UserID:    userID,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Role:      role,
		IP:        c.ClientIP(),
		UserAgent: trimForLog(c.Request.UserAgent(), 255),
		Status:    status,
		Reason:    trimForLog(reason, 255),
	})
}

func (h *Handler) recordOperation(c *gin.Context, action, resource, resourceID, status, detail string) {
	user := currentUser(c)
	h.db.Create(&models.OperationLog{
		UserID:     user.ID,
		UserName:   user.Name,
		Role:       user.Role,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Method:     c.Request.Method,
		Path:       c.FullPath(),
		IP:         c.ClientIP(),
		Status:     status,
		Detail:     detail,
	})
}

func (h *Handler) permissionsForRole(role string) []string {
	var rows []models.RolePermission
	if err := h.db.Where("role = ? AND enabled = ?", role, true).Find(&rows).Error; err == nil && len(rows) > 0 {
		permissions := make([]string, 0, len(rows))
		for _, row := range rows {
			permissions = append(permissions, row.Permission)
		}
		return permissions
	}
	return fallbackPermissions(role)
}

func fallbackPermissions(role string) []string {
	switch role {
	case "user":
		return []string{"appointment:create", "appointment:reschedule", "appointment:cancel", "review:create", "favorite:manage", "family:manage", "report:view"}
	case "doctor":
		return []string{"doctor:appointment:update", "report:create", "customer:view"}
	case "admin":
		return []string{"admin:user:manage", "admin:doctor:review", "admin:package:manage", "admin:resource:manage", "admin:operation:manage", "admin:system:manage", "admin:data:exchange", "admin:permission:manage"}
	default:
		return []string{}
	}
}

func (h *Handler) familyMemberBelongsTo(userID, familyMemberID uint) bool {
	var count int64
	h.db.Model(&models.FamilyMember{}).Where("id = ? AND user_id = ?", familyMemberID, userID).Count(&count)
	return count > 0
}

func (h *Handler) createAppointmentNotifications(appointment models.Appointment, kind, title, content string) {
	if appointment.UserID == 0 {
		return
	}
	body := content
	if appointment.Date != "" {
		body = fmt.Sprintf("%s 时间：%s %s-%s，机构：%s，套餐：%s。", content, appointment.Date, appointment.StartTime, appointment.EndTime, appointment.Institution.Name, appointment.Package.Name)
	}
	h.db.Create(&models.Notification{UserID: appointment.UserID, Channel: "in_app", Type: kind, Title: title, Content: body, Status: "unread"})
	h.db.Create(&models.Notification{UserID: appointment.UserID, Channel: "sms_mock", Type: kind, Title: "短信模拟：" + title, Content: body, Status: "unread"})
	if appointment.Status == "booked" {
		h.db.Create(&models.Notification{UserID: appointment.UserID, Channel: "in_app", Type: "checkup_reminder", Title: "体检前提醒", Content: "请携带有效证件，按预约时间到达体检机构。体检前一天建议清淡饮食，部分项目需空腹。", Status: "unread"})
	}
}

func (h *Handler) buildScheduleSlot(c *gin.Context, req scheduleSlotRequest, bookedCount int) (models.ScheduleSlot, bool) {
	var doctor models.User
	if err := h.db.Where("id = ? AND role = ? AND status = ?", req.DoctorID, "doctor", "active").First(&doctor).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active doctor not found"})
		return models.ScheduleSlot{}, false
	}
	var institution models.CheckupInstitution
	if err := h.db.Where("id = ? AND status = ?", req.InstitutionID, "active").First(&institution).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "active institution not found"})
		return models.ScheduleSlot{}, false
	}
	capacity := req.Capacity
	if capacity <= 0 {
		capacity = 1
	}
	if capacity < bookedCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "capacity cannot be lower than booked count"})
		return models.ScheduleSlot{}, false
	}
	endTime := strings.TrimSpace(req.EndTime)
	if endTime == "" {
		var err error
		endTime, err = addMinutes(req.StartTime, 30)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
			return models.ScheduleSlot{}, false
		}
	}
	period := strings.TrimSpace(req.Period)
	if period == "" {
		period = "上午"
		if req.StartTime >= "12:00" {
			period = "下午"
		}
	}
	return models.ScheduleSlot{
		DoctorID:      req.DoctorID,
		InstitutionID: req.InstitutionID,
		Date:          req.Date,
		Period:        period,
		Category:      strings.TrimSpace(req.Category),
		StartTime:     req.StartTime,
		EndTime:       endTime,
		Capacity:      capacity,
		BookedCount:   bookedCount,
		Status:        normalizeStatus(req.Status, "available"),
	}, true
}

func (h *Handler) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenText := bearerToken(c.GetHeader("Authorization"))
		if tokenText == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := auth.ParseToken(tokenText, h.config.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		sessionUserID, err := h.redis.Get(c.Request.Context(), auth.SessionKey(claims.SessionID)).Result()
		if err != nil || sessionUserID != strconv.Itoa(int(claims.UserID)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}
		var user models.User
		if err := h.db.First(&user, claims.UserID).Error; err != nil || user.Status != "active" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}
		c.Set("user", user)
		c.Set("claims", claims)
		c.Next()
	}
}

func (h *Handler) requireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		user := currentUser(c)
		if !allowed[user.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}

func currentUser(c *gin.Context) models.User {
	value, _ := c.Get("user")
	user, _ := value.(models.User)
	return user
}

func currentClaims(c *gin.Context) *auth.Claims {
	value, _ := c.Get("claims")
	claims, _ := value.(*auth.Claims)
	return claims
}

func bearerToken(header string) string {
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Code     string `json:"code"`
	Role     string `json:"role"`
}

type registerUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
	IDCard   string `json:"idCard"`
}

type registerDoctorRequest struct {
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Code       string `json:"code" binding:"required"`
	Password   string `json:"password" binding:"required"`
	EmployeeNo string `json:"employeeNo" binding:"required"`
	Department string `json:"department" binding:"required"`
	Title      string `json:"title" binding:"required"`
}

type appointmentRequest struct {
	PackageID       uint   `json:"packageId" binding:"required"`
	InstitutionID   uint   `json:"institutionId" binding:"required"`
	FamilyMemberID  uint   `json:"familyMemberId"`
	SlotID          uint   `json:"slotId"`
	AppointmentType string `json:"appointmentType" binding:"required"`
	Date            string `json:"date" binding:"required"`
	Period          string `json:"period" binding:"required"`
	Note            string `json:"note"`
	PaymentStatus   string `json:"paymentStatus"`
	InvoiceTitle    string `json:"invoiceTitle"`
	InvoiceTaxNo    string `json:"invoiceTaxNo"`
}

type rescheduleRequest struct {
	InstitutionID uint   `json:"institutionId" binding:"required"`
	SlotID        uint   `json:"slotId"`
	Date          string `json:"date" binding:"required"`
	Period        string `json:"period" binding:"required"`
	Note          string `json:"note"`
}

type profileRequest struct {
	Name        string `json:"name" binding:"required"`
	Gender      string `json:"gender"`
	Age         int    `json:"age"`
	IDCard      string `json:"idCard"`
	AvatarURL   string `json:"avatarUrl"`
	Bio         string `json:"bio"`
	EmailNotify bool   `json:"emailNotify"`
}

type emailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type updateEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type packageRequest struct {
	Name        string  `json:"name" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required"`
	Items       string  `json:"items" binding:"required"`
	Status      string  `json:"status"`
}

type checkupItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Department  string  `json:"department"`
	Price       float64 `json:"price"`
	DurationMin int     `json:"durationMin"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
}

type packageItemRequest struct {
	PackageID uint `json:"packageId" binding:"required"`
	ItemID    uint `json:"itemId" binding:"required"`
	SortOrder int  `json:"sortOrder"`
	Required  bool `json:"required"`
}

type scheduleSlotRequest struct {
	DoctorID      uint   `json:"doctorId" binding:"required"`
	InstitutionID uint   `json:"institutionId" binding:"required"`
	Date          string `json:"date" binding:"required"`
	Period        string `json:"period"`
	Category      string `json:"category" binding:"required"`
	StartTime     string `json:"startTime" binding:"required"`
	EndTime       string `json:"endTime"`
	Capacity      int    `json:"capacity"`
	Status        string `json:"status"`
}

type reportRequest struct {
	AppointmentID  uint   `json:"appointmentId" binding:"required"`
	Summary        string `json:"summary" binding:"required"`
	Conclusion     string `json:"conclusion" binding:"required"`
	Recommendation string `json:"recommendation"`
}

type reviewRequest struct {
	AppointmentID uint   `json:"appointmentId" binding:"required"`
	Rating        int    `json:"rating" binding:"required"`
	Content       string `json:"content"`
}

type reviewReplyRequest struct {
	Reply  string `json:"reply"`
	Status string `json:"status"`
}

type couponRequest struct {
	Name        string  `json:"name" binding:"required"`
	Code        string  `json:"code" binding:"required"`
	Type        string  `json:"type"`
	Value       float64 `json:"value" binding:"required"`
	MinAmount   float64 `json:"minAmount"`
	PackageID   uint    `json:"packageId"`
	Status      string  `json:"status"`
	StartDate   string  `json:"startDate"`
	EndDate     string  `json:"endDate"`
	Description string  `json:"description"`
}

type announcementRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Audience string `json:"audience"`
	Status   string `json:"status"`
}

type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

type rolePermissionRequest struct {
	Enabled bool `json:"enabled"`
}

type systemSettingRequest struct {
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type doctorProfileRequest struct {
	Department  string `json:"department" binding:"required"`
	Title       string `json:"title"`
	Specialties string `json:"specialties" binding:"required"`
}

type familyMemberRequest struct {
	Name     string `json:"name" binding:"required"`
	Relation string `json:"relation" binding:"required"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
	IDCard   string `json:"idCard"`
	Phone    string `json:"phone"`
}

func normalizeStatus(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
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

func csvValue(record []string, index map[string]int, key string) string {
	i, ok := index[key]
	if !ok || i >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[i])
}

func trimForLog(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func parseIDParam(c *gin.Context, name, message string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return 0, false
	}
	return id, true
}

func (h *Handler) archiveByID(model any, id int, resource string) error {
	result := h.db.Model(model).Where("id = ? AND status <> ?", id, "deleted").Update("status", "deleted")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%s not found", resource)
	}
	return nil
}

func validateSettingValue(valueType, value string) error {
	switch normalizeStatus(valueType, "string") {
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("setting value must be numeric")
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("setting value must be true or false")
		}
	case "string", "json":
		return nil
	default:
		return fmt.Errorf("unsupported setting value type")
	}
	return nil
}

func syncDoctorAvailableSlots(tx *gorm.DB, doctorID uint, categories []string) error {
	var slots []models.ScheduleSlot
	if err := tx.Where("doctor_id = ? AND status = ? AND booked_count = 0", doctorID, "available").
		Order("date asc, start_time asc, id asc").
		Find(&slots).Error; err != nil {
		return err
	}
	for index, slot := range slots {
		category := categories[index%len(categories)]
		if slot.Category == category {
			continue
		}
		if err := tx.Model(&models.ScheduleSlot{}).Where("id = ?", slot.ID).Update("category", category).Error; err != nil {
			return err
		}
	}
	return nil
}

func paginationParams(c *gin.Context) (int, int, bool) {
	if c.Query("page") == "" && c.Query("pageSize") == "" {
		return 0, 0, false
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, true
}

func respondPaginated[T any](c *gin.Context, query *gorm.DB, page, pageSize int, dest *[]T) {
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(dest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    *dest,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) verifyAuthEmailCode(c *gin.Context, email, code string) bool {
	value, err := h.redis.Get(c.Request.Context(), authEmailCodeKey(email)).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email code expired"})
		return false
	}
	if value != code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email code"})
		return false
	}
	return true
}

func (h *Handler) emailExists(email string) bool {
	var count int64
	h.db.Model(&models.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

func syntheticPhone(email string) string {
	return "E" + emailHash(email)[:18]
}

func emailHash(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func authEmailCodeKey(email string) string {
	return "auth-email-code:" + emailHash(email)
}

func authEmailCodeCooldownKey(email string) string {
	return "auth-email-code-cooldown:" + emailHash(email)
}

func emailCodeKey(userID uint) string {
	return fmt.Sprintf("email-code:%d", userID)
}

func emailCodeCooldownKey(userID uint) string {
	return fmt.Sprintf("email-code-cooldown:%d", userID)
}

func generateEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateOrderNo() string {
	return "YY" + time.Now().Format("20060102150405") + randomDigits(4)
}

func generateReportNo() string {
	return "BG" + time.Now().Format("20060102150405") + randomDigits(4)
}

func randomDigits(length int) string {
	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return strings.Repeat("0", length)
	}
	return fmt.Sprintf("%0*d", length, n.Int64())
}

func renderAppointmentHTML(appointment models.Appointment) string {
	rows := [][2]string{
		{"订单号", appointment.OrderNo},
		{"客户", appointment.User.Name},
		{"预约类型", appointment.AppointmentType},
		{"体检分类", appointment.Category},
		{"体检机构", appointment.Institution.Name},
		{"机构地址", appointment.Institution.Address},
		{"套餐", appointment.Package.Name},
		{"项目明细", appointment.Package.Items},
		{"医生", appointment.Doctor.Name + " " + appointment.Doctor.Title},
		{"日期", appointment.Date},
		{"时间", appointment.StartTime + "-" + appointment.EndTime},
		{"备注", appointment.Note},
		{"状态", appointment.Status},
	}
	return renderDocumentHTML("体检预约订单", rows, "请按预约时间携带有效证件到检。")
}

func renderReportHTML(report models.Report) string {
	rows := [][2]string{
		{"报告编号", report.ReportNo},
		{"订单号", report.Appointment.OrderNo},
		{"客户", report.User.Name},
		{"体检机构", report.Appointment.Institution.Name},
		{"套餐", report.Appointment.Package.Name},
		{"医生", report.Doctor.Name + " " + report.Doctor.Title},
		{"检查摘要", report.Summary},
		{"体检结论", report.Conclusion},
		{"健康建议", report.Recommendation},
		{"报告时间", report.CreatedAt.Format("2006-01-02 15:04")},
	}
	return renderDocumentHTML("体检报告详情", rows, "本报告仅供健康管理参考，如有不适请及时就医。")
}

func renderDocumentHTML(title string, rows [][2]string, footer string) string {
	var builder strings.Builder
	builder.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>`)
	builder.WriteString(template.HTMLEscapeString(title))
	builder.WriteString(`</title><style>body{font-family:Arial,"Microsoft YaHei",sans-serif;margin:0;background:#f3f6fa;color:#1f2d3d}.doc{max-width:860px;margin:32px auto;background:#fff;border:1px solid #d8e2ec;padding:32px}.head{border-bottom:3px solid #1f78b4;padding-bottom:16px;margin-bottom:24px}.head h1{margin:0;font-size:28px}.head p{margin:8px 0 0;color:#6b7c8f}.grid{display:grid;grid-template-columns:160px 1fr;border-top:1px solid #e3ebf2;border-left:1px solid #e3ebf2}.label,.value{padding:13px 16px;border-right:1px solid #e3ebf2;border-bottom:1px solid #e3ebf2}.label{font-weight:700;background:#f8fafc}.value{white-space:pre-wrap}.footer{margin-top:24px;color:#6b7c8f}</style></head><body><main class="doc"><section class="head"><h1>`)
	builder.WriteString(template.HTMLEscapeString(title))
	builder.WriteString(`</h1><p>东软熙心健康体检管理系统</p></section><section class="grid">`)
	for _, row := range rows {
		builder.WriteString(`<div class="label">`)
		builder.WriteString(template.HTMLEscapeString(row[0]))
		builder.WriteString(`</div><div class="value">`)
		builder.WriteString(template.HTMLEscapeString(row[1]))
		builder.WriteString(`</div>`)
	}
	builder.WriteString(`</section><p class="footer">`)
	builder.WriteString(template.HTMLEscapeString(footer))
	builder.WriteString(`</p></main></body></html>`)
	return builder.String()
}
