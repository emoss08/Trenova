package agentruntimeports

import "github.com/emoss08/trenova/shared/pulid"

// RunReader says whom a read of agent runs is for. A run on an assistant
// conversation is visible only to the person who owns that conversation: a
// read for a person keeps to the conversations they own, and a read for a
// principal that is no person (an agent, an API key) sees none of them. The
// zero value is UnattendedRunRead, the runtime's, a worker's or a system read,
// which sees every run in the tenant.
type RunReader struct {
	forPerson bool
	userID    pulid.ID
}

func UnattendedRunRead() RunReader { return RunReader{} }

// PersonRunRead is a read made for a request. A nil user id is a principal
// that is no person, and sees no conversation's runs.
func PersonRunRead(userID pulid.ID) RunReader {
	return RunReader{forPerson: true, userID: userID}
}

func (r RunReader) ForPerson() bool { return r.forPerson }

func (r RunReader) UserID() pulid.ID { return r.userID }
