import { useNavigationStore } from "@/stores/navigation-store";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { Outlet } from "@tanstack/react-router";
import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { frameOfMatches, roomRailOfMatches } from "../shell-frame";
import { PageLayout, SidebarLayout } from "../sidebar-layout";

vi.mock("../module-rail", () => ({
  ModuleRail: ({
    fold,
    search,
    panelFolded,
  }: {
    fold?: boolean;
    search: boolean;
    panelFolded: boolean;
  }) => (
    <nav
      aria-label="Modules"
      data-search={search || undefined}
      data-panel-folded={panelFolded || undefined}
    >
      {fold && <button type="button">Hide sidebar</button>}
    </nav>
  ),
}));
vi.mock("../rail-panel", () => ({
  RailPanel: ({ shown }: { shown: boolean }) => (
    <aside aria-label="Sidebar" hidden={!shown || undefined} />
  ),
  RailPanelBody: () => null,
}));
vi.mock("../../assistant/assistant-widget", () => ({
  AssistantWidget: () => null,
  useAssistantSideInset: () => null,
}));
vi.mock("../../command-palette/command-palette-mount", () => ({ CommandPaletteMount: () => null }));
vi.mock("../../keyboard-shortcuts-dialog", () => ({ KeyboardShortcutsDialog: () => null }));
vi.mock("../../header", () => ({
  SidebarToggle: () => <button type="button">Hide sidebar</button>,
  HistoryNavigation: () => <button type="button">Go back</button>,
  FavoriteToggle: () => <button type="button">Add to favorites</button>,
}));
vi.mock("@/hooks/use-filtered-navigation", () => ({ useFilteredNavigation: () => [] }));
vi.mock("@/hooks/use-sidebar-navigation", () => ({
  useSidebarNavigation: () => ({ views: new Map(), home: null, settings: null }),
}));
vi.mock("@/stores/update-store", () => ({
  useUpdateStore: (select: (state: { fetchStatus: () => Promise<void> }) => unknown) =>
    select({ fetchStatus: () => Promise.resolve() }),
}));

