package aitraining

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrExportInactive = errors.New("the training export is no longer active")

const (
	AnonymizationMethod = "trenova.deterministic-pseudonymization/v1"
	MaxExamplePages     = 100
	MaxExamplePageRunes = 20000
	splitBuckets        = 100
)

type ManifestPart struct {
	Key      string `json:"key"`
	Split    Split  `json:"split"`
	Examples int    `json:"examples"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
}

type ManifestCounts struct {
	Total      int `json:"total"`
	Train      int `json:"train"`
	Validation int `json:"validation"`
}

type Manifest struct {
	Format                  string             `json:"format"`
	ExampleFormat           string             `json:"exampleFormat"`
	AnonymizationMethod     string             `json:"anonymizationMethod"`
	ExportID                pulid.ID           `json:"exportId"`
	Task                    aicorrection.Task  `json:"task"`
	Status                  ExportStatus       `json:"status"`
	CapturedFrom            int64              `json:"capturedFrom"`
	CapturedTo              int64              `json:"capturedTo"`
	MaxPerOrganization      int                `json:"maxPerOrganization"`
	ValidationPercent       int                `json:"validationPercent"`
	OrganizationsConsidered int                `json:"organizationsConsidered"`
	OrganizationsIncluded   int                `json:"organizationsIncluded"`
	Examples                ManifestCounts     `json:"examples"`
	Dropped                 map[DropReason]int `json:"dropped"`
	Parts                   []ManifestPart     `json:"parts"`
	CreatedAt               int64              `json:"createdAt"`
	FinishedAt              int64              `json:"finishedAt"`
}

func NewManifest(e *TrainingExport, finishedAt int64) *Manifest {
	parts := make([]ManifestPart, 0, len(e.Parts))
	for i := range e.Parts {
		parts = append(parts, ManifestPart{
			Key:      e.Parts[i].Key,
			Split:    e.Parts[i].Split,
			Examples: e.Parts[i].Examples,
			Bytes:    e.Parts[i].Bytes,
			SHA256:   e.Parts[i].SHA256,
		})
	}
	dropped := make(map[DropReason]int, len(e.Dropped))
	for reason, count := range e.Dropped {
		dropped[reason] = count
	}

	return &Manifest{
		Format:                  ManifestFormat,
		ExampleFormat:           e.Format,
		AnonymizationMethod:     AnonymizationMethod,
		ExportID:                e.ID,
		Task:                    e.Task,
		Status:                  e.Status,
		CapturedFrom:            e.CapturedFrom,
		CapturedTo:              e.CapturedTo,
		MaxPerOrganization:      e.MaxPerOrganization,
		ValidationPercent:       e.ValidationPercent,
		OrganizationsConsidered: e.OrganizationsConsidered,
		OrganizationsIncluded:   e.OrganizationsIncluded,
		Examples: ManifestCounts{
			Total:      e.ExamplesTotal,
			Train:      e.TrainExamples,
			Validation: e.ValidationExamples,
		},
		Dropped:    dropped,
		Parts:      parts,
		CreatedAt:  e.CreatedAt,
		FinishedAt: finishedAt,
	}
}

func SplitFor(exportID, correctionID pulid.ID, validationPercent int) Split {
	if validationPercent <= 0 {
		return SplitTrain
	}
	sum := sha256.Sum256([]byte(exportID.String() + ":" + correctionID.String()))
	if int(binary.BigEndian.Uint16(sum[:2])%splitBuckets) < validationPercent {
		return SplitValidation
	}

	return SplitTrain
}
