package api

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"blogron/util"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// adminCreds holds the lazily-initialised credentials loaded from environment.
// ADMIN_USER and ADMIN_PASSWORD are injected by install.sh into the systemd unit.
var (
	adminCreds struct {
		username string
		hash     []byte
	}
	adminCredsOnce sync.Once
)

func loadAdminCreds() {
	adminCredsOnce.Do(func() {
		adminCreds.username = os.Getenv("ADMIN_USER")
		if adminCreds.username == "" {
			adminCreds.username = "admin"
		}
		pass := os.Getenv("ADMIN_PASSWORD")
		if pass == "" {
			pass = "changeme"
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		adminCreds.hash = hash
	})
}

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
// Body: { "username": "admin", "password": "..." }
func Login(w http.ResponseWriter, r *http.Request) {
	loadAdminCreds()

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username != adminCreds.username {
		util.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword(adminCreds.hash, []byte(req.Password)); err != nil {
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
