import type { DocumentUploadSession } from "@/types/document";

const DB_NAME = "trenova-document-uploads";
const DB_VERSION = 1;
const STORE_NAME = "sessions";

type PersistedUploadRecord = {
  sessionId: string;
  resourceId: string;
  resourceType: string;
  session: DocumentUploadSession;
  file: File;
};

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "sessionId" });
      }
    };

    request.onsuccess = () => resolve(request.result);
    request.onerror = () =>
      reject(request.error ?? new Error("Failed to open upload persistence"));
  });
}

async function withStore<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T> | void,
): Promise<T | undefined> {
  const db = await openDB();

  return await new Promise<T | undefined>((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, mode);
    const store = tx.objectStore(STORE_NAME);
    const request = run(store);

    tx.oncomplete = () => resolve(request?.result);
    tx.onerror = () => reject(tx.error ?? new Error("IndexedDB transaction failed"));
    tx.onabort = () => reject(tx.error ?? new Error("IndexedDB transaction aborted"));
  }).finally(() => db.close());
}

export async function persistDocumentUploadSession(
  record: PersistedUploadRecord,
): Promise<void> {
  await withStore("readwrite", (store) => store.put(record));
}

export async function removeDocumentUploadSession(sessionId: string): Promise<void> {
  await withStore("readwrite", (store) => store.delete(sessionId));
}

export async function listPersistedDocumentUploadSessions(
  resourceType: string,
  resourceId: string,
): Promise<PersistedUploadRecord[]> {
  const records = (await withStore<PersistedUploadRecord[]>("readonly", (store) =>
    store.getAll(),
  )) ?? [];

  return records.filter(
    (record) =>
      record.resourceId === resourceId && record.resourceType === resourceType,
  );
}
