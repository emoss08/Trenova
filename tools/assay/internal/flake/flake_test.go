package flake

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func obs(pkg, test, fp string, verdict Verdict, unix int64) Observation {
	return Observation{
		Package: pkg, Test: test, Fingerprint: fp,
		Verdict: verdict, Source: "test", Unix: unix,
	}
}

// TestDetectRequiresConflictAtTheSameFingerprint pins the definition: a
// verdict that changed across fingerprints is the code changing, and reporting
// it as a flake would teach people to ignore the report.
func TestDetectRequiresConflictAtTheSameFingerprint(t *testing.T) {
	flakes := Detect([]Observation{
		obs("p", "TestMoved", "fp1", VerdictPass, 1),
		obs("p", "TestMoved", "fp2", VerdictFail, 2),
		obs("p", "TestRed", "fp1", VerdictFail, 1),
		obs("p", "TestRed", "fp1", VerdictFail, 2),
		obs("p", "TestFlaky", "fp1", VerdictPass, 1),
		obs("p", "TestFlaky", "fp1", VerdictFail, 2),
	})

	require.Len(t, flakes, 1, "only the same-fingerprint conflict is a flake")
	assert.Equal(t, "TestFlaky", flakes[0].Test)
	assert.Equal(t, 1, flakes[0].Passes())
	assert.Equal(t, 1, flakes[0].Fails())
}

func TestDetectCountsTimeoutAsAFailingVerdict(t *testing.T) {
	flakes := Detect([]Observation{
		obs("p", "TestSlow", "fp1", VerdictPass, 1),
		obs("p", "TestSlow", "fp1", VerdictTimeout, 2),
	})

	require.Len(t, flakes, 1)
	assert.Equal(t, 1, flakes[0].Fails())
}

func TestDetectReportsOnlyConflictingEvidence(t *testing.T) {
	flakes := Detect([]Observation{
		obs("p", "TestFlaky", "fpOld", VerdictFail, 1),
		obs("p", "TestFlaky", "fpNew", VerdictPass, 2),
		obs("p", "TestFlaky", "fpNew", VerdictFail, 3),
	})

	require.Len(t, flakes, 1)
	require.Len(t, flakes[0].Evidence, 1,
		"the always-failing old fingerprint is a red test at that state, not evidence of flakiness")
	assert.Equal(t, "fpNew", flakes[0].Evidence[0].Fingerprint)
}

func TestJournalRoundTrips(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), "/some/root")
	require.NoError(t, err)

	want := []Observation{
		obs("p", "TestA", "fp1", VerdictPass, 10),
		obs("p", "TestB", "fp1", VerdictFail, 11),
	}
	require.NoError(t, j.Record(want))

	got, err := j.Load()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestJournalIsNilSafe(t *testing.T) {
	var j *Journal

	require.NoError(t, j.Record([]Observation{obs("p", "T", "fp", VerdictPass, 1)}))
	got, err := j.Load()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestJournalSeparatesWorkspaces(t *testing.T) {
	cacheDir := t.TempDir()

	a, err := OpenJournal(cacheDir, "/repo/a")
	require.NoError(t, err)
	b, err := OpenJournal(cacheDir, "/repo/b")
	require.NoError(t, err)

	require.NoError(t, a.Record([]Observation{obs("p", "TestA", "fp", VerdictPass, 1)}))

	fromB, err := b.Load()
	require.NoError(t, err)
	assert.Empty(t, fromB, "two repositories must never mix evidence")
}

// TestJournalSurvivesConcurrentWriters pins the locking: a watch session and a
// mutation run both record, and neither may tear or drop the other's lines.
func TestJournalSurvivesConcurrentWriters(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), "/root")
	require.NoError(t, err)

	const writers, perWriter = 8, 25
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perWriter {
				_ = j.Record([]Observation{
					obs("p", "Test", "fp", VerdictPass, int64(w*1000+i)),
				})
			}
		}()
	}
	wg.Wait()

	got, err := j.Load()
	require.NoError(t, err)
	assert.Len(t, got, writers*perWriter, "concurrent appends must not lose or tear lines")
}

