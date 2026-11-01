package invoiceshareresolver

const (
	defaultCandidateLimit = 10
	maxCandidateLimit     = 25
)

func candidateLimit(limit *int) int {
	if limit == nil || *limit <= 0 {
		return defaultCandidateLimit
	}

	return min(*limit, maxCandidateLimit)
}
