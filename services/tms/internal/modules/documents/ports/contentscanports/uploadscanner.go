package contentscanports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type uploadScanner struct {
	gate Gate
}

func NewUploadScanner(gate Gate) kernelports.UploadScanner {
	return uploadScanner{gate: gate}
}

func (s uploadScanner) ScanUpload(ctx context.Context, req *kernelports.UploadScanRequest) error {
	if _, err := Check(ctx, s.gate, &ScanRequest{
		TenantInfo: req.TenantInfo,
		Source:     SourceUpload,
		FileName:   req.FileName,
		Content:    req.Content,
	}); err != nil {
		return RefusalError(req.Field, err)
	}
	return nil
}
