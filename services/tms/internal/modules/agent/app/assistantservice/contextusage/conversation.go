package contextusage

const (
	// historyLimit bounds how much of a long conversation is replayed. The whole
	// thread would eventually exceed any context window, and the most recent
	// turns are the ones that carry the thread of the question. Tool results
	// count toward it, and the runtime cuts the older ones down before the
	// replay, so the limit is set by how many turns should stay in view
	// rather than by how much data those turns fetched.
	historyLimit = 120
)
