package agentwaitservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentwaitjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// facts reads the records a wait names, to refuse a wait on something already
// true and to look again when a timed wait comes due.
type facts struct {
	moves       repositories.ShipmentMoveRepository
	assignments repositories.AssignmentRepository
	occurrences repositories.DetentionOccurrenceRepository
	telematics  repositories.TelematicsRepository
	carriers    repositories.CarrierRepository
	customers   repositories.CustomerRepository
	subjects    repositories.AgentSubjectRepository
}

// alreadyTrue refuses a wait on something that has happened: the agent is
// told what is true now and acts on it rather than waiting for nothing.
func alreadyTrue(format string, args ...any) error {
	return errortypes.NewBusinessError(
		"There is nothing to wait for: " + fmt.Sprintf(format, args...) +
			" Act on that now instead of waiting.",
	)
}

// recordMissingError is a record a wait names that is not in the organization.
type recordMissingError struct{ what string }

func (e *recordMissingError) Error() string {
	return "No " + e.what + " with that id is in this organization."
}

func notFound(what string) error { return &recordMissingError{what: what} }

// assess checks the records the wait names and says when it comes due, for a
// wait that comes due at a time.
func (f *facts) assess(ctx context.Context, wait *agentwait.Wait, now int64) (*int64, error) {
	tenant := tenantOf(wait)
	c := wait.Condition

	switch wait.Kind {
	case agentwait.KindTime:
		if c.At < now+minLead {
			return nil, alreadyTrue("that time is now or has passed.")
		}
		return &c.At, nil

	case agentwait.KindStopArrival, agentwait.KindStopDeparture:
		return nil, f.assessStopVisit(
			ctx,
			tenant,
			wait,
		) //nolint:nilnil // an event, not a time, ends it

	case agentwait.KindReply:
		return nil, f.assessParty(ctx, tenant, c) //nolint:nilnil // an event, not a time, ends it

	case agentwait.KindAppointmentNear:
		stop, err := f.stop(ctx, tenant, c.ShipmentMoveID, c.StopID)
		if err != nil {
			return nil, err
		}
		if stop.ActualArrival != nil {
			return nil, alreadyTrue("the stop was arrived at %s.", stamp(*stop.ActualArrival))
		}
		due := stop.ScheduledWindowStart - int64(c.MinutesBefore)*60
		if due < now+minLead {
			return nil, alreadyTrue("the appointment at %s is already less than %d minutes away.",
				stamp(stop.ScheduledWindowStart), c.MinutesBefore)
		}
		return &due, nil

	case agentwait.KindFreeTimeEnding:
		occurrence, err := f.occurrence(ctx, tenant, c.DetentionOccurrenceID)
		if err != nil {
			return nil, err
		}
		if !occurrence.IsOpen {
			return nil, alreadyTrue("the truck has already left this stop.")
		}
		due := occurrence.FreeTimeExpiresAt - int64(c.MinutesBefore)*60
		if due < now+minLead {
			return nil, alreadyTrue("free time ends at %s, already less than %d minutes away.",
				stamp(occurrence.FreeTimeExpiresAt), c.MinutesBefore)
		}
		return &due, nil

	case agentwait.KindHOSDriveBelow:
		state, err := f.telematics.GetWorkerHOSState(ctx, repositories.GetWorkerHOSStateRequest{
			TenantInfo: tenant,
			WorkerID:   c.WorkerID,
		})
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return nil, errortypes.NewBusinessError(
					"No hours-of-service clock is reported for this driver, so their drive " +
						"time cannot be watched. Check their telematics mapping.")
			}
			return nil, err
		}
		left := projectedDriveMs(state, now)
		if left < int64(c.DriveMinutesBelow)*int64(time.Minute/time.Millisecond) {
			return nil, alreadyTrue("the driver has about %s of drive time left.",
				durationText(left))
		}
		return crossingAt(state, c.DriveMinutesBelow), nil

	default:
		return nil, errortypes.NewBusinessError("Unknown kind of wait")
	}
}

func (f *facts) assessStopVisit(
	ctx context.Context,
	tenant pagination.TenantInfo,
	wait *agentwait.Wait,
) error {
	c := wait.Condition
	move, err := f.move(ctx, tenant, c.ShipmentMoveID)
	if err != nil {
		return err
	}
	if move.Status == shipment.MoveStatusCompleted || move.Status == shipment.MoveStatusCanceled {
		return alreadyTrue("the move is %s.", move.Status)
	}
	if c.StopID.IsNil() {
		return nil
	}

	stop := stopOf(move, c.StopID)
	if stop == nil {
		return notFound("stop on that move")
	}
	if wait.Kind == agentwait.KindStopArrival && stop.ActualArrival != nil {
		return alreadyTrue("the truck arrived at %s at %s.", stopName(stop),
			stamp(*stop.ActualArrival))
	}
	if wait.Kind == agentwait.KindStopDeparture && stop.ActualDeparture != nil {
		return alreadyTrue("the truck left %s at %s.", stopName(stop),
			stamp(*stop.ActualDeparture))
	}

	return nil
}

