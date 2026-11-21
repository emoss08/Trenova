package trenova

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
)

type Partner struct {
	ID                 string         `json:"id,omitempty"`
	Kind               string         `json:"kind"`
	Status             string         `json:"status"`
	Code               string         `json:"code"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	DefaultTransportID string         `json:"defaultTransportId,omitempty"`
	CustomerID         string         `json:"customerId,omitempty"`
	Timezone           string         `json:"timezone"`
	Country            string         `json:"country"`
	ContactName        string         `json:"contactName"`
	ContactEmail       string         `json:"contactEmail"`
	EnabledForInbound  bool           `json:"enabledForInbound"`
	EnabledForOutbound bool           `json:"enabledForOutbound"`
	Settings           map[string]any `json:"settings"`
	Version            int64          `json:"version"`
	DefaultTransport   *Profile       `json:"defaultTransport,omitempty"`
}

type Profile struct {
	ID             string            `json:"id,omitempty"`
	EDIPartnerID   string            `json:"ediPartnerId"`
	Method         string            `json:"method"`
	Status         string            `json:"status"`
	Name           string            `json:"name"`
	Config         map[string]any    `json:"config"`
	Secrets        map[string]string `json:"secrets,omitempty"`
	AS2InboundPath string            `json:"as2InboundPath,omitempty"`
	Version        int64             `json:"version"`
}

func (c *Client) Partners(ctx context.Context, search string) ([]Partner, error) {
	var found page[Partner]
	query := url.Values{"query": {search}, "limit": {"100"}}
	if err := c.get(ctx, listPath("/edi/partners/", query), &found); err != nil {
		return nil, err
	}
	return found.Results, nil
}

func (c *Client) FindPartner(ctx context.Context, code string) (*Partner, error) {
	found, err := c.Partners(ctx, code)
	if err != nil {
		return nil, err
	}
	for index := range found {
		if strings.EqualFold(found[index].Code, code) {
			return c.Partner(ctx, found[index].ID)
		}
	}
	return nil, nil
}

func (c *Client) Partner(ctx context.Context, id string) (*Partner, error) {
	partner := new(Partner)
	err := c.get(ctx, "/edi/partners/"+url.PathEscape(id)+"/", partner)
	return partner, err
}

func (c *Client) CreatePartner(ctx context.Context, partner *Partner) (*Partner, error) {
	created := new(Partner)
	err := c.post(ctx, "/edi/partners/", partner, created)
	return created, err
}

func (c *Client) UpdatePartner(ctx context.Context, partner *Partner) (*Partner, error) {
	updated := new(Partner)
	body := *partner
	body.DefaultTransport = nil
	err := c.put(ctx, "/edi/partners/"+url.PathEscape(partner.ID)+"/", &body, updated)
	return updated, err
}

func (c *Client) CreateProfile(ctx context.Context, profile *Profile) (*Profile, error) {
	created := new(Profile)
	err := c.post(ctx, "/edi/communication-profiles/", profile, created)
	return created, err
}

func (c *Client) UpdateProfile(ctx context.Context, profile *Profile) (*Profile, error) {
	updated := new(Profile)
	err := c.put(ctx, "/edi/communication-profiles/"+url.PathEscape(profile.ID)+"/",
		profile, updated)
	return updated, err
}

type ConnectionTest struct {
	Success bool `json:"success"`
	Checks  []struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"checks"`
}

func (c *Client) TestProfile(ctx context.Context, id string) (*ConnectionTest, error) {
	result := new(ConnectionTest)
	err := c.post(ctx, "/edi/communication-profiles/"+url.PathEscape(id)+"/test-connection/",
		nil, result)
	return result, err
}

func (c *Client) PollProfile(ctx context.Context, id string) error {
	return c.post(ctx, "/edi/communication-profiles/"+url.PathEscape(id)+"/poll/", nil, nil)
}

type Project struct {
	ID              string   `json:"id"`
	EDIPartnerID    string   `json:"ediPartnerId"`
	ActiveVersionID string   `json:"activeVersionId"`
	ActiveVersion   *Version `json:"activeVersion,omitempty"`
	Version         int64    `json:"version"`
}

type Readiness struct {
	Errors               int  `json:"errors"`
	Failing              int  `json:"failing"`
	Tests                int  `json:"tests"`
	UncoveredFlowClauses int  `json:"uncoveredFlowClauses"`
	Ready                bool `json:"ready"`
}

type SampleRef struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type Version struct {
	ID           string        `json:"id"`
	ProjectID    string        `json:"projectId"`
	Number       int64         `json:"number"`
	Status       string        `json:"status"`
	Files        []lading.File `json:"files"`
	Samples      []SampleRef   `json:"samples"`
	Manifest     lading.Check  `json:"manifest"`
	Readiness    *Readiness    `json:"readiness"`
	ErrorCount   int           `json:"errorCount"`
	WarningCount int           `json:"warningCount"`
	Version      int64         `json:"version"`
	UpdatedAt    int64         `json:"updatedAt"`
}

