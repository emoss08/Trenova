package permission

func (r *Registry) registerProtectedFieldResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceProtectedPersonal.String(),
		DisplayName: "Personal details",
		Description: "A worker's date of birth, licence, TWIC and credential numbers, and drug test dates",
		Category:    "Protected fields",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "See personal details"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:    ResourceProtectedMargins.String(),
		DisplayName: "Margins",
		Description: "The margin floor and pay ceiling a rate agreement sets against the sell price",
		Category:    "Protected fields",
		Operations: []OperationDefinition{
			{Operation: OpRead, DisplayName: "View", Description: "See margins"},
		},
		DefaultSensitivity: SensitivityConfidential,
	})
}
