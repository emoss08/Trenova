import type { WatchtowerAgent } from "@/lib/graphql/watchtower";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WatchtowerFeed } from "../watchtower-feed";
import { watchtowerItem } from "./item-fixture";

const NOW = Math.floor(Date.now() / 1000);

const api = vi.hoisted(() => ({
  fetchWatchtowerFeed: vi.fn(),
  fetchWatchtowerCounts: vi.fn(),
  fetchWatchtowerCleared: vi.fn(),
  fetchWatchtowerHandOffAgents: vi.fn(),
  fetchWatchtowerItem: vi.fn(),
  markWatchtowerItemSeen: vi.fn(),
  resolveWatchtowerItem: vi.fn(),
  snoozeWatchtowerItem: vi.fn(),
  handOffWatchtowerItem: vi.fn(),
}));
/** Every toast the screen raised, of any kind, in order. */
const toasts = vi.hoisted(() => ({ shown: [] as { kind: string; title: unknown }[] }));

vi.mock("@/lib/graphql/watchtower", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/watchtower")>()),
  ...api,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
    threads: () => ({ queryKey: ["threads"] }),
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => {
  const raise = (kind: string) => (title: unknown) => {
    toasts.shown.push({ kind, title });
    return kind;
  };
  const toast = Object.assign(raise("message"), {
    success: raise("success"),
    error: raise("error"),
    warning: raise("warning"),
    info: raise("info"),
    dismiss: vi.fn(),
  });
  return { toast };
});

/** A refused write as the GraphQL error presenter sends it: a validation problem on `version`. */
function versionMismatch() {
  const fieldErrors = [
    {
      field: "version",
      code: "VERSION_MISMATCH",
      message: "This exception changed since you opened it; refresh and try again",
    },
  ];
  return new GraphQLRequestError({
    kind: "graphql",
    message: "This exception changed since you opened it; refresh and try again",
    status: 200,
    graphQLErrors: [
      {
        message: "This exception changed since you opened it; refresh and try again",
        code: "VALIDATION_FAILED",
        type: "https://trenova.app/problems/validation-error",
        errors: fieldErrors,
        extensions: {
          code: "VALIDATION_FAILED",
          type: "https://trenova.app/problems/validation-error",
          errors: fieldErrors,
        },
      },
    ],
  });
}

function queue(items: ReturnType<typeof watchtowerItem>[]) {
  api.fetchWatchtowerFeed.mockResolvedValue({
    items,
    endCursor: null,
    hasNextPage: false,
    seenAt: 0,
  });
  api.fetchWatchtowerCounts.mockResolvedValue({ active: items.length, activeCritical: 0 });
}

function renderFeed() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/desk/watchtower"]}>
        <WatchtowerFeed />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const late = watchtowerItem({
  id: "wt_late",
  title: "SEED-SHP-007 will miss its window",
  dueAt: NOW + 190 * 60,
  occurredAt: NOW - 240,
});
const lapse = watchtowerItem({
  id: "wt_lapse",
  title: "Lakeshore Logistics' cargo insurance lapses",
  severity: "Critical",
  sourceKind: "CarrierIntelEvent",
  dueAt: NOW + 9 * 86_400,
  dueLabel: "Insurance lapses",
  seen: true,
  agentWork: {
    agentId: "adef_risk",
    runId: "ar_risk",
    status: "Working",
    step: 1,
    steps: ["Read", "Decide", "Approve"],
    note: "",
    failed: false,
    handedAt: NOW - 60,
    agent: {
      id: "adef_risk",
      name: "Carrier risk desk",
      description: "Watches carrier insurance and authority.",
      icon: "shield",
      accent: "rose",
      template: "CarrierRiskDesk",
    },
  },
});
const fyi = watchtowerItem({
  id: "wt_fyi",
  title: "Acme sent 6 status updates we didn't expect",
  severity: "Info",
  dueAt: null,
  dueLabel: null,
  seen: true,
});

const monitor: WatchtowerAgent = {
  id: "adef_monitor",
  name: "Load monitor",
  description: "Watches loads in transit.",
  icon: "radar",
  accent: "sky",
  template: "LoadMonitor",
};
const carrierDesk: WatchtowerAgent = {
  id: "adef_carrier",
  name: "Carrier risk desk",
  description: "Checks carriers before they haul.",
  icon: "shield",
  accent: "violet",
  template: "CarrierRiskDesk",
};

/** What the server answers a hand-off with: the item, now the agent's. */
const handedOff = (item: ReturnType<typeof watchtowerItem>) => ({
  item: { ...item, seen: true },
  runId: "ar_1",
  subscribers: [],
  candidates: [],
  templates: [],
});

