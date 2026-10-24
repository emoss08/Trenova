package listquery

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

type testStatus string

type testOwner struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	Name           string `json:"name"`
	Secret         string `json:"-"`
}

type testTruck struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	BusinessUnitID string     `json:"businessUnitId"`
	OwnerID        string     `json:"ownerId"`
	Code           string     `json:"code"`
	Status         testStatus `json:"status"`
	Year           *int64     `json:"year"`
	Weight         *float64   `json:"weight"`
	Description    string     `json:"description"`
	Active         bool       `json:"active"`
	SearchVector   string     `json:"-"`
	Token          string     `json:"-"`
	CreatedAt      int64      `json:"createdAt"`

	Owner *testOwner   `json:"owner,omitempty"`
	Loads []*testOwner `json:"loads,omitempty"`
}

func (t *testTruck) GetTableName() string { return "test_trucks" }

func (t *testTruck) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "Owner", Queryable: true},
			{Field: "loads", Queryable: true},
		},
	}
}

type testPlainTruck struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	BusinessUnitID string     `json:"businessUnitId"`
	OwnerID        string     `json:"ownerId"`
	Code           string     `json:"code"`
	Status         testStatus `json:"status"`
	Year           *int64     `json:"year"`
	Weight         *float64   `json:"weight"`
	Description    string     `json:"description"`
	Active         bool       `json:"active"`
	SearchVector   string     `json:"-"`
	Token          string     `json:"-"`
	CreatedAt      int64      `json:"createdAt"`

	Owner *testOwner `json:"owner,omitempty"`
}

func (t *testPlainTruck) GetTableName() string { return "test_plain_trucks" }

func (t *testPlainTruck) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "year", Type: domaintypes.FieldTypeNumber},
		},
	}
}

func tenantColumns() []schema.Column {
	return []schema.Column{
		{Name: "id", Field: "ID", Kind: schema.KindString, Flags: schema.FlagPK},
		{
			Name:  "organization_id",
			Field: "OrganizationID",
			Kind:  schema.KindString,
			Flags: schema.FlagPK | schema.FlagTenant,
		},
		{
			Name:  "business_unit_id",
			Field: "BusinessUnitID",
			Kind:  schema.KindString,
			Flags: schema.FlagPK | schema.FlagTenant,
		},
	}
}

var testJSONNames = map[string]string{
	"ID":             "id",
	"OrganizationID": "organizationId",
	"BusinessUnitID": "businessUnitId",
	"OwnerID":        "ownerId",
	"Code":           "code",
	"Status":         "status",
	"Year":           "year",
	"Weight":         "weight",
	"Description":    "description",
	"Active":         "active",
	"CreatedAt":      "createdAt",
	"Name":           "name",
}

func quoted(cols []schema.Column) []schema.Column {
	for i := range cols {
		cols[i].Quoted = schema.Quote(cols[i].Name)
		cols[i].JSON = testJSONNames[cols[i].Field]
	}

	return cols
}

func newOwnersTable() *schema.Table {
	t := &schema.Table{
		Name:   "test_owners",
		Quoted: schema.Quote("test_owners"),
		Alias:  "o",
		Columns: quoted(append(tenantColumns(),
			schema.Column{Name: "name", Field: "Name", Kind: schema.KindString},
			schema.Column{Name: "secret", Field: "Secret", Kind: schema.KindString},
			schema.Column{Name: "truck_id", Field: "TruckID", Kind: schema.KindString},
		)),
	}
	schema.Derive(t)

	return t
}

func newTrucksTable(name, alias string, owners *schema.Table) *schema.Table {
	t := &schema.Table{
		Name:   name,
		Quoted: schema.Quote(name),
		Alias:  alias,
		Columns: quoted(append(
			tenantColumns(),
			schema.Column{Name: "owner_id", Field: "OwnerID", Kind: schema.KindString},
			schema.Column{Name: "code", Field: "Code", Kind: schema.KindString},
			schema.Column{
				Name:  "status",
				Field: "Status",
				Kind:  schema.KindString,
				Enum:  []schema.EnumValue{{Value: "Available"}, {Value: "OutOfService"}},
			},
			schema.Column{
				Name:  "year",
				Field: "Year",
				Kind:  schema.KindInt64,
				Flags: schema.FlagNullable,
			},
			schema.Column{
				Name:  "weight",
				Field: "Weight",
				Kind:  schema.KindFloat64,
				Flags: schema.FlagNullable,
			},
			schema.Column{
				Name:  "description",
				Field: "Description",
				Kind:  schema.KindString,
				Flags: schema.FlagNullZero,
			},
			schema.Column{Name: "active", Field: "Active", Kind: schema.KindBool},
			schema.Column{
				Name:  "search_vector",
				Field: "SearchVector",
				Kind:  schema.KindTSVector,
				Flags: schema.FlagReadOnly | schema.FlagNoSelect,
			},
			schema.Column{Name: "token", Field: "Token", Kind: schema.KindString},
			schema.Column{Name: "created_at", Field: "CreatedAt", Kind: schema.KindInt64},
		)),
		Relations: []schema.Relation{
			{
				Name:   "Owner",
				JSON:   "owner",
				Alias:  "ow",
				Kind:   schema.BelongsTo,
				Target: owners,
				Local:  []int32{3},
				Remote: []int32{0},
			},
			{
				Name:   "Loads",
				JSON:   "loads",
				Alias:  "ld",
				Kind:   schema.HasMany,
				Target: owners,
				Local:  []int32{0},
				Remote: []int32{5},
			},
		},
	}
	schema.Derive(t)

	return t
}

var (
	testOwners = newOwnersTable()
	testTrucks = keel.Handle[testTruck]{
		Ref: &keel.Ref{Table: newTrucksTable("test_trucks", "t", testOwners)},
	}
	testPlainTrucks = keel.Handle[testPlainTruck]{
		Ref: &keel.Ref{Table: newTrucksTable("test_plain_trucks", "pt", testOwners)},
	}
)

func render(t *testing.T, q keeltest.Renderer) (string, []any) {
	t.Helper()

	return keeltest.Render(t, q)
}
