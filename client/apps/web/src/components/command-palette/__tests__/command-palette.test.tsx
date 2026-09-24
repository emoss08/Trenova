import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { PaletteIntent } from "../palette-model";
import type { HomeInput, RemoteState } from "../palette-sections";
import { command, page, record } from "./palette-test-fixtures";

const { runIntent, remoteState, homeState, remoteArgs } = vi.hoisted(() => ({
  runIntent: vi.fn<(intent: PaletteIntent) => void>(),
  remoteArgs: [] as { enabled: boolean; query: string; scope: string }[],
  remoteState: { current: null as null | RemoteState },
  homeState: { current: null as null | HomeInput },
}));

vi.mock("../use-palette-intents", () => ({
  usePaletteIntentRunner: () => runIntent,
}));

vi.mock("@/components/assistant/use-ask", () => ({
  useAsk: () => ({ turn: null, ask: vi.fn(), stop: vi.fn(), reset: vi.fn(), keep: vi.fn() }),
}));

vi.mock("@/components/assistant/use-page-context", () => ({
  usePageContext: () => () => null,
}));

vi.mock("../_components/preview/palette-preview", () => ({
  PalettePreview: ({ item }: { item: { key: string } | null }) => (
    <div data-testid="preview">{item?.key ?? "tips"}</div>
  ),
}));

vi.mock("../_components/preview/preview-queries", () => ({
  prefetchRecordPreview: vi.fn(),
}));

const PAGES = [
  page(),
  page({
    id: "workers",
    title: "Workers",
    trail: "People > Workers",
    href: "/hr/workers",
    module: "People",
    keywords: ["Workers"],
  }),
];
const COMMANDS = [command()];

vi.mock("../use-palette-data", () => ({
  useUnreadCount: () => 0,
  usePaletteCatalog: () => ({
    pages: PAGES,
    pageIndex: new Map(PAGES.map((entry) => [entry.href, entry])),
    commands: COMMANDS,
    suggested: COMMANDS,
    actionContext: { mac: true, canReach: () => true, canUseAssistant: true },
  }),
  usePinnedPages: () => ({ pages: [], pinnedUrls: new Set<string>(), loading: false }),
  usePaletteHome: () => homeState.current,
  usePaletteRemoteSearch: (args: { enabled: boolean; query: string; scope: string }) => {
    remoteArgs.push(args);
    return { ...remoteState.current, retry: vi.fn() };
  },
}));

const { CommandPalette } = await import("../command-palette");

const SHIPMENT = record({ title: "PRO-1001", metadata: { proNumber: "PRO-1001" } });

function renderPalette() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/billing/invoices"]}>
        <CommandPalette />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function input(): HTMLInputElement {
  return screen.getByRole("combobox");
}

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

beforeEach(() => {
  runIntent.mockReset();
  remoteArgs.length = 0;
  remoteState.current = { groups: [], loading: false, error: false, ready: false };
  homeState.current = {
    recentRecords: [SHIPMENT],
    attention: { rows: [], loading: false },
    notifications: { items: [], unread: 0, loading: false },
    pinnedPages: { pages: [], loading: false },
    recentPages: [],
    suggested: COMMANDS,
  };
  useRecentRecordsStore.setState({ recordsByOrganization: {} });
  useAuthStore.setState({
    user: { currentOrganizationId: "org_1" } as ReturnType<typeof useAuthStore.getState>["user"],
  });
  useCommandPaletteStore.setState({ open: true });
});

afterEach(() => {
  useCommandPaletteStore.setState({ open: false });
});

describe("CommandPalette", () => {
  it("shows recent records and quick actions before anything is typed", () => {
    renderPalette();

    expect(screen.getByText("Recent records")).toBeInTheDocument();
    expect(screen.getByText("PRO-1001")).toBeInTheDocument();
    expect(screen.getByText("Quick actions")).toBeInTheDocument();
    expect(screen.getByText("New shipment")).toBeInTheDocument();
  });

  it("opens the selected record on Enter and remembers it as recent", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Enter}");

    expect(runIntent).toHaveBeenCalledWith({ type: "navigate", href: SHIPMENT.href });
    expect(useRecentRecordsStore.getState().recordsByOrganization.org_1?.[0]?.id).toBe("shp_1");
  });

  it("ranks pages locally as the person types, with an exact match as the top hit", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "workers");

    const topHit = screen.getByRole("group", { name: /Top hit/ });
    expect(within(topHit).getByRole("option", { name: /Workers/ })).toBeInTheDocument();
    expect(screen.queryByText("Invoices")).not.toBeInTheDocument();
  });

  it("opens a new tab on Mod+Enter without closing on a plain Enter first", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Meta>}{Enter}{/Meta}");

    expect(runIntent).toHaveBeenCalledTimes(1);
    expect(runIntent).toHaveBeenCalledWith({ type: "new-tab", href: SHIPMENT.href });
  });

  it("copies the selected record's number on Alt+C", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Alt>}c{/Alt}");

    expect(runIntent).toHaveBeenCalledWith({ type: "copy", text: "PRO-1001" });
    expect(input()).toHaveValue("");
  });

  it("lists a row's actions on the right arrow and comes back on Escape without closing", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{ArrowRight}");

    expect(screen.getByText("Actions for PRO-1001")).toBeInTheDocument();
    expect(screen.getByText("Copy PRO number")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByText("Actions for PRO-1001")).not.toBeInTheDocument());
    expect(useCommandPaletteStore.getState().open).toBe(true);
    expect(screen.getByText("Recent records")).toBeInTheDocument();
  });

  it("filters the action list by what is typed and runs the chosen one", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{ArrowRight}");
    await user.type(input(), "copy link{Enter}");

    expect(runIntent).toHaveBeenCalledWith({ type: "copy-link", href: SHIPMENT.href });
  });

  it("moves through scopes with Tab and back to everything with Backspace", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Tab}");
    expect(screen.getByRole("tab", { name: /Shipments/ })).toHaveAttribute("aria-selected", "true");

    await user.type(input(), "{Backspace}");
    expect(screen.getByRole("tab", { name: /All/ })).toHaveAttribute("aria-selected", "true");
  });

  it("commits an @ mention to a record scope", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "@cust ");

    expect(screen.getByRole("tab", { name: /Customers/ })).toHaveAttribute("aria-selected", "true");
    expect(input()).toHaveValue("");
  });

  it("commits a mention prefix that can only mean one kind of record", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "@work ");

    expect(screen.getByRole("tab", { name: /Workers/ })).toHaveAttribute("aria-selected", "true");
  });

  it("does not claim nothing matched when record search failed", async () => {
    remoteState.current = { groups: [], loading: false, error: true, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "zzzzqqq");

    expect(screen.getByText("Record search is unavailable right now.")).toBeInTheDocument();
    expect(screen.queryByText("Nothing matches that")).not.toBeInTheDocument();
  });

  it("says so inside the list when record search fails, and keeps pages", async () => {
    remoteState.current = { groups: [], loading: false, error: true, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "workers");

    expect(screen.getByText("Record search is unavailable right now.")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /Workers/ })).toBeInTheDocument();
  });

  it("takes a question to the assistant instead of searching records with it", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "how many loads are late?");

    expect(screen.getByRole("option", { name: /Ask the assistant/ })).toBeInTheDocument();
    expect(remoteArgs.at(-1)?.enabled).toBe(false);
  });

  it("offers the empty state with a way forward when nothing matches", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "zzzzqqq");

    expect(screen.getByText("Nothing matches that")).toBeInTheDocument();
  });
});
