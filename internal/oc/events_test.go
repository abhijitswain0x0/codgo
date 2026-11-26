package oc

import (
	"encoding/json"
	"testing"
)

func TestDecode(t *testing.T) {
	raw := `{"type":"permission.asked","properties":{"id":"per_1","sessionID":"ses_1","permission":"bash","patterns":["echo hi"],"metadata":{"command":"echo hi"},"always":["echo *"]}}`
	var e Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatal(err)
	}
	p, ok := Decode(e).(PermissionAsked)
	if !ok || p.ID != "per_1" || p.Metadata["command"] != "echo hi" {
		t.Fatalf("bad decode: %+v", Decode(e))
	}
	if Decode(Event{Type: "plugin.added"}) != nil {
		t.Fatal("noise event should be dropped")
	}
}
