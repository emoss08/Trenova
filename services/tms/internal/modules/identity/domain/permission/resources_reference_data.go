package permission

func (r *Registry) registerReferenceDataResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceServiceType.String(),
		DisplayName:        "Service Type",
		Description:        "Service type definitions",
		Category:           "Shipments",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

}
