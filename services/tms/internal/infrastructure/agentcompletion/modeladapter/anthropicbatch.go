package modeladapter

import (
	"context"
	"errors"
	"slices"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	anthropicBatchSucceeded = "succeeded"
	anthropicBatchErrored   = "errored"
)

var errEmptyBatch = errors.New("a batch needs at least one request")

var _ BatchRunner = anthropicAdapter{}

func (anthropicAdapter) Batches(provider *aiprovider.Provider) bool {
	return provider != nil && provider.OnClaudeAPI()
}

func (a anthropicAdapter) SubmitBatch(
	ctx context.Context,
	call *Call,
	requests []BatchRequest,
) (*BatchSubmission, error) {
	if len(requests) == 0 {
		return nil, errEmptyBatch
	}

	params := anthropic.BetaMessageBatchNewParams{
		Requests: make([]anthropic.BetaMessageBatchNewParamsRequest, 0, len(requests)),
	}
	for _, request := range requests {
		message := anthropicBatchMessage(ctx, call, request.Request)
		for _, beta := range message.Betas {
			if !slices.Contains(params.Betas, beta) {
				params.Betas = append(params.Betas, beta)
			}
		}
		params.Requests = append(params.Requests, anthropic.BetaMessageBatchNewParamsRequest{
			CustomID: request.CustomID,
			Params:   anthropicBatchParams(&message),
		})
	}

	service, err := anthropicMessages(ctx, call, false, a.credentials)
	if err != nil {
		return nil, err
	}
	batch, err := service.Batches.New(ctx, params, anthropicBlockingOptions(call)...)
	if err != nil {
		return nil, err
	}

	return &BatchSubmission{
		ID:        batch.ID,
		Status:    string(batch.ProcessingStatus),
		ExpiresAt: batch.ExpiresAt.Unix(),
	}, nil
}

func anthropicBatchMessage(
	ctx context.Context,
	call *Call,
	request *Request,
) anthropic.BetaMessageNewParams {
	item := *call
	item.Request = request
	item.CacheDiagnostics = false
	attached := attachFiles(ctx, &item)
	params := anthropicParams(&item, &attached)
	applyOutputSchema(&params, &item)

	return params
}

func anthropicBatchParams(
	p *anthropic.BetaMessageNewParams,
) anthropic.BetaMessageBatchNewParamsRequestParams {
	return anthropic.BetaMessageBatchNewParamsRequestParams{
		MaxTokens:    p.MaxTokens,
		Messages:     p.Messages,
		Model:        p.Model,
		InferenceGeo: p.InferenceGeo,
		Temperature:  p.Temperature,
		TopK:         p.TopK,
		TopP:         p.TopP,
		Container: anthropic.BetaMessageBatchNewParamsRequestParamsContainerUnion{
			OfContainers: p.Container.OfContainers,
			OfString:     p.Container.OfString,
		},
		FallbackCreditToken: anthropic.BetaMessageBatchNewParamsRequestParamsFallbackCreditTokenUnion{
			OfString:              p.FallbackCreditToken.OfString,
			OfFallbackCreditToken: p.FallbackCreditToken.OfFallbackCreditToken,
		},
		Fallbacks:         p.Fallbacks,
		Speed:             string(p.Speed),
		CacheControl:      p.CacheControl,
		Compaction:        p.Compaction,
		ContextManagement: p.ContextManagement,
		Diagnostics:       p.Diagnostics,
		MCPServers:        p.MCPServers,
		Metadata:          p.Metadata,
		OutputConfig:      p.OutputConfig,
		OutputFormat:      p.OutputFormat,
		ServiceTier:       string(p.ServiceTier),
		StopSequences:     p.StopSequences,
		System:            p.System,
		Thinking:          p.Thinking,
		ToolChoice:        p.ToolChoice,
		Tools:             p.Tools,
	}
}

func (a anthropicAdapter) PollBatch(
	ctx context.Context,
	call *Call,
	id string,
) (*BatchProgress, error) {
	service, err := anthropicMessages(ctx, call, false, a.credentials)
	if err != nil {
		return nil, err
	}
	batch, err := service.Batches.Get(
		ctx,
		id,
		anthropic.BetaMessageBatchGetParams{},
		anthropicBlockingOptions(call)...,
	)
	if err != nil {
		return nil, err
	}

	counts := batch.RequestCounts

	return &BatchProgress{
		Ended:      batch.ProcessingStatus == anthropic.BetaMessageBatchProcessingStatusEnded,
		RawStatus:  string(batch.ProcessingStatus),
		Processing: int(counts.Processing),
		Succeeded:  int(counts.Succeeded),
		Errored:    int(counts.Errored),
		Canceled:   int(counts.Canceled),
		Expired:    int(counts.Expired),
	}, nil
}

func (a anthropicAdapter) ReadBatch(
	ctx context.Context,
	call *Call,
	id string,
	each func(*BatchItem) error,
) error {
	service, err := anthropicMessages(ctx, call, false, a.credentials)
	if err != nil {
		return err
	}
	stream := service.Batches.ResultsStreaming(
		ctx,
		id,
		anthropic.BetaMessageBatchResultsParams{},
		anthropicBlockingOptions(call)...,
	)
	defer func() { _ = stream.Close() }()

	for stream.Next() {
		entry := stream.Current()
		if err = each(anthropicBatchItem(&entry)); err != nil {
			return err
		}
	}

	return stream.Err()
}

func anthropicBatchItem(entry *anthropic.BetaMessageBatchIndividualResponse) *BatchItem {
	item := &BatchItem{CustomID: entry.CustomID}
	switch entry.Result.Type {
	case anthropicBatchSucceeded:
		item.Response = anthropicMessageResponse(&entry.Result.Message, nil)
	case anthropicBatchErrored:
		item.FailureCode = stringutils.FirstNonEmpty(
			entry.Result.Error.Error.Type,
			anthropicBatchErrored,
		)
		item.FailureMessage = entry.Result.Error.Error.Message
	default:
		item.FailureCode = entry.Result.Type
		item.NotRun = true
	}

	return item
}
