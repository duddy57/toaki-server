package config

import (
	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/duddy57/toaki-server/internal/tenants"
	"github.com/duddy57/toaki-server/internal/users"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func MountHTTPHandler(sv *fuego.Server, rec *redis.Client, logger *zap.Logger, db *gorm.DB, session *scs.SessionManager, mail shared.MailerService, env *Config) {
	user := users.UsersResources{
		UsersService: users.NewUsersService(
			db,
			logger,
			session,
			mail,
			rec,
			env.CorsOrigin,
		),
		Session: session,
	}

	org := tenants.OrganizationResources{
		OrganizationService: tenants.NewOrganizationService(
			db,
			logger,
			session,
		),
		Session: session,
	}

	user.Routes(sv)
	org.Routes(sv)
}
