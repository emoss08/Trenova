package customerupdateservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	SourceAgentEmail   = "email_customer"
	SourceDelayNotice  = "notify_shipment_delay"
	RecentWindowSecs   = int64(3600)
	recentPageSize     = 25
	metadataSource     = "tool"
	metadataRecipients = "recipients"
	metadataSubject    = "subject"
)

var ErrAlreadyTold = errors.New(
	"this customer was already emailed about this shipment within the hour",
)

var emailSources = []string{SourceAgentEmail, SourceDelayNotice}

func AlreadyTold(
	ctx context.Context,
	comments services.ShipmentCommentService,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
	now int64,
) (bool, error) {
	if comments == nil {
		return false, nil
	}

	page, err := comments.ListByShipmentID(ctx, &repositories.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: recentPageSize},
		},
		Cursor:     pagination.CursorInfo{Limit: recentPageSize},
		ShipmentID: shipmentID,
		Filters: repositories.ShipmentCommentListFilters{
			Types: []shipment.CommentType{shipment.CommentTypeCustomerUpdate},
		},
	})
	if err != nil {
		return false, err
	}

	for _, comment := range page.Items {
		if comment == nil || comment.CreatedAt < now-RecentWindowSecs {
			continue
		}
		if WasEmailed(comment) {
			return true, nil
		}
	}

	return false, nil
}

func WasEmailed(comment *shipment.ShipmentComment) bool {
	if comment == nil || comment.Metadata == nil {
		return false
	}
	source, _ := comment.Metadata[metadataSource].(string)

	return slices.Contains(emailSources, source)
}

func Recipients(
	ctx context.Context,
	customers repositories.CustomerRepository,
	sp *shipment.Shipment,
	tenant pagination.TenantInfo,
) (recipients []string, customerName string, err error) {
	if sp.CustomerID.IsNil() {
		return nil, "", errors.New("the shipment has no customer to write to")
	}

	entity, err := customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:                    sp.CustomerID,
		TenantInfo:            tenant,
		CustomerFilterOptions: repositories.CustomerFilterOptions{IncludeEmailProfile: true},
	})
	if err != nil {
		return nil, "", err
	}

	if entity.EmailProfile != nil {
		recipients = stringutils.SplitEmailList(entity.EmailProfile.ToRecipients)
	}
	if len(recipients) == 0 {
		return nil, "", fmt.Errorf(
			"%s has no notice recipients on file; a person must add an email profile "+
				"to the customer before anything can be sent",
			entity.Name,
		)
	}

	return recipients, entity.Name, nil
}

func Brand(
	ctx context.Context,
	orgRepo repositories.OrganizationRepository,
	inliner services.AssetInliner,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentEmailContext,
) {
	if orgRepo == nil {
		return
	}

	org, err := orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	out.CompanyName = org.Name

	if dataURI, logoErr := services.ResolveLogoDataURI(ctx, inliner, org.LogoURL); logoErr == nil {
		out.LogoDataURI = dataURI
	}
}

type CommentParams struct {
	Tenant     pagination.TenantInfo
	ShipmentID pulid.ID
	Recipients []string
	Subject    string
	Body       string
	Origin     string
	Source     string
	Extra      map[string]any
}

func Comment(p *CommentParams) *services.CreateSystemShipmentCommentRequest {
	metadata := map[string]any{
		shipment.CommentMetadataOrigin: p.Origin,
		metadataSource:                 p.Source,
		metadataRecipients:             p.Recipients,
		metadataSubject:                p.Subject,
	}
	for key, value := range p.Extra {
		metadata[key] = value
	}

	return &services.CreateSystemShipmentCommentRequest{
		TenantInfo: p.Tenant,
		ShipmentID: p.ShipmentID,
		Comment: fmt.Sprintf(
			"Emailed %s: %s\n\n%s",
			strings.Join(p.Recipients, ", "),
			p.Subject,
			p.Body,
		),
		Type:       shipment.CommentTypeCustomerUpdate,
		Visibility: shipment.CommentVisibilityOperations,
		Priority:   shipment.CommentPriorityNormal,
		Metadata:   metadata,
	}
}
