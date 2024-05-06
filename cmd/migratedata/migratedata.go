package migratedata

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/commenttype"
	"github.com/emoss08/trenova/internal/ent/equipmentmanufactuer"
	"github.com/emoss08/trenova/internal/ent/equipmenttype"
	"github.com/emoss08/trenova/internal/ent/generalledgeraccount"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/organizationfeatureflag"
	"github.com/emoss08/trenova/internal/ent/revenuecode"
	"github.com/emoss08/trenova/internal/ent/role"
	"github.com/emoss08/trenova/internal/ent/tractor"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/fatih/color"
	"golang.org/x/crypto/bcrypt"
)

// SeedBusinessUnits add the initial business units to the database.
func SeedBusinessUnits(ctx context.Context, client *ent.Client) (*ent.BusinessUnit, error) {
	// Check if the business unit already exists.
	bu, err := client.BusinessUnit.Query().Where(businessunit.NameEQ("Trenova Transportation")).Only(ctx)
	switch {
	// If not, create the business unit
	case ent.IsNotFound(err):
		bu, err = client.BusinessUnit.
			Create().
			SetName("Trenova Transportation").
			SetEntityKey("TREN").
			SetPhoneNumber("123-456-7890").
			SetAddress("1234 Main St").
			Save(ctx)
		if err != nil {
			log.Panicf("Failed creating business unit: %v", err)
		}
	case err != nil:
		log.Panicf("Failed querying business unit: %v", err)
	}

	return bu, nil
}

// SeedOrganization adds the initial organization to the database.
func SeedOrganization(
	ctx context.Context, client *ent.Client, bu *ent.BusinessUnit,
) (*ent.Organization, error) {
	// Check if the organization already exists.
	org, err := client.Organization.Query().Where(organization.And(
		organization.NameEQ("Trenova Transporation"),
		organization.ScacCodeEQ("TREN"),
	)).Only(ctx)
	switch {
	// If not, create the organization
	case ent.IsNotFound(err):
		org, err = client.Organization.
			Create().
			SetName("Trenova Transporation").
			SetScacCode("TREN").
			SetBusinessUnit(bu).
			Save(ctx)
		if err != nil {
			log.Panicf("Failed creating organization: %v", err)
		}
	case err != nil:
		log.Panicf("Failed querying organization: %v", err)
	}

	return org, nil
}

func SeedAccountingControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Accounting control
	acExists, err := org.QueryAccountingControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying invoice control: %v", err)
	}

	// If not, create the Accounting control
	if !acExists {
		err = client.AccountingControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating invoice control: %v", err)
		}
	}

	return err
}

func SeedBillingControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Billing controls
	bcExists, err := org.QueryBillingControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying billing control: %v", err)
	}

	// If not, create the Billing controls
	if !bcExists {
		err = client.BillingControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating billing control: %v", err)
		}
	}

	return err
}

func SeedInvoiceControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Invoice controls
	icExists, err := org.QueryInvoiceControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying invoice control: %v", err)
	}

	// If not, create the Invoice controls
	if !icExists {
		err = client.InvoiceControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating invoice control: %v", err)
		}
	}

	return err
}

func SeedDispatchControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Dispatch controls
	dcExists, err := org.QueryDispatchControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying dispatch control: %v", err)
	}

	// If not, create the Dispatch controls
	if !dcExists {
		err = client.DispatchControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating dispatch control: %v", err)
		}
	}

	return err
}

func SeedShipmentControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Shipment Controls
	scExists, err := org.QueryShipmentControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying shipment control: %v", err)
	}

	// If not, create the Shipment Controls
	if !scExists {
		err = client.ShipmentControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating shipment control: %v", err)
		}
	}

	return err
}

func SeedRouteControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has Route Controls
	rcExists, err := org.QueryRouteControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying route control: %v", err)
	}

	// If not, create the Route Controls
	if !rcExists {
		err = client.RouteControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating route control: %v", err)
		}
	}

	return err
}

func SeedEmailControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	ecExists, err := org.QueryEmailControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying email control: %v", err)
	}

	if !ecExists {
		err = client.EmailControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating email control: %v", err)
		}
	}
	return err
}

func SeedFeasibilityToolControl(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the Organization already has Feasibility tool controls
	ftExists, err := org.QueryFeasibilityToolControl().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying feasibility tool control: %v", err)
	}

	// If not, create the Feasibility tool controls
	if !ftExists {
		err = client.FeasibilityToolControl.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating feasibility tool control: %v", err)
		}
	}
	return err
}

