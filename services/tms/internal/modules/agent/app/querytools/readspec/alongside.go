package readspec

// Alongside is a read an unpaged list makes beside its own, for records
// of another kind its next call needs (the templates a checklist can be
// started from, a worker's recognitions), carried in the result under key.
// It is made only when the reader may read that kind, as the repository tool
// read it only then, and a read keyed by the call's own arguments only when
// the call names them.
type Alongside struct {
	Key       string
	Operation string
	// what names the records, for the note a cut-short read leaves.
	What string
	// inputs are the table input's fields every connection read sends
	// (active templates, started by hand), or the variables a keyed read
	// sends beside its keys (the open swaps only, at most twenty).
	Inputs map[string]any
	// scoped reads it only when the call names one of the list's references
	// (the credential types beside one worker's credentials, not beside what
	// falls due across the organization).
	Scoped bool
	// unless are the list's parameters that, named, leave it unread: a call
	// that names a credential type already knows the types, and a dozen
	// workers checked for one each came back with the whole catalog again.
	Unless []string
	// withheld is the note a result carries when the reader may not read
	// the kind, so the model knows the records exist and are hidden.
	Withheld string
	// shape applies to each record what its resolver does not (what a sync
	// record's status means).
	Shape func(row map[string]any)
	// note is what a read made instead of a list's always says beside its
	// records (a document with no state was never sent).
	Note string
}
