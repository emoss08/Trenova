// Package approvalworkflow runs the Draft → InReview → Active review cycle that
// several configuration entities share.
//
// Formula templates and rate agreements both price shipments, and both are
// things an organization wants reviewed before they can. The cycle is identical
// in each: check the move is legal for the current status, stamp who did it and
// when, save, and record the change in the audit log. Only the entity, its
// status type, and the fields each step stamps differ, and those are what the
// caller supplies.
//
// Keeping it in one place matters more than the line count saved. A second copy
// would eventually disagree with the first about the order of the checks — and
// the order is the whole guarantee, because it is what stops a rejected
// agreement from having already been saved as active.
package approvalworkflow

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// Request is what a caller asks for: move this entity to its next state, with
// this comment, on behalf of this tenant and user.
type Request struct {
	TenantInfo pagination.TenantInfo
	EntityID   pulid.ID
	Comment    string
}

// Transition describes one legal step of a review cycle.
//
// Apply is where the entity-specific stamping lives — which fields record the
// submitter, the approver, the comment. Everything around it is the same
// whatever is being reviewed.
type Transition[T any, S comparable] struct {
	Operation string
	From      S
	// AlsoFrom lists further statuses the step may start from, for a cycle
	// where two states share a next step — an archived template is submitted
	// for review the same way a draft is.
	AlsoFrom     []S
	To           S
	PermissionOp permission.Operation
	AuditComment string
	Apply        func(entity T, req *Request, now int64)

	// Validate runs after the status check and before anything is stamped or
	// saved, so a transition-specific rule can refuse the move while the entity
	// is still untouched. Nil means no extra rule.
	Validate func(ctx context.Context, entity T, req *Request) error

	// AfterSave runs after the entity is saved and before the audit record, for
	// work that must land with the transition, such as a version snapshot. Its
	// error fails the transition, so callers wanting atomicity run Apply inside
	// a transaction. Nil means no extra work.
	AfterSave func(ctx context.Context, updated T, req *Request) error
}

// Engine binds a review cycle to one entity's storage, status rules and audit
// trail.
//
// The closures are the seams that differ between entities; nothing else does.
type Engine[T any, S comparable] struct {
	// Label names the entity in the error a caller sees, so "Cannot transition
	// rate agreement status from Active to Active" reads as a sentence.
	Label string

	Load          func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (T, error)
	Save          func(ctx context.Context, entity T) (T, error)
	StatusOf      func(entity T) S
	SetStatus     func(entity T, status S)
	CanTransition func(from, to S) bool
	Snapshot      func(entity T) T

	// Audit records the change. It is given the saved entity and the snapshot
	// taken before the transition, so the log carries a real before and after.
	Audit func(updated, original T, operation permission.Operation, req *Request, comment string)

	// Now exists so tests can pin the moment a transition claims to have
	// happened. Zero means the wall clock.
	Now func() int64
}

// Apply runs one transition, or explains why it cannot.
//
// The order is deliberate: the status is checked before anything is written, so
// an illegal move leaves the entity exactly as it was.
func (e Engine[T, S]) Apply(
	ctx context.Context,
	req *Request,
	transition Transition[T, S],
) (T, error) {
	var zero T

	entity, err := e.Load(ctx, req.EntityID, req.TenantInfo)
	if err != nil {
		return zero, err
	}

	current := e.StatusOf(entity)
	if !transition.startsFrom(current) || !e.CanTransition(current, transition.To) {
		return zero, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot transition %s status from %v to %v",
				e.Label,
				current,
				transition.To,
			),
		)
	}

	if transition.Validate != nil {
		if err = transition.Validate(ctx, entity, req); err != nil {
			return zero, err
		}
	}

	original := e.Snapshot(entity)

	e.SetStatus(entity, transition.To)
	transition.Apply(entity, req, e.now())

	updated, err := e.Save(ctx, entity)
	if err != nil {
		return zero, err
	}

	if transition.AfterSave != nil {
		if err = transition.AfterSave(ctx, updated, req); err != nil {
			return zero, err
		}
	}

	if e.Audit != nil {
		e.Audit(updated, original, transition.PermissionOp, req, transition.AuditComment)
	}

	return updated, nil
}

func (t Transition[T, S]) startsFrom(status S) bool {
	return status == t.From || slices.Contains(t.AlsoFrom, status)
}

// RequireComment refuses a transition that would leave no record of why.
//
// Rejection is the case that matters: sending something back without saying
// what is wrong with it wastes the next person's time, and by then the reviewer
// has moved on.
func RequireComment(req *Request, action string) error {
	if req.Comment != "" {
		return nil
	}

	return errortypes.NewValidationError(
		"comment",
		errortypes.ErrRequired,
		"A comment is required when "+action,
	)
}

// Clock is the moment the engine stamps transitions with, exposed so a
// transition's own rules measure age against the same instant.
func (e Engine[T, S]) Clock() int64 {
	return e.now()
}

func (e Engine[T, S]) now() int64 {
	if e.Now != nil {
		return e.Now()
	}

	return timeutils.NowUnix()
}
