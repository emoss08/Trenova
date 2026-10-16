import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import { Profiler } from "react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { PaletteIntent } from "../palette-model";
import type { HomeInput, RemoteState } from "../palette-sections";
import { command, page, record } from "./palette-test-fixtures";

const {
  runIntent,
  remoteState,
  remoteFoundFor,
  homeState,
  remoteArgs,
  previewKeys,
  ask,
  globalSearch,
} = vi.hoisted(() => ({
  runIntent: vi.fn<(intent: PaletteIntent) => void>(),
  ask: vi.fn(),
  globalSearch: vi.fn(),
  remoteArgs: [] as { enabled: boolean; query: string; scope: string }[],
  remoteState: { current: null as null | RemoteState },
  remoteFoundFor: { current: null as null | string },
  homeState: { current: null as null | HomeInput },
  previewKeys: [] as string[],
}));

vi.mock("../use-palette-intents", () => ({
  usePaletteIntentRunner: () => runIntent,
}));

vi.mock("@/components/assistant/use-ask", () => ({
  useAsk: () => ({ turn: null, ask, stop: vi.fn(), reset: vi.fn(), keep: vi.fn() }),
}));

vi.mock("@/services/api", () => ({
  apiService: { globalSearchService: { search: globalSearch } },
}));

vi.mock("@/components/assistant/use-page-context", () => ({
  usePageContext: () => () => null,
}));

vi.mock("../_components/preview/palette-preview", () => ({
  PalettePreview: ({ item }: { item: { key: string } | null }) => {
    previewKeys.push(item?.key ?? "tips");
    return <div data-testid="preview">{item?.key ?? "tips"}</div>;
  },
}));

vi.mock("../_components/preview/preview-queries", () => ({
  prefetchRecordPreview: vi.fn(),
}));

const PAGES = [
  page(),
  page({
    id: "workers",
    title: "Workers",
    trail: ["People"],
    href: "/hr/workers",
    module: "People",
    keywords: ["Workers"],
  }),
];
const COMMANDS = [command()];

vi.mock("../use-palette-data", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../use-palette-data")>()),
  useUnreadCount: () => 0,
  usePaletteCatalog: () => ({
    pages: PAGES,
    pageIndex: new Map(PAGES.map((entry) => [entry.href, entry])),
    commands: COMMANDS,
    quickActions: COMMANDS,
    actionContext: {
      mac: true,
      canReach: () => true,
      canUseAssistant: true,
      canCreateShipment: true,
      relatedCommands: () => [],
    },
  }),
  usePinnedPages: () => ({ pages: [], pinnedUrls: new Set<string>(), loading: false }),
  usePaletteHome: () => homeState.current,
  usePaletteRemoteSearch: (args: { enabled: boolean; query: string; scope: string }) => {
    remoteArgs.push(args);
    return {
      ...remoteState.current,
      query: remoteFoundFor.current ?? args.query,
      refreshing: false,
      retry: vi.fn(),
    };
  },
}));

const { CommandPalette } = await import("../command-palette");

const SHIPMENT = record({ title: "PRO-1001", metadata: { proNumber: "PRO-1001" } });
const VISITED_AT = 1_700_000_000_000;

function renderPalette(onCommit: () => void = () => undefined) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/billing/invoices"]}>
        <Profiler id="palette" onRender={onCommit}>
          <CommandPalette />
        </Profiler>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/** Distinct values in the order they came, with repeats in a row folded into one. */
function sequence(values: readonly string[]): string[] {
  return values.filter((value, index) => value !== values[index - 1]);
}

function input(): HTMLInputElement {
  return screen.getByRole("combobox");
}

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

