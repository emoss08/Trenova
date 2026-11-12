package commands

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	ns      = "trenova"
	release = "trenova"
	chart   = "trenova"
)

func labelsFor(component string) map[string]string {
	return map[string]string{labelName: chart, labelInstance: release, labelComponent: component}
}

func deployment(component string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: "trenova-" + component, Namespace: ns, Labels: labelsFor(component),
	}}
}

func statefulSet(component string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name: "trenova-" + component, Namespace: ns, Labels: labelsFor(component),
	}}
}

type fakeControlPlane struct {
	commands []instanceapi.Command
	answers  map[uuid.UUID][]instanceapi.CommandResult
	failOnce bool
}

func (f *fakeControlPlane) Commands(context.Context) ([]instanceapi.Command, error) {
	return f.commands, nil
}

func (f *fakeControlPlane) CompleteCommand(_ context.Context, id uuid.UUID, r instanceapi.CommandResult) error {
	if f.failOnce {
		f.failOnce = false
		return assert.AnError
	}
	if f.answers == nil {
		f.answers = map[uuid.UUID][]instanceapi.CommandResult{}
	}
	f.answers[id] = append(f.answers[id], r)
	return nil
}

func runner(t *testing.T, objects ...any) (*Runner, *fake.Clientset) {
	t.Helper()
	kube := fake.NewClientset()
	for _, o := range objects {
		var err error
		switch o := o.(type) {
		case *appsv1.Deployment:
			_, err = kube.AppsV1().Deployments(ns).Create(t.Context(), o, metav1.CreateOptions{})
		case *appsv1.StatefulSet:
			_, err = kube.AppsV1().StatefulSets(ns).Create(t.Context(), o, metav1.CreateOptions{})
		case *corev1.Secret:
			_, err = kube.CoreV1().Secrets(ns).Create(t.Context(), o, metav1.CreateOptions{})
		case *corev1.Pod:
			_, err = kube.CoreV1().Pods(ns).Create(t.Context(), o, metav1.CreateOptions{})
		}
		require.NoError(t, err)
	}
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	return New(&Params{
		Kube: kube, Namespace: ns, Release: release, Chart: chart,
		Logger: slog.New(slog.DiscardHandler), Now: func() time.Time { return now },
	}), kube
}

