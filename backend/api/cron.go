package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"blogron/util"
)

const cronDir = "/var/spool/cron/crontabs"

type CronJob struct {
	ID       int    `json:"id"`
	Minute   string `json:"minute"`
	Hour     string `json:"hour"`
	Day      string `json:"day"`
	Month    string `json:"month"`
	Weekday  string `json:"weekday"`
	Command  string `json:"command"`
	User     string `json:"user"`
	Schedule string `json:"schedule"` // human-readable summary
	Enabled  bool   `json:"enabled"`
}

type createCronRequest struct {
	Minute  string `json:"minute"`
	Hour    string `json:"hour"`
	Day     string `json:"day"`
	Month   string `json:"month"`
	Weekday string `json:"weekday"`
	Command string `json:"command"`
	User    string `json:"user"`
}

// ListCronJobs godoc
// GET /api/cron?user=john
func ListCronJobs(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	var jobs []CronJob
	if user != "" {
		user = util.Sanitize(user)
		userJobs := readCrontab(user)
		jobs = append(jobs, userJobs...)
	} else {
		// Read all crontabs
		entries, err := os.ReadDir(cronDir)
		if err == nil {
			for _, e := range entries {
				userJobs := readCrontab(e.Name())
				jobs = append(jobs, userJobs...)
			}
		}
		// Also read /etc/crontab for system jobs
		systemJobs := readSystemCrontab()
		jobs = append(jobs, systemJobs...)
	}

	util.WriteJSON(w, http.StatusOK, jobs)
}

// CreateCronJob godoc
// POST /api/cron
func CreateCronJob(w http.ResponseWriter, r *http.Request) {
	var req createCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if req.Command == "" {
		util.WriteError(w, http.StatusBadRequest, "command is required")
		return
	}

	// Validate cron fields (very basic — allow *, numbers, ranges, lists)
	for _, field := range []string{req.Minute, req.Hour, req.Day, req.Month, req.Weekday} {
		if !isValidCronField(field) {
			util.WriteError(w, http.StatusBadRequest, "invalid cron field: "+field)
			return
		}
	}

	user := req.User
	if user == "" {
		user = "root"
	}
	user = util.Sanitize(user)

	cronLine := fmt.Sprintf("%s %s %s %s %s %s",
		req.Minute, req.Hour, req.Day, req.Month, req.Weekday, req.Command)

	if err := appendCrontab(user, cronLine); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to write crontab: "+err.Error())
		return
	}

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status":  "created",
		"user":    user,
		"cron":    cronLine,
	})
}

// DeleteCronJob godoc
// DELETE /api/cron/{id}?user=john
func DeleteCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi_urlParam(r, "id")
	user := util.Sanitize(r.URL.Query().Get("user"))
	if user == "" {
		user = "root"
	}

	id := 0
	for _, c := range idStr {
		id = id*10 + int(c-'0')
	}

	if err := removeCronLine(user, id); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateCronJob godoc
// PUT /api/cron/{id}
func UpdateCronJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi_urlParam(r, "id")
	id := 0
	for _, c := range idStr {
		id = id*10 + int(c-'0')
	}

	var req createCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	user := req.User
	if user == "" {
		user = "root"
	}
	user = util.Sanitize(user)

	newLine := fmt.Sprintf("%s %s %s %s %s %s",
		req.Minute, req.Hour, req.Day, req.Month, req.Weekday, req.Command)

	if err := updateCronLine(user, id, newLine); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// RunCronNow godoc - triggers a cron command immediately
// POST /api/cron/{id}/run
func RunCronNow(w http.ResponseWriter, r *http.Request) {
	idStr := chi_urlParam(r, "id")
	user := util.Sanitize(r.URL.Query().Get("user"))
	if user == "" {
		user = "root"
	}

	id := 0
	for _, c := range idStr {
		id = id*10 + int(c-'0')
	}

	jobs := readCrontab(user)
	for _, job := range jobs {
		if job.ID == id {
			// Execute the job command
			go func(cmd string) {
				util.RunCmd("bash", "-c", cmd)
			}(job.Command)
			util.WriteJSON(w, http.StatusOK, map[string]string{"status": "triggered", "command": job.Command})
			return
		}
	}
	util.WriteError(w, http.StatusNotFound, "cron job not found")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func readCrontab(user string) []CronJob {
	crontabFile := fmt.Sprintf("%s/%s", cronDir, user)
	data, err := os.ReadFile(crontabFile)
	if err != nil {
		return nil
	}
	return parseCronLines(string(data), user)
}

func readSystemCrontab() []CronJob {
	data, err := os.ReadFile("/etc/crontab")
	if err != nil {
		return nil
	}
	return parseCronLines(string(data), "system")
}

func parseCronLines(content, user string) []CronJob {
	var jobs []CronJob
	id := 1
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "MAILTO") ||
			strings.HasPrefix(line, "PATH") || strings.HasPrefix(line, "SHELL") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		enabled := true
		offset := 0
		// system crontab has an extra "user" field
		cmd := strings.Join(parts[5+offset:], " ")

		jobs = append(jobs, CronJob{
			ID:       id,
			Minute:   parts[0],
			Hour:     parts[1],
			Day:      parts[2],
			Month:    parts[3],
			Weekday:  parts[4],
			Command:  cmd,
			User:     user,
			Schedule: humanReadableCron(parts[0], parts[1], parts[2], parts[3], parts[4]),
			Enabled:  enabled,
		})
		id++
	}
	return jobs
}

func appendCrontab(user, line string) error {
	crontabFile := fmt.Sprintf("%s/%s", cronDir, user)
	f, err := os.OpenFile(crontabFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

func removeCronLine(user string, id int) error {
	crontabFile := fmt.Sprintf("%s/%s", cronDir, user)
	data, err := os.ReadFile(crontabFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	jobIdx := 0
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isJob := trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
			!strings.HasPrefix(trimmed, "MAILTO") && !strings.HasPrefix(trimmed, "PATH")
		if isJob {
			jobIdx++
			if jobIdx == id {
				continue // skip this line (delete it)
			}
		}
		kept = append(kept, line)
	}
	return os.WriteFile(crontabFile, []byte(strings.Join(kept, "\n")), 0600)
}

func updateCronLine(user string, id int, newLine string) error {
	crontabFile := fmt.Sprintf("%s/%s", cronDir, user)
	data, err := os.ReadFile(crontabFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	jobIdx := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isJob := trimmed != "" && !strings.HasPrefix(trimmed, "#")
		if isJob {
			jobIdx++
			if jobIdx == id {
				lines[i] = newLine
				break
			}
		}
	}
	return os.WriteFile(crontabFile, []byte(strings.Join(lines, "\n")), 0600)
}

func isValidCronField(field string) bool {
	if field == "" {
		return false
	}
	// Allow: * */n n n-m n,m
	for _, c := range field {
		if c != '*' && c != '/' && c != '-' && c != ',' && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func humanReadableCron(min, hour, day, month, weekday string) string {
	if min == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
		return "Every minute"
	}
	if min == "0" && hour == "*" {
		return "Every hour"
	}
	if min == "0" && hour == "0" && day == "*" && month == "*" && weekday == "*" {
		return "Daily at midnight"
	}
	if min == "0" && hour == "0" && day == "*" && month == "*" && weekday == "0" {
		return "Weekly on Sunday"
	}
	if min == "0" && hour == "0" && day == "1" {
		return "Monthly on the 1st"
	}
	return fmt.Sprintf("%s %s %s %s %s", min, hour, day, month, weekday)
}