beforeEach(() => {
  runIntent.mockReset();
  ask.mockReset();
  globalSearch.mockReset();
  remoteArgs.length = 0;
  previewKeys.length = 0;
  remoteFoundFor.current = null;
  remoteState.current = { groups: [], loading: false, error: false, ready: false };
  homeState.current = {
    recentRecords: [{ record: SHIPMENT, visitedAt: VISITED_AT }],
    attention: { rows: [], loading: false },
    pinnedPages: { pages: [], loading: false },
    quickActions: COMMANDS,
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
  it("shows where the person just was and quick actions before anything is typed", () => {
    renderPalette();

    expect(screen.getByText("Jump back in")).toBeInTheDocument();
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

  it("opens a new tab on Mod+Enter without running a plain Enter first", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Meta>}{Enter}{/Meta}");

    expect(runIntent).toHaveBeenCalledTimes(1);
    expect(runIntent).toHaveBeenCalledWith({ type: "new-tab", href: SHIPMENT.href });
  });

  it("copies the selected record's number on Alt+C and keeps what was typed", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Alt>}c{/Alt}");

    expect(runIntent).toHaveBeenCalledWith({ type: "copy", text: "PRO-1001", what: "pro" });
    expect(input()).toHaveValue("");
  });

  it("copies the selected record's link on Mod+L", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Meta>}l{/Meta}");

    expect(runIntent).toHaveBeenCalledWith({ type: "copy-link", href: SHIPMENT.href });
  });

  it("opens the actions menu on the right arrow and closes only it on Escape", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{ArrowRight}");

    const menu = screen.getByRole("dialog", { name: "Actions for PRO-1001" });
    expect(within(menu).getByRole("option", { name: /Copy PRO number/ })).toBeInTheDocument();

    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Actions for PRO-1001" }),
      ).not.toBeInTheDocument(),
    );
    expect(useCommandPaletteStore.getState().open).toBe(true);
    expect(screen.getByText("Jump back in")).toBeInTheDocument();
  });

  it("opens the actions menu on Mod+K too", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Meta>}k{/Meta}");

    expect(screen.getByRole("dialog", { name: "Actions for PRO-1001" })).toBeInTheDocument();
  });

  it("filters the action list by what is typed and runs the chosen one", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{ArrowRight}");
    await user.keyboard("copy link{Enter}");

    expect(runIntent).toHaveBeenCalledWith({ type: "copy-link", href: SHIPMENT.href });
  });

  it("steps back one layer per Escape: query, then scope, then closes", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Tab}");
    await user.type(input(), "pro");
    await user.keyboard("{Escape}");

    expect(input()).toHaveValue("");
    expect(screen.getByRole("tab", { name: /Shipments/ })).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{Escape}");
    expect(screen.getByRole("tab", { name: /All/ })).toHaveAttribute("aria-selected", "true");
    expect(useCommandPaletteStore.getState().open).toBe(true);

    await user.keyboard("{Escape}");
    await waitFor(() => expect(useCommandPaletteStore.getState().open).toBe(false));
  });

  it("moves through scopes with Tab and back to everything with Backspace", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Tab}");
    expect(screen.getByRole("tab", { name: /Shipments/ })).toHaveAttribute("aria-selected", "true");

    await user.type(input(), "{Backspace}");
    expect(screen.getByRole("tab", { name: /All/ })).toHaveAttribute("aria-selected", "true");
  });

  it("browses a record scope with nothing typed", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Tab}");

    expect(remoteArgs.at(-1)).toEqual({ enabled: true, query: "", scope: "shipment" });
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

  it("does not search with a mention that is still being typed", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "@doc");

    expect(screen.getByText("Jump back in")).toBeInTheDocument();
    expect(remoteArgs.at(-1)?.query).toBe("");
  });

  it("searches only commands after a leading >", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), ">new");

    expect(screen.getByRole("option", { name: /New shipment/ })).toBeInTheDocument();
    expect(remoteArgs.at(-1)?.scope).toBe("commands");
  });

  it("does not claim nothing matched when record search failed", async () => {
    remoteState.current = { groups: [], loading: false, error: true, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "zzzzqqq");

    expect(screen.getByText("Record search is unavailable right now.")).toBeInTheDocument();
    expect(screen.queryByText("Nothing called “zzzzqqq”")).not.toBeInTheDocument();
  });

  it("says so inside the list when record search fails, and keeps pages", async () => {
    remoteState.current = { groups: [], loading: false, error: true, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "workers");

    expect(screen.getByText("Record search is unavailable right now.")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /Workers/ })).toBeInTheDocument();
  });

  it("searches records for a question's search words, under the way to ask Nova", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "Tell me about SEED-PAY-002?");

    expect(screen.getByRole("option", { name: /Ask Nova/ })).toBeInTheDocument();
    await waitFor(() =>
      expect(remoteArgs.at(-1)).toEqual({ enabled: true, query: "SEED-PAY-002", scope: "all" }),
    );
  });

  it("asks Nova with the records its search found that the question names", async () => {
    const groups = [
      {
        entityType: "shipment",
        label: "Shipments",
        hits: [
          {
            id: "shp_2",
            entityType: "shipment",
            title: "SEED-PAY-002",
            href: "/shipments/shp_2",
          },
          {
            id: "shp_5",
            entityType: "shipment",
            title: "SEED-PAY-005",
            href: "/shipments/shp_5",
          },
        ],
      },
    ];
    globalSearch.mockResolvedValue({ query: "SEED-PAY-002", groups });
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "Tell me about SEED-PAY-002?");
    await user.keyboard("{Enter}");

    expect(ask).toHaveBeenCalledTimes(1);
    const [question, options] = ask.mock.calls[0]!;
    expect(question).toBe("Tell me about SEED-PAY-002?");
    expect(options.mentions).toBeUndefined();
    await expect(options.findMentions(new AbortController().signal)).resolves.toEqual([
      { type: "shipment", id: "shp_2", label: "SEED-PAY-002" },
    ]);
    expect(globalSearch).toHaveBeenCalledWith("SEED-PAY-002", 5, undefined, {
      signal: expect.any(AbortSignal),
    });
  });

  it("ends every search with a way to ask Nova", async () => {
    remoteState.current = { groups: [], loading: false, error: false, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "workers");

    const options = screen.getAllByRole("option");
    expect(options.at(-1)).toHaveAccessibleName(/Ask Nova/);
  });

  it("names what was searched for when nothing matches, and offers to ask instead", async () => {
    remoteState.current = { groups: [], loading: false, error: false, ready: true };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "zzzzqqq");

    expect(screen.getByText("Nothing called “zzzzqqq”")).toBeInTheDocument();
    expect(screen.getByText("Try asking instead")).toBeInTheDocument();
  });

  it("asks about the selected record on Mod+I with that record's starter questions", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "{Meta>}i{/Meta}");

    expect(runIntent).toHaveBeenCalledWith({
      type: "ask-about",
      subject: { type: "shipment", id: "shp_1", label: "PRO-1001" },
    });
  });

  it("keeps the chosen row until the new list is drawn, and starts that list on its first row", async () => {
    remoteState.current = {
      groups: [
        {
          entityType: "shipment",
          label: "Shipments",
          hits: [{ id: "shp_2", entityType: "shipment", title: "PRO-2002", href: "/shp_2" }],
        },
      ],
      loading: false,
      error: false,
      ready: true,
    };
    const commits: { first: string | undefined; primary: string }[] = [];
    const user = userEvent.setup();
    renderPalette(() => {
      const first = document.querySelector<HTMLElement>("[data-palette-row]");
      commits.push({
        first: first?.dataset.paletteKey,
        primary: screen.queryByText("Run", { exact: true }) ? "Run" : "Open",
      });
    });

    await user.type(input(), "{ArrowDown}");
    expect(screen.getByRole("option", { name: /New shipment/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    commits.length = 0;

    await user.type(input(), "s");

    const newList = commits.filter(({ first }) => first === "record:shipment:shp_2");
    expect(newList.length).toBeGreaterThan(0);
    expect(newList.map(({ primary }) => primary)).not.toContain("Run");
    expect(
      commits.filter(({ first }) => first === "recent-record:shipment:shp_1").map((c) => c.primary),
    ).not.toContain("Open");
    expect(screen.getByRole("option", { name: /PRO-2002/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("holds the preview while the person types and shows the row the list lands on", async () => {
    const user = userEvent.setup({ delay: 120 });
    renderPalette();

    await user.type(input(), "w{Backspace}i");

    await waitFor(() =>
      expect(screen.getByTestId("preview")).toHaveTextContent("page:/billing/invoices"),
    );
    expect(sequence(previewKeys)).toEqual([
      "recent-record:shipment:shp_1",
      "page:/billing/invoices",
    ]);
  });

  it("marks found records for the words that found them while the next search is on its way", async () => {
    remoteFoundFor.current = "acme";
    remoteState.current = {
      groups: [
        {
          entityType: "customer",
          label: "Customers",
          hits: [{ id: "cus_1", entityType: "customer", title: "Acme Freight", href: "/cus_1" }],
        },
      ],
      loading: false,
      error: false,
      ready: true,
    };
    const user = userEvent.setup();
    renderPalette();

    await user.type(input(), "acme fr");

    const row = screen.getByRole("option", { name: /Acme Freight/ });
    expect([...row.querySelectorAll("mark")].map((mark) => mark.textContent)).toEqual(["Acme"]);
  });
});
