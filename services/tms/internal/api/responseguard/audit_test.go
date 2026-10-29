package responseguard

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// auditedRecord is a record with protected fields as the audit trail stores
// it: its JSON, under the resource its entries name.
type auditedRecord struct {
	resource permission.Resource
	build    func() protectedfield.Withholder
}

// auditedRecords are the stored records with protected fields: the personal
// details records and every domain record restMargins carries. A ports read
// model is computed per request and never written to the audit trail.
func auditedRecords() []auditedRecord {
	records := []auditedRecord{
		{
			resource: permission.ResourceWorker,
			build: func() protectedfield.Withholder {
				return &worker.Worker{FirstName: "Rosa", Profile: &worker.WorkerProfile{
					DOB:            315532800,
					LicenseNumber:  "D1234567",
					TWICCardNumber: "TWIC-998877",
					LastDrugTest:   1767225600,
				}}
			},
		},
		{
			resource: permission.ResourceWorkerCredential,
			build: func() protectedfield.Withholder {
				return &workercredential.WorkerCredential{Number: "CDL-445566"}
			},
		},
	}
	for _, entry := range restMargins() {
		if strings.Contains(typeKey(entry.build()), "/ports/") {
			continue
		}
		resource := permission.Resource("")
		if _, quote := entry.build().(*ratequote.RateQuote); quote {
			resource = permission.ResourceRateQuote
		}
		records = append(records, auditedRecord{resource: resource, build: entry.build})
	}

	return records
}

// withheldLeaves lists the JSON paths a value holds that are set before and
// empty after, by the same emptiness the REST tests read.
func withheldLeaves(before, after any, path string, out *[]string) {
	switch typed := before.(type) {
	case map[string]any:
		afterMap, _ := after.(map[string]any)
		for key, value := range typed {
			next := key
			if path != "" {
				next = path + "." + key
			}
			withheldLeaves(value, afterMap[key], next, out)
		}
	case []any:
		afterList, _ := after.([]any)
		for i, value := range typed {
			var other any
			if i < len(afterList) {
				other = afterList[i]
			}
			withheldLeaves(value, other, fmt.Sprintf("%s[%d]", path, i), out)
		}
	default:
		if valueAtPath(before) != nil && valueAtPath(after) == nil {
			*out = append(*out, path)
		}
	}
}

func valueAtPath(value any) any {
	if text, ok := value.(string); ok && text == "" {
		return nil
	}
	return value
}

func leafAt(doc any, path string) any {
	current := doc
	for path != "" {
		var step string
		step, path = nextStep(path)
		switch typed := current.(type) {
		case map[string]any:
			current = typed[step]
		case []any:
			var index int
			if _, err := fmt.Sscanf(step, "%d", &index); err != nil || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	return current
}

func nextStep(path string) (string, string) {
	for i, r := range path {
		switch r {
		case '.':
			return path[:i], path[i+1:]
		case '[':
			if i > 0 {
				return path[:i], path[i:]
			}
			end := 1
			for end < len(path) && path[end] != ']' {
				end++
			}
			rest := path[min(end+1, len(path)):]
			if len(rest) > 0 && rest[0] == '.' {
				rest = rest[1:]
			}
			return path[1:end], rest
		}
	}
	return path, ""
}

// Every protected field a record's WithholdFields clears in a response is
// also withheld from that record's JSON in the audit trail, so an audit
// reader without the group never reads a diff the record would not show.
func TestEveryProtectedFieldIsWithheldFromTheAuditTrail(t *testing.T) {
	t.Parallel()

	checked := 0
	for _, record := range auditedRecords() {
		key := typeKey(record.build())
		state := encode(t, record.build())
		cleared := record.build()
		cleared.WithholdFields(0)

		var paths []string
		withheldLeaves(state, encode(t, cleared), "", &paths)
		require.NotEmptyf(t, paths, "%s withholds nothing; give it a protected field", key)

		entry := &audit.Entry{Resource: record.resource, CurrentState: state}
		entry.WithholdFields(0)
		for _, path := range paths {
			checked++
			assert.Nilf(t, leafAt(entry.CurrentState, path),
				"%s.%s is withheld from a response but not from an audit entry of %q; "+
					"declare it in protectedfield.Paths", key, path, record.resource)
		}

		full := &audit.Entry{Resource: record.resource, CurrentState: encode(t, record.build())}
		full.WithholdFields(protectedfield.AllGroups())
		for _, path := range paths {
			assert.NotNilf(t, leafAt(full.CurrentState, path),
				"%s.%s was withheld from a reader who holds every group", key, path)
		}
	}
	assert.Positive(t, checked)
}

func TestAnAuditEntryResponseWithholdsItsProtectedFields(t *testing.T) {
	t.Parallel()

	entry := func() *audit.Entry {
		return &audit.Entry{
			Resource:     permission.ResourceCostingControl,
			CurrentState: map[string]any{"targetMarginPercent": "15", "milesPerGallon": "6.5"},
			Changes: map[string]any{
				"targetMarginPercent": map[string]any{"from": "10", "to": "15", "type": "updated"},
			},
		}
	}

	body, _ := serve(
		t,
		[]permission.Resource{permission.ResourceProtectedMargins},
		entry(),
		guarded,
	)
	assert.Nil(t, object(t, body, "currentState")["targetMarginPercent"])
	assert.Equal(t, "6.5", object(t, body, "currentState")["milesPerGallon"])
	assert.Nil(t, object(t, body, "changes", "targetMarginPercent")["to"])

	body, _ = serve(t, nil, entry(), guarded)
	assert.Equal(t, "15", object(t, body, "currentState")["targetMarginPercent"])
}
