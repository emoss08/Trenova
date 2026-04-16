package weatheralert

import "testing"

func TestValidateAlertCategory(t *testing.T) {
	t.Parallel()

	valid := AlertCategoryHeat
	if err := ValidateAlertCategory(&valid); err != nil {
		t.Fatalf("expected valid alert category, got %v", err)
	}

	invalid := AlertCategory("coastal_marine_tsunami")
	if err := ValidateAlertCategory(&invalid); err == nil {
		t.Fatal("expected invalid alert category error")
	}
}
