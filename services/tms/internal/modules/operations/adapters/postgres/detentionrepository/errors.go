package detentionrepository

import "github.com/emoss08/trenova/pkg/dberror"

var occurrenceErrors = dberror.Mapping{Entity: "DetentionOccurrence"}

var evidenceErrors = dberror.Mapping{Entity: "DetentionEvidence"}

var noticeErrors = dberror.Mapping{Entity: "DetentionNotice"}

var policyErrors = dberror.Mapping{Entity: "DetentionPolicy"}
