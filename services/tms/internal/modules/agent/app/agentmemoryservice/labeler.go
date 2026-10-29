package agentmemoryservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

type LabelerParams struct {
	fx.In

	Customers customerports.CustomerRepository
	Locations locationports.LocationRepository
	Workers   workerports.WorkerRepository
	Carriers  carrierports.CarrierRepository
}

// repositoryLabeler names a subject by reading it, which also proves it
// exists in the tenant asking.
type repositoryLabeler struct {
	customers customerports.CustomerRepository
	locations locationports.LocationRepository
	workers   workerports.WorkerRepository
	carriers  carrierports.CarrierRepository
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
	ref agentmemoryports.MemorySubjectRef,
) (string, error) {
	switch ref.Type {
	case agentmemory.MemorySubjectCustomer:
		entity, err := l.customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return stringutils.FirstNonEmpty(entity.Name, entity.Code), nil
	case agentmemory.MemorySubjectLocation:
		entity, err := l.locations.GetByID(ctx, locationports.GetLocationByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return stringutils.FirstNonEmpty(entity.Name, entity.Code), nil
	case agentmemory.MemorySubjectWorker:
		entity, err := l.workers.GetByID(ctx, workerports.GetWorkerByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(entity.FirstName + " " + entity.LastName), nil
	case agentmemory.MemorySubjectCarrier:
		entity, err := l.carriers.GetByID(ctx, carrierports.GetCarrierByIDRequest{
			ID:         ref.ID,
			TenantInfo: tenant,
		})
		if err != nil {
			return "", err
		}

		return stringutils.FirstNonEmpty(entity.Name, entity.DBAName), nil
	default:
		return "", fmt.Errorf("%q is not a subject a memory can be about", ref.Type)
	}
}
