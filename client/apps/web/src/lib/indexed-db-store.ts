import { clear, createStore, del, get, keys, set, type UseStore } from "idb-keyval";

/** A string key-value store, as the persisted query cache uses one. */
export type KeyValueStore = {
  get: (key: string) => Promise<string | undefined>;
  set: (key: string, value: string) => Promise<void>;
  delete: (key: string) => Promise<void>;
  keys: () => Promise<string[]>;
  clear: () => Promise<void>;
};

/**
 * One object store in the browser's IndexedDB, opened on first use. Null where the browser
 * has no IndexedDB. Operations reject when the browser refuses storage (some private modes,
 * a full disk); callers decide what that means for them.
 */
export function indexedDbStore(database: string, objectStore: string): KeyValueStore | null {
  if (typeof indexedDB === "undefined") return null;

  let store: UseStore | null = null;
  const open = () => (store ??= createStore(database, objectStore));

  return {
    get: (key) => get<string>(key, open()),
    set: (key, value) => set(key, value, open()),
    delete: (key) => del(key, open()),
    keys: async () => (await keys<IDBValidKey>(open())).map(String),
    clear: () => clear(open()),
  };
}
