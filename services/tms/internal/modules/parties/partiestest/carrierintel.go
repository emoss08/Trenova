package partiestest

import (
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelrule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const CarrierIntelNow = int64(1_790_000_000)

func Decimal(v int64) *decimal.Decimal {
	d := decimal.NewFromInt(v)
	return &d
}

func HealthyCarrierProfile() *carrierintel.Profile {
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
			BIPDOnFile:    Decimal(1_000_000),
			BIPDRequired:  Decimal(750_000),
			CargoOnFile:   Decimal(100_000),
			CargoRequired: Decimal(5_000),
		},
		Safety: &carrierintel.Safety{
			Rating:            carrierintel.SafetyRatingSatisfactory,
			ISSValue:          new(40),
			RiskScore:         carrierintel.RiskLevelLow,
			OutOfServiceOrder: new(false),
		},
		Basics: []carrierintel.BasicMeasure{
			{Basic: fleetsafety.BasicUnsafeDriving, Percentile: new(20.0)},
			{Basic: fleetsafety.BasicHOSCompliance, Percentile: new(35.0)},
		},
		Operations: &carrierintel.Operations{
			MCS150At: new(CarrierIntelNow - 100*timeutils.SecondsPerDay),
		},
		Network: &carrierintel.Network{SharedPhones: new(0)},
		ChangeHistory: &carrierintel.ChangeHistory{
			PhoneChanges: new(0),
		},
	}
	p.NormalizeCoverage()
	return p
}

func EvaluateCarrierIntel(
	p *carrierintel.Profile,
	mutate func(in *carrierintelrule.EvaluateInput),
) []carrierintelrule.Finding {
	in := &carrierintelrule.EvaluateInput{
		Profile: p,
		Now:     CarrierIntelNow,
		Subject: carrierintel.SubjectTypeCarrier,
	}
	if mutate != nil {
		mutate(in)
	}
	return carrierintelrule.EvaluateFindings(in)
}

func CarrierIntelFinding(
	findings []carrierintelrule.Finding,
	code carrierintelrule.RuleCode,
) *carrierintelrule.Finding {
	for idx := range findings {
		if findings[idx].Code == code {
			return &findings[idx]
		}
	}
	return nil
}
