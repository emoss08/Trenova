package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// VariablesAndFormatsSeed Creates variables and formats data
type VariablesAndFormatsSeed struct {
	seedhelpers.BaseSeed
}

// NewVariablesAndFormatsSeed creates a new variables_and_formats seed
func NewVariablesAndFormatsSeed() *VariablesAndFormatsSeed {
	seed := &VariablesAndFormatsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"VariablesAndFormats",
		"1.0.0",
		"Creates variables and formats data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies("USStates", "AdminAccount", "Permissions", "HazmatExpiration", "Customers")

	return seed
}

// Run executes the seed
func (s *VariablesAndFormatsSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// Check if formats already exist
			var formatCount int
			err := db.NewSelect().
				Model((*variable.VariableFormat)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &formatCount)
			if err != nil {
				return err
			}

			if formatCount > 0 {
				seedhelpers.LogSuccess("Variable formats already exist, skipping")
				return nil
			}

			defaultOrg, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return fmt.Errorf("get default organization: %w", err)
			}

			defaultBU, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return fmt.Errorf("get default business unit: %w", err)
			}

			formats := []*variable.VariableFormat{
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Currency USD",
					Description:    "Formats numbers as US dollars with commas and 2 decimal places",
					ValueType:      variable.ValueTypeCurrency,
					FormatSQL:      "TO_CHAR(:value::numeric, 'FM$999,999,999.00')",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Date MM/DD/YYYY",
					Description:    "Formats dates as MM/DD/YYYY",
					ValueType:      variable.ValueTypeDate,
					FormatSQL:      "TO_CHAR(:value::timestamp, 'MM/DD/YYYY')",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Date Full",
					Description:    "Formats dates as Month DD, YYYY",
					ValueType:      variable.ValueTypeDate,
					FormatSQL:      "TO_CHAR(:value::timestamp, 'Mon DD, YYYY')",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Uppercase",
					Description:    "Converts text to uppercase",
					ValueType:      variable.ValueTypeString,
					FormatSQL:      "UPPER(:value)",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Proper Case",
					Description:    "Converts text to proper case (first letter of each word capitalized)",
					ValueType:      variable.ValueTypeString,
					FormatSQL:      "INITCAP(:value)",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Phone Format",
					Description:    "Formats 10-digit phone numbers as (XXX) XXX-XXXX",
					ValueType:      variable.ValueTypeString,
					FormatSQL:      "CONCAT('(', SUBSTRING(:value, 1, 3), ') ', SUBSTRING(:value, 4, 3), '-', SUBSTRING(:value, 7, 4))",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Percentage",
					Description:    "Formats decimal as percentage with 2 decimal places",
					ValueType:      variable.ValueTypeNumber,
					FormatSQL:      "CONCAT(ROUND(:value::numeric * 100, 2), '%')",
					IsActive:       true,
					IsSystem:       false,
				},
				{
					ID:             pulid.MustNew("vfm_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Name:           "Boolean Yes/No",
					Description:    "Converts true/false to Yes/No",
					ValueType:      variable.ValueTypeBoolean,
					FormatSQL:      "CASE WHEN :value::boolean THEN 'Yes' ELSE 'No' END",
					IsActive:       true,
					IsSystem:       false,
				},
			}

			if _, err := tx.NewInsert().Model(&formats).Exec(ctx); err != nil {
				return fmt.Errorf("failed to bulk insert variable formats: %w", err)
			}

			// Now create variables
			variables := []*variable.Variable{
				// Customer context variables
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "customerName",
					DisplayName:    "Customer Name",
					Description:    "The customer's business name",
					Category:       "Customer Information",
					Query:          "SELECT name FROM customers WHERE id = :customerId",
					AppliesTo:      variable.ContextCustomer,
					RequiredParams: []string{"customerId"},
					DefaultValue:   "Valued Customer",
					FormatID:       &formats[3].ID, // Uppercase format
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"customer", "basic"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "customerEmail",
					DisplayName:    "Customer Email",
					Description:    "The primary email address for the customer",
					Category:       "Customer Information",
					Query:          "SELECT email FROM customers WHERE id = :customerId",
					AppliesTo:      variable.ContextCustomer,
					RequiredParams: []string{"customerId"},
					DefaultValue:   "customer@example.com",
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"customer", "contact"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "customerAddress",
					DisplayName:    "Customer Address",
					Description:    "The customer's full address",
					Category:       "Customer Information",
					Query:          "SELECT CONCAT(address_line_1, COALESCE(', ' || address_line_2, ''), ', ', city, ', ', us.abbreviation, ' ', postal_code) FROM customers c JOIN us_states us ON c.state_id = us.id WHERE c.id = :customerId",
					AppliesTo:      variable.ContextCustomer,
					RequiredParams: []string{"customerId"},
					DefaultValue:   "",
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"customer", "address"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "customerCreatedDate",
					DisplayName:    "Customer Created Date",
					Description:    "The date the customer was added to the system",
					Category:       "Customer Information",
					Query:          "SELECT TO_CHAR(to_timestamp(created_at), 'MM/DD/YYYY') FROM customers WHERE id = :customerId",
					AppliesTo:      variable.ContextCustomer,
					RequiredParams: []string{"customerId"},
					DefaultValue:   "",
					FormatID:       &formats[1].ID, // Date MM/DD/YYYY format
					ValueType:      variable.ValueTypeDate,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"customer", "date"},
				},

				// Organization context variables
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "organizationName",
					DisplayName:    "Organization Name",
					Description:    "The name of your organization",
					Category:       "Organization",
					Query:          "SELECT name FROM organizations WHERE id = :organizationId",
					AppliesTo:      variable.ContextOrganization,
					RequiredParams: []string{"organizationId"},
					DefaultValue:   "",
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"organization", "basic"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "organizationPhone",
					DisplayName:    "Organization Phone",
					Description:    "The main phone number for the organization",
					Category:       "Organization",
					Query:          "SELECT phone_number FROM organizations WHERE id = :organizationId",
					AppliesTo:      variable.ContextOrganization,
					RequiredParams: []string{"organizationId"},
					DefaultValue:   "",
					FormatID:       &formats[5].ID, // Phone format
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"organization", "contact"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "organizationAddress",
					DisplayName:    "Organization Address",
					Description:    "The organization's full address",
					Category:       "Organization",
					Query:          "SELECT CONCAT(address_line_1, COALESCE(', ' || address_line_2, ''), ', ', city, ', ', us.abbreviation, ' ', postal_code) FROM organizations o JOIN us_states us ON o.state_id = us.id WHERE o.id = :organizationId",
					AppliesTo:      variable.ContextOrganization,
					RequiredParams: []string{"organizationId"},
					DefaultValue:   "",
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       false,
					IsValidated:    true,
					Tags:           []string{"organization", "address"},
				},

				// System context variables
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "currentDate",
					DisplayName:    "Current Date",
					Description:    "Today's date",
					Category:       "System",
					Query:          "SELECT CURRENT_DATE",
					AppliesTo:      variable.ContextSystem,
					RequiredParams: []string{},
					DefaultValue:   "",
					FormatID:       &formats[2].ID, // Date Full format
					ValueType:      variable.ValueTypeDate,
					IsActive:       true,
					IsSystem:       true,
					IsValidated:    true,
					Tags:           []string{"system", "date"},
				},
				{
					ID:             pulid.MustNew("var_"),
					BusinessUnitID: defaultBU.ID,
					OrganizationID: defaultOrg.ID,
					Key:            "currentYear",
					DisplayName:    "Current Year",
					Description:    "The current year",
					Category:       "System",
					Query:          "SELECT EXTRACT(YEAR FROM CURRENT_DATE)::TEXT",
					AppliesTo:      variable.ContextSystem,
					RequiredParams: []string{},
					DefaultValue:   "",
					ValueType:      variable.ValueTypeString,
					IsActive:       true,
					IsSystem:       true,
					IsValidated:    true,
					Tags:           []string{"system", "date"},
				},
			}

			if _, err := tx.NewInsert().Model(&variables).Exec(ctx); err != nil {
				return fmt.Errorf("failed to bulk insert variables: %w", err)
			}

			seedhelpers.LogSuccess(
				"Created variables and formats fixtures",
				fmt.Sprintf("- %d variable formats created", len(formats)),
				fmt.Sprintf("- %d variables created", len(variables)),
				"- Customer context variables: customerName, customerEmail, customerAddress, customerCreatedDate",
				"- Organization context variables: organizationName, organizationPhone, organizationAddress",
				"- System context variables: currentDate, currentYear",
			)

			return nil
		},
	)
}
