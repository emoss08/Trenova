package supportkeys

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	MaxGrants     = 100
	maxGrantID    = 100
	maxOrgName    = 200
	maxStaffEmail = 254
)

var organizationID = regexp.MustCompile(`^org_[0-9A-HJKMNP-TV-Z]{26}$`)

type TMS interface {
	SupportKeys(ctx context.Context) (tms.HeldSupportKeys, error)
	PutSupportKeys(ctx context.Context, keys tms.SupportKeys) error
	DeleteSupportKeys(ctx context.Context) error
}

type Relay struct {
	tms    TMS
	logger *slog.Logger
}

func New(t TMS, logger *slog.Logger) *Relay {
	return &Relay{tms: t, logger: logger}
}

func (r *Relay) Sync(ctx context.Context, desired *instanceapi.SpecSupport) error {
	if desired == nil {
		return nil
	}
	return r.deliver(ctx, desired.InstanceId, desired.StaffDomain, desired.Keys)
}

func (r *Relay) SyncShared(ctx context.Context, desired *instanceapi.SharedSupport) error {
	if desired != nil {
		return r.deliver(ctx, "", desired.StaffDomain, desired.Keys)
	}
	held, err := r.tms.SupportKeys(ctx)
	if err != nil {
		return err
	}
	if len(held.KeyIDs) == 0 {
		return nil
	}
	if err = r.tms.DeleteSupportKeys(ctx); err != nil {
		return err
	}
	r.logger.Info(
		"support access is off for the server; its keys were removed from the TMS",
		"removedKeyIds", held.KeyIDs,
	)
	return nil
}

func (r *Relay) deliver(
	ctx context.Context,
	instanceID, staffDomain string,
	given []instanceapi.SupportKey,
) error {
	want := keyIDs(given)
	held, err := r.tms.SupportKeys(ctx)
	if err != nil {
		return err
	}
	if held.InstanceID == instanceID &&
		held.StaffDomain == staffDomain &&
		sameSet(held.KeyIDs, want) {
		return nil
	}
	keys := make([]tms.SupportKey, 0, len(given))
	for _, k := range given {
		keys = append(keys, tms.SupportKey{
			Kid: k.Kid,
			Kty: k.Kty,
			Crv: k.Crv,
			Alg: k.Alg,
			Use: k.Use,
			X:   k.X,
		})
	}
	if err = r.tms.PutSupportKeys(ctx, tms.SupportKeys{
		InstanceID:  instanceID,
		StaffDomain: staffDomain,
		Keys:        keys,
	}); err != nil {
		return fmt.Errorf("key ids %s: %w", strings.Join(want, ","), err)
	}
	r.logger.Info(
		"support access keys sent to the TMS",
		"staffDomain", staffDomain,
		"keyIds", want,
		"replacedKeyIds", held.KeyIDs,
	)
	return nil
}

func keyIDs(keys []instanceapi.SupportKey) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.Kid)
	}
	return out
}

func sameSet(held, want []string) bool {
	a, b := slices.Clone(held), slices.Clone(want)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}

func Grants(held []tms.SupportGrant) (grants []instanceapi.SupportGrant, dropped []string) {
	grants = make([]instanceapi.SupportGrant, 0, min(len(held), MaxGrants))
	seen := make(map[string]bool, len(held))
	for i := range held {
		if len(grants) == MaxGrants {
			break
		}
		grant, ok := reported(&held[i])
		if !ok || seen[grant.Id] {
			dropped = append(dropped, held[i].ID)
			continue
		}
		seen[grant.Id] = true
		grants = append(grants, grant)
	}
	return grants, dropped
}

func reported(g *tms.SupportGrant) (instanceapi.SupportGrant, bool) {
	grant := instanceapi.SupportGrant{
		Id:               g.ID,
		OrganizationId:   g.OrganizationID,
		OrganizationName: clip(strings.TrimSpace(g.OrganizationName), maxOrgName),
		StaffEmail:       strings.ToLower(g.StaffEmail),
		Scope:            instanceapi.SupportGrantScope(g.Scope),
		ExpiresAt:        g.ExpiresAt.UTC(),
	}
	ok := grant.Id != "" && len(grant.Id) <= maxGrantID &&
		organizationID.MatchString(grant.OrganizationId) &&
		grant.OrganizationName != "" &&
		len(grant.StaffEmail) <= maxStaffEmail &&
		grant.Scope.Valid() && !grant.ExpiresAt.IsZero()
	return grant, ok
}

func SameGrants(held, last *[]instanceapi.SupportGrant) bool {
	if held == nil {
		return true
	}
	if last == nil || len(*held) != len(*last) {
		return false
	}
	for i := range *held {
		a, b := (*held)[i], (*last)[i]
		if a.Id != b.Id || a.Scope != b.Scope || !a.ExpiresAt.Equal(b.ExpiresAt) ||
			a.OrganizationName != b.OrganizationName || a.StaffEmail != b.StaffEmail {
			return false
		}
	}
	return true
}

func clip(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
