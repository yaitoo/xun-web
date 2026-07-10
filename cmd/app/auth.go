package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// hashPassword creates a SHA256 hash of the password
// Note: In production, use bcrypt or argon2
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// checkPassword verifies a password against a hash
func checkPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

// generateSessionID creates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// encodeSession encodes session data into a base64 string for cookie storage
func encodeSession(s *Session) string {
	data, _ := json.Marshal(s)
	return base64.URLEncoding.EncodeToString(data)
}

// decodeSession decodes a base64 session string back into a Session struct
func decodeSession(encoded string) (*Session, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Initials returns the first letter of first and last name, uppercased
func (u *User) Initials() string {
	parts := strings.Fields(u.Name)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		return strings.ToUpper(string(parts[0][0]))
	}
	return strings.ToUpper(string(parts[0][0]) + string(parts[len(parts)-1][0]))
}
