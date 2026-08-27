package users

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"
	"uuid"

	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
	oopszap "github.com/samber/oops/loggers/zap"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

const (
	memory      = 64 * 1024 // 64 MB
	iterations  = 3
	parallelism = 2
	saltLength  = 16
	keyLength   = 32
)

type ServiceImpl struct {
	l       *zap.Logger
	db      *gorm.DB
	session *scs.SessionManager
	mailer  shared.MailerService
	red     *redis.Client
	webUrl  string
}

func (u *ServiceImpl) ResetPassword(ctx context.Context, resetToken string, body UpdatePasswordRequest) error {
	tokenKey := fmt.Sprintf("reset_token::%s", resetToken)
	userIDStr, err := u.red.GetDel(ctx, tokenKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fuego.BadRequestError{
				Title:  "Token Inválido",
				Detail: "O link de recuperação expirou ou já foi utilizado",
			}
		}

		u.l.Error("Failed to retrieve reset token from redis",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	parsedUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fuego.BadRequestError{
			Title:  "Dados Inválidos",
			Detail: "Identificador de usuário corrompido no token",
		}
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		u.l.Error("Failed to generate salt",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	hash := argon2.IDKey([]byte(body.Password), salt, iterations, memory, parallelism, keyLength)
	if err := u.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", parsedUserID).
		Updates(map[string]any{
			"password": hash,
			"salt":     salt,
		}).Error; err != nil {
		u.l.Error("Failed to update user password",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return fuego.NotFoundError{
				Title:  "Usuário não encontrado",
				Detail: "Não foi possível encontrar a conta informada",
			}
		}
		
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return nil
}

func (u *ServiceImpl) CreateUsers(ctx context.Context, body CreateUserRequest) (uuid.UUID, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		u.l.Error("Failed to generate salt",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return uuid.Nil(), fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}
	hash := argon2.IDKey([]byte(body.Password), salt, iterations, memory, parallelism, keyLength)

	user := User{
		Name:     body.Name,
		Email:    body.Email,
		Password: hash,
		Salt:     salt,
	}
	if err := u.db.WithContext(ctx).Create(&user).Error; err != nil {
		u.l.Error("Failed to create user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return uuid.Nil(), fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return user.Base.ID, nil
}
func (u *ServiceImpl) LoginUsers(ctx context.Context, body LoginRequest) error {
	var user User
	if err := u.db.WithContext(ctx).Where("email = ?", body.Email).First(&user).Error; err != nil {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fuego.NotFoundError{
				Err:    err,
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: "Credenciais invalidas",
			}
		}
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	ok, err := verifyPassword(body.Password, user.Password, user.Salt)
	if err != nil {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	if !ok {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	err = u.session.RenewToken(ctx)
	if err != nil {
		u.l.Error("failed to renew token", zap.Error(err))
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	u.session.Put(ctx, "user_id", user.ID)

	return nil
}
func (u *ServiceImpl) LogoutUsers(ctx context.Context) error {
	err := u.session.RenewToken(ctx)
	if err != nil {
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	u.session.Remove(ctx, "user_id")

	return nil
}
func (u *ServiceImpl) GetUser(ctx context.Context) (User, error) {
	userID, ok := u.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		u.l.Error("failed to get user id", zap.Error(errors.New("failed to get user id")))
		return User{}, fuego.UnauthorizedError{
			Err:    errors.New("user not found"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa esta autenticado",
		}
	}

	var user User
	if err := u.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, fuego.NotFoundError{
				Err:    err,
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: "Credenciais invalidas",
			}
		}
		return User{}, fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return User{
		ID: user.ID,

		Name:  user.Name,
		Email: user.Email,

		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil

}
func (u *ServiceImpl) DeleteUsers(ctx context.Context) error {
	userID, ok := u.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		u.l.Error("failed to get user id", zap.Error(errors.New("failed to get user id")))
		return fuego.UnauthorizedError{
			Err:    errors.New("user not found"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa esta autenticado",
		}
	}

	if err := u.db.WithContext(ctx).Where("id = ?", userID).Delete(&User{}).Error; err != nil {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fuego.NotFoundError{
				Err:    err,
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: "Credenciais invalidas",
			}
		}
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return nil
}
func (u *ServiceImpl) UpdateUsers(ctx context.Context, body UpdateRequest) error {
	userID, ok := u.session.Get(ctx, "user_id").(uuid.UUID)
	if !ok {
		u.l.Error("failed to get user id", zap.Error(errors.New("failed to get user id")))
		return fuego.UnauthorizedError{
			Err:    errors.New("user not found"),
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Ops! você precisa esta autenticado",
		}
	}

	if err := u.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(body).Error; err != nil {
		u.l.Error("Failed to get user",
			zap.Object("error", oopszap.OopsMarshalFunc(err)),
			zap.String("stacktrace", oopszap.OopsStackMarshaller(err)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fuego.NotFoundError{
				Err:    err,
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: "Credenciais invalidas",
			}
		}
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	return nil
}
func (u *ServiceImpl) RequestPasswordReset(ctx context.Context, body RequestUpdatePassword) error {
	var user User

	if err := u.db.WithContext(ctx).Where("email = ?", body.Email).First(&user).Error; err != nil {
		u.l.Error("failed to get user id", zap.Error(errors.New("failed to get user id")))

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	resetToken := uuid.NewV4()
	expireTime := 5 * time.Minute

	tokenKey := fmt.Sprintf("reset_token::%s", resetToken.String())
	if err := u.red.Set(ctx, tokenKey, user.ID.String(), expireTime).Err(); err != nil {
		u.l.Error("failed to generate user reset token", zap.Error(err), zap.String("stacktrace", oopszap.OopsStackMarshaller(err)))
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Ops! algo deu errado, tente novamente mais tarde",
		}
	}

	resetURL := fmt.Sprintf("%s/api/reset_password?token=%s", u.webUrl, resetToken.String())

	payload := shared.ResetPasswordData{
		To:           user.Email,
		Name:         user.Name,
		RedirectLink: resetURL,
		ExpiresIn:    "5 minutos",
	}

	if err := u.mailer.SendResetPasswordEmail(payload); err != nil {
		u.l.Error("failed to send email", zap.Error(err), zap.String("stacktrace", oopszap.OopsStackMarshaller(err)))
		_ = u.red.Del(ctx, tokenKey).Err()
		return fuego.InternalServerError{
			Err:    err,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Falha ao enfileirar o e-mail de recuperação",
		}
	}
	return nil

}

var _ Service = &ServiceImpl{}

func NewUsersService(db *gorm.DB, l *zap.Logger, session *scs.SessionManager, mail shared.MailerService, red *redis.Client, webUrl string) Service {
	return &ServiceImpl{
		l,
		db,
		session,
		mail,
		red,
		webUrl,
	}
}

func verifyPassword(plainTextPassword string, storedHash []byte, storedSalt []byte) (bool, error) {
	hashToCompare := argon2.IDKey(
		[]byte(plainTextPassword),
		storedSalt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	if subtle.ConstantTimeCompare(storedHash, hashToCompare) == 1 {
		return true, nil
	}

	return false, errors.New("senha incorreta")
}
