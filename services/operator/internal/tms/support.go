package tms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

type SupportKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	X   string `json:"x"`
}

type SupportKeys struct {
	InstanceID  string       `json:"instanceId"`
	StaffDomain string       `json:"staffDomain"`
	Keys        []SupportKey `json:"keys"`
}

type HeldSupportKeys struct {
	InstanceID  string   `json:"instanceId,omitempty"`
	StaffDomain string   `json:"staffDomain,omitempty"`
	KeyIDs      []string `json:"keyIds"`
	ReceivedAt  string   `json:"receivedAt,omitempty"`
}

type SupportGrant struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	StaffEmail       string    `json:"staffEmail"`
	Scope            string    `json:"scope"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

func (c *Client) SupportKeys(ctx context.Context) (HeldSupportKeys, error) {
	u := c.base.JoinPath("managed", "v1", "support-access")
	out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		return HeldSupportKeys{}, fmt.Errorf("read the TMS's support access keys: %w", err)
	}
	var held HeldSupportKeys
	if err = sonic.Unmarshal(out, &held); err != nil {
		return HeldSupportKeys{}, fmt.Errorf("read the TMS's support access keys: %w", err)
	}
	if held.KeyIDs == nil {
		held.KeyIDs = []string{}
	}
	return held, nil
}

func (c *Client) PutSupportKeys(ctx context.Context, keys SupportKeys) error {
	if keys.Keys == nil {
		keys.Keys = []SupportKey{}
	}
	body, err := sonic.Marshal(keys)
	if err != nil {
		return errors.New("send the support access keys to the TMS: cannot encode them")
	}
	u := c.base.JoinPath("managed", "v1", "support-access")
	if _, err = c.do(ctx, http.MethodPut, u, body, http.StatusNoContent); err != nil {
		return fmt.Errorf("send the support access keys to the TMS: %w", err)
	}
	return nil
}

func (c *Client) SupportGrants(ctx context.Context) ([]SupportGrant, error) {
	u := c.base.JoinPath("managed", "v1", "support-access", "grants")
	out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("read the TMS's live support grants: %w", err)
	}
	var answer struct {
		Grants []SupportGrant `json:"grants"`
	}
	if err = sonic.Unmarshal(out, &answer); err != nil {
		return nil, fmt.Errorf("read the TMS's live support grants: %w", err)
	}
	if answer.Grants == nil {
		answer.Grants = []SupportGrant{}
	}
	return answer.Grants, nil
}
