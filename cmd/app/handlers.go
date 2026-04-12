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

func handleLanding(c *xun.Context) error {
	return c.View(map[string]any{
		"title": "Xun Web Starter",
	}, "pages/index")
}

// --- Auth Pages ---

func handleLoginPage(c *xun.Context) error {
	return c.View(map[string]any{
		"title": "Login",
	}, "pages/login")
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

	// Create session
	session := &Session{
		ID:     generateSessionID(),
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}
	c.Set("session", session)

	if isHTMX(c) {
		c.WriteHeader(htmx.HxRedirect, "/dashboard")
		c.WriteStatus(http.StatusOK)
		return nil
	}

	c.Redirect("/dashboard")
	return nil
}

func handleRegisterPage(c *xun.Context) error {
	return c.View(map[string]any{
		"title": "Register",
	}, "pages/register")
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

	// Auto-login
	session := &Session{
		ID:     generateSessionID(),
		UserID: userID,
		Email:  email,
		Name:   name,
	}
	c.Set("session", session)

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
	if isHTMX(c) {
		c.WriteHeader(htmx.HxRedirect, "/")
		c.WriteStatus(http.StatusOK)
		return nil
	}
	c.Redirect("/")
	return nil
}

// --- Dashboard ---

func handleDashboard(c *xun.Context) error {
	session := c.Get("session").(*Session)
	return c.View(map[string]any{
		"title": "Dashboard",
		"name":  session.Name,
	}, "pages/dashboard")
}

// --- User CRUD ---

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

	return c.View(map[string]any{
		"title": "User Management",
		"users": users,
	}, "pages/users")
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

	return c.View(map[string]any{
		"users": users,
	}, "views/users/list")
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

	return c.View(map[string]any{
		"user": u,
	}, "views/users/row")
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

// --- Helpers ---

func renderHXRetarget(c *xun.Context, target, message string) error {
	c.WriteHeader(htmx.HxRetarget, "#"+target)
	return c.View(map[string]any{"message": message}, "views/error-message")
}
