package workercredential

// NextExpiry is the earliest expiry among the worker's required credentials —
// what the roster sorts by and the expiry forecast counts down to. Optional
// credentials are excluded: nobody should be chased over a certificate the
// role does not need.
func (s *WorkerCredentialSummary) NextExpiry() *int64 {
	if s == nil {
		return nil
	}
	var earliest *int64
	for _, item := range s.Items {
		if item == nil || !item.Required || item.Credential == nil {
			continue
		}
		expiry := item.Credential.ExpiresAt
		if expiry == nil || *expiry <= 0 {
			continue
		}
		if earliest == nil || *expiry < *earliest {
			earliest = expiry
		}
	}
	if earliest == nil {
		return nil
	}
	value := *earliest
	return &value
}
