package documenttemplate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This mirrors permission/registry_coverage_test.go: the constant list and the
// registration calls are separate declarations, and nothing but a test can keep them
// in step. Forgetting a Register call would otherwise ship a kind that exists in the
// enum, appears in nothing, and fails at render time with "unknown template kind".

const kindDeclarationFile = "kind_gen.go"

// parseKindConstants reads the Kind constants straight out of the source.
func parseKindConstants(t *testing.T) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, kindDeclarationFile, nil, parser.SkipObjectResolution)
	require.NoError(t, err, "could not parse %s", kindDeclarationFile)

	var values []string

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, specOK := spec.(*ast.ValueSpec)
			if !specOK {
				continue
			}
			// Only constants explicitly typed Kind.
			ident, identOK := valueSpec.Type.(*ast.Ident)
			if !identOK || ident.Name != "Kind" {
				continue
			}
			for _, value := range valueSpec.Values {
				literal, litOK := value.(*ast.BasicLit)
				if !litOK || literal.Kind != token.STRING {
					continue
				}
				unquoted, unquoteErr := strconv.Unquote(literal.Value)
				require.NoError(t, unquoteErr)
				values = append(values, unquoted)
			}
		}
	}

	require.NotEmpty(t, values, "no Kind constants found in %s", kindDeclarationFile)
	return values
}

// TestEveryDeclaredKindIsRegistered is the gate. A constant with no definition is a
// kind that resolves to nothing at render time.
func TestEveryDeclaredKindIsRegistered(t *testing.T) {
	declared := parseKindConstants(t)
	registry := NewRegistry()

	for _, value := range declared {
		def, ok := registry.Get(Kind(value))
		assert.True(t, ok,
			"kind %q is declared in %s but never registered; add a Register call in "+
				"the matching register*Kinds method", value, kindDeclarationFile)
		if ok {
			assert.Equal(t, Kind(value), def.Kind)
		}
	}
}

// TestNoUnregisteredKindsInRegistry is the reverse: a registered kind with no
// constant cannot be referenced from Go, so nothing would ever request it.
func TestNoUnregisteredKindsInRegistry(t *testing.T) {
	declared := parseKindConstants(t)
	registry := NewRegistry()

	for _, def := range registry.All() {
		assert.Contains(t, declared, string(def.Kind),
			"kind %q is registered but has no constant in %s", def.Kind, kindDeclarationFile)
	}
}

// TestAllKindsMatchesDeclaredConstants keeps the convenience slice honest, since the
// coverage tests and the seeder both walk it.
func TestAllKindsMatchesDeclaredConstants(t *testing.T) {
	declared := parseKindConstants(t)

	listed := make([]string, 0, len(AllKinds()))
	for _, kind := range AllKinds() {
		listed = append(listed, string(kind))
	}

	slices.Sort(declared)
	slices.Sort(listed)
	assert.Equal(t, declared, listed,
		"AllKinds() must list every declared Kind constant")
}

func TestKindDefinitionsAreCoherent(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		t.Run(string(def.Kind), func(t *testing.T) {
			assert.NotEmpty(t, def.DisplayName, "the admin catalog shows this")
			assert.NotEmpty(t, def.Description, "the admin catalog shows this")
			assert.NotEmpty(t, def.Category, "kinds are grouped by category")
			assert.NotEmpty(t, def.Channels)
			assert.NotEmpty(t, def.Variables)

			// Page setup only means something for a document.
			if def.Paged {
				assert.True(t, def.HasChannel(ChannelPDF),
					"only a PDF kind can be paged")
			}
			if def.HasChannel(ChannelPDF) {
				assert.True(t, def.Paged,
					"a PDF kind must be paged, or its page size and margins are ignored")
			}

			// An email kind needs all three slots: a subject, a rendered body, and a
			// plain-text fallback. Shipping without the fallback is how a message
			// ends up filtered as spam.
			if def.HasChannel(ChannelEmailHTML) {
				assert.True(t, def.HasChannel(ChannelSubject),
					"an email kind needs a subject channel")
				assert.True(t, def.HasChannel(ChannelEmailText),
					"an email kind needs a plain-text channel")
			}
		})
	}
}

