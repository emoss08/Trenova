package kernelports

import "context"

type HeavyJob string

const (
	HeavyJobPDF        = HeavyJob("pdf")
	HeavyJobExtraction = HeavyJob("extraction")
	HeavyJobExport     = HeavyJob("export")
	HeavyJobImport     = HeavyJob("import")
	HeavyJobPreview    = HeavyJob("preview")
	HeavyJobRemoval    = HeavyJob("removal")
)

type HeavyJobSlots interface {
	Hold(ctx context.Context, job HeavyJob) (func(), error)
	TryHold(ctx context.Context, job HeavyJob) (func(), bool)
}
