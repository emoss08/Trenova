package dirstore_test

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/dirstore"
	"github.com/emoss08/trenova/operator/internal/securefile"
	"github.com/emoss08/trenova/operator/internal/tms"
)

func enrolled(t *testing.T, lifetime time.Duration) (dirstore.Store, *credential.Manager, string) {
	t.Helper()
	cp := controlplanetest.NewShared(t, uuid.New())
	cp.SetLifetime(lifetime)
	dir := filepath.Join(t.TempDir(), "identity")
	enrolment := filepath.Join(t.TempDir(), "enrolment.json")
	body, err := sonic.Marshal(cp.Bundle(7 * 24 * time.Hour))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(enrolment, body, 0o600))

	store, err := dirstore.New(dirstore.Params{Dir: dir, Enrolment: enrolment})
	require.NoError(t, err)
	transport := &credential.Transport{}
	lazy := controlplane.NewLazy(&http.Client{Transport: transport, Timeout: 10 * time.Second})
	creds := credential.NewManager(&credential.ManagerParams{
		Store:     store,
		Transport: transport,
		Renewer:   lazy,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	lazy.Locate(func() string { return creds.Current().APIURL })
	require.NoError(t, creds.Start(t.Context()))
	return store, creds, dir
}

func TestTheIdentityIsKeptPrivatelyAndReadBack(t *testing.T) {
	t.Parallel()
	store, creds, dir := enrolled(t, 30*24*time.Hour)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	info, err = os.Stat(filepath.Join(dir, dirstore.IdentityFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	loaded, ok, err := store.Load(t.Context())
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, credential.KindSharedServer, loaded.Kind)
	assert.Equal(t, current(creds).Fingerprint(), loaded.Fingerprint())
	assert.Equal(t, creds.Current().InstanceID, loaded.InstanceID)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary file is left behind")
}

func TestAnIdentityOthersCanReadIsRefused(t *testing.T) {
	t.Parallel()
	store, _, dir := enrolled(t, 30*24*time.Hour)
	require.NoError(t, os.Chmod(filepath.Join(dir, dirstore.IdentityFile), 0o644))

	_, _, err := store.Load(t.Context())
	require.ErrorIs(t, err, securefile.ErrInsecure)
}

func TestARenewedCertificateReplacesTheStoredOne(t *testing.T) {
	t.Parallel()
	store, creds, _ := enrolled(t, time.Minute)
	before := current(creds).Fingerprint()

	require.NoError(t, creds.RenewIfDue(t.Context()))

	loaded, ok, err := store.Load(t.Context())
	require.NoError(t, err)
	require.True(t, ok)
	assert.NotEqual(t, before, loaded.Fingerprint())
	assert.Equal(t, current(creds).Fingerprint(), loaded.Fingerprint())
}

func TestNoBundleAndNoIdentityIsNotEnrolled(t *testing.T) {
	t.Parallel()
	store, err := dirstore.New(dirstore.Params{Dir: filepath.Join(t.TempDir(), "identity")})
	require.NoError(t, err)

	_, ok, err := store.Bundle(t.Context())
	require.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = store.Load(t.Context())
	require.NoError(t, err)
	assert.False(t, ok)
}

func current(m *credential.Manager) *credential.Identity {
	ident := m.Current()
	return &ident
}

func TestABundleOthersCanWriteIsRefused(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "identity")
	enrolment := filepath.Join(t.TempDir(), "enrolment.json")
	require.NoError(t, os.WriteFile(enrolment, []byte("{}"), 0o600))
	require.NoError(t, os.Chmod(enrolment, 0o666))
	store, err := dirstore.New(dirstore.Params{Dir: dir, Enrolment: enrolment})
	require.NoError(t, err)

	_, _, err = store.Bundle(t.Context())
	require.ErrorIs(t, err, securefile.ErrInsecure, "anyone could swap the server's identity")
}

func TestABundleThatIsALinkIsRefused(t *testing.T) {
	t.Parallel()
	target := filepath.Join(t.TempDir(), "enrolment.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o444))
	link := filepath.Join(t.TempDir(), "enrolment.json")
	require.NoError(t, os.Symlink(target, link))
	store, err := dirstore.New(dirstore.Params{Dir: filepath.Join(t.TempDir(), "identity"), Enrolment: link})
	require.NoError(t, err)

	_, _, err = store.Bundle(t.Context())
	require.ErrorIs(t, err, securefile.ErrInsecure)
}

func TestTheDocumentedSharedServerShapeStartsAndRestarts(t *testing.T) {
	t.Parallel()
	cp := controlplanetest.NewShared(t, uuid.New())
	cp.SetLifetime(30 * 24 * time.Hour)
	secrets := filepath.Join(t.TempDir(), "secrets")
	require.NoError(t, os.Mkdir(secrets, 0o700))
	enrolment := filepath.Join(secrets, "agent_enrolment")
	body, err := sonic.Marshal(cp.Bundle(7 * 24 * time.Hour))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(enrolment, body, 0o444))
	require.NoError(t, os.Chmod(enrolment, 0o444))
	tokenFile := filepath.Join(secrets, "managed_token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("token\n"), 0o444))
	require.NoError(t, os.Chmod(tokenFile, 0o444))
	stateDir := filepath.Join(t.TempDir(), "trenova-tenant-agent")
	require.NoError(t, os.Mkdir(stateDir, 0o700))

	start := func() *credential.Manager {
		store, serr := dirstore.New(dirstore.Params{
			Dir:       filepath.Join(stateDir, "identity"),
			Enrolment: enrolment,
		})
		require.NoError(t, serr)
		transport := &credential.Transport{}
		lazy := controlplane.NewLazy(&http.Client{Transport: transport, Timeout: 10 * time.Second})
		creds := credential.NewManager(&credential.ManagerParams{
			Store:     store,
			Transport: transport,
			Renewer:   lazy,
			Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		lazy.Locate(func() string { return creds.Current().APIURL })
		require.NoError(t, creds.Start(t.Context()))
		return creds
	}
	first := start()
	second := start()
	assert.Equal(t, current(first).Fingerprint(), current(second).Fingerprint(),
		"a restart with the bundle still mounted keeps the identity it enrolled")

	info, err := os.Stat(filepath.Join(stateDir, "identity", dirstore.IdentityFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	token, err := tms.NewFileTokens(tokenFile).Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "token", token)
}

func TestAnIdentityThatIsALinkIsRefused(t *testing.T) {
	t.Parallel()
	store, _, dir := enrolled(t, 30*24*time.Hour)
	path := filepath.Join(dir, dirstore.IdentityFile)
	elsewhere := filepath.Join(t.TempDir(), "swapped.json")
	require.NoError(t, os.Rename(path, elsewhere))
	require.NoError(t, os.Symlink(elsewhere, path))

	_, _, err := store.Load(t.Context())
	require.ErrorIs(t, err, securefile.ErrInsecure, "a link is followed nowhere")
}

func TestACredentialDirectoryThatIsALinkIsRefused(t *testing.T) {
	t.Parallel()
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "identity")
	require.NoError(t, os.Symlink(target, link))

	_, err := dirstore.New(dirstore.Params{Dir: link})
	require.ErrorIs(t, err, securefile.ErrInsecure)
}

func TestATruncatedIdentityIsReportedInvalidSoTheBundleEnrolsAgain(t *testing.T) {
	t.Parallel()
	store, _, dir := enrolled(t, 30*24*time.Hour)
	require.NoError(t, os.WriteFile(filepath.Join(dir, dirstore.IdentityFile), []byte(`{"kind":`), 0o600))

	_, ok, err := store.Load(t.Context())
	require.ErrorIs(t, err, credential.ErrIdentityInvalid)
	assert.False(t, ok)
}
