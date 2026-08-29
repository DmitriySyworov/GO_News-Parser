package model

import (
	"time"

	"gorm.io/gorm"
)

type Users struct {
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
	Name      string         `gorm:"column:name"`
	Email     string         `gorm:"column:email"`
	Password  string         `gorm:"column:password"`
	UserUUID  string         `gorm:"column:user_uuid"`
}

type OutboxDeletedUsers struct {
	EventUUID        string   `gorm:"column:event_uuid;type:UUID;primaryKey"`
	UUIDDeletedUsers []string `gorm:"column:uuid_deleted_users;type:UUID[]"`
}
