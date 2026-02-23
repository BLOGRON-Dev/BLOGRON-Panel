package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"blogron/util"
)

const (
	vsftpdUserListFile = "/etc/vsftpd.userlist"
	vsftpdPasswdFile   = "/etc/vsftpd.passwd"
)

type FTPUser struct {
	Username string `json:"username"`
	HomeDir  string `json:"home_dir"`
	Active   bool   `json:"active"`
}

// ListFTPUsers godoc
// GET /api/ftp
func ListFTPUsers(w http.ResponseWriter, r *http.Request) {
	users := readFTPUsers()
	util.WriteJSON(w, http.StatusOK, users)
}

// CreateFTPUser godoc
// POST /api/ftp
func CreateFTPUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		HomeDir  string `json:"home_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	username := util.Sanitize(body.Username)
	if username == "" || len(body.Password) < 8 {
		util.WriteError(w, http.StatusBadRequest, "invalid username or password too short")
		return
	}

	homeDir := body.HomeDir
	if homeDir == "" {
		homeDir = fmt.Sprintf("/var/www/%s", username)
	}
	homeDir, _ = safePath(strings.TrimPrefix(homeDir, "/var/www"))

	// Create system user with no login shell for FTP-only access
	util.RunCmd("useradd", "-m", "-d", homeDir, "-s", "/usr/sbin/nologin", username)
	util.RunCmd("chpasswd", username+":"+body.Password)
	util.RunCmd("chown", username+":"+username, homeDir)

	// Add to vsftpd user list
	appendLine(vsftpdUserListFile, username)

	// Restart vsftpd
	util.RunCmd("systemctl", "restart", "vsftpd")

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status":   "created",
		"username": username,
		"home_dir": homeDir,
	})
}

// DeleteFTPUser godoc
// DELETE /api/ftp/{username}
func DeleteFTPUser(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	if username == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid username")
		return
	}

	removeLine(vsftpdUserListFile, username)
	util.RunCmd("userdel", username) // don't use -r to preserve files
	util.RunCmd("systemctl", "restart", "vsftpd")

	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateFTPPassword godoc
// PUT /api/ftp/{username}
func UpdateFTPPassword(w http.ResponseWriter, r *http.Request) {
	username := util.Sanitize(chi_urlParam(r, "username"))
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body.Password) < 8 {
		util.WriteError(w, http.StatusBadRequest, "password too short")
		return
	}
	util.RunCmd("chpasswd", username+":"+body.Password)
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func readFTPUsers() []FTPUser {
	var users []FTPUser
	f, err := os.Open(vsftpdUserListFile)
	if err != nil {
		return users
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		username := strings.TrimSpace(scanner.Text())
		if username == "" || strings.HasPrefix(username, "#") {
			continue
		}
		homeDir := "/var/www/" + username
		if info, err := os.Stat("/home/" + username); err == nil && info.IsDir() {
			homeDir = "/home/" + username
		}
		users = append(users, FTPUser{
			Username: username,
			HomeDir:  homeDir,
			Active:   true,
		})
	}
	return users
}