func restartedAtOf(t *testing.T, kube *fake.Clientset, name string) string {
	t.Helper()
	d, err := kube.AppsV1().Deployments(ns).Get(t.Context(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return d.Spec.Template.Annotations[restartedAt]
}

func restart(component string) instanceapi.Command {
	cmd := instanceapi.Command{Id: uuid.New(), Kind: instanceapi.Restart, IssuedAt: time.Now()}
	if component != "" {
		cmd.Component = &component
	}
	return cmd
}

func TestRestartsOneComponentAndLeavesTheRest(t *testing.T) {
	t.Parallel()
	r, kube := runner(t, deployment("api"), deployment("worker"))
	cmd := restart("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}

	require.NoError(t, r.Run(t.Context(), cp))

	assert.Equal(t, "2026-11-12T09:00:00Z", restartedAtOf(t, kube, "trenova-api"))
	assert.Empty(t, restartedAtOf(t, kube, "trenova-worker"))
	require.Len(t, cp.answers[cmd.Id], 1)
	assert.Equal(t, instanceapi.Succeeded, cp.answers[cmd.Id][0].Status)
}

func TestRestartsEveryWorkloadWhenNoComponentIsNamed(t *testing.T) {
	t.Parallel()
	r, kube := runner(t, deployment("api"), deployment("worker"), statefulSet("redis"))
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}

	require.NoError(t, r.Run(t.Context(), cp))

	assert.NotEmpty(t, restartedAtOf(t, kube, "trenova-api"))
	assert.NotEmpty(t, restartedAtOf(t, kube, "trenova-worker"))
	set, err := kube.AppsV1().StatefulSets(ns).Get(t.Context(), "trenova-redis", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, set.Spec.Template.Annotations[restartedAt])
	assert.Equal(t, "restarted 3 workload(s)", *cp.answers[cmd.Id][0].Message)
}

func TestRefusesToRestartTheDatabaseOrAnUnknownComponent(t *testing.T) {
	t.Parallel()
	r, _ := runner(t, deployment("api"))
	database, unknown := restart("postgres"), restart("nothing")
	cp := &fakeControlPlane{commands: []instanceapi.Command{database, unknown}}

	require.NoError(t, r.Run(t.Context(), cp))

	assert.Equal(t, instanceapi.Failed, cp.answers[database.Id][0].Status)
	assert.Contains(t, *cp.answers[database.Id][0].Message, "database")
	assert.Equal(t, instanceapi.Failed, cp.answers[unknown.Id][0].Status)
}

// A command listed again was not answered yet, not issued again: it is
// answered with the result it already has, and its work is not repeated.
func TestCarriesACommandOutOnceEvenWhenItsAnswerIsLost(t *testing.T) {
	t.Parallel()
	r, kube := runner(t, deployment("api"))
	cmd := restart("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}, failOnce: true}

	require.Error(t, r.Run(t.Context(), cp))
	first := restartedAtOf(t, kube, "trenova-api")
	require.NotEmpty(t, first)

	// The clock moves on; a second restart would write a later time.
	later := time.Date(2026, 11, 12, 9, 5, 0, 0, time.UTC)
	r.now = func() time.Time { return later }
	require.NoError(t, r.Run(t.Context(), cp))

	assert.Equal(t, first, restartedAtOf(t, kube, "trenova-api"))
	require.Len(t, cp.answers[cmd.Id], 1)
	assert.Equal(t, instanceapi.Succeeded, cp.answers[cmd.Id][0].Status)
}

func TestCollectedLogsNameTheirPodAndNeedAComponent(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-api-1", Namespace: ns, Labels: labelsFor("api")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
	}
	r, _ := runner(t, pod)
	component := "api"
	withComponent := instanceapi.Command{Id: uuid.New(), Kind: instanceapi.CollectLogs, Component: &component}
	without := instanceapi.Command{Id: uuid.New(), Kind: instanceapi.CollectLogs}
	cp := &fakeControlPlane{commands: []instanceapi.Command{withComponent, without}}

	require.NoError(t, r.Run(t.Context(), cp))

	got := cp.answers[withComponent.Id][0]
	require.Equal(t, instanceapi.Succeeded, got.Status)
	require.NotNil(t, got.Output)
	// The fake clientset's log is one line, "fake logs".
	assert.Equal(t, "trenova-api-1/api: fake logs\n", *got.Output)
	assert.Equal(t, instanceapi.Failed, cp.answers[without.Id][0].Status)
}

func TestNoLineLeavesWithASecretInIt(t *testing.T) {
	t.Parallel()
	r, _ := runner(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-app", Namespace: ns},
		Data: map[string][]byte{
			"DATABASE_PASSWORD": []byte("s3cr3t-db-pass"),
			"tls.key":           []byte("-----BEGIN KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC\n-----END KEY-----"),
			"short":             []byte("on"),
		},
	})
	redact, err := r.redactor(t.Context())
	require.NoError(t, err)

	tests := []struct{ name, in, want string }{
		{"a secret's value", "connecting with s3cr3t-db-pass now", "connecting with [redacted] now"},
		{"its base64", "env=czNjcjN0LWRiLXBhc3M=", "env=[redacted]"},
		{"one line of a key", "key MIIEvQIBADANBgkqhkiG9w0BAQEFAASC loaded", "key [redacted] loaded"},
		{"a short value is left: it is an ordinary word", "feature on", "feature on"},
		{
			"credentials in a URL the namespace does not hold",
			"dial postgres://app:hunter2hunter2@db:5432/tms failed",
			"dial postgres://[redacted]@db:5432/tms failed",
		},
		// The token and the header value both go: removing too much of a
		// credential line is the safe way to be wrong.
		{"a bearer token", "Authorization: Bearer abcdefghijklmnop", "Authorization: [redacted] [redacted]"},
		{"an assigned secret", `{"apiKey":"sk_live_12345","user":"dana"}`, `{"apiKey":[redacted],"user":"dana"}`},
		{"an ordinary line", "listening on :8080", "listening on :8080"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, redact(tt.in), tt.name)
	}
}
