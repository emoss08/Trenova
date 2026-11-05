package capturedevices

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPairingRequest() *StartPairingRequest {
	return &StartPairingRequest{
		MachineName:  "DISPATCH-07",
		WindowsUser:  `ACME\jdoe`,
		AgentVersion: "1.4.2",
		Architecture: capture.ArchitectureX64,
		OSVersion:    "Windows 11 Pro 23H2",
	}
}

func TestValidateMachine_AcceptsARealMachine(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateMachine(validPairingRequest()))
}

func TestValidateMachine_OptionalFieldsMayBeEmpty(t *testing.T) {
	t.Parallel()

	req := validPairingRequest()
	req.WindowsUser = ""
	req.OSVersion = ""

	require.NoError(t, validateMachine(req))
}

func TestValidateMachine_BoundsEveryFieldToItsColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*StartPairingRequest)
		field  string
	}{
		{
			name: "machine name",
			mutate: func(r *StartPairingRequest) {
				r.MachineName = strings.Repeat("m", maxMachineNameLength+1)
			},
			field: "machineName",
		},
		{
			name: "windows user",
			mutate: func(r *StartPairingRequest) {
				r.WindowsUser = strings.Repeat("u", maxWindowsUserLength+1)
			},
			field: "windowsUser",
		},
		{
			name: "agent version",
			mutate: func(r *StartPairingRequest) {
				r.AgentVersion = strings.Repeat("1", maxAgentVersionLength+1)
			},
			field: "agentVersion",
		},
		{
			name: "os version",
			mutate: func(r *StartPairingRequest) {
				r.OSVersion = strings.Repeat("w", maxOSVersionLength+1)
			},
			field: "osVersion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := validPairingRequest()
			tt.mutate(req)

			err := validateMachine(req)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
		})
	}
}

func TestValidateMachine_RefusesTextThatCannotBeShownAsIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*StartPairingRequest)
		field  string
	}{
		{
			name:   "a direction override",
			mutate: func(r *StartPairingRequest) { r.MachineName = "PC\u202eTNEMYAP" },
			field:  "machineName",
		},
		{
			name:   "a line break inside the text",
			mutate: func(r *StartPairingRequest) { r.WindowsUser = "jdoe\nApproved by IT" },
			field:  "windowsUser",
		},
		{
			name:   "a control character",
			mutate: func(r *StartPairingRequest) { r.OSVersion = "Windows\x1b[2J" },
			field:  "osVersion",
		},
		{
			name:   "a zero width character",
			mutate: func(r *StartPairingRequest) { r.AgentVersion = "1.4\u200b.2" },
			field:  "agentVersion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := validPairingRequest()
			tt.mutate(req)

			err := validateMachine(req)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.field)
		})
	}
}

func TestValidateMachine_AcceptsNamesInAnyScript(t *testing.T) {
	t.Parallel()

	req := validPairingRequest()
	req.MachineName = "  事務所-PC  "
	req.WindowsUser = `ACME\josé`

	require.NoError(t, validateMachine(req))
}

func TestSameNetwork(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "one address", left: "203.0.113.7", right: "203.0.113.7", want: true},
		{name: "two addresses", left: "203.0.113.7", right: "203.0.113.8"},
		{name: "mapped v4", left: "::ffff:203.0.113.7", right: "203.0.113.7", want: true},
		{name: "one v6 network", left: "2001:db8:1:2::1", right: "2001:db8:1:2::9", want: true},
		{name: "two v6 networks", left: "2001:db8:1:2::1", right: "2001:db8:1:3::1"},
		{name: "v4 and v6", left: "203.0.113.7", right: "2001:db8:1:2::1"},
		{name: "no machine address", left: "203.0.113.7", right: ""},
		{name: "no reader address", left: "", right: "203.0.113.7"},
		{name: "not an address", left: "nonsense", right: "nonsense"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, sameNetwork(tt.left, tt.right))
		})
	}
}

func TestAskedFromSameNetwork_ReadsTheRequestAddress(t *testing.T) {
	t.Parallel()

	ctx := requestmeta.With(t.Context(), requestmeta.New("req", "203.0.113.7", "browser"))

	assert.True(t, askedFromSameNetwork(ctx, "203.0.113.7"))
	assert.False(t, askedFromSameNetwork(ctx, "198.51.100.1"))
	assert.False(t, askedFromSameNetwork(t.Context(), "203.0.113.7"))
}
