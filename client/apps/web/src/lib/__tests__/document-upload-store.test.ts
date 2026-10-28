import type { DocumentUploadSession } from "@trenova/shared/types/document";
import { describe, expect, it } from "vitest";
import {
  createDocumentUploadStore,
  type PersistedUploadRecord,
  type UploadRecordBackend,
} from "../document-upload-store";

const NOW_MS = 1_790_000_000_000;
const JANE = "usr_jane:org_1";
const OMAR = "usr_omar:org_1";

function memoryBackend(): UploadRecordBackend & { records: Map<string, PersistedUploadRecord> } {
  const records = new Map<string, PersistedUploadRecord>();
  return {
    records,
    put: async (record) => void records.set(record.sessionId, record),
    delete: async (sessionId) => void records.delete(sessionId),
    all: async () => [...records.values()],
    clear: async () => records.clear(),
  };
}

function record(
  sessionId: string,
  { scope = JANE, resourceId = "shp_1", expiresAt = NOW_MS / 1000 + 3600 } = {},
): PersistedUploadRecord {
  return {
    sessionId,
    scope,
    resourceId,
    resourceType: "shipment",
    session: { id: sessionId, expiresAt } as DocumentUploadSession,
    file: new File(["%PDF-1.7"], "bol.pdf", { type: "application/pdf" }),
  };
}

function storeSignedInAs(scope: string | null) {
  const backend = memoryBackend();
  let signedIn = scope;
  const store = createDocumentUploadStore(backend, {
    activeScope: () => signedIn,
    now: () => NOW_MS,
  });
  return {
    backend,
    store,
    signIn: (next: string | null) => {
      signedIn = next;
    },
  };
}

describe("document upload records", () => {
  it("keeps an upload's file for the person who started it, for the resource it goes to", async () => {
    const { backend, store } = storeSignedInAs(JANE);
    await store.persist(record("dus_1"));
    await store.persist(record("dus_2", { resourceId: "shp_2" }));

    const listed = await store.list(JANE, "shipment", "shp_1");

    expect(listed.map((entry) => entry.sessionId)).toEqual(["dus_1"]);
    expect([...backend.records.keys()]).toEqual(["dus_1", "dus_2"]);
  });

  it("never offers one person's upload to another, and deletes it when they look", async () => {
    const { backend, store, signIn } = storeSignedInAs(JANE);
    await store.persist(record("dus_1"));
    signIn(OMAR);

    expect(await store.list(OMAR, "shipment", "shp_1")).toEqual([]);
    expect(backend.records.size).toBe(0);
  });

  it("deletes an upload the server has already expired", async () => {
    const { backend, store } = storeSignedInAs(JANE);
    await store.persist(record("dus_old", { expiresAt: NOW_MS / 1000 - 1 }));
    await store.persist(record("dus_new"));

    const listed = await store.list(JANE, "shipment", "shp_1");

    expect(listed.map((entry) => entry.sessionId)).toEqual(["dus_new"]);
    expect([...backend.records.keys()]).toEqual(["dus_new"]);
  });

  it("drops a write from an upload whose session ended while it ran", async () => {
    const { backend, store, signIn } = storeSignedInAs(JANE);
    signIn(null);
    await store.persist(record("dus_1"));
    signIn(OMAR);
    await store.persist(record("dus_2"));

    expect(backend.records.size).toBe(0);
  });

  it("deletes every upload's file at once", async () => {
    const { backend, store } = storeSignedInAs(JANE);
    await store.persist(record("dus_1"));
    await store.persist(record("dus_2", { resourceId: "shp_2" }));

    await store.clear();

    expect(backend.records.size).toBe(0);
  });

  it("keeps nothing where the browser has no storage", async () => {
    const store = createDocumentUploadStore(null, { activeScope: () => JANE, now: () => NOW_MS });
    await store.persist(record("dus_1"));

    expect(await store.list(JANE, "shipment", "shp_1")).toEqual([]);
  });
});
