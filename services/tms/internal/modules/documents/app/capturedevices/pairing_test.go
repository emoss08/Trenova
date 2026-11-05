package capturedevices

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
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
