package main

import (
	"net/http"
	"time"

	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/cookie"
)

const sessionCookieName = "xun_web_session"

func sessionMiddleware(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		// Get session from signed cookie
		value, _, err := cookie.GetSigned(c, sessionCookieName, []byte(sessionSecret))
		if err == nil {
			session, deerr := decodeSession(value)
			if deerr == nil && session != nil && session.UserID != 0 {
				c.Set("session", session)
			}
		}

		// Execute handler
		err = next(c)

		// If session was set, update signed cookie
		if session, ok := c.Get("session").(*Session); ok && session != nil && session.UserID != 0 {
			encoded := encodeSession(session)
			cookie.SetSigned(c, http.Cookie{
				Name:     sessionCookieName,
				Value:    encoded,
				Path:     "/",
				HttpOnly: true,
				Secure:   false, // Set to true in production with HTTPS
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(time.Hour * 24 * 7 / time.Second), // 7 days
			}, []byte(sessionSecret))
		}

		return err
	}
}
