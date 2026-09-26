package invoicerunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// AdjustMembership applies a whole operator edit at once.
//
// Excludes, includes and moves arrive together rather than as separate calls so
// the edit lands in one transaction and leaves one audit entry. A trail of
// single-row changes is not something anyone can read back later to understand
// why a statement looked the way it did.
func (s *Service) AdjustMembership(
	ctx context.Context,
	req *servicesports.AdjustInvoiceRunMembershipRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	run, err := s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            req.RunID,
		TenantInfo:    req.TenantInfo,
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateAdjust(run); multiErr != nil {
		return nil, multiErr
	}

	touched, err := applyMembership(run, req)
	if err != nil {
		return nil, err
	}

	if len(touched) == 0 {
		return run, nil
	}

	updates := make([]*invoicerun.InvoiceRunGroupItem, 0, len(touched))
	for _, item := range touched {
		updates = append(updates, item)
	}
	if err = s.repo.UpdateItems(ctx, req.TenantInfo, updates); err != nil {
		return nil, err
	}

	if err = s.resyncGroups(ctx, run); err != nil {
		return nil, err
	}

	s.audit(ctx, run, actor, permission.OpUpdate, "Invoice run membership adjusted")

	return s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            req.RunID,
		TenantInfo:    req.TenantInfo,
		IncludeGroups: true,
		IncludeItems:  true,
	})
}

// resyncGroups recomputes the totals an operator reads after an edit, and the
// run's rollup with them.
func (s *Service) resyncGroups(ctx context.Context, run *invoicerun.InvoiceRun) error {
	// Re-read so a moved item is counted against the group it landed in rather
	// than the one it left.
	fresh, err := s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            run.ID,
		TenantInfo:    tenantOf(run),
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		return err
	}

	for _, group := range fresh.Groups {
		if group == nil {
			continue
		}
		group.SyncTotals()
		if _, err = s.repo.UpdateGroup(ctx, group); err != nil {
			return err
		}
	}

	fresh.SyncTotals()
	if _, err = s.repo.Update(ctx, fresh); err != nil {
		return err
	}

	return nil
}

type itemIndex struct {
	items  map[pulid.ID]*invoicerun.InvoiceRunGroupItem
	groups map[pulid.ID]*invoicerun.InvoiceRunGroup
}

func indexItems(run *invoicerun.InvoiceRun) itemIndex {
	index := itemIndex{
		items:  make(map[pulid.ID]*invoicerun.InvoiceRunGroupItem),
		groups: make(map[pulid.ID]*invoicerun.InvoiceRunGroup),
	}

	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		index.groups[group.ID] = group
		for _, item := range group.Items {
			if item == nil {
				continue
			}
			item.GroupID = group.ID
			index.items[item.ID] = item
		}
	}

	return index
}

func unknownItem(id pulid.ID) error {
	return errortypes.NewValidationError(
		"itemId",
		errortypes.ErrInvalid,
		"Shipment {0} is not on this run", id.String(),
	)
}

func unknownGroup(id pulid.ID) error {
	return errortypes.NewValidationError(
		"targetGroupId",
		errortypes.ErrInvalid,
		"Group {0} is not on this run", id.String(),
	)
}
