package logs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestALineKeepsOneRowAndDoesNotEndInAnEscapedBreak(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 17, 15, 23, 15, 588000000, time.UTC)
	same := func(s string) string { return s }
	cases := []struct {
		name, stored, want string
	}{
		{"a line stored with its own break", "{\"level\":\"info\",\"msg\":\"ready\"}\n", `{"level":"info","msg":"ready"}`},
		{"a line stored with a carriage return and a break", "ready\r\n", "ready"},
		{"a line stored without one", "ready", "ready"},
		{"breaks inside a line stay, escaped", "panic: boom\n\tgoroutine 1 [running]\n", `panic: boom\n` + "\tgoroutine 1 [running]"},
	}
	for _, c := range cases {
		e := &Entry{Time: at, Pod: "trenova-api-0", Container: "api", Line: c.stored}
		require.Equal(t, "trenova-api-0/api: 2026-11-17T15:23:15.588Z "+c.want+"\n", viewLine(e, same), c.name)
		require.Equal(t, "2026-11-17T15:23:15.588Z trenova-api-0 api "+c.want+"\n", bundleLine(e, same), c.name)
	}
}
