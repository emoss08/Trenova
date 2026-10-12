package detector

import "errors"

// ErrMalformedFinding reports a detector returning something that cannot be
// shown to a person. It is a programming error rather than a data condition, so
// the run drops the finding and logs loudly instead of surfacing it.
var ErrMalformedFinding = errors.New("detector produced a malformed finding")
