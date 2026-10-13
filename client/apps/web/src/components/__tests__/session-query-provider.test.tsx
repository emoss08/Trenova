import type { KeyValueStore } from "@/lib/indexed-db-store";
import { queries } from "@/lib/queries";
import { createSessionPersistence } from "@/lib/session-cache";
import { QueryClient, useIsRestoring } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SessionQueryProvider } from "../session-query-provider";

const SIDEBAR_KEY = queries.sidebarPreferences.effective().queryKey;

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

function RestoringState() {
  return <span>{useIsRestoring() ? "restoring" : "ready"}</span>;
}

function setup() {
  const store = memoryStore();
  let signedIn: string | null = null;
  const persistence = createSessionPersistence(store, {
    enabled: true,
    buster: () => "build-a",
    activeScope: () => signedIn,
  });
  const client = new QueryClient();
  const view = (scope: string | null) => {
    signedIn = scope;
    return (
      <SessionQueryProvider client={client} scope={scope} persistence={persistence}>
        <RestoringState />
      </SessionQueryProvider>
    );
  };
  return { store, client, view };
}

// Saves are throttled to one a second, so a test waits past that before reading the disk.
const THROTTLED = { timeout: 3000 };

const written = (store: ReturnType<typeof memoryStore>, scope: string) =>
  store.entries.get(scope) ?? "";

describe("SessionQueryProvider", () => {
  it("holds queries until the scope's cache is restored, then lets them run", async () => {
    const { store, client, view } = setup();
    store.entries.set(
      "usr_1:org_1",
      JSON.stringify({
        timestamp: Date.now(),
        buster: "build-a",
        clientState: {
          mutations: [],
          queries: [
            {
              queryKey: SIDEBAR_KEY,
              queryHash: JSON.stringify(SIDEBAR_KEY),
              state: {
                data: { sections: ["billing"] },
                dataUpdatedAt: Date.now(),
                status: "success",
              },
            },
          ],
        },
      }),
    );

    render(view("usr_1:org_1"));

    expect(screen.getByText("restoring")).toBeInTheDocument();
    await screen.findByText("ready");
    expect(client.getQueryData(SIDEBAR_KEY)).toEqual({ sections: ["billing"] });
  });

  it("saves into the new person's cache after a sign-out and another sign-in in the same tab", async () => {
    const { store, client, view } = setup();
    const { rerender } = render(view("usr_1:org_1"));
    await screen.findByText("ready");

    client.setQueryData(SIDEBAR_KEY, { sections: ["first person's"] });
    await waitFor(
      () => expect(written(store, "usr_1:org_1")).toContain("first person's"),
      THROTTLED,
    );

    rerender(view(null));
    client.clear();
    rerender(view("usr_2:org_1"));
    await screen.findByText("ready");

    client.setQueryData(SIDEBAR_KEY, { sections: ["second person's"] });
    await waitFor(
      () => expect(written(store, "usr_2:org_1")).toContain("second person's"),
      THROTTLED,
    );
    await new Promise((resolve) => setTimeout(resolve, 1100));

    expect(written(store, "usr_1:org_1")).not.toContain("second person's");
    expect(store.entries.has("usr_1:org_1")).toBe(false);
  });

  it("saves under the new organization after a switch, and forgets the previous one", async () => {
    const { store, client, view } = setup();
    const { rerender } = render(view("usr_1:org_1"));
    await screen.findByText("ready");
    client.setQueryData(SIDEBAR_KEY, { sections: ["org one"] });
    await waitFor(() => expect(written(store, "usr_1:org_1")).toContain("org one"), THROTTLED);

    rerender(view("usr_1:org_2"));
    await screen.findByText("ready");
    client.setQueryData(SIDEBAR_KEY, { sections: ["org two"] });

    await waitFor(() => expect(written(store, "usr_1:org_2")).toContain("org two"), THROTTLED);
    await waitFor(() => expect(store.entries.has("usr_1:org_1")).toBe(false), THROTTLED);
  });
});
