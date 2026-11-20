package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{name: "no command", args: nil, code: exitUsage, want: "conform"},
		{name: "help", args: []string{"help"}, code: exitOK, want: "commands:"},
		{name: "unknown command", args: []string{"soar"}, code: exitUsage, want: `"soar"`},
		{name: "conform flags", args: []string{"conform", "-h"}, code: exitOK, want: "-replay"},
		{name: "serve flags", args: []string{"serve", "-h"}, code: exitOK, want: "-public-url"},
		{name: "run flags", args: []string{"run", "-h"}, code: exitOK, want: "-trenova-cli"},
		{name: "soak flags", args: []string{"soak", "-h"}, code: exitOK, want: "-conversations"},
		{
			name: "a run out of range",
			args: []string{"run", "--run", "900", "--run-file", ""},
			code: exitUsage,
			want: "--run is 1..899",
		},
		{
			name: "serve an unknown partner",
			args: []string{"serve", "--partners", "../../partners", "--partner", "nobody"},
			code: exitFailed,
			want: `no partner "nobody"`,
		},
		{
			name: "a bad replay",
			args: []string{"conform", "--replay", "manufacturer"},
			code: exitUsage,
			want: "not <partner>:<seed>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := dispatch(context.Background(), tt.args, &stdout, &stderr)
			if code != tt.code || !strings.Contains(stderr.String()+stdout.String(), tt.want) {
				t.Fatalf("code %d, want %d; output:\n%s%s", code, tt.code, stdout.String(),
					stderr.String())
			}
		})
	}
}

func TestParseReplay(t *testing.T) {
	t.Parallel()

	name, seed, err := parseReplay("manufacturer:37")
	if err != nil || name != "manufacturer" || seed != 37 {
		t.Fatalf("got %q %d %v", name, seed, err)
	}
	for _, bad := range []string{":4", "manufacturer:", "manufacturer:0", "manufacturer:x"} {
		if _, _, err := parseReplay(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestTheNextRunFollowsTheLast(t *testing.T) {
	t.Parallel()
	path := t.TempDir() + "/run"
	first, err := nextRun(0, path)
	if err != nil || first != 1 {
		t.Fatalf("first run: %d %v", first, err)
	}
	if second, _ := nextRun(0, path); second != 2 {
		t.Fatalf("second run: %d", second)
	}
	if given, _ := nextRun(899, path); given != 899 {
		t.Fatalf("a given run: %d", given)
	}
	if wrapped, _ := nextRun(0, path); wrapped != 1 {
		t.Fatalf("the run after 899: %d", wrapped)
	}
}

func TestTheDefaultPublicURLFollowsTheListenPort(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ listen, want string }{
		{":9310", "localhost:9310"},
		{"0.0.0.0:9310", "localhost:9310"},
		{"127.0.0.1:9311", "127.0.0.1:9311"},
	} {
		if got := localHost(c.listen); got != c.want {
			t.Errorf("localHost(%q) = %q, want %q", c.listen, got, c.want)
		}
	}
}
