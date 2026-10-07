package shipmentsuggestionservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/customerupdateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	coverageHorizon    = 24 * time.Hour
	maxCoverageItems   = 5
	maxDelayItems      = 5
	maxDetentionItems  = 5
	hosWarnMs          = int64(2 * time.Hour / time.Millisecond)
	coverageCandidates = 3
	clockLayout        = "15:04"
	reviewLabel        = "Review"
	minutesPerHour     = 60
)

type ruleInput struct {
	tenant   pagination.TenantInfo
	timezone string
	loc      *time.Location
	now      time.Time
	caps     *services.ShipmentBoardCapabilities
	board    *dispatchconsoleservice.Board
}

func (in *ruleInput) clock(unix int64) string {
	return time.Unix(unix, 0).In(in.loc).Format(clockLayout)
}

type rule interface {
	name() string
	collect(ctx context.Context, s *Service, in *ruleInput) ([]*services.ShipmentSuggestion, error)
}

func defaultRules() []rule {
	return []rule{coverageRule{}, delayRule{}, hoursRule{}, detentionRule{}, retenderRule{}}
}

func (s *Service) compute(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
	loc *time.Location,
) (*queue, error) {
	caps, err := s.capabilities.Capabilities(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	board, err := s.console.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo:     tenantInfo,
		IncludeCovered: true,
	})
	if err != nil {
		return nil, err
	}

	in := &ruleInput{
		tenant:   tenantInfo,
		timezone: timezone,
		loc:      loc,
		now:      s.now(),
		caps:     caps,
		board:    board,
	}
	items := make([]*services.ShipmentSuggestion, 0, maxQueueItems)
	for _, r := range s.rules {
		found, ruleErr := r.collect(ctx, s, in)
		if ruleErr != nil {
			s.l.Warn("suggestion rule failed", zap.String("rule", r.name()), zap.Error(ruleErr))
			continue
		}
		items = append(items, found...)
	}

	slices.SortStableFunc(items, func(a, b *services.ShipmentSuggestion) int {
		return cmp.Or(
			cmp.Compare(toneRank(a.Tone), toneRank(b.Tone)),
			cmp.Compare(dueOf(a), dueOf(b)),
		)
	})
	if len(items) > maxQueueItems {
		items = items[:maxQueueItems]
	}

	return &queue{Items: items}, nil
}

func toneRank(tone services.SuggestionTone) int {
	switch tone {
	case services.SuggestionToneDanger:
		return 0
	case services.SuggestionToneWarning:
		return 1
	case services.SuggestionToneAccent:
		return 2
	case services.SuggestionToneBrand:
		return 3
	default:
		return 3
	}
}

func dueOf(item *services.ShipmentSuggestion) int64 {
	if item.DueAt == nil {
		return 1<<62 - 1
	}

	return *item.DueAt
}

func lane(move *dispatchconsoleservice.BoardMove) string {
	return move.OriginCity + " → " + move.DestinationCity
}

func money(value *float64) string {
	if value == nil {
		return ""
	}

	return "$" + decimal.NewFromFloat(*value).StringFixed(0)
}

func duration(minutes int) string {
	hours, rest := minutes/minutesPerHour, minutes%minutesPerHour
	switch {
	case hours == 0:
		return fmt.Sprintf("%dm", rest)
	case rest == 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dh %dm", hours, rest)
	}
}

func hoursClock(ms int64) string {
	minutes := ms / int64(time.Minute/time.Millisecond)

	return fmt.Sprintf("%d:%02d", minutes/minutesPerHour, minutes%minutesPerHour)
}

func ptr[T any](value T) *T { return &value }

func compactImpact(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}

	return out
}

type coverageRule struct{}

func (coverageRule) name() string { return "coverage" }

