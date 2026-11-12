// Package caseownerquery is the one predicate every list of work about
// records shares: whether a row's record has a case the reader may see.
package caseownerquery

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner/caseownerdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare/recordsharedb"
)

// InSight keeps a row about a record whose case is someone else's away from
// a reader who neither owns that case, covers for its owner, nor sees every
// case, unless the record was shared with them: a share brings its case into
// their sight to read. A row about no record, or about an unowned one, is
// everyone's. It is one probe of the owners' unique index per row, and one of
// the shares' for a row that fails it, whatever the tenant's size.
func InSight(record keel.Expr, sight *caseowner.Sight) keel.Pred {
	if sight == nil || sight.All {
		return nil
	}

	o := caseownerdb.Owners
	notSomeoneElses := keel.NotExists(
		o.Where(o.RecordID.EqCol(record), o.OwnerID.NotIn(sight.As...)),
	)
	if sight.Viewer.IsNil() {
		return notSomeoneElses
	}

	s := recordsharedb.Shares

	return keel.Or(
		notSomeoneElses,
		keel.Exists(s.Where(
			s.RecordID.EqCol(record),
			s.SharedWithID.Eq(sight.Viewer),
			s.RevokedAt.IsNull(),
		)),
	)
}