func (f *facts) assessParty(
	ctx context.Context,
	tenant pagination.TenantInfo,
	c *agentwait.Condition,
) error {
	switch {
	case c.ShipmentID.IsNotNil():
		found, err := f.subjects.Exists(ctx, repositories.AgentSubjectExistsRequest{
			TenantInfo:  tenant,
			SubjectType: agent.SubjectShipment,
			SubjectID:   c.ShipmentID,
		})
		if err != nil {
			return err
		}
		if !found {
			return notFound("shipment")
		}
	case c.CarrierID.IsNotNil():
		if _, err := f.carriers.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         c.CarrierID,
			TenantInfo: tenant,
		}); err != nil {
			return missing(err, "carrier")
		}
	default:
		if _, err := f.customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         c.CustomerID,
			TenantInfo: tenant,
		}); err != nil {
			return missing(err, "customer")
		}
	}

	return nil
}

// recheck reads a timed wait's record again when it comes due.
func (f *facts) recheck(
	ctx context.Context,
	wait *agentwait.Wait,
	now int64,
) (*agentwaitjobs.Check, error) {
	tenant := tenantOf(wait)
	c := wait.Condition

	switch wait.Kind { //nolint:exhaustive // only timed waits come due
	case agentwait.KindAppointmentNear:
		stop, err := f.stop(ctx, tenant, c.ShipmentMoveID, c.StopID)
		if err != nil {
			return goneOr(err, "The stop is no longer on the move.")
		}
		if stop.ActualArrival != nil {
			return met("The truck arrived at %s at %s, before the appointment came round.",
				stopName(stop), stamp(*stop.ActualArrival)), nil
		}
		due := stop.ScheduledWindowStart - int64(c.MinutesBefore)*60
		if due > now+rearmSlack {
			return &agentwaitjobs.Check{DueAt: &due}, nil
		}
		return met("The appointment at %s is at %s.", stopName(stop),
			stamp(stop.ScheduledWindowStart)), nil

	case agentwait.KindFreeTimeEnding:
		occurrence, err := f.occurrence(ctx, tenant, c.DetentionOccurrenceID)
		if err != nil {
			return goneOr(err, "The detention record is gone.")
		}
		if !occurrence.IsOpen {
			return met("The truck left before free time ran out."), nil
		}
		due := occurrence.FreeTimeExpiresAt - int64(c.MinutesBefore)*60
		if due > now+rearmSlack {
			return &agentwaitjobs.Check{DueAt: &due}, nil
		}
		return met("Free time ends at %s.", stamp(occurrence.FreeTimeExpiresAt)), nil

	case agentwait.KindHOSDriveBelow:
		state, err := f.telematics.GetWorkerHOSState(ctx, repositories.GetWorkerHOSStateRequest{
			TenantInfo: tenant,
			WorkerID:   c.WorkerID,
		})
		if err != nil {
			return nil, err
		}
		left := projectedDriveMs(state, now)
		if left < int64(c.DriveMinutesBelow)*int64(time.Minute/time.Millisecond) {
			return met("Drive time left: about %s, projected from the clock read at %s "+
				"while driving.", durationText(left), stamp(state.RecordedAt)), nil
		}
		return &agentwaitjobs.Check{DueAt: crossingAt(state, c.DriveMinutesBelow)}, nil

	default:
		return met("It is %s.", stamp(now)), nil
	}
}

func met(format string, args ...any) *agentwaitjobs.Check {
	return &agentwaitjobs.Check{Met: true, Detail: fmt.Sprintf(format, args...)}
}

// goneOr ends a wait whose record disappeared, saying so, and reports any
// other failure for the activity to retry.
func goneOr(err error, gone string) (*agentwaitjobs.Check, error) {
	var missing *recordMissingError
	if errors.As(err, &missing) {
		return &agentwaitjobs.Check{Met: true, Detail: gone}, nil
	}

	return nil, err
}

func missing(err error, what string) error {
	if errortypes.IsNotFoundError(err) {
		return notFound(what)
	}

	return err
}

func (f *facts) move(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, error) {
	move, err := f.moves.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            moveID,
		TenantInfo:        tenant,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, missing(err, "move")
	}

	return move, nil
}

func (f *facts) stop(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID, stopID pulid.ID,
) (*shipment.Stop, error) {
	move, err := f.move(ctx, tenant, moveID)
	if err != nil {
		return nil, err
	}
	stop := stopOf(move, stopID)
	if stop == nil {
		return nil, notFound("stop on that move")
	}

	return stop, nil
}

func (f *facts) occurrence(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*detention.DetentionOccurrence, error) {
	occurrence, err := f.occurrences.GetByID(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
		OccurrenceID: id,
		TenantInfo:   tenant,
	})
	if err != nil {
		return nil, missing(err, "detention record")
	}

	return occurrence, nil
}

func stopName(stop *shipment.Stop) string {
	if stop.Location != nil && stop.Location.Name != "" {
		return stop.Location.Name
	}

	return "the stop"
}

func stamp(at int64) string {
	return time.Unix(at, 0).UTC().Format(timeLayout)
}
