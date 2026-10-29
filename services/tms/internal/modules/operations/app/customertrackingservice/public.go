package customertrackingservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const maxPublicNotes = 20

type resolvedLink struct {
	ctx      context.Context
	link     *customertracking.Link
	tenant   pagination.TenantInfo
	shipment *shipment.Shipment
	profile  *customertracking.Profile
}

func invalidLinkError() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This tracking link is no longer valid",
	)
}

// resolveToken turns a raw token into the shipment it shows. Every way a link
// can fail (unknown, revoked, expired, the shipment gone or moved to another
// customer, links turned off for the customer) answers the same, so a guessed
// token learns nothing.
func (s *Service) resolveToken(ctx context.Context, token string) (*resolvedLink, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, invalidLinkError()
	}
	link, err := s.links.GetByHash(ctx, tokenutils.Hash(token))
	if err != nil {
		return nil, err
	}
	if !link.IsUsable(s.now()) {
		return nil, invalidLinkError()
	}

	tenant := pagination.TenantInfo{OrgID: link.OrganizationID, BuID: link.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	sh, err := s.loadShipment(ctx, tenant, link.ShipmentID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, invalidLinkError()
		}
		return nil, err
	}
	if sh.CustomerID != link.CustomerID {
		return nil, invalidLinkError()
	}
	profile, err := s.profileOrDefault(ctx, tenant, sh.CustomerID)
	if err != nil {
		return nil, err
	}
	if !profile.LinkEnabled {
		return nil, invalidLinkError()
	}
	return &resolvedLink{ctx: ctx, link: link, tenant: tenant, shipment: sh, profile: profile}, nil
}

func (s *Service) PreviewByToken(
	ctx context.Context,
	token string,
) (*customertrackingports.PublicTrackingView, error) {
	resolved, err := s.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	view, err := s.buildView(resolved.ctx, resolved, resolved.link.ExpiresAt)
	if err != nil {
		return nil, err
	}

	if touchErr := s.links.RecordView(
		resolved.ctx,
		resolved.tenant,
		resolved.link.ID,
		s.now(),
	); touchErr != nil {
		s.l.Warn("failed to record customer tracking link view",
			zap.String("linkId", resolved.link.ID.String()),
			zap.Error(touchErr))
	}
	return view, nil
}

func (s *Service) PreviewForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.PublicTrackingView, error) {
	sh, err := s.loadShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return nil, err
	}
	profile, err := s.profileOrDefault(ctx, tenantInfo, sh.CustomerID)
	if err != nil {
		return nil, err
	}
	resolved := &resolvedLink{ctx: ctx, tenant: tenantInfo, shipment: sh, profile: profile}
	return s.buildView(ctx, resolved, s.now()+profile.LinkTTLSeconds())
}

func (s *Service) buildView(
	ctx context.Context,
	resolved *resolvedLink,
	expiresAt int64,
) (*customertrackingports.PublicTrackingView, error) {
	sh, profile := resolved.shipment, resolved.profile

	etas, err := s.etasByStop(ctx, resolved.tenant, sh.ID, profile)
	if err != nil {
		return nil, err
	}
	view := &customertrackingports.PublicTrackingView{
		CompanyName:       s.companyName(ctx, resolved.tenant),
		ShipmentProNumber: sh.ProNumber,
		Fields:            profile.VisibleFields,
		ExpiresAt:         expiresAt,
		Closed:            isClosed(sh),
	}
	if view.Fields == nil {
		view.Fields = []customertracking.Field{}
	}
	view.Stops = publicStops(route(sh), etas, profile)
	if profile.Shows(customertracking.FieldStatus) {
		view.Status = publicStatus(sh, etas)
	}
	if profile.Shows(customertracking.FieldReferences) {
		view.BOL = sh.BOL
		view.CustomerReference = sh.ExternalReference
	}
	if profile.Shows(customertracking.FieldFreight) {
		view.Pieces, view.Weight = sh.Pieces, sh.Weight
	}
	if err = s.fillOptionalParts(ctx, resolved, view); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) fillOptionalParts(
	ctx context.Context,
	resolved *resolvedLink,
	view *customertrackingports.PublicTrackingView,
) error {
	tenant, sh, profile := resolved.tenant, resolved.shipment, resolved.profile
	if profile.Shows(customertracking.FieldCarrier) ||
		profile.Shows(customertracking.FieldEquipment) {
		descriptors, err := s.facts.Descriptors(ctx, tenant, sh.ID)
		if err != nil {
			return err
		}
		if profile.Shows(customertracking.FieldCarrier) {
			view.CarrierName = descriptors.CarrierName
		}
		if profile.Shows(customertracking.FieldEquipment) {
			view.Equipment = descriptors.Equipment
		}
	}
	if profile.Shows(customertracking.FieldLastLocation) && !view.Closed {
		location, err := s.facts.LastReportedLocation(ctx, tenant, sh.ID)
		if err != nil {
			return err
		}
		view.LastLocation = location
	}
	if profile.Shows(customertracking.FieldHolds) {
		holds, err := s.facts.CustomerVisibleHolds(ctx, tenant, sh.ID)
		if err != nil {
			return err
		}
		view.Holds = holds
	}
	if profile.Shows(customertracking.FieldNotes) {
		notes, err := s.facts.CustomerVisibleNotes(ctx, tenant, sh.ID, maxPublicNotes)
		if err != nil {
			return err
		}
		view.Notes = notes
	}
	if profile.Shows(customertracking.FieldProofOfDelivery) {
		pods, err := s.facts.LatestPods(ctx, tenant, []pulid.ID{sh.ID})
		if err != nil {
			return err
		}
		if pod := pods[sh.ID]; pod != nil {
			view.ProofOfDelivery = &customertrackingports.PublicPod{
				DocumentID: pod.DocumentID,
				FileName:   pod.FileName,
				FiledAt:    pod.FiledAt,
			}
		}
	}
	return nil
}

