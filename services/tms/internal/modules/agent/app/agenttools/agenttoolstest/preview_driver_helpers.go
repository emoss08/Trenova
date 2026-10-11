package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
)

func (s *StubCredentialActor) PreviewRenewal(
	_ context.Context,
	req *workercredentialservice.RenewalRequest,
) (*workercredentialservice.RenewalPreview, error) {
	expires := int64(1_790_000_000)

	return &workercredentialservice.RenewalPreview{
		Credentials: []*worker.WorkerCredential{{
			ID:             req.CredentialIDs[0],
			ExpiresAt:      &expires,
			CredentialType: &worker.WorkerCredentialType{Name: "CDL"},
		}},
		Notification: &notificationports.DriverNotificationPreview{
			WorkerID:   req.WorkerID,
			WorkerName: "Marcus Dell",
			Reachable:  false,
			Title:      "Renew your CDL",
			Message:    req.Note,
		},
	}, nil
}
