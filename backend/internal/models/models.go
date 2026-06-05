package models

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:64;not null"`
	Phone        string    `json:"phone" gorm:"size:64;uniqueIndex"`
	PasswordHash string    `json:"-" gorm:"size:255;not null;default:''"`
	Role         string    `json:"role" gorm:"size:16;not null"`
	Status       string    `json:"status" gorm:"size:16;not null;default:'active'"`
	Gender       string    `json:"gender" gorm:"size:16"`
	Age          int       `json:"age"`
	IDCard       string    `json:"idCard" gorm:"size:32"`
	Email        string    `json:"email" gorm:"size:128;index"`
	AvatarURL    string    `json:"avatarUrl" gorm:"size:255"`
	Bio          string    `json:"bio" gorm:"type:text"`
	EmailNotify  bool      `json:"emailNotify" gorm:"not null;default:true"`
	EmployeeNo   string    `json:"employeeNo" gorm:"size:32"`
	Department   string    `json:"department" gorm:"size:64"`
	Title        string    `json:"title" gorm:"size:64"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CheckupInstitution struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Address   string    `json:"address" gorm:"size:255;not null"`
	Phone     string    `json:"phone" gorm:"size:32"`
	OpenHours string    `json:"openHours" gorm:"size:128"`
	Status    string    `json:"status" gorm:"size:16;not null;default:'active'"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CheckupPackage struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Price       float64   `json:"price" gorm:"not null"`
	Items       string    `json:"items" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:16;not null;default:'active'"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Appointment struct {
	ID              uint               `json:"id" gorm:"primaryKey"`
	OrderNo         string             `json:"orderNo" gorm:"size:32;uniqueIndex"`
	UserID          uint               `json:"userId" gorm:"not null;index"`
	DoctorID        uint               `json:"doctorId" gorm:"index"`
	InstitutionID   uint               `json:"institutionId" gorm:"not null;index"`
	SlotID          uint               `json:"slotId" gorm:"index"`
	PackageID       uint               `json:"packageId" gorm:"not null;index"`
	AppointmentType string             `json:"appointmentType" gorm:"size:32;not null;default:'个人体检'"`
	Date            string             `json:"date" gorm:"size:16;not null"`
	Period          string             `json:"period" gorm:"size:32;not null"`
	StartTime       string             `json:"startTime" gorm:"size:8"`
	EndTime         string             `json:"endTime" gorm:"size:8"`
	Status          string             `json:"status" gorm:"size:24;not null;default:'booked'"`
	Note            string             `json:"note" gorm:"type:text"`
	User            User               `json:"user"`
	Doctor          User               `json:"doctor"`
	Institution     CheckupInstitution `json:"institution"`
	Package         CheckupPackage     `json:"package"`
	Slot            ScheduleSlot       `json:"slot"`
	Report          *Report            `json:"report,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type ScheduleSlot struct {
	ID            uint               `json:"id" gorm:"primaryKey"`
	DoctorID      uint               `json:"doctorId" gorm:"not null;index"`
	InstitutionID uint               `json:"institutionId" gorm:"not null;index"`
	Date          string             `json:"date" gorm:"size:16;not null;index"`
	Period        string             `json:"period" gorm:"size:32;not null;index"`
	StartTime     string             `json:"startTime" gorm:"size:8;not null"`
	EndTime       string             `json:"endTime" gorm:"size:8;not null"`
	Capacity      int                `json:"capacity" gorm:"not null;default:1"`
	BookedCount   int                `json:"bookedCount" gorm:"not null;default:0"`
	Status        string             `json:"status" gorm:"size:16;not null;default:'available'"`
	Doctor        User               `json:"doctor"`
	Institution   CheckupInstitution `json:"institution"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

type WaitlistEntry struct {
	ID              uint               `json:"id" gorm:"primaryKey"`
	UserID          uint               `json:"userId" gorm:"not null;index"`
	PackageID       uint               `json:"packageId" gorm:"not null;index"`
	InstitutionID   uint               `json:"institutionId" gorm:"not null;index"`
	AppointmentType string             `json:"appointmentType" gorm:"size:32;not null;default:'个人体检'"`
	Date            string             `json:"date" gorm:"size:16;not null;index"`
	Period          string             `json:"period" gorm:"size:32;not null;index"`
	Note            string             `json:"note" gorm:"type:text"`
	Status          string             `json:"status" gorm:"size:24;not null;default:'waiting'"`
	User            User               `json:"user"`
	Institution     CheckupInstitution `json:"institution"`
	Package         CheckupPackage     `json:"package"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type Report struct {
	ID             uint        `json:"id" gorm:"primaryKey"`
	ReportNo       string      `json:"reportNo" gorm:"size:32;uniqueIndex"`
	AppointmentID  uint        `json:"appointmentId" gorm:"uniqueIndex;not null"`
	UserID         uint        `json:"userId" gorm:"not null;index"`
	DoctorID       uint        `json:"doctorId" gorm:"not null;index"`
	Summary        string      `json:"summary" gorm:"type:text;not null"`
	Conclusion     string      `json:"conclusion" gorm:"type:text;not null"`
	Recommendation string      `json:"recommendation" gorm:"type:text"`
	Appointment    Appointment `json:"appointment"`
	User           User        `json:"user"`
	Doctor         User        `json:"doctor"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

type MailLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"index"`
	To        string    `json:"to" gorm:"size:128;not null"`
	Subject   string    `json:"subject" gorm:"size:255;not null"`
	Body      string    `json:"body" gorm:"type:text"`
	Status    string    `json:"status" gorm:"size:24;not null"`
	Error     string    `json:"error" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
}
