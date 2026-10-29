package detentionnoticeservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// noticePDFReferenceType names the business object a generated notice PDF is
// about, for the generated-documents audit row.
const noticePDFReferenceType = "detention_occurrence"

// noticePDFTitle names the document in a reader's PDF viewer and in the file they
// save. A customer filing a dispute should be able to tell two notices apart
// without opening them.
func noticePDFTitle(occurrence *detention.DetentionOccurrence) string {
	ref := occurrence.ShipmentProNumber
	if ref == "" {
		ref = occurrence.ShipmentID.String()
	}

	return "Detention Notice " + ref
}

func noticePDFFileName(occurrence *detention.DetentionOccurrence) string {
	ref := occurrence.ShipmentProNumber
	if ref == "" {
		ref = occurrence.ShipmentID.String()
	}

	return fmt.Sprintf("detention-notice-%s-%s.pdf", ref, occurrence.ID.String())
}

// noticePDFFiling is one notice PDF on its way into document storage.
type noticePDFFiling struct {
	TenantInfo pagination.TenantInfo
	Notice     *detentionnotice.DetentionNotice
	Occurrence *detention.DetentionOccurrence
	PDF        *detentionnoticerender.NoticePDF
}

// actor attributes the upload. An automatic notice has no sender, so the system
// owns it either way.
func (f *noticePDFFiling) actor() permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		UserID:         f.TenantInfo.UserID,
		OrganizationID: f.Notice.OrganizationID,
		BusinessUnitID: f.Notice.BusinessUnitID,
	}
}

func (f *noticePDFFiling) size() int64 {
	return int64(len(f.PDF.Content))
}

// noticeActorUserID names whoever the filing is attributed to. An automatic
// notice has no sender, and the upload session is happy with a nil user — the
// system is the actor either way.
func noticeActorUserID(notice *detentionnotice.DetentionNotice) pulid.ID {
	if notice.SentByID != nil && !notice.SentByID.IsNil() {
		return *notice.SentByID
	}

	return pulid.Nil
}