func (coverageRule) collect(
	ctx context.Context,
	s *Service,
	in *ruleInput,
) ([]*services.ShipmentSuggestion, error) {
	horizon := in.now.Add(coverageHorizon).Unix()
	open := make([]*dispatchconsoleservice.BoardMove, 0)
	for _, move := range in.board.Moves {
		if move == nil || move.IsCovered || move.LiveTender != nil ||
			move.OriginWindowStart > horizon {
			continue
		}
		open = append(open, move)
	}
	slices.SortStableFunc(open, func(a, b *dispatchconsoleservice.BoardMove) int {
		return cmp.Compare(a.OriginWindowStart, b.OriginWindowStart)
	})
	open = open[:min(maxCoverageItems, len(open))]

	operation := in.caps.OperationType
	out := make([]*services.ShipmentSuggestion, 0, len(open))
	for _, move := range open {
		var item *services.ShipmentSuggestion
		if operation.RunsAssets() {
			driver, err := s.bestDriver(ctx, in.tenant, move.MoveID)
			if err != nil {
				return nil, err
			}
			if driver != nil {
				item = assignSuggestion(in, move, driver)
			}
		}
		if item == nil && operation.RunsBrokerage() {
			item = tenderSuggestion(in, move)
		}
		if item == nil {
			item = uncoverableSuggestion(in, move)
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *Service) bestDriver(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID pulid.ID,
) (*dispatchcandidateservice.CandidateScore, error) {
	ranked, err := s.console.GetMoveCandidates(ctx, &dispatchconsoleservice.MoveCandidatesRequest{
		TenantInfo: tenant,
		MoveID:     moveID,
		Limit:      coverageCandidates,
	})
	if err != nil {
		return nil, err
	}
	for _, candidate := range ranked {
		if candidate != nil && !candidate.Blocked() {
			return candidate, nil
		}
	}

	return nil, nil //nolint:nilnil // no driver can take the move
}

func coverageTone(in *ruleInput, move *dispatchconsoleservice.BoardMove) services.SuggestionTone {
	if move.OriginWindowStart < in.now.Unix() {
		return services.SuggestionToneDanger
	}

	return services.SuggestionToneWarning
}

func pickupDue(move *dispatchconsoleservice.BoardMove) *int64 {
	if move.OriginWindowEnd != nil && *move.OriginWindowEnd > 0 {
		return ptr(*move.OriginWindowEnd)
	}

	return ptr(move.OriginWindowStart)
}

func assignSuggestion(
	in *ruleInput,
	move *dispatchconsoleservice.BoardMove,
	driver *dispatchcandidateservice.CandidateScore,
) *services.ShipmentSuggestion {
	first := stringutils.FirstName(driver.WorkerName)
	deadhead := ""
	if driver.DeadheadMiles != nil {
		deadhead = fmt.Sprintf("%.0f mi out", *driver.DeadheadMiles)
	}
	reason := assignReason(in, deadhead, driver.DriveRemainingMs, *pickupDue(move))

	return &services.ShipmentSuggestion{
		Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindCoverage, move.MoveID).
			String(),
		Kind:       services.SuggestionCoverage,
		Tone:       coverageTone(in, move),
		ShipmentID: move.ShipmentID,
		ProNumber:  move.ProNumber,
		Title:      fmt.Sprintf("Assign %s to %s", driver.WorkerName, lane(move)),
		Reason:     reason,
		Impact: compactImpact(
			money(move.Revenue),
			deadhead,
			fmt.Sprintf("%d%% fit", driver.Score),
		),
		Primary: services.SuggestionAction{
			Type:      services.SuggestionActionAssignDriver,
			Label:     "Assign " + first,
			MoveID:    move.MoveID,
			WorkerID:  driver.WorkerID,
			TractorID: driver.TractorID,
		},
		ManualLabel: "Assign a driver",
		DueAt:       pickupDue(move),
	}
}

func assignReason(in *ruleInput, deadhead string, driveRemainingMs, pickupDue int64) string {
	parts := make([]string, 0, 2)
	if deadhead != "" {
		parts = append(parts, deadhead)
	}
	if in.caps.HOS && driveRemainingMs > 0 {
		hours := duration(int(driveRemainingMs / int64(time.Minute/time.Millisecond)))
		parts = append(parts, "with "+hours+" of drive time left")
	}
	closes := "Pickup closes at " + in.clock(pickupDue) + "."
	if len(parts) == 0 {
		return closes
	}

	return stringutils.CapitalizeFirst(strings.Join(parts, " ")) + ". " + closes
}

func tenderSuggestion(
	in *ruleInput,
	move *dispatchconsoleservice.BoardMove,
) *services.ShipmentSuggestion {
	return &services.ShipmentSuggestion{
		Key:        shipmentsuggestion.NewKey(shipmentsuggestion.KindTender, move.MoveID).String(),
		Kind:       services.SuggestionTender,
		Tone:       coverageTone(in, move),
		ShipmentID: move.ShipmentID,
		ProNumber:  move.ProNumber,
		Title:      "Tender " + lane(move),
		Reason: fmt.Sprintf(
			"Nothing covers it yet and pickup closes at %s. It goes to the routing guide, or the best-priced carrier when no guide matches.",
			in.clock(*pickupDue(move)),
		),
		Impact: compactImpact(money(move.Revenue), "pickup "+in.clock(move.OriginWindowStart)),
		Primary: services.SuggestionAction{
			Type:   services.SuggestionActionTenderCarrier,
			Label:  "Tender",
			MoveID: move.MoveID,
		},
		ManualLabel: "Tender to a carrier",
		DueAt:       pickupDue(move),
	}
}

func uncoverableSuggestion(
	in *ruleInput,
	move *dispatchconsoleservice.BoardMove,
) *services.ShipmentSuggestion {
	return &services.ShipmentSuggestion{
		Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindCoverage, move.MoveID).
			String(),
		Kind:       services.SuggestionCoverage,
		Tone:       coverageTone(in, move),
		ShipmentID: move.ShipmentID,
		ProNumber:  move.ProNumber,
		Title: fmt.Sprintf(
			"%s needs a driver",
			stringutils.WithDefault(move.ProNumber, lane(move)),
		),
		Reason: fmt.Sprintf(
			"No driver is free in time for the pickup at %s.",
			in.clock(move.OriginWindowStart),
		),
		Impact: compactImpact(money(move.Revenue)),
		Primary: services.SuggestionAction{
			Type:   services.SuggestionActionReview,
			Label:  reviewLabel,
			MoveID: move.MoveID,
		},
		ManualLabel: "Find a driver",
		DueAt:       pickupDue(move),
	}
}