func SeedGoogleAPI(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// CHeck if the Organization already has Google API
	gaExists, err := org.QueryGoogleAPI().Exist(ctx)
	if err != nil {
		log.Panicf("Failed querying google api: %v", err)
	}

	// If not, create the Google API
	if !gaExists {
		err = client.GoogleApi.Create().
			SetOrganization(org).
			SetBusinessUnit(bu).
			SetAPIKey("API_KEY").
			SetMileageUnit("Imperial").
			SetTrafficModel("BestGuess").
			SetAddCustomerLocation(false).
			SetAutoGeocode(false).
			Exec(ctx)
		if err != nil {
			log.Panicf("Failed creating google api: %v", err)
		}
	}

	return err
}

func SeedRevenueCodes(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization has no revenue codes
	rcCount, err := client.RevenueCode.Query().Where(revenuecode.HasOrganizationWith(organization.ID(org.ID))).Count(ctx)

	// If not, create the revenue codes
	if rcCount == 0 {
		// Create 100 revenue codes
		log.Println("Creating revenue codes...")
		for i := 0; i < 100; i++ {
			_, err = client.RevenueCode.Create().
				SetOrganization(org).
				SetBusinessUnit(bu).
				SetCode("RC" + strconv.Itoa(i)).
				SetDescription("Revenue Code " + strconv.Itoa(i)).
				Save(ctx)
			if err != nil {
				log.Panicf("Failed creating revenue code: %v", err)
			}
		}
	}

	return err
}

func SeedGeneralLedgerAccounts(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization has no general ledger accounts
	glCount, err := client.GeneralLedgerAccount.Query().Where(generalledgeraccount.HasOrganizationWith(organization.ID(org.ID))).Count(ctx)

	// If not, create the general ledger accounts
	if glCount == 0 {
		// Create 100 general ledger accounts
		log.Println("Creating general ledger accounts...")

		// Account number must be in the format 1000-00, 1100-00, ..., 1000-01, etc.
		for i := 0; i < 2; i++ {
			// Increment the first part by 100 each iteration, starting from 1000
			firstPart := 1000 + (i/10)*100

			// Use the last digit of i for the second part to avoid duplication
			secondPart := i % 10

			accountNumber := fmt.Sprintf("%04d-%02d", firstPart, secondPart)

			_, err = client.GeneralLedgerAccount.Create().
				SetOrganization(org).
				SetBusinessUnit(bu).
				SetAccountNumber(accountNumber).
				SetAccountType(generalledgeraccount.AccountTypeAsset).
				Save(ctx)
			if err != nil {
				log.Panicf("Failed creating general ledger account: %v", err)
			}
		}
	}

	return err
}

func SeedAdminAccount(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the admin account exists
	_, err := client.User.Query().Where(user.UsernameEQ("admin")).Only(ctx)
	switch {
	// If not, create the admin account
	case ent.IsNotFound(err):
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		adminRole, err := client.Role.Query().Where(role.NameEQ("Admin")).Only(ctx)
		if err != nil {
			log.Panicf("Failed querying admin role: %v", err)
		}
		_, err = client.User.
			Create().
			SetUsername("admin").
			SetPassword(string(hashedPassword)).
			SetEmail("admin@trenova.app").
			SetName("System Administrator").
			SetOrganization(org).
			SetBusinessUnit(bu).
			SetIsAdmin(true).
			SetIsSuperAdmin(true).
			AddRoles(adminRole).
			Save(ctx)

		// Print out the admin account credentials
		color.Yellow("✅ Admin account created successfully")
		color.Yellow("-----------------------------")
		color.Yellow("Admin account credentials:")
		color.Yellow("Email: admin@trenova.app")
		color.Yellow("Password: admin")
		color.Yellow("-----------------------------")

		if err != nil {
			log.Panicf("Failed creating admin account: %v", err)
		}
	case err != nil:
		log.Panicf("Failed querying admin account: %v", err)
	}

	return err
}

