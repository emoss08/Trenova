package logs

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/redact"
	"github.com/emoss08/trenova/operator/internal/storage"
	"github.com/emoss08/trenova/operator/internal/tms"
)

var errLokiDown = errors.New("the log store answered 503")

type fakeSource struct {
	mu       sync.Mutex
	entries  map[string][]Entry
	calls    int
	failAt   int
	tooLarge int
	limits   []int
}

func newSource() *fakeSource {
	return &fakeSource{entries: map[string][]Entry{}}
}

func (f *fakeSource) add(component string, at time.Time, pod, container, line string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[component] = append(f.entries[component], Entry{
		Time: at.UTC(), Pod: pod, Container: container, Line: line, stream: pod + "/" + container,
	})
}

func (f *fakeSource) Components(context.Context, time.Time, time.Time) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	names := make([]string, 0, len(f.entries))
	for name := range f.entries {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func (f *fakeSource) NewestFirst(_ context.Context, q PageQuery) ([]Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.limits = append(f.limits, q.Limit)
	if f.failAt > 0 && f.calls >= f.failAt {
		return nil, errLokiDown
	}
	if f.tooLarge > 0 && q.Limit > f.tooLarge {
		return nil, ErrPageTooLarge
	}
	var out []Entry
	for _, e := range f.entries[q.Component] {
		if !e.Time.Before(q.Start) && e.Time.Before(q.End) {
			out = append(out, e)
		}
	}
	slices.SortStableFunc(out, newestFirst)
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

type stored struct {
	body     []byte
	metadata map[string]string
}

type fakeObjects struct {
	mu      sync.Mutex
	objects map[string]stored
	puts    int
	failPut int
	status  int
	none    bool
}

func newObjects() *fakeObjects {
	return &fakeObjects{objects: map[string]stored{}}
}

func (f *fakeObjects) Put(_ context.Context, obj *storage.Object) error {
	body, err := io.ReadAll(obj.Body)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts++
	if f.none {
		return storage.ErrNoDocumentsBucket
	}
	if f.failPut > 0 {
		f.failPut--
		return &storage.ObjectError{Status: f.status, Op: "upload"}
	}
	sum := sha256.Sum256(body)
	if int64(len(body)) != obj.Size || hex.EncodeToString(sum[:]) != obj.SHA256 {
		return errors.New("the upload's size or digest does not match its body")
	}
	f.objects[obj.Key] = stored{body: body, metadata: obj.Metadata}
	return nil
}

func (f *fakeObjects) Head(_ context.Context, key string) (storage.ObjectInfo, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.none {
		return storage.ObjectInfo{}, false, storage.ErrNoDocumentsBucket
	}
	o, ok := f.objects[key]
	if !ok {
		return storage.ObjectInfo{}, false, nil
	}
	return storage.ObjectInfo{Size: int64(len(o.body)), Metadata: o.metadata}, true, nil
}

func (f *fakeObjects) get(key string) (stored, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.objects[key]
	return o, ok
}

func (f *fakeObjects) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.puts
}

type fakeCP struct {
	mu            sync.Mutex
	byRequest     map[string]uuid.UUID
	registrations int
	updates       map[uuid.UUID][]instanceapi.LogBundleUpdate
}

func newCP() *fakeCP {
	return &fakeCP{byRequest: map[string]uuid.UUID{}, updates: map[uuid.UUID][]instanceapi.LogBundleUpdate{}}
}

func (f *fakeCP) RegisterLogBundle(_ context.Context, reg *instanceapi.LogBundleRegistration) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registrations++
	id, ok := f.byRequest[reg.RequestId]
	if !ok {
		id = uuid.New()
		f.byRequest[reg.RequestId] = id
	}
	return id, nil
}

func (f *fakeCP) UpdateLogBundle(_ context.Context, id uuid.UUID, u instanceapi.LogBundleUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates[id] = append(f.updates[id], u)
	return nil
}

func (f *fakeCP) statuses(id uuid.UUID) []instanceapi.LogBundleUpdateStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []instanceapi.LogBundleUpdateStatus
	for _, u := range f.updates[id] {
		out = append(out, u.Status)
	}
	return out
}

func (f *fakeCP) last(id uuid.UUID) instanceapi.LogBundleUpdate {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.updates[id]
	return list[len(list)-1]
}

type noTMS struct{}

func (noTMS) LogRequests(context.Context) ([]tms.LogRequest, error) { return nil, nil }

func (noTMS) UpdateLogRequest(context.Context, string, tms.LogRequestUpdate) error { return nil }

type secrets struct {
	values [][]byte
	err    error
}

func (s secrets) Load(context.Context) (*redact.Redactor, error) {
	if s.err != nil {
		return nil, s.err
	}
	return redact.New(s.values), nil
}

type world struct {
	src     *fakeSource
	objects *fakeObjects
	cp      *fakeCP
	dir     string
	svc     *Service
}

func newService(t *testing.T, src Source, objects Objects, cp ControlPlane, t2 TMS, caps Caps, dir string) *Service {
	t.Helper()
	svc, err := New(&Params{
		Source:       src,
		Objects:      objects,
		ControlPlane: cp,
		TMS:          t2,
		Redactors:    secrets{values: [][]byte{[]byte("s3cr3t-db-pass")}},
		InstanceID:   func() uuid.UUID { return uuid.MustParse("6f1c2c43-2f52-4a55-9f3b-6c1d1b3b9a10") },
		Dir:          dir,
		Caps:         caps,
		Logger:       slog.New(slog.DiscardHandler),
	})
	require.NoError(t, err)
	return svc
}

func newTestWorld(t *testing.T, caps Caps) *world {
	t.Helper()
	w := &world{src: newSource(), objects: newObjects(), cp: newCP(), dir: t.TempDir()}
	w.svc = newService(t, w.src, w.objects, w.cp, noTMS{}, caps, w.dir)
	return w
}

func (w *world) start(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = w.svc.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func scratchFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func unzip(t *testing.T, body []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		out[f.Name] = string(data)
	}
	return out
}
