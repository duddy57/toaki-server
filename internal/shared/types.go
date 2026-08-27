package shared

import (
	"time"
	"uuid"

	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID     `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) (err error) {
	b.ID = uuid.NewV7()
	return
}
