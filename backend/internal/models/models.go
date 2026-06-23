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
	Specialties  string    `json:"specialties" gorm:"type:text"`
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
	Category    string    `json:"category" gorm:"size:64;not null;default:'综合体检'"`
	Description string    `json:"description" gorm:"type:text"`
	Price       float64   `json:"price" gorm:"not null"`
	Items       string    `json:"items" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:16;not null;default:'active'"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CheckupItem struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Category    string    `json:"category" gorm:"size:64;not null"`
	Department  string    `json:"department" gorm:"size:64"`
	Price       float64   `json:"price" gorm:"not null;default:0"`
	DurationMin int       `json:"durationMin" gorm:"not null;default:10"`
	Description string    `json:"description" gorm:"type:text"`
	Status      string    `json:"status" gorm:"size:24;not null;default:'active';index"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PackageItem struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	PackageID uint           `json:"packageId" gorm:"not null;uniqueIndex:idx_package_item"`
	ItemID    uint           `json:"itemId" gorm:"not null;uniqueIndex:idx_package_item"`
	SortOrder int            `json:"sortOrder" gorm:"not null;default:0"`
	Required  bool           `json:"required" gorm:"not null;default:true"`
	Package   CheckupPackage `json:"package"`
	Item      CheckupItem    `json:"item"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type Coupon struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Code        string    `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Type        string    `json:"type" gorm:"size:24;not null;default:'amount'"`
	Value       float64   `json:"value" gorm:"not null"`
	MinAmount   float64   `json:"minAmount" gorm:"not null;default:0"`
	PackageID   uint      `json:"packageId" gorm:"index"`
	Status      string    `json:"status" gorm:"size:24;not null;default:'active';index"`
	StartDate   string    `json:"startDate" gorm:"size:16"`
	EndDate     string    `json:"endDate" gorm:"size:16"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Appointment struct {
	ID              uint               `json:"id" gorm:"primaryKey"`
	OrderNo         string             `json:"orderNo" gorm:"size:32;uniqueIndex"`
	UserID          uint               `json:"userId" gorm:"not null;index"`
	FamilyMemberID  uint               `json:"familyMemberId" gorm:"index"`
	DoctorID        uint               `json:"doctorId" gorm:"index"`
	InstitutionID   uint               `json:"institutionId" gorm:"not null;index"`
	SlotID          uint               `json:"slotId" gorm:"index"`
	PackageID       uint               `json:"packageId" gorm:"not null;index"`
	AppointmentType string             `json:"appointmentType" gorm:"size:32;not null;default:'个人体检'"`
	Category        string             `json:"category" gorm:"size:64;not null;default:'综合体检';index"`
	Date            string             `json:"date" gorm:"size:16;not null"`
	Period          string             `json:"period" gorm:"size:32;not null"`
	StartTime       string             `json:"startTime" gorm:"size:8"`
	EndTime         string             `json:"endTime" gorm:"size:8"`
	Status          string             `json:"status" gorm:"size:24;not null;default:'booked'"`
	Note            string             `json:"note" gorm:"type:text"`
	PaymentStatus   string             `json:"paymentStatus" gorm:"size:24;not null;default:'unpaid'"`
	InvoiceTitle    string             `json:"invoiceTitle" gorm:"size:128"`
	InvoiceTaxNo    string             `json:"invoiceTaxNo" gorm:"size:64"`
	User            User               `json:"user"`
	FamilyMember    FamilyMember       `json:"familyMember"`
	Doctor          User               `json:"doctor"`
	Institution     CheckupInstitution `json:"institution"`
	Package         CheckupPackage     `json:"package"`
	Slot            ScheduleSlot       `json:"slot"`
	Report          *Report            `json:"report,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type FamilyMember struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	Relation  string    `json:"relation" gorm:"size:32;not null"`
	Gender    string    `json:"gender" gorm:"size:16"`
	Age       int       `json:"age"`
	IDCard    string    `json:"idCard" gorm:"size:32"`
	Phone     string    `json:"phone" gorm:"size:32"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PackageFavorite struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"userId" gorm:"not null;uniqueIndex:idx_user_package_favorite"`
	PackageID uint           `json:"packageId" gorm:"not null;uniqueIndex:idx_user_package_favorite"`
	Package   CheckupPackage `json:"package"`
	CreatedAt time.Time      `json:"createdAt"`
}

type PackageBrowseHistory struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"userId" gorm:"not null;uniqueIndex:idx_user_package_browse"`
	PackageID uint           `json:"packageId" gorm:"not null;uniqueIndex:idx_user_package_browse"`
	Package   CheckupPackage `json:"package"`
	ViewCount int            `json:"viewCount" gorm:"not null;default:1"`
	ViewedAt  time.Time      `json:"viewedAt" gorm:"index"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type Notification struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"userId" gorm:"not null;index"`
	Channel   string     `json:"channel" gorm:"size:24;not null;default:'in_app'"`
	Type      string     `json:"type" gorm:"size:32;not null"`
	Title     string     `json:"title" gorm:"size:128;not null"`
	Content   string     `json:"content" gorm:"type:text"`
	Status    string     `json:"status" gorm:"size:24;not null;default:'unread';index"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type ServiceReview struct {
	ID            uint               `json:"id" gorm:"primaryKey"`
	UserID        uint               `json:"userId" gorm:"not null;index"`
	AppointmentID uint               `json:"appointmentId" gorm:"not null;uniqueIndex"`
	PackageID     uint               `json:"packageId" gorm:"not null;index"`
	InstitutionID uint               `json:"institutionId" gorm:"not null;index"`
	DoctorID      uint               `json:"doctorId" gorm:"index"`
	Rating        int                `json:"rating" gorm:"not null"`
	Content       string             `json:"content" gorm:"type:text"`
	Reply         string             `json:"reply" gorm:"type:text"`
	Status        string             `json:"status" gorm:"size:24;not null;default:'published';index"`
	User          User               `json:"user"`
	Appointment   Appointment        `json:"appointment"`
	Package       CheckupPackage     `json:"package"`
	Institution   CheckupInstitution `json:"institution"`
	Doctor        User               `json:"doctor"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

type SystemAnnouncement struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"size:128;not null"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	Audience    string     `json:"audience" gorm:"size:24;not null;default:'all';index"`
	Status      string     `json:"status" gorm:"size:24;not null;default:'draft';index"`
	PublishedAt *time.Time `json:"publishedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type ScheduleSlot struct {
	ID            uint               `json:"id" gorm:"primaryKey"`
	DoctorID      uint               `json:"doctorId" gorm:"not null;index"`
	InstitutionID uint               `json:"institutionId" gorm:"not null;index"`
	Date          string             `json:"date" gorm:"size:16;not null;index"`
	Period        string             `json:"period" gorm:"size:32;not null;index"`
	Category      string             `json:"category" gorm:"size:64;not null;default:'综合体检';index"`
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
	Category        string             `json:"category" gorm:"size:64;not null;default:'综合体检';index"`
	Date            string             `json:"date" gorm:"size:16;not null;index"`
	Period          string             `json:"period" gorm:"size:32;not null;index"`
	StartTime       string             `json:"startTime" gorm:"size:8;index"`
	EndTime         string             `json:"endTime" gorm:"size:8"`
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
