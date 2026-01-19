package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID               uuid.UUID    `gorm:"type:char(36);primarykey"`
	Email              string       `gorm:"column:email"`
	Nickname           *string      `gorm:"column:nickname"`
	Signature          *string      `gorm:"column:signature"`
	Role               string       `gorm:"type:enum('interviewee', 'interviewer');default:'interviewee'"`
	Status             *string      `gorm:"column:status"`
	PassedDirections   *string      `gorm:"type:json"`
	PassedDirectionsBy *string      `gorm:"type:json"`
	Application        *Application `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Application struct {
	ID         uint      `gorm:"primarykey"`
	RealName   string    `gorm:"column:real_name;not null"`
	Phone      string    `gorm:"column:phone;not null"`
	Gender     string    `gorm:"type:enum('male', 'female');not null"`
	Department string    `gorm:"column:department;not null"`
	Major      string    `gorm:"column:major;not null"`
	StudentId  string    `gorm:"column:student_id;not null"`
	Directions *string   `gorm:"type:json"`
	Resume     string    `gorm:"column:resume;type:text;not null"`
	UserID     uuid.UUID `gorm:"type:char(36);uniqueIndex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
