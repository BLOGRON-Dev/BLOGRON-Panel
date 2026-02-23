package api

import (
	"encoding/json"
	"net/http"
	"time"

	"blogron/util"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// In production, load admin credentials from a secure config file or environment.
// Never hardcode credentials in source code.
var adminPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
	User    string `json:"user"`
}

// Login godoc
// POST /api/auth/login
// Body: { "username": "root", "password": "..." }
func Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Only allow the admin user for now.
	// Extend this to check a database of panel users.
	adminUser := "admin"
	if req.Username != adminUser {
		util.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword(adminPasswordHash, []byte(req.Password)); err != nil {
		util.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	expiry := time.Now().Add(8 * time.Hour)
	claims := jwt.MapClaims{
		"sub":  req.Username,
		"role": "admin",
		"exp":  expiry.Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(util.JWTSecret())
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "could not sign token")
		return
	}

	util.WriteJSON(w, http.StatusOK, loginResponse{
		Token:   signed,
		Expires: expiry.Unix(),
		User:    req.Username,
	})
}