type VersionDetail struct {
	Version *Version        `json:"version"`
	Samples []lading.Sample `json:"samples"`
}

type CreateProject struct {
	PartnerID   string     `json:"partnerId"`
	Role        string     `json:"role"`
	Sets        []string   `json:"sets"`
	Them        *[2]string `json:"them,omitempty"`
	Us          *[2]string `json:"us,omitempty"`
	Group       *[2]string `json:"group,omitempty"`
	Separators  []string   `json:"separators,omitempty"`
	Usage       string     `json:"usage,omitempty"`
	Acknowledge string     `json:"acknowledge,omitempty"`
}

func (c *Client) PartnerProject(ctx context.Context, partnerID string) (*Project, error) {
	project := new(Project)
	err := c.get(ctx, "/edi/lading/projects/by-partner/"+url.PathEscape(partnerID)+"/", project)
	if IsNotFound(err) {
		return nil, nil
	}
	return project, err
}

func (c *Client) CreateProject(ctx context.Context, req *CreateProject) (*Project, error) {
	project := new(Project)
	err := c.post(ctx, "/edi/lading/projects/", req, project)
	return project, err
}

func (c *Client) Versions(ctx context.Context, projectID string) ([]*Version, error) {
	var out struct {
		Items []*Version `json:"items"`
	}
	err := c.get(ctx, "/edi/lading/projects/"+url.PathEscape(projectID)+"/versions/", &out)
	return out.Items, err
}

func (c *Client) VersionDetail(ctx context.Context, id string) (*VersionDetail, error) {
	detail := new(VersionDetail)
	err := c.get(ctx, "/edi/lading/versions/"+url.PathEscape(id)+"/", detail)
	return detail, err
}

func (c *Client) NewDraft(ctx context.Context, projectID, fromVersionID string) (*Version, error) {
	version := new(Version)
	err := c.post(ctx, "/edi/lading/projects/"+url.PathEscape(projectID)+"/drafts/",
		map[string]string{"fromVersionId": fromVersionID}, version)
	return version, err
}

type SaveDraft struct {
	Files   []lading.File   `json:"files"`
	Samples []lading.Sample `json:"samples"`
	Notes   string          `json:"notes"`
	Version int64           `json:"version"`
}

func (c *Client) SaveDraft(
	ctx context.Context,
	versionID string,
	req *SaveDraft,
) (*Version, error) {
	version := new(Version)
	err := c.put(ctx, "/edi/lading/versions/"+url.PathEscape(versionID)+"/", req, version)
	return version, err
}

func (c *Client) TestVersion(ctx context.Context, versionID string) (*lading.TestReport, error) {
	report := new(lading.TestReport)
	err := c.post(ctx, "/edi/lading/versions/"+url.PathEscape(versionID)+"/test/", nil, report)
	return report, err
}

func (c *Client) VersionAction(ctx context.Context, versionID, action string) (*Version, error) {
	version := new(Version)
	err := c.post(ctx, "/edi/lading/versions/"+url.PathEscape(versionID)+"/"+action+"/",
		nil, version)
	return version, err
}

type Replay struct {
	ID      string                `json:"id"`
	Map     string                `json:"map"`
	Status  string                `json:"status"`
	Summary *lading.ReplaySummary `json:"summary"`
	Error   string                `json:"error"`
}

func (c *Client) StartReplay(
	ctx context.Context,
	versionID, mapName string,
	from, to int64,
) (*Replay, error) {
	replay := new(Replay)
	err := c.post(ctx, "/edi/lading/versions/"+url.PathEscape(versionID)+"/replays/",
		map[string]any{"map": mapName, "from": from, "to": to}, replay)
	return replay, err
}

func (c *Client) Replay(ctx context.Context, id string) (*Replay, error) {
	replay := new(Replay)
	err := c.get(ctx, "/edi/lading/replays/"+url.PathEscape(id)+"/", replay)
	return replay, err
}

type FlowInstance struct {
	ID           string              `json:"id"`
	EDIPartnerID string              `json:"ediPartnerId"`
	Flow         string              `json:"flow"`
	Key          string              `json:"key"`
	Status       string              `json:"status"`
	Reason       *wasmmodule.Message `json:"reason"`
	WakeAt       *int64              `json:"wakeAt"`
	ShipmentID   string              `json:"shipmentId"`
	TransferID   string              `json:"transferId"`
	CreatedAt    int64               `json:"createdAt"`
	UpdatedAt    int64               `json:"updatedAt"`
}

func (f *FlowInstance) Why() string {
	if f.Reason == nil || f.Reason.IsZero() {
		return ""
	}
	return f.Reason.String()
}

type FlowEvent struct {
	Kind      string              `json:"kind"`
	Detail    jsonutils.RawJSON   `json:"detail"`
	Error     *wasmmodule.Message `json:"error"`
	MessageID string              `json:"messageId"`
	CreatedAt int64               `json:"createdAt"`
}

