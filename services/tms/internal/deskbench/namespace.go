package deskbench

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/connection"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	namespacepb "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	DefaultNamespace     = "deskbench"
	namespaceRetention   = time.Hour
	namespaceReadyWithin = 45 * time.Second
	namespacePollEvery   = 500 * time.Millisecond
)

var (
	ErrProductionEnvironment = errors.New(
		"the desk bench runs real turns and approves real writes; it refuses to run outside development",
	)
	ErrTemporalProfile = errors.New(
		"the desk bench needs a Temporal server it can add a namespace to; " +
			"it refuses a profile-configured (cloud) connection",
	)
	ErrDefaultNamespace = errors.New(
		"the desk bench must not share the default namespace with the dev worker; pick another one",
	)
)

func AssertSafe(cfg *config.Config, namespace string) error {
	if cfg.App.IsProduction() || cfg.App.IsStaging() {
		return ErrProductionEnvironment
	}
	if cfg.Temporal.UsesProfile() {
		return ErrTemporalProfile
	}
	if namespace == "" || namespace == cfg.Temporal.GetNamespace() {
		return ErrDefaultNamespace
	}

	return nil
}

func EnsureNamespace(ctx context.Context, cfg *config.TemporalConfig, namespace string) error {
	scoped := *cfg
	scoped.Namespace = namespace

	options, _, err := connection.BuildOptions(&scoped)
	if err != nil {
		return fmt.Errorf("configure temporal for the bench: %w", err)
	}

	namespaces, err := client.NewNamespaceClient(options)
	if err != nil {
		return fmt.Errorf("connect to temporal: %w", err)
	}
	defer namespaces.Close()

	if err = register(ctx, namespaces, namespace); err != nil {
		return err
	}

	workflows, err := client.DialContext(ctx, options)
	if err != nil {
		return fmt.Errorf("connect to temporal namespace %s: %w", namespace, err)
	}
	defer workflows.Close()

	return awaitNamespace(ctx, workflows, namespace)
}

func register(ctx context.Context, namespaces client.NamespaceClient, namespace string) error {
	described, err := namespaces.Describe(ctx, namespace)
	if err == nil {
		return keepRetentionShort(ctx, namespaces, namespace, described)
	}

	var missing *serviceerror.NamespaceNotFound
	if !errors.As(err, &missing) {
		return fmt.Errorf("describe temporal namespace %s: %w", namespace, err)
	}

	err = namespaces.Register(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace,
		Description:                      "Desk bench: turns driven by trenova desk, isolated from the dev worker",
		WorkflowExecutionRetentionPeriod: durationpb.New(namespaceRetention),
	})
	var exists *serviceerror.NamespaceAlreadyExists
	if err != nil && !errors.As(err, &exists) {
		return fmt.Errorf("register temporal namespace %s: %w", namespace, err)
	}

	return nil
}

// keepRetentionShort trims a namespace an earlier bench registered with a
// longer retention. The local dev server keeps every closed run in memory
// until retention lets it go, and three days of bench runs filled its memory
// limit until it stopped answering polls; a run's files hold all it needs.
func keepRetentionShort(
	ctx context.Context,
	namespaces client.NamespaceClient,
	namespace string,
	described *workflowservice.DescribeNamespaceResponse,
) error {
	current := described.GetConfig().GetWorkflowExecutionRetentionTtl()
	if current != nil && current.AsDuration() <= namespaceRetention {
		return nil
	}

	err := namespaces.Update(ctx, &workflowservice.UpdateNamespaceRequest{
		Namespace: namespace,
		Config: &namespacepb.NamespaceConfig{
			WorkflowExecutionRetentionTtl: durationpb.New(namespaceRetention),
		},
	})
	if err != nil {
		return fmt.Errorf("shorten temporal namespace %s retention: %w", namespace, err)
	}

	return nil
}

func awaitNamespace(ctx context.Context, workflows client.Client, namespace string) error {
	ctx, cancel := context.WithTimeout(ctx, namespaceReadyWithin)
	defer cancel()

	ticker := time.NewTicker(namespacePollEvery)
	defer ticker.Stop()

	for {
		_, err := workflows.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
			PageSize:  1,
		})
		if err == nil {
			return nil
		}

		var missing *serviceerror.NamespaceNotFound
		if !errors.As(err, &missing) {
			return fmt.Errorf("read temporal namespace %s: %w", namespace, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("temporal namespace %s was not ready in time: %w", namespace, err)
		case <-ticker.C:
		}
	}
}

const (
	namespaceLookupAttempts = 4
	namespaceLookupBackoff  = time.Second
)

func retryNamespaceLookup(ctx context.Context, start func() error) error {
	var err error
	for attempt := range namespaceLookupAttempts {
		if err = start(); err == nil {
			return nil
		}
		var missing *serviceerror.NamespaceNotFound
		if !errors.As(err, &missing) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(namespaceLookupBackoff * time.Duration(attempt+1)):
		}
	}

	return err
}
