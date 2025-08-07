/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GeneratePreviewRequest struct {
	File         []byte              `json:"file"`
	FileName     string              `json:"fileName"`
	OrgID        pulid.ID            `json:"orgId"`
	UserID       string              `json:"userId"`
	ResourceID   pulid.ID            `json:"resourceId"`
	ResourceType permission.Resource `json:"resourceType"`
	BucketName   string              `json:"bucketName"`
}

type GeneratePreviewResponse struct {
	PreviewPath string `json:"previewPath"`
	PreviewURL  string `json:"previewUrl"`

	// * Async Job fields
	JobID   string `json:"jobId,omitempty"`
	IsAsync bool   `json:"isAsync,omitempty"`
	Status  string `json:"status,omitempty"`
	Error   string `json:"error,omitempty"`
}

type GetPreviewURLRequest struct {
	PreviewPath string        `json:"previewPath"`
	BucketName  string        `json:"bucketName"`
	OrgID       pulid.ID      `json:"orgId"`
	ExpiryTime  time.Duration `json:"expiryTime"`
}

type DeletePreviewRequest struct {
	PreviewPath string   `json:"previewPath"`
	BucketName  string   `json:"bucketName"`
	OrgID       pulid.ID `json:"orgId"`
}

type PreviewService interface {
	GeneratePreview(
		ctx context.Context,
		req *GeneratePreviewRequest,
	) (*GeneratePreviewResponse, error)
	GetPreviewURL(ctx context.Context, req *GetPreviewURLRequest) (string, error)
	DeletePreview(ctx context.Context, req *DeletePreviewRequest) error
}
