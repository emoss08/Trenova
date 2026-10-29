package carrierfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const carrierSeedDaySeconds = int64(86400)

type CarrierSeed struct {
}

// carrierPolicyDef expresses a policy window in days relative to the seed run so
// the eligibility matrix stays true no matter when the database is reseeded.
type carrierPolicyDef struct {
	policyType     carrier.InsurancePolicyType
	policyNumber   string
	providerName   string
	coverageAmount string
	effectiveDays  int64
	expirationDays int64
	isVerified     bool
}

type carrierContactDef struct {
	name                      string
	title                     string
	email                     string
	phone                     string
	isPrimary                 bool
	receivesRateConfirmations bool
}

type carrierDef struct {
	code              string
	name              string
	dbaName           string
	carrierType       carrier.Type
	status            carrier.Status
	complianceStatus  carrier.ComplianceStatus
	safetyRating      carrier.SafetyRating
	dotNumber         string
	mcNumber          string
	scac              string
	addressLine1      string
	city              string
	postalCode        string
	stateAbbr         string
	phone             string
	email             string
	taxID             string
	paymentMethod     carrier.PaymentMethod
	paymentTermDays   int
	remitToName       string
	remitAddressLine1 string
	remitCity         string
	remitPostalCode   string
	remitStateAbbr    string
	notes             string
	policies          []carrierPolicyDef
	contacts          []carrierContactDef
}

func validAutoLiabilityPolicy(number string) carrierPolicyDef {
	return carrierPolicyDef{
		policyType:     carrier.InsurancePolicyTypeAutoLiability,
		policyNumber:   number,
		providerName:   "Great West Casualty",
		coverageAmount: "1000000",
		effectiveDays:  -180,
		expirationDays: 185,
		isVerified:     true,
	}
}

func validCargoLiabilityPolicy(number string) carrierPolicyDef {
	return carrierPolicyDef{
		policyType:     carrier.InsurancePolicyTypeCargoLiability,
		policyNumber:   number,
		providerName:   "Northland Insurance",
		coverageAmount: "250000",
		effectiveDays:  -180,
		expirationDays: 185,
		isVerified:     true,
	}
}

