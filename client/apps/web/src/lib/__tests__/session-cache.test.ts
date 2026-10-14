import type { KeyValueStore } from "@/lib/indexed-db-store";
import { attention } from "@/lib/queries/attention";
import { homeLayout } from "@/lib/queries/home-layout";
import { organization } from "@/lib/queries/organization";
import { pageFavorite } from "@/lib/queries/page-favorite";
import { sidebarPreferences } from "@/lib/queries/sidebar-preferences";
import { QueryClient } from "@tanstack/react-query";
import {
  persistQueryClientRestore,
  persistQueryClientSave,
} from "@tanstack/react-query-persist-client";
import { describe, expect, it, vi } from "vitest";
import {
  createSessionPersistence,
  isShellQuery,
  retainShellQueries,
  SHELL_CACHE_MAX_AGE_MS,
  sessionScope,
} from "../session-cache";

function memoryStore(): KeyValueStore & { entries: Map<string, string> } {
  const entries = new Map<string, string>();
  return {
    entries,
    get: async (key: string) => entries.get(key),
    set: async (key: string, value: string) => void entries.set(key, value),
    delete: async (key: string) => void entries.delete(key),
    keys: async () => [...entries.keys()],
    clear: async () => entries.clear(),
  };
}

const SIDEBAR_KEY = sidebarPreferences.effective().queryKey;
const ATTENTION_KEY = attention.summary().queryKey;
const RECORD_KEY = ["shipment", "detail", "shp_1"];
const LOGO_KEY = organization.logo("org_1").queryKey;

function clientWithData(): QueryClient {
  const client = new QueryClient();
  client.setQueryData(SIDEBAR_KEY, { sections: ["dispatch"] });
  client.setQueryData(ATTENTION_KEY, { agentDecisions: 3 });
  client.setQueryData(RECORD_KEY, { proNumber: "S-1001", consignee: "Jane Doe" });
  client.setQueryData(LOGO_KEY, "https://storage.example.com/logo.png?signature=abc");
  return client;
}

function persistenceFor(
  store: KeyValueStore,
  { signedIn, buster = "build-a" }: { signedIn: () => string | null; buster?: string },
) {
  return createSessionPersistence(store, {
    enabled: true,
    buster: () => buster,
    activeScope: signedIn,
  });
}

async function save(store: ReturnType<typeof memoryStore>, scope: string, buster = "build-a") {
  const persistence = persistenceFor(store, { signedIn: () => scope, buster });
  await persistQueryClientSave({
    queryClient: clientWithData(),
    ...persistence.persistOptions(scope),
  });
  await vi.waitFor(() => expect(store.entries.has(scope)).toBe(true));
}

async function restore(
  store: KeyValueStore,
  scope: string | null,
  {
    buster = "build-a",
    signedIn = () => scope,
  }: { buster?: string; signedIn?: () => string | null } = {},
) {
  const persistence = persistenceFor(store, { signedIn, buster });
  const client = new QueryClient();
  await persistQueryClientRestore({ queryClient: client, ...persistence.persistOptions(scope) });
  return client;
}

describe("isShellQuery", () => {
  it("allows the queries that draw the frame and first screen, whatever their arguments", () => {
    expect(isShellQuery({ queryKey: SIDEBAR_KEY })).toBe(true);
    expect(isShellQuery({ queryKey: ATTENTION_KEY })).toBe(true);
    expect(isShellQuery({ queryKey: pageFavorite.check("/shipments").queryKey })).toBe(true);
    expect(isShellQuery({ queryKey: homeLayout.effective().queryKey })).toBe(true);
  });

  it("refuses records, and the logo's expiring storage URL", () => {
    expect(isShellQuery({ queryKey: RECORD_KEY })).toBe(false);
    expect(isShellQuery({ queryKey: LOGO_KEY })).toBe(false);
    expect(isShellQuery({ queryKey: ["attention"] })).toBe(false);
  });
});

describe("retainShellQueries", () => {
  it("keeps the shell queries in memory as long as their copy on disk is trusted", () => {
    const client = new QueryClient();
    retainShellQueries(client);

    expect(client.getQueryDefaults(SIDEBAR_KEY).gcTime).toBe(SHELL_CACHE_MAX_AGE_MS);
    expect(client.getQueryDefaults(homeLayout.effective().queryKey).gcTime).toBe(
      SHELL_CACHE_MAX_AGE_MS,
    );
    expect(client.getQueryDefaults(RECORD_KEY).gcTime).toBeUndefined();
  });
});

