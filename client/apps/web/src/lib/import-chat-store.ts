const DB_NAME = "trenova-import-chat";
const DB_VERSION = 1;
const STORE_NAME = "conversations";

export type PersistedChatMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
  toolCalls?: Array<{
    name: string;
    callId: string;
    status: "running" | "completed" | "error";
    result?: string;
  }>;
  suggestions?: Array<{
    label: string;
    prompt: string;
    type?: "prompt" | "input" | "action" | "date";
    placeholder?: string;
    action?: string;
    submitLabel?: string;
  }>;
};

type PersistedConversation = {
  documentId: string;
  conversationId?: string;
  messages: PersistedChatMessage[];
  updatedAt: number;
};

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "documentId" });
      }
    };

    request.onsuccess = () => resolve(request.result);
    request.onerror = () =>
      reject(request.error ?? new Error("Failed to open chat persistence"));
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
    tx.onerror = () =>
      reject(tx.error ?? new Error("IndexedDB transaction failed"));
    tx.onabort = () =>
      reject(tx.error ?? new Error("IndexedDB transaction aborted"));
  }).finally(() => db.close());
}

export async function saveConversation(
  documentId: string,
  conversationId: string | undefined,
  messages: PersistedChatMessage[],
): Promise<void> {
  const record: PersistedConversation = {
    documentId,
    conversationId,
    messages,
    updatedAt: Date.now(),
  };
  await withStore("readwrite", (store) => store.put(record));
}

export async function loadConversation(
  documentId: string,
): Promise<{ conversationId?: string; messages: PersistedChatMessage[] } | null> {
  const result = await withStore<PersistedConversation>("readonly", (store) =>
    store.get(documentId),
  );
  if (!result) return null;

  // Expire conversations older than 1 hour
  if (Date.now() - result.updatedAt > 60 * 60 * 1000) {
    await clearConversation(documentId);
    return null;
  }

  return { conversationId: result.conversationId, messages: result.messages };
}

export async function clearConversation(documentId: string): Promise<void> {
  await withStore("readwrite", (store) => store.delete(documentId));
}
