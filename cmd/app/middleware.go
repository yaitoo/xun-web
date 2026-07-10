package main

import (
	"net/http"
	"time"

	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/cookie"
)

const sessionCookieName = "xun_web_session"

// writeSessionCookie encodes the session into a signed cookie and
// attaches it to the response. Use this from handlers BEFORE redirecting
// or otherwise committing the response — xun's response writer may not
// flush late header changes reliably.
func writeSessionCookie(c *xun.Context, s *Session) {
	encoded := encodeSession(s)
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

// clearSessionCookie removes the signed session cookie. Safe to call
// from handlers before any redirect.
func clearSessionCookie(c *xun.Context) {
	cookie.Delete(c, http.Cookie{
		Name: sessionCookieName,
		Path: "/",
	})
}

func sessionMiddleware(next xun.HandleFunc) xun.HandleFunc {
	return func(c *xun.Context) error {
		// Get session from signed cookie
		value, _, err := cookie.GetSigned(c, sessionCookieName, []byte(sessionSecret))
		if err == nil {
			if session, deerr := decodeSession(value); deerr == nil && session != nil && session.UserID != 0 {
				c.Set("session", session)
			}
		}

		// Execute handler
		return next(c)
	}
}
