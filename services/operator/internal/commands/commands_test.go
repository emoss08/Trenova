package commands

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	commands    []instanceapi.Command
	emailEvents bool
	listErr     error
	answers     map[uuid.UUID][]instanceapi.CommandResult
	failOnce    bool
}

func (f *fakeControlPlane) Commands(context.Context) (instanceapi.CommandList, error) {
	if f.listErr != nil {
		return instanceapi.CommandList{}, f.listErr
	}
	return instanceapi.CommandList{Commands: f.commands, EmailEvents: f.emailEvents}, nil
}

func (f *fakeControlPlane) CompleteCommand(
	_ context.Context,
	id uuid.UUID,
	r instanceapi.CommandResult,
) error {
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

func runner(t *testing.T, cp ControlPlane, objects ...any) (*Runner, *fake.Clientset) {
	t.Helper()
	return runnerWith(t, cp, &fakeGrants{}, &fakeLogs{}, objects...)
}

func runnerWith(
	t *testing.T,
	cp ControlPlane,
	grants Grants,
	kept Logs,
	objects ...any,
) (*Runner, *fake.Clientset) {
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
	r, err := New(&Params{
		ControlPlane: cp, Kube: kube, Grants: grants, Logs: kept, Namespace: ns, Release: release,
		Chart: chart, Logger: slog.New(slog.DiscardHandler), Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	return r, kube
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

func collect(component string) instanceapi.Command {
	lines := 200
	return instanceapi.Command{
		Id: uuid.New(), Kind: instanceapi.CollectLogs, Component: &component, Lines: &lines,
		IssuedAt: time.Now(), RequestedBy: customer(),
	}
}

func customer() *instanceapi.RequestedBy {
	return &instanceapi.RequestedBy{Kind: instanceapi.RequestedByCustomer, Email: "dana@acme.example"}
}

func staff(email string) *instanceapi.RequestedBy {
	return &instanceapi.RequestedBy{Kind: instanceapi.RequestedByStaff, Email: openapi_types.Email(email)}
}

func TestARunnerWithoutItsControlPlaneIsRefused(t *testing.T) {
	t.Parallel()
	_, err := New(&Params{
		Kube: fake.NewClientset(), Namespace: ns, Release: release, Chart: chart,
		Logger: slog.New(slog.DiscardHandler),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the control plane")
}

func TestTheRunnerSaysWhetherSystemEmailEventsWait(t *testing.T) {
	t.Parallel()
	cp := &fakeControlPlane{}
	r, _ := runner(t, cp)
	assert.False(t, r.EmailEventsWaiting(), "nothing is known before the first listing")

	cp.emailEvents = true
	require.NoError(t, r.Run(t.Context()))
	assert.True(t, r.EmailEventsWaiting())

	cp.listErr = assert.AnError
	require.Error(t, r.Run(t.Context()))
	assert.False(t, r.EmailEventsWaiting(), "a failed listing says nothing waits")

	cp.listErr, cp.emailEvents = nil, false
	require.NoError(t, r.Run(t.Context()))
	assert.False(t, r.EmailEventsWaiting())
}

func TestRestartsOneComponentAndLeavesTheRest(t *testing.T) {
	t.Parallel()
	cmd := restart("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, deployment("api"), deployment("worker"))

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, "2026-11-12T09:00:00Z", restartedAtOf(t, kube, "trenova-api"))
	assert.Empty(t, restartedAtOf(t, kube, "trenova-worker"))
	require.Len(t, cp.answers[cmd.Id], 1)
	assert.Equal(t, instanceapi.CommandResultStatusSucceeded, cp.answers[cmd.Id][0].Status)
}

func TestRestartsEveryWorkloadWhenNoComponentIsNamed(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, deployment("api"), deployment("worker"), statefulSet("redis"))

	require.NoError(t, r.Run(t.Context()))

	assert.NotEmpty(t, restartedAtOf(t, kube, "trenova-api"))
	assert.NotEmpty(t, restartedAtOf(t, kube, "trenova-worker"))
	set, err := kube.AppsV1().StatefulSets(ns).Get(t.Context(), "trenova-redis", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, set.Spec.Template.Annotations[restartedAt])
	assert.Equal(t, "restarted 3 workload(s)", *cp.answers[cmd.Id][0].Message)
}

func TestRestartsAStatefulComponentByName(t *testing.T) {
	t.Parallel()
	cmd := restart("redis")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, deployment("api"), statefulSet("redis"))

	require.NoError(t, r.Run(t.Context()))

	set, err := kube.AppsV1().StatefulSets(ns).Get(t.Context(), "trenova-redis", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, set.Spec.Template.Annotations[restartedAt])
	assert.Empty(t, restartedAtOf(t, kube, "trenova-api"))
	assert.Equal(t, "restarted 1 workload(s)", *cp.answers[cmd.Id][0].Message)
}

func TestRefusesToRestartTheDatabaseOrAnUnknownComponent(t *testing.T) {
	t.Parallel()
	database, unknown := restart("postgres"), restart("nothing")
	cp := &fakeControlPlane{commands: []instanceapi.Command{database, unknown}}
	r, _ := runner(t, cp, deployment("api"))

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[database.Id][0].Status)
	assert.Contains(t, *cp.answers[database.Id][0].Message, "database")
	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[unknown.Id][0].Status)
	assert.Contains(t, *cp.answers[unknown.Id][0].Message, "nothing")
}

func TestCarriesACommandOutOnceEvenWhenItsAnswerIsLost(t *testing.T) {
	t.Parallel()
	cmd := restart("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}, failOnce: true}
	r, kube := runner(t, cp, deployment("api"))

	require.Error(t, r.Run(t.Context()))
	first := restartedAtOf(t, kube, "trenova-api")
	require.NotEmpty(t, first)

	later := time.Date(2026, 11, 12, 9, 5, 0, 0, time.UTC)
	r.now = func() time.Time { return later }
	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t, first, restartedAtOf(t, kube, "trenova-api"))
	require.Len(t, cp.answers[cmd.Id], 1)
	assert.Equal(t, instanceapi.CommandResultStatusSucceeded, cp.answers[cmd.Id][0].Status)
}

func TestCollectedLogsNameTheirPodAndNeedAComponent(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-api-1", Namespace: ns, Labels: labelsFor("api")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
	}
	withComponent := collect("api")
	without := instanceapi.Command{Id: uuid.New(), Kind: instanceapi.CollectLogs}
	cp := &fakeControlPlane{commands: []instanceapi.Command{withComponent, without}}
	r, _ := runner(t, cp, pod)

	require.NoError(t, r.Run(t.Context()))

	got := cp.answers[withComponent.Id][0]
	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status)
	require.NotNil(t, got.Output)
	assert.Equal(t, "trenova-api-1/api: fake logs\n", *got.Output)
	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[without.Id][0].Status)
}

