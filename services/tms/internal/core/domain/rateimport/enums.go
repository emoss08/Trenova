package rateimport

// Status is where an import is in its life.
//
// The states exist because a rate sheet is reviewed before it is applied. An
// import that went straight from upload to committed would be the thing bulk
// ingest is most often blamed for: a tariff nobody read replacing one somebody
// negotiated.
type Status string

const (
	// StatusPending is uploaded but not yet read.
	StatusPending = Status("Pending")
	// StatusParsed means the rows were read and the dry run is ready to review.
	StatusParsed = Status("Parsed")
	// StatusCommitted means the sheet was applied to the agreement.
	StatusCommitted = Status("Committed")
	// StatusFailed means the sheet could not be read at all.
	StatusFailed = Status("Failed")
	// StatusDiscarded means somebody read the dry run and said no.
	StatusDiscarded = Status("Discarded")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusParsed, StatusCommitted, StatusFailed, StatusDiscarded:
		return true
	default:
		return false
	}
}

// CanCommit reports whether this import is still waiting to be applied.
//
// Committing twice would amend the agreement twice, so the state is what stops
// a second click from doing it again.
func (s Status) CanCommit() bool {
	return s == StatusParsed
}

// SourceFormat is what kind of file was uploaded.
type SourceFormat string

const (
	SourceFormatCSV  = SourceFormat("CSV")
	SourceFormatXLSX = SourceFormat("XLSX")
)

func (f SourceFormat) String() string { return string(f) }

func (f SourceFormat) IsValid() bool {
	return f == SourceFormatCSV || f == SourceFormatXLSX
}
