package agentmemoryservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type LabelerParams struct {
	fx.In

	Customers repositories.CustomerRepository
	Locations repositories.LocationRepository
	Workers   repositories.WorkerRepository
	Carriers  repositories.CarrierRepository
}

// repositoryLabeler names a subject by reading it, which also proves it
// exists in the tenant asking.
type repositoryLabeler struct {
	customers repositories.CustomerRepository
	locations repositories.LocationRepository
	workers   repositories.WorkerRepository
	carriers  repositories.CarrierRepository
}

func NewLabeler(p LabelerParams) SubjectLabeler {
	return &repositoryLabeler{
		customers: p.Customers,
		locations: p.Locations,
		workers:   p.Workers,
		carriers:  p.Carriers,
	}
}

func (l *repositoryLabeler) Label(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ref repositories.MemorySubjectRef,
) (string, error) {
	switch ref.Type {
	case agent.MemorySubjectCustomer:
		entity, err := l.customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return firstNonEmpty(entity.Name, entity.Code), nil
	case agent.MemorySubjectLocation:
		entity, err := l.locations.GetByID(ctx, repositories.GetLocationByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return firstNonEmpty(entity.Name, entity.Code), nil
	case agent.MemorySubjectWorker:
		entity, err := l.workers.GetByID(ctx, repositories.GetWorkerByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(entity.FirstName + " " + entity.LastName), nil
	case agent.MemorySubjectCarrier:
		entity, err := l.carriers.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return firstNonEmpty(entity.Name, entity.DBAName), nil
	default:
		return "", fmt.Errorf("%q is not a subject a memory can be about", ref.Type)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
