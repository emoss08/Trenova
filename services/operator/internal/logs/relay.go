package logs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	tmsTimeout      = 30 * time.Second
	maxNameSize     = 200
	conflictMessage = "Trenova support cannot take more logs from this instance right now. Try again later."
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

type TMS interface {
	LogRequests(ctx context.Context) ([]tms.LogRequest, error)
	UpdateLogRequest(ctx context.Context, id string, update tms.LogRequestUpdate) error
}

type request struct {
	bundle uuid.UUID
	final  *tms.LogRequestUpdate
}

func (s *Service) Relay(ctx context.Context) error {
	list, err := s.tms.LogRequests(ctx)
	if err != nil {
		return err
	}
	listed := make(map[string]struct{}, len(list))
	var errs []error
	for i := range list {
		listed[list[i].ID] = struct{}{}
		if err = s.relayOne(ctx, &list[i]); err != nil {
			errs = append(errs, err)
		}
	}
	s.mu.Lock()
	for id, r := range s.requests {
		if _, ok := listed[id]; !ok && r.final != nil {
			delete(s.requests, id)
		}
	}
	s.mu.Unlock()
	return errors.Join(errs...)
}

func (s *Service) relayOne(ctx context.Context, req *tms.LogRequest) error {
	if !requestIDPattern.MatchString(req.ID) {
		return errors.New("the TMS listed a log request whose ID is not one")
	}
	s.mu.Lock()
	known := s.requests[req.ID]
	var final *tms.LogRequestUpdate
	if known != nil && known.final != nil {
		f := *known.final
		final = &f
	}
	s.mu.Unlock()
	switch {
	case final != nil:
		return s.tell(ctx, req.ID, *final)
	case known != nil:
		return nil
	}
	if msg := PeriodProblem(req.Since, req.Until, s.caps.Period); msg != "" {
		return s.tell(ctx, req.ID, failedUpdate(msg))
	}
	if !s.Kept() {
		return s.tell(ctx, req.ID, failedUpdate(NotKeptMessage))
	}
	reg := instanceapi.LogBundleRegistration{
		RequestId: req.ID,
		Since:     req.Since.UTC(),
		Until:     req.Until.UTC(),
		RequestedBy: instanceapi.LogRequester{
			Email: openapi_types.Email(req.RequestedBy.Email),
		},
	}
	if name := []rune(req.RequestedBy.Name); len(name) > 0 {
		trimmed := string(name[:min(len(name), maxNameSize)])
		reg.RequestedBy.Name = &trimmed
	}
	id, err := s.cp.RegisterLogBundle(ctx, reg)
	var problem *controlplane.ProblemError
	if errors.As(err, &problem) {
		switch problem.Status {
		case http.StatusConflict:
			message := problem.Detail
			if message == "" {
				message = conflictMessage
			}
			return s.tell(ctx, req.ID, failedUpdate(message))
		case http.StatusBadRequest:
			return s.tell(ctx, req.ID, failedUpdate(
				"Trenova support did not accept this log request: "+problem.Error(),
			))
		}
	}
	if err != nil {
		return fmt.Errorf("register the log request %s: %w", req.ID, err)
	}
	s.mu.Lock()
	s.requests[req.ID] = &request{bundle: id}
	s.mu.Unlock()
	if err = s.tell(ctx, req.ID, tms.LogRequestUpdate{
		Status: tms.LogRequestCollecting, BundleID: id.String(),
	}); err != nil {
		s.logger.WarnContext(ctx, "cannot tell the TMS a log request is being collected; collecting anyway",
			"request", req.ID, "error", err)
	}
	requestID := req.ID
	submitted := s.Submit(&Job{
		BundleID: id,
		Since:    req.Since.UTC(),
		Until:    req.Until.UTC(),
		Done:     func(o Outcome) { s.finishRequest(requestID, id, o) },
	})
	if submitted == QueueFull {
		s.mu.Lock()
		delete(s.requests, req.ID)
		s.mu.Unlock()
		s.logger.InfoContext(ctx, "the log bundle queue is full; the request waits for the next tick",
			"request", req.ID)
	}
	return nil
}

func (s *Service) finishRequest(requestID string, bundle uuid.UUID, o Outcome) {
	update := failedUpdate(o.Message)
	if o.Status == instanceapi.LogBundleStored {
		update = tms.LogRequestUpdate{Status: tms.LogRequestSent, BundleID: bundle.String()}
	}
	s.mu.Lock()
	s.requests[requestID] = &request{bundle: bundle, final: &update}
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), tmsTimeout)
	defer cancel()
	if err := s.tell(ctx, requestID, update); err != nil {
		s.logger.Warn("cannot tell the TMS how a log request ended; telling it again on the next tick",
			"request", requestID, "error", err)
	}
}

func (s *Service) tell(ctx context.Context, id string, update tms.LogRequestUpdate) error {
	err := s.tms.UpdateLogRequest(ctx, id, update)
	if errors.Is(err, tms.ErrLogRequestSettled) {
		s.logger.InfoContext(ctx, "the TMS had already settled a log request; leaving it as it is",
			"request", id, "status", string(update.Status))
		return nil
	}
	if err != nil {
		return fmt.Errorf("tell the TMS how the log request %s goes: %w", id, err)
	}
	return nil
}

func failedUpdate(message string) tms.LogRequestUpdate {
	if message == "" {
		message = "Trenova could not collect the logs."
	}
	return tms.LogRequestUpdate{Status: tms.LogRequestFailed, Message: health.Clean(message, maxMessage)}
}

func PeriodProblem(since, until time.Time, longest time.Duration) string {
	switch {
	case since.IsZero() || until.IsZero():
		return "The request does not say which period to collect."
	case !until.After(since):
		return "The period ends before it starts."
	case until.Sub(since) > longest:
		return fmt.Sprintf("The period is longer than the %d days of logs the instance keeps.",
			int(longest.Hours()/24))
	}
	return ""
}