type delayRule struct{}

func (delayRule) name() string { return "delay" }

func (delayRule) collect(
	ctx context.Context,
	s *Service,
	in *ruleInput,
) ([]*services.ShipmentSuggestion, error) {
	watch, err := s.watchlist.Watchlist(ctx, in.tenant, in.timezone)
	if err != nil {
		return nil, err
	}

	out := make([]*services.ShipmentSuggestion, 0, maxDelayItems)
	for _, late := range watch.Deliveries.WorstLate {
		if len(out) == maxDelayItems {
			break
		}
		told, tellErr := customerupdateservice.AlreadyTold(
			ctx, s.comments, in.tenant, late.ShipmentID, in.now.Unix(),
		)
		if tellErr != nil {
			return nil, tellErr
		}
		if told {
			continue
		}

		behind := duration(late.DeltaMinutes)
		customer := stringutils.WithDefault(late.CustomerName, "the customer")
		out = append(out, &services.ShipmentSuggestion{
			Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindDelayNotice, late.ShipmentID).
				String(),
			Kind:       services.SuggestionDelayNotice,
			Tone:       services.SuggestionToneDanger,
			ShipmentID: late.ShipmentID,
			ProNumber:  late.ProNumber,
			Title:      fmt.Sprintf("Let %s know about the delay", customer),
			Reason: fmt.Sprintf(
				"%s will reach %s about %s after its delivery appointment.",
				late.ProNumber, late.City, behind,
			),
			Impact: compactImpact("+" + behind),
			Primary: services.SuggestionAction{
				Type:  services.SuggestionActionNotifyCustomer,
				Label: "Send update",
				Message: fmt.Sprintf(
					"Shipment %s to %s is running about %s behind its delivery appointment. "+
						"We are watching it closely and will update you as it gets closer.",
					late.ProNumber, late.City, behind,
				),
			},
			ManualLabel: "Email the customer",
		})
	}

	return out, nil
}

type hoursRule struct{}

func (hoursRule) name() string { return "hours-of-service" }

