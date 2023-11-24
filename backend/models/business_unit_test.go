package models_test

import (
	"backend/models"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestBeforeCreate(t *testing.T) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatalf("Error creating mock database: %s", err)
	}

	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})

	if err != nil {
		t.Fatalf("Error opening gorm database: %s", err)
	}

	bu := &models.BusinessUnit{
		Name: "Test Business Unit",
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "business_units" WHERE entity_key = \$1`).
		WithArgs("TESTBUSI01").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err = bu.BeforeCreate(gormDB)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, bu.ID)
	assert.Equal(t, "TESTBUSI01", bu.EntityKey)
}

func TestPaid(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour) // set to 24 hours in the future

	testCases := []struct {
		name   string
		bu     models.BusinessUnit
		result bool
	}{
		{
			name:   "PaidUntil is nil",
			bu:     models.BusinessUnit{},
			result: false,
		},
		{
			name:   "PaidUntil in the past",
			bu:     models.BusinessUnit{PaidUntil: &time.Time{}},
			result: false,
		},
		{
			name:   "PaidUntil in the future",
			bu:     models.BusinessUnit{PaidUntil: &futureTime},
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.result, tc.bu.Paid())
		})
	}
}
