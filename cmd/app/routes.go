package main

import (
	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/htmx"
)

func isHTMXRequest(c *xun.Context) bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

func setupRoutes(app *xun.App) {
	// Serve htmx.js
	app.HandleFunc("GET /htmx.js", htmx.HandleFunc())

	// Auth middleware
	authMiddleware := func(next xun.HandleFunc) xun.HandleFunc {
		return func(c *xun.Context) error {
			session, _ := c.Get("session").(*Session)
			if session == nil || session.UserID == 0 {
				if isHTMXRequest(c) {
					c.WriteHeader(htmx.HxRedirect, "/login")
					c.WriteStatus(200)
					return nil
				}
				c.Redirect("/login")
				return nil
			}
			c.Set("user_id", session.UserID)
			return next(c)
		}
	}

	// Public pages
	app.Get("/", handleLanding)
	app.Get("/login", handleLoginPage)
	app.Post("/login", handleLogin)
	app.Get("/register", handleRegisterPage)
	app.Post("/register", handleRegister)
	app.Post("/logout", handleLogout)

	// Protected pages (dashboard)
	dashboard := app.Group("/dashboard")
	dashboard.Use(authMiddleware)
	dashboard.Get("/", handleDashboard)
	dashboard.Get("/users", handleUserList)
	dashboard.Post("/users", handleUserCreate)
	dashboard.Put("/users/:id", handleUserUpdate)
	dashboard.Delete("/users/:id", handleUserDelete)

	// HTMX partial views
	dashboard.Get("/views/users/list", handleUserListView)
	dashboard.Get("/views/users/row/:id", handleUserRowView)
}
