package completionengine

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

func StructuredRun(req *llmports.StructuredCompletionRequest) *RunRequest {
	task := req.Task
	if task == "" {
		task = aitask.TaskGeneral
	}

	return &RunRequest{
		TenantInfo:          req.TenantInfo,
		Task:                task,
		System:              req.System,
		UserContent:         req.Context.Text(),
		Files:               req.Files,
		Schema:              req.OutputSchema,
		SchemaName:          req.SchemaName,
		MaxTokens:           req.MaxTokens,
		PreferredProviderID: req.PreferredProviderID,
		RequireProvider:     req.RequireProvider,
		HedgeAfter:          req.HedgeAfter,
		Reasoning:           req.Reasoning,
		Attribution:         req.Attribution,
	}
}

func StructuredSystem(run *RunRequest, mode aiprovider.StructuredOutputMode) string {
	return modeladapter.WithSchemaInstruction(run.System, run.Schema, mode)
}

// structuredMessages is the one user message a structured call sends: the
// context, then an entry for each file. A model that reads files is sent each
// file in place of its entry; any other reads the entry, which names it.
func structuredMessages(req *RunRequest) []modeladapter.Message {
	if len(req.Files) == 0 {
		return modeladapter.UserMessage(req.UserContent)
	}

	var content strings.Builder
	content.WriteString(req.UserContent)
	files := make([]llmports.MessageFile, 0, len(req.Files))
	for _, file := range req.Files {
		content.WriteString("\n\n")
		entry := fmt.Sprintf("[Attached file: %s]", file.FileName)
		file.EntryStart = content.Len()
		content.WriteString(entry)
		file.EntryEnd = content.Len()
		if file.NativeEntry == "" {
			file.NativeEntry = fmt.Sprintf("[The file %s is attached]", file.FileName)
		}
		files = append(files, file)
	}

	return []modeladapter.Message{{
		Role:    llmports.RoleUser,
		Content: content.String(),
		Files:   files,
	}}
}
