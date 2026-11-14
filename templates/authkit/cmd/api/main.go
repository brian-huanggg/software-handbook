// Command api serves the authkit demo: session and token authentication
// side by side under /session/* and /token/*.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brian-huanggg/software-handbook/templates/authkit/middleware"
	"github.com/brian-huanggg/software-handbook/templates/authkit/password"
	"github.com/brian-huanggg/software-handbook/templates/authkit/session"
	"github.com/brian-huanggg/software-handbook/templates/authkit/store"
	"github.com/brian-huanggg/software-handbook/templates/authkit/token"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(healthcheckURL(os.Getenv("ADDR"))); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	users := store.NewUserStore(pool)
	access, err := token.NewAccess(token.AccessOptions{
		Secret:          cfg.JWTSecret,
		PreviousSecrets: cfg.JWTPreviousSecrets,
		Issuer:          cfg.Issuer,
	})
	if err != nil {
		return err
	}
	sessions := session.NewManager(session.NewPostgresStore(pool), session.Options{
		TTL:             sessionTTL,
		InsecureCookies: cfg.InsecureCookies,
		Roles:           rolesFromStore(users),
	})
	refresh := token.NewRefreshManager(token.NewPostgresRefreshStore(pool), token.RefreshOptions{TTL: refreshTTL})
	handler, err := newHandler(deps{
		Users: users,
		// Each Argon2id call holds ~19 MiB; cap how many run at once.
		Hasher:           password.NewLimited(password.NewArgon2id(cfg.Pepper, cfg.PreviousPeppers...), runtime.NumCPU()),
		Sessions:         sessions,
		Access:           access,
		Refresh:          refresh,
		RefreshCookieTTL: refreshTTL,
		InsecureCookies:  cfg.InsecureCookies,
		AllowedOrigins:   cfg.AllowedOrigins,
		AuthIPLimit:      middleware.NewLimiter(authIPEvery, cfg.AuthIPBurst),
		LoginEmailLimit:  middleware.NewLimiter(loginEmailEvery, cfg.LoginEmailBurst),
	})
	if err != nil {
		return err
	}
	go runJanitor(ctx, janitorInterval,
		purger{"sessions", sessions.Purge},
		purger{"refresh_tokens", refresh.Purge},
	)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", cfg.Addr, "insecure_cookies", cfg.InsecureCookies)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
