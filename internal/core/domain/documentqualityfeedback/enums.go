// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package documentqualityfeedback

type FeedbackType string

const (
	// FeedbackTypeGood indicates that the document was correctly assessed
	FeedbackTypeGood = FeedbackType("Good")

	// FeedbackTypeBad indicates that the document was not correctly assessed
	FeedbackTypeBad = FeedbackType("Bad")

	// FeedbackTypeUnclear indicates that the document was not clear enough to assess
	FeedbackTypeUnclear = FeedbackType("Unclear")
)
