package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"health-checkup/backend/internal/auth"
	"health-checkup/backend/internal/config"
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
}

func NewRouter(db *gorm.DB, redisClient *redis.Client, cfg config.Config) *gin.Engine {
	handler := &Handler{db: db, redis: redisClient, config: cfg}
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(middleware.IPRateLimit(120, time.Minute))

	api := router.Group("/api")
	api.GET("/health", handler.health)
	api.GET("/packages", handler.packages)

	authGroup := api.Group("/auth")
	authGroup.Use(middleware.IPRateLimit(20, time.Minute))
	authGroup.POST("/register/user", handler.registerUser)
	authGroup.POST("/register/doctor", handler.registerDoctor)
	authGroup.POST("/login", handler.login)

	protected := api.Group("")
	protected.Use(handler.authRequired())
	protected.POST("/auth/logout", handler.logout)
	protected.GET("/auth/me", handler.me)
	protected.GET("/appointments", handler.appointments)
	protected.POST("/appointments", handler.requireRole("user"), handler.createAppointment)
	protected.PATCH("/appointments/:id/status", handler.requireRole("doctor", "admin"), handler.updateAppointmentStatus)
	protected.GET("/reports", handler.reports)
	protected.POST("/reports", handler.requireRole("doctor"), handler.createReport)
	protected.GET("/users", handler.requireRole("doctor", "admin"), handler.users)
	protected.PATCH("/users/:id/status", handler.requireRole("admin"), handler.updateUserStatus)
	protected.POST("/seed", handler.requireRole("admin"), handler.seed)

	return router
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) registerUser(c *gin.Context) {
	var req registerUserRequest
	if !bind(c, &req) {
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := models.User{
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: passwordHash,
		Role:         "user",
		Status:       "active",
		Gender:       req.Gender,
		Age:          req.Age,
		IDCard:       req.IDCard,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone already exists or invalid user data"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) registerDoctor(c *gin.Context) {
	var req registerDoctorRequest
	if !bind(c, &req) {
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := models.User{
		Name:         strings.TrimSpace(req.Name),
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: passwordHash,
		Role:         "doctor",
		Status:       "pending",
		EmployeeNo:   req.EmployeeNo,
		Department:   req.Department,
		Title:        req.Title,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone already exists or invalid doctor data"})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if !bind(c, &req) {
		return
	}
	var user models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid phone or password"})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is not active", "status": user.Status})
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid phone or password"})
		return
	}
	token, err := auth.IssueToken(c.Request.Context(), h.redis, h.config.JWTSecret, time.Duration(h.config.TokenHours)*time.Hour, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue token failed"})
		return
	}
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

func (h *Handler) packages(c *gin.Context) {
	var packages []models.CheckupPackage
	if err := h.db.Order("price asc").Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, packages)
}

func (h *Handler) appointments(c *gin.Context) {
	current := currentUser(c)
	var appointments []models.Appointment
	query := h.db.Preload("User").Preload("Package").Preload("Report").Order("created_at desc")
	if current.Role == "user" {
		query = query.Where("user_id = ?", current.ID)
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
	appointment := models.Appointment{
		UserID:    current.ID,
		PackageID: req.PackageID,
		Date:      req.Date,
		Period:    req.Period,
		Status:    "booked",
		Note:      req.Note,
	}
	if err := h.db.Create(&appointment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.db.Preload("User").Preload("Package").First(&appointment, appointment.ID)
	c.JSON(http.StatusCreated, appointment)
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
	if req.Status != "booked" && req.Status != "checked" && req.Status != "reported" {
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
	query := h.db.Preload("Appointment.Package").Preload("User").Preload("Doctor").Order("created_at desc")
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
	report := models.Report{
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
	h.db.Preload("Appointment.Package").Preload("User").Preload("Doctor").First(&report, report.ID)
	c.JSON(http.StatusCreated, report)
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
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
	Gender   string `json:"gender"`
	Age      int    `json:"age"`
	IDCard   string `json:"idCard"`
}

type registerDoctorRequest struct {
	Name       string `json:"name" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required"`
	EmployeeNo string `json:"employeeNo" binding:"required"`
	Department string `json:"department" binding:"required"`
	Title      string `json:"title" binding:"required"`
}

type appointmentRequest struct {
	PackageID uint   `json:"packageId" binding:"required"`
	Date      string `json:"date" binding:"required"`
	Period    string `json:"period" binding:"required"`
	Note      string `json:"note"`
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
