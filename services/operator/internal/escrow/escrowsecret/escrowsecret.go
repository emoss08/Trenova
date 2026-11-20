package escrowsecret

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/escrow"
)

const (
	labelManagedBy = "app.kubernetes.io/managed-by"
	labelPartOf    = "app.kubernetes.io/part-of"
	labelComponent = "app.kubernetes.io/component"
	annotationNote = "trenova.app/escrow"
	managedBy      = "trenova-operator"
	note           = "the instance's keys held in escrow by the Trenova Cloud control plane; " +
		"restored from it when missing, never a Helm resource"
)

type Store struct {
	client    kubernetes.Interface
	cp        escrow.ControlPlane
	namespace string
	name      string
	logger    *slog.Logger
}

type Params struct {
	Client       kubernetes.Interface
	ControlPlane escrow.ControlPlane
	Namespace    string
	Name         string
	Logger       *slog.Logger
}

func New(p *Params) *Store {
	return &Store{
		client:    p.Client,
		cp:        p.ControlPlane,
		namespace: p.Namespace,
		name:      p.Name,
		logger:    p.Logger,
	}
}

func (s *Store) Ensure(ctx context.Context) error {
	secrets := s.client.CoreV1().Secrets(s.namespace)
	sec, err := secrets.Get(ctx, s.name, metav1.GetOptions{})
	missing := apierrors.IsNotFound(err)
	if err != nil && !missing {
		return fmt.Errorf("read the escrow Secret %s/%s: %w", s.namespace, s.name, err)
	}
	if !missing {
		problem := escrow.Validate(escrow.FromData(sec.Data))
		if problem == nil {
			return nil
		}
		s.logger.Warn(
			"the escrow Secret is incomplete or invalid; restoring it from the control plane",
			"secret", s.namespace+"/"+s.name,
			"reason", problem.Error(),
		)
	}
	fetched, err := s.cp.Secrets(ctx)
	if err != nil {
		return fmt.Errorf("fetch the escrowed secrets: %w", err)
	}
	if err = escrow.Validate(fetched); err != nil {
		return fmt.Errorf("the control plane's escrowed secrets: %w", err)
	}
	if missing {
		_, err = secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        s.name,
				Namespace:   s.namespace,
				Labels:      labels(nil),
				Annotations: annotations(nil),
			},
			Type: corev1.SecretTypeOpaque,
			Data: escrow.ToData(nil, fetched),
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create the escrow Secret %s/%s: %w", s.namespace, s.name, err)
		}
		s.logger.Info("escrow Secret created", "secret", s.namespace+"/"+s.name)
		return nil
	}
	next := sec.DeepCopy()
	next.Labels = labels(next.Labels)
	next.Annotations = annotations(next.Annotations)
	next.Data = escrow.ToData(next.Data, fetched)
	if _, err = secrets.Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update the escrow Secret %s/%s: %w", s.namespace, s.name, err)
	}
	s.logger.Info("escrow Secret restored", "secret", s.namespace+"/"+s.name)
	return nil
}

func labels(existing map[string]string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[labelManagedBy] = managedBy
	out[labelPartOf] = "trenova"
	out[labelComponent] = "escrow"
	return out
}

func annotations(existing map[string]string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[annotationNote] = note
	return out
}
