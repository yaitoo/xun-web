package main

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/cache"
	"github.com/yaitoo/xun/ext/htmx"
)

//go:embed app/components
//go:embed app/layouts
//go:embed app/pages
//go:embed app/public
//go:embed app/views
var fsys embed.FS

var sessionSecret = os.Getenv("SESSION_SECRET")

// Version and Commit are injected at build time via -ldflags -X (see the
// Makefile). They default to "dev" / "unknown" for `go run`.
var (
	Version = "dev"
	Commit  = "unknown"
)

// shutdownTimeout is the maximum duration the server will wait for
// in-flight requests to finish during a graceful shutdown.
const shutdownTimeout = 15 * time.Second

func main() {

	var err error
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	err = loadConfig()
	if err != nil {
		log.Println("failed to load config", err)
		os.Exit(1)
	}

	db, err = setupSQLite(ctx)
	if err != nil {
		log.Println("failed to setup sqlite", err)
		os.Exit(1)
	}
	defer db.Close() //nolint:errcheck

	mux := http.NewServeMux()
	app := createApp(mux)

	if sessionSecret == "" {
		sessionSecret = "change-me-in-production"
	}

	// Setup routes
	setupRoutes(app)

	// Reference the build-time identity vars so the linker keeps them;
	// without this read, dead-code elimination would strip them and the
	// -X ldflags injection in the Makefile would be a no-op.
	slog.Info("xun-web",
		"version", Version,
		"commit", Commit,
	)

	app.Start()

	if err := runServers(mux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// runServers starts HTTP and/or HTTPS listeners based on configuration and
// blocks until one of them returns. When the first listener exits, it
// initiates a graceful shutdown of the other (if any) and waits for the
// shutdown timeout before returning.
//
// All listeners share the same handler — the *http.ServeMux that was
// injected into the xun.App via xun.WithMux.
func runServers(handler http.Handler) error {
	httpAddr := strings.TrimSpace(viper.GetString("addr.http"))
	httpsAddr := strings.TrimSpace(viper.GetString("addr.https"))
	tlsEnabled := viper.GetBool("tls.enabled")
	certFile := viper.GetString("tls.cert_file")
	keyFile := viper.GetString("tls.key_file")

	startHTTP := httpAddr != ""
	startHTTPS := httpsAddr != "" && tlsEnabled

	if !startHTTP && !startHTTPS {
		return errors.New("no listener configured: set addr.http and/or tls.enabled with addr.https")
	}

	servers := make(map[string]*http.Server)

	if startHTTP {
		servers["http"] = &http.Server{
			Addr:              httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
		slog.Info("starting http server", "addr", httpAddr)
	}

	if startHTTPS {
		if certFile == "" || keyFile == "" {
			return errors.New("tls.enabled is true but tls.cert_file or tls.key_file is empty")
		}
		servers["https"] = &http.Server{
			Addr:              httpsAddr,
			Handler:           handler,
			TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
			ReadHeaderTimeout: 10 * time.Second,
		}
		slog.Info("starting https server", "addr", httpsAddr, "cert", certFile, "key", keyFile)
	}

	// Two channels so we can distinguish fatal bind errors (which must
	// surface a non-zero exit code immediately) from graceful shutdowns
	// (which must drain in-flight requests and exit 0).
	//
	//   bindErrCh: receives the FIRST listener that fails to bind. The
	//              caller treats any value sent here as fatal.
	//   doneCh:    receives nil from each listener goroutine after it
	//              exits, so the shutdown path can wait for all
	//              goroutines to actually return.
	bindErrCh := make(chan error, len(servers))
	doneCh := make(chan struct{}, len(servers))

	for name, srv := range servers {
		go func(name string, srv *http.Server) {
			var err error
			if name == "https" {
				err = srv.ListenAndServeTLS(certFile, keyFile)
			} else {
				err = srv.ListenAndServe()
			}
			// http.ErrServerClosed is the expected result of
			// srv.Shutdown — it's not an error.
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			if err != nil {
				// Bind failure (e.g. "address already in use") or any
				// other listener error. Surface it as fatal.
				bindErrCh <- fmt.Errorf("%s: %w", name, err)
			}
			doneCh <- struct{}{}
		}(name, srv)
	}

	// Wait for SIGINT/SIGTERM or a fatal bind error.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig.String())
	case err := <-bindErrCh:
		// Bind failure (or any other listener error). Log it, then
		// bail out immediately with a non-zero exit code. We do NOT
		// call Shutdown on the other listeners — they may have
		// bound successfully and are serving real traffic, and a
		// bind failure on one shouldn't be hidden behind a clean
		// exit. systemd / docker / k8s will see the non-zero exit
		// and restart the process.
		slog.Error("listener failed to start, exiting", "err", err)
		signal.Stop(sigCh)
		os.Exit(1)
	}

	// Stop receiving further signals and start graceful shutdown.
	signal.Stop(sigCh)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	for name, srv := range servers {
		slog.Info("shutting down server", "name", name, "addr", srv.Addr)
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "name", name, "err", err)
		}
	}

	// Drain doneCh so all listener goroutines exit before we return.
	// We don't care about any late bindErrCh values here: by the time
	// we got here, the only way into shutdown was via a signal, so a
	// subsequent listener error after Shutdown is benign.
	for i := 0; i < len(servers); i++ {
		<-doneCh
	}

	return nil
}