func (s *Service) etasByStop(
	ctx context.Context,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
	profile *customertracking.Profile,
) (map[pulid.ID]*stopeta.StopETA, error) {
	result := make(map[pulid.ID]*stopeta.StopETA)
	if !profile.Shows(customertracking.FieldEta) && !profile.Shows(customertracking.FieldStatus) {
		return result, nil
	}
	rows, err := s.stopEtas.ListByShipmentIDs(ctx, tenant, []pulid.ID{shipmentID})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.StopID] = row
	}
	return result, nil
}

func publicStops(
	stops []routeStop,
	etas map[pulid.ID]*stopeta.StopETA,
	profile *customertracking.Profile,
) []customertrackingports.PublicStop {
	result := make([]customertrackingports.PublicStop, 0, len(stops))
	for i, rs := range stops {
		stop := rs.stop
		city, state := place(stop)
		row := customertrackingports.PublicStop{
			Sequence:  i + 1,
			Type:      string(stop.Type),
			City:      city,
			StateCode: state,
		}
		if profile.Shows(customertracking.FieldLocationNames) {
			if stop.Location != nil {
				row.LocationName = stop.Location.Name
			}
			row.AddressLine = stop.AddressLine
		}
		if profile.Shows(customertracking.FieldWindows) {
			start := stop.ScheduledWindowStart
			row.WindowStart = &start
			row.WindowEnd = stop.ScheduledWindowEnd
		}
		if profile.Shows(customertracking.FieldActuals) {
			row.ArrivedAt = stop.ActualArrival
			row.DepartedAt = stop.ActualDeparture
		}
		if profile.Shows(customertracking.FieldEta) && stop.ActualArrival == nil {
			if eta := etas[stop.ID]; eta != nil && !eta.Arrived {
				row.EstimatedArrival = eta.EstimatedArrival
				if eta.Verdict != stopeta.VerdictUnknown {
					row.Running = string(eta.Verdict)
				}
			}
		}
		result = append(result, row)
	}
	return result
}

func publicStatus(
	sh *shipment.Shipment,
	etas map[pulid.ID]*stopeta.StopETA,
) customertrackingports.PublicStatus {
	switch sh.Status {
	case shipmentvocab.StatusCanceled:
		return customertrackingports.PublicStatusCanceled
	case shipmentvocab.StatusCompleted,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusInvoiced:
		return customertrackingports.PublicStatusDelivered
	case shipmentvocab.StatusDelayed:
		return customertrackingports.PublicStatusDelayed
	case shipmentvocab.StatusNew,
		shipmentvocab.StatusPartiallyAssigned,
		shipmentvocab.StatusAssigned,
		shipmentvocab.StatusInTransit,
		shipmentvocab.StatusPartiallyCompleted:
	}
	for _, eta := range etas {
		if eta.NeedsAttention() {
			return customertrackingports.PublicStatusDelayed
		}
	}
	if sh.Status == shipmentvocab.StatusInTransit ||
		sh.Status == shipmentvocab.StatusPartiallyCompleted {
		return customertrackingports.PublicStatusInTransit
	}
	return customertrackingports.PublicStatusPlanned
}

func (s *Service) DownloadPodByToken(
	ctx context.Context,
	token string,
) (*customertrackingports.DownloadResult, error) {
	resolved, err := s.resolveToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !resolved.profile.Shows(customertracking.FieldProofOfDelivery) {
		return nil, invalidLinkError()
	}
	if s.documents == nil {
		return nil, errortypes.NewBusinessError("Document downloads are not configured")
	}
	ctx = resolved.ctx
	pods, err := s.facts.LatestPods(ctx, resolved.tenant, []pulid.ID{resolved.shipment.ID})
	if err != nil {
		return nil, err
	}
	pod := pods[resolved.shipment.ID]
	if pod == nil {
		return nil, errortypes.NewNotFoundError("The proof of delivery is not on file yet")
	}
	content, err := s.documents.GetDownloadContent(ctx, documentports.GetDocumentByIDRequest{
		ID:         pod.DocumentID,
		TenantInfo: resolved.tenant,
	})
	if err != nil {
		return nil, err
	}
	return &customertrackingports.DownloadResult{
		FileName:           content.Document.OriginalName,
		ContentType:        content.ContentType,
		ContentLength:      content.ContentLength,
		ContentDisposition: content.ContentDisposition,
		Body:               content.Body,
	}, nil
}
