package carrierintelrule_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateFindings_HealthyCarrierHasNoHits(t *testing.T) {
	t.Parallel()

	findings := partiestest.EvaluateCarrierIntel(partiestest.HealthyCarrierProfile(), nil)
	for _, f := range findings {
		assert.True(t, f.Unverifiable, "unexpected hit %s: %s", f.Code, f.Message)
	}
	assert.Empty(t, carrierintelrule.BlockingCodes(findings))
}

func TestEvaluateFindings_BlockingRules(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		code   carrierintelrule.RuleCode
		mutate func(p *carrierintel.Profile)
	}{
		{
			name: "usdot inactive",
			code: carrierintelrule.RuleUSDOTInactive,
			mutate: func(p *carrierintel.Profile) {
				p.Identity.USDOTStatus = "INACTIVE"
			},
		},
		{
			name: "authority inactive",
			code: carrierintelrule.RuleAuthorityInactive,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.Status = carrierintel.AuthorityStatusInactive
			},
		},
		{
			name: "authority revoked",
			code: carrierintelrule.RuleAuthorityRevoked,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.Status = carrierintel.AuthorityStatusRevoked
			},
		},
		{
			name: "pending revocation",
			code: carrierintelrule.RuleAuthorityPendingRevoke,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.RevocationPending = true
			},
		},
		{
			name: "no bipd on file",
			code: carrierintelrule.RuleInsuranceNoneOnFile,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.BIPDOnFile = partiestest.Decimal(0)
			},
		},
		{
			name: "bipd below required",
			code: carrierintelrule.RuleInsuranceBIPDBelow,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.BIPDOnFile = partiestest.Decimal(500_000)
			},
		},
		{
			name: "cargo below required",
			code: carrierintelrule.RuleInsuranceCargoBelow,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.CargoOnFile = partiestest.Decimal(1_000)
			},
		},
		{
			name: "pending cancellation",
			code: carrierintelrule.RuleInsurancePendingCancel,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.PendingCancelAt = new(
					partiestest.CarrierIntelNow + 10*timeutils.SecondsPerDay,
				)
			},
		},
		{
			name: "out of service order",
			code: carrierintelrule.RuleSafetyOOSOrder,
			mutate: func(p *carrierintel.Profile) {
				p.Safety.OutOfServiceOrder = new(true)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := partiestest.HealthyCarrierProfile()
			tc.mutate(p)
			finding := partiestest.CarrierIntelFinding(
				partiestest.EvaluateCarrierIntel(p, nil),
				tc.code,
			)
			require.NotNil(t, finding)
			assert.Equal(t, carrierintel.RuleActionBlock, finding.Action)
			assert.True(t, finding.IsBlocking())
			assert.NotEmpty(t, finding.Message)
		})
	}
}

func TestEvaluateFindings_BIPDOrganizationMinimum(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	findings := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Settings = carrierintelrule.RuleSettings{
			carrierintelrule.RuleInsuranceBIPDBelow: {
				Action: carrierintel.RuleActionBlock,
				Params: map[string]string{"minimum": "2000000"},
			},
		}
	})

	finding := partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleInsuranceBIPDBelow)
	require.NotNil(t, finding)
	assert.Contains(t, finding.Message, "$2,000,000")
}

func TestEvaluateFindings_PendingCancellationIgnoresPastFilingCancellations(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Insurance.Filings = []carrierintel.InsuranceFiling{
		{
			Type:              carrierintel.InsuranceFilingTypeBIPD,
			Status:            "H",
			CancelEffectiveAt: new(partiestest.CarrierIntelNow - 400*timeutils.SecondsPerDay),
		},
		{
			Type:              carrierintel.InsuranceFilingTypeCargo,
			Status:            "A",
			CancelEffectiveAt: new(partiestest.CarrierIntelNow + 12*timeutils.SecondsPerDay),
		},
	}
	finding := partiestest.CarrierIntelFinding(
		partiestest.EvaluateCarrierIntel(p, nil),
		carrierintelrule.RuleInsurancePendingCancel,
	)
	require.NotNil(t, finding)
	assert.Equal(t, "An insurance cancellation takes effect in 12 days", finding.Message)

	p.Insurance.Filings = p.Insurance.Filings[:1]
	assert.Nil(
		t,
		partiestest.CarrierIntelFinding(
			partiestest.EvaluateCarrierIntel(p, nil),
			carrierintelrule.RuleInsurancePendingCancel,
		),
	)
}

