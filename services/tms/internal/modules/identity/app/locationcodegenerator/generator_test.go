package locationcodegenerator

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type stubSequenceConfigRepo struct {
	doc *sequenceconfig.SequenceConfigDocument
	err error
}

func (r stubSequenceConfigRepo) GetByTenant(
	context.Context,
	tenantports.GetSequenceConfigRequest,
) (*sequenceconfig.SequenceConfigDocument, error) {
	return r.doc, r.err
}

func (r stubSequenceConfigRepo) UpdateByTenant(
	context.Context,
	*sequenceconfig.SequenceConfigDocument,
) (*sequenceconfig.SequenceConfigDocument, error) {
	return nil, nil
}

type captureSequenceGenerator struct {
	testutil.TestSequenceGenerator
	req *seqgen.GenerateRequest
}

func (g *captureSequenceGenerator) Generate(
	_ context.Context,
	req *seqgen.GenerateRequest,
) (string, error) {
	g.req = req
	return g.SingleValue, nil
}

func TestGeneratorGenerateBuildsReadablePrefix(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	seq := &captureSequenceGenerator{
		TestSequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "ACM-DAL-TX-001"},
	}
	generator := &Generator{
		sequenceConfigRepo: sequenceConfigRepo(orgID, buID, nil),
		sequenceGenerator:  seq,
	}

	code, err := generator.Generate(t.Context(), tenantports.LocationCodeGenerateRequest{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Input: tenantports.LocationCodeInput{
			Name:              "Acme Warehouse",
			City:              "Dallas",
			StateAbbreviation: "TX",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "ACM-DAL-TX-001", code)
	require.NotNil(t, seq.req)
	require.Equal(t, sequenceconfig.SequenceTypeLocationCode, seq.req.Type)
	require.Equal(t, fixedLocationCodePeriod, seq.req.Time)
	require.Equal(t, "ACM-DAL-TX", seq.req.Format.Prefix)
	require.Equal(t, 3, seq.req.Format.SequenceDigits)
	require.True(t, seq.req.Format.UseSeparators)
	require.Equal(t, "-", seq.req.Format.SeparatorChar)
}

func TestGeneratorBuildPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    tenantports.LocationCodeInput
		strategy *sequenceconfig.LocationCodeStrategy
		want     string
	}{
		{
			name: "default name city state",
			input: tenantports.LocationCodeInput{
				Name:              "Acme Warehouse",
				City:              "Dallas",
				StateAbbreviation: "TX",
			},
			strategy: nil,
			want:     "ACM-DAL-TX",
		},
		{
			name: "missing city and state collapses adjacent fallback tokens",
			input: tenantports.LocationCodeInput{
				Name: "Acme Warehouse",
			},
			strategy: nil,
			want:     "ACM-LOC",
		},
		{
			name: "short components are preserved",
			input: tenantports.LocationCodeInput{
				Name:              "AB",
				City:              "LA",
				StateAbbreviation: "CA",
			},
			strategy: nil,
			want:     "AB-LA-CA",
		},
		{
			name: "invalid characters are removed",
			input: tenantports.LocationCodeInput{
				Name:              "A*c&m!e",
				City:              "Dal@las",
				StateAbbreviation: "T.X.",
			},
			strategy: nil,
			want:     "ACM-DAL-TX",
		},
		{
			name: "lowercase and empty separator",
			input: tenantports.LocationCodeInput{
				Name:              "Acme Warehouse",
				City:              "Dallas",
				StateAbbreviation: "TX",
			},
			strategy: &sequenceconfig.LocationCodeStrategy{
				Components: []sequenceconfig.LocationCodeComponent{
					sequenceconfig.LocationCodeComponentName,
					sequenceconfig.LocationCodeComponentCity,
				},
				ComponentWidth: 2,
				SequenceDigits: 2,
				Separator:      "",
				Casing:         sequenceconfig.LocationCodeCasingLower,
				FallbackPrefix: "LOC",
			},
			want: "acda",
		},
		{
			name: "postal code component",
			input: tenantports.LocationCodeInput{
				Name:       "Acme Warehouse",
				PostalCode: "75201-1234",
			},
			strategy: &sequenceconfig.LocationCodeStrategy{
				Components: []sequenceconfig.LocationCodeComponent{
					sequenceconfig.LocationCodeComponentName,
					sequenceconfig.LocationCodeComponentPostalCode,
				},
				ComponentWidth: 3,
				SequenceDigits: 3,
				Separator:      "_",
				Casing:         sequenceconfig.LocationCodeCasingUpper,
				FallbackPrefix: "LOC",
			},
			want: "ACM_752",
		},
	}

	generator := &Generator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := generator.BuildPrefix(tt.input, tt.strategy)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGeneratorGenerateUsesDefaultStrategyWhenConfigIsMissing(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	seq := &captureSequenceGenerator{
		TestSequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "AB-LA-CA-001"},
	}
	generator := &Generator{
		sequenceConfigRepo: stubSequenceConfigRepo{
			doc: &sequenceconfig.SequenceConfigDocument{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Configs:        []*sequenceconfig.SequenceConfig{},
			},
		},
		sequenceGenerator: seq,
	}

	_, err := generator.Generate(t.Context(), tenantports.LocationCodeGenerateRequest{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Input: tenantports.LocationCodeInput{
			Name:              "AB",
			City:              "LA",
			StateAbbreviation: "CA",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, seq.req)
	require.Equal(t, "AB-LA-CA", seq.req.Format.Prefix)
	require.Equal(t, 3, seq.req.Format.SequenceDigits)
}

func sequenceConfigRepo(
	orgID, buID pulid.ID,
	strategy *sequenceconfig.LocationCodeStrategy,
) tenantports.SequenceConfigRepository {
	return stubSequenceConfigRepo{
		doc: &sequenceconfig.SequenceConfigDocument{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Configs: []*sequenceconfig.SequenceConfig{
				{
					OrganizationID:       orgID,
					BusinessUnitID:       buID,
					SequenceType:         sequenceconfig.SequenceTypeLocationCode,
					Prefix:               "LOC",
					SequenceDigits:       3,
					LocationCodeStrategy: strategy,
				},
			},
		},
	}
}

var _ tenantports.SequenceConfigRepository = stubSequenceConfigRepo{}
var _ seqgen.Generator = (*captureSequenceGenerator)(nil)
var _ = pagination.TenantInfo{}
