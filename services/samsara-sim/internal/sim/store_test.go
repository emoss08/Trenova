package sim

import "testing"

func TestStoreCRUD(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{
		Addresses: []Record{
			{
				"id":   "addr-1",
				"name": "HQ",
			},
		},
	})

	created, err := store.Create(ResourceAddresses, Record{"name": "Yard"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if recordID(created) == "" {
		t.Fatal("expected created address id")
	}

	addressID := recordID(created)
	got, err := store.Get(ResourceAddresses, addressID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if stringValue(got, "name") != "Yard" {
		t.Fatalf("expected Yard name, got %q", stringValue(got, "name"))
	}

	updated, err := store.Patch(ResourceAddresses, addressID, Record{"name": "Updated Yard"})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if stringValue(updated, "name") != "Updated Yard" {
		t.Fatalf("expected Updated Yard, got %q", stringValue(updated, "name"))
	}

	if err = store.Delete(ResourceAddresses, addressID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err = store.Get(ResourceAddresses, addressID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestStoreWebhookTargetsFilter(t *testing.T) {
	t.Parallel()

	store := NewStore(&Fixture{
		Webhooks: []Record{
			{
				"id":         "wh-1",
				"name":       "Address hook",
				"url":        "http://localhost/hook-1",
				"eventTypes": []any{"AddressCreated"},
			},
			{
				"id":   "wh-2",
				"name": "Catch all",
				"url":  "http://localhost/hook-2",
			},
		},
	})

	targets := store.WebhookTargets("AddressCreated")
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	targets = store.WebhookTargets("FormSubmitted")
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "wh-2" {
		t.Fatalf("expected wh-2 fallback target, got %s", targets[0].ID)
	}
}