func TestCollectsEveryComponentTheReportNames(t *testing.T) {
	t.Parallel()
	releasePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "trenova-release-4-x7k2p", Namespace: ns, Labels: labelsFor("release"),
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "prepare"}, {Name: "migrate"}},
			Containers:     []corev1.Container{{Name: "done"}},
		},
	}
	postgres := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-postgres-1", Namespace: ns, Labels: labelsFor("postgres")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "postgres"}}},
	}
	redis := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-redis-0", Namespace: ns, Labels: labelsFor("redis")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "redis"}}},
	}
	cmds := []instanceapi.Command{collect("release"), collect("postgres"), collect("redis")}
	cp := &fakeControlPlane{commands: cmds}
	r, _ := runner(t, cp, releasePod, postgres, redis)

	require.NoError(t, r.Run(t.Context()))

	want := []string{
		"trenova-release-4-x7k2p/prepare: fake logs\n" +
			"trenova-release-4-x7k2p/migrate: fake logs\n" +
			"trenova-release-4-x7k2p/done: fake logs\n",
		"trenova-postgres-1/postgres: fake logs\n",
		"trenova-redis-0/redis: fake logs\n",
	}
	for i, cmd := range cmds {
		got := cp.answers[cmd.Id][0]
		require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status, *cmd.Component)
		assert.Equal(t, want[i], *got.Output, *cmd.Component)
	}
}

func TestACrashingContainersPreviousLogComesFirst(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-api-1", Namespace: ns, Labels: labelsFor("api")},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name:         "api",
			RestartCount: 4,
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
			},
			LastTerminationState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{ExitCode: 1},
			},
		}}},
	}
	cmd := collect("api")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, _ := runner(t, cp, pod)

	require.NoError(t, r.Run(t.Context()))

	assert.Equal(t,
		"trenova-api-1/api (previous): fake logs\ntrenova-api-1/api: fake logs\n",
		*cp.answers[cmd.Id][0].Output,
	)
}

func TestKeepsTheNewestPodsWhenAComponentHasMany(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	var objects []any
	for i := range maxPods + 2 {
		objects = append(objects, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "trenova-worker-" + string(rune('a'+i)),
				Namespace:         ns,
				Labels:            labelsFor("worker"),
				CreationTimestamp: metav1.NewTime(base.Add(time.Duration(i) * time.Minute)),
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker"}}},
		})
	}
	cmd := collect("worker")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, _ := runner(t, cp, objects...)

	require.NoError(t, r.Run(t.Context()))

	out := *cp.answers[cmd.Id][0].Output
	assert.True(t, strings.HasPrefix(out, "trenova-worker-g/worker:"), out)
	assert.NotContains(t, out, "trenova-worker-a/")
	assert.NotContains(t, out, "trenova-worker-b/")
}

func TestAFailureMessageIsOneLineTheControlPlaneAccepts(t *testing.T) {
	t.Parallel()
	got := failed(assert.AnError)
	require.NotNil(t, got.Message)
	assert.NotContains(t, *got.Message, "\n")
	long := failed(&longError{})
	assert.LessOrEqual(t, len([]rune(*long.Message)), maxMessage)
}

type longError struct{}

func (*longError) Error() string { return strings.Repeat("é\n", 600) }

func TestNoLineLeavesWithASecretInIt(t *testing.T) {
	t.Parallel()
	r, _ := runner(t, &fakeControlPlane{}, &corev1.Secret{
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
		{"a bearer token", "Authorization: Bearer abcdefghijklmnop", "Authorization: [redacted] [redacted]"},
		{"an assigned secret", `{"apiKey":"sk_live_12345","user":"dana"}`, `{"apiKey":[redacted],"user":"dana"}`},
		{"an ordinary line", "listening on :8080", "listening on :8080"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, redact(tt.in), tt.name)
	}
}
