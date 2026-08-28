package tenants

import (
	"context"
	"uuid"

	"github.com/duddy57/toaki-server/internal/members"
	"github.com/duddy57/toaki-server/internal/shared"
)

type Organizations struct {
	shared.Base `gorm:"embedded"`
	Name        string           `gorm:"type:varchar(255);not null" json:"name"`
	Slug        string           `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	OwnerId     uuid.UUID        `gorm:"type:uuid;not null" json:"owner_id"`
	Members     []members.Member `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE;" json:"members,omitempty"`
}
type CreateOrganization struct {
	Name string `gorm:"type:varchar(255);not null" json:"name"`
}
type CreateOrganizaationSuccess struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}
type UpdateRequest struct {
	Name string `form:"name" json:"name"`
}

type Service interface {
	CreateOrganization(context.Context, CreateOrganization) (uuid.UUID, error)
	GetOrganization(context.Context, uuid.UUID) (Organizations, error)
	DeleteOrganization(context.Context, uuid.UUID) error
	UpdateOrganization(context.Context, UpdateRequest, uuid.UUID) error

	// ActiveEmail(context.Context, ActiveEmail) error
}
