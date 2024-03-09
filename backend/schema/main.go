package main

import (
	"fmt"
	"io"
	"os"
	"trenova-go-backend/app/models"

	"ariga.io/atlas-provider-gorm/gormschema"
)

func main() {
	var models = []any{
		&models.BusinessUnit{},
		&models.Organization{},
		&models.EmailProfile{},
		&models.TableChangeAlert{},
		&models.GeneralLedgerAccount{},
		&models.Tag{},
		&models.RevenueCode{},
		&models.DivisionCode{},
		&models.AccountingControl{},
		&models.JobTitle{},
		&models.User{},
		&models.UserFavorite{},
	}

	stmts, err := gormschema.New("postgres").Load(models...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	io.WriteString(os.Stdout, stmts)
}
