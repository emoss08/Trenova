package triagepeekservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	customerPeekShipments = 5
	customerPeekMessages  = 5
)

var errNoShipmentReader = errortypes.NewBusinessError(
	"Shipments cannot be read here on this installation",
)

type Params struct {
	fx.In

	Messages  inboundmessageports.InboundMessageRepository
	Work      inboundmessageports.TriageWorkRepository
	Customers customerports.CustomerRepository
	Orgs      tenantports.OrganizationRepository
	Peeks     inboundmessageports.TriageShipmentPeeker  `optional:"true"`
	Stops     inboundmessageports.TriageStopTimesReader `optional:"true"`
}

type Service struct {
	messages  inboundmessageports.InboundMessageRepository
	work      inboundmessageports.TriageWorkRepository
	customers customerports.CustomerRepository
	orgs      tenantports.OrganizationRepository
	peeks     inboundmessageports.TriageShipmentPeeker
	stops     inboundmessageports.TriageStopTimesReader
	now       func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		messages:  p.Messages,
		work:      p.Work,
		customers: p.Customers,
		orgs:      p.Orgs,
		peeks:     p.Peeks,
		stops:     p.Stops,
		now:       timeutils.NowUnix,
	}
}

func (s *Service) ShipmentPeek(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*inboundmessageports.ShipmentPeek, error) {
	peeks, err := s.shipmentPeeks(ctx, tenantInfo, []pulid.ID{shipmentID})
	if err != nil {
		return nil, err
	}

	peek, ok := peeks[shipmentID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Shipment not found")
	}

	return peek, nil
}

func (s *Service) shipmentPeeks(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*inboundmessageports.ShipmentPeek, error) {
	if s.peeks == nil {
		return nil, errNoShipmentReader
	}
	if len(shipmentIDs) == 0 {
		return map[pulid.ID]*inboundmessageports.ShipmentPeek{}, nil
	}

	peeks, err := s.peeks.ShipmentPeeks(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}

	onFile, err := s.work.DocumentTypesOnShipments(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, err
	}
	for id, peek := range peeks {
		peek.DocumentsOnFile = onFile[id]
		if peek.DocumentsOnFile == nil {
			peek.DocumentsOnFile = []string{}
		}
		if peek.Stops == nil {
			peek.Stops = []*inboundmessageports.PeekStop{}
		}
	}

	return peeks, nil
}

type CustomerPeekRequest struct {
	TenantInfo    pagination.TenantInfo
	CustomerID    pulid.ID
	WithShipments bool
}

func (s *Service) CustomerPeek(
	ctx context.Context,
	req *CustomerPeekRequest,
) (*inboundmessageports.CustomerPeek, error) {
	entity, err := s.customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         req.CustomerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	triage, err := s.work.CustomerTriage(ctx, &inboundmessageports.CustomerTriageRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: req.CustomerID,
		Limit:      customerPeekMessages,
	})
	if err != nil {
		return nil, err
	}

	peek := &inboundmessageports.CustomerPeek{
		ID:              entity.ID,
		Code:            entity.Code,
		Name:            entity.Name,
		Since:           entity.CreatedAt,
		WaitingInTriage: triage.Waiting,
		Shipments:       []*inboundmessageports.ShipmentPeek{},
		Messages:        triage.Recent,
	}
	if !req.WithShipments || s.peeks == nil {
		return peek, nil
	}

	_, location, err := triagenova.Organization(ctx, s.orgs, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	recent, err := s.peeks.CustomerShipments(
		ctx,
		&inboundmessageports.TriageCustomerShipmentsRequest{
			TenantInfo: req.TenantInfo,
			CustomerID: req.CustomerID,
			WeekStart:  timeutils.WeekStart(s.now(), location),
			Limit:      customerPeekShipments,
		},
	)
	if err != nil {
		return nil, err
	}
	peek.ShipmentsThisWeek = recent.ThisWeek

	peeks, err := s.shipmentPeeks(ctx, req.TenantInfo, recent.ShipmentIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range recent.ShipmentIDs {
		if shipment, ok := peeks[id]; ok {
			peek.Shipments = append(peek.Shipments, shipment)
		}
	}

	return peek, nil
}

func (s *Service) StopTimes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessageports.StopTimes, error) {
	message, err := s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         messageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if message.MatchedShipmentID.IsNil() || s.stops == nil {
		return nil, nil //nolint:nilnil // no shipment is linked, so there are no stop times
	}

	return s.stops.StopTimes(ctx, tenantInfo, message.MatchedShipmentID)
}

func (s *Service) Readings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentIDs []pulid.ID,
) (map[pulid.ID]*inboundmessageports.DocumentReading, error) {
	contents, err := s.work.ContentsByDocumentIDs(ctx, tenantInfo, documentIDs)
	if err != nil {
		return nil, err
	}

	readings := make(map[pulid.ID]*inboundmessageports.DocumentReading, len(contents))
	for id, content := range contents {
		if reading := ReadingOf(content); reading != nil {
			readings[id] = reading
		}
	}

	return readings, nil
}

func ReadingOf(content *documentcontent.Content) *inboundmessageports.DocumentReading {
	if content == nil || !content.Status.Settled() ||
		content.Status == documentcontent.StatusFailed {
		return nil
	}

	findings := content.Findings()
	kind := content.DetectedDocumentKind
	if findings != nil && strings.TrimSpace(findings.Kind) != "" {
		kind = findings.Kind
	}
	if strings.TrimSpace(kind) == "" && findings == nil {
		return nil
	}

	reading := &inboundmessageports.DocumentReading{
		DocumentType: HumanizeKind(kind),
		Confidence:   content.ClassificationConfidence,
		PageCount:    content.PageCount,
		Fields:       []*inboundmessageports.DocumentField{},
	}
	if findings == nil {
		return reading
	}

	for _, field := range findings.Fields {
		tone := inboundmessage.FactToneNeutral
		if field.ReviewRequired {
			tone = inboundmessage.FactToneWarning
		}
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = HumanizeKind(field.Key)
		}
		reading.Fields = append(reading.Fields, &inboundmessageports.DocumentField{
			Label: label,
			Value: field.Value,
			Tone:  tone,
		})
	}

	return reading
}

func HumanizeKind(kind string) string {
	trimmed := strings.TrimSpace(kind)
	if trimmed == "" {
		return ""
	}
	if strings.ContainsAny(trimmed, "_-. ") {
		return stringutils.CapitalizeFirst(strings.ToLower(stringutils.HumanizeSnakeCase(trimmed)))
	}

	return stringutils.HumanizeCamelCaseSentence(trimmed)
}
