package ui

import "testing"

func TestParseUnified(t *testing.T) {
	d := "Index: f\n===\n--- f\n+++ f\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+BETA\n gamma\n"
	ls := parseUnified(d)
	if len(ls) != 4 {
		t.Fatalf("got %d lines", len(ls))
	}
	if ls[1].op != '-' || ls[1].num != 2 || ls[2].op != '+' || ls[2].num != 2 {
		t.Fatalf("bad numbering: %+v", ls)
	}
	if a, r := diffCounts(ls); a != 1 || r != 1 {
		t.Fatalf("counts %d %d", a, r)
	}
}

func TestSynthWrite(t *testing.T) {
	ls := synthDiff(map[string]any{"content": "a\nb\n"})
	if len(ls) != 2 || ls[1].num != 2 {
		t.Fatalf("%+v", ls)
	}
}
