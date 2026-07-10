package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/htmx"
)

// Session represents a user session
type Session struct {
	ID     string
	UserID int64
	Email  string
	Name   string
}

// User represents a user record
type User struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

// isHTMX checks if the request is an HTMX request
func isHTMX(c *xun.Context) bool {
	return c.Request.Header.Get("HX-Request") == "true"
}

// --- Landing Page ---

// handleLanding renders the public landing page. The page has no
// business model (it's just marketing chrome), so `.Data` is nil; the
// `title` auxiliary value goes on TempData so the layout can read it.
func handleLanding(c *xun.Context) error {
	c.Set("title", "Xun Web Starter")
	return c.View(nil, "index")
}

// --- Auth Pages ---

func handleLoginPage(c *xun.Context) error {
	c.Set("title", "Login")
	return c.View(nil, "login")
}

func handleLogin(c *xun.Context) error {
	email := c.Request.FormValue("email")
	password := c.Request.FormValue("password")

	if email == "" || password == "" {
		return renderHXRetarget(c, "error-message", "Email and password are required")
	}

	ctx := context.Background()

	var user User
	err := db.QueryRowContext(ctx, "SELECT id, email, name, password_hash FROM users WHERE email = ?", email).
		Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return renderHXRetarget(c, "error-message", "Invalid email or password")
		}
		return err
	}

	// Simple password check (in production, use bcrypt)
	if !checkPassword(password, user.PasswordHash) {
		return renderHXRetarget(c, "error-message", "Invalid email or password")
	}

	// Create session and persist it via signed cookie BEFORE redirecting.
	session := &Session{
		ID:     generateSessionID(),
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}
	c.Set("session", session)
	writeSessionCookie(c, session)

	if isHTMX(c) {
		c.WriteHeader(htmx.HxRedirect, "/dashboard")
		c.WriteStatus(http.StatusOK)
		return nil
	}

	c.Redirect("/dashboard")
	return nil
}

func handleRegisterPage(c *xun.Context) error {
	c.Set("title", "Register")
	return c.View(nil, "register")
}

func handleRegister(c *xun.Context) error {
	name := c.Request.FormValue("name")
	email := c.Request.FormValue("email")
	password := c.Request.FormValue("password")

	if name == "" || email == "" || password == "" {
		return renderHXRetarget(c, "error-message", "All fields are required")
	}

	if len(password) < 6 {
		return renderHXRetarget(c, "error-message", "Password must be at least 6 characters")
	}

	ctx := context.Background()

	// Check if email exists
	var exists bool
	db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if exists {
		return renderHXRetarget(c, "error-message", "Email already registered")
	}

	// Hash password
	hash := hashPassword(password)

	// Insert user
	result, err := db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		name, email, hash, time.Now())
	if err != nil {
		return err
	}

	userID, _ := result.LastInsertId()

	// Auto-login: persist session in signed cookie before redirect/htmx-write.
	session := &Session{
		ID:     generateSessionID(),
		UserID: userID,
		Email:  email,
		Name:   name,
	}
	c.Set("session", session)
	writeSessionCookie(c, session)

	if isHTMX(c) {
		c.WriteHeader(htmx.HxRedirect, "/dashboard")
		c.WriteStatus(http.StatusOK)
		return nil
	}

	c.Redirect("/dashboard")
	return nil
}

func handleLogout(c *xun.Context) error {
	c.Set("session", nil)
	clearSessionCookie(c)
	if isHTMX(c) {
		c.WriteHeader(htmx.HxRedirect, "/")
		c.WriteStatus(http.StatusOK)
		return nil
	}
	c.Redirect("/")
	return nil
}

// --- Dashboard ---

// handleDashboard renders the dashboard welcome page. There is no
// per-page business data — the current user comes from sessionMiddleware
// (TempData.session), and the page title is plain chrome (TempData.title).
//
// The page lives at `pages/dashboard/index.html`, whose PageRoute
// auto-registered the `HtmlViewer` on this very route — so `c.View(nil)`
// works without a name argument (xun picks the route's viewers by
// Accept header). Same goes for every other PageRoute handler below.
func handleDashboard(c *xun.Context) error {
	c.Set("title", "Dashboard")
	return c.View(nil)
}