func SeedNormalAccount(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the normal account exists
	_, err := client.User.Query().Where(user.UsernameEQ("normie")).Only(ctx)
	switch {
	// If not, create the normal account
	case ent.IsNotFound(err):
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("user"), bcrypt.DefaultCost)
		_, err = client.User.
			Create().
			SetUsername("normie").
			SetPassword(string(hashedPassword)).
			SetEmail("regular@trenova.app").
			SetName("Regular User").
			SetOrganization(org).
			SetBusinessUnit(bu).
			SetIsAdmin(false).
			SetIsSuperAdmin(false).
			Save(ctx)

		// Print out the normal account credentials
		color.Yellow("✅ Normal account created successfully")
		color.Yellow("-----------------------------")
		color.Yellow("Normal account credentials:")
		color.Yellow("Email: regular@trenova.app")
		color.Yellow("Password: user")
		color.Yellow("-----------------------------")

		if err != nil {
			log.Panicf("Failed creating normal account: %v", err)
		}

	case err != nil:
		log.Panicf("Failed querying normal account: %v", err)
	}

	return err
}

func SeedEquipmentTypes(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has equipment types
	etCount, err := client.EquipmentType.Query().
		Where(
			equipmenttype.HasOrganizationWith(organization.ID(org.ID)),
		).Count(ctx)

	// If not, create the equipment types
	if etCount == 0 {
		err = client.EquipmentType.CreateBulk(
			client.EquipmentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetCode("TRAILER").
				SetEquipmentClass("Trailer"),
			client.EquipmentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetCode("TRACTOR").
				SetEquipmentClass("Tractor"),
		).Exec(ctx)
	}
	return err
}

func SeedEquipmentManufacturers(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has equipment manufacturers
	etCount, err := client.EquipmentManufactuer.Query().
		Where(
			equipmentmanufactuer.HasOrganizationWith(organization.ID(org.ID)),
		).Count(ctx)

	// If not, create the equipment manufacturers
	if etCount == 0 {
		log.Println("Adding standard equipment manufacturers...")

		err = client.EquipmentManufactuer.CreateBulk(
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Volvo"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Peterbilt"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Kenworth"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Freightliner"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("International"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Mack"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Western Star"),
			client.EquipmentManufactuer.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Paccar"),
		).Exec(ctx)
	}

	return err
}

func SeedTractors(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has tractors
	tractorCount, err := client.Tractor.Query().
		Where(
			tractor.HasOrganizationWith(organization.ID(org.ID)),
		).Count(ctx)

	// If not, create the tractors
	if tractorCount == 0 {
		log.Println("Adding standard tractors...")

		// Create 10 tractors
		for i := 0; i < 10; i++ {
			_, err = client.Tractor.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("Available").
				SetVin("1HGBH41JXMN109186").
				SetLicensePlateNumber("1HGBH41").
				SetYear(2021).
				SetModel("579").
				Save(ctx)
			if err != nil {
				log.Panicf("Failed creating tractor: %v", err)
			}
		}
	}

	return err
}

func SeedCommentTypes(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the organization already has comment types
	etCount, err := client.CommentType.Query().
		Where(
			commenttype.HasOrganizationWith(organization.ID(org.ID)),
		).Count(ctx)

	// If not, create the comment types
	if etCount == 0 {
		log.Println("Adding standard comment types...")

		err = client.CommentType.CreateBulk(
			client.CommentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Dispatch").
				SetSeverity("Low").
				SetDescription("Dispatch comment (Will transmit to the driver)"),
			client.CommentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Billing").
				SetSeverity("Low").
				SetDescription("Billing comment (Will transmit to the billing department)"),
			client.CommentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Safety").
				SetSeverity("Low").
				SetDescription("Safety comment (Will transmit to the safety department)"),
			client.CommentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Maintenance").
				SetSeverity("Low").
				SetDescription("Maintenance comment (Will transmit to the maintenance department)"),
			client.CommentType.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetStatus("A").
				SetName("Review").
				SetSeverity("High").
				SetDescription("Comment when a review is needed"),
		).Exec(ctx)
	}
	if err != nil {
		log.Panicf("Failed creating comment types: %v", err)
		return err
	}

	return nil
}

