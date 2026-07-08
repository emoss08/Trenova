package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type EDIPartnerReadinessItem struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Complete bool   `json:"complete"`
}

type EDIPartnerReadiness struct {
	PartnerID      pulid.ID                  `json:"partnerId"`
	Ready          bool                      `json:"ready"`
	CompletedCount int                       `json:"completedCount"`
	TotalCount     int                       `json:"totalCount"`
	Items          []EDIPartnerReadinessItem `json:"items"`
}

type GetEDIPartnerReadinessRequest struct {
	TenantInfo pagination.TenantInfo
	PartnerIDs []pulid.ID
}

func (s *Service) GetPartnerReadiness(
	ctx context.Context,
	req *GetEDIPartnerReadinessRequest,
) ([]*EDIPartnerReadiness, error) {
	rows, err := s.partnerRepo.GetReadiness(ctx, &repositories.GetEDIPartnerReadinessRequest{
		TenantInfo: req.TenantInfo,
		PartnerIDs: req.PartnerIDs,
	})
	if err != nil {
		return nil, err
	}
	states := make([]*EDIPartnerReadiness, 0, len(rows))
	for _, row := range rows {
		states = append(states, buildPartnerReadiness(row))
	}
	return states, nil
}

func buildPartnerReadiness(row *repositories.EDIPartnerReadinessRow) *EDIPartnerReadiness {
	items := []EDIPartnerReadinessItem{
		{
			Key:      "details",
			Label:    "Partner details (contact email and timezone)",
			Complete: row.ContactEmail != "" && row.Timezone != "",
		},
		{
			Key:      "communication-profile",
			Label:    "Active communication profile",
			Complete: row.HasActiveProfile,
		},
		{
			Key:      "mappings",
			Label:    "Entity mappings defined",
			Complete: row.HasMappingProfile,
		},
	}
	if row.EnabledForInbound {
		items = append(items, EDIPartnerReadinessItem{
			Key:      "inbound-document-profile",
			Label:    "Active inbound document profile",
			Complete: row.HasInboundDocProfile,
		})
	}
	if row.EnabledForOutbound {
		items = append(items, EDIPartnerReadinessItem{
			Key:      "outbound-document-profile",
			Label:    "Active outbound document profile",
			Complete: row.HasOutboundDocProfile,
		})
	}
	if row.Kind == string(edi.PartnerKindExternal) {
		items = append(items, EDIPartnerReadinessItem{
			Key:      "test-case",
			Label:    "At least one passing test case",
			Complete: row.HasPassingTestCase,
		})
	}

	state := &EDIPartnerReadiness{
		PartnerID:  row.PartnerID,
		TotalCount: len(items),
		Items:      items,
	}
	for _, item := range items {
		if item.Complete {
			state.CompletedCount++
		}
	}
	state.Ready = state.CompletedCount == state.TotalCount
	return state
}
