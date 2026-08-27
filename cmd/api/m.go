package main

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uuid"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/config"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/duddy57/toaki-server/internal/users"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-fuego/fuego"
	"github.com/joho/godotenv"
	"github.com/phenpessoa/gutils/netutils/httputils"
	"go.uber.org/zap"
)

func main() {
	gob.Register(uuid.UUID{})
	if err := godotenv.Load(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to load env: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	if err := run(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "something went wrong: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "all systems offline, exiting...")
}

func run(ctx context.Context) error {
	cfg, err := config.LoadEnv()
	if err != nil {
		return err
	}

	logger, err := config.NewLogger(cfg.GoEnv)
	if err != nil {
		return err
	}

	logger = logger.Named("toaki_server")
	defer logger.Sync()

	// Passe apenas o Addr ("localhost:6633") e o DB (0)
	rec := config.NewRedisClient(
		cfg.Redis.Addr, // "localhost:6633"
		cfg.Redis.DB,   // 0 (int)
		ctx,
	)
	defer func() { _ = rec.Close() }()

	sessions := scs.New()
	sessions.Store = goredisstore.New(rec)
	sessions.Lifetime = 24 * time.Hour
	sessions.Cookie.HttpOnly = false
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Name = "sperium_session"
	sessions.Cookie.Secure = false

	// csrfMiddleware := csrf.Protect(
	// 	[]byte(os.Getenv("GOBID_CSRF_KEY")),
	// 	csrf.Secure(false), // DEV ONLY
	// )

	c := cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CorsOrigin},
		AllowCredentials: cfg.GoEnv == "production",
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Cookie"},
	})

	db, err := config.InitDB(cfg.GoEnv, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&users.User{})
	if err != nil {
		return err
	}

	mailerService, err := shared.NewMailer(&shared.MailerConfig{
		Host:     cfg.Mailer.Hostname,
		Port:     cfg.Mailer.Port,
		Username: cfg.Mailer.Username,
		Password: cfg.Mailer.Password,
		From:     cfg.Mailer.From,
		Workers:  3,
		BufSize:  50,
	})
	if err != nil {
		return err
	}
	defer mailerService.Close()

	sv := fuego.NewServer(
		fuego.WithoutLogger(),
		fuego.WithGlobalMiddlewares(
			c,
			middleware.RequestID,
			middleware.Recoverer,
			middleware.Logger,
			sessions.LoadAndSave,
			httputils.ChiLogger(logger),
			// csrfMiddleware
		),
	)

	config.MountHTTPHandler(sv, rec, logger, db, sessions, mailerService, cfg)

	defer func() {
		const timeout = 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := sv.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown server", zap.Error(err))
		}
	}()
	errChan := make(chan error, 1)
	go func() {
		fmt.Println("server starting on:", sv.Addr)
		if err := sv.Run(); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}
