package main

import (
	"path/filepath"
	"testing"
)

// TestNoUsableDB covers the fresh/empty-DB startup guard: openDB must not
// fail on a table-less file, loadInitial must fail, and the two recovery
// paths (empty start, adopt) must leave a usable database.
func TestNoUsableDB(t *testing.T) {
	appDir = t.TempDir()
	dbPath = filepath.Join(appDir, "test.db")

	if err := openDB(); err != nil {
		t.Fatalf("openDB on fresh file must not fail: %v", err)
	}
	if err := loadInitial(); err == nil {
		t.Fatal("loadInitial on fresh file must fail (no tables)")
	}

	if err := createEmptyDB(); err != nil {
		t.Fatalf("createEmptyDB: %v", err)
	}
	if len(bens) != 0 || nomBS != "" {
		t.Fatalf("unexpected state after empty start: bens=%v nomBS=%q", bens, nomBS)
	}

	if err := adoptDB(dbPath); err != nil {
		t.Fatalf("adoptDB(self): %v", err)
	}
	if err := loadInitial(); err != nil {
		t.Fatalf("loadInitial after adopt(self): %v", err)
	}
}
