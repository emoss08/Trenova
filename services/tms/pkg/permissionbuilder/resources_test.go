package permissionbuilder

import "testing"

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"APIToken", "api_token"},
		{"WorkerPTO", "worker_pto"},
		{"AILog", "ai_log"},
		{"User", "user"},
		{"BusinessUnit", "business_unit"},
		{"CustomerEmailProfile", "customer_email_profile"},
		{"OrganizationMembership", "organization_membership"},
		{"UsState", "us_state"},
		{"BillingProfileDocumentType", "billing_profile_document_type"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}