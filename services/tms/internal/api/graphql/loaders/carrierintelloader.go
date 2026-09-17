package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type carrierIntelReader interface {
	CurrentSnapshotsByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	CurrentSnapshotsBySubjects(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		subjectType carrierintel.SubjectType,
		subjectIDs []string,
	) ([]*carrierintel.CarrierIntelSnapshot, error)
	OpenEventCountsByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) (map[pulid.ID]int, error)
	EnrollmentsByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) ([]*carrierintel.CarrierMonitoringEnrollment, error)
}

type CarrierIntelLoaderFactoryParams struct {
	fx.In

	CarrierIntelService *carrierintelservice.Service
}

type CarrierIntelSnapshotByCarrierIDLoaderFactory struct {
	reader carrierIntelReader
}

func NewCarrierIntelSnapshotByCarrierIDLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierIntelSnapshotByCarrierIDLoaderFactory {
	return &CarrierIntelSnapshotByCarrierIDLoaderFactory{reader: p.CarrierIntelService}
}

func (f *CarrierIntelSnapshotByCarrierIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintel.CarrierIntelSnapshot, error) {
			snapshots, err := f.reader.CurrentSnapshotsByCarrierIDs(ctx, tenantInfo, ids)
			if err != nil {
				return nil, err
			}
			grouped := make(map[pulid.ID][]*carrierintel.CarrierIntelSnapshot, len(snapshots))
			for _, snapshot := range snapshots {
				grouped[snapshot.CarrierID] = append(grouped[snapshot.CarrierID], snapshot)
			}
			return grouped, nil
		},
	))
}

type CarrierIntelSnapshotByCustomerIDLoaderFactory struct {
	reader carrierIntelReader
}

func NewCarrierIntelSnapshotByCustomerIDLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierIntelSnapshotByCustomerIDLoaderFactory {
	return &CarrierIntelSnapshotByCustomerIDLoaderFactory{reader: p.CarrierIntelService}
}

func (f *CarrierIntelSnapshotByCustomerIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintel.CarrierIntelSnapshot, error) {
			subjectIDs := make([]string, 0, len(ids))
			for _, id := range ids {
				subjectIDs = append(subjectIDs, id.String())
			}
			snapshots, err := f.reader.CurrentSnapshotsBySubjects(
				ctx, tenantInfo, carrierintel.SubjectTypeCustomer, subjectIDs,
			)
			if err != nil {
				return nil, err
			}
			grouped := make(map[pulid.ID][]*carrierintel.CarrierIntelSnapshot, len(snapshots))
			for _, snapshot := range snapshots {
				customerID, parseErr := pulid.Parse(snapshot.SubjectID)
				if parseErr != nil {
					continue
				}
				grouped[customerID] = append(grouped[customerID], snapshot)
			}
			return grouped, nil
		},
	))
}

type CarrierIntelOpenEventCountLoaderFactory struct {
	reader carrierIntelReader
}

func NewCarrierIntelOpenEventCountLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierIntelOpenEventCountLoaderFactory {
	return &CarrierIntelOpenEventCountLoaderFactory{reader: p.CarrierIntelService}
}

func (f *CarrierIntelOpenEventCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(
		batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.reader.OpenEventCountsByCarrierIDs(ctx, tenantInfo, ids)
		}),
	)
}

type CarrierMonitoringEnrollmentByCarrierIDLoaderFactory struct {
	reader carrierIntelReader
}

func NewCarrierMonitoringEnrollmentByCarrierIDLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory {
	return &CarrierMonitoringEnrollmentByCarrierIDLoaderFactory{reader: p.CarrierIntelService}
}

func (f *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintel.CarrierMonitoringEnrollment] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintel.CarrierMonitoringEnrollment, error) {
			rows, err := f.reader.EnrollmentsByCarrierIDs(ctx, tenantInfo, ids)
			if err != nil {
				return nil, err
			}
			grouped := make(map[pulid.ID][]*carrierintel.CarrierMonitoringEnrollment, len(rows))
			for _, row := range rows {
				grouped[row.CarrierID] = append(grouped[row.CarrierID], row)
			}
			return grouped, nil
		},
	))
}
