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
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"authkit/password"
	"authkit/session"
	"authkit/store"
	"authkit/token"
)

func main() {
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
	access, err := token.NewAccess(token.AccessOptions{Secret: cfg.JWTSecret, Issuer: cfg.Issuer})
	if err != nil {
		return err
	}
	handler, err := newHandler(deps{
		Users:  users,
		Hasher: password.NewArgon2id(cfg.Pepper),
		Sessions: session.NewManager(session.NewPostgresStore(pool), session.Options{
			TTL:             sessionTTL,
			InsecureCookies: cfg.InsecureCookies,
			Roles:           rolesFromStore(users),
		}),
		Access:           access,
		Refresh:          token.NewRefreshManager(token.NewPostgresRefreshStore(pool), token.RefreshOptions{TTL: refreshTTL}),
		RefreshCookieTTL: refreshTTL,
		InsecureCookies:  cfg.InsecureCookies,
	})
	if err != nil {
		return err
	}

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
