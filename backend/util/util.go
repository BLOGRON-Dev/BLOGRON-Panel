package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// ── JSON helpers ────────────────────────────────────────────────────────────

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// ── JWT secret ──────────────────────────────────────────────────────────────

func JWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Fallback for development — set a real secret in production!
		secret = "change-me-in-production-use-env-var"
	}
	return []byte(secret)
}

// ── Safe command runner ──────────────────────────────────────────────────────

// RunCmd executes a whitelisted system command via sudo and returns combined output.
// Only commands that appear in the allowlist are executed.
func RunCmd(name string, args ...string) (string, error) {
	if err := validateCommand(name, args); err != nil {
		return "", err
	}

	// Prefix with sudo for privileged operations
	cmdArgs := append([]string{name}, args...)
	cmd := exec.Command("sudo", cmdArgs...)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %s", strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// allowedCommands restricts what RunCmd can execute.
// This is a critical security boundary — never remove this check.
var allowedCommands = map[string]bool{
	"useradd":        true,
	"userdel":        true,
	"usermod":        true,
	"passwd":         true,
	"chpasswd":       true,
	"nginx":          true,
	"systemctl":      true,
	"certbot":        true,
	"mysql":          true,
	"mysqladmin":     true,
	"mkdir":          true,
	"rm":             true,
	"mv":             true,
	"ls":             true,
	"cat":            true,
	"find":           true,
	"df":             true,
	"free":           true,
	"uptime":         true,
	"journalctl":     true,
	"ln":             true,
	"chmod":          true,
	"chown":          true,
}

func validateCommand(name string, args []string) error {
	if !allowedCommands[name] {
		return fmt.Errorf("command %q is not allowed", name)
	}
	// Basic argument sanitization — reject shell metacharacters
	for _, arg := range args {
		for _, ch := range []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\n", "\r"} {
			if strings.Contains(arg, ch) {
				return fmt.Errorf("argument contains disallowed character: %q", arg)
			}
		}
	}
	return nil
}

// Sanitize strips anything that isn't alphanumeric, dot, dash, or underscore.
// Use for usernames, domain names, database names, etc.
func Sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
