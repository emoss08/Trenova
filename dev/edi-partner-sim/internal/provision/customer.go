package provision

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
)

const (
	customerCodePrefix = "SIM"
	customerCodeLength = 10
	searchLimit        = 25
	tenderSet          = "204"
)

func CustomerCode(name string) string {
	code := customerCodePrefix + strings.ToUpper(name)
	if len(code) > customerCodeLength {
		code = code[:customerCodeLength]
	}
	return code
}

func (o *Options) customer(
	ctx context.Context,
	def *partner.Definition,
	result *Result,
) error {
	if def.Partner.Role == partner.RoleCarrier {
		return nil
	}
	code := CustomerCode(def.Name())
	options, err := o.Trenova.SelectOptions(ctx, trenova.ResourceCustomer, code, searchLimit)
	if err != nil {
		return fmt.Errorf("look for the customer %s: %w", code, err)
	}
	for index := range options {
		if strings.EqualFold(options[index].MetaString("code"), code) {
			result.CustomerID, result.CustomerLabel = options[index].ID, options[index].Display()
			result.note("customer %s bills the partner's tenders", options[index].Label)
			return nil
		}
	}
	load, err := def.Load(tenderSet, 1)
	if err != nil {
		return fmt.Errorf("read the partner's bill-to: %w", err)
	}
	billTo := load.BillTo
	if billTo.Street == "" || billTo.City == "" || billTo.State == "" || billTo.Zip == "" {
		return errors.New("the partner's bill-to has no full address to create its customer from")
	}
	state, err := findState(ctx, o.Trenova, billTo.State)
	if err != nil {
		return err
	}
	created, err := o.Trenova.CreateCustomer(ctx, &trenova.NewCustomer{
		Code:         code,
		Name:         def.Partner.Display + " (EDI simulator)",
		AddressLine1: billTo.Street,
		City:         billTo.City,
		StateID:      state,
		PostalCode:   billTo.Zip,
	})
	if err != nil {
		return fmt.Errorf("create the customer %s: %w", code, err)
	}
	result.CustomerID, result.CustomerLabel = created.ID, created.Display()
	result.note("created customer %s to bill the partner's tenders", created.Label)
	return nil
}

func findState(ctx context.Context, client *trenova.Client, abbreviation string) (string, error) {
	options, err := client.SelectOptions(ctx, trenova.ResourceUSState, abbreviation, searchLimit)
	if err != nil {
		return "", fmt.Errorf("look for the state %s: %w", abbreviation, err)
	}
	for index := range options {
		if strings.EqualFold(options[index].MetaString("abbreviation"), abbreviation) {
			return options[index].ID, nil
		}
	}
	return "", fmt.Errorf("Trenova has no state %s", abbreviation)
}
