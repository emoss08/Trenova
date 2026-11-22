package documenttemplateservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/emaildesign"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/zap"
)

const (
	brandTTL        = 10 * time.Minute
	brandCacheBytes = 64 << 20
	brandEntryBase  = 256
)

type brandBook struct {
	orgs    tenantports.OrganizationRepository
	inliner documenttemplateports.AssetInliner
	cache   *otter.Cache[pulid.ID, emaildesign.Brand]
	l       *zap.Logger
}

func newBrandBook(
	orgs tenantports.OrganizationRepository,
	inliner documenttemplateports.AssetInliner,
	l *zap.Logger,
) *brandBook {
	return &brandBook{
		orgs:    orgs,
		inliner: inliner,
		l:       l,
		cache: otter.Must(&otter.Options[pulid.ID, emaildesign.Brand]{
			MaximumWeight:    brandCacheBytes,
			Weigher:          brandWeight,
			ExpiryCalculator: otter.ExpiryWriting[pulid.ID, emaildesign.Brand](brandTTL),
		}),
	}
}

func brandWeight(_ pulid.ID, brand emaildesign.Brand) uint32 {
	weight := min(len(brand.Name)+len(brand.Mark)+brandEntryBase, brandCacheBytes)
	return uint32(weight)
}

func (b *brandBook) brandFor(ctx context.Context, tenant pagination.TenantInfo) emaildesign.Brand {
	if b == nil || tenant.OrgID.IsNil() {
		return emaildesign.Brand{}
	}
	if brand, ok := b.cache.GetIfPresent(tenant.OrgID); ok {
		return brand
	}

	org, err := b.orgs.GetByID(ctx, tenantports.GetOrganizationByIDRequest{TenantInfo: tenant})
	if err != nil {
		b.l.Warn("could not read the organization for an email header; sending it without one",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err))
		return emaildesign.Brand{}
	}

	brand := emaildesign.Brand{Name: org.Name}
	if source := stringutils.FirstNonEmpty(org.IconURL, org.LogoURL); source != "" &&
		b.inliner != nil {
		mark, markErr := b.inliner.ResolveImageDataURI(ctx, source)
		if markErr != nil {
			b.l.Warn("could not load the organization mark for an email header",
				zap.String("organizationId", tenant.OrgID.String()),
				zap.Error(markErr))
		}
		brand.Mark = mark
	}

	b.cache.Set(tenant.OrgID, brand)
	return brand
}
