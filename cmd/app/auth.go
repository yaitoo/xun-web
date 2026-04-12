package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
