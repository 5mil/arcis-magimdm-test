package desk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDesk(t *testing.T) {
	d := New(t.TempDir())
	addr, err := d.Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + addr
	res, err := http.Get(base + "/health")
	if err != nil || res.StatusCode != 200 {
		t.Fatal(res.StatusCode, err)
	}
	page, _ := http.Get(base + "/")
	html, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if !strings.Contains(string(html), "MagiMDM") {
		t.Fatal("ui missing")
	}
	raw, _ := json.Marshal(map[string]string{"name": "study-pc", "platform": "windows", "token": "desk"})
	enroll, err := http.Post(base+"/api/agent/enroll", "application/json", bytes.NewReader(raw))
	if err != nil || enroll.StatusCode != 200 {
		t.Fatalf("enroll %v", enroll.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(enroll.Body).Decode(&body)
	if body["uuid"] == "" {
		t.Fatal("no uuid")
	}
	pollRaw, _ := json.Marshal(map[string]string{"uuid": body["uuid"]})
	poll, err := http.Post(base+"/api/agent/poll", "application/json", bytes.NewReader(pollRaw))
	if err != nil || poll.StatusCode != 200 {
		t.Fatal(poll.StatusCode, err)
	}
	var pol map[string]any
	_ = json.NewDecoder(poll.Body).Decode(&pol)
	policy, _ := pol["policy"].(map[string]any)
	if policy["mining_enabled"] != false || policy["arcis_for_student"] != false {
		t.Fatalf("policy %+v", policy)
	}
	link, err := http.Get(base + "/interop/arcis")
	if err != nil || link.StatusCode != 200 {
		t.Fatal("interop")
	}
}
