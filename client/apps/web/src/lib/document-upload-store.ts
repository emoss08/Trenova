import { activeSessionScope } from "@/lib/session-scope";
import type { DocumentUploadSession } from "@trenova/shared/types/document";

const DB_NAME = "trenova-document-uploads";
const DB_VERSION = 1;
const STORE_NAME = "sessions";

/**
 * An upload in progress, with the file it is sending, kept so a reload or a dropped
 * connection resumes it from the parts the server already has instead of starting over.
 * The file is a business document, so a record belongs to the session that started it
 * (`scope`, see lib/session-scope), is deleted when the upload finishes, fails or is
 * cancelled, and every record goes when a session ends (clearSessionCache).
 */
export type PersistedUploadRecord = {
  sessionId: string;
  scope: string;
  resourceId: string;
  resourceType: string;
  session: DocumentUploadSession;
  file: File;
};

/** Where the records live: one object store keyed by upload session. */
export type UploadRecordBackend = {
  put: (record: PersistedUploadRecord) => Promise<void>;
  delete: (sessionId: string) => Promise<void>;
  all: () => Promise<PersistedUploadRecord[]>;
  clear: () => Promise<void>;
};

export type DocumentUploadStore = {
  /** Saves a record, unless the session that started the upload is no longer signed in. */
  persist: (record: PersistedUploadRecord) => Promise<void>;
  remove: (sessionId: string) => Promise<void>;
  /**
   * The scope's unexpired uploads to one resource. Records for any other scope, and
   * uploads the server has already expired, can never resume here, so they are deleted.
   */
  list: (
    scope: string,
    resourceType: string,
    resourceId: string,
  ) => Promise<PersistedUploadRecord[]>;
  /** Deletes every record, whoever's it is. */
  clear: () => Promise<void>;
};

type DocumentUploadStoreOptions = {
  /** The scope signed in right now, read at the moment of every write. */
  activeScope: () => string | null;
  /** Milliseconds since the epoch. */
  now: () => number;
};

const NOTHING: PersistedUploadRecord[] = [];

/** The upload records over a backend; with no backend, nothing is kept. */
export function createDocumentUploadStore(
  backend: UploadRecordBackend | null,
  { activeScope, now }: DocumentUploadStoreOptions,
): DocumentUploadStore {
  if (backend === null) {
    return {
      persist: async () => undefined,
      remove: async () => undefined,
      list: async () => NOTHING,
      clear: async () => undefined,
    };
  }

  return {
    persist: async (record) => {
      // An upload still running when its session ends or changes hands would otherwise
      // write its file back after the session's records were cleared.
      if (activeScope() !== record.scope) return;
      await backend.put(record);
    },
    remove: (sessionId) => backend.delete(sessionId),
    list: async (scope, resourceType, resourceId) => {
      const records = await backend.all();
      const nowSeconds = now() / 1000;
      const unusable = records.filter(
        (record) => record.scope !== scope || record.session.expiresAt <= nowSeconds,
      );
      await Promise.all(unusable.map((record) => backend.delete(record.sessionId)));

      return records.filter(
        (record) =>
          !unusable.includes(record) &&
          record.resourceType === resourceType &&
          record.resourceId === resourceId,
      );
    },
    clear: () => backend.clear(),
  };
}

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
    request.onerror = () => reject(request.error ?? new Error("Failed to open upload persistence"));
  });
}

async function withStore<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T> | undefined,
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

function indexedDbUploadBackend(): UploadRecordBackend | null {
  if (typeof indexedDB === "undefined") return null;

  return {
    put: async (record) => void (await withStore("readwrite", (store) => store.put(record))),
    delete: async (sessionId) =>
      void (await withStore("readwrite", (store) => store.delete(sessionId))),
    all: async () =>
      (await withStore<PersistedUploadRecord[]>("readonly", (store) => store.getAll())) ?? [],
    clear: async () => void (await withStore("readwrite", (store) => store.clear())),
  };
}

const documentUploadStore = createDocumentUploadStore(indexedDbUploadBackend(), {
  activeScope: activeSessionScope,
  now: Date.now,
});

export const persistDocumentUploadSession = documentUploadStore.persist;
export const removeDocumentUploadSession = documentUploadStore.remove;
export const listPersistedDocumentUploadSessions = documentUploadStore.list;
export const clearDocumentUploadSessions = documentUploadStore.clear;
