package toolresult

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReturnedRecords_KnowsAKindABoardTakesThatHasNoPageOfItsOwn(t *testing.T) {
	t.Parallel()

	content := `{"items":[` +
		`{"occurrence":{"id":"dto_01M85EJ92J7WG468KK49V8818D","shipmentProNumber":"S26110041283015",` +
		`"locationName":"Northline Grocers DC"},"minutesOnSite":255,"holdBack":"NoticeLate"},` +
		`{"occurrence":{"id":"dto_01M85EJ92J7WG468KK49V8818E","shipmentProNumber":"S26110041319442"}}` +
		`]}`

	refs := ReturnedRecords(content)
	require.Len(t, refs, 2)
	assert.Equal(t, "detention_occurrence", refs[0].Type)
	assert.Equal(t, "dto_01M85EJ92J7WG468KK49V8818D", refs[0].ID)
	assert.Equal(t, "S26110041283015", refs[0].Label, "a stay is named by its shipment's PRO")
	assert.Equal(t, "S26110041319442", refs[1].Label)

	assert.Empty(t, CitedRecords(content), "a cited record still needs a page of its own")
}

func TestReturnedRecords_NamesAPartRecordByTheRecordItIsPartOf(t *testing.T) {
	t.Parallel()

	content := `{"columns":["id","workerId","credentialType","health","worker"],"count":2,` +
		`"items":[` +
		`{"credentialType":{"code":"MED_CARD","id":"wct_01M7V48E5HERT6ZQN8GJBGY46C",` +
		`"name":"DOT Medical Card"},"health":"Expired",` +
		`"id":"wcred_01M7V48E7D6K4KRNHMBA7X53YW","status":"Active",` +
		`"worker":{"firstName":"Jane","id":"wrk_01M7V48DVVQGDF0KG409BJH1FJ",` +
		`"lastName":"Doe","wholeName":"Jane Doe"}},` +
		`{"credentialType":{"code":"MED_CARD","id":"wct_01M7V48E5HERT6ZQN8GJBGY46C",` +
		`"name":"DOT Medical Card"},"health":"Expiring soon",` +
		`"id":"wcred_01M7V48E7E1FF9PPHATT0922WL","status":"Active",` +
		`"worker":{"firstName":"Robert","id":"wrk_01M7V48DVVQGDF0KG409BJH1FK",` +
		`"lastName":"Brown","wholeName":"Robert Brown"}}` +
		`]}`

	refs := ReturnedRecords(content)
	require.Len(t, refs, 2)
	assert.Equal(t, "worker_credential", refs[0].Type)
	assert.Equal(t, "Jane Doe", refs[0].Label,
		"a credential has no name of its own; the reply names it by its worker")
	assert.Equal(t, "Robert Brown", refs[1].Label)
}
