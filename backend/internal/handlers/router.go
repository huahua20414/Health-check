package handlers

import (
	"net/http"
	"strconv"

	"health-checkup/backend/internal/models"
	"health-checkup/backend/internal/seed"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewRouter(db *gorm.DB) *gin.Engine {
	handler := &Handler{db: db}
	router := gin.Default()
	router.Use(cors.Default())

	api := router.Group("/api")
	api.GET("/health", handler.health)
	api.POST("/login", handler.login)
	api.GET("/packages", handler.packages)
	api.GET("/appointments", handler.appointments)
	api.POST("/appointments", handler.createAppointment)
	api.PATCH("/appointments/:id/status", handler.updateAppointmentStatus)
	api.GET("/reports", handler.reports)
	api.POST("/reports", handler.createReport)
	api.POST("/seed", handler.seed)

	return router
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Role  string `json:"role" binding:"required"`
		Name  string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := req.Name
	if name == "" && req.Role == "doctor" {
		name = "体检医生"
	}
	if name == "" {
		name = "体检用户"
	}
	user := models.User{Phone: req.Phone}
	if err := h.db.Where(models.User{Phone: req.Phone}).FirstOrCreate(&user, models.User{Name: name, Phone: req.Phone, Role: req.Role}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
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
	var appointments []models.Appointment
	query := h.db.Preload("User").Preload("Package").Preload("Report").Order("created_at desc")
	if userID := c.Query("userId"); userID != "" {
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
	var req struct {
		UserID    uint   `json:"userId" binding:"required"`
		PackageID uint   `json:"packageId" binding:"required"`
		Date      string `json:"date" binding:"required"`
		Period    string `json:"period" binding:"required"`
		Note      string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	appointment := models.Appointment{
		UserID:    req.UserID,
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
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&models.Appointment{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}

func (h *Handler) reports(c *gin.Context) {
	var reports []models.Report
	query := h.db.Preload("Appointment.Package").Preload("User").Preload("Doctor").Order("created_at desc")
	if userID := c.Query("userId"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reports)
}

func (h *Handler) createReport(c *gin.Context) {
	var req struct {
		AppointmentID  uint   `json:"appointmentId" binding:"required"`
		DoctorID       uint   `json:"doctorId" binding:"required"`
		Summary        string `json:"summary" binding:"required"`
		Conclusion     string `json:"conclusion" binding:"required"`
		Recommendation string `json:"recommendation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var appointment models.Appointment
	if err := h.db.First(&appointment, req.AppointmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
		return
	}
	report := models.Report{
		AppointmentID:  appointment.ID,
		UserID:         appointment.UserID,
		DoctorID:       req.DoctorID,
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

func (h *Handler) seed(c *gin.Context) {
	if err := seed.Run(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "seeded"})
}
