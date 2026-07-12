package state

import (
	"testing"
	"time"
)

func TestPutGetRemove(t *testing.T) {
	dir := t.TempDir()
	p := Proc{App: "djin", PID: 1234, URL: "http://127.0.0.1:5000", Started: time.Now()}
	if err := Put(dir, p); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Get(dir, "djin")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got.PID != 1234 {
		t.Errorf("PID = %d", got.PID)
	}

	// Upsert replaces, not duplicates.
	p.PID = 5678
	if err := Put(dir, p); err != nil {
		t.Fatal(err)
	}
	procs, _ := Load(dir)
	if len(procs) != 1 || procs[0].PID != 5678 {
		t.Errorf("upsert failed: %+v", procs)
	}

	if err := Remove(dir, "djin"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := Get(dir, "djin"); ok {
		t.Error("entry should be gone after Remove")
	}
}

func TestGetMissingAndEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if _, ok, err := Get(dir, "nope"); ok || err != nil {
		t.Errorf("ok=%v err=%v", ok, err)
	}
	if err := Remove(dir, "nope"); err != nil {
		t.Errorf("Remove on empty state should be a no-op: %v", err)
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	Put(dir, Proc{App: "alive", PID: 1, URL: "http://127.0.0.1:1", Started: time.Now()})
	Put(dir, Proc{App: "dead", PID: 2, URL: "http://127.0.0.1:2", Started: time.Now()})
	survivors, err := Prune(dir, func(p Proc) bool { return p.App == "alive" })
	if err != nil {
		t.Fatal(err)
	}
	if len(survivors) != 1 || survivors[0].App != "alive" {
		t.Errorf("survivors = %+v", survivors)
	}
	procs, _ := Load(dir)
	if len(procs) != 1 {
		t.Errorf("prune should persist: %+v", procs)
	}
}
