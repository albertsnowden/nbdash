// Command dashboard serves the NetBird management UI.
//
// It replaces the Next.js dashboard's "static export behind nginx" model
// with a single Go binary: a React SPA embedded and served alongside a JSON
// API (internal/api) that holds the OIDC session server-side and proxies
// the Management API on the SPA's behalf — the SPA itself never sees a
// bearer token. The SPA's own build output is embedded too, so the binary
// has no runtime file dependencies.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/albertsnowden/nbdash/internal/api"
	"github.com/albertsnowden/nbdash/internal/auth"
	"github.com/albertsnowden/nbdash/internal/config"
	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// version is stamped at build time:
//
//	go build -ldflags "-X main.version=v0.1.0"
//
// It is shown in the sidebar and logged at startup so a deployed binary can be
// identified without file hashes.
var version = "dev"

func main() {
	// Accepted as a bare subcommand (`nbdash version`, the form most CLIs a
	// systemd-managed binary's operator would reach for first) as well as
	// the two flag spellings — checked directly against os.Args rather than
	// through package flag, since flag.Parse() treats a positional "version"
	// (no leading dash) as an ordinary argument, not this flag, and silently
	// falls through to run(), which then fails on missing env config instead
	// of just printing the version.
	if len(os.Args) > 1 && slices.Contains([]string{"version", "-version", "--version"}, os.Args[1]) {
		fmt.Println(version)
		return
	}

	// systemd captures stdout into the journal and adds its own timestamps, so
	// the handler omits them to avoid a doubled time column in journalctl.
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	authenticator, err := discoverWithRetry(ctx, cfg, auth.NewStore(), log)
	if err != nil {
		return err
	}

	server := api.New(cfg, authenticator, nbapi.New(cfg.APIOrigin), log)
	routes, err := server.Routes()
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           routes,
		ReadHeaderTimeout: 10 * time.Second,
		// Bounds the time to read the full request, not just headers — without
		// it a client that trickles in a body (deliberately or on a bad
		// connection) pins a connection and its goroutine indefinitely, the
		// same slowloris shape ReadHeaderTimeout guards against for headers.
		// Every request body here is a small JSON API call; MaxBytesReader in
		// requireSameSiteOrigin bounds its size, this bounds how long that read
		// may take.
		ReadTimeout: 15 * time.Second,
		// Without WriteTimeout a client that stops reading the response body
		// pins a connection and its goroutine indefinitely. The largest single
		// response here is the SPA's own bundled JS/CSS (well under a
		// megabyte), so 30s is still generous.
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("dashboard listening",
			"version", version,
			"addr", cfg.ListenAddr,
			"api", cfg.APIOrigin,
			"issuer", cfg.AuthAuthority,
			"token_source", cfg.TokenSource,
			// The one line that makes a client-mode misconfiguration visible in
			// the journal before it becomes a user's failed login: which mode
			// actually got selected, not just what was intended.
			"oidc_client_mode", authenticator.Mode(),
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

// discoverWithRetry performs OIDC discovery, retrying briefly so the dashboard
// can start alongside its identity provider rather than crash-looping while the
// provider is still coming up.
func discoverWithRetry(ctx context.Context, cfg *config.Config, store *auth.Store, log *slog.Logger) (*auth.Authenticator, error) {
	const attempts = 5

	var lastErr error
	for i := range attempts {
		authenticator, err := auth.New(ctx, cfg, store, log)
		if err == nil {
			return authenticator, nil
		}
		// A provider that doesn't support PKCE will not start supporting it
		// between attempts — this is a static misconfiguration, not the
		// transient "provider still coming up" case this loop exists for, so
		// retrying it would only delay reporting the real problem by up to the
		// ~30s this loop's backoff totals.
		if errors.Is(err, auth.ErrPublicClientPKCEUnsupported) {
			return nil, err
		}
		lastErr = err

		delay := time.Duration(1<<i) * time.Second
		log.Warn("oidc discovery failed, retrying", "error", err, "in", delay)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, lastErr
}
