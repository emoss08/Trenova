package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

type emailProfileSelector interface {
	SelectProfileOptions(
		ctx context.Context,
		req *emailports.EmailProfileSelectOptionsRequest,
	) (*pagination.ListResult[*email.Profile], error)
}

type emailProfileRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SenderName  string `json:"senderName,omitempty"`
	SenderEmail string `json:"senderEmail"`
	ReplyTo     string `json:"replyTo,omitempty"`
}

type listEmailProfilesTool struct {
	profiles emailProfileSelector
}

func newListEmailProfilesTool(profiles emailProfileSelector) agentports.AgentQueryTool {
	return &listEmailProfilesTool{profiles: profiles}
}

func (t *listEmailProfilesTool) Name() string { return "inbox_list_email_profiles" }

func (t *listEmailProfilesTool) Description() string {
	return "The sending profiles email can go out from: each has a name, a sender " +
		"address and a reply-to. customer_send_email and shipment_request_documents take one " +
		"as profileId. Pick the profile whose name or sender fits the message, such " +
		"as operations for a status update and billing for a document request; with " +
		"one profile there is nothing to choose."
}

func (t *listEmailProfilesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against the profile name and sender.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listEmailProfilesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceEmailProfile,
	})
}

func (t *listEmailProfilesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query := optionalString(params.Params, "query")
	criteria := filtercatalog.NewCriteria("email profiles").At(clockFor(params))
	criteria.Text(query)

	result, err := t.profiles.SelectProfileOptions(
		ctx,
		&emailports.EmailProfileSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantOf(params),
				Pagination: pagination.Info{Limit: maxListLimit},
				Query:      query,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	rows := make([]emailProfileRow, 0, len(result.Items))
	for _, profile := range result.Items {
		if profile == nil || profile.Status != email.ProfileStatusActive {
			continue
		}
		rows = append(rows, emailProfileRow{
			ID:          profile.ID.String(),
			Name:        profile.Name,
			Description: profile.Description,
			SenderName:  profile.SenderName,
			SenderEmail: profile.SenderEmail,
			ReplyTo:     profile.ReplyToEmail,
		})
	}

	return searchResult(criteria, rows, len(rows)), nil
}
