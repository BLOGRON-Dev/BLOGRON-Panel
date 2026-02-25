package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"blogron/util"

	"github.com/go-chi/chi/v5"
)

// ── System Stats ─────────────────────────────────────────────────────────────

type SystemStats struct {
	CPU     CPUStat  `json:"cpu"`
	RAM     RAMStat  `json:"ram"`
	Disk    DiskStat `json:"disk"`
	Uptime  string   `json:"uptime"`
	LoadAvg string   `json:"load_avg"`
	OS      string   `json:"os"`
}

type CPUStat struct {
	UsedPct float64 `json:"used_pct"`
	Cores   int     `json:"cores"`
}

type RAMStat struct {
	TotalMB int64   `json:"total_mb"`
	UsedMB  int64   `json:"used_mb"`
	FreeMB  int64   `json:"free_mb"`
	UsedPct float64 `json:"used_pct"`
}

type DiskStat struct {
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	FreePct float64 `json:"free_pct"`
	UsedPct float64 `json:"used_pct"`
}

// GetSystemStats godoc
// GET /api/system/stats
func GetSystemStats(w http.ResponseWriter, r *http.Request) {
	stats := SystemStats{
		OS: readOSRelease(),
	}

	stats.CPU = getCPUStat()
	stats.RAM = getRAMStat()
	stats.Disk = getDiskStat()
	stats.Uptime = getUptime()
	stats.LoadAvg = getLoadAvg()

	util.WriteJSON(w, http.StatusOK, stats)
}

func getCPUStat() CPUStat {
	// Read /proc/stat twice with a short interval for accurate usage
	s1 := readCPULine()
	time.Sleep(200 * time.Millisecond)
	s2 := readCPULine()

	idle1 := s1[3]
	idle2 := s2[3]
	total1 := sum(s1)
	total2 := sum(s2)

	totalDiff := float64(total2 - total1)
	idleDiff := float64(idle2 - idle1)

	usedPct := 0.0
	if totalDiff > 0 {
		usedPct = (1 - idleDiff/totalDiff) * 100
	}

	cores := 1
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "processor") {
				cores++
			}
		}
	}

	return CPUStat{UsedPct: usedPct, Cores: cores}
}

func readCPULine() []int64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return make([]int64, 10)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)[1:] // skip "cpu" label
			vals := make([]int64, len(fields))
			for i, f := range fields {
				vals[i], _ = strconv.ParseInt(f, 10, 64)
			}
			return vals
		}
	}
	return make([]int64, 10)
}

func sum(vals []int64) int64 {
	var s int64
	for _, v := range vals {
		s += v
	}
	return s
}

func getRAMStat() RAMStat {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return RAMStat{}
	}
	m := map[string]int64{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			val, _ := strconv.ParseInt(parts[1], 10, 64)
			m[key] = val
		}
	}
	totalKB := m["MemTotal"]
	freeKB := m["MemAvailable"]
	usedKB := totalKB - freeKB
	usedPct := 0.0
	if totalKB > 0 {
		usedPct = float64(usedKB) / float64(totalKB) * 100
	}
	return RAMStat{
		TotalMB: totalKB / 1024,
		UsedMB:  usedKB / 1024,
		FreeMB:  freeKB / 1024,
		UsedPct: usedPct,
	}
}

func getDiskStat() DiskStat {
	out, err := util.RunCmd("df", "-BG", "--output=size,used,avail,pcent", "/")
	if err != nil {
		return DiskStat{}
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return DiskStat{}
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return DiskStat{}
	}
	parse := func(s string) float64 {
		s = strings.TrimRight(s, "G%")
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	total := parse(fields[0])
	used := parse(fields[1])
	pct := parse(fields[3])
	return DiskStat{
		TotalGB: total,
		UsedGB:  used,
		FreePct: 100 - pct,
		UsedPct: pct,
	}
}

func getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "unknown"
	}
	secs, _ := strconv.ParseFloat(parts[0], 64)
	d := int(secs / 86400)
	h := int(secs/3600) % 24
	m := int(secs/60) % 60
	return fmt.Sprintf("%dd %dh %dm", d, h, m)
}

func getLoadAvg() string {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return "unknown"
	}
	return strings.Join(parts[:3], " ")
}

func readOSRelease() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		}
	}
	return "Linux"
}

// ── Services ─────────────────────────────────────────────────────────────────

type Service struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Active  bool   `json:"active"`
	PID     string `json:"pid"`
	Uptime  string `json:"uptime"`
}

var monitoredServices = []string{"nginx", "mariadb", "ssh", "postfix", "dovecot", "named", "vsftpd", "fail2ban", "cron"}

func GetServices(w http.ResponseWriter, r *http.Request) {
	services := make([]Service, 0, len(monitoredServices))
	for _, name := range monitoredServices {
		svc := queryService(name)
		services = append(services, svc)
	}
	util.WriteJSON(w, http.StatusOK, services)
}

func queryService(name string) Service {
	out, err := util.RunCmd("systemctl", "show", name, "--property=ActiveState,MainPID,ActiveEnterTimestamp")
	svc := Service{Name: name}
	if err != nil {
		svc.Status = "unknown"
		return svc
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "ActiveState":
			svc.Status = parts[1]
			svc.Active = parts[1] == "active"
		case "MainPID":
			svc.PID = parts[1]
		case "ActiveEnterTimestamp":
			svc.Uptime = parts[1]
		}
	}
	return svc
}

func RestartService(w http.ResponseWriter, r *http.Request) {
	serviceAction(w, r, "restart")
}

func StopService(w http.ResponseWriter, r *http.Request) {
	serviceAction(w, r, "stop")
}

func StartService(w http.ResponseWriter, r *http.Request) {
	serviceAction(w, r, "start")
}

func serviceAction(w http.ResponseWriter, r *http.Request, action string) {
	name := chi_urlParam(r, "name")
	name = util.Sanitize(name)
	if name == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid service name")
		return
	}

	allowed := map[string]bool{}
	for _, s := range monitoredServices {
		allowed[s] = true
	}
	if !allowed[name] {
		util.WriteError(w, http.StatusForbidden, "service not managed by this panel")
		return
	}

	if _, err := util.RunCmd("systemctl", action, name); err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": name, "action": action})
}

// ── Logs ──────────────────────────────────────────────────────────────────────

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Unit    string `json:"unit"`
}

func GetLogs(w http.ResponseWriter, r *http.Request) {
	unit := r.URL.Query().Get("unit")
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "100"
	}

	args := []string{"-n", lines, "--no-pager", "--output=short-iso"}
	if unit != "" {
		unit = util.Sanitize(unit)
		args = append(args, "-u", unit)
	}

	out, err := util.RunCmd("journalctl", args...)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	entries := parseJournalLines(out)
	util.WriteJSON(w, http.StatusOK, entries)
}

func parseJournalLines(raw string) []LogEntry {
	var entries []LogEntry
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		entry := LogEntry{Message: line}
		// Rough parse: "2024-06-01T06:42:11+0000 hostname unit[pid]: message"
		parts := strings.SplitN(line, " ", 4)
		if len(parts) >= 4 {
			entry.Time = parts[0]
			entry.Message = parts[3]
			if strings.Contains(strings.ToLower(parts[3]), "error") || strings.Contains(strings.ToLower(parts[3]), "fail") {
				entry.Level = "ERROR"
			} else if strings.Contains(strings.ToLower(parts[3]), "warn") {
				entry.Level = "WARN"
			} else {
				entry.Level = "INFO"
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// chi_urlParam extracts a named URL parameter using the chi router context.
func chi_urlParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
