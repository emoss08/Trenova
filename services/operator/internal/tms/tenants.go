package tms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
)

const (
	tenantPage    = 200
	maxTenantRuns = 50
)

var ErrNotShared = errors.New("the TMS is not a shared server: its tenant routes answer 409")

type TenantAdministrator struct {
	InviteTokenSHA256 string `json:"inviteTokenSha256,omitempty"`
	SetUp             bool   `json:"setUp"`
}

type TenantArchive struct {
	CommandID string    `json:"commandId"`
	Location  string    `json:"location"`
	SizeBytes int64     `json:"sizeBytes"`
	SHA256    string    `json:"sha256"`
	TakenAt   time.Time `json:"takenAt"`
}

type TenantRefusal struct {
	Revision int64  `json:"revision"`
	Code     string `json:"code"`
	Reason   string `json:"reason"`
}

type TenantStatus struct {
	InstanceID      string               `json:"instanceId"`
	AppliedRevision *int64               `json:"appliedRevision,omitempty"`
	State           string               `json:"state"`
	Administrator   *TenantAdministrator `json:"administrator,omitempty"`
	LastArchive     *TenantArchive       `json:"lastArchive,omitempty"`
	Refused         *TenantRefusal       `json:"refused,omitempty"`
}

type TenantReading struct {
	Meter    string    `json:"meter"`
	Hour     time.Time `json:"hour"`
	Quantity int64     `json:"quantity"`
}

type TenantReadings struct {
	InstanceID string          `json:"instanceId"`
	Readings   []TenantReading `json:"readings"`
}

type TenantMeter struct {
	Meter     string `json:"meter"`
	Unit      string `json:"unit"`
	Quantity  int64  `json:"quantity"`
	Allowance int64  `json:"allowance"`
	HardCap   bool   `json:"hardCap"`
	Ceiling   int64  `json:"ceiling,omitempty"`
	Stopped   bool   `json:"stopped"`
}

type TenantUsage struct {
	InstanceID string        `json:"instanceId"`
	Window     *Window       `json:"window,omitempty"`
	Overage    bool          `json:"overage"`
	Meters     []TenantMeter `json:"meters"`
}

type TenantUsageRefused struct {
	InstanceID string `json:"instanceId"`
	Code       string `json:"code"`
	Reason     string `json:"reason,omitempty"`
}

type TenantUsageAnswer struct {
	Applied int                  `json:"applied"`
	Refused []TenantUsageRefused `json:"refused"`
}

func (c *Client) PutTenant(
	ctx context.Context,
	instanceID string,
	entry []byte,
) (TenantStatus, error) {
	u := c.base.JoinPath("managed", "v1", "tenants", instanceID)
	out, err := c.do(ctx, http.MethodPut, u, entry, http.StatusOK)
	if err != nil {
		return TenantStatus{}, fmt.Errorf("apply a tenant on the TMS: %w", sharedOnly(err))
	}
	var status TenantStatus
	if err = sonic.Unmarshal(out, &status); err != nil {
		return TenantStatus{}, fmt.Errorf("apply a tenant on the TMS: %w", err)
	}
	return status, nil
}

func (c *Client) Tenants(ctx context.Context) ([]TenantStatus, error) {
	all := make([]TenantStatus, 0)
	after := ""
	for range maxTenantRuns {
		u := c.base.JoinPath("managed", "v1", "tenants")
		q := url.Values{"limit": {strconv.Itoa(tenantPage)}}
		if after != "" {
			q.Set("after", after)
		}
		u.RawQuery = q.Encode()
		out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
		if err != nil {
			return nil, fmt.Errorf("read the TMS's tenants: %w", sharedOnly(err))
		}
		var page struct {
			Tenants []TenantStatus `json:"tenants"`
			Next    string         `json:"next"`
		}
		if err = sonic.Unmarshal(out, &page); err != nil {
			return nil, fmt.Errorf("read the TMS's tenants: %w", err)
		}
		all = append(all, page.Tenants...)
		if page.Next == "" {
			return all, nil
		}
		after = page.Next
	}
	return nil, errors.New("read the TMS's tenants: more pages than a server holds tenants")
}

func (c *Client) TenantReadings(
	ctx context.Context,
	from, to time.Time,
) ([]TenantReadings, error) {
	all := make([]TenantReadings, 0)
	after := ""
	for range maxTenantRuns {
		u := c.base.JoinPath("managed", "v1", "tenants", "readings")
		q := url.Values{
			"from":  {from.UTC().Format(time.RFC3339)},
			"to":    {to.UTC().Format(time.RFC3339)},
			"limit": {strconv.Itoa(tenantPage)},
		}
		if after != "" {
			q.Set("after", after)
		}
		u.RawQuery = q.Encode()
		out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
		if err != nil {
			return nil, fmt.Errorf("read the tenants' usage: %w", sharedOnly(err))
		}
		var page struct {
			Tenants []TenantReadings `json:"tenants"`
			Next    string           `json:"next"`
		}
		if err = sonic.Unmarshal(out, &page); err != nil {
			return nil, fmt.Errorf("read the tenants' usage: %w", err)
		}
		all = append(all, page.Tenants...)
		if page.Next == "" {
			return all, nil
		}
		after = page.Next
	}
	return nil, errors.New("read the tenants' usage: more pages than a server holds tenants")
}

func (c *Client) PutTenantUsage(
	ctx context.Context,
	tenants []TenantUsage,
) (TenantUsageAnswer, error) {
	for i := range tenants {
		if tenants[i].Meters == nil {
			tenants[i].Meters = []TenantMeter{}
		}
	}
	body, err := sonic.Marshal(struct {
		Tenants []TenantUsage `json:"tenants"`
	}{Tenants: tenants})
	if err != nil {
		return TenantUsageAnswer{}, fmt.Errorf("send the tenants' usage answers: %w", err)
	}
	u := c.base.JoinPath("managed", "v1", "tenants", "usage")
	out, err := c.do(ctx, http.MethodPut, u, body, http.StatusOK)
	if err != nil {
		return TenantUsageAnswer{}, fmt.Errorf(
			"send the tenants' usage answers: %w",
			sharedOnly(err),
		)
	}
	var answer TenantUsageAnswer
	if err = sonic.Unmarshal(out, &answer); err != nil {
		return TenantUsageAnswer{}, fmt.Errorf("send the tenants' usage answers: %w", err)
	}
	return answer, nil
}

func sharedOnly(err error) error {
	var status *StatusError
	if errors.As(err, &status) && status.Status == http.StatusConflict {
		return fmt.Errorf("%w: %w", ErrNotShared, err)
	}
	return err
}
