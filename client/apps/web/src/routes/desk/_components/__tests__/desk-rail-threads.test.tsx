import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { ThreadListFilter } from "@/services/assistant";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { stubLayout } from "@/test/layout";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskRailThreads, type DeskRailThreadsProps } from "../desk-rail-threads";

const DAY = 24 * 60 * 60;

const api = vi.hoisted(() => ({
  listThreads:
    vi.fn<(options: { filter?: ThreadListFilter; cursor?: string }) => Promise<unknown>>(),
  countThreadShelves: vi.fn<(edges: readonly number[]) => Promise<unknown>>(),
}));

vi.mock("@/services/api", () => ({ apiService: { assistantService: api } }));

vi.mock("@/components/assistant/use-active-turns", () => ({
  useLiveThreadIds: () => new Set<string>(),
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({ user: { id: "usr_1", name: "Avery Lane", username: "avery", timezone: "UTC" } }),
}));

const now = Math.floor(Date.now() / 1000);

function thread(id: string, overrides: Partial<AssistantThread> = {}): AssistantThread {
  const lastMessageAt = overrides.lastMessageAt ?? now - 60;
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agtd_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: id,
    status: "Active",
    lastMessageAt,
    version: 0,
    createdAt: Math.min(now - 120, lastMessageAt),
    updatedAt: lastMessageAt,
    ...overrides,
  };
}

/** What the server holds on each shelf, keyed as the rail reads it. */
type Shelves = {
  pinned?: AssistantThread[];
  dated?: AssistantThread[];
  older?: AssistantThread[];
};

function serve(shelves: Shelves) {
  api.listThreads.mockImplementation(({ filter }) => {
    let items: AssistantThread[] = [];
    if (filter?.pinned) {
      items = shelves.pinned ?? [];
    } else if (filter?.touchedFrom === undefined && filter?.touchedBefore !== undefined) {
      items = shelves.older ?? [];
    } else if (filter?.touchedFrom !== undefined) {
      const from = filter.touchedFrom;
      const before = filter.touchedBefore ?? Number.POSITIVE_INFINITY;
      items = (shelves.dated ?? []).filter(
        (item) => item.lastMessageAt >= from && item.lastMessageAt < before,
      );
    }
    return Promise.resolve({ items, nextCursor: "" });
  });
  api.countThreadShelves.mockImplementation((edges) => {
    const dated = shelves.dated ?? [];
    const bands = edges.map((edge, index) =>
      dated.filter(
        (item) =>
          item.lastMessageAt >= edge && (index === 0 || item.lastMessageAt < edges[index - 1]),
      ),
    );
    return Promise.resolve({
      pinned: shelves.pinned?.length ?? 0,
      bands: [...bands.map((band) => band.length), shelves.older?.length ?? 0],
    });
  });
}

const askedFor = (shelf: (filter: ThreadListFilter) => boolean) =>
  api.listThreads.mock.calls.some(([options]) => options.filter && shelf(options.filter));
const pinnedShelf = (filter: ThreadListFilter) => filter.pinned === true;
const olderShelf = (filter: ThreadListFilter) =>
  filter.pinned === false && filter.touchedFrom === undefined;

