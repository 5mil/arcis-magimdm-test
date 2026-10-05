package desk

import (
	_ "embed"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed ui/index.html
var indexHTML []byte

type Device struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Token    string `json:"token"`
	Policy   string `json:"policy"`
	LastSeen string `json:"last_seen"`
}

type Policy struct {
	Name            string `json:"name"`
	Mode            string `json:"mode"`
	MiningEnabled   bool   `json:"mining_enabled"`
	ArcisForStudent bool   `json:"arcis_for_student"`
}

type Desk struct {
	mu       sync.Mutex
	root     string
	devices  []Device
	policies []Policy
	audit    []string
	token    string
	arcisURL string
}

func New(root string) *Desk {
	d := &Desk{root: root, arcisURL: "http://127.0.0.1:9090"}
	d.policies = []Policy{
		{Name: "SchoolDay", Mode: "school", MiningEnabled: false, ArcisForStudent: false},
		{Name: "AfterHours", Mode: "free", MiningEnabled: false, ArcisForStudent: false},
		{Name: "Exam", Mode: "exam", MiningEnabled: false, ArcisForStudent: false},
		{Name: "Lock", Mode: "lock", MiningEnabled: false, ArcisForStudent: false},
	}
	d.audit = append(d.audit, "desk started")
	return d
}

func DataDir() string {
	if v := os.Getenv("MAGIMDM_DATA"); v != "" {
		_ = os.MkdirAll(v, 0o755)
		return v
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		base, _ = os.UserConfigDir()
	}
	p := filepath.Join(base, "MagiMDM")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func (d *Desk) Listen(port int) (string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil && port != 0 {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.route)
	go http.Serve(ln, mux)
	return ln.Addr().String(), nil
}

func (d *Desk) route(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
		return
	}
	body := map[string]any{}
	if r.Body != nil {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(raw, &body)
	}
	switch {
	case r.URL.Path == "/health":
		writeJSON(w, 200, map[string]any{"status": "ok", "platform": "magimdm.exe", "arcis": d.arcisURL})
	case r.URL.Path == "/login" && r.Method == http.MethodPost:
		user, _ := body["user"].(string)
		pass, _ := body["pass"].(string)
		if user != "admin" || pass != "changeme" {
			writeJSON(w, 401, map[string]string{"error": "parent login failed"})
			return
		}
		buf := make([]byte, 8)
		_, _ = rand.Read(buf)
		d.mu.Lock()
		d.token = hex.EncodeToString(buf)
		d.mu.Unlock()
		writeJSON(w, 200, map[string]string{"token": d.token, "role": "parent"})
	case r.URL.Path == "/api/agent/enroll" && r.Method == http.MethodPost:
		name, _ := body["name"].(string)
		platform, _ := body["platform"].(string)
		token, _ := body["token"].(string)
		if name == "" {
			name = "pc"
		}
		if platform == "" {
			platform = "windows"
		}
		id := make([]byte, 8)
		_, _ = rand.Read(id)
		dev := Device{UUID: hex.EncodeToString(id), Name: name, Platform: platform, Token: token, Policy: "SchoolDay", LastSeen: time.Now().Format(time.RFC3339)}
		d.mu.Lock()
		d.devices = append(d.devices, dev)
		d.audit = append(d.audit, "enroll "+dev.Name)
		d.mu.Unlock()
		writeJSON(w, 200, map[string]string{"uuid": dev.UUID})
	case r.URL.Path == "/api/agent/poll" && r.Method == http.MethodPost:
		uuid, _ := body["uuid"].(string)
		d.mu.Lock()
		defer d.mu.Unlock()
		for i := range d.devices {
			if d.devices[i].UUID == uuid {
				d.devices[i].LastSeen = time.Now().Format(time.RFC3339)
				writeJSON(w, 200, map[string]any{"policy": d.policyByName(d.devices[i].Policy), "commands": []any{}})
				return
			}
		}
		writeJSON(w, 404, map[string]string{"error": "not enrolled"})
	case r.URL.Path == "/home":
		link := d.arcisHealth()
		d.mu.Lock()
		devs := append([]Device(nil), d.devices...)
		pols := append([]Policy(nil), d.policies...)
		d.mu.Unlock()
		writeJSON(w, 200, map[string]any{
			"devices": devs,
			"policies": pols,
			"arcis": link,
			"student_arcis": false,
		})
	case r.URL.Path == "/devices":
		d.mu.Lock()
		defer d.mu.Unlock()
		writeJSON(w, 200, map[string]any{"devices": d.devices})
	case r.URL.Path == "/policies":
		writeJSON(w, 200, map[string]any{"policies": d.policies})
	case r.URL.Path == "/interop/arcis":
		writeJSON(w, 200, d.arcisHealth())
	case r.URL.Path == "/interop/hours" && r.Method == http.MethodPost:
		text, _ := body["text"].(string)
		if text == "" {
			writeJSON(w, 400, map[string]string{"error": "missing text"})
			return
		}
		writeJSON(w, 200, d.pushHours(text))
	default:
		writeJSON(w, 404, map[string]string{"error": "not found"})
	}
}

func (d *Desk) policyByName(name string) Policy {
	for _, p := range d.policies {
		if p.Name == name {
			return p
		}
	}
	return d.policies[0]
}

func (d *Desk) arcisHealth() map[string]any {
	res, err := http.Get(d.arcisURL + "/health")
	if err != nil {
		return map[string]any{"arcis": "down", "url": d.arcisURL, "error": err.Error()}
	}
	defer res.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(res.Body).Decode(&payload)
	return map[string]any{"arcis": "up", "url": d.arcisURL, "health": payload}
}

func (d *Desk) pushHours(text string) map[string]any {
	raw, _ := json.Marshal(map[string]string{"title": "MagiMDM hour log", "text": text, "author": "magimdm"})
	req, err := http.NewRequest(http.MethodPost, d.arcisURL+"/library/ingest", strings.NewReader(string(raw)))
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "arcis": "down", "error": err.Error()}
	}
	defer res.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(res.Body).Decode(&payload)
	d.mu.Lock()
	d.audit = append(d.audit, "hour log sent to arcis")
	d.mu.Unlock()
	return map[string]any{"ok": res.StatusCode == 200, "arcis": payload}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func SelfTest() int {
	dir, err := os.MkdirTemp("", "magimdm-selftest-")
	if err != nil {
		fmt.Println("SELF-TEST FAILED", err)
		return 1
	}
	defer os.RemoveAll(dir)
	d := New(dir)
	addr, err := d.Listen(0)
	if err != nil {
		fmt.Println("SELF-TEST FAILED", err)
		return 1
	}
	res, err := http.Get("http://" + addr + "/health")
	if err != nil || res.StatusCode != 200 {
		fmt.Println("SELF-TEST FAILED health")
		return 1
	}
	fmt.Println("SELF-TEST PASSED — MagiMDM desk contained in magimdm.exe")
	return 0
}
