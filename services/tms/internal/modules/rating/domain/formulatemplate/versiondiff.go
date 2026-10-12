package formulatemplate

import (
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type VersionDiff struct {
	FromVersion int64                            `json:"fromVersion"`
	ToVersion   int64                            `json:"toVersion"`
	Changes     map[string]jsonutils.FieldChange `json:"changes"`
	ChangeCount int                              `json:"changeCount"`
}

type ForkLineage struct {
	TemplateID       pulid.ID      `json:"templateId"`
	TemplateName     string        `json:"templateName"`
	SourceTemplateID *pulid.ID     `json:"sourceTemplateId,omitempty"`
	SourceVersion    *int64        `json:"sourceVersion,omitempty"`
	ForkedTemplates  []ForkLineage `json:"forkedTemplates,omitempty"`
}
