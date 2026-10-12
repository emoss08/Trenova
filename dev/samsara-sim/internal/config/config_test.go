package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestDefaultEnablesRateLimitsAndDisablesState(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if !cfg.RateLimits.Enabled || cfg.RateLimits.Multiplier != 1 {
		t.Fatalf("expected rate limits enabled at 1x, got %+v", cfg.RateLimits)
	}
	if cfg.State.Path != "" || cfg.State.FlushDelay != 250*time.Millisecond {
		t.Fatalf("expected persistence disabled with a 250ms flush delay, got %+v", cfg.State)
	}
}

func TestLoadKeepsRateLimitDefaultsWhenOmitted(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, "server:\n  port: 9000\n"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.RateLimits.Enabled || cfg.RateLimits.Multiplier != 1 {
		t.Fatalf("expected default rate limits, got %+v", cfg.RateLimits)
	}
	if cfg.State.Path != "" {
		t.Fatalf("expected persistence to stay disabled, got %q", cfg.State.Path)
	}
}

func TestLoadParsesRateLimitsAndState(t *testing.T) {
	t.Parallel()

	cfg, err := Load(writeConfig(t, `
rateLimits:
  enabled: false
  multiplier: 2.5
state:
  path: "  ./data/state.json  "
  flushDelay: 1s
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.RateLimits.Enabled || cfg.RateLimits.Multiplier != 2.5 {
		t.Fatalf("expected disabled 2.5x rate limits, got %+v", cfg.RateLimits)
	}
	if cfg.State.Path != "./data/state.json" || cfg.State.FlushDelay != time.Second {
		t.Fatalf("expected trimmed state path and 1s delay, got %+v", cfg.State)
	}
}

func TestLoadNormalizesInvalidRateLimitAndStateValues(t *testing.T) {
	t.Parallel()

	for _, multiplier := range []string{"0", "-3", ".nan", ".inf"} {
		cfg, err := Load(writeConfig(t, "rateLimits:\n  multiplier: "+multiplier+
			"\nstate:\n  flushDelay: -5s\n"))
		if err != nil {
			t.Fatalf("load multiplier %s: %v", multiplier, err)
		}
		if cfg.RateLimits.Multiplier != 1 {
			t.Fatalf(
				"expected multiplier %s to normalize to 1, got %v",
				multiplier,
				cfg.RateLimits.Multiplier,
			)
		}
		if cfg.State.FlushDelay != 250*time.Millisecond {
			t.Fatalf("expected flushDelay to normalize, got %v", cfg.State.FlushDelay)
		}
	}
}