beforeEach(() => {
  useNavigationStore.setState({ sidebarCollapsed: false, compactSidebarOpen: false });
  vi.stubGlobal(
    "matchMedia",
    (query: string) =>
      ({
        matches: false,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

async function renderAt(path: string) {
  const page = (title: string) => (
    <PageLayout pageHeaderProps={{ title, description: `What ${title} is for` }}>
      <p>{title} body</p>
    </PageLayout>
  );
  return renderAtRoutes(
    [
      { path: "/triage", staticData: { frame: "room" }, component: () => page("Triage") },
      {
        path: "/studio",
        staticData: { frame: "room", roomRail: "stands" },
        component: () => page("Studio"),
      },
      { path: "/shipments", component: () => page("Shipments") },
      { path: "/dispatch/outlook", component: () => page("Freight outlook") },
      { path: "/accounting/journal-entries/$id", component: () => page("Journal entry") },
    ],
    {
      initialEntries: [path],
      layout: () => (
        <SidebarLayout>
          <Outlet />
        </SidebarLayout>
      ),
    },
  );
}

/**
 * A page sits beside the rail and the module's panel and owns the only bar
 * above its body: the fold, history and the trail lead its title, the star
 * follows its actions. Triage wants the window for its own rail, list and
 * reader: the module rail stays and carries the fold for the room's rail,
 * the panel goes, and the page's title bar stays only for assistive tech.
 */
describe("the shell's frame", () => {
  it("takes the frame the deepest route that names one asks for", () => {
    expect(frameOfMatches([])).toBe("page");
    expect(frameOfMatches([{ staticData: undefined }, { staticData: { roomRail: "folds" } }])).toBe(
      "page",
    );
    expect(
      frameOfMatches([{ staticData: { frame: "page" } }, { staticData: { frame: "room" } }]),
    ).toBe("room");
    expect(
      frameOfMatches([{ staticData: { frame: "room" } }, { staticData: { frame: "page" } }]),
    ).toBe("page");
  });

  it("draws a page beside the rail and the panel, with one bar of its own", async () => {
    await renderAt("/shipments");

    expect(await screen.findByText("Shipments body")).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Modules" })).not.toHaveAttribute("data-search");
    expect(screen.getByRole("complementary", { name: "Sidebar" })).toBeInTheDocument();

    const bar = screen
      .getByRole("heading", { name: "Shipments", level: 1 })
      .closest("[data-slot=page-header]");
    expect(bar).not.toHaveClass("sr-only");
    expect(bar).toContainElement(screen.getByRole("button", { name: "Hide sidebar" }));
    expect(bar).toContainElement(screen.getByRole("button", { name: "Go back" }));
    expect(bar).toContainElement(screen.getByRole("button", { name: "Add to favorites" }));
  });

  it("leads the title with the trail above it, never with Home or the title itself", async () => {
    await renderAt("/dispatch/outlook");

    await screen.findByText("Freight outlook body");
    const trail = screen.getByRole("navigation", { name: "Breadcrumb" });
    expect(trail).toHaveTextContent("Dispatch management");
    expect(trail).not.toHaveTextContent("Home");
    expect(trail).not.toHaveTextContent("Freight outlook");
    expect(screen.getByRole("link", { name: "Dispatch management" })).toHaveAttribute(
      "href",
      "/dispatch",
    );
  });

  it("leaves out a folder of the URL that no page or section fills", async () => {
    await renderAt("/accounting/journal-entries/je_01HZX2K3M4N5P6Q7R8S9T0V1");

    await screen.findByText("Journal entry body");
    const trail = screen.getByRole("navigation", { name: "Breadcrumb" });
    expect(Array.from(trail.querySelectorAll("a")).map((link) => link.textContent)).toEqual([
      "Accounting management",
    ]);
  });

  it("draws no trail on a module's own landing page", async () => {
    await renderAt("/shipments");

    await screen.findByText("Shipments body");
    expect(screen.queryByRole("navigation", { name: "Breadcrumb" })).not.toBeInTheDocument();
  });

  it("gives a room the window beside the rail, which carries the fold and search", async () => {
    await renderAt("/triage");

    expect(await screen.findByText("Triage body")).toBeInTheDocument();
    const rail = screen.getByRole("navigation", { name: "Modules" });
    expect(rail).toHaveAttribute("data-search");
    expect(rail).toContainElement(screen.getByRole("button", { name: "Hide sidebar" }));
    expect(screen.queryByRole("complementary", { name: "Sidebar" })).not.toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Breadcrumb" })).not.toBeInTheDocument();
  });

  it("keeps a room's title and description for assistive tech without drawing its title bar", async () => {
    await renderAt("/triage");

    const heading = await screen.findByRole("heading", { name: "Triage", level: 1 });
    expect(heading.closest("[data-slot=page-header]")).toHaveClass("sr-only");
    expect(screen.getByText("What Triage is for")).toBeInTheDocument();
    expect(screen.queryByRole("button")).toHaveAccessibleName("Hide sidebar");
  });

  it("takes the room rail the deepest route that names one asks for, folding by default", () => {
    expect(roomRailOfMatches([])).toBe("folds");
    expect(roomRailOfMatches([{ staticData: { frame: "room" } }])).toBe("folds");
    expect(
      roomRailOfMatches([
        { staticData: { roomRail: "folds" } },
        { staticData: { roomRail: "stands" } },
      ]),
    ).toBe("stands");
    expect(
      roomRailOfMatches([
        { staticData: { roomRail: "stands" } },
        { staticData: { roomRail: "folds" } },
      ]),
    ).toBe("folds");
  });

  it("folds a room's rail with ⌘B", async () => {
    await renderAt("/triage");

    await screen.findByText("Triage body");
    expect(screen.getByRole("navigation", { name: "Modules" })).not.toHaveAttribute(
      "data-panel-folded",
    );

    fireEvent.keyDown(document, { key: "b", metaKey: true });

    expect(useNavigationStore.getState().sidebarCollapsed).toBe(true);
    expect(screen.getByRole("navigation", { name: "Modules" })).toHaveAttribute(
      "data-panel-folded",
    );
  });

  it("keeps a standing room's rail open, with the panel folded in its favour and no fold", async () => {
    await renderAt("/studio");

    await screen.findByText("Studio body");
    const rail = screen.getByRole("navigation", { name: "Modules" });
    expect(rail).toHaveAttribute("data-panel-folded");
    expect(rail).toHaveAttribute("data-search");
    expect(screen.queryByRole("button", { name: "Hide sidebar" })).not.toBeInTheDocument();
    expect(screen.queryByRole("complementary", { name: "Sidebar" })).not.toBeInTheDocument();

    fireEvent.keyDown(document, { key: "b", metaKey: true });

    expect(useNavigationStore.getState().sidebarCollapsed).toBe(false);
  });
});
