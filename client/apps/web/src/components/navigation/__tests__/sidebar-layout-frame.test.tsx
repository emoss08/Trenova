import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { frameOfMatches } from "../shell-frame";
import { PageLayout, SidebarLayout } from "../sidebar-layout";

vi.mock("../workspace-header", () => ({
  WorkspaceHeader: ({ leading }: { leading?: ReactNode }) => (
    <header data-testid="app-header">{leading}</header>
  ),
  WorkspaceContextBar: () => <nav aria-label="Breadcrumb" />,
}));
vi.mock("../workspace-sidebar", () => ({
  WorkspaceSidebar: () => <aside aria-label="Sidebar" />,
}));
vi.mock("../variants/classic-sidebar", () => ({ ClassicSidebar: () => null }));
vi.mock("../user-settings-host", () => ({ UserSettingsHost: () => null }));
vi.mock("../../assistant/assistant-widget", () => ({
  AssistantWidget: () => null,
  useAssistantSideInset: () => null,
}));
vi.mock("../../command-palette/command-palette-mount", () => ({ CommandPaletteMount: () => null }));
vi.mock("../../keyboard-shortcuts-dialog", () => ({ KeyboardShortcutsDialog: () => null }));
vi.mock("../../header", () => ({
  Header: () => null,
  SidebarToggle: () => <button type="button">Hide sidebar</button>,
}));
vi.mock("@/hooks/use-filtered-navigation", () => ({ useFilteredNavigation: () => [] }));
vi.mock("@/stores/update-store", () => ({
  useUpdateStore: (select: (state: { fetchStatus: () => Promise<void> }) => unknown) =>
    select({ fetchStatus: () => Promise.resolve() }),
}));

beforeEach(() => {
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

function renderAt(path: string) {
  const page = (title: string) => (
    <PageLayout pageHeaderProps={{ title, description: `What ${title} is for` }}>
      <p>{title} body</p>
    </PageLayout>
  );
  const router = createMemoryRouter(
    [
      {
        element: (
          <SidebarLayout>
            <Outlet />
          </SidebarLayout>
        ),
        children: [
          { path: "/triage", handle: { frame: "room" }, element: page("Triage") },
          { path: "/shipments", element: page("Shipments") },
        ],
      },
    ],
    { initialEntries: [path] },
  );

  return render(<RouterProvider router={router} />);
}

/**
 * Triage, like the Desk, wants the window for its own rail, list and reader.
 * Unlike the Desk it stays inside the app: the header, its search, the module
 * switcher and the way to everywhere else remain. What goes is the module's
 * sidebar and the bar that says where you are, which the room's rail and title
 * replace, and the page's title bar, whose title stays for assistive tech.
 */
describe("the shell's frame", () => {
  it("takes the frame the deepest route that names one asks for", () => {
    expect(frameOfMatches([])).toBe("page");
    expect(frameOfMatches([{ handle: undefined }, { handle: { crumb: "Triage" } }])).toBe("page");
    expect(frameOfMatches([{ handle: { frame: "page" } }, { handle: { frame: "room" } }])).toBe(
      "room",
    );
    expect(frameOfMatches([{ handle: { frame: "room" } }, { handle: { frame: "page" } }])).toBe(
      "page",
    );
  });

  it("draws a page under the module's sidebar, the bar that says where you are and its title bar", async () => {
    renderAt("/shipments");

    expect(await screen.findByText("Shipments body")).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Sidebar" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Breadcrumb" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Shipments", level: 1 }).parentElement).not.toHaveClass(
      "sr-only",
    );
    expect(screen.getByTestId("app-header")).toBeEmptyDOMElement();
  });

  it("gives a room the window under the header, with the sidebar's toggle leading the header", async () => {
    renderAt("/triage");

    expect(await screen.findByText("Triage body")).toBeInTheDocument();
    expect(screen.queryByRole("complementary", { name: "Sidebar" })).not.toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Breadcrumb" })).not.toBeInTheDocument();
    expect(screen.getByTestId("app-header")).toContainElement(
      screen.getByRole("button", { name: "Hide sidebar" }),
    );
  });

  it("keeps a room's title and description for assistive tech without drawing its title bar", async () => {
    renderAt("/triage");

    const heading = await screen.findByRole("heading", { name: "Triage", level: 1 });
    expect(heading.closest("[data-slot=page-header]")).toHaveClass("sr-only");
    expect(screen.getByText("What Triage is for")).toBeInTheDocument();
    expect(screen.queryByRole("button")).toHaveAccessibleName("Hide sidebar");
  });
});
