package dispatchmarketservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Repo          dispatchmarketports.Repository
	Organizations tenantports.OrganizationRepository
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	repo          dispatchmarketports.Repository
	organizations tenantports.OrganizationRepository
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.dispatch-market"),
		db:            p.DB,
		repo:          p.Repo,
		organizations: p.Organizations,
	}
}

func (s *Service) grid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (dispatchmarket.Grid, error) {
	location, err := orgzone.Location(ctx, s.organizations, tenantInfo)
	if err != nil {
		return dispatchmarket.Grid{}, err
	}

	return dispatchmarket.GridIn(location), nil
}

func (s *Service) Valuer(
	ctx context.Context,
	req *dispatchmarketports.ValuerRequest,
) (*dispatchmarket.Valuer, error) {
	grid, err := s.grid(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	cells := dispatchmarket.CellsOf(req.Points)
	if len(cells) == 0 {
		return dispatchmarket.NewValuer(grid, nil), nil
	}

	values, err := s.repo.ListByCells(ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: req.TenantInfo,
		Cells:      cells,
	})
	if err != nil {
		return nil, err
	}

	return dispatchmarket.NewValuer(grid, values), nil
}

func (s *Service) Record(ctx context.Context, req *dispatchmarketports.RecordRequest) error {
	if len(req.Values) == 0 {
		return nil
	}
	grid, err := s.grid(ctx, req.TenantInfo)
	if err != nil {
		return err
	}
	samples := dispatchmarket.Aggregate(grid, req.ObservedAt, req.Values)
	if len(samples) == 0 {
		return nil
	}

	return s.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		existing, err := s.repo.LockByCells(ctx, &dispatchmarketports.ListByCellsRequest{
			TenantInfo: req.TenantInfo,
			Cells:      dispatchmarket.SampleCells(samples),
		})
		if err != nil {
			return err
		}

		changed := dispatchmarket.FoldAll(&dispatchmarket.FoldParams{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Existing:       existing,
			Samples:        samples,
		})

		return s.repo.Upsert(ctx, changed)
	})
}
