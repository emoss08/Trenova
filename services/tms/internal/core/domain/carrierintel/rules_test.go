package carrierintel_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNow = int64(1_790_000_000)

func dec(v int64) *decimal.Decimal {
	d := decimal.NewFromInt(v)
	return &d
}

func healthyProfile() *carrierintel.Profile {
	p := &carrierintel.Profile{
		Identity: &carrierintel.Identity{
			DOTNumber:    "818175",
			DocketNumber: "277621",
			LegalName:    "ACME FREIGHT LLC",
			USDOTStatus:  "ACTIVE",
		},
		Authority: &carrierintel.Authority{
			Common: &carrierintel.AuthorityGrant{
				Status:  carrierintel.AuthorityStatusActive,
				AgeDays: new(900),
			},
			Contract: &carrierintel.AuthorityGrant{Status: carrierintel.AuthorityStatusNone},
			Broker:   &carrierintel.AuthorityGrant{Status: carrierintel.AuthorityStatusNone},
		},
		Insurance: &carrierintel.Insurance{
			BIPDOnFile:    dec(1_000_000),
			BIPDRequired:  dec(750_000),
			CargoOnFile:   dec(100_000),
			CargoRequired: dec(5_000),
		},
		Safety: &carrierintel.Safety{
			Rating:            carrierintel.SafetyRatingSatisfactory,
			ISSValue:          new(40),
			RiskScore:         carrierintel.RiskLevelLow,
			OutOfServiceOrder: new(false),
		},
		Basics: []carrierintel.BasicMeasure{
			{Basic: worker.BasicUnsafeDriving, Percentile: new(20.0)},
			{Basic: worker.BasicHOSCompliance, Percentile: new(35.0)},
		},
		Operations: &carrierintel.Operations{MCS150At: new(testNow - 100*timeutils.SecondsPerDay)},
		Network:    &carrierintel.Network{SharedPhones: new(0)},
		ChangeHistory: &carrierintel.ChangeHistory{
			PhoneChanges: new(0),
		},
	}
	p.NormalizeCoverage()
	return p
}

func evaluate(
	p *carrierintel.Profile,
	mutate func(in *carrierintel.EvaluateInput),
) []carrierintel.Finding {
	in := &carrierintel.EvaluateInput{
		Profile: p,
		Now:     testNow,
		Subject: carrierintel.SubjectTypeCarrier,
	}
	if mutate != nil {
		mutate(in)
	}
	return carrierintel.EvaluateFindings(in)
}

func findingByCode(
	findings []carrierintel.Finding,
	code carrierintel.RuleCode,
) *carrierintel.Finding {
	for idx := range findings {
		if findings[idx].Code == code {
			return &findings[idx]
		}
	}
	return nil
}

func TestEvaluateFindings_HealthyCarrierHasNoHits(t *testing.T) {
	t.Parallel()

	findings := evaluate(healthyProfile(), nil)
	for _, f := range findings {
		assert.True(t, f.Unverifiable, "unexpected hit %s: %s", f.Code, f.Message)
	}
	assert.Empty(t, carrierintel.BlockingCodes(findings))
}

func TestEvaluateFindings_BlockingRules(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		code   carrierintel.RuleCode
		mutate func(p *carrierintel.Profile)
	}{
		{
			name: "usdot inactive",
			code: carrierintel.RuleUSDOTInactive,
			mutate: func(p *carrierintel.Profile) {
				p.Identity.USDOTStatus = "INACTIVE"
			},
		},
		{
			name: "authority inactive",
			code: carrierintel.RuleAuthorityInactive,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.Status = carrierintel.AuthorityStatusInactive
			},
		},
		{
			name: "authority revoked",
			code: carrierintel.RuleAuthorityRevoked,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.Status = carrierintel.AuthorityStatusRevoked
			},
		},
		{
			name: "pending revocation",
			code: carrierintel.RuleAuthorityPendingRevoke,
			mutate: func(p *carrierintel.Profile) {
				p.Authority.Common.RevocationPending = true
			},
		},
		{
			name: "no bipd on file",
			code: carrierintel.RuleInsuranceNoneOnFile,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.BIPDOnFile = dec(0)
			},
		},
		{
			name: "bipd below required",
			code: carrierintel.RuleInsuranceBIPDBelow,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.BIPDOnFile = dec(500_000)
			},
		},
		{
			name: "cargo below required",
			code: carrierintel.RuleInsuranceCargoBelow,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.CargoOnFile = dec(1_000)
			},
		},
		{
			name: "pending cancellation",
			code: carrierintel.RuleInsurancePendingCancel,
			mutate: func(p *carrierintel.Profile) {
				p.Insurance.PendingCancelAt = new(testNow + 10*timeutils.SecondsPerDay)
			},
		},
		{
			name: "out of service order",
			code: carrierintel.RuleSafetyOOSOrder,
			mutate: func(p *carrierintel.Profile) {
				p.Safety.OutOfServiceOrder = new(true)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := healthyProfile()
			tc.mutate(p)
			finding := findingByCode(evaluate(p, nil), tc.code)
			require.NotNil(t, finding)
			assert.Equal(t, carrierintel.RuleActionBlock, finding.Action)
			assert.True(t, finding.IsBlocking())
			assert.NotEmpty(t, finding.Message)
		})
	}
}

