package health

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// A component that is not ready says why when its pods can tell: a container
// that cannot start or keeps failing, a pod that cannot be scheduled, or the
// latest warning about the pod or its volume. Without this the control plane
// sees only counts and CloudNativePG's phase ("Setting up primary"), and an
// instance that never comes up cannot be diagnosed from outside, because its
// VM accepts no connections.

const (
	labelCNPGCluster = "cnpg.io/cluster"
	labelJobName     = "job-name"
)

// Waiting reasons that only say a container has not started yet; the cause,
// if any, is in the pod's events.
var startingReasons = []string{"", "ContainerCreating", "PodInitializing"}

// observed is what one check reads to explain components that are not ready.
type observed struct {
	pods     []corev1.Pod
	claims   []corev1.PersistentVolumeClaim
	warnings map[string]corev1.Event // latest Warning per "Kind/name"
}

// observe reads the namespace's pods, claims and warnings. A failed read
// leaves that part empty: status reports go out without causes rather than
// not at all.
func (c *Checker) observe(ctx context.Context) observed {
	o := observed{warnings: map[string]corev1.Event{}}
	if pods, err := c.kube.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{}); err == nil {
		o.pods = pods.Items
	}
	if claims, err := c.kube.CoreV1().PersistentVolumeClaims(c.namespace).
		List(ctx, metav1.ListOptions{}); err == nil {
		o.claims = claims.Items
	}
	events, err := c.kube.CoreV1().Events(c.namespace).
		List(ctx, metav1.ListOptions{FieldSelector: "type=" + corev1.EventTypeWarning})
	if err != nil {
		return o
	}
	for i := range events.Items {
		e := events.Items[i]
		if e.Type != corev1.EventTypeWarning {
			continue
		}
		key := e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name
		if prev, ok := o.warnings[key]; !ok || eventTime(&e).After(eventTime(&prev)) {
			o.warnings[key] = e
		}
	}
	return o
}

// cause explains why the pods a selector picks are not ready, or returns ""
// when nothing more specific than the component's own message is known.
func (o *observed) cause(sel labels.Selector) string {
	if sel == nil || sel.Empty() {
		return ""
	}
	pods := []*corev1.Pod{}
	for i := range o.pods {
		if sel.Matches(labels.Set(o.pods[i].Labels)) && !podReady(&o.pods[i]) {
			pods = append(pods, &o.pods[i])
		}
	}
	// The newest pod is the one being retried; older ones are often stale.
	slices.SortFunc(pods, func(a, b *corev1.Pod) int {
		return b.CreationTimestamp.Compare(a.CreationTimestamp.Time)
	})
	for _, p := range pods {
		if why := o.podCause(p); why != "" {
			return "pod " + p.Name + ": " + why
		}
	}
	for i := range o.claims {
		pvc := &o.claims[i]
		if !sel.Matches(labels.Set(pvc.Labels)) || pvc.Status.Phase != corev1.ClaimPending {
			continue
		}
		why := "pending"
		if e, ok := o.warnings["PersistentVolumeClaim/"+pvc.Name]; ok {
			why = e.Reason + ": " + firstLine(e.Message)
		}
		return "volume claim " + pvc.Name + ": " + why
	}
	return ""
}

func (o *observed) podCause(p *corev1.Pod) string {
	statuses := slices.Concat(p.Status.InitContainerStatuses, p.Status.ContainerStatuses)
	for i := range statuses {
		s := &statuses[i]
		if w := s.State.Waiting; w != nil && !slices.Contains(startingReasons, w.Reason) {
			return joinReason(s.Name, w.Reason, w.Message)
		}
	}
	for i := range statuses {
		s := &statuses[i]
		if t := s.State.Terminated; t != nil && t.ExitCode != 0 {
			why := fmt.Sprintf("exited %d (%s)", t.ExitCode, t.Reason)
			return joinReason(s.Name, why, t.Message)
		}
		if t := s.LastTerminationState.Terminated; t != nil && t.ExitCode != 0 && !s.Ready {
			why := fmt.Sprintf("last exited %d (%s)", t.ExitCode, t.Reason)
			return joinReason(s.Name, why, t.Message)
		}
	}
	if p.Status.Phase == corev1.PodFailed {
		return strings.TrimSpace(p.Status.Reason + ": " + firstLine(p.Status.Message))
	}
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
			return cond.Reason + ": " + firstLine(cond.Message)
		}
	}
	if e, ok := o.warnings["Pod/"+p.Name]; ok {
		return e.Reason + ": " + firstLine(e.Message)
	}
	return ""
}

// explain puts the cause first, so the 200-character limit cuts the counts,
// not the reason.
func explain(comp Component, cause string) Component {
	if comp.Ready || cause == "" {
		return comp
	}
	if comp.Message == "" {
		comp.Message = trim(cause)
	} else {
		comp.Message = trim(cause + "; " + comp.Message)
	}
	return comp
}

func podReady(p *corev1.Pod) bool {
	if p.Status.Phase == corev1.PodSucceeded {
		return true
	}
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func joinReason(container, reason, message string) string {
	out := container + " " + reason
	if m := firstLine(message); m != "" {
		out += ": " + m
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

func eventTime(e *corev1.Event) time.Time {
	switch {
	case !e.EventTime.IsZero():
		return e.EventTime.Time
	case !e.LastTimestamp.IsZero():
		return e.LastTimestamp.Time
	default:
		return e.CreationTimestamp.Time
	}
}
