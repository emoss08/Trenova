package release_test

import (
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	ggcrregistry "github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/registry"

	"github.com/emoss08/trenova/operator/internal/release"
)

var errNotTrusted = errors.New("not trusted")

type recordingProvenance struct {
	refuse    bool
	chart     []byte
	signature []byte
}

func (*recordingProvenance) Required() bool { return true }

func (p *recordingProvenance) Verify(chart, signature []byte) error {
	p.chart, p.signature = chart, signature
	if p.refuse {
		return errNotTrusted
	}
	return nil
}

type ociWorld struct {
	host   string
	client *registry.Client
}

func discardLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func newOCIWorld(t *testing.T) *ociWorld {
	t.Helper()
	srv := httptest.NewServer(ggcrregistry.New(ggcrregistry.Logger(discardLogger())))
	t.Cleanup(srv.Close)
	client, err := registry.NewClient(
		registry.ClientOptPlainHTTP(),
		registry.ClientOptWriter(io.Discard),
	)
	require.NoError(t, err)
	return &ociWorld{host: strings.TrimPrefix(srv.URL, "http://"), client: client}
}

func (w *ociWorld) source(p release.Provenance) *release.Source {
	return release.NewSource(release.SourceParams{
		Ref:        "oci://" + w.host + "/charts/trenova",
		Registry:   w.client,
		Provenance: p,
	})
}

func (w *ociWorld) push(t *testing.T, version, tag string, signature []byte) []byte {
	t.Helper()
	archive := packaged(t, "trenova", version)
	opts := []registry.PushOption{registry.PushOptStrictMode(false)}
	if signature != nil {
		opts = append(opts, registry.PushOptProvData(signature))
	}
	_, err := w.client.Push(archive, w.host+"/charts/trenova:"+tag, opts...)
	require.NoError(t, err)
	return archive
}

func packaged(t *testing.T, name, version string) []byte {
	t.Helper()
	path, err := chartutil.Save(&chart.Chart{Metadata: &chart.Metadata{
		APIVersion: chart.APIVersionV2,
		Name:       name,
		Version:    version,
	}}, t.TempDir())
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	return body
}

func TestHandsTheProvenanceLayerToTheVerifier(t *testing.T) {
	t.Parallel()
	w := newOCIWorld(t)
	archive := w.push(t, "1.2.3", "1.2.3", []byte(`{"bundle":"the signature"}`))
	verifier := &recordingProvenance{}

	ch, err := w.source(verifier).Load("1.2.3")

	require.NoError(t, err)
	assert.Equal(t, "1.2.3", ch.Metadata.Version)
	assert.Equal(t, archive, verifier.chart)
	assert.JSONEq(t, `{"bundle":"the signature"}`, string(verifier.signature))
}

func TestRefusesToLoadAChartWithoutAProvenanceLayer(t *testing.T) {
	t.Parallel()
	w := newOCIWorld(t)
	w.push(t, "1.2.3", "1.2.3", nil)
	verifier := &recordingProvenance{}

	_, err := w.source(verifier).Load("1.2.3")

	require.Error(t, err)
	assert.Nil(t, verifier.chart)
}

func TestRefusesAChartTheVerifierRejects(t *testing.T) {
	t.Parallel()
	w := newOCIWorld(t)
	w.push(t, "1.2.3", "1.2.3", []byte("the signature"))
	source := w.source(&recordingProvenance{refuse: true})

	_, err := source.Load("1.2.3")
	require.ErrorIs(t, err, errNotTrusted)

	_, err = source.Load("1.2.3")
	require.ErrorIs(t, err, errNotTrusted, "a refused chart is never cached")
}

func TestRefusesAnOlderSignedReleaseMovedOntoTheTag(t *testing.T) {
	t.Parallel()
	w := newOCIWorld(t)
	w.push(t, "1.0.0", "1.2.3", []byte("the signature of 1.0.0"))

	_, err := w.source(&recordingProvenance{}).Load("1.2.3")

	require.ErrorIs(t, err, release.ErrChartIdentity)
}

func TestLoadsAnUnsignedChartOnlyWhenVerificationIsOff(t *testing.T) {
	t.Parallel()
	w := newOCIWorld(t)
	w.push(t, "1.2.3", "1.2.3", nil)

	ch, err := w.source(release.Unverified{}).Load("1.2.3")

	require.NoError(t, err)
	assert.Equal(t, "1.2.3", ch.Metadata.Version)
}

func rendered(t *testing.T, dir string, digests string, vals map[string]any) string {
	t.Helper()
	ch, err := loader.Load(dir)
	require.NoError(t, err)
	if digests != "" {
		ch.Files = append(ch.Files, &chart.File{Name: "image-digests.yaml", Data: []byte(digests)})
	}
	cfg, _, _ := setup(t)
	install := action.NewInstall(cfg)
	install.ReleaseName = "trenova"
	install.Namespace = "trenova"
	install.DryRun = true
	install.ClientOnly = true
	install.KubeVersion = &capabilities().KubeVersion
	rel, err := install.Run(ch, vals)
	require.NoError(t, err)
	return rel.Manifest
}

const pinnedRedis = "sha256:6169dabab9f4050073203cf18517e83848f0f475b066992ebf90d38089e7ed7c"

func TestAPublishedChartRunsItsImagesByDigest(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	repository, tag := redisImage(t, ch)
	digests := `"` + repository + ":" + tag + `": "` + pinnedRedis + `"` + "\n"

	manifest := rendered(t, chartDir, digests, vals("trenova.example.com"))

	assert.Contains(t, manifest, repository+":"+tag+"@"+pinnedRedis)
}

func TestAnOverriddenImageLosesThePublishedDigest(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	repository, tag := redisImage(t, ch)
	digests := `"` + repository + ":" + tag + `": "` + pinnedRedis + `"` + "\n"
	overridden := vals("trenova.example.com")
	overridden["redis"] = map[string]any{"image": map[string]any{"tag": "9"}}

	manifest := rendered(t, chartDir, digests, overridden)

	assert.Contains(t, manifest, repository+":9")
	assert.NotContains(t, manifest, pinnedRedis)
}

func TestAChartWithoutDigestsRunsItsImagesByTag(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	repository, tag := redisImage(t, ch)

	manifest := rendered(t, chartDir, "", vals("trenova.example.com"))

	assert.Contains(t, manifest, "image: "+repository+":"+tag+"\n")
}

func redisImage(t *testing.T, ch *chart.Chart) (repository, tag string) {
	t.Helper()
	redis, ok := ch.Values["redis"].(map[string]any)
	require.True(t, ok)
	image, ok := redis["image"].(map[string]any)
	require.True(t, ok)
	repository, _ = image["repository"].(string)
	tag, _ = image["tag"].(string)
	require.NotEmpty(t, repository)
	require.NotEmpty(t, tag)
	return repository, tag
}