// getFsys returns the fs.FS to use for templates and assets.
// In dev, it returns the local "./app" directory; otherwise it uses the embedded FS under root.
func getFsys() fs.FS {

	fi, err := os.Stat("./app")
	if err == nil && fi.IsDir() {
		return os.DirFS("./app")
	}

	app, _ := fs.Sub(fsys, "app")

	if app == nil {
		return fsys
	}
	return app
}

func createApp(mux *http.ServeMux) *xun.App {

	app := xun.New(
		xun.WithMux(mux),
		xun.WithFsys(getFsys()),
		xun.WithWatch(),
		xun.WithHandlerViewers(&xun.JsonViewer{}),
		xun.WithInterceptor(htmx.New()),
		xun.WithBuildAssetURL(func(path string) bool {
			return strings.HasPrefix(path, "/assets/")
		}),
	)

	app.Use(sessionMiddleware,
		cache.New(
			cache.Match("/assets/", "", 7*24*time.Hour),
			cache.Match("", "favicon.ico", 365*24*time.Hour),
		))

	return app
}

func loadConfig() error {
	viper.SetDefault("addr.http", ":80")
	viper.SetDefault("addr.https", ":443")
	viper.SetDefault("tls.enabled", false)
	viper.SetDefault("tls.cert_file", "./certs/server.crt")
	viper.SetDefault("tls.key_file", "./certs/server.key")

	var conf string
	flag.StringVar(&conf, "conf", "app.yml", "config file")
	flag.Parse()

	// Search paths, in priority order:
	//   1. -conf flag value (if it contains a path separator, e.g. "conf/app.yml")
	//   2. The directory of the running binary (production: ./bin/app looks for ./bin/app.yml)
	//   3. The source-relative cmd/app/ directory (development: `go run ./cmd/app` from repo root)
	//   4. The current working directory (fallback: `go run ./cmd/app` from inside cmd/app/)
	if strings.Contains(conf, "/") || strings.Contains(conf, string(os.PathSeparator)) {
		viper.SetConfigFile(conf)
	} else {
		viper.SetConfigName(conf)
		viper.SetConfigType("yml")

		if exe, err := os.Executable(); err == nil {
			viper.AddConfigPath(filepath.Dir(exe))
		}
		// Source-relative: this file is cmd/app/main.go, so the config
		// ships next to it. Only meaningful in dev (the directory won't
		// exist at runtime when the binary is in ./bin/), so viper will
		// just skip it silently if absent.
		viper.AddConfigPath("cmd/app")
		viper.AddConfigPath(".")
	}

	// Environment-variable override layer. The Makefile sources `.env`
	// before launching the app, so secrets and per-developer overrides
	// land here. Mapping rule: APP_<SECTION>_<KEY> → <section>.<key>,
	// so APP_SESSION_SECRET overrides session.secret, APP_DB_DSN
	// overrides db.dsn, etc. Viper's default precedence (env > config
	// file > default) means .env wins over the YAML file but the YAML
	// file still provides the non-overridden base values.
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Load config
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("fail to read config", err)
	}

	return err
}