// carrierDefs deliberately spans every branch of
// carrier.EvaluateCarrierEligibility: two blocker classes per required policy
// type, the unqualified and inactive blockers, the 30-day expiry warning
// window, and enough clean carriers for a three-deep waterfall.
func carrierDefs() []carrierDef {
	return []carrierDef{
		{
			code:              "BRT",
			name:              "Blue Ridge Transport",
			dbaName:           "Blue Ridge",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "1204471",
			mcNumber:          "482119",
			scac:              "BRTX",
			addressLine1:      "1840 Industrial Pkwy",
			city:              "Dallas",
			postalCode:        "75207",
			stateAbbr:         "TX",
			phone:             "+12145550118",
			email:             "dispatch@blueridgetransport.example.com",
			taxID:             "75-1840221",
			paymentMethod:     carrier.PaymentMethodACHManual,
			paymentTermDays:   21,
			remitToName:       "Blue Ridge Transport LLC",
			remitAddressLine1: "PO Box 44120",
			remitCity:         "Dallas",
			remitPostalCode:   "75244",
			remitStateAbbr:    "TX",
			notes:             "Primary contract carrier on the Dallas outbound lanes.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-118420"),
				validCargoLiabilityPolicy("NLI-CL-118420"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Marcy Alvarado",
					title:                     "Dispatch Manager",
					email:                     "dispatch@blueridgetransport.example.com",
					phone:                     "+12145550118",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
				{
					name:  "Dean Whitlock",
					title: "Accounts Receivable",
					email: "ar@blueridgetransport.example.com",
					phone: "+12145550119",
				},
			},
		},
		{
			code:              "SPL",
			name:              "Summit Peak Logistics",
			carrierType:       carrier.TypeContract,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "2318904",
			mcNumber:          "771205",
			scac:              "SMPK",
			addressLine1:      "77 Foundry Rd",
			city:              "Chicago",
			postalCode:        "60616",
			stateAbbr:         "IL",
			phone:             "+13125550142",
			email:             "dispatch@summitpeaklogistics.example.com",
			taxID:             "36-2318904",
			paymentMethod:     carrier.PaymentMethodCheck,
			paymentTermDays:   30,
			remitToName:       "Summit Peak Logistics Inc",
			remitAddressLine1: "77 Foundry Rd Ste 200",
			remitCity:         "Chicago",
			remitPostalCode:   "60616",
			remitStateAbbr:    "IL",
			notes:             "Cargo certificate renewal pending; expires inside the warning window.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-231890"),
				{
					policyType:     carrier.InsurancePolicyTypeCargoLiability,
					policyNumber:   "NLI-CL-231890",
					providerName:   "Northland Insurance",
					coverageAmount: "150000",
					effectiveDays:  -350,
					expirationDays: 15,
					isVerified:     true,
				},
			},
			contacts: []carrierContactDef{
				{
					name:                      "Priya Raghunathan",
					title:                     "Director of Dispatch",
					email:                     "dispatch@summitpeaklogistics.example.com",
					phone:                     "+13125550142",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
		{
			code:              "IHF",
			name:              "Ironhorse Freight Lines",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingConditional,
			dotNumber:         "1907733",
			mcNumber:          "690441",
			scac:              "IHFL",
			addressLine1:      "512 Rail Yard Dr",
			city:              "Denver",
			postalCode:        "80216",
			stateAbbr:         "CO",
			phone:             "+13035550177",
			email:             "dispatch@ironhorsefreight.example.com",
			taxID:             "84-1907733",
			paymentMethod:     carrier.PaymentMethodCheck,
			paymentTermDays:   30,
			remitToName:       "Ironhorse Freight Lines LLC",
			remitAddressLine1: "512 Rail Yard Dr",
			remitCity:         "Denver",
			remitPostalCode:   "80216",
			remitStateAbbr:    "CO",
			notes:             "Auto liability lapsed — blocked until a new certificate is filed.",
			policies: []carrierPolicyDef{
				{
					policyType:     carrier.InsurancePolicyTypeAutoLiability,
					policyNumber:   "GWC-AL-190773",
					providerName:   "Great West Casualty",
					coverageAmount: "1000000",
					effectiveDays:  -395,
					expirationDays: -30,
					isVerified:     true,
				},
				validCargoLiabilityPolicy("NLI-CL-190773"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Curtis Beale",
					title:                     "Dispatch Supervisor",
					email:                     "dispatch@ironhorsefreight.example.com",
					phone:                     "+13035550177",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
		{
			code:              "CSC",
			name:              "Cedar Street Carriers",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "3044182",
			mcNumber:          "905117",
			scac:              "CDST",
			addressLine1:      "2900 Cedar St",
			city:              "Phoenix",
			postalCode:        "85009",
			stateAbbr:         "AZ",
			phone:             "+16025550163",
			email:             "dispatch@cedarstreetcarriers.example.com",
			taxID:             "86-3044182",
			paymentMethod:     carrier.PaymentMethodCheck,
			paymentTermDays:   30,
			remitToName:       "Cedar Street Carriers LLC",
			remitAddressLine1: "2900 Cedar St",
			remitCity:         "Phoenix",
			remitPostalCode:   "85009",
			remitStateAbbr:    "AZ",
			notes:             "No cargo liability certificate on file — blocked from coverage.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-304418"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Renata Oyelaran",
					title:                     "Dispatch Lead",
					email:                     "dispatch@cedarstreetcarriers.example.com",
					phone:                     "+16025550163",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
		{
			code:              "HPT",
			name:              "Harbor Point Trucking",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusPending,
			safetyRating:      carrier.SafetyRatingNotRated,
			dotNumber:         "3551029",
			mcNumber:          "1120544",
			scac:              "HRBP",
			addressLine1:      "410 Dockside Ave",
			city:              "Miami",
			postalCode:        "33132",
			stateAbbr:         "FL",
			phone:             "+13055550194",
			email:             "dispatch@harborpointtrucking.example.com",
			taxID:             "65-3551029",
			paymentMethod:     carrier.PaymentMethodCheck,
			paymentTermDays:   30,
			remitToName:       "Harbor Point Trucking Corp",
			remitAddressLine1: "410 Dockside Ave",
			remitCity:         "Miami",
			remitPostalCode:   "33132",
			remitStateAbbr:    "FL",
			notes:             "Onboarding packet under review — compliance not yet qualified.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-355102"),
				validCargoLiabilityPolicy("NLI-CL-355102"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Tomas Escalante",
					title:                     "Operations Manager",
					email:                     "dispatch@harborpointtrucking.example.com",
					phone:                     "+13055550194",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
		{
			code:              "LSH",
			name:              "Lone Star Haulers",
			carrierType:       carrier.TypeContract,
			status:            carrier.StatusInactive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "1668250",
			mcNumber:          "604318",
			scac:              "LNSR",
			addressLine1:      "6100 Lamar Blvd",
			city:              "Dallas",
			postalCode:        "75235",
			stateAbbr:         "TX",
			phone:             "+12145550151",
			email:             "dispatch@lonestarhaulers.example.com",
			taxID:             "75-1668250",
			paymentMethod:     carrier.PaymentMethodCheck,
			paymentTermDays:   30,
			remitToName:       "Lone Star Haulers LP",
			remitAddressLine1: "6100 Lamar Blvd",
			remitCity:         "Dallas",
			remitPostalCode:   "75235",
			remitStateAbbr:    "TX",
			notes:             "Deactivated after the seasonal contract ended.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-166825"),
				validCargoLiabilityPolicy("NLI-CL-166825"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Gwen Marbury",
					title:                     "Dispatch Manager",
					email:                     "dispatch@lonestarhaulers.example.com",
					phone:                     "+12145550151",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
		{
			code:              "GPC",
			name:              "Great Plains Carriers",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "2775013",
			mcNumber:          "833960",
			scac:              "GPLC",
			addressLine1:      "3300 Prairie Loop",
			city:              "Chicago",
			postalCode:        "60632",
			stateAbbr:         "IL",
			phone:             "+13125550126",
			email:             "dispatch@greatplainscarriers.example.com",
			taxID:             "36-2775013",
			paymentMethod:     carrier.PaymentMethodACHManual,
			paymentTermDays:   15,
			remitToName:       "Great Plains Carriers Inc",
			remitAddressLine1: "3300 Prairie Loop",
			remitCity:         "Chicago",
			remitPostalCode:   "60632",
			remitStateAbbr:    "IL",
			notes:             "Backup capacity on the Midwest lanes.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-277501"),
				validCargoLiabilityPolicy("NLI-CL-277501"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Elliot Nakagawa",
					title:                     "Dispatch Manager",
					email:                     "dispatch@greatplainscarriers.example.com",
					phone:                     "+13125550126",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
				{
					name:  "Sondra Bayliss",
					title: "Safety Director",
					email: "safety@greatplainscarriers.example.com",
					phone: "+13125550127",
				},
			},
		},
		{
			code:              "PCT",
			name:              "Pacific Crest Transport",
			carrierType:       carrier.TypeCommon,
			status:            carrier.StatusActive,
			complianceStatus:  carrier.ComplianceStatusQualified,
			safetyRating:      carrier.SafetyRatingSatisfactory,
			dotNumber:         "2049886",
			mcNumber:          "715802",
			scac:              "PCCT",
			addressLine1:      "915 Alameda St",
			city:              "Los Angeles",
			postalCode:        "90021",
			stateAbbr:         "CA",
			phone:             "+12135550109",
			email:             "dispatch@pacificcresttransport.example.com",
			taxID:             "95-2049886",
			paymentMethod:     carrier.PaymentMethodACHManual,
			paymentTermDays:   21,
			remitToName:       "Pacific Crest Transport LLC",
			remitAddressLine1: "915 Alameda St",
			remitCity:         "Los Angeles",
			remitPostalCode:   "90021",
			remitStateAbbr:    "CA",
			notes:             "West coast origin capacity.",
			policies: []carrierPolicyDef{
				validAutoLiabilityPolicy("GWC-AL-204988"),
				validCargoLiabilityPolicy("NLI-CL-204988"),
			},
			contacts: []carrierContactDef{
				{
					name:                      "Hollis Trammell",
					title:                     "Dispatch Manager",
					email:                     "dispatch@pacificcresttransport.example.com",
					phone:                     "+12135550109",
					isPrimary:                 true,
					receivesRateConfirmations: true,
				},
			},
		},
	}
}

// carrierSeedIdentity carries the tenant and lookup keys a seeded carrier row
// needs, keeping the builders free of database access so the eligibility matrix
// can be asserted without a live database.
type carrierSeedIdentity struct {
	orgID        pulid.ID
	buID         pulid.ID
	carrierID    pulid.ID
	stateID      pulid.ID
	remitStateID pulid.ID
	now          int64
}

func buildCarrier(def carrierDef, ident carrierSeedIdentity) *carrier.Carrier {
	stateID := ident.stateID
	remitStateID := ident.remitStateID
	taxIDType := carrier.TaxIDTypeEIN

	entity := &carrier.Carrier{
		ID:                ident.carrierID,
		BusinessUnitID:    ident.buID,
		OrganizationID:    ident.orgID,
		StateID:           &stateID,
		RemitStateID:      &remitStateID,
		Status:            def.status,
		Code:              def.code,
		Name:              def.name,
		DBAName:           def.dbaName,
		CarrierType:       def.carrierType,
		DOTNumber:         def.dotNumber,
		MCNumber:          def.mcNumber,
		SCAC:              def.scac,
		ComplianceStatus:  def.complianceStatus,
		SafetyRating:      def.safetyRating,
		TaxID:             def.taxID,
		TaxIDType:         &taxIDType,
		W9OnFile:          true,
		Is1099Eligible:    def.carrierType == carrier.TypeContract,
		PaymentMethod:     def.paymentMethod,
		PaymentTermDays:   def.paymentTermDays,
		RemitToName:       def.remitToName,
		RemitAddressLine1: def.remitAddressLine1,
		RemitCity:         def.remitCity,
		RemitPostalCode:   def.remitPostalCode,
		AddressLine1:      def.addressLine1,
		City:              def.city,
		PostalCode:        def.postalCode,
		Phone:             def.phone,
		Email:             def.email,
		Notes:             def.notes,
		CreatedAt:         ident.now,
		UpdatedAt:         ident.now,
	}

	if def.complianceStatus == carrier.ComplianceStatusQualified {
		qualifiedAt := ident.now - (90 * carrierSeedDaySeconds)
		entity.QualifiedAt = &qualifiedAt
	}

	return entity
}

func buildCarrierContacts(
	def carrierDef,
	ident carrierSeedIdentity,
) []*carrier.CarrierContact {
	contacts := make([]*carrier.CarrierContact, 0, len(def.contacts))
	for _, contactDef := range def.contacts {
		contacts = append(contacts, &carrier.CarrierContact{
			ID:                        pulid.MustNew("carc_"),
			BusinessUnitID:            ident.buID,
			OrganizationID:            ident.orgID,
			CarrierID:                 ident.carrierID,
			Name:                      contactDef.name,
			Title:                     contactDef.title,
			Email:                     contactDef.email,
			Phone:                     contactDef.phone,
			IsPrimary:                 contactDef.isPrimary,
			ReceivesRateConfirmations: contactDef.receivesRateConfirmations,
			CreatedAt:                 ident.now,
			UpdatedAt:                 ident.now,
		})
	}

	return contacts
}

func buildCarrierPolicies(
	def carrierDef,
	ident carrierSeedIdentity,
) ([]*carrier.CarrierInsurancePolicy, error) {
	policies := make([]*carrier.CarrierInsurancePolicy, 0, len(def.policies))
	for _, policyDef := range def.policies {
		coverage, err := decimal.NewFromString(policyDef.coverageAmount)
		if err != nil {
			return nil, fmt.Errorf(
				"parse coverage amount %s for carrier %s: %w",
				policyDef.coverageAmount,
				def.name,
				err,
			)
		}

		policies = append(policies, &carrier.CarrierInsurancePolicy{
			ID:             pulid.MustNew("carins_"),
			BusinessUnitID: ident.buID,
			OrganizationID: ident.orgID,
			CarrierID:      ident.carrierID,
			PolicyType:     policyDef.policyType,
			PolicyNumber:   policyDef.policyNumber,
			ProviderName:   policyDef.providerName,
			CoverageAmount: coverage,
			EffectiveDate:  ident.now + (policyDef.effectiveDays * carrierSeedDaySeconds),
			ExpirationDate: ident.now + (policyDef.expirationDays * carrierSeedDaySeconds),
			IsVerified:     policyDef.isVerified,
			CreatedAt:      ident.now,
			UpdatedAt:      ident.now,
		})
	}

	return policies, nil
}

func (s *CarrierSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()

	for _, def := range carrierDefs() {
		state, stateErr := tenantfixtures.StateByAbbreviation(ctx, db, def.stateAbbr)
		if stateErr != nil {
			return fmt.Errorf("get state %s: %w", def.stateAbbr, stateErr)
		}

		remitState := state
		if def.remitStateAbbr != def.stateAbbr {
			remitState, stateErr = tenantfixtures.StateByAbbreviation(ctx, db, def.remitStateAbbr)
			if stateErr != nil {
				return fmt.Errorf("get remit state %s: %w", def.remitStateAbbr, stateErr)
			}
		}

		ident := carrierSeedIdentity{
			orgID:        org.ID,
			buID:         org.BusinessUnitID,
			carrierID:    pulid.MustNew("car_"),
			stateID:      state.ID,
			remitStateID: remitState.ID,
			now:          now,
		}

		entity := buildCarrier(def, ident)
		if err = carrierdb.Carriers.Create(ctx, db, entity); err != nil {
			return fmt.Errorf("insert carrier %s: %w", def.name, err)
		}

		if err = s.insertContacts(ctx, db, sr, def, ident); err != nil {
			return err
		}

		if err = s.insertPolicies(ctx, db, sr, def, ident); err != nil {
			return err
		}

	}

	return nil
}

func (s *CarrierSeed) insertContacts(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	def carrierDef,
	ident carrierSeedIdentity,
) error {
	contacts := buildCarrierContacts(def, ident)
	if len(contacts) == 0 {
		return nil
	}

	if err := carrierdb.CarrierContacts.CreateMany(ctx, db, contacts); err != nil {
		return fmt.Errorf("insert contacts for carrier %s: %w", def.name, err)
	}

	return nil
}

func (s *CarrierSeed) insertPolicies(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	def carrierDef,
	ident carrierSeedIdentity,
) error {
	policies, err := buildCarrierPolicies(def, ident)
	if err != nil {
		return err
	}
	if len(policies) == 0 {
		return nil
	}

	if err = carrierdb.CarrierInsurancePolicies.CreateMany(ctx, db, policies); err != nil {
		return fmt.Errorf("insert insurance policies for carrier %s: %w", def.name, err)
	}

	return nil
}
