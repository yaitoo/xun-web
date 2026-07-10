package main

import (
	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/htmx"
)

func setupRoutes(app *xun.App) {
	// Serve htmx.js
	app.HandleFunc("GET /htmx.js", htmx.HandleFunc())

	// Auth middleware
	//
	// Note: we do NOT stash anything on TempData here. sessionMiddleware
	// (registered globally via app.Use) has already verified the cookie
	// and put the *Session on TempData["session"] by the time this runs,
	// so handlers and templates can read it directly as {{.session}}.
	authMiddleware := func(next xun.HandleFunc) xun.HandleFunc {
		return func(c *xun.Context) error {
			session, _ := c.Get("session").(*Session)
			if session == nil || session.UserID == 0 {
				if htmx.IsHxRequest(c) {
					htmx.WriteRedirect(c, "/login")
					return nil
				}
				c.Redirect("/login")
				return nil
			}
			return next(c)
		}
	}

	// Public pages.
	//
	// The "GET /{$}" patterns below are NOT a quirk — they are how xun's
	// PageRoute system auto-registers files named `pages/<name>/index.html`.
	// `splitFile` appends `{$}` to any pattern ending in `/`, so the
	// route table has `GET /{$}` and `GET /dashboard/{$}` (for
	// `pages/dashboard.html` → wait, that's `pages/dashboard.html` →
	// not a `index.html` file — so this is a regular route). The
	// `/{$}` form matches exactly "/" with no path suffix, which is
	// what Go 1.22 ServeMux requires for the root index page.
	app.Get("/{$}", handleLanding)
	app.Get("/login", handleLoginPage)
	app.Post("/login", handleLogin)
	app.Get("/register", handleRegisterPage)
	app.Post("/register", handleRegister)
	app.Post("/logout", handleLogout)

	// Protected pages (dashboard).
	//
	// The dashboard auto-registered `GET /dashboard/{$}` for
	// `pages/dashboard.html`? — no: that file is named `dashboard.html`
	// directly so its pattern is `GET /dashboard`. The dashboard
	// group `app.Group("/dashboard")` adds the prefix; `dashboard.Get`
	// registers under that prefix. The order in setupRoutes is:
	//   1. createApp() loads templates and auto-registers every page
	//      file as a route with a nil-data handler.
	//   2. setupRoutes() runs and registers / overrides handlers via
	//      app.Get / dashboard.Get. If a route already exists for the
	//      pattern, xun overwrites the handler (but keeps the viewers).
	dashboard := app.Group("/dashboard")
	// sessionMiddleware must be re-applied per group — xun's
	// app.Use middlewares do NOT propagate to sub-groups. The
	// group's Next() only chains the group's own Use()s.
	dashboard.Use(sessionMiddleware, authMiddleware)
	// pages/dashboard/index.html auto-registers GET /dashboard/{$},
	// not GET /dashboard — splitFile appends {$} to any pattern
	// ending in /, and index.html stripping yields the trailing slash.
	// We override with the same {$} pattern so the file's
	// default nil-data handler is replaced with our handler that
	// supplies `title` on TempData.
	dashboard.Get("/{$}", handleDashboard)
	dashboard.Get("/users", handleUserList)
	dashboard.Post("/users", handleUserCreate)
	dashboard.Put("/users/{id}", handleUserUpdate)
	dashboard.Delete("/users/{id}", handleUserDelete)

	// HTMX partial views
	dashboard.Get("/views/users/list", handleUserListView)
	dashboard.Get("/views/users/row/{id}", handleUserRowView)

	// PageRoute demo: a single-user detail page. The template lives
	// at `app/pages/dashboard/users/{id}.html` and was auto-registered
	// by the HtmlViewEngine when the app started (with a default
	// handler that passes nil data). This `dashboard.Get` call
	// overrides that handler with one that loads the User and passes
	// it as `.Data` so the template can render `{{.Data.Name}}`,
	// `{{.Data.Email}}`, etc.
	dashboard.Get("/users/{id}", handleUserDetail)

	// WebSocket demo page. The page is served through xun (so it
	// inherits the standard sessionMiddleware / layout pipeline); the
	// upgrade endpoint itself lives in setupNativeRoutes below.
	dashboard.Get("/ws", handleWSDemoPage)
}

// setupNativeRoutes registers handlers that need the raw
// *http.ResponseWriter that xun's high-level API hides — most commonly
// protocols that hijack the connection (WebSocket, SSE, HTTP/2 streams)
// or third-party http.Handler integrations (prometheus, pprof, etc.).
//
// Routes registered here go directly onto the underlying
// *http.ServeMux via app.Mux(). They:
//
//   - bypass ALL xun middleware (app.Use AND group.Use)
//   - are not listed in app.Start()'s startup log
//   - own their own status, headers, body, and logging
//   - must re-implement anything xun would normally do for them
//     (session validation, request IDs, error-to-status mapping)
//
// See cmd/app/ws.go for the WebSocket implementation, which performs
// its own session-cookie check (wsAuthOK) to compensate for the
// bypassed sessionMiddleware.
func setupNativeRoutes(app *xun.App, hub *wsHub) {
	// WebSocket chat endpoint. The pattern uses Go 1.22 ServeMux
	// syntax; the same route is NOT registered via app.Get/HandleFunc
	// above because xun's wrapper would prevent the Hijack() call.
	app.Mux().HandleFunc("GET /api/ws", wsHandler(hub))
}
