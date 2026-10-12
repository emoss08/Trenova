package permission

func (r *Registry) registerReferenceDataResources() {
	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceServiceType.String(),
		DisplayName:        "Service Type",
		Description:        "Service type definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceDelayCode.String(),
		DisplayName:        "Delay Code",
		Description:        "Delay code definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceReasonCode.String(),
		DisplayName:        "Reason Code",
		Description:        "Reason code definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceCommentType.String(),
		DisplayName:        "Comment Type",
		Description:        "Comment type definitions",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityInternal,
	})

	_ = r.Register(&ResourceDefinition{
		Resource:           ResourceTag.String(),
		DisplayName:        "Tag",
		Description:        "Tag management",
		Category:           "Reference Data",
		Operations:         standardOps,
		DefaultSensitivity: SensitivityPublic,
	})
}
