package documentquerytools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRecordLabels struct {
	labels []*captureports.CaptureRecordLabel
	err    error
	asked  []*captureports.ListCaptureRecordLabelsRequest
}

func (s *stubRecordLabels) Labels(
	_ context.Context,
	req *captureports.ListCaptureRecordLabelsRequest,
) ([]*captureports.CaptureRecordLabel, error) {
	s.asked = append(s.asked, req)

	return s.labels, s.err
}

type stubUploader struct {
	users []*tenant.User
	err   error
	asked []tenantports.GetUsersByIDsRequest
}

func (s *stubUploader) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	s.asked = append(s.asked, req)

	return s.users, s.err
}

func noLabels() *stubRecordLabels { return &stubRecordLabels{} }

func noUploader() *stubUploader { return &stubUploader{} }

func uploadedDocument(profile document.ProcessingProfile) *document.Document {
	doc := summaryDocument()
	doc.OriginalName = "405515639Rate Confirmation.pdf"
	doc.ProcessingProfile = profile
	doc.UploadedByID = pulid.MustNew("usr_")
	doc.CreatedAt = 1_790_000_000

	return doc
}

func readOrigin(
	t *testing.T,
	doc *document.Document,
	labels *stubRecordLabels,
	users *stubUploader,
) (*documentSummary, error) {
	t.Helper()

	tool := NewGetDocumentTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			ContentText: "Rate confirmation",
		}},
		labels,
		users,
		&querytooltest.FakePermissions{Allowed: true},
	)
	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	if err != nil {
		return nil, err
	}

	return result.(*documentSummary), nil
}

// "Who sent this" is answered from how the file came in: a file someone
// uploaded names them, rather than leaving the model to search the inbox for
// a sender.
func TestGetDocument_NamesWhoUploadedItAndTheRecordItIsOn(t *testing.T) {
	t.Parallel()

	doc := uploadedDocument(document.ProcessingProfileNone)
	ownerID, err := pulid.Parse(doc.ResourceID)
	require.NoError(t, err)
	labels := &stubRecordLabels{labels: []*captureports.CaptureRecordLabel{
		{ResourceType: "shipment", ID: ownerID, Title: "S1000042", Subtitle: "BOL-7"},
	}}
	users := &stubUploader{users: []*tenant.User{{ID: doc.UploadedByID, Name: "Dana Reyes"}}}

	summary, err := readOrigin(t, doc, labels, users)

	require.NoError(t, err)
	assert.Equal(t, "Dana Reyes", summary.UploadedBy)
	assert.Equal(t, int64(1_790_000_000), summary.UploadedAt)
	assert.Equal(t, "uploaded by a person", summary.CameInThrough)
	assert.Equal(t, "shipment "+doc.ResourceID, summary.AttachedTo)
	assert.Equal(t, "S1000042", summary.AttachedLabel)
	require.Len(t, labels.asked, 1)
	assert.Equal(t, "shipment", labels.asked[0].ResourceType)
	assert.Equal(t, []pulid.ID{ownerID}, labels.asked[0].IDs)
	require.Len(t, users.asked, 1)
	assert.Equal(t, []pulid.ID{doc.UploadedByID}, users.asked[0].UserIDs)
}

func TestGetDocument_SaysHowEachKindOfIntakeCameIn(t *testing.T) {
	t.Parallel()

	cases := map[document.ProcessingProfile]string{
		document.ProcessingProfileNone:                   "uploaded by a person",
		document.ProcessingProfileRateConfirmationImport: "uploaded by a person through the rate confirmation import",
		document.ProcessingProfileAssistantAttachment:    "attached by a person to a conversation with Nova",
		document.ProcessingProfileInboundAttachment:      "an attachment on an email to a monitored mailbox",
		document.ProcessingProfileCapture:                "scanned or printed into Trenova",
		document.ProcessingProfileFactoringRemittance:    "a factor's remittance",
		document.ProcessingProfileShipmentPaperwork:      "uploaded by a person as a shipment's paperwork",
	}
	for profile, want := range cases {
		summary, err := readOrigin(t, uploadedDocument(profile), noLabels(), noUploader())
		require.NoError(t, err, profile)
		assert.Equal(t, want, summary.CameInThrough, profile)
	}
}

func TestGetDocument_LeavesAnUploaderNoLongerInTheTenantUnnamed(t *testing.T) {
	t.Parallel()

	summary, err := readOrigin(
		t, uploadedDocument(document.ProcessingProfileNone), noLabels(), noUploader(),
	)

	require.NoError(t, err)
	assert.Empty(t, summary.UploadedBy)
	assert.Equal(t, "uploaded by a person", summary.CameInThrough)
}

func TestGetDocument_SurfacesAFailureToReadTheUploader(t *testing.T) {
	t.Parallel()

	_, err := readOrigin(
		t,
		uploadedDocument(document.ProcessingProfileNone),
		noLabels(),
		&stubUploader{err: errors.New("connection reset")},
	)

	require.ErrorContains(t, err, "read who uploaded document")
	require.ErrorContains(t, err, "connection reset")
}

func TestGetDocument_SurfacesAFailureToNameTheRecord(t *testing.T) {
	t.Parallel()

	_, err := readOrigin(
		t,
		uploadedDocument(document.ProcessingProfileNone),
		&stubRecordLabels{err: errors.New("connection reset")},
		noUploader(),
	)

	require.ErrorContains(t, err, "name the shipment document")
	require.ErrorContains(t, err, "connection reset")
}

// A file attached to a conversation is not on a record anyone files paperwork
// onto, so there is no record to name and no label is asked for; who attached
// it is still named.
func TestGetDocument_AsksNoLabelForAFileOnAConversation(t *testing.T) {
	t.Parallel()

	owner := querytooltest.PersonParams(permissionports.PrincipalTypeUser)
	doc := conversationDocument(owner.Actor.UserID)
	doc.ProcessingProfile = document.ProcessingProfileAssistantAttachment
	owner.Params = map[string]any{"documentId": doc.ID.String()}
	labels := noLabels()
	users := &stubUploader{users: []*tenant.User{{ID: owner.Actor.UserID, Name: "Dana Reyes"}}}

	tool := NewGetDocumentTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			ContentText: "Rate confirmation for load 4471",
		}},
		labels,
		users,
		&querytooltest.FakePermissions{Allowed: true},
	)
	result, err := tool.Query(t.Context(), owner)

	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Empty(t, labels.asked)
	assert.Empty(t, summary.AttachedLabel)
	assert.Equal(t, "Dana Reyes", summary.UploadedBy)
	assert.Equal(t, "attached by a person to a conversation with Nova", summary.CameInThrough)
}

// The command palette's get_record opens a kind through the read whose name,
// in reading order, is get_<kind>; the document read must be that read.
func TestGetDocument_ReadsAsTheDocumentKindsGet(t *testing.T) {
	t.Parallel()

	tool := NewGetDocumentTool(
		&stubDocumentRepo{},
		&stubContentReader{},
		noLabels(),
		noUploader(),
		&querytooltest.FakePermissions{Allowed: true},
	)

	assert.Equal(t, "get_document", agenttoolports.ReadingOrder(tool.Name()))
}
