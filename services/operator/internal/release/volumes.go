package release

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	instanceLabel = "app.kubernetes.io/instance"
	volumePoll    = time.Second
)

func (r *Releaser) dropVolumes(ctx context.Context) error {
	claims := r.kube.CoreV1().PersistentVolumeClaims(r.namespace)
	selector := metav1.ListOptions{LabelSelector: instanceLabel + "=" + r.name}
	left, err := claims.List(ctx, selector)
	if err != nil {
		return fmt.Errorf("list the volumes of the undeployed release %s: %w", r.name, err)
	}
	if len(left.Items) == 0 {
		return nil
	}
	for i := range left.Items {
		name := left.Items[i].Name
		err = claims.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf(
				"remove the volume %s of the undeployed release %s: %w",
				name,
				r.name,
				err,
			)
		}
		r.logger.Warn(
			"removed a volume of a release that never deployed",
			"release",
			r.name,
			"volume",
			name,
		)
	}
	err = wait.PollUntilContextTimeout(
		ctx,
		volumePoll,
		actionTimeout,
		true,
		func(ctx context.Context) (bool, error) {
			remaining, listErr := claims.List(ctx, selector)
			if listErr != nil {
				return false, listErr
			}
			return len(remaining.Items) == 0, nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"wait for the volumes of the undeployed release %s to go: %w",
			r.name,
			err,
		)
	}
	return nil
}
