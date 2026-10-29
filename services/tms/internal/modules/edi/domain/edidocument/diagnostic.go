package edidocument

type Diagnostic struct {
	Severity        ValidationSeverity `json:"severity"`
	Code            string             `json:"code"`
	SegmentID       string             `json:"segmentId"`
	ElementPosition int                `json:"elementPosition"`
	Path            string             `json:"path"`
	Message         string             `json:"message"`
	SuggestedFix    string             `json:"suggestedFix"`
}

func HasErrors(diagnostics []Diagnostic) bool {
	for index := range diagnostics {
		if diagnostics[index].Severity == ValidationSeverityError {
			return true
		}
	}
	return false
}
