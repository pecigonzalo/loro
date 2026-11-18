package lib

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestUniqueIsStableAndJoinsArgs(t *testing.T) {
	prev := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = prev }()

	first := Unique("svc", "-a")
	if again := Unique("svc", "-a"); again != first {
		t.Errorf("same input gave different output: %q vs %q", first, again)
	}
	if !strings.Contains(first, "svc-a") {
		t.Errorf("args were not joined: %q", first)
	}
}

func TestUniqueRotatesThroughPool(t *testing.T) {
	prev := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = prev }()

	seen := map[string]bool{}
	for _, name := range []string{"rot-1", "rot-2", "rot-3", "rot-4"} {
		seen[Unique(name)[:5]] = true // ANSI prefix identifies the colour
	}
	if len(seen) != len(colorPool) {
		t.Errorf("expected %d distinct colours across %d new inputs, got %d", len(colorPool), len(colorPool), len(seen))
	}
}