function renderRail(props: Partial<DeskRailThreadsProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/desk"]}>
        <DeskRailThreads
          place="today"
          agentsById={new Map<string, AgentChoice>()}
          activeThreadId={null}
          activeThread={null}
          canWatch
          canDecide
          watchtowerCount={0}
          decisionsCount={0}
          decisionsWaitHere={false}
          onSearch={vi.fn()}
          onSettings={vi.fn()}
          onSetPinned={vi.fn()}
          onDelete={vi.fn()}
          folded={false}
          onFold={vi.fn()}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

let restoreLayout = () => {};

beforeEach(() => {
  api.listThreads.mockReset();
  api.countThreadShelves.mockReset();
  useDeskSettingsStore.setState({ collapsedShelves: [] });
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
});

/**
 * Each shelf of the rail is read from the server on its own, and only while
 * it is open: a folded shelf costs its count and nothing more.
 */
describe("DeskRailThreads", () => {
  it("reads each open shelf with its own range and counts them all", async () => {
    serve({
      pinned: [thread("p", { pinned: true, title: "Pinned one" })],
      dated: [thread("t", { title: "Today's one" })],
      older: [thread("o", { lastMessageAt: now - 90 * DAY, title: "Old one" })],
    });
    renderRail();

    expect(await screen.findByText("Pinned one")).toBeInTheDocument();
    expect(await screen.findByText("Today's one")).toBeInTheDocument();
    expect(await screen.findByText("Old one")).toBeInTheDocument();
    expect(api.countThreadShelves).toHaveBeenCalledTimes(1);
    expect(api.countThreadShelves.mock.calls[0][0]).toHaveLength(4);
  });

  it("asks nothing of a folded shelf but still draws its heading", async () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["older", "pinned"] });
    serve({
      pinned: [thread("p", { pinned: true })],
      dated: [thread("t", { title: "Today's one" })],
      older: [
        thread("o1", { lastMessageAt: now - 90 * DAY }),
        thread("o2", { lastMessageAt: now - 95 * DAY }),
      ],
    });
    renderRail();

    expect(await screen.findByText("Today's one")).toBeInTheDocument();
    expect(
      await screen.findByRole("button", { name: "Older", expanded: false }),
    ).toBeInTheDocument();
    expect(askedFor(olderShelf)).toBe(false);
    expect(askedFor(pinnedShelf)).toBe(false);
  });

  it("reads a folded shelf once it is opened", async () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["older"] });
    serve({ older: [thread("o", { lastMessageAt: now - 90 * DAY, title: "Old one" })] });
    renderRail();

    fireEvent.click(
      await screen.findByRole("button", { name: "Older", expanded: false }),
    );

    expect(await screen.findByText("Old one")).toBeInTheDocument();
    expect(askedFor(olderShelf)).toBe(true);
  });

  // A conversation touched since an older shelf was read is held by two
  // shelves until that shelf is read again: it is drawn once, where its last
  // touch puts it, from the later copy.
  it("draws a conversation two shelves hold once, from its latest copy", async () => {
    const before = thread("x", { lastMessageAt: now - 90 * DAY, title: "Before reply" });
    const after = thread("x", { lastMessageAt: now - 30, title: "After reply" });
    serve({ dated: [after], older: [before] });
    // The server holds the conversation once, so it counts it once, where it is now.
    api.countThreadShelves.mockResolvedValue({ pinned: 0, bands: [1, 0, 0, 0, 0] });
    renderRail();

    expect(await screen.findByText("After reply")).toBeInTheDocument();
    await waitFor(() => expect(askedFor(olderShelf)).toBe(true));
    expect(screen.queryByText("Before reply")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Older/ })).not.toBeInTheDocument();
  });

  it("keeps the open conversation on its folded shelf without reading the shelf", async () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["older"] });
    const open = thread("o", { lastMessageAt: now - 90 * DAY, title: "Open one" });
    serve({ older: [open, thread("o2", { lastMessageAt: now - 91 * DAY })] });
    renderRail({ place: "thread", activeThreadId: "o", activeThread: open });

    expect(await screen.findByText("Open one")).toBeInTheDocument();
    expect(askedFor(olderShelf)).toBe(false);
  });

  it("says there are no chats only once every shelf has been counted empty", async () => {
    serve({});
    renderRail();

    expect(await screen.findByText(/Your chats will show up here/)).toBeInTheDocument();
  });

  it("does not say there are no chats while the shelves are being counted", () => {
    serve({});
    api.countThreadShelves.mockReturnValue(new Promise(() => undefined));
    renderRail();

    expect(screen.queryByText(/Your chats will show up here/)).not.toBeInTheDocument();
  });
});
