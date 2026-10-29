package partiesloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelreviewservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
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
	) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error)
	CurrentSnapshotsBySubjects(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		subjectType carrierintel.SubjectType,
		subjectIDs []string,
	) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error)
	OpenEventCountsByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type carrierIntelEnrollmentReader interface {
	EnrollmentsByCarrierIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) ([]*carrierintelenrollment.CarrierMonitoringEnrollment, error)
}

type CarrierIntelLoaderFactoryParams struct {
	fx.In

	CarrierIntelReviewService     *carrierintelreviewservice.Service
	CarrierIntelEnrollmentService *carrierintelenrollmentservice.Service
}

type CarrierIntelSnapshotByCarrierIDLoaderFactory struct {
	reader carrierIntelReader
}

func NewCarrierIntelSnapshotByCarrierIDLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierIntelSnapshotByCarrierIDLoaderFactory {
	return &CarrierIntelSnapshotByCarrierIDLoaderFactory{reader: p.CarrierIntelReviewService}
}

func newCarrierIntelSnapshotByCarrierIDLoader(f *CarrierIntelSnapshotByCarrierIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintelsnapshot.CarrierIntelSnapshot] {
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
			snapshots, err := f.reader.CurrentSnapshotsByCarrierIDs(ctx, tenantInfo, ids)
			if err != nil {
				return nil, err
			}
			grouped := make(
				map[pulid.ID][]*carrierintelsnapshot.CarrierIntelSnapshot,
				len(snapshots),
			)
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
	return &CarrierIntelSnapshotByCustomerIDLoaderFactory{reader: p.CarrierIntelReviewService}
}

func newCarrierIntelSnapshotByCustomerIDLoader(f *CarrierIntelSnapshotByCustomerIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintelsnapshot.CarrierIntelSnapshot] {
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
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
			grouped := make(
				map[pulid.ID][]*carrierintelsnapshot.CarrierIntelSnapshot,
				len(snapshots),
			)
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
	return &CarrierIntelOpenEventCountLoaderFactory{reader: p.CarrierIntelReviewService}
}

func newCarrierIntelOpenEventCountLoader(f *CarrierIntelOpenEventCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(
		loadercore.BatchCountFunc(
			func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
				return f.reader.OpenEventCountsByCarrierIDs(ctx, tenantInfo, ids)
			},
		),
	)
}

type CarrierMonitoringEnrollmentByCarrierIDLoaderFactory struct {
	reader carrierIntelEnrollmentReader
}

func NewCarrierMonitoringEnrollmentByCarrierIDLoaderFactory(
	p CarrierIntelLoaderFactoryParams,
) *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory {
	return &CarrierMonitoringEnrollmentByCarrierIDLoaderFactory{
		reader: p.CarrierIntelEnrollmentService,
	}
}

func newCarrierMonitoringEnrollmentByCarrierIDLoader(
	f *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*carrierintelenrollment.CarrierMonitoringEnrollment] {
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*carrierintelenrollment.CarrierMonitoringEnrollment, error) {
			rows, err := f.reader.EnrollmentsByCarrierIDs(ctx, tenantInfo, ids)
			if err != nil {
				return nil, err
			}
			grouped := make(
				map[pulid.ID][]*carrierintelenrollment.CarrierMonitoringEnrollment,
				len(rows),
			)
			for _, row := range rows {
				grouped[row.CarrierID] = append(grouped[row.CarrierID], row)
			}
			return grouped, nil
		},
	))
}

var (
	carrierIntelSnapshotByCarrierIDKey = loadercore.NewKey(
		newCarrierIntelSnapshotByCarrierIDLoader,
	)
	carrierIntelSnapshotByCustomerIDKey = loadercore.NewKey(
		newCarrierIntelSnapshotByCustomerIDLoader,
	)
	carrierIntelOpenEventCountKey = loadercore.NewKey(
		newCarrierIntelOpenEventCountLoader,
	)
	carrierMonitoringEnrollmentByCarrierIDKey = loadercore.NewKey(
		newCarrierMonitoringEnrollmentByCarrierIDLoader,
	)
)

func CarrierIntelSnapshotByCarrierID(
	ctx context.Context,
	key string,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return loadercore.Load(ctx, carrierIntelSnapshotByCarrierIDKey, key)
}

func (f *CarrierIntelSnapshotByCarrierIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func CarrierIntelSnapshotByCustomerID(
	ctx context.Context,
	key string,
) ([]*carrierintelsnapshot.CarrierIntelSnapshot, error) {
	return loadercore.Load(ctx, carrierIntelSnapshotByCustomerIDKey, key)
}

func (f *CarrierIntelSnapshotByCustomerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func CarrierIntelOpenEventCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, carrierIntelOpenEventCountKey, key)
}

func (f *CarrierIntelOpenEventCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func CarrierMonitoringEnrollmentByCarrierID(
	ctx context.Context,
	key string,
) ([]*carrierintelenrollment.CarrierMonitoringEnrollment, error) {
	return loadercore.Load(ctx, carrierMonitoringEnrollmentByCarrierIDKey, key)
}

func (f *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
