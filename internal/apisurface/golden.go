package apisurface

import (
	"os"
	"testing"
)

// AssertGolden generates the API surface for dir and compares it against the
// golden file at goldenPath. Run with UPDATE_GOLDEN=1 to (re)write the
// golden file after a deliberate, reviewed API change.
func AssertGolden(t *testing.T, dir, goldenPath string) {
	t.Helper()

	got, err := Generate(dir)
	if err != nil {
		t.Fatalf("generating API surface for %s: %v", dir, err)
	}

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden file %s: %v", goldenPath, err)
		}
		t.Logf("wrote golden file %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s (run with UPDATE_GOLDEN=1 to create it): %v", goldenPath, err)
	}

	if got != string(want) {
		t.Errorf(
			"API surface for %s does not match %s.\n"+
				"If this change is intentional, update the golden file with:\n"+
				"  UPDATE_GOLDEN=1 go test ./... -run TestAPISurface\n"+
				"and add a changie fragment for the change.\n\ngot:\n%s\nwant:\n%s",
			dir, goldenPath, got, want,
		)
	}
}
