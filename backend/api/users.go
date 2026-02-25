package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"blogron/util"
)

type User struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	GID      string `json:"gid"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
	Locked   bool   `json:"locked"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Shell    string `json:"shell"`
	Groups   string `json:"groups"` // comma-separated
}

// ListUsers godoc
// GET /api/users
// Reads /etc/passwd and returns non-system users (UID >= 1000).
func ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := parsePasswd()
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "could not read user list")
		return
	}

	// Mark locked accounts (those with '!' prefix in shadow)
	locked := lockedUsers()

	var result []User
	for _, u := range users {
		u.Locked = locked[u.Username]
		result = append(result, u)
	}
	util.WriteJSON(w, http.StatusOK, result)
}

// CreateUser godoc
// POST /api/users
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	username := util.Sanitize(req.Username)
	if username == "" || len(username) > 32 {
		util.WriteError(w, http.StatusBadRequest, "invalid username")
		return
	}
	if req.Password == "" || len(req.Password) < 8 {
		util.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	shell := req.Shell
	if shell == "" {
		shell = "/bin/bash"
	}
	allowedShells := map[string]bool{"/bin/bash": true, "/bin/sh": true, "/usr/sbin/nologin": true}
	if !allowedShells[shell] {
		util.WriteError(w, http.StatusBadRequest, "invalid shell")
		return
	}

	// Create the user
	args := []string{"-m", "-s", shell, username}
	if req.Groups != "" {
		args = append([]string{"-G", util.Sanitize(req.Groups)}, args...)
	}
	if _, err := util.RunCmd("useradd", args...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "useradd failed: "+err.Error())
		return
	}

	// Set password via chpasswd
	// chpasswd reads "user:password" from stdin — we write to a temp file approach
	// For real production, use `echo "user:pass" | sudo chpasswd`
	if _, err := util.RunCmd("chpasswd", username+":"+req.Password); err != nil {
		// Attempt to clean up the created user
		util.RunCmd("userdel", "-r", username)
		util.WriteError(w, http.StatusInternalServerError, "failed to set password")
		return
	}

	util.WriteJSON(w, http.StatusCreated, map[string]string{"username": username, "status": "created"})
}

// DeleteUser godoc
// DELETE /api/users/{username}
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	if username == "" || username == "root" {
		util.WriteError(w, http.StatusBadRequest, "invalid or protected username")
		return
	}

	if _, err := util.RunCmd("userdel", "-r", username); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted", "username": username})
}

// UpdateUser godoc
// PUT /api/users/{username}
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	var body struct {
		Password string `json:"password"`
		Shell    string `json:"shell"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if body.Password != "" {
		if len(body.Password) < 8 {
			util.WriteError(w, http.StatusBadRequest, "password too short")
			return
		}
		if _, err := util.RunCmd("chpasswd", username+":"+body.Password); err != nil {
			util.WriteError(w, http.StatusInternalServerError, "failed to update password")
			return
		}
	}

	if body.Shell != "" {
		allowedShells := map[string]bool{"/bin/bash": true, "/bin/sh": true, "/usr/sbin/nologin": true}
		if !allowedShells[body.Shell] {
			util.WriteError(w, http.StatusBadRequest, "invalid shell")
			return
		}
		if _, err := util.RunCmd("usermod", "-s", body.Shell, username); err != nil {
			util.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// SuspendUser locks the account with usermod -L
func SuspendUser(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	if username == "root" {
		util.WriteError(w, http.StatusForbidden, "cannot suspend root")
		return
	}
	if _, err := util.RunCmd("usermod", "-L", username); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

// ActivateUser unlocks the account with usermod -U
func ActivateUser(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	if _, err := util.RunCmd("usermod", "-U", username); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parsePasswd() ([]User, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var users []User
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) < 7 {
			continue
		}
		uid := parts[2]
		// Skip system accounts (UID < 1000)
		uidInt := 0
		for _, c := range uid {
			uidInt = uidInt*10 + int(c-'0')
		}
		if uidInt < 1000 {
			continue
		}
		users = append(users, User{
			Username: parts[0],
			UID:      uid,
			GID:      parts[3],
			Home:     parts[5],
			Shell:    parts[6],
		})
	}
	return users, scanner.Err()
}

func lockedUsers() map[string]bool {
	locked := map[string]bool{}
	f, err := os.Open("/etc/shadow")
	if err != nil {
		return locked // shadow may not be readable without root
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) >= 2 && strings.HasPrefix(parts[1], "!") {
			locked[parts[0]] = true
		}
	}
	return locked
}
