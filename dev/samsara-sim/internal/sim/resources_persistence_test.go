package sim

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestAddressContactWebhookAndLiveShareWritesPersistAndReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	srv := newFleetTestServer(t, fleetServerOptions{})
	enableTestPersistence(t, srv.store, path)

	contact := requireStatus(t, srv, http.MethodPost, "/contacts", map[string]any{
		"firstName": "Rosa", "lastName": "Delgado",
	}, http.StatusOK).data(t)
	body := austinCircleAddress("Persisted Dock")
	body["contactIds"] = []any{recordID(contact)}
	body["tagIds"] = []any{tagAustinTerminal}
	address := requireStatus(t, srv, http.MethodPost, "/addresses", body, http.StatusOK).data(t)
	webhook := Record(requireStatus(t, srv, http.MethodPost, "/webhooks", map[string]any{
		"name":       "Persisted",
		"url":        "https://hooks.example.com/p",
		"eventTypes": []any{"AddressUpdated"},
	}, http.StatusOK).Payload)
	share := requireStatus(t, srv, http.MethodPost, "/live-shares", map[string]any{
		"name": "Persisted share", "type": "assetsNearLocation",
		"assetsNearLocationLinkConfig": map[string]any{"addressId": recordID(address)},
	}, http.StatusOK).data(t)
	if err := srv.store.ClosePersistence(); err != nil {
		t.Fatalf("close persistence: %v", err)
	}

	restored := loadDefaultFixtureStore(t)
	if status := enableTestPersistence(t, restored, path); !status.Restored {
		t.Fatalf("expected the state file to be restored, got %+v", status)
	}
	for resource, id := range map[Resource]string{
		ResourceContacts:   recordID(contact),
		ResourceAddresses:  recordID(address),
		ResourceWebhooks:   recordID(webhook),
		ResourceLiveShares: recordID(share),
	} {
		if _, err := restored.Get(resource, id); err != nil {
			t.Fatalf("expected %s %s to survive a restart: %v", resource, id, err)
		}
	}
	stored, err := restored.Get(ResourceWebhooks, recordID(webhook))
	if err != nil || stringValue(stored, "secretKey") != stringValue(webhook, "secretKey") {
		t.Fatalf("expected the generated secretKey to persist, got %v %v", stored, err)
	}

	if err = restored.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err = restored.Get(ResourceContacts, recordID(contact)); err == nil {
		t.Fatal("expected reset to drop the API-created contact")
	}
	contacts, err := restored.List(ResourceContacts)
	if err != nil || len(contacts) != 4 {
		t.Fatalf("expected reset to restore the 4 fixture contacts, got %d %v", len(contacts), err)
	}
}
