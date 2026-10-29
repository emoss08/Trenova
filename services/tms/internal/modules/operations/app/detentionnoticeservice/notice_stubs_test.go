package detentionnoticeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
)

// stubNoticeRepo records what the send path filed. The detention repositories
// have no generated mocks and the project forbids running mockery, so the two
// calls under test are captured by hand.
type stubNoticeRepo struct {
	attached  []*detentionports.AttachNoticePDFDocumentRequest
	attachErr error
}

func (r *stubNoticeRepo) Create(
	_ context.Context,
	entity *detentionnotice.DetentionNotice,
) (*detentionnotice.DetentionNotice, error) {
	return entity, nil
}

func (r *stubNoticeRepo) AttachPDFDocument(
	_ context.Context,
	req *detentionports.AttachNoticePDFDocumentRequest,
) error {
	r.attached = append(r.attached, req)
	return r.attachErr
}

func (r *stubNoticeRepo) Update(
	_ context.Context,
	entity *detentionnotice.DetentionNotice,
) (*detentionnotice.DetentionNotice, error) {
	return entity, nil
}

func (r *stubNoticeRepo) List(
	_ context.Context,
	_ *detentionports.ListDetentionNoticesRequest,
) ([]*detentionnotice.DetentionNotice, error) {
	return nil, nil
}

func (r *stubNoticeRepo) ListQueued(
	_ context.Context,
	_ *detentionports.ListNoticesDueRequest,
) ([]*detentionnotice.DetentionNotice, error) {
	return nil, nil
}

func pdfOccurrence() *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:                pulid.MustNew("dto_"),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		ShipmentID:        pulid.MustNew("shp_"),
		ShipmentProNumber: "SHP-1001",
		Currency:          "USD",
	}
}
