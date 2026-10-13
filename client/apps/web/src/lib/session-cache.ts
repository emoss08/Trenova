import { buildId } from "@/lib/build-id";
import { indexedDbStore, type KeyValueStore } from "@/lib/indexed-db-store";
import { attention } from "@/lib/queries/attention";
import { homeLayout } from "@/lib/queries/home-layout";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { sidebarPreferences } from "@/lib/queries/sidebar-preferences";
import { userOrganization } from "@/lib/queries/user";
import { queryClient } from "@/lib/query-client";
import { createAsyncStoragePersister } from "@tanstack/query-async-storage-persister";
import {
  partialMatchKey,
  type Query,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import type { PersistQueryClientOptions, Persister } from "@tanstack/react-query-persist-client";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";

// What draws the app's frame and its first screen, written to the browser's disk so a
// returning session paints them before the network answers. Nothing else is ever written:
// a business record — a shipment, an invoice, a customer, anyone's personal details —
// would stay on a shared dispatch-office PC for the next person to find if nobody signed
// out. Add a query here only if a stranger at that PC could read it without harm.
const SHELL_QUERIES: readonly QueryKey[] = [
  sidebarPreferences.effective._def,
  sidebarPreferences.options._def,
  pageFavorite.all._def,
  pageFavorite.check._def,
  userOrganization.all._def,
  attention.summary._def,
  homeLayout.effective._def,
];

/** How long a persisted cache is trusted, and how long its queries are kept in memory. */
export const SHELL_CACHE_MAX_AGE_MS = 24 * 60 * 60 * 1000;
const WRITE_THROTTLE_MS = 1000;

/** Whether a query is one of the shell queries allowed onto disk. */
export function isShellQuery(query: Pick<Query, "queryKey">): boolean {
  return SHELL_QUERIES.some((key) => partialMatchKey(query.queryKey, key));
}

/**
 * Keeps the shell queries in memory for as long as they are trusted on disk. The disk holds
 * a snapshot of what is in memory, so a shell query collected after the default ten
 * minutes unused (the home layout, while the person works elsewhere) would vanish from the
 * next snapshot and not be there to restore the next morning.
 */
export function retainShellQueries(client: QueryClient): void {
  for (const key of SHELL_QUERIES) {
    client.setQueryDefaults(key, { gcTime: SHELL_CACHE_MAX_AGE_MS });
  }
}

/**
 * Whose cache this is: the signed-in person in their current organization, or null when
 * there is no one to keep it for. A cache is only read or written for the scope that is
 * signed in at that moment, so one person's frame never paints for another, nor one
 * organization's for the next.
 */
export function sessionScope(
  user: Pick<User, "id" | "currentOrganizationId"> | null,
): string | null {
  if (!user?.id || !user.currentOrganizationId) return null;
  return `${user.id}:${user.currentOrganizationId}`;
}

const NOTHING_KEPT: Persister = {
  persistClient: () => undefined,
  restoreClient: () => undefined,
  removeClient: () => undefined,
};

export type SessionPersistence = {
  /** Restore and save options for one scope. */
  persistOptions: (scope: string | null) => Omit<PersistQueryClientOptions, "queryClient">;
  /** Deletes every persisted cache except the scope's own (all of them for null). */
  forgetOtherScopes: (scope: string | null) => Promise<void>;
  /** Deletes every persisted cache. */
  forgetAll: () => Promise<void>;
};

type SessionPersistenceOptions = {
  /** False on the dev server, whose code changes under a cache without its build changing. */
  enabled: boolean;
  /** Names the running build; a cache another build wrote is discarded. */
  buster: () => string;
  /** The scope signed in right now, read at the moment of every disk read and write. */
  activeScope: () => string | null;
};

/** The query cache's persistence over a store; with no store, or disabled, nothing is kept. */
export function createSessionPersistence(
  store: KeyValueStore | null,
  { enabled, buster, activeScope }: SessionPersistenceOptions,
): SessionPersistence {
  const kept = enabled && store !== null;

  const persister = (scope: string | null): Persister => {
    if (!kept || scope === null) return NOTHING_KEPT;
    // Writes are throttled and reads are asynchronous, so either can still be on its way
    // when the session ends or changes hands; it then reads nothing and writes nothing.
    const stillSignedIn = () => activeScope() === scope;
    return createAsyncStoragePersister({
      key: scope,
      throttleTime: WRITE_THROTTLE_MS,
      storage: {
        getItem: async (key) => {
          const value = await store.get(key);
          return stillSignedIn() ? (value ?? null) : null;
        },
        setItem: async (key, value) => {
          if (stillSignedIn()) await store.set(key, value);
        },
        removeItem: (key) => store.delete(key),
      },
    });
  };

  return {
    persistOptions: (scope) => ({
      persister: persister(scope),
      buster: buster(),
      maxAge: SHELL_CACHE_MAX_AGE_MS,
      dehydrateOptions: {
        shouldDehydrateQuery: (query) => query.state.status === "success" && isShellQuery(query),
        shouldDehydrateMutation: () => false,
      },
    }),
    forgetOtherScopes: async (scope) => {
      if (!kept) return;
      const others = (await store.keys()).filter((key) => key !== scope);
      await Promise.all(others.map((key) => store.delete(key)));
    },
    forgetAll: async () => {
      if (kept) await store.clear();
    },
  };
}

export const sessionPersistence = createSessionPersistence(
  indexedDbStore("trenova-query-cache", "clients"),
  {
    enabled: import.meta.env.PROD,
    buster: () => buildId(),
    activeScope: () => sessionScope(useAuthStore.getState().user),
  },
);

/**
 * Ends what the cache holds for a session, in memory and on disk. Every way a session ends
 * or changes hands — signing out, switching organization, a new sign-in, an expired session
 * — goes through here. A failure to reach the disk is reported and not thrown: the caller
 * is ending the session regardless, and a cache left behind is only ever read for the scope
 * that wrote it.
 */
export async function clearSessionCache(): Promise<void> {
  queryClient.clear();
  try {
    await sessionPersistence.forgetAll();
  } catch (error) {
    console.error("Could not delete the persisted query cache:", error);
  }
}