// handleWSDemoPage renders a tiny single-page chat UI that connects to
// the WebSocket endpoint at /api/ws. The page itself is served through
// the normal xun pipeline (so sessionMiddleware runs); only the WS
// upgrade endpoint bypasses it.
func handleWSDemoPage(c *xun.Context) error {
	c.Set("title", "WebSocket Demo")
	return c.View(nil)
}

// --- User CRUD ---

// handleUserList renders the full Users management page. The `users`
// slice is the page's main business data → `.Data`. The page title is
// auxiliary chrome → TempData.title.
func handleUserList(c *xun.Context) error {
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return err
		}
		users = append(users, u)
	}

	c.Set("title", "User Management")
	return c.View(users)
}

func handleUserListView(c *xun.Context) error {
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return err
		}
		users = append(users, u)
	}

	return c.View(users, "views/users/list")
}

func handleUserRowView(c *xun.Context) error {
	id, _ := strconv.ParseInt(c.Request.PathValue("id"), 10, 64)

	ctx := context.Background()

	var u User
	err := db.QueryRowContext(ctx, "SELECT id, name, email, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.WriteStatus(http.StatusNotFound)
			return nil
		}
		return err
	}

	return c.View(u)
}

func handleUserCreate(c *xun.Context) error {
	name := c.Request.FormValue("name")
	email := c.Request.FormValue("email")
	password := c.Request.FormValue("password")

	if name == "" || email == "" || password == "" {
		return renderHXRetarget(c, "users-list", "All fields are required")
	}

	ctx := context.Background()

	hash := hashPassword(password)
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)",
		name, email, hash, time.Now())
	if err != nil {
		return err
	}

	// Return updated list via htmx
	return handleUserListView(c)
}

func handleUserUpdate(c *xun.Context) error {
	id, _ := strconv.ParseInt(c.Request.PathValue("id"), 10, 64)
	name := c.Request.FormValue("name")
	email := c.Request.FormValue("email")

	if name == "" || email == "" {
		return renderHXRetarget(c, "user-row-"+c.Request.PathValue("id"), "Name and email are required")
	}

	ctx := context.Background()

	_, err := db.ExecContext(ctx, "UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, id)
	if err != nil {
		return err
	}

	// Return updated row via htmx
	return handleUserRowView(c)
}

func handleUserDelete(c *xun.Context) error {
	id, _ := strconv.ParseInt(c.Request.PathValue("id"), 10, 64)

	ctx := context.Background()

	_, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}

	// HTMX will remove the row via hx-swap="delete"
	c.WriteStatus(http.StatusOK)
	return nil
}

// handleUserDetail renders the per-user detail page bound to
// `pages/dashboard/users/{id}.html`. This is the PageRoute demo:
//   - The page file's name pattern `{id}` matches Go 1.22 ServeMux
//     syntax, so xun auto-registers `GET /dashboard/users/{id}` and
//     serves the template.
//   - `routes.go → setupRoutes` calls `dashboard.Get("/users/{id}",
//     handleUserDetail)` AFTER the auto-registration, overwriting the
//     default nil-data handler with one that loads the User and
//     passes it as `.Data` for `{{.Data.Name}}` / `{{.Data.Email}}`.
//
// PageRoute is the xun idiom for "render a server-side page bound to
// a URL with optional path parameters". It's the simplest form of SSR
// in xun: zero JS, one HTML file, one Go handler that returns a
// struct.
func handleUserDetail(c *xun.Context) error {
	id, err := strconv.ParseInt(c.Request.PathValue("id"), 10, 64)
	if err != nil {
		c.WriteStatus(http.StatusBadRequest)
		return nil
	}

	ctx := context.Background()

	var u User
	err = db.QueryRowContext(ctx, "SELECT id, name, email, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.WriteStatus(http.StatusNotFound)
			return nil
		}
		return err
	}

	c.Set("title", "User #"+strconv.FormatInt(u.ID, 10))
	return c.View(u)
}

// --- Helpers ---

func renderHXRetarget(c *xun.Context, target, message string) error {
	c.WriteHeader(htmx.HxRetarget, "#"+target)
	c.Set("message", message)
	return c.View(nil, "views/error-message")
}
