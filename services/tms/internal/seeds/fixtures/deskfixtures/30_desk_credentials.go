package deskfixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
)

const (
	queueMedicalCardDays = 12
	queueHazmatDays      = 46
)

// credentialsAsTheQueueSaysThem makes the two credential warnings in the
// seeded queue true of the records they are about. The queue says the first
// worker's medical card lapses in 12 days and the second's hazmat endorsement
// in 46; an agent asked about either reads the worker's own papers, and one
// that found a card good for another year would be right to say the warning
// was wrong. A case has to rest on what the record holds.
func (s *DeskSeed) credentialsAsTheQueueSaysThem(
	ctx context.Context,
	db *keel.DB,
	refs *deskSeedRefs,
) error {
	if len(refs.workers) == 0 {
		return nil
	}
	if err := s.expireIn(ctx, db, refs, refs.workers[0], queueMedicalCardDays,
		"MED_CARD", "DOT_PHYSICAL"); err != nil {
		return err
	}
	if len(refs.workers) < 2 {
		return nil
	}

	return s.expireIn(ctx, db, refs, refs.workers[1], queueHazmatDays, "HAZMAT")
}

func (s *DeskSeed) expireIn(
	ctx context.Context,
	db *keel.DB,
	refs *deskSeedRefs,
	holder *worker.Worker,
	days int64,
	codes ...string,
) error {
	t := workercredentialdb.WorkerCredentialTypes
	c := workercredentialdb.WorkerCredentials
	expires := refs.now + days*deskSeedDay

	for _, code := range codes {
		kind, err := t.AllTenants().Where(
			t.OrganizationID.Eq(refs.org.ID),
			t.BusinessUnitID.Eq(refs.org.BusinessUnitID),
			t.Code.Eq(code),
		).First(ctx, db)
		if errors.Is(err, keel.ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("look for the %s credential type: %w", code, err)
		}

		held, err := c.AllTenants().Where(
			c.OrganizationID.Eq(refs.org.ID),
			c.BusinessUnitID.Eq(refs.org.BusinessUnitID),
			c.WorkerID.Eq(holder.ID),
			c.CredentialTypeID.Eq(kind.ID),
			c.Status.Eq(workercredential.CredentialStatusActive),
		).First(ctx, db)
		switch {
		case errors.Is(err, keel.ErrNotFound):
			issued := refs.now - 300*deskSeedDay
			if err = c.Create(ctx, db, &workercredential.WorkerCredential{
				OrganizationID:   refs.org.ID,
				BusinessUnitID:   refs.org.BusinessUnitID,
				WorkerID:         holder.ID,
				CredentialTypeID: kind.ID,
				Status:           workercredential.CredentialStatusActive,
				IssuedAt:         &issued,
				ExpiresAt:        &expires,
			}); err != nil {
				return fmt.Errorf("insert the %s credential: %w", code, err)
			}
		case err != nil:
			return fmt.Errorf("look for the worker's %s credential: %w", code, err)
		default:
			held.ExpiresAt = &expires
			if err = c.Save(ctx, db, held, c.ExpiresAt); err != nil {
				return fmt.Errorf("set when the %s credential expires: %w", code, err)
			}
		}
	}

	return nil
}
