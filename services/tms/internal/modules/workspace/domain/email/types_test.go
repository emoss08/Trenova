package email

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryEmailTypeDeclaresAKind(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, Types())
	for _, typ := range Types() {
		assert.True(t, typ.Kind().IsValid(), "email type %q declares no kind", typ)
		if typ.IsInvitation() {
			assert.Equal(t, KindSystem, typ.Kind(), "invitation %q must be a system email", typ)
		}
	}
}

func TestEveryDeclaredEmailTypeIsDefinedOnce(t *testing.T) {
	t.Parallel()

	file, err := parser.ParseFile(token.NewFileSet(), "types.go", nil, 0)
	require.NoError(t, err)

	declared := map[string]struct{}{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			for i, name := range value.Names {
				if !strings.HasPrefix(name.Name, "Type") {
					continue
				}
				require.Less(t, i, len(value.Values), name.Name)
				call, isCall := value.Values[i].(*ast.CallExpr)
				require.True(t, isCall, "%s must be made by defineType or defineInvitation", name.Name)
				fn, isIdent := call.Fun.(*ast.Ident)
				require.True(t, isIdent, name.Name)
				assert.Contains(t, []string{"defineType", "defineInvitation"}, fn.Name, name.Name)
				declared[name.Name] = struct{}{}
			}
		}
	}

	assert.Len(t, Types(), len(declared), "two email types share a name")
}

func TestAnUndefinedEmailTypeHasNoKind(t *testing.T) {
	t.Parallel()

	assert.False(t, Type("Unknown").IsValid())
	assert.Empty(t, Type("Unknown").Kind())
	assert.False(t, Type("Unknown").IsSystem())
}

func TestEmailToTheOrganizationsOwnPeopleIsSystemEmail(t *testing.T) {
	t.Parallel()

	for _, typ := range []Type{
		TypePasswordReset,
		TypeAccountInvitation,
		TypeDriverPortalInvitation,
		TypeNotification,
		TypeNotificationDigest,
		TypeProposalReminder,
		TypeRecordShare,
		TypeReportDelivery,
	} {
		assert.Equal(t, KindSystem, typ.Kind(), typ)
	}
	assert.True(t, TypeAccountInvitation.IsInvitation())
	assert.True(t, TypeDriverPortalInvitation.IsInvitation())
	assert.False(t, TypePasswordReset.IsInvitation())
}

func TestEmailOnTheOrganizationsBehalfIsCustomerFacing(t *testing.T) {
	t.Parallel()

	for _, typ := range []Type{
		TypeInvoice,
		TypeCustomerStatement,
		TypePaymentReminder,
		TypeCustomerStatusUpdate,
		TypeCustomerTrackingLink,
		TypeCarrierTrackingLink,
		TypeTenderOffer,
		TypeRateConfirmation,
		TypeDetentionNotice,
		TypeInboundReply,
	} {
		assert.Equal(t, KindCustomerFacing, typ.Kind(), typ)
		assert.False(t, typ.IsSystem(), typ)
	}
}
