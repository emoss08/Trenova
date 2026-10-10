package agenttoolservice

func workforceToolProviders() []any {
	providers := make([]any, 0, 64)
	for _, group := range [][]any{
		safetyToolProviders(),
		drugAlcoholToolProviders(),
		leaveToolProviders(),
		trainingToolProviders(),
		checklistToolProviders(),
		reviewToolProviders(),
		dqfToolProviders(),
		credentialToolProviders(),
		injuryToolProviders(),
		ptoToolProviders(),
		permitToolProviders(),
		driverPayReviewToolProviders(),
		payrollToolProviders(),
	} {
		providers = append(providers, group...)
	}

	return providers
}
