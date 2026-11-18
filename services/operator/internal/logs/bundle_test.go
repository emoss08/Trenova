package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/redact"
)

var t0 = time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

func testSpec(dir string) *bundleSpec {
	return &bundleSpec{
		bundleID:   uuid.New(),
		instanceID: uuid.New(),
		since:      t0,
		until:      t0.Add(24 * time.Hour),
		dir:        dir,
		redact:     redact.New([][]byte{[]byte("s3cr3t-db-pass")}).Line,
		created:    t0.Add(25 * time.Hour),
		pageLimit:  4,
	}
}

func readBundle(t *testing.T, b *Bundle) (map[string]string, Manifest) {
	t.Helper()
	body, err := os.ReadFile(b.Path)
	require.NoError(t, err)
	require.Equal(t, int64(len(body)), b.Size)
	files := unzip(t, body)
	var m Manifest
	require.NoError(t, sonic.Unmarshal([]byte(files[manifestName]), &m))
	return files, m
}

func lines(text string) []string {
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

func component(m Manifest, name string) ComponentResult {
	for _, c := range m.Components {
		if c.Component == name {
			return c
		}
	}
	return ComponentResult{}
}

func TestThePagerReturnsEveryEntryOnceNewestFirstAcrossPageBoundaries(t *testing.T) {
	t.Parallel()
	src := newSource()
	for i := range 25 {
		pod := "api-a"
		if i%2 == 1 {
			pod = "api-b"
		}
		src.add("api", t0.Add(time.Duration(i/2)*time.Minute), pod, "api", fmt.Sprintf("line %02d", i))
	}
	p := newPager(src, "api", t0, t0.Add(time.Hour), 4)

	var got []Entry
	for {
		e, ok, err := p.next(t.Context())
		require.NoError(t, err)
		if !ok {
			break
		}
		got = append(got, e)
	}

	require.Len(t, got, 25)
	seen := map[string]bool{}
	for i, e := range got {
		require.False(t, seen[e.Line], "%s twice", e.Line)
		seen[e.Line] = true
		if i > 0 {
			require.False(t, e.Time.After(got[i-1].Time), "newest first")
		}
	}
	assert.Greater(t, src.calls, 6, "the period was read a page at a time")
}

func TestAPageThatIsTooLargeIsAskedForAgainSmaller(t *testing.T) {
	t.Parallel()
	src := newSource()
	for i := range 10 {
		src.add("api", t0.Add(time.Duration(i)*time.Second), "api-a", "api", fmt.Sprintf("line %d", i))
	}
	src.tooLarge = 15
	p := newPager(src, "api", t0, t0.Add(time.Hour), 40)

	count := 0
	for {
		_, ok, err := p.next(t.Context())
		require.NoError(t, err)
		if !ok {
			break
		}
		count++
	}
	require.Equal(t, 10, count)
	assert.Contains(t, src.limits, 20)
	assert.Equal(t, 10, src.limits[len(src.limits)-1])
}

func TestABundleHasOneFilePerSourceOldestFirstAndAManifest(t *testing.T) {
	t.Parallel()
	src := newSource()
	for i := range 25 {
		src.add("api", t0.Add(time.Duration(i)*time.Minute), "trenova-api-1", "api", fmt.Sprintf("request %02d", i))
	}
	src.add("worker", t0.Add(time.Hour), "trenova-worker-1", "worker", "job ran\nsecond line")
	spec := testSpec(t.TempDir())

	b, err := writeBundle(t.Context(), src, spec, DefaultCaps)
	require.NoError(t, err)
	files, m := readBundle(t, b)

	api := lines(files["api.log"])
	require.Len(t, api, 25)
	assert.Equal(t, "2026-11-14T00:00:00Z trenova-api-1 api request 00", api[0])
	assert.Equal(t, "2026-11-14T00:24:00Z trenova-api-1 api request 24", api[24])
	assert.Equal(t, "2026-11-14T01:00:00Z trenova-worker-1 worker job ran\\nsecond line\n", files["worker.log"])
	assert.Equal(t, spec.bundleID, m.BundleID)
	assert.Equal(t, spec.instanceID, m.InstanceID)
	assert.False(t, m.TotalCapReached)
	assert.Equal(t, int64(25), component(m, "api").Lines)
	assert.False(t, component(m, "api").Truncated)
	assert.Equal(t, t0, *component(m, "api").Oldest)
	assert.Equal(t, []string{spec.bundleID.String() + ".zip.part"}, scratchFiles(t, spec.dir),
		"only the zip is left, for the caller to upload and remove")
}

func TestASourceOverItsCapKeepsItsNewestLinesAndSaysSo(t *testing.T) {
	t.Parallel()
	src := newSource()
	for i := range 100 {
		src.add("api", t0.Add(time.Duration(i)*time.Second), "trenova-api-1", "api", fmt.Sprintf("request %02d", i))
	}
	caps := Caps{Period: 7 * 24 * time.Hour, Component: 1000, Total: 1 << 20, Sources: 50}

	b, err := writeBundle(t.Context(), src, testSpec(t.TempDir()), caps)
	require.NoError(t, err)
	files, m := readBundle(t, b)

	api := component(m, "api")
	assert.True(t, api.Truncated)
	assert.Equal(t, CutComponent, api.Cut)
	assert.LessOrEqual(t, api.Bytes, int64(1000))
	assert.Less(t, api.Lines, int64(100))
	got := lines(files["api.log"])
	assert.Len(t, got, int(api.Lines))
	assert.True(t, strings.HasSuffix(got[len(got)-1], "request 99"), "the newest line is kept")
	assert.False(t, m.TotalCapReached)
	assert.Equal(t, int64(1000), m.Caps.ComponentBytes)
}

func TestTheBundleCapIsSharedAndASmallSourceIsKeptWhole(t *testing.T) {
	t.Parallel()
	src := newSource()
	padding := strings.Repeat("x", 160)
	for i := range 200 {
		at := t0.Add(time.Duration(i) * time.Second)
		src.add("api", at, "trenova-api-1", "api", fmt.Sprintf("%03d %s", i, padding))
		src.add("worker", at, "trenova-worker-1", "worker", fmt.Sprintf("%03d %s", i, padding))
	}
	src.add("redis", t0, "trenova-redis-0", "redis", "ready")
	caps := Caps{Period: 7 * 24 * time.Hour, Component: 1 << 20, Total: 30000, Sources: 50}

	b, err := writeBundle(t.Context(), src, testSpec(t.TempDir()), caps)
	require.NoError(t, err)
	_, m := readBundle(t, b)

	assert.True(t, m.TotalCapReached)
	var total int64
	for _, c := range m.Components {
		total += c.Bytes
	}
	assert.LessOrEqual(t, total, int64(30000))
	assert.False(t, component(m, "redis").Truncated)
	assert.Equal(t, int64(1), component(m, "redis").Lines)
	for _, name := range []string{"api", "worker"} {
		c := component(m, name)
		assert.True(t, c.Truncated, name)
		assert.Equal(t, CutBundle, c.Cut, name)
		assert.Greater(t, c.Lines, int64(40), "%s had a fair share", name)
	}
}

func TestEveryLineIsRedacted(t *testing.T) {
	t.Parallel()
	src := newSource()
	src.add("api", t0, "trenova-api-1", "api", "connecting with s3cr3t-db-pass now")
	src.add("api", t0.Add(time.Second), "trenova-api-1", "api", "dial postgres://app:hunter2hunter2@db:5432/tms")

	b, err := writeBundle(t.Context(), src, testSpec(t.TempDir()), DefaultCaps)
	require.NoError(t, err)
	files, _ := readBundle(t, b)

	assert.NotContains(t, files["api.log"], "s3cr3t-db-pass")
	assert.NotContains(t, files["api.log"], "hunter2hunter2")
	assert.Contains(t, files["api.log"], "connecting with [redacted] now")
	assert.Contains(t, files["api.log"], "postgres://[redacted]@db:5432/tms")
}

func TestAPeriodLongerThanTheCapIsRefused(t *testing.T) {
	t.Parallel()
	spec := testSpec(t.TempDir())
	spec.until = spec.since.Add(8 * 24 * time.Hour)

	_, err := writeBundle(t.Context(), newSource(), spec, DefaultCaps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "longer than 7 days")
}

func TestLokiFailingPartwayLeavesNoScratchFile(t *testing.T) {
	t.Parallel()
	src := newSource()
	for i := range 30 {
		src.add("api", t0.Add(time.Duration(i)*time.Second), "trenova-api-1", "api", fmt.Sprintf("line %d", i))
	}
	src.failAt = 3
	spec := testSpec(t.TempDir())

	_, err := writeBundle(t.Context(), src, spec, DefaultCaps)

	require.ErrorIs(t, err, errLokiDown)
	assert.Empty(t, scratchFiles(t, spec.dir))
}

func TestLinesReadBackwardsAcrossChunks(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "spill")
	want := []string{"first", strings.Repeat("y", reverseChunk+17), "third", strings.Repeat("z", 3*reverseChunk), "last"}
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(want, "\n")+"\n"), 0o600))
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	info, err := f.Stat()
	require.NoError(t, err)

	var got []string
	require.NoError(t, reverseLines(f, info.Size(), func(line []byte) error {
		got = append(got, string(line))
		return nil
	}))

	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[len(want)-1-i], got[i])
	}
}