func TestEvaluateFindings_PendingCancellationOutsideWindow(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Insurance.PendingCancelAt = new(partiestest.CarrierIntelNow + 60*timeutils.SecondsPerDay)
	assert.Nil(
		t,
		partiestest.CarrierIntelFinding(
			partiestest.EvaluateCarrierIntel(p, nil),
			carrierintelrule.RuleInsurancePendingCancel,
		),
	)
}

func TestEvaluateFindings_OffActionSkipsRule(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Safety.OutOfServiceOrder = new(true)
	findings := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Settings = carrierintelrule.RuleSettings{
			carrierintelrule.RuleSafetyOOSOrder: {Action: carrierintel.RuleActionOff},
		}
	})

	assert.Nil(t, partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleSafetyOOSOrder))
}

func TestEvaluateFindings_UnverifiableWhenSectionMissing(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Network = nil
	p.Coverage = nil
	p.NormalizeCoverage()
	finding := partiestest.CarrierIntelFinding(
		partiestest.EvaluateCarrierIntel(p, nil),
		carrierintelrule.RuleFraudNetworkSharing,
	)
	require.NotNil(t, finding)
	assert.True(t, finding.Unverifiable)
	assert.False(t, finding.IsBlocking())
	assert.False(t, finding.IsAdvisory())
}

func TestEvaluateFindings_BrokerAuthorityAndBond(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Authority.Broker = &carrierintel.AuthorityGrant{Status: carrierintel.AuthorityStatusInactive}
	p.Insurance.BondOnFile = partiestest.Decimal(10_000)
	findings := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Subject = carrierintel.SubjectTypeCustomer
		in.BrokerAuthority = true
	})

	require.NotNil(
		t,
		partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleAuthorityInactive),
	)
	bond := partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleInsuranceBondBelow)
	require.NotNil(t, bond)
	assert.Contains(t, bond.Message, "$75,000")
	assert.Nil(
		t,
		partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleInsuranceBIPDBelow),
	)
	assert.Nil(
		t,
		partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleFraudNetworkSharing),
	)
}

func TestEvaluateFindings_ExemptCarrierSkipsAuthority(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Authority.Common.Status = carrierintel.AuthorityStatusNone
	findings := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.AuthorityExempt = true
	})

	assert.Nil(t, partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleAuthorityInactive))
}

func TestEvaluateFindings_WarnRules(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Safety.Rating = carrierintel.SafetyRatingConditional
	p.Safety.ISSValue = new(88)
	p.Basics[0].Alert = true
	p.Authority.Common.AgeDays = new(45)
	p.Network.SharedPhones = new(3)
	p.ChangeHistory.PhoneChanges = new(3)
	p.ChangeHistory.PhoneLastChangedAt = new(
		partiestest.CarrierIntelNow - 5*timeutils.SecondsPerDay,
	)
	p.Operations.MCS150At = new(partiestest.CarrierIntelNow - 800*timeutils.SecondsPerDay)

	findings := partiestest.EvaluateCarrierIntel(p, nil)
	for _, code := range []carrierintelrule.RuleCode{
		carrierintelrule.RuleSafetyConditional,
		carrierintelrule.RuleSafetyISSHigh,
		carrierintelrule.RuleBasicsAlert,
		carrierintelrule.RuleIdentityNewEntrant,
		carrierintelrule.RuleFraudNetworkSharing,
		carrierintelrule.RuleFraudContactChurn,
	} {
		finding := partiestest.CarrierIntelFinding(findings, code)
		require.NotNil(t, finding, code)
		assert.Equal(t, carrierintel.RuleActionWarn, finding.Action, code)
		assert.True(t, finding.IsAdvisory(), code)
	}
	mcs := partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleOperationsMCS150Stale)
	require.NotNil(t, mcs)
	assert.Equal(t, carrierintel.RuleActionNotify, mcs.Action)
	assert.Empty(t, carrierintelrule.BlockingCodes(findings))
	assert.Equal(t, carrierintel.RiskLevelHigh, carrierintelrule.DeriveRiskLevel(p, findings))
}

