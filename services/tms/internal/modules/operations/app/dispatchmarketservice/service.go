package dispatchmarketservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	DB     kernelports.Transactor
	Repo   dispatchmarketports.Repository
}

type Service struct {
	l    *zap.Logger
	db   kernelports.Transactor
	repo dispatchmarketports.Repository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.dispatch-market"),
		db:   p.DB,
		repo: p.Repo,
	}
}

func (s *Service) Valuer(
	ctx context.Context,
	req *dispatchmarketports.ValuerRequest,
) (*dispatchmarket.Valuer, error) {
	cells := dispatchmarket.CellsOf(req.Points)
	if len(cells) == 0 {
		return dispatchmarket.NewValuer(nil), nil
	}

	values, err := s.repo.ListByCells(ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: req.TenantInfo,
		Cells:      cells,
	})
	if err != nil {
		return nil, err
	}

	return dispatchmarket.NewValuer(values), nil
}

func (s *Service) Record(ctx context.Context, req *dispatchmarketports.RecordRequest) error {
	samples := dispatchmarket.Aggregate(req.ObservedAt, req.Values)
	if len(samples) == 0 {
		return nil
	}

	return s.db.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		existing, err := s.repo.ListByCells(ctx, &dispatchmarketports.ListByCellsRequest{
			TenantInfo: req.TenantInfo,
			Cells:      dispatchmarket.SampleCells(samples),
			ForUpdate:  true,
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