describe("WatchtowerFeed", () => {
  let restoreLayout: () => void;

  beforeEach(() => {
    restoreLayout = stubLayout(36, 280);
    toasts.shown = [];
    Object.values(api).forEach((mock) => mock.mockReset());
    api.fetchWatchtowerCleared.mockResolvedValue(2);
    api.fetchWatchtowerHandOffAgents.mockResolvedValue([]);
    api.fetchWatchtowerItem.mockRejectedValue(new Error("offline"));
    api.markWatchtowerItemSeen.mockImplementation((id: string) =>
      Promise.resolve(watchtowerItem({ id, seen: true })),
    );
  });

  afterEach(() => restoreLayout());

  it("reads the queue in deadline lanes with the agents already on something above it", async () => {
    queue([fyi, lapse, late]);
    renderFeed();

    expect(await screen.findByText("Act now")).toBeDefined();
    expect(screen.getByText("Coming up")).toBeDefined();
    expect(screen.getByText("For your information")).toBeDefined();
    expect(screen.queryByText("Today")).toBeNull();
    expect(screen.getByText("Carrier risk desk is on it")).toBeDefined();
    expect(screen.getByText("On it")).toBeDefined();
    expect(screen.getByText("1 working")).toBeDefined();
    expect(screen.getByText("2 of 5 cleared")).toBeDefined();
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(late.title);
  });

  it("marks the item it opens seen", async () => {
    queue([late]);
    renderFeed();

    await screen.findByRole("heading", { level: 1 });
    await waitFor(() => expect(api.markWatchtowerItemSeen).toHaveBeenCalledWith("wt_late"));
  });

  it("finishes an item with E and moves on to the next one without a toast", async () => {
    queue([late, fyi]);
    api.resolveWatchtowerItem.mockResolvedValue({
      ...late,
      status: "Resolved" as const,
      resolvedAt: NOW,
      version: 4,
    });
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.keyDown(document, { key: "e" });

    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(fyi.title),
    );
    await waitFor(() =>
      expect(api.resolveWatchtowerItem).toHaveBeenCalledWith(
        { id: "wt_late", version: 3, code: "Fixed" },
        { idempotencyKey: expect.any(String) },
      ),
    );
    expect(toasts.shown).toEqual([]);
  });

  it("snoozes an item with S until tomorrow morning and moves on without a toast", async () => {
    queue([late, fyi]);
    api.snoozeWatchtowerItem.mockResolvedValue({ ...late, status: "Snoozed" as const, version: 4 });
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.keyDown(document, { key: "s" });

    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(fyi.title),
    );
    await waitFor(() =>
      expect(api.snoozeWatchtowerItem).toHaveBeenCalledWith(
        { id: "wt_late", version: 3, until: expect.any(Number) },
        { idempotencyKey: expect.any(String) },
      ),
    );
    const [{ until }] = api.snoozeWatchtowerItem.mock.calls[0] as [{ until: number }];
    expect(until).toBeGreaterThan(NOW);
    expect(toasts.shown).toEqual([]);
  });

  it("puts an item back when finishing it fails", async () => {
    queue([late, fyi]);
    api.resolveWatchtowerItem.mockRejectedValue(new Error("offline"));
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.click(screen.getByRole("button", { name: /Done/ }));

    await waitFor(() => expect(api.resolveWatchtowerItem).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(late.title),
    );
    expect(toasts.shown.map((shown) => shown.kind)).toEqual(["error"]);
  });

  it("walks the queue with J and K and ignores keys typed into a field", async () => {
    queue([late, fyi]);
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.keyDown(document, { key: "j" });
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(fyi.title);
    fireEvent.keyDown(document, { key: "k" });
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(late.title);

    const field = document.createElement("input");
    document.body.appendChild(field);
    fireEvent.keyDown(field, { key: "j" });
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(late.title);
    field.remove();
  });

  it("shows the agent working an item and its step", async () => {
    queue([lapse]);
    renderFeed();

    const block = await screen.findByRole("region", { name: "Agent on it" });
    expect(within(block).getByText("Step 2 of 3 · Work out what to do")).toBeDefined();
    expect(screen.queryByRole("button", { name: /Hand off/ })).toBeNull();
  });

  it("renders what a finished run said as markdown, the first lines first and the rest on Show more", async () => {
    const said = [
      "Both moves on PRO S26 are uncovered (ID shp_01HZY8Q6N8W4C3K2E5R7T9V1XB).",
      "",
      "**What I found**",
      "- **Move 1:** Los Angeles Terminal pickup, then Dallas Warehouse delivery.",
      "- **Move 2:** Dallas Warehouse pickup, then Chicago Distribution Center delivery.",
    ].join("\n");
    queue([
      {
        ...lapse,
        agentWork: { ...lapse.agentWork!, status: "Done", step: 3, note: said },
      },
    ]);
    const proto = HTMLElement.prototype;
    const scroll = Object.getOwnPropertyDescriptor(proto, "scrollHeight");
    const client = Object.getOwnPropertyDescriptor(proto, "clientHeight");
    Object.defineProperty(proto, "scrollHeight", { configurable: true, get: () => 120 });
    Object.defineProperty(proto, "clientHeight", { configurable: true, get: () => 36 });
    try {
      renderFeed();

      const block = await screen.findByRole("region", { name: "Agent on it" });
      expect(within(block).getByText("Done")).toBeDefined();
      expect(within(block).getByText("What I found").tagName).toBe("STRONG");
      expect(
        within(block)
          .getAllByRole("listitem")
          .map((item) => item.textContent),
      ).toEqual([
        "Move 1: Los Angeles Terminal pickup, then Dallas Warehouse delivery.",
        "Move 2: Dallas Warehouse pickup, then Chicago Distribution Center delivery.",
      ]);
      expect(within(block).getByText("Both moves on PRO S26 are uncovered.")).toBeDefined();
      expect(block.textContent).not.toContain("**");
      const body = block.querySelector(".dk-w3-agn-body");
      expect(body?.hasAttribute("data-clamped")).toBe(true);

      fireEvent.click(within(block).getByRole("button", { name: "Show more" }));

      expect(body?.hasAttribute("data-clamped")).toBe(false);
      const less = within(block).getByRole("button", { name: "Show less" });
      expect(less.getAttribute("aria-expanded")).toBe("true");
    } finally {
      if (scroll) Object.defineProperty(proto, "scrollHeight", scroll);
      else delete (proto as { scrollHeight?: number }).scrollHeight;
      if (client) Object.defineProperty(proto, "clientHeight", client);
      else delete (proto as { clientHeight?: number }).clientHeight;
    }
  });

  it("offers no Show more when the whole note fits", async () => {
    queue([
      {
        ...lapse,
        agentWork: { ...lapse.agentWork!, status: "Done", step: 3, note: "Renewal on file." },
      },
    ]);
    renderFeed();

    const block = await screen.findByRole("region", { name: "Agent on it" });
    expect(within(block).getByText("Renewal on file.")).toBeDefined();
    expect(within(block).queryByRole("button", { name: "Show more" })).toBeNull();
  });

  it("draws one Hand off button that opens the list when no agent is suggested", async () => {
    queue([late]);
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    expect(screen.queryByRole("button", { name: "Choose who to hand it to" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /^Hand off/ }));

    expect(
      await screen.findByText("No agent can take this yet. Start one from AI Control."),
    ).toBeDefined();
    expect(api.handOffWatchtowerItem).not.toHaveBeenCalled();
  });

  it("hands the item to the agent picked from the list", async () => {
    queue([late]);
    api.fetchWatchtowerHandOffAgents.mockResolvedValue([monitor]);
    api.handOffWatchtowerItem.mockResolvedValue(handedOff(late));
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.click(screen.getByRole("button", { name: /^Hand off/ }));
    fireEvent.click(await screen.findByRole("menuitem", { name: /Load monitor/ }));

    await waitFor(() =>
      expect(api.handOffWatchtowerItem).toHaveBeenCalledWith("wt_late", "adef_monitor", {
        idempotencyKey: expect.any(String),
      }),
    );
  });

  it("hands straight to the suggested agent, and lists only the others under the caret", async () => {
    const suggestedItem = {
      ...late,
      suggestedAgentId: carrierDesk.id,
      suggestedAgent: carrierDesk,
    };
    queue([suggestedItem]);
    api.fetchWatchtowerHandOffAgents.mockResolvedValue([carrierDesk, monitor]);
    api.handOffWatchtowerItem.mockResolvedValue(handedOff(suggestedItem));
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.click(screen.getByRole("button", { name: "Choose who to hand it to" }));
    expect(await screen.findByRole("menuitem", { name: /Load monitor/ })).toBeDefined();
    expect(screen.queryByRole("menuitem", { name: /Carrier risk desk/ })).toBeNull();
    fireEvent.keyDown(document.activeElement ?? document.body, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("menuitem")).toBeNull());

    fireEvent.click(screen.getByRole("button", { name: /^Hand off/ }));

    await waitFor(() =>
      expect(api.handOffWatchtowerItem).toHaveBeenCalledWith("wt_late", "adef_carrier", {
        idempotencyKey: expect.any(String),
      }),
    );
    expect(screen.queryByRole("menuitem")).toBeNull();
  });

  it("puts back the item as the server has it, without a toast, when it changed underneath", async () => {
    queue([late, fyi]);
    api.resolveWatchtowerItem.mockRejectedValue(versionMismatch());
    api.fetchWatchtowerItem.mockResolvedValue({
      ...late,
      version: 9,
      title: "SEED-SHP-007 now five hours late",
    });
    renderFeed();
    await screen.findByRole("heading", { level: 1 });

    fireEvent.keyDown(document, { key: "e" });

    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 1 }).textContent).toBe(
        "SEED-SHP-007 now five hours late",
      ),
    );
    expect(toasts.shown).toEqual([]);

    api.resolveWatchtowerItem.mockResolvedValue({ ...late, status: "Resolved", version: 10 });
    fireEvent.keyDown(document, { key: "e" });
    await waitFor(() =>
      expect(api.resolveWatchtowerItem).toHaveBeenLastCalledWith(
        { id: "wt_late", version: 9, code: "Fixed" },
        { idempotencyKey: expect.any(String) },
      ),
    );
  });

  it("says all clear once nothing is left", async () => {
    queue([]);
    renderFeed();

    expect(
      await screen.findByText(
        "You cleared 2 today. New signals show up here the moment they happen.",
      ),
    ).toBeDefined();
  });
});
