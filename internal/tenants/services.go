package tenants

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/members"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/go-fuego/fuego"
	oopszap "github.com/samber/oops/loggers/zap"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceImpl struct {
	l       *zap.Logger
	db      *gorm.DB
	session *scs.SessionManager
}

func (o *ServiceImpl) CreateOrganization(ctx context.Context, body CreateOrganization) (uuid.UUID, error) {
	userID, ok := o.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		o.l.Error("failed to get user id from session")
		return uuid.Nil(), fuego.UnauthorizedError{
			Err:    errors.New("user not authenticated"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa estar autenticado",
		}
	}

	org := Organizations{
		Name:    body.Name,
		Slug:    shared.GenerateSlug(body.Name),
		OwnerId: userID,
	}
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		ownerMember := members.Member{
			OrganizationID: org.ID,
			UserID:         userID,
			Role:           members.Owner,
		}
		if err := tx.Create(&ownerMember).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		o.l.Error("error creating organization transaction", zap.Error(err))

		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return uuid.Nil(), fuego.ConflictError{
				Err:    err,
				Title:  "Conflict",
				Status: http.StatusConflict,
				Detail: "Ops! alguém já está usando esse nome/slug de organização",
			}
		}

		return uuid.Nil(), fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return org.ID, nil
}
func (o *ServiceImpl) GetOrganization(ctx context.Context, orgId uuid.UUID) (Organizations, error) {
	userID, ok := o.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		o.l.Error("failed to get user id from session")
		return Organizations{}, fuego.UnauthorizedError{
			Err:    errors.New("user not authenticated"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa estar autenticado",
		}
	}

	var org Organizations

	err := o.db.WithContext(ctx).
		Scopes(shared.ForTenant(orgId)).
		Preload("Members").
		Joins("JOIN members ON members.organization_id = organizations.id").
		Where("organizations.id = ? AND members.user_id = ?", orgId, userID).
		First(&org).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Organizations{}, fuego.NotFoundError{
				Err:    err,
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: "Organização não encontrada ou acesso não autorizado",
			}
		}

		o.l.Error("Failed to get Organization",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return Organizations{}, fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return org, nil
}

func (o *ServiceImpl) DeleteOrganization(ctx context.Context, orgId uuid.UUID) error {
	userID, ok := o.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		o.l.Error("failed to get user id from session")
		return fuego.UnauthorizedError{
			Err:    errors.New("user not authenticated"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa estar autenticado",
		}
	}

	var count int64
	if err := o.db.WithContext(ctx).
		Table("members").
		Where("organization_id = ? AND user_id = ?", orgId, userID).
		Count(&count).Error; err != nil {
		o.l.Error("failed to verify organization access", zap.Error(err))
		return fuego.UnauthorizedError{
			Err:    err,
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Organização não encontrada ou acesso não autorizado",
		}
	}

	if count == 0 {
		return fuego.NotFoundError{
			Err:    errors.New("organization not found"),
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: "Organização não encontrada ou acesso não autorizado",
		}
	}

	result := o.db.WithContext(ctx).
		Scopes(shared.ForTenant(orgId)).
		Where("id = ?", orgId).
		Delete(&Organizations{})

	if err := result.Error; err != nil {
		o.l.Error("failed to delete organization", zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		return fuego.NotFoundError{
			Err:    errors.New("organization not found"),
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: "Organização não encontrada",
		}
	}

	return nil
}

func (o *ServiceImpl) UpdateOrganization(ctx context.Context, body UpdateRequest, orgId uuid.UUID) error {
	userID, ok := o.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		o.l.Error("failed to get user id from session")
		return fuego.UnauthorizedError{
			Err:    errors.New("user not authenticated"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa estar autenticado",
		}
	}

	var org Organizations
	err := o.db.WithContext(ctx).
		Table("organizations").
		Joins("INNER JOIN members ON members.organization_id = organizations.id").
		Where("organizations.id = ? AND members.user_id = ? AND members.role = ?", orgId, userID, members.Owner).
		First(&org).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fuego.UnauthorizedError{
				Err:    err,
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "Organização não encontrada ou acesso não autorizado",
			}
		}
		o.l.Error("failed to verify organization access", zap.Error(err))
		return err
	}

	if body.Name != "" {
		org.Name = body.Name
		org.Slug = shared.GenerateSlug(body.Name)
	}
	if err := o.db.WithContext(ctx).Save(&org).Error; err != nil {
		o.l.Error("failed to update organization", zap.Error(err))
		return err
	}

	return nil
}

var _ Service = &ServiceImpl{}

func NewOrganizationService(db *gorm.DB, l *zap.Logger, session *scs.SessionManager) Service {
	return &ServiceImpl{
		l,
		db,
		session,
	}
}