func SeedResources(
	ctx context.Context, client *ent.Client,
) error {
	// Check if the resources already exists
	etCount, err := client.Resource.Query().Count(ctx)

	// If not, create the resources
	if etCount == 0 {
		log.Println("Adding resources...")

		err = client.Resource.CreateBulk(
			client.Resource.Create().
				SetType("AccessorialCharge").
				SetDescription("Represents accessorial charges in the system."),
			client.Resource.Create().
				SetType("AccountingControl").
				SetDescription("Represents accounting controls in the system."),
			client.Resource.Create().
				SetType("BillingControl").
				SetDescription("Represents billing controls in the system."),
			client.Resource.Create().
				SetType("BusinessUnit").
				SetDescription("Represents business units in the system."),
			client.Resource.Create().
				SetType("ChargeType").
				SetDescription("Represents charge types in the system."),
			client.Resource.Create().
				SetType("CommentType").
				SetDescription("Represents comment types in the system."),
			client.Resource.Create().
				SetType("Commodity").
				SetDescription("Represents commodities in the system."),
			client.Resource.Create().
				SetType("Customer").
				SetDescription("Represents customers in the system."),
			client.Resource.Create().
				SetType("CustomReport").
				SetDescription("Represents custom reports in the system."),
			client.Resource.Create().
				SetType("DelayCode").
				SetDescription("Represents delay codes in the system."),
			client.Resource.Create().
				SetType("DispatchControl").
				SetDescription("Represents dispatch controls in the system."),
			client.Resource.Create().
				SetType("DivisionCode").
				SetDescription("Represents division codes in the system."),
			client.Resource.Create().
				SetType("DocumentClassification").
				SetDescription("Represents document classifications in the system."),
			client.Resource.Create().
				SetType("EmailControl").
				SetDescription("Represents email controls in the system."),
			client.Resource.Create().
				SetType("EmailProfile").
				SetDescription("Represents email profiles in the system."),
			client.Resource.Create().
				SetType("EquipmentManufacturer").
				SetDescription("Represents equipment manufacturers in the system."),
			client.Resource.Create().
				SetType("EquipmentType").
				SetDescription("Represents equipment types in the system."),
			client.Resource.Create().
				SetType("FeasibilityToolControl").
				SetDescription("Represents feasibility tool controls in the system."),
			client.Resource.Create().
				SetType("FeatureFlag").
				SetDescription("Represents feature flags in the system."),
			client.Resource.Create().
				SetType("FleetCode").
				SetDescription("Represents fleet codes in the system."),
			client.Resource.Create().
				SetType("FormulaTemplate").
				SetDescription("Represents formula templates in the system."),
			client.Resource.Create().
				SetType("GeneralLedgerAccount").
				SetDescription("Represents general ledger accounts in the system."),
			client.Resource.Create().
				SetType("GoogleApi").
				SetDescription("Represents google apis in the system."),
			client.Resource.Create().
				SetType("HazardousMaterial").
				SetDescription("Represents hazardous materials in the system."),
			client.Resource.Create().
				SetType("HazardousMaterialSegregation").
				SetDescription("Represents hazardous material segregations in the system."),
			client.Resource.Create().
				SetType("InvoiceControl").
				SetDescription("Represents invoice controls in the system."),
			client.Resource.Create().
				SetType("Location").
				SetDescription("Represents locations in the system."),
			client.Resource.Create().
				SetType("LocationCategory").
				SetDescription("Represents location categories in the system."),
			client.Resource.Create().
				SetType("LocationComment").
				SetDescription("Represents location comments in the system."),
			client.Resource.Create().
				SetType("LocationContacts").
				SetDescription("Represents location contacts in the system."),
			client.Resource.Create().
				SetType("Organization").
				SetDescription("Represents organizations in the system."),
			client.Resource.Create().
				SetType("OrganizationFeatureFlag").
				SetDescription("Represents organization feature flags in the system."),
			client.Resource.Create().
				SetType("Permission").
				SetDescription("Represents permissions in the system."),
			client.Resource.Create().
				SetType("QualifierCode").
				SetDescription("Represents qualifier codes in the system."),
			client.Resource.Create().
				SetType("ReasonCode").
				SetDescription("Represents reason codes in the system."),
			client.Resource.Create().
				SetType("RevenueCode").
				SetDescription("Represents revenue codes in the system."),
			client.Resource.Create().
				SetType("Role").
				SetDescription("Represents roles in the system."),
			client.Resource.Create().
				SetType("RouteControl").
				SetDescription("Represents route controls in the system."),
			client.Resource.Create().
				SetType("ServiceType").
				SetDescription("Represents service types in the system."),
			client.Resource.Create().
				SetType("Shipment").
				SetDescription("Represents shipments in the system."),
			client.Resource.Create().
				SetType("ShipmentCharge").
				SetDescription("Represents shipment charges in the system."),
			client.Resource.Create().
				SetType("ShipmentComment").
				SetDescription("Represents shipment comment in the system."),
			client.Resource.Create().
				SetType("ShipmentCommodity").
				SetDescription("Represents shipment commodities in the system."),
			client.Resource.Create().
				SetType("ShipmentControl").
				SetDescription("Represents shipment controls in the system."),
			client.Resource.Create().
				SetType("ShipmentDocumentation").
				SetDescription("Represents shipment documentations in the system."),
			client.Resource.Create().
				SetType("ShipmentMove").
				SetDescription("Represents shipment moves in the system."),
			client.Resource.Create().
				SetType("ShipmentRoute").
				SetDescription("Represents shipment routes in the system."),
			client.Resource.Create().
				SetType("ShipmentType").
				SetDescription("Represents shipment types in the system."),
			client.Resource.Create().
				SetType("Stop").
				SetDescription("Represents stops in the system."),
			client.Resource.Create().
				SetType("TableChangeAlert").
				SetDescription("Represents table change alerts in the system."),
			client.Resource.Create().
				SetType("Tag").
				SetDescription("Represents tags in the system."),
			client.Resource.Create().
				SetType("Tractor").
				SetDescription("Represents tractors in the system."),
			client.Resource.Create().
				SetType("Trailer").
				SetDescription("Represents trailers in the system."),
			client.Resource.Create().
				SetType("User").
				SetDescription("Represents users in the system."),
			client.Resource.Create().
				SetType("UserFavorite").
				SetDescription("Represents user favorites in the system."),
			client.Resource.Create().
				SetType("UserNotification").
				SetDescription("Represents user notifications in the system."),
			client.Resource.Create().
				SetType("UserReport").
				SetDescription("Represents user reports in the system."),
			client.Resource.Create().
				SetType("UsState").
				SetDescription("Represents us states in the system."),
			client.Resource.Create().
				SetType("Worker").
				SetDescription("Represents workers in the system."),
			client.Resource.Create().
				SetType("WorkerComment").
				SetDescription("Represents worker comments in the system."),
			client.Resource.Create().
				SetType("WorkerContact").
				SetDescription("Represents worker contacts in the system."),
			client.Resource.Create().
				SetType("WorkerProfile").
				SetDescription("Represents worker profile in the system."),
		).Exec(ctx)
	}

	if err != nil {
		log.Panicf("Failed creating resources: %v", err)
		return err
	}

	return nil
}

