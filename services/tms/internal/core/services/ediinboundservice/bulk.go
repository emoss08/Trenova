package ediinboundservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BulkReprocessInboundFilesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	FileIDs    []pulid.ID            `json:"fileIds"`
}

func (s *Service) BulkReprocessInboundFiles(
	ctx context.Context,
	req *BulkReprocessInboundFilesRequest,
) (*ediservice.BulkEDIActionResult, error) {
	if err := ediservice.ValidateBulkEDIActionIDs("fileIds", req.FileIDs); err != nil {
		return nil, err
	}
	result := &ediservice.BulkEDIActionResult{
		Succeeded: make([]pulid.ID, 0, len(req.FileIDs)),
		Failed:    make([]ediservice.BulkEDIActionFailure, 0),
	}
	for _, fileID := range req.FileIDs {
		if _, err := s.ProcessInboundFile(ctx, &ProcessInboundFileRequest{
			FileID:     fileID,
			TenantInfo: req.TenantInfo,
			Reprocess:  true,
		}); err != nil {
			result.Failed = append(result.Failed, ediservice.BulkEDIActionFailure{
				ID:    fileID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, fileID)
	}
	return result, nil
}
