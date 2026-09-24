package agent

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type Fingerprint struct {
	DefinitionVersion int64    `json:"definitionVersion"`
	PromptHash        string   `json:"promptHash,omitempty"`
	ToolSpecHash      string   `json:"toolSpecHash,omitempty"`
	Model             string   `json:"model,omitempty"`
	ProviderID        pulid.ID `json:"providerId,omitempty"`
	PromptVersion     string   `json:"promptVersion,omitempty"`
	InstructionsHash  string   `json:"instructionsHash,omitempty"`
	Tools             []string `json:"tools,omitempty"`
}

type FingerprintField string

const (
	FingerprintFieldDefinition = FingerprintField("definitionVersion")
	FingerprintFieldPrompt     = FingerprintField("prompt")
	FingerprintFieldTools      = FingerprintField("tools")
	FingerprintFieldModel      = FingerprintField("model")
	FingerprintFieldProvider   = FingerprintField("provider")
)

type FingerprintChange struct {
	Field FingerprintField `json:"field"`
	From  string           `json:"from,omitempty"`
	To    string           `json:"to,omitempty"`
}

func (f *Fingerprint) Hash() string {
	if f == nil {
		return ""
	}

	var builder strings.Builder
	builder.Grow(256)
	builder.WriteString("v1|")
	builder.WriteString(strconv.FormatInt(f.DefinitionVersion, 10))
	builder.WriteByte('|')
	builder.WriteString(f.PromptHash)
	builder.WriteByte('|')
	builder.WriteString(f.ToolSpecHash)
	builder.WriteByte('|')
	builder.WriteString(f.Model)
	builder.WriteByte('|')
	builder.WriteString(f.ProviderID.String())

	return hashutils.SHA256Hex(builder.String())
}

func (f *Fingerprint) Served(model string, providerID pulid.ID) *Fingerprint {
	if f == nil {
		return nil
	}

	served := *f
	if model != "" {
		served.Model = model
	}
	if providerID.IsNotNil() {
		served.ProviderID = providerID
	}

	return &served
}

func (f *Fingerprint) Changes(previous *Fingerprint) []FingerprintChange {
	if f == nil || previous == nil {
		return nil
	}

	changes := make([]FingerprintChange, 0, 5)
	if f.DefinitionVersion != previous.DefinitionVersion {
		changes = append(changes, FingerprintChange{
			Field: FingerprintFieldDefinition,
			From:  strconv.FormatInt(previous.DefinitionVersion, 10),
			To:    strconv.FormatInt(f.DefinitionVersion, 10),
		})
	}
	if f.PromptHash != previous.PromptHash {
		changes = append(changes, FingerprintChange{Field: FingerprintFieldPrompt})
	}
	if f.ToolSpecHash != previous.ToolSpecHash {
		changes = append(changes, FingerprintChange{Field: FingerprintFieldTools})
	}
	if f.Model != previous.Model {
		changes = append(changes, FingerprintChange{
			Field: FingerprintFieldModel,
			From:  previous.Model,
			To:    f.Model,
		})
	}
	if f.ProviderID != previous.ProviderID {
		changes = append(changes, FingerprintChange{
			Field: FingerprintFieldProvider,
			From:  previous.ProviderID.String(),
			To:    f.ProviderID.String(),
		})
	}

	return changes
}