func TestEvaluateFindings_BasicsPercentileOverride(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	findings := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Settings = carrierintelrule.RuleSettings{
			carrierintelrule.RuleBasicsAlert: {
				Action: carrierintel.RuleActionWarn,
				Params: map[string]string{"basics": "HOSCompliance", "percentile": "30"},
			},
		}
	})

	finding := partiestest.CarrierIntelFinding(findings, carrierintelrule.RuleBasicsAlert)
	require.NotNil(t, finding)
	assert.Contains(t, finding.Message, "HOSCompliance")
	assert.NotContains(t, finding.Message, "UnsafeDriving")
}

func TestEvaluateFindings_Overrides(t *testing.T) {
	t.Parallel()

	p := partiestest.HealthyCarrierProfile()
	p.Safety.OutOfServiceOrder = new(true)
	overrideID := pulid.MustNew("ciovr_")

	active := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Overrides = []carrierintelrule.OverrideRef{
			{
				ID:        overrideID,
				RuleCode:  carrierintelrule.RuleSafetyOOSOrder,
				ExpiresAt: partiestest.CarrierIntelNow + 100,
			},
		}
	})

	finding := partiestest.CarrierIntelFinding(active, carrierintelrule.RuleSafetyOOSOrder)
	require.NotNil(t, finding)
	assert.True(t, finding.Overridden)
	assert.Equal(t, overrideID, finding.OverrideID)
	assert.False(t, finding.IsBlocking())
	assert.True(t, finding.IsAdvisory())

	expired := partiestest.EvaluateCarrierIntel(p, func(in *carrierintelrule.EvaluateInput) {
		in.Overrides = []carrierintelrule.OverrideRef{
			{
				ID:        overrideID,
				RuleCode:  carrierintelrule.RuleSafetyOOSOrder,
				ExpiresAt: partiestest.CarrierIntelNow - 1,
			},
		}
	})

	assert.True(
		t,
		partiestest.CarrierIntelFinding(expired, carrierintelrule.RuleSafetyOOSOrder).IsBlocking(),
	)
}

func TestEvaluateFindings_NotFound(t *testing.T) {
	t.Parallel()

	findings := carrierintelrule.EvaluateFindings(&carrierintelrule.EvaluateInput{
		Now:      partiestest.CarrierIntelNow,
		Subject:  carrierintel.SubjectTypeCarrier,
		NotFound: true,
	})
	require.Len(t, findings, 1)
	assert.Equal(t, carrierintelrule.RuleIdentityNotFound, findings[0].Code)
	assert.Equal(t, carrierintel.RuleActionWarn, findings[0].Action)
}

func TestNewlyRaisedAndNewBlockingCodes(t *testing.T) {
	t.Parallel()

	prior := []carrierintelrule.Finding{
		{Code: carrierintelrule.RuleSafetyOOSOrder, Action: carrierintel.RuleActionBlock},
		{Code: carrierintelrule.RuleSafetyISSHigh, Action: carrierintel.RuleActionWarn},
	}
	current := []carrierintelrule.Finding{
		{Code: carrierintelrule.RuleSafetyOOSOrder, Action: carrierintel.RuleActionBlock},
		{Code: carrierintelrule.RuleAuthorityRevoked, Action: carrierintel.RuleActionBlock},
		{Code: carrierintelrule.RuleSafetyISSHigh, Action: carrierintel.RuleActionWarn},
		{
			Code:         carrierintelrule.RuleFraudNetworkSharing,
			Action:       carrierintel.RuleActionWarn,
			Unverifiable: true,
		},
	}

	added := carrierintelrule.NewBlockingCodes(prior, current)
	require.Len(t, added, 1)
	assert.Equal(t, carrierintelrule.RuleAuthorityRevoked, added[0].Code)

	raised := carrierintelrule.NewlyRaised(prior, current)
	require.Len(t, raised, 1)
	assert.Equal(t, carrierintelrule.RuleAuthorityRevoked, raised[0].Code)
}

func TestCatalogIntegrity(t *testing.T) {
	t.Parallel()

	seen := map[carrierintelrule.RuleCode]bool{}
	for _, def := range carrierintelrule.Catalog() {
		assert.False(t, seen[def.Code], "duplicate %s", def.Code)
		seen[def.Code] = true
		assert.True(t, def.DefaultAction.IsValid())
		assert.True(t, def.RecommendedAction.IsValid())
		assert.NotEmpty(t, def.Label)
		assert.True(t, def.Code.IsValid())
		for _, param := range def.Params {
			assert.NotEmpty(t, param.Label)
		}
	}
}
