package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User represents a registered account in SocialFlow.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Credentials carries the login or registration payload.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// AuthClaims are the custom JWT claims embedded in the http-only cookie.
// ActiveWorkspaceID is mutable (changed on workspace switch).
type AuthClaims struct {
	jwt.RegisteredClaims
	UserID            string `json:"uid"`
	Email             string `json:"eml"`
	ActiveWorkspaceID string `json:"wid,omitempty"`
	Role              string `json:"rol,omitempty"`
}

// HashPassword hashes a plaintext password using bcrypt at default cost.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
