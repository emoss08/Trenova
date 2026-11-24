package carrier

type RequiredCover struct {
	Policy  *CarrierInsurancePolicy
	Missing []InsurancePolicyType
}

func (c RequiredCover) Whole() bool {
	return c.Policy != nil && len(c.Missing) == 0
}

func SoonestRequiredCover(policies []*CarrierInsurancePolicy) RequiredCover {
	holder := &Carrier{InsurancePolicies: policies}
	var cover RequiredCover
	for _, policyType := range RequiredInsurancePolicyTypes {
		policy := holder.RequiredPolicy(policyType)
		if policy == nil {
			cover.Missing = append(cover.Missing, policyType)
			continue
		}
		if cover.Policy == nil || policy.ExpirationDate < cover.Policy.ExpirationDate {
			cover.Policy = policy
		}
	}

	return cover
}
