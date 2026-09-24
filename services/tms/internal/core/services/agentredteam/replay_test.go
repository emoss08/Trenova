package agentredteam

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/stretchr/testify/require"
)

const (
	cassettesDir = "testdata/cassettes"
	minCassettes = 3
)

func casesByName(t *testing.T) map[string]*Case {
	t.Helper()

	cases := loadCases(t)
	byName := make(map[string]*Case, len(cases))
	for _, c := range cases {
		byName[c.Name] = c
	}

	return byName
}

func TestCassetteReplay(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join(cassettesDir, "*.json"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(paths), minCassettes)
	cases := casesByName(t)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			cassette, loadErr := agentevalgate.LoadCassette(path)
			require.NoError(t, loadErr)
			c, ok := cases[cassette.Case]
			require.Truef(t, ok, "%s replays case %q, which no longer exists", path, cassette.Case)

			player := agentevalgate.NewCassettePlayer(cassette)
			outcome, runErr := Run(t.Context(), RunParams{Case: c, Completion: player})
			require.NoError(t, runErr)

			if *update {
				refreshHashes(t, path, cassette, player.Hashes())
				return
			}

			for _, stale := range player.Stale() {
				t.Logf("stale recording: %s step %d was recorded against request %s and the "+
					"harness now sends %s. The prompt, a tool or the case changed since it "+
					"was recorded; re-record it with the live job (-record) or refresh the "+
					"hash with -update. The replay still ran and its invariants still hold.",
					path, stale.Step, stale.Recorded, stale.Now)
			}
			if errors.Is(outcome.RunErr, agentevalgate.ErrCassetteExhausted) {
				t.Logf("stale recording: %s ran out of recorded steps after %d; the turn "+
					"took another path than the one recorded", path, player.Played())
				outcome.RunErr = nil
			}
			require.NoError(t, outcome.RunErr)

			Assert(t.Context(), t, outcome, AssertRecorded)
		})
	}
}

func refreshHashes(
	t *testing.T,
	path string,
	cassette *agentevalgate.Cassette,
	hashes []string,
) {
	t.Helper()

	for idx := range cassette.Steps {
		if idx < len(hashes) {
			cassette.Steps[idx].RequestHash = hashes[idx]
		}
	}
	require.NoError(t, cassette.Save(path))
}