func TestJournalSkipsTornLines(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), "/root")
	require.NoError(t, err)
	require.NoError(t, j.Record([]Observation{obs("p", "TestA", "fp", VerdictPass, 1)}))

	file, err := os.OpenFile(j.path(), os.O_WRONLY|os.O_APPEND, 0o644)
	require.NoError(t, err)
	_, err = file.WriteString(`{"package":"p","test":"TestTo`)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	got, loadErr := j.Load()
	require.NoError(t, loadErr, "a torn line from a crashed process must not make the history unreadable")
	assert.Len(t, got, 1)
}

func TestCompactionKeepsTheNewestObservations(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), "/root")
	require.NoError(t, err)

	var all []Observation
	for i := range maxPerTest + 50 {
		all = append(all, obs("p", "TestBusy", "fp", VerdictPass, int64(i)))
	}
	all = append(all, obs("p", "TestQuiet", "fp", VerdictFail, 5))
	require.NoError(t, j.Record(all))

	require.NoError(t, j.compactLocked())

	got, err := j.Load()
	require.NoError(t, err)

	busy := 0
	quiet := 0
	var minBusy int64 = 1 << 60
	for _, o := range got {
		switch o.Test {
		case "TestBusy":
			busy++
			if o.Unix < minBusy {
				minBusy = o.Unix
			}
		case "TestQuiet":
			quiet++
		}
	}
	assert.Equal(t, maxPerTest, busy, "compaction bounds each test's history")
	assert.Equal(t, 1, quiet, "compaction must not evict other tests' evidence")
	assert.Equal(t, int64(50), minBusy, "the newest observations are the ones kept")
}

func TestParseVerboseOutputTakesTopLevelVerdictsOnly(t *testing.T) {
	output := []byte(`=== RUN   TestAlpha
--- PASS: TestAlpha (0.00s)
=== RUN   TestBeta
=== RUN   TestBeta/case
    --- FAIL: TestBeta/case (0.00s)
--- FAIL: TestBeta (0.01s)
=== RUN   TestGamma
--- PASS: TestGamma (0.00s)
FAIL
`)

	passes, fails := ParseVerboseOutput(output)

	assert.Equal(t, []string{"TestAlpha", "TestGamma"}, passes)
	assert.Equal(t, []string{"TestBeta"}, fails,
		"the failing subtest already fails its parent, which is the name -test.run selects")
}

func TestParseVerboseOutputLetsAFailureWin(t *testing.T) {
	output := []byte("--- PASS: TestOdd (0.00s)\n--- FAIL: TestOdd (0.00s)\n")

	passes, fails := ParseVerboseOutput(output)

	assert.Empty(t, passes)
	assert.Equal(t, []string{"TestOdd"}, fails,
		"a contradictory parse must resolve toward the failure; losing one is the dangerous direction")
}

func TestQuarantineRoundTripsAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".assay", "flaky-tests.json")

	q := NewQuarantine()
	assert.Equal(t, 1, q.Add(QuarantineEntry{Package: "p", Test: "TestFlaky", Note: "n"}))
	assert.Equal(t, 0, q.Add(QuarantineEntry{Package: "p", Test: "TestFlaky"}),
		"re-quarantining must not duplicate the entry")
	require.NoError(t, q.Save(path))

	loaded, err := LoadQuarantine(path)
	require.NoError(t, err)
	assert.True(t, loaded.Contains("p", "TestFlaky"))
	assert.False(t, loaded.Contains("p", "TestOther"))
	assert.Equal(t, map[string][]string{"p": {"TestFlaky"}}, loaded.Blocked())
}

func TestQuarantineIsNilSafe(t *testing.T) {
	var q *Quarantine

	assert.False(t, q.Contains("p", "Test"))
	assert.Zero(t, q.Len())
	assert.Nil(t, q.Blocked())
}

func TestMissingQuarantineIsEmpty(t *testing.T) {
	q, err := LoadQuarantine(filepath.Join(t.TempDir(), "absent.json"))

	require.NoError(t, err)
	assert.Zero(t, q.Len())
}
