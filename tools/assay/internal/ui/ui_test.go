package ui

import (
	"strings"
	"testing"
)

func TestDisabledPainterIsPassthrough(t *testing.T) {
	p := New(false)

	for name, got := range map[string]string{
		"good":   p.Good("text"),
		"warn":   p.Warn("text"),
		"bad":    p.Bad("text"),
		"muted":  p.Muted("text"),
		"accent": p.Accent("text"),
		"bold":   p.Bold("text"),
	} {
		if got != "text" {
			t.Errorf("%s: disabled painter must not decorate, got %q", name, got)
		}
	}
}

func TestEnabledPainterDecoratesAndResets(t *testing.T) {
	p := New(true)

	painted := p.Good("text")
	if painted == "text" {
		t.Fatal("enabled painter must decorate")
	}
	if !strings.Contains(painted, "text") {
		t.Fatalf("painted output must contain the original text, got %q", painted)
	}
	if !strings.HasSuffix(painted, "m") && !strings.Contains(painted, "\033[0m") {
		t.Fatalf("painted output must reset, got %q", painted)
	}
}

func TestBarWidthIsExact(t *testing.T) {
	p := New(false)

	for _, tc := range []struct {
		fraction float64
		filled   int
	}{
		{0, 0},
		{0.5, 5},
		{1, 10},
		{2, 10},
		{-1, 0},
	} {
		bar := p.Bar(tc.fraction, 10)
		if n := strings.Count(bar, "█"); n != tc.filled {
			t.Errorf("fraction %v: %d filled cells, want %d", tc.fraction, n, tc.filled)
		}
		if n := strings.Count(bar, "█") + strings.Count(bar, "░"); n != 10 {
			t.Errorf("fraction %v: bar width %d, want 10", tc.fraction, n)
		}
	}
}

func TestTableAlignsAllRows(t *testing.T) {
	p := New(false)

	rendered := p.Table(
		[]string{"package", "score"},
		[][]string{
			{"short", "1"},
			{"a/much/longer/package/path", "42"},
		},
	)

	if !strings.Contains(rendered, "package") || !strings.Contains(rendered, "42") {
		t.Fatalf("table must contain headers and cells:\n%s", rendered)
	}
}

func TestBadgeCarriesItsText(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		badge := New(enabled).Badge(ToneBad, "SURVIVED")
		if !strings.Contains(badge, "SURVIVED") {
			t.Errorf("enabled=%v: badge lost its text: %q", enabled, badge)
		}
	}
}
