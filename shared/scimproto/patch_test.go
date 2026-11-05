package scimproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodePatch(t *testing.T, body string) []PatchOperation {
	t.Helper()

	req, err := DecodePatch([]byte(body), 50)
	require.NoError(t, err)

	return req.Operations
}

func TestApplyUserPatch_ReadsEntraStringBooleansAndFilteredPaths(t *testing.T) {
	t.Parallel()

	user := &User{
		UserName: "ada@example.com",
		Active:   BoolOf(true),
		Emails:   []MultiValue{{Value: "ada@example.com", Type: "work", Primary: true}},
	}
	ops := decodePatch(t, `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [
			{"op": "Replace", "path": "active", "value": "False"},
			{"op": "Replace", "path": "emails[type eq \"work\"].value", "value": "ada.l@example.com"},
			{"op": "Add", "path": "name.givenName", "value": "Ada"},
			{"op": "Replace", "path": "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department", "value": "Ops"}
		]
	}`)

	require.NoError(t, ApplyUserPatch(user, ops))
	assert.False(t, user.IsActive())
	assert.Equal(t, "ada.l@example.com", user.PrimaryEmail())
	require.NotNil(t, user.Name)
	assert.Equal(t, "Ada", user.Name.GivenName)
}

func TestApplyUserPatch_ReadsOktaPathlessReplace(t *testing.T) {
	t.Parallel()

	user := &User{UserName: "ada@example.com", Active: BoolOf(true)}
	ops := decodePatch(t, `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "replace", "value": {"active": false, "name": {"familyName": "Lovelace"}}}]
	}`)

	require.NoError(t, ApplyUserPatch(user, ops))
	assert.False(t, user.IsActive())
	assert.Equal(t, "Lovelace", user.Name.FamilyName)
}

func TestApplyUserPatch_RefusesToRemoveTheUserName(t *testing.T) {
	t.Parallel()

	user := &User{UserName: "ada@example.com"}
	ops := decodePatch(t, `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "remove", "path": "userName"}]
	}`)

	err := ApplyUserPatch(user, ops)
	scimErr, ok := AsError(err)
	require.True(t, ok)
	assert.Equal(t, ScimTypeMutability, scimErr.ScimType)
	assert.Equal(t, "ada@example.com", user.UserName)
}

func TestParseGroupPatch_KeepsMemberChangesInOrder(t *testing.T) {
	t.Parallel()

	ops := decodePatch(t, `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [
			{"op": "add", "path": "members", "value": [{"value": "usr_A"}, {"value": "usr_B"}]},
			{"op": "remove", "path": "members[value eq \"usr_A\"]"},
			{"op": "replace", "path": "displayName", "value": "Dispatch"}
		]
	}`)

	patch, err := ParseGroupPatch(ops)
	require.NoError(t, err)
	require.NotNil(t, patch.DisplayName)
	assert.Equal(t, "Dispatch", *patch.DisplayName)
	assert.Equal(t, []MemberOperation{
		{Change: MemberAdd, IDs: []string{"usr_A", "usr_B"}},
		{Change: MemberRemove, IDs: []string{"usr_A"}},
	}, patch.Members)
	assert.Equal(t, 3, patch.MemberCount())
}

func TestParseGroupPatch_RemovingMembersWithoutAValueEmptiesTheGroup(t *testing.T) {
	t.Parallel()

	ops := decodePatch(t, `{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "remove", "path": "members"}]
	}`)

	patch, err := ParseGroupPatch(ops)
	require.NoError(t, err)
	assert.Equal(t, []MemberOperation{{Change: MemberReplace, IDs: []string{}}}, patch.Members)
}

func TestDecodePatch_RefusesTooManyOperationsAndUnknownVerbs(t *testing.T) {
	t.Parallel()

	_, err := DecodePatch([]byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "move", "path": "active", "value": true}]
	}`), 10)
	require.Error(t, err)

	_, err = DecodePatch([]byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "add", "path": "active", "value": true}, {"op": "add", "path": "title", "value": "x"}]
	}`), 1)
	scimErr, ok := AsError(err)
	require.True(t, ok)
	assert.Equal(t, ScimTypeTooMany, scimErr.ScimType)
}
