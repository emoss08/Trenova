package airetrievalrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pgvector"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ agentports.AIRetrievalRepository = (*repository)(nil)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Probe  *postgres.CapabilityProbe
	Logger *zap.Logger
}

type repository struct {
	db    *postgres.Connection
	probe *postgres.CapabilityProbe
	l     *zap.Logger
}

func New(p Params) agentports.AIRetrievalRepository {
	return &repository{
		db:    p.DB,
		probe: p.Probe,
		l:     p.Logger.Named("postgres.airetrieval-repository"),
	}
}

func (r *repository) VectorAvailability(ctx context.Context) (airetrieval.Availability, error) {
	vector, err := r.probe.VectorSupport(ctx)
	if err != nil {
		return airetrieval.Availability{}, fmt.Errorf("probe vector search: %w", err)
	}

	return availabilityFrom(vector), nil
}

func availabilityFrom(vector pgvector.VectorSupport) airetrieval.Availability {
	availability := airetrieval.Availability{
		ExtensionInstalled: vector.ExtensionInstalled(),
		ExtensionVersion:   vector.ExtensionVersion,
	}

	switch {
	case vector.Ready():
		availability.Available = true
	case vector.State == pgvector.VectorExtensionTooOld:
		availability.Reason = airetrieval.UnavailableReasonTooOld
	case vector.State == pgvector.VectorSchemaMissing:
		availability.Reason = airetrieval.UnavailableReasonSchemaMissing
	default:
		availability.Reason = airetrieval.UnavailableReasonExtensionMissing
	}

	return availability
}

func (r *repository) requireVector(ctx context.Context) error {
	availability, err := r.VectorAvailability(ctx)
	if err != nil {
		return err
	}

	return availability.Err()
}

func (r *repository) vectorReady(ctx context.Context) (bool, error) {
	availability, err := r.VectorAvailability(ctx)
	if err != nil {
		return false, err
	}

	return availability.Available, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf(
		"%w: "+format,
		append([]any{airetrieval.ErrInvalidStorageRequest}, args...)...)
}

func validateTenant(tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return invalid("organization and business unit are required")
	}

	return nil
}

func validateModelKey(modelKey string) error {
	switch {
	case strings.TrimSpace(modelKey) == "":
		return invalid("a model key is required")
	case len(modelKey) > airetrieval.MaxModelKeyChars:
		return invalid("a model key is at most %d characters", airetrieval.MaxModelKeyChars)
	default:
		return nil
	}
}

func validateSource(source *agentports.AIRetrievalSourceRef) error {
	if source == nil {
		return invalid("a source is required")
	}
	if err := validateTenant(source.TenantInfo); err != nil {
		return err
	}
	if !source.SourceType.IsValid() {
		return invalid("source type %q is not one retrieval indexes", source.SourceType)
	}
	if source.SourceID.IsNil() {
		return invalid("a source id is required")
	}

	return nil
}

func validateDimensions(dimensions int) error {
	if !airetrieval.IsAllowedDimension(dimensions) {
		return fmt.Errorf("%w: got %d", airetrieval.ErrUnsupportedDimensions, dimensions)
	}

	return nil
}

func validateVector(vector []float32, dimensions int) error {
	if len(vector) != dimensions {
		return fmt.Errorf(
			"%w: %d values for %d dimensions",
			airetrieval.ErrEmbeddingLength,
			len(vector),
			dimensions,
		)
	}

	return nil
}

func validateContentHash(hash string) error {
	if hash == "" || len(hash) > airetrieval.ContentHashChars {
		return invalid("a content hash is 1 to %d characters", airetrieval.ContentHashChars)
	}

	return nil
}

func validateSourceTypes(sourceTypes []airetrieval.SourceType) error {
	for _, sourceType := range sourceTypes {
		if !sourceType.IsValid() {
			return invalid("source type %q is not one retrieval indexes", sourceType)
		}
	}

	return nil
}