func SeedPermissions(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the permissions already exist
	etCount, err := client.Permission.Query().Count(ctx)
	if err != nil {
		log.Panic("Failed checking existing permissions")
		return err
	}

	if etCount == 0 {
		log.Println("Adding base permissions...")

		resources, err := client.Resource.Query().All(ctx)
		if err != nil {
			log.Panic("Failed querying resources")
			return err
		}

		// Detailed permissions for each action
		actions := []struct {
			action           string
			readDescription  string
			writeDescription string
		}{
			{"view", "Can view all", "Can view all"},
			{"add", "Can view all", "Can add, edit, and delete"},
			{"edit", "Can view all", "Can add, edit, and delete"},
			{"delete", "Can view all", "Can add, edit, and delete"},
		}

		for _, resource := range resources {
			resourceTypeLower := strings.ToLower(resource.Type)
			for _, action := range actions {
				// Format codename, label, and descriptions
				codename := fmt.Sprintf("%s.%s", resourceTypeLower, action.action)
				label := fmt.Sprintf("%s %s", strings.Title(action.action), resource.Type)
				readDescription := fmt.Sprintf("%s %s.", action.readDescription, resource.Type)
				writeDescription := fmt.Sprintf("%s %s.", action.writeDescription, resource.Type)

				// Create the permission
				_, err = client.Permission.Create().
					SetBusinessUnit(bu).
					SetOrganization(org).
					SetCodename(codename).
					SetResource(resource).
					SetAction(action.action).
					SetLabel(label).
					SetReadDescription(readDescription).
					SetWriteDescription(writeDescription).
					Save(ctx)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SeedRoles(
	ctx context.Context, client *ent.Client, org *ent.Organization, bu *ent.BusinessUnit,
) error {
	// Check if the roles already exists
	etCount, err := client.Role.Query().Count(ctx)

	// If not, create the roles
	if etCount == 0 {
		log.Println("Adding base roles...")

		roles := []string{"Admin", "Dispatcher", "Driver", "Billing", "Safety", "Maintenance"}

		for _, roleName := range roles {
			_, err = client.Role.Create().
				SetBusinessUnit(bu).
				SetOrganization(org).
				SetName(roleName).
				SetDescription(fmt.Sprintf("Base role for %s", roleName)).
				Save(ctx)
			if err != nil {
				log.Panicf("Failed creating role: %v", err)
				return err
			}
		}
	}
	if err != nil {
		log.Panicf("Failed creating roles: %v", err)
		return err
	}

	// Add all permissions to the admin role
	adminRole, err := client.Role.Query().Where(role.NameEQ("Admin")).Only(ctx)
	if err != nil {
		log.Panicf("Failed querying admin role: %v", err)
		return err
	}

	permissions, err := client.Permission.Query().All(ctx)
	if err != nil {
		log.Panicf("Failed querying permissions: %v", err)
		return err
	}

	_, err = adminRole.Update().
		AddPermissions(permissions...).
		Save(ctx)
	if err != nil {
		log.Panicf("Failed adding permissions to admin role: %v", err)
		return err
	}

	return nil
}

func SeedFeatureFlags(
	ctx context.Context, client *ent.Client, org *ent.Organization,
) error {
	// Check if the feature flags already exist in the system.
	etCount, err := client.FeatureFlag.Query().Count(ctx)

	// If not, create the default feature flags
	if etCount == 0 {
		log.Println("Adding feature flags...")

		err = client.FeatureFlag.CreateBulk(
			client.FeatureFlag.Create().
				SetName("Color Accessibility Options").
				SetCode("ENABLE_COLOR_BLIND_MODE").
				SetBeta(true).
				SetDescription("This flag enables Color Blind Mode, offering users a choice of color vision deficiency simulations to adapt the application's color scheme for better readability and visual comfort. Modes include Tritanopia, Protanopia, Deuteranopia, Deuteranomaly, and Protanomaly. This inclusivity-focused feature is designed to cater to users with various color vision impairments, ensuring a more accessible and user-friendly experience."),
			client.FeatureFlag.Create().
				SetName("Shipment Map View").
				SetCode("ENABLE_SHIP_MAP_VIEW").
				SetBeta(true).
				SetDescription("Activating this flag introduces a novel shipment map view in the shipment management interface. It provides a visual representation of workers and orders, along with interactive functionalities like drag-and-drop assignment of workers to orders. This feature enhances the user's operational efficiency by offering a more intuitive and interactive way to manage shipments."),
			client.FeatureFlag.Create().
				SetName("Beam").
				SetCode("ENABLE_BEAM").
				SetBeta(true).
				SetDescription("Activating this flag will enable the Beam feature, which allows users to utilize Trenova's very own LLM (Large Language Model). "),
			client.FeatureFlag.Create().
				SetName("Billing Client").
				SetCode("ENABLE_BILLING_CLIENT").
				SetBeta(true).
				SetDescription("This feature flag enables the Billing Client, which allows users to manage their billing information and payment methods."),
			client.FeatureFlag.Create().
				SetName("Pricing Tool").
				SetCode("ENABLE_PRICING_TOOL").
				SetBeta(true).
				SetDescription("This feature flag enables the Pricing Tool, which allows users to manage their pricing information."),
			client.FeatureFlag.Create().
				SetName("Worker Feasibility Tool").
				SetCode("ENABLE_WORKER_FEAS_TOOL").
				SetBeta(true).
				SetDescription("This feature flag enables the Worker Feasibility Tool, which allows users to manage their worker feasibility information."),
			client.FeatureFlag.Create().
				SetName("Document Studio").
				SetCode("ENABLE_DOC_STUDIO").
				SetBeta(true).
				SetDescription("This feature flag enables the Document Studio, which allows users to manage their document templates. "),
		).Exec(ctx)
	}
	if err != nil {
		log.Panicf("Failed creating feature flags: %v", err)
		return err
	}

	// for each feature flag create an organization feature flag
	featureFlags, err := client.FeatureFlag.Query().All(ctx)
	if err != nil {
		log.Panicf("Failed querying feature flags: %v", err)
		return err
	}

	// Check if the organization already has feature flags
	ofCount, err := client.OrganizationFeatureFlag.Query().Where(organizationfeatureflag.HasOrganizationWith(organization.ID(org.ID))).Count(ctx)
	if err != nil {
		log.Panicf("Failed querying organization feature flags: %v", err)
		return err
	}

	if ofCount > 0 {
		log.Println("Organization already has feature flags")
		return nil
	}

	for _, featureFlag := range featureFlags {
		_, err = client.OrganizationFeatureFlag.Create().
			SetOrganization(org).
			SetFeatureFlag(featureFlag).
			SetIsEnabled(true).
			Save(ctx)
		if err != nil {
			log.Panicf("Failed creating organization feature flag: %v", err)
			return err
		}
	}

	return nil
}
