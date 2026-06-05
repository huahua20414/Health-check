package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
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
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(middleware.IPRateLimit(120, time.Minute))

	api := router.Group("/api")
	api.GET("/health", handler.health)
	api.GET("/packages", handler.packages)
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
	protected.GET("/schedule/slots", handler.scheduleSlots)
	protected.GET("/waitlist", handler.requireRole("user"), handler.waitlist)
	protected.PATCH("/appointments/:id/status", handler.requireRole("doctor", "admin"), handler.updateAppointmentStatus)
	protected.GET("/reports", handler.reports)
	protected.POST("/reports", handler.requireRole("doctor"), handler.createReport)
	protected.POST("/packages", handler.requireRole("admin"), handler.createPackage)
	protected.PATCH("/packages/:id", handler.requireRole("admin"), handler.updatePackage)
	protected.GET("/users", handler.requireRole("doctor", "admin"), handler.users)
	protected.PATCH("/users/:id/status", handler.requireRole("admin"), handler.updateUserStatus)
	protected.GET("/mail-logs", handler.requireRole("admin"), handler.mailLogs)
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
	if !h.verifyAuthEmailCode(c, email, req.Code) {
		return
	}
	var candidates []models.User
	if err := h.db.Where("email = ?", email).Find(&candidates).Error; err != nil || len(candidates) == 0 {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email, password or code"})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not active", "status": user.Status})
		return
	}
	token, err := auth.IssueToken(c.Request.Context(), h.redis, h.config.JWTSecret, time.Duration(h.config.TokenHours)*time.Hour, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
	h.redis.Del(c.Request.Context(), authEmailCodeKey(email))
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
	query := h.db.Order("price asc")
	if c.GetHeader("Authorization") == "" {
		query = query.Where("status = ?", "active")
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

func (h *Handler) appointments(c *gin.Context) {
	current := currentUser(c)
	var appointments []models.Appointment
	query := h.db.Preload("User").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").Preload("Report").Order("created_at desc")
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
	appointmentType := normalizeStatus(req.AppointmentType, "个人体检")
	var result any
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var slot models.ScheduleSlot
		query := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("status = ? AND booked_count < capacity AND institution_id = ?", "available", req.InstitutionID)
		if req.SlotID != 0 {
			query = query.Where("id = ?", req.SlotID)
		} else {
			query = query.Where("date = ? AND period = ?", req.Date, req.Period).Order("start_time asc")
		}
		err := query.First(&slot).Error
		if err != nil {
			waitDate := req.Date
			waitPeriod := req.Period
			waitInstitutionID := req.InstitutionID
			if req.SlotID != 0 {
				var requestedSlot models.ScheduleSlot
				if lookupErr := tx.First(&requestedSlot, req.SlotID).Error; lookupErr == nil {
					waitDate = requestedSlot.Date
					waitPeriod = requestedSlot.Period
					waitInstitutionID = requestedSlot.InstitutionID
				}
			}
			wait := models.WaitlistEntry{
				UserID:          current.ID,
				PackageID:       req.PackageID,
				InstitutionID:   waitInstitutionID,
				AppointmentType: appointmentType,
				Date:            waitDate,
				Period:          waitPeriod,
				Note:            req.Note,
				Status:          "waiting",
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
			DoctorID:        slot.DoctorID,
			InstitutionID:   slot.InstitutionID,
			SlotID:          slot.ID,
			PackageID:       req.PackageID,
			AppointmentType: appointmentType,
			Date:            slot.Date,
			Period:          slot.Period,
			StartTime:       slot.StartTime,
			EndTime:         slot.EndTime,
			Status:          "booked",
			Note:            req.Note,
		}
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		if err := tx.Model(&slot).Update("booked_count", slot.BookedCount+1).Error; err != nil {
			return err
		}
		tx.Preload("User").Preload("Doctor").Preload("Institution").Preload("Package").Preload("Slot").First(&appointment, appointment.ID)
		result = gin.H{"type": "appointment", "appointment": appointment}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payload, ok := result.(gin.H); ok && payload["type"] == "appointment" {
		if appointment, ok := payload["appointment"].(models.Appointment); ok {
			h.sendAppointmentMail(appointment, "体检预约成功")
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
	query := h.db.Preload("Appointment.Institution").Preload("Appointment.Package").Preload("User").Preload("Doctor").Order("created_at desc")
	if current.Role == "user" {
		query = query.Where("user_id = ?", current.ID)
	} else if userID := c.Query("userId"); userID != "" {
		query = query.Where("user_id = ?", userID)
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
	if date := c.Query("date"); date != "" {
		query = query.Where("date = ?", date)
	}
	if period := c.Query("period"); period != "" {
		query = query.Where("period = ?", period)
	}
	if err := query.Find(&slots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slots)
}

func (h *Handler) waitlist(c *gin.Context) {
	current := currentUser(c)
	var entries []models.WaitlistEntry
	if err := h.db.Preload("Institution").Preload("Package").Where("user_id = ?", current.ID).Order("created_at desc").Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) mailLogs(c *gin.Context) {
	var logs []models.MailLog
	if err := h.db.Order("created_at desc").Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) createPackage(c *gin.Context) {
	var req packageRequest
	if !bind(c, &req) {
		return
	}
	pkg := models.CheckupPackage{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Items:       req.Items,
		Status:      normalizeStatus(req.Status, "active"),
	}
	if err := h.db.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
	c.JSON(http.StatusOK, pkg)
}

func (h *Handler) users(c *gin.Context) {
	var users []models.User
	query := h.db.Order("created_at desc")
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
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
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

func (h *Handler) seed(c *gin.Context) {
	if err := seed.Run(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
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
	if err := tx.Where("date = ? AND period = ? AND institution_id = ? AND status = ?", slot.Date, slot.Period, slot.InstitutionID, "waiting").Order("created_at asc").First(&wait).Error; err != nil {
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
	Code     string `json:"code" binding:"required"`
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
	SlotID          uint   `json:"slotId"`
	AppointmentType string `json:"appointmentType" binding:"required"`
	Date            string `json:"date" binding:"required"`
	Period          string `json:"period" binding:"required"`
	Note            string `json:"note"`
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
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required"`
	Items       string  `json:"items" binding:"required"`
	Status      string  `json:"status"`
}

type reportRequest struct {
	AppointmentID  uint   `json:"appointmentId" binding:"required"`
	Summary        string `json:"summary" binding:"required"`
	Conclusion     string `json:"conclusion" binding:"required"`
	Recommendation string `json:"recommendation"`
}

type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

func normalizeStatus(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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
