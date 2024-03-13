package database

import (
	"context"
	"log"
	"time"

	"github.com/emoss08/trenova/models"
	"gorm.io/gorm/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Dbinstance struct {
	DB *gorm.DB
}

var DB Dbinstance

type DBConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

const dbContextTimeOut = 10 * time.Second

// ConnectDB Connect to the database.
func ConnectDB(config DBConfig) (*gorm.DB, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbContextTimeOut)
	defer cancel()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  config.DSN,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		NowFunc:     func() time.Time { return time.Now().Local() },
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, cancel, err
	}

	// Migrating Database
	mods := []interface{}{
		&models.BusinessUnit{},
		&models.Organization{},
		&models.EmailProfile{},
		&models.AccountingControl{},
		&models.BillingControl{},
		&models.DispatchControl{},
		&models.InvoiceControl{},
		&models.ShipmentControl{},
		&models.FeasibilityToolControl{},
		&models.RouteControl{},
		&models.Token{},
		&models.User{},
		&models.UserNotifications{},
		&models.UserFavorite{},
		&models.JobTitle{},
		&models.Tag{},
		&models.GeneralLedgerAccount{},
		&models.DivisionCode{},
		&models.RevenueCode{},
		&models.TableChangeAlert{},
		&models.QualifierCode{},
		&models.HazardousMaterial{},
		&models.Commodity{},
	}

	if migrateErr := db.AutoMigrate(mods...); migrateErr != nil {
		return nil, cancel, migrateErr
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, cancel, err
	}

	select {
	case <-ctx.Done():
		return nil, cancel, ctx.Err()
	default:
		// Set connection pool settings
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	log.Println("Connected to the database")

	DB = Dbinstance{DB: db}

	return db, cancel, nil
}
