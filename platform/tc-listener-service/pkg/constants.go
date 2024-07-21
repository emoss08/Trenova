package pkg

type CommitStyle int

const (
	AutoCommit CommitStyle = iota
	ManualCommitRecords
	ManualCommitUncommitted
)
