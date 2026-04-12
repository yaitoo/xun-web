package main

import (
	"net/http"
	"time"

	"github.com/yaitoo/xun"
)

const sessionCookieName = "xun_web_session"

func sessionMiddleware(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		// Inject db into context
		c.Set("db", db)

		// Get session from cookie
		if cookie, err := c.Request.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			// In production, you'd look up the session in a session store
			// For this demo, we just pass the cookie value through
			// The actual session data would be stored server-side
			session := &Session{
				ID: cookie.Value,
			}
			c.Set("session", session)
		}

		// Execute handler
		err := next(c)

		// If session was set, update cookie
		if session, ok := c.Get("session").(*Session); ok && session != nil && session.ID != "" {
			http.SetCookie(c.Response, &http.Cookie{
				Name:     sessionCookieName,
				Value:    session.ID,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Set to true in production with HTTPS
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(time.Hour * 24 * 7 / time.Second), // 7 days
			})
		}

		return err
	}
}
