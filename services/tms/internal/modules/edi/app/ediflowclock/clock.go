package ediflowclock

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const advancedEventKind = "advanced"

type timerDispatcher interface {
	DispatchFlowTimer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		instanceID pulid.ID,
	) (*edioutbound.FlowTimer, error)
}

type Params struct {
	fx.In

	FlowRepo ladingports.FlowRepository
	Outbound *edioutbound.Service
}

type Service struct {
	flows  ladingports.FlowRepository
	timers timerDispatcher
	now    func() time.Time
}

func New(p Params) *Service {
	return &Service{
		flows:  p.FlowRepo,
		timers: p.Outbound,
		now:    time.Now,
	}
}

type AdvanceRequest struct {
	InstanceID pulid.ID
	By         time.Duration
	ToDue      bool
}

type Advance struct {
	By          time.Duration
	PreviousDue *int64
	Timer       *edioutbound.FlowTimer
}

type advancedDetail struct {
	By int64 `json:"by"`
}

func (s *Service) Advance(ctx context.Context, req *AdvanceRequest) (*Advance, error) {
	if err := validate(req); err != nil {
		return nil, err
	}

	ref, err := s.flows.LocateInstance(ctx, req.InstanceID)
	if err != nil {
		return nil, err
	}
	tenant := ref.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	instance, err := s.flows.GetInstance(ctx, ladingports.GetFlowInstanceRequest{
		TenantInfo: tenant,
		ID:         ref.ID,
	})
	if err != nil {
		return nil, err
	}
	if instance.Status == ladingproject.FlowInstanceStatusClosed {
		return nil, errortypes.NewBusinessError(
			"Flow instance {0} is closed; a closed instance has no clock to advance",
			instance.ID.String(),
		)
	}

	by, err := s.distance(instance, req)
	if err != nil {
		return nil, err
	}
	previousDue := instance.WakeAt
	if err = s.shift(ctx, instance, by); err != nil {
		return nil, err
	}

	timer, err := s.timers.DispatchFlowTimer(ctx, tenant, instance.ID)
	if timer == nil {
		return nil, fmt.Errorf("wake flow instance %s: %w", instance.ID, err)
	}
	advance := &Advance{By: by, PreviousDue: previousDue, Timer: timer}
	if err != nil {
		return advance, fmt.Errorf("carry out what flow instance %s asked: %w", instance.ID, err)
	}
	return advance, nil
}

func validate(req *AdvanceRequest) error {
	switch {
	case req.InstanceID.IsNil():
		return errortypes.NewValidationError(
			"instance",
			errortypes.ErrRequired,
			"Name the flow instance to advance",
		)
	case req.ToDue && req.By != 0:
		return errortypes.NewValidationError(
			"by",
			errortypes.ErrInvalid,
			"Give either a duration or the next due time, not both",
		)
	case req.ToDue:
		return nil
	case req.By < 0:
		return errortypes.NewValidationError(
			"by",
			errortypes.ErrInvalid,
			"A flow's clock only moves forward; {0} is negative",
			req.By.String(),
		)
	case req.By < time.Second:
		return errortypes.NewValidationError(
			"by",
			errortypes.ErrRequired,
			"Give a duration of at least one second, or advance to the next due time",
		)
	default:
		return nil
	}
}

func (s *Service) distance(
	instance *ladingproject.FlowInstance,
	req *AdvanceRequest,
) (time.Duration, error) {
	if !req.ToDue {
		return req.By.Truncate(time.Second), nil
	}
	if instance.WakeAt == nil {
		return 0, errortypes.NewBusinessError(
			"Flow instance {0} waits on no timer, so it has no next due time",
			instance.ID.String(),
		)
	}
	remaining := *instance.WakeAt - s.now().Unix()
	if remaining <= 0 {
		return 0, nil
	}
	return time.Duration(remaining) * time.Second, nil
}

func (s *Service) shift(
	ctx context.Context,
	instance *ladingproject.FlowInstance,
	by time.Duration,
) error {
	seconds := int64(by / time.Second)
	if seconds == 0 {
		return nil
	}

	state, err := lading.ShiftFlowState(instance.State, seconds)
	if err != nil {
		return fmt.Errorf("shift the state of flow instance %s: %w", instance.ID, err)
	}
	instance.State = state
	if instance.WakeAt != nil {
		wake := *instance.WakeAt - seconds
		instance.WakeAt = &wake
	}

	saved, err := s.flows.SaveInstance(ctx, instance)
	if err != nil {
		return err
	}

	detail, err := sonic.Marshal(advancedDetail{By: seconds})
	if err != nil {
		return fmt.Errorf("describe the advance: %w", err)
	}
	if err = s.flows.AppendEvent(ctx, &ladingproject.FlowEvent{
		OrganizationID: saved.OrganizationID,
		BusinessUnitID: saved.BusinessUnitID,
		InstanceID:     saved.ID,
		Kind:           advancedEventKind,
		Detail:         detail,
		Actions:        []lading.FlowAction{},
	}); err != nil {
		return fmt.Errorf("journal the advance: %w", err)
	}
	return nil
}