func TestEvaluateFindings_BIPDOrganizationMinimum(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	findings := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Settings = carrierintel.RuleSettings{
			carrierintel.RuleInsuranceBIPDBelow: {
				Action: carrierintel.RuleActionBlock,
				Params: map[string]string{"minimum": "2000000"},
			},
		}
	})
	finding := findingByCode(findings, carrierintel.RuleInsuranceBIPDBelow)
	require.NotNil(t, finding)
	assert.Contains(t, finding.Message, "$2,000,000")
}

func TestEvaluateFindings_PendingCancellationIgnoresPastFilingCancellations(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Insurance.Filings = []carrierintel.InsuranceFiling{
		{
			Type:              carrierintel.InsuranceFilingTypeBIPD,
			Status:            "H",
			CancelEffectiveAt: new(testNow - 400*timeutils.SecondsPerDay),
		},
		{
			Type:              carrierintel.InsuranceFilingTypeCargo,
			Status:            "A",
			CancelEffectiveAt: new(testNow + 12*timeutils.SecondsPerDay),
		},
	}
	finding := findingByCode(evaluate(p, nil), carrierintel.RuleInsurancePendingCancel)
	require.NotNil(t, finding)
	assert.Equal(t, "An insurance cancellation takes effect in 12 days", finding.Message)

	p.Insurance.Filings = p.Insurance.Filings[:1]
	assert.Nil(t, findingByCode(evaluate(p, nil), carrierintel.RuleInsurancePendingCancel))
}

func TestEvaluateFindings_PendingCancellationOutsideWindow(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Insurance.PendingCancelAt = new(testNow + 60*timeutils.SecondsPerDay)
	assert.Nil(t, findingByCode(evaluate(p, nil), carrierintel.RuleInsurancePendingCancel))
}

func TestEvaluateFindings_OffActionSkipsRule(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Safety.OutOfServiceOrder = new(true)
	findings := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Settings = carrierintel.RuleSettings{
			carrierintel.RuleSafetyOOSOrder: {Action: carrierintel.RuleActionOff},
		}
	})
	assert.Nil(t, findingByCode(findings, carrierintel.RuleSafetyOOSOrder))
}

func TestEvaluateFindings_UnverifiableWhenSectionMissing(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Network = nil
	p.Coverage = nil
	p.NormalizeCoverage()
	finding := findingByCode(evaluate(p, nil), carrierintel.RuleFraudNetworkSharing)
	require.NotNil(t, finding)
	assert.True(t, finding.Unverifiable)
	assert.False(t, finding.IsBlocking())
	assert.False(t, finding.IsAdvisory())
}

func TestEvaluateFindings_BrokerAuthorityAndBond(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Authority.Broker = &carrierintel.AuthorityGrant{Status: carrierintel.AuthorityStatusInactive}
	p.Insurance.BondOnFile = dec(10_000)
	findings := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Subject = carrierintel.SubjectTypeCustomer
		in.BrokerAuthority = true
	})
	require.NotNil(t, findingByCode(findings, carrierintel.RuleAuthorityInactive))
	bond := findingByCode(findings, carrierintel.RuleInsuranceBondBelow)
	require.NotNil(t, bond)
	assert.Contains(t, bond.Message, "$75,000")
	assert.Nil(t, findingByCode(findings, carrierintel.RuleInsuranceBIPDBelow))
	assert.Nil(t, findingByCode(findings, carrierintel.RuleFraudNetworkSharing))
}

func TestEvaluateFindings_ExemptCarrierSkipsAuthority(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Authority.Common.Status = carrierintel.AuthorityStatusNone
	findings := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.AuthorityExempt = true
	})
	assert.Nil(t, findingByCode(findings, carrierintel.RuleAuthorityInactive))
}

func TestEvaluateFindings_WarnRules(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Safety.Rating = carrierintel.SafetyRatingConditional
	p.Safety.ISSValue = new(88)
	p.Basics[0].Alert = true
	p.Authority.Common.AgeDays = new(45)
	p.Network.SharedPhones = new(3)
	p.ChangeHistory.PhoneChanges = new(3)
	p.ChangeHistory.PhoneLastChangedAt = new(testNow - 5*timeutils.SecondsPerDay)
	p.Operations.MCS150At = new(testNow - 800*timeutils.SecondsPerDay)

	findings := evaluate(p, nil)
	for _, code := range []carrierintel.RuleCode{
		carrierintel.RuleSafetyConditional,
		carrierintel.RuleSafetyISSHigh,
		carrierintel.RuleBasicsAlert,
		carrierintel.RuleIdentityNewEntrant,
		carrierintel.RuleFraudNetworkSharing,
		carrierintel.RuleFraudContactChurn,
	} {
		finding := findingByCode(findings, code)
		require.NotNil(t, finding, code)
		assert.Equal(t, carrierintel.RuleActionWarn, finding.Action, code)
		assert.True(t, finding.IsAdvisory(), code)
	}
	mcs := findingByCode(findings, carrierintel.RuleOperationsMCS150Stale)
	require.NotNil(t, mcs)
	assert.Equal(t, carrierintel.RuleActionNotify, mcs.Action)
	assert.Empty(t, carrierintel.BlockingCodes(findings))
	assert.Equal(t, carrierintel.RiskLevelHigh, p.DeriveRiskLevel(findings))
}

