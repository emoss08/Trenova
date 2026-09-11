package config

import "testing"

func TestGetGraphQLAccessMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode GraphQLAccessMode
		want GraphQLAccessMode
	}{
		{name: "unset defaults to disabled", mode: "", want: GraphQLAccessModeDisabled},
		{name: "unknown defaults to disabled", mode: "bogus", want: GraphQLAccessModeDisabled},
		{name: "disabled", mode: GraphQLAccessModeDisabled, want: GraphQLAccessModeDisabled},
		{name: "observe", mode: GraphQLAccessModeObserve, want: GraphQLAccessModeObserve},
		{name: "enforce", mode: GraphQLAccessModeEnforce, want: GraphQLAccessModeEnforce},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &PlatformControlPlaneConfig{GraphQLAccessMode: tt.mode}
			if got := cfg.GetGraphQLAccessMode(); got != tt.want {
				t.Fatalf("GetGraphQLAccessMode() = %q, want %q", got, tt.want)
			}
		})
	}
}