// TestCustomerScopedKindsCanActuallyResolveACustomer catches an assignment that
// would never fire: a kind with no customer in scope cannot be assigned per
// customer, and offering it in the UI only produces confusion.
func TestCustomerScopedKindsCanActuallyResolveACustomer(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		if !def.CustomerScoped {
			continue
		}

		t.Run(string(def.Kind), func(t *testing.T) {
			paths := registry.Paths(def.Kind)

			hasCustomer := false
			for _, path := range paths {
				if strings.Contains(path, "Customer") || strings.Contains(path, "BillTo") {
					hasCustomer = true
					break
				}
			}
			assert.True(t, hasCustomer,
				"%s is customer-scoped but its context names no customer, so a "+
					"per-customer assignment could never be meaningful", def.Kind)
		})
	}
}

func TestNotCustomerScopedKinds(t *testing.T) {
	registry := NewRegistry()

	// A driver invitation goes to a worker and a scheduled report to a subscriber.
	// Neither has a customer, so neither may be customer-scoped.
	for _, kind := range []Kind{KindDriverPortalInvitationEmail, KindReportDeliveryEmail, KindReportPDF} {
		def, ok := registry.Get(kind)
		require.True(t, ok)
		assert.False(t, def.CustomerScoped,
			"%s has no customer in scope", kind)
	}
}

func TestRegisterRejectsIncoherentDefinitions(t *testing.T) {
	tests := []struct {
		name string
		def  *KindDefinition
	}{
		{"nil", nil},
		{"no key", &KindDefinition{DisplayName: "x"}},
		{
			"no channels",
			&KindDefinition{Kind: Kind("test.x"), sampleFactory: func() any { return struct{}{} }},
		},
		{"no sample", &KindDefinition{Kind: Kind("test.x"), Channels: []Channel{ChannelPDF}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &Registry{kinds: map[Kind]*KindDefinition{}}
			assert.Error(t, registry.Register(tt.def))
		})
	}
}

func TestRegisterRejectsDuplicates(t *testing.T) {
	registry := &Registry{kinds: map[Kind]*KindDefinition{}}
	def := &KindDefinition{
		Kind:          Kind("test.dup"),
		Channels:      []Channel{ChannelPDF},
		sampleFactory: func() any { return struct{}{} },
	}

	require.NoError(t, registry.Register(def))
	assert.Error(t, registry.Register(def))
}

func TestStarterKeyMatchesEveryKind(t *testing.T) {
	registry := NewRegistry()

	for _, def := range registry.All() {
		key := def.StarterKey()
		assert.NotEmpty(t, key)
		assert.NotContains(t, key, "/", "a starter key becomes a filename")
		assert.NotContains(t, key, string(filepath.Separator))
	}
}

func TestCategoriesAreStable(t *testing.T) {
	registry := NewRegistry()

	categories := registry.Categories()
	assert.True(t, slices.IsSorted(categories))
	assert.Contains(t, categories, "Billing")
	assert.Contains(t, categories, "Detention")
	assert.Contains(t, categories, "Reporting")

	for _, category := range categories {
		assert.NotEmpty(t, registry.ByCategory(category))
	}
}

func TestKindDeclarationFileExists(t *testing.T) {
	// The coverage tests read this file by name; a rename must fail loudly here
	// rather than silently stop checking anything.
	_, err := os.Stat(kindDeclarationFile)
	require.NoError(t, err)
}

func TestUnknownKindLookups(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.Get(Kind("nope.nope"))
	assert.False(t, ok)
	assert.Nil(t, registry.Paths(Kind("nope.nope")))
	assert.Nil(t, registry.RequiredPaths(Kind("nope.nope")))

	_, err := registry.SampleContext(Kind("nope.nope"))
	assert.Error(t, err)

	_, err = registry.HostileProbeContext(Kind("nope.nope"))
	assert.Error(t, err)
}

func TestChannelFieldMapping(t *testing.T) {
	tests := []struct {
		channel Channel
		want    string
	}{
		{ChannelPDF, "bodyHtml"},
		{ChannelEmailHTML, "bodyHtml"},
		{ChannelEmailText, "bodyText"},
		{ChannelSubject, "subject"},
		{ChannelNotificationTitle, "notificationTitle"},
		{ChannelNotificationBody, "notificationMessage"},
	}

	for _, tt := range tests {
		t.Run(string(tt.channel), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.channel.Field())
		})
	}
}

func TestChannelMapsToEngineChannel(t *testing.T) {
	// The domain owns the enum the database and GraphQL see; the engine has its own
	// copy so it can stay dependency-free. They must agree.
	for _, channel := range []Channel{
		ChannelPDF, ChannelEmailHTML, ChannelEmailText,
		ChannelSubject, ChannelNotificationTitle, ChannelNotificationBody,
	} {
		assert.Equal(t, string(channel), string(channel.Engine()))
	}
}
