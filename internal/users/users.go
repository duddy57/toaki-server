package users

import (
	"context"
	"uuid"

	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/duddy57/toaki-server/internal/tenants"
)

type Users struct {
	shared.Base `gorm:"embedded"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Email       string `gorm:"size:255;uniqueIndex;not null" json:"email"`

	Password []byte `gorm:"type:bytea;not null" json:"-"`
	Salt     []byte `gorm:"type:bytea;not null" json:"-"`

	Tenant *tenants.Organizations `gorm:"-" json:"organizations,omitzero"`
}

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,min=5,max=255"`
	Password string `json:"password" validate:"required,min=8,max=30"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,min=5,max=255"`
	Password string `json:"password" validate:"required,min=8,max=30"`
}
type UpdateRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

type RequestUpdatePassword struct {
	Email string `json:"email" validate:"required,email,min=5,max=255"`
}
type UpdatePasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=30"`
}
type CreateUserSuccess struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"message"`
}

type Service interface {
	CreateUsers(context.Context, CreateUserRequest) (uuid.UUID, error)
	LoginUsers(context.Context, LoginRequest) error
	LogoutUsers(context.Context) error
	GetUser(context.Context) (Users, error)
	DeleteUsers(context.Context) error
	UpdateUsers(context.Context, UpdateRequest) error
	RequestPasswordReset(context.Context, RequestUpdatePassword) error
	ResetPassword(context.Context, string, UpdatePasswordRequest) error

	// ActiveEmail(context.Context, ActiveEmail) error
}
