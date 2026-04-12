package main

import (
	"net/http"
	"time"

	"github.com/yaitoo/xun"
)

const sessionCookieName = "xun_web_session"

func sessionMiddleware(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		// Get session from cookie
		if cookie, err := c.Request.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			session, err := decodeSession(cookie.Value)
			if err == nil && session != nil && session.UserID != 0 {
				c.Set("session", session)
			}
		}

		// Execute handler
		err := next(c)

		// If session was set, update cookie
		if session, ok := c.Get("session").(*Session); ok && session != nil && session.UserID != 0 {
			http.SetCookie(c.Response, &http.Cookie{
				Name:     sessionCookieName,
				Value:    encodeSession(session),
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
