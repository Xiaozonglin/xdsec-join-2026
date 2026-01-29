package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID               uuid.UUID    `gorm:"type:char(36);primarykey" json:"id"`
	Email              string       `gorm:"column:email" json:"email"`
	Nickname           *string      `gorm:"column:nickname" json:"nickname"`
	Signature          string       `gorm:"column:signature" json:"signature"`
	Role               string       `gorm:"type:enum('interviewee', 'interviewer');default:'interviewee'" json:"role"`
	Status             string       `gorm:"type:enum('r1_pending', 'r1_passed', 'r2_pending', 'r2_passed', 'rejected', 'offer');default:'r1_pending'" json:"status"`
	Directions         string       `gorm:"type:json" json:"directions"`
	PassedDirections   string       `gorm:"type:json" json:"passedDirections"`
	PassedDirectionsBy string       `gorm:"type:json" json:"passedDirectionsBy"`
	Application        *Application `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"application,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
	PassWord           string       `gorm:"column:password" json:"-"`
}

type Application struct {
	ID         uint      `gorm:"primarykey" json:"-"`
	RealName   string    `gorm:"column:real_name;not null" json:"realName"`
	Phone      string    `gorm:"column:phone;not null" json:"phone"`
	Gender     string    `gorm:"type:enum('male', 'female');not null" json:"gender"`
	Department string    `gorm:"column:department;not null" json:"department"`
	Major      string    `gorm:"column:major;not null" json:"major"`
	StudentId  string    `gorm:"column:student_id;not null" json:"studentId"`
	Directions string    `gorm:"type:json" json:"directions"`
	Resume     string    `gorm:"column:resume;type:text;not null" json:"resume"`
	UserID     uuid.UUID `gorm:"type:char(36);uniqueIndex" json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Announcement struct {
	UUID      uuid.UUID `gorm:"type:char(36);primarykey" json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Pinned    bool      `gorm:"column:pinned;default:false" json:"pinned"`
	AuthorId  uuid.UUID `gorm:"column:author_id" json:"authorId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Task struct {
	UUID         uuid.UUID `gorm:"type:char(36);primarykey" json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	TargetUserId uuid.UUID `gorm:"column:target_user_id" json:"targetUserId"`
	AssignedBy   uuid.UUID `gorm:"column:assigned_by" json:"assignedBy"`
	Report       string    `json:"report"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type EmailCode struct {
	UUID      uuid.UUID `gorm:"type:char(36);primarykey"`
	Email     string    `gorm:"column:email;index"`
	Code      string    `gorm:"column:code;length:6"`
	Purpose   string    `gorm:"type:enum('register', 'reset', 'profile')"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	Used      bool      `gorm:"column:used;default:false"`
	CreatedAt time.Time
}

type EmailRateLimit struct {
	UUID      uuid.UUID `gorm:"type:char(36);primarykey"`
	Email     string    `gorm:"column:email;index;not null"`
	LastSent  time.Time `gorm:"column:last_sent;not null"`
	CreatedAt time.Time
}

type Comment struct {
	UUID      uuid.UUID `gorm:"type:char(36);primarykey" json:"id"`
	Content   string    `gorm:"column:content;type:text;not null" json:"content"`
	IntervieweeID uuid.UUID `gorm:"column:interviewee_id;type:char(36);index;not null" json:"intervieweeId"`
	InterviewerID uuid.UUID `gorm:"column:interviewer_id;type:char(36);not null" json:"interviewerId"`
	CreatedAt time.Time `json:"createdAt"`
}
