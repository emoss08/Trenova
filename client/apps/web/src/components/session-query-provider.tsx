import type { SessionPersistence } from "@/lib/session-cache";
import { IsRestoringProvider, QueryClientProvider, type QueryClient } from "@tanstack/react-query";
import {
  persistQueryClientRestore,
  persistQueryClientSubscribe,
} from "@tanstack/react-query-persist-client";
import { useEffect, useState, type ReactNode } from "react";

type SessionQueryProviderProps = {
  client: QueryClient;
  /** The signed-in person in their organization (lib/session-cache sessionScope). */
  scope: string | null;
  persistence: SessionPersistence;
  children: ReactNode;
};

/**
 * The query cache with the shell's queries kept on disk for the session's scope. Whenever the
 * scope changes — sign-in, sign-out, an organization switch — the previous scope's saving
 * stops and the new scope's cache is restored and saved from then on; queries wait for that
 * restore, so none is fetched only to be overwritten by it.
 *
 * TanStack's PersistQueryClientProvider is not used because it restores and subscribes once
 * for the life of the page: after a sign-out and another person's sign-in in the same tab it
 * would go on saving into the first person's cache.
 */
export function SessionQueryProvider({
  client,
  scope,
  persistence,
  children,
}: SessionQueryProviderProps) {
  // undefined until the first restore finishes; then the scope whose cache was restored.
  const [restoredScope, setRestoredScope] = useState<string | null | undefined>(undefined);

  useEffect(() => {
    let stopped = false;
    let unsubscribe: (() => void) | undefined;
    const options = { queryClient: client, ...persistence.persistOptions(scope) };

    persistQueryClientRestore(options)
      .catch((error: unknown) =>
        console.error("Could not restore the persisted query cache:", error),
      )
      .finally(() => {
        if (stopped) return;
        unsubscribe = persistQueryClientSubscribe(options);
        setRestoredScope(scope);
      });

    // No scope is not a session ending: a page load knows nobody until the server answers
    // the session check, and a session that ends clears every cache itself
    // (clearSessionCache). Forgetting here would delete the cache the load is about to
    // restore.
    if (scope !== null) {
      persistence.forgetOtherScopes(scope).catch((error: unknown) => {
        console.error("Could not delete another session's persisted query cache:", error);
      });
    }

    return () => {
      stopped = true;
      unsubscribe?.();
    };
  }, [client, persistence, scope]);

  return (
    <QueryClientProvider client={client}>
      <IsRestoringProvider value={restoredScope !== scope}>{children}</IsRestoringProvider>
    </QueryClientProvider>
  );
}
