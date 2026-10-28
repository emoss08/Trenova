package customerupdateservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	SourceAgentEmail   = shipment.CommentToolEmailCustomer
	SourceDelayNotice  = shipment.CommentToolDelayNotice
	RecentWindowSecs   = int64(3600)
	recentPageSize     = 25
	metadataSource     = shipment.CommentMetadataTool
	metadataRecipients = "recipients"
	metadataSubject    = "subject"
)

var ErrAlreadyTold = errors.New(
	"this customer was already emailed about this shipment within the hour",
)

func AlreadyTold(
	ctx context.Context,
	comments shipmentcommentports.ShipmentCommentService,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
	now int64,
) (bool, error) {
	if comments == nil {
		return false, nil
	}

	page, err := comments.ListByShipmentID(ctx, &shipmentcommentports.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: recentPageSize},
		},
		Cursor:     pagination.CursorInfo{Limit: recentPageSize},
		ShipmentID: shipmentID,
		Filters: shipmentcommentports.ShipmentCommentListFilters{
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

	return slices.Contains(shipment.CustomerEmailTools(), source)
}

func Recipients(
	ctx context.Context,
	customers customerports.CustomerRepository,
	sp *shipment.Shipment,
	tenant pagination.TenantInfo,
) (recipients []string, customerName string, err error) {
	if sp.CustomerID.IsNil() {
		return nil, "", errors.New("the shipment has no customer to write to")
	}

	entity, err := customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:                    sp.CustomerID,
		TenantInfo:            tenant,
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeEmailProfile: true},
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
	orgRepo tenantports.OrganizationRepository,
	inliner documenttemplateports.AssetInliner,
	tenantInfo pagination.TenantInfo,
	out *documenttemplate.AgentEmailContext,
) {
	if orgRepo == nil {
		return
	}

	org, err := orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return
	}

	out.CompanyName = org.Name

	if dataURI, logoErr := documenttemplateports.ResolveLogoDataURI(
		ctx,
		inliner,
		org.LogoURL,
	); logoErr == nil {
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

func Comment(p *CommentParams) *shipmentcommentports.CreateSystemShipmentCommentRequest {
	metadata := map[string]any{
		shipment.CommentMetadataOrigin: p.Origin,
		metadataSource:                 p.Source,
		metadataRecipients:             p.Recipients,
		metadataSubject:                p.Subject,
	}
	for key, value := range p.Extra {
		metadata[key] = value
	}

	return &shipmentcommentports.CreateSystemShipmentCommentRequest{
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