func TestEvaluateFindings_BasicsPercentileOverride(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	findings := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Settings = carrierintel.RuleSettings{
			carrierintel.RuleBasicsAlert: {
				Action: carrierintel.RuleActionWarn,
				Params: map[string]string{"basics": "HOSCompliance", "percentile": "30"},
			},
		}
	})
	finding := findingByCode(findings, carrierintel.RuleBasicsAlert)
	require.NotNil(t, finding)
	assert.Contains(t, finding.Message, "HOSCompliance")
	assert.NotContains(t, finding.Message, "UnsafeDriving")
}

func TestEvaluateFindings_Overrides(t *testing.T) {
	t.Parallel()

	p := healthyProfile()
	p.Safety.OutOfServiceOrder = new(true)
	overrideID := pulid.MustNew("ciovr_")

	active := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Overrides = []carrierintel.OverrideRef{
			{ID: overrideID, RuleCode: carrierintel.RuleSafetyOOSOrder, ExpiresAt: testNow + 100},
		}
	})
	finding := findingByCode(active, carrierintel.RuleSafetyOOSOrder)
	require.NotNil(t, finding)
	assert.True(t, finding.Overridden)
	assert.Equal(t, overrideID, finding.OverrideID)
	assert.False(t, finding.IsBlocking())
	assert.True(t, finding.IsAdvisory())

	expired := evaluate(p, func(in *carrierintel.EvaluateInput) {
		in.Overrides = []carrierintel.OverrideRef{
			{ID: overrideID, RuleCode: carrierintel.RuleSafetyOOSOrder, ExpiresAt: testNow - 1},
		}
	})
	assert.True(t, findingByCode(expired, carrierintel.RuleSafetyOOSOrder).IsBlocking())
}

func TestEvaluateFindings_NotFound(t *testing.T) {
	t.Parallel()

	findings := carrierintel.EvaluateFindings(&carrierintel.EvaluateInput{
		Now:      testNow,
		Subject:  carrierintel.SubjectTypeCarrier,
		NotFound: true,
	})
	require.Len(t, findings, 1)
	assert.Equal(t, carrierintel.RuleIdentityNotFound, findings[0].Code)
	assert.Equal(t, carrierintel.RuleActionWarn, findings[0].Action)
}

func TestNewlyRaisedAndNewBlockingCodes(t *testing.T) {
	t.Parallel()

	prior := []carrierintel.Finding{
		{Code: carrierintel.RuleSafetyOOSOrder, Action: carrierintel.RuleActionBlock},
		{Code: carrierintel.RuleSafetyISSHigh, Action: carrierintel.RuleActionWarn},
	}
	current := []carrierintel.Finding{
		{Code: carrierintel.RuleSafetyOOSOrder, Action: carrierintel.RuleActionBlock},
		{Code: carrierintel.RuleAuthorityRevoked, Action: carrierintel.RuleActionBlock},
		{Code: carrierintel.RuleSafetyISSHigh, Action: carrierintel.RuleActionWarn},
		{
			Code:         carrierintel.RuleFraudNetworkSharing,
			Action:       carrierintel.RuleActionWarn,
			Unverifiable: true,
		},
	}

	added := carrierintel.NewBlockingCodes(prior, current)
	require.Len(t, added, 1)
	assert.Equal(t, carrierintel.RuleAuthorityRevoked, added[0].Code)

	raised := carrierintel.NewlyRaised(prior, current)
	require.Len(t, raised, 1)
	assert.Equal(t, carrierintel.RuleAuthorityRevoked, raised[0].Code)
}

func TestValidateRuleSettings(t *testing.T) {
	t.Parallel()

	control := carrierintel.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	control.Rules = carrierintel.RuleSettings{
		carrierintel.RuleSafetyISSHigh: {
			Action: carrierintel.RuleActionWarn,
			Params: map[string]string{"minimum": "150"},
		},
		carrierintel.RuleCode("nope.rule"): {Action: carrierintel.RuleActionWarn},
		carrierintel.RuleBasicsAlert: {
			Action: carrierintel.RuleAction("Maybe"),
			Params: map[string]string{"basics": "UnsafeDriving,Bogus"},
		},
	}

	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "rules.safety.iss_high.params.minimum")
	assert.Contains(t, fields, "rules.nope.rule")
	assert.Contains(t, fields, "rules.basics.alert.action")
	assert.Contains(t, fields, "rules.basics.alert.params.basics")
}

func TestControlValidate_DefaultsAreValid(t *testing.T) {
	t.Parallel()

	control := carrierintel.NewDefaultControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestCatalogIntegrity(t *testing.T) {
	t.Parallel()

	seen := map[carrierintel.RuleCode]bool{}
	for _, def := range carrierintel.Catalog() {
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