func (hoursRule) collect(
	_ context.Context,
	_ *Service,
	in *ruleInput,
) ([]*services.ShipmentSuggestion, error) {
	if !in.caps.HOS || !in.caps.OperationType.RunsAssets() {
		return nil, nil
	}

	moveByWorker := make(map[pulid.ID]*dispatchconsoleservice.BoardMove, len(in.board.Moves))
	for _, move := range in.board.Moves {
		if move != nil && move.AssignedWorkerID.IsNotNil() {
			moveByWorker[move.AssignedWorkerID] = move
		}
	}

	out := make([]*services.ShipmentSuggestion, 0)
	for _, driver := range in.board.Drivers {
		if driver == nil || driver.BoardDriver == nil || driver.HOSIsStale ||
			driver.HOSRecordedAt == 0 || driver.DriveRemainingMs >= hosWarnMs ||
			driver.Availability != dispatchconsoleservice.AvailabilityWorking {
			continue
		}
		move, ok := moveByWorker[driver.WorkerID]
		if !ok {
			continue
		}
		name := strings.TrimSpace(driver.FirstName + " " + driver.LastName)
		out = append(out, &services.ShipmentSuggestion{
			Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindHoursOfService, move.MoveID).
				String(),
			Kind:       services.SuggestionHoursOfService,
			Tone:       services.SuggestionToneWarning,
			ShipmentID: move.ShipmentID,
			ProNumber:  move.ProNumber,
			Title:      fmt.Sprintf("%s runs out of hours before %s", name, move.DestinationCity),
			Reason: fmt.Sprintf(
				"%s of drive time left on %s.",
				hoursClock(driver.DriveRemainingMs), move.ProNumber,
			),
			Impact: compactImpact(hoursClock(driver.DriveRemainingMs) + " HOS"),
			Primary: services.SuggestionAction{
				Type:     services.SuggestionActionReview,
				Label:    reviewLabel,
				MoveID:   move.MoveID,
				WorkerID: driver.WorkerID,
			},
			ManualLabel: "Plan a relay",
			DueAt: ptr(
				in.now.Unix() + driver.DriveRemainingMs/int64(time.Second/time.Millisecond),
			),
		})
	}

	return out, nil
}

type detentionRule struct{}

func (detentionRule) name() string { return "detention" }

func (detentionRule) collect(
	ctx context.Context,
	s *Service,
	in *ruleInput,
) ([]*services.ShipmentSuggestion, error) {
	if s.detention == nil {
		return nil, nil
	}
	desk, err := s.detention.ListDesk(ctx, in.tenant)
	if err != nil {
		return nil, err
	}

	out := make([]*services.ShipmentSuggestion, 0, maxDetentionItems)
	for _, entry := range desk {
		if len(out) == maxDetentionItems {
			break
		}
		occurrence := entry.Occurrence
		if occurrence == nil || occurrence.Status != detention.OccurrenceStatusPending {
			continue
		}
		amount := "$" + occurrence.BillableAmount.StringFixed(0)
		facility := stringutils.WithDefault(occurrence.LocationName, "the stop")
		out = append(out, &services.ShipmentSuggestion{
			Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindDetention, occurrence.ID).
				String(),
			Kind:       services.SuggestionDetention,
			Tone:       services.SuggestionToneAccent,
			ShipmentID: occurrence.ShipmentID,
			ProNumber:  occurrence.ShipmentProNumber,
			Title:      "Approve detention at " + facility,
			Reason: fmt.Sprintf(
				"%s of billable time on %s is waiting for approval.",
				duration(int(occurrence.BillableMinutes)), occurrence.ShipmentProNumber,
			),
			Impact: compactImpact(amount),
			Primary: services.SuggestionAction{
				Type:                  services.SuggestionActionApproveDetention,
				Label:                 "Approve " + amount,
				DetentionOccurrenceID: occurrence.ID,
			},
			ManualLabel: "Review detention",
		})
	}

	return out, nil
}

type retenderRule struct{}

func (retenderRule) name() string { return "retender" }

func (retenderRule) collect(
	_ context.Context,
	_ *Service,
	in *ruleInput,
) ([]*services.ShipmentSuggestion, error) {
	if !in.caps.OperationType.RunsBrokerage() {
		return nil, nil
	}

	out := make([]*services.ShipmentSuggestion, 0)
	for _, move := range in.board.Moves {
		if move == nil || move.LiveTender == nil ||
			move.LiveTender.Status != tender.StatusNeedsReview {
			continue
		}
		out = append(out, &services.ShipmentSuggestion{
			Key: shipmentsuggestion.NewKey(shipmentsuggestion.KindRetender, move.MoveID).
				String(),
			Kind:       services.SuggestionRetender,
			Tone:       services.SuggestionToneDanger,
			ShipmentID: move.ShipmentID,
			ProNumber:  move.ProNumber,
			Title:      fmt.Sprintf("The tender for %s needs a person", lane(move)),
			Reason: fmt.Sprintf(
				"Its offers ran out without an acceptance, and pickup is at %s.",
				in.clock(move.OriginWindowStart),
			),
			Impact: compactImpact(money(move.Revenue)),
			Primary: services.SuggestionAction{
				Type:   services.SuggestionActionReview,
				Label:  reviewLabel,
				MoveID: move.MoveID,
			},
			ManualLabel: "Review the tender",
			DueAt:       pickupDue(move),
		})
	}

	return out, nil
}
