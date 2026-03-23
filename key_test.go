package cachez_test

import (
	"testing"

	cachez "github.com/dreamph/cachez"
)

func TestGetKey(t *testing.T) {
	if got := cachez.GetKey("1", "2", "3"); got != "1:2:3" {
		t.Fatalf("expected 1:2:3, got %q", got)
	}

	if got := cachez.GetKey("1:", ":2", ":3"); got != "1:2:3" {
		t.Fatalf("expected 1:2:3, got %q", got)
	}

	if got := cachez.GetKey(); got != "" {
		t.Fatalf("expected empty string for no keys, got %q", got)
	}
}
