package lib

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

// resetColors forces colours on and isolates the package-global colour state
// for the duration of a test.
func resetColors(t *testing.T) {
	t.Helper()
	prevNoColor, prevIndex, prevUsed := color.NoColor, colorIndex, usedColors
	color.NoColor = false
	colorIndex, usedColors = 0, map[string]int{}
	t.Cleanup(func() {
		color.NoColor, colorIndex, usedColors = prevNoColor, prevIndex, prevUsed
	})
}

func TestUniqueIsStableAndJoinsArgs(t *testing.T) {
	resetColors(t)

	first := Unique("svc", "-a")
	if again := Unique("svc", "-a"); again != first {
		t.Errorf("same input gave different output: %q vs %q", first, again)
	}
	if !strings.Contains(first, "svc-a") {
		t.Errorf("args were not joined: %q", first)
	}
}

func TestUniqueRotatesThroughPool(t *testing.T) {
	resetColors(t)

	for i, name := range []string{"rot-1", "rot-2", "rot-3", "rot-4", "rot-5"} {
		want := colorPool[i%len(colorPool)].Sprint(name)
		if got := Unique(name); got != want {
			t.Errorf("Unique(%q) = %q, want %q", name, got, want)
		}
	}
}