describe("sessionScope", () => {
  it("names the person in their current organization", () => {
    expect(sessionScope({ id: "usr_1", currentOrganizationId: "org_1" })).toBe("usr_1:org_1");
  });

  it("keeps nothing without both a person and an organization", () => {
    expect(sessionScope(null)).toBeNull();
    expect(sessionScope({ id: "usr_1", currentOrganizationId: undefined })).toBeNull();
    expect(sessionScope({ id: undefined, currentOrganizationId: "org_1" })).toBeNull();
  });
});

describe("session persistence", () => {
  it("writes only shell queries to disk and restores them for the same person and organization", async () => {
    const store = memoryStore();
    await save(store, "usr_1:org_1");

    const written = store.entries.get("usr_1:org_1") ?? "";
    expect(written).toContain("dispatch");
    expect(written).not.toContain("S-1001");
    expect(written).not.toContain("Jane Doe");
    expect(written).not.toContain("signature=abc");

    const client = await restore(store, "usr_1:org_1");
    expect(client.getQueryData(SIDEBAR_KEY)).toEqual({ sections: ["dispatch"] });
    expect(client.getQueryData(ATTENTION_KEY)).toEqual({ agentDecisions: 3 });
    expect(client.getQueryData(RECORD_KEY)).toBeUndefined();
  });

  it("never restores one scope's cache for another person or organization", async () => {
    const store = memoryStore();
    await save(store, "usr_1:org_1");

    expect((await restore(store, "usr_2:org_1")).getQueryData(SIDEBAR_KEY)).toBeUndefined();
    expect((await restore(store, "usr_1:org_2")).getQueryData(SIDEBAR_KEY)).toBeUndefined();
    expect((await restore(store, null)).getQueryData(SIDEBAR_KEY)).toBeUndefined();
  });

  it("restores nothing when the session changed hands while the cache was being read", async () => {
    const store = memoryStore();
    await save(store, "usr_1:org_1");

    const client = await restore(store, "usr_1:org_1", { signedIn: () => "usr_2:org_1" });
    expect(client.getQueryData(SIDEBAR_KEY)).toBeUndefined();
  });

  it("drops a write still on its way when the session has ended", async () => {
    const store = memoryStore();
    let signedIn: string | null = "usr_1:org_1";
    const persistence = persistenceFor(store, { signedIn: () => signedIn });
    const options = persistence.persistOptions("usr_1:org_1");

    signedIn = null;
    await persistQueryClientSave({ queryClient: clientWithData(), ...options });
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(store.entries.size).toBe(0);
  });

  it("discards a cache another build wrote", async () => {
    const store = memoryStore();
    await save(store, "usr_1:org_1", "build-a");

    const client = await restore(store, "usr_1:org_1", { buster: "build-b" });
    expect(client.getQueryData(SIDEBAR_KEY)).toBeUndefined();
    expect(store.entries.has("usr_1:org_1")).toBe(false);
  });

  it("discards a cache older than a day", async () => {
    const store = memoryStore();
    await save(store, "usr_1:org_1");
    const persisted = JSON.parse(store.entries.get("usr_1:org_1") ?? "{}");
    persisted.timestamp -= SHELL_CACHE_MAX_AGE_MS + 1;
    store.entries.set("usr_1:org_1", JSON.stringify(persisted));

    const client = await restore(store, "usr_1:org_1");
    expect(client.getQueryData(SIDEBAR_KEY)).toBeUndefined();
  });

  it("forgets every other scope's cache, or all of them when no one is signed in", async () => {
    const store = memoryStore();
    store.entries.set("usr_1:org_1", "{}");
    store.entries.set("usr_1:org_2", "{}");
    store.entries.set("usr_2:org_1", "{}");
    const persistence = persistenceFor(store, { signedIn: () => "usr_1:org_2" });

    await persistence.forgetOtherScopes("usr_1:org_2");
    expect([...store.entries.keys()]).toEqual(["usr_1:org_2"]);

    await persistence.forgetOtherScopes(null);
    expect(store.entries.size).toBe(0);
  });

  it("forgets everything at once", async () => {
    const store = memoryStore();
    store.entries.set("usr_1:org_1", "{}");
    await persistenceFor(store, { signedIn: () => null }).forgetAll();
    expect(store.entries.size).toBe(0);
  });

  it("keeps and touches nothing when disabled, as on the dev server", async () => {
    const store = memoryStore();
    store.entries.set("usr_1:org_1", "{}");
    const persistence = createSessionPersistence(store, {
      enabled: false,
      buster: () => "b",
      activeScope: () => "usr_9:org_9",
    });

    await persistQueryClientSave({
      queryClient: clientWithData(),
      ...persistence.persistOptions("usr_9:org_9"),
    });
    await persistence.forgetAll();

    expect([...store.entries.keys()]).toEqual(["usr_1:org_1"]);
  });
});
