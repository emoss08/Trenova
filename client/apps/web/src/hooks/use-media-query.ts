import * as React from "react";

type QueryStore = {
  list: MediaQueryList;
  listeners: Set<() => void>;
  onChange: () => void;
};

const stores = new Map<string, QueryStore>();

/**
 * One MediaQueryList and one change listener per query, shared by every component asking it: a
 * form with forty date fields asks "is this a desktop?" once, not forty times.
 */
function storeFor(query: string): QueryStore {
  let store = stores.get(query);
  if (!store) {
    const listeners = new Set<() => void>();
    store = {
      list: matchMedia(query),
      listeners,
      onChange: () => {
        for (const listener of listeners) listener();
      },
    };
    stores.set(query, store);
  }
  return store;
}

export function useMediaQuery(query: string) {
  const subscribe = React.useCallback(
    (onChange: () => void) => {
      const store = storeFor(query);
      if (store.listeners.size === 0) store.list.addEventListener("change", store.onChange);
      store.listeners.add(onChange);

      return () => {
        store.listeners.delete(onChange);
        if (store.listeners.size === 0) {
          store.list.removeEventListener("change", store.onChange);
          stores.delete(query);
        }
      };
    },
    [query],
  );
  const snapshot = React.useCallback(() => storeFor(query).list.matches, [query]);

  return React.useSyncExternalStore(subscribe, snapshot, () => false);
}
