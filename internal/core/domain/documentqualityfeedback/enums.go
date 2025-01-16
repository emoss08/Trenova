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
