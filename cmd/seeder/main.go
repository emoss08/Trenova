package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/businessunit"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/revenuecode"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/ent/user"
	"github.com/emoss08/trenova/ent/worker"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func generateRandomString(length int) string {
	// The characters that can be used in the random string
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// The resulting random string
	result := make([]byte, length)

	// Generate a random string by randomly selecting characters from the chars slice
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Initialize the database
	client := database.NewEntClient(os.Getenv("DB_DSN"))

	defer client.Close()

	if os.Getenv("ENV") == "production" {
		log.Panic("Cannot run seeder in production environment")
	}

	ctx := context.Background()

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

	// Check if the organization has no general ledger accounts
	glCount, err := client.GeneralLedgerAccount.Query().
		Where(generalledgeraccount.HasOrganizationWith(organization.ID(org.ID))).Count(ctx)

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

	// Check if the organization has no workers.
	workerCount, err := client.Worker.Query().
		Where(worker.HasOrganizationWith(organization.ID(org.ID))).Count(ctx)

	if workerCount == 0 {
		// Create 100 workers with WorkerProfiles
		log.Println("Creating workers...")

		// Generate a random worker code that is 10 characters long and unique.
		// The worker code must be unique within the organization.
		workerCodes := make(map[string]bool)

		for i := 0; i < 100; i++ {
			unique := false

			var code string
			for !unique {
				code = "W" + generateRandomString(9)
				if !workerCodes[code] {
					unique = true
					workerCodes[code] = true
				}
			}

			if err != nil {
				log.Panicf("Failed starting transaction: %v", err)
			}

			worker, workerErr := client.Worker.Create().
				SetOrganization(org).
				SetBusinessUnit(bu).
				SetCode(code).
				SetFirstName("Worker").
				SetLastName(strconv.Itoa(i)).
				Save(ctx)
			if workerErr != nil {
				log.Panicf("Failed creating worker: %v", workerErr)
			}

			_, err = client.WorkerProfile.Create().
				SetOrganization(org).
				SetBusinessUnit(bu).
				SetWorker(worker).
				SetWorkerID(worker.ID).
				SetLicenseNumber("123456789").
				Save(ctx)
			if err != nil {
				log.Panicf("Failed creating worker profile: %v", err)
			}
		}
	}

	// Check if the admin account exists
	_, adminErr := client.User.Query().Where(user.UsernameEQ("admin")).Only(ctx)
	switch {
	// If not, create the admin account
	case ent.IsNotFound(adminErr):
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
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
			Save(ctx)

		// Print out the admin account credentials
		color.Yellow("✅ Admin account created successfully")
		color.Yellow("-----------------------------")
		color.Yellow("Admin account credentials:")
		color.Yellow("Username: admin")
		color.Yellow("Password: admin")
		color.Yellow("-----------------------------")

		if err != nil {
			log.Panicf("Failed creating admin account: %v", err)
		}
	case err != nil:
		log.Panicf("Failed querying admin account: %v", err)
	}

	// Print success message
	color.Green("✅ Seeder ran successfully")
}
