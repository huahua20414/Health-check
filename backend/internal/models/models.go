package models

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:64;not null"`
	Phone        string    `json:"phone" gorm:"size:32;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null;default:''"`
	Role         string    `json:"role" gorm:"size:16;not null"`
	Status       string    `json:"status" gorm:"size:16;not null;default:'active'"`
	Gender       string    `json:"gender" gorm:"size:16"`
	Age          int       `json:"age"`
	IDCard       string    `json:"idCard" gorm:"size:32"`
	EmployeeNo   string    `json:"employeeNo" gorm:"size:32"`
	Department   string    `json:"department" gorm:"size:64"`
	Title        string    `json:"title" gorm:"size:64"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CheckupPackage struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Price       float64   `json:"price" gorm:"not null"`
	Items       string    `json:"items" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Appointment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"userId" gorm:"not null;index"`
	PackageID uint           `json:"packageId" gorm:"not null;index"`
	Date      string         `json:"date" gorm:"size:16;not null"`
	Period    string         `json:"period" gorm:"size:32;not null"`
	Status    string         `json:"status" gorm:"size:24;not null;default:'booked'"`
	Note      string         `json:"note" gorm:"type:text"`
	User      User           `json:"user"`
	Package   CheckupPackage `json:"package"`
	Report    *Report        `json:"report,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type Report struct {
	ID             uint        `json:"id" gorm:"primaryKey"`
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
