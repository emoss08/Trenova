package models_test

import (
	"backend/models"
	"testing"
)

func TestOrganizationBeforeCreate(t *testing.T) {
	org := models.Organization{
		Name:     "Test Organization",
		ScacCode: "test",
	}

	err := org.BeforeCreate(nil)

	if err != nil {
		t.Errorf("BeforeCreate Returned an error: %v", err)
	}

	// Check if Name is Correctly Title Cased

	expectedName := "Test Organization"

	if org.Name != expectedName {
		t.Errorf("Expected Name to be %v, got %v", expectedName, org.Name)
	}

	expectedScacCode := "TEST"

	if org.ScacCode != expectedScacCode {
		t.Errorf("Expected ScacCode to be %v, got %v", expectedScacCode, org.ScacCode)
	}
}
