package runtimetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/productguideports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
)

type NoExtensions struct{}

func (NoExtensions) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	return agentextension.Active{}, nil
}

type NoVectors struct{}

func (NoVectors) Vectorize(
	context.Context,
	*retrievalports.QueryVectorRequest,
) (retrievalports.QueryVector, error) {
	return retrievalports.QueryVector{}, nil
}

func (NoVectors) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{}, nil
}

func (NoVectors) Similarities(
	context.Context,
	*retrievalports.CatalogSimilarityRequest,
) (retrievalports.CatalogSimilarities, error) {
	return retrievalports.CatalogSimilarities{}, nil
}

type NoScripts struct{}

func (NoScripts) ScriptsOffered(context.Context, pagination.TenantInfo) (bool, error) {
	return false, nil
}

func (NoScripts) ScriptSettings(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (aiprovider.ScriptSettings, error) {
	return aiprovider.ScriptSettings{}, nil
}

type NoChanges struct{}

func (NoChanges) Head(context.Context, pulid.ID, pulid.ID) (string, error) {
	return "", nil
}

func (NoChanges) Since(
	context.Context,
	*realtimeports.RecordChangesRequest,
) (*realtimeports.RecordChanges, error) {
	return &realtimeports.RecordChanges{}, nil
}

type UnusedMemories struct {
	agentmemoryports.AgentMemoryService
}

func (UnusedMemories) ForContext(
	context.Context,
	*agentmemoryports.MemoryContextRequest,
) (*agentmemoryports.MemoryContext, error) {
	return &agentmemoryports.MemoryContext{}, nil
}

func (UnusedMemories) RecordUse(context.Context, agentmemoryports.RecordMemoryUseRequest) error {
	return nil
}

type NoProfiles struct{}

func (NoProfiles) ForPrompt(
	context.Context,
	agentprofileports.PromptProfilesRequest,
) (agentprofileports.PromptProfiles, error) {
	return agentprofileports.PromptProfiles{}, nil
}

type NoGuide struct {
	productguideports.ProductGuide
}

func (NoGuide) PageForPath(string) (*productguide.Page, bool) {
	return nil, false
}