type FlowDetail struct {
	Instance *FlowInstance `json:"instance"`
	Events   []*FlowEvent  `json:"events"`
}

func (c *Client) FlowInstances(
	ctx context.Context,
	partnerID, key string,
) ([]*FlowInstance, error) {
	query := url.Values{"limit": {"100"}}
	if partnerID != "" {
		query.Set("partnerId", partnerID)
	}
	if key != "" {
		query.Set("query", key)
	}
	var found page[*FlowInstance]
	if err := c.get(ctx, listPath("/edi/flows/instances/", query), &found); err != nil {
		return nil, err
	}
	if key == "" {
		return found.Results, nil
	}
	out := make([]*FlowInstance, 0, len(found.Results))
	for _, instance := range found.Results {
		if instance.Key == key {
			out = append(out, instance)
		}
	}
	return out, nil
}

func (c *Client) FlowsByStatus(
	ctx context.Context,
	partnerID string,
	statuses []string,
	limit int,
) ([]*FlowInstance, error) {
	query := url.Values{"partnerId": {partnerID}, "limit": {strconv.Itoa(limit)}}
	for _, status := range statuses {
		query.Add("status", status)
	}
	var found page[*FlowInstance]
	err := c.get(ctx, listPath("/edi/flows/instances/", query), &found)
	return found.Results, err
}

func (c *Client) FlowDetail(ctx context.Context, id string) (*FlowDetail, error) {
	detail := new(FlowDetail)
	err := c.get(ctx, "/edi/flows/instances/"+url.PathEscape(id)+"/", detail)
	return detail, err
}

func (c *Client) ReleaseFlow(ctx context.Context, id string) error {
	return c.post(ctx, "/edi/flows/instances/"+url.PathEscape(id)+"/release/", nil, nil)
}

type Transfer struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	TargetShipmentID string `json:"targetShipmentId"`
	SourceShipmentID string `json:"sourceShipmentId"`
	RejectionReason  string `json:"rejectionReason"`
	FailureReason    string `json:"failureReason"`
}

func (c *Client) Transfer(ctx context.Context, id string) (*Transfer, error) {
	transfer := new(Transfer)
	err := c.get(ctx, "/edi/transfers/"+url.PathEscape(id)+"/", transfer)
	return transfer, err
}

func (c *Client) ApproveTransfer(
	ctx context.Context,
	id string,
	mappings []MappingItem,
) (*Transfer, error) {
	if mappings == nil {
		mappings = []MappingItem{}
	}
	transfer := new(Transfer)
	err := c.post(ctx, "/edi/transfers/"+url.PathEscape(id)+"/approve/",
		map[string]any{"mappings": mappings}, transfer)
	return transfer, err
}

func (c *Client) RejectTransfer(ctx context.Context, id, reason string) (*Transfer, error) {
	transfer := new(Transfer)
	err := c.post(ctx, "/edi/transfers/"+url.PathEscape(id)+"/reject/",
		map[string]string{"reason": reason}, transfer)
	return transfer, err
}

type Message struct {
	ID                       string `json:"id"`
	Direction                string `json:"direction"`
	TransactionSet           string `json:"transactionSet"`
	InterchangeControlNumber string `json:"interchangeControlNumber"`
	GroupControlNumber       string `json:"groupControlNumber"`
	TransactionControlNumber string `json:"transactionControlNumber"`
	DeliveryStatus           string `json:"deliveryStatus"`
	AckStatus                string `json:"ackStatus"`
	FlowInstanceID           string `json:"flowInstanceId"`
}

func (c *Client) OutboundMessages(
	ctx context.Context,
	partnerID, set string,
	limit int,
) ([]*Message, error) {
	query := url.Values{
		"partnerId": {partnerID},
		"direction": {"Outbound"},
		"limit":     {strconv.Itoa(limit)},
	}
	if set != "" {
		query.Set("transactionSet", set)
	}
	var found page[*Message]
	err := c.get(ctx, listPath("/edi/messages/", query), &found)
	return found.Results, err
}

func (c *Client) PartnerProfile(
	ctx context.Context,
	partnerID, profileID, name string,
) (*Profile, error) {
	var out struct {
		Profiles struct {
			Edges []struct {
				Node Profile `json:"node"`
			} `json:"edges"`
		} `json:"ediCommunicationProfiles"`
	}
	err := c.GraphQL(ctx, `query EdiSimProfiles($input: DataTableConnectionInput!) {
  ediCommunicationProfiles(input: $input) {
    edges { node { id ediPartnerId method status name config as2InboundPath version } }
  }
}`, map[string]any{"input": map[string]any{"first": 100, "query": name}}, &out)
	if err != nil {
		return nil, err
	}
	for index := range out.Profiles.Edges {
		node := &out.Profiles.Edges[index].Node
		if node.ID == profileID || (profileID == "" && node.EDIPartnerID == partnerID &&
			node.Name == name) {
			return node, nil
		}
	}
	return nil, nil
}
