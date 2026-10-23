import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useNavigationStore } from "@/stores/navigation-store";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SidebarLayout } from "../sidebar-layout";

const offered = vi.hoisted(() => ({ value: true, sheetLoads: 0 }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: false, isLoading: false }),
  usePageAccessOffered: () => offered.value,
}));
vi.mock("../../page-access/page-access-sheet", async () => {
  offered.sheetLoads += 1;
  const { useAppDialogOpen } = await import("@/stores/app-dialogs-store");
  return {
    PageAccessSheet: function PageAccessSheet() {
      return <p>{useAppDialogOpen("page-access") ? "sheet open" : "sheet closed"}</p>;
    },
  };
});
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
  offered.value = true;
  useAppDialogsStore.setState({ active: null, prepared: [] });
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

function renderShell() {
  const router = createMemoryRouter(
    [{ path: "*", element: <SidebarLayout>{null}</SidebarLayout> }],
    { initialEntries: ["/shipments"] },
  );
  return render(<RouterProvider router={router} />);
}

// Happy DOM reports Ctrl+Alt as AltGr, which a browser on Linux or macOS does not.
function pressPageAccess() {
  const event = new KeyboardEvent("keydown", {
    key: "p",
    code: "KeyP",
    ctrlKey: true,
    altKey: true,
    bubbles: true,
    cancelable: true,
  });
  Object.defineProperty(event, "getModifierState", { value: () => false });
  act(() => {
    fireEvent(document.body, event);
  });
}

describe("the shell's page access panel", () => {
  it("reads nothing of the panel until it is first opened", () => {
    renderShell();

    expect(screen.queryByText(/^sheet /)).not.toBeInTheDocument();
    expect(offered.sheetLoads).toBe(0);
  });

  it("opens with Ctrl+Alt+P and stays mounted once closed, so it can close in place", async () => {
    renderShell();

    pressPageAccess();
    expect(await screen.findByText("sheet open")).toBeInTheDocument();

    pressPageAccess();
    expect(await screen.findByText("sheet closed")).toBeInTheDocument();
  });

  it("reads the panel, still closed, as soon as it is asked for", async () => {
    renderShell();

    act(() => useAppDialogsStore.getState().prepareDialog("page-access"));

    expect(await screen.findByText("sheet closed")).toBeInTheDocument();
    expect(useAppDialogsStore.getState().active).toBeNull();
  });

  it("opens when something else opens it, such as the user menu", async () => {
    renderShell();

    act(() => useAppDialogsStore.getState().openDialog("page-access"));

    expect(await screen.findByText("sheet open")).toBeInTheDocument();
  });

  it("leaves Ctrl+Alt+P alone for someone the panel is not offered to", () => {
    offered.value = false;
    renderShell();

    pressPageAccess();

    expect(useAppDialogsStore.getState().active).toBeNull();
  });
});
