package reportcatalog

import (
	"errors"
	"testing"
)

func TestDefaultCatalogIsIndexed(t *testing.T) {
	if len(Default.Entities) == 0 {
		t.Fatal("Default catalog has no entities")
	}
	if Default.Version == "" {
		t.Fatal("Default catalog has no version")
	}

	shipment, ok := Default.Entity("shipment")
	if !ok {
		t.Fatal("shipment entity not found")
	}
	if shipment.Table.Name != "shipments" {
		t.Errorf("shipment table = %q, want shipments", shipment.Table.Name)
	}
	if !shipment.Tenant.IsTenanted() {
		t.Error("shipment must be tenanted")
	}
	if shipment.OwnershipColumn != "owner_id" {
		t.Errorf("shipment ownership column = %q, want owner_id", shipment.OwnershipColumn)
	}

	pro, ok := shipment.Field("proNumber")
	if !ok {
		t.Fatal("shipment.proNumber not found")
	}
	if pro.Label != "PRO Number" {
		t.Errorf("proNumber label = %q, want PRO Number", pro.Label)
	}

	charge, ok := shipment.Field("totalChargeAmount")
	if !ok {
		t.Fatal("shipment.totalChargeAmount not found")
	}
	if charge.Type != FieldDecimal || charge.Format != FormatMoney {
		t.Errorf("totalChargeAmount type/format = %s/%s, want decimal/money", charge.Type, charge.Format)
	}
	if !charge.SupportsAggregation(AggSum) {
		t.Error("totalChargeAmount must support SUM")
	}
	if charge.Groupable {
		t.Error("decimal fields must not default to groupable")
	}
}

func TestEveryEntityIsInternallyConsistent(t *testing.T) {
	for i := range Default.Entities {
		entity := &Default.Entities[i]

		if len(entity.Table.PrimaryKey) == 0 {
			t.Errorf("entity %q has no primary key", entity.Key)
		}

		for j := range entity.Fields {
			field := &entity.Fields[j]
			if field.Column.Alias != entity.Table.Alias {
				t.Errorf("entity %q field %q alias %q != table alias %q",
					entity.Key, field.Key, field.Column.Alias, entity.Table.Alias)
			}
			if field.Type == FieldJSON && (field.Filterable || field.Groupable) {
				t.Errorf("entity %q json field %q is filterable/groupable", entity.Key, field.Key)
			}
		}

		for j := range entity.Edges {
			edge := &entity.Edges[j]
			if edge.Source != entity.Key {
				t.Errorf("entity %q edge %q has source %q", entity.Key, edge.Name, edge.Source)
			}
			if _, ok := Default.Entity(edge.Target); !ok {
				t.Errorf("entity %q edge %q targets unknown entity %q", entity.Key, edge.Name, edge.Target)
			}
			if edge.Cardinality == CardinalityM2M && edge.Through == nil {
				t.Errorf("entity %q m2m edge %q has no through join", entity.Key, edge.Name)
			}
			if edge.Cardinality != CardinalityM2M && len(edge.Join) == 0 {
				t.Errorf("entity %q edge %q has no join pairs", entity.Key, edge.Name)
			}
		}
	}
}

func TestResolvePath(t *testing.T) {
	base, resolved, err := Default.ResolvePath("shipment", []string{"customer"})
	if err != nil {
		t.Fatalf("ResolvePath(shipment, customer) error = %v", err)
	}
	if base.Key != "shipment" {
		t.Errorf("base = %q, want shipment", base.Key)
	}
	if terminal := resolved.Terminal(base); terminal.Key != "customer" {
		t.Errorf("terminal = %q, want customer", terminal.Key)
	}
	if resolved.CrossesToMany() {
		t.Error("shipment→customer must not cross to-many")
	}

	_, resolved, err = Default.ResolvePath(
		"shipment",
		[]string{"moves", "assignment", "primaryWorker"},
	)
	if err != nil {
		t.Fatalf("ResolvePath(shipment, moves.assignment.primaryWorker) error = %v", err)
	}
	if !resolved.CrossesToMany() {
		t.Error("path through moves must be flagged to-many")
	}
	if terminal := resolved.Terminal(nil); terminal.Key != "worker" {
		t.Errorf("terminal = %q, want worker", terminal.Key)
	}

	_, _, err = Default.ResolvePath("shipment", []string{"nonexistent"})
	if !errors.Is(err, ErrUnknownEdge) {
		t.Errorf("expected ErrUnknownEdge, got %v", err)
	}

	_, _, err = Default.ResolvePath("nonexistent", nil)
	if !errors.Is(err, ErrUnknownEntity) {
		t.Errorf("expected ErrUnknownEntity, got %v", err)
	}
}

func TestEmptyPathResolvesToBase(t *testing.T) {
	base, resolved, err := Default.ResolvePath("order", nil)
	if err != nil {
		t.Fatalf("ResolvePath(order, nil) error = %v", err)
	}
	if len(resolved.Steps) != 0 {
		t.Errorf("expected no steps, got %d", len(resolved.Steps))
	}
	if terminal := resolved.Terminal(base); terminal.Key != "order" {
		t.Errorf("terminal = %q, want order", terminal.Key)
	}
}
