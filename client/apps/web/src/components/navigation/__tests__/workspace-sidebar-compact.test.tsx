import { useNavigationStore } from "@/stores/navigation-store";
import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WorkspaceSidebar } from "../workspace-sidebar";

vi.mock("@/hooks/use-sidebar-navigation", () => ({
  useSidebarNavigation: () => ({
    home: { id: "home", label: "Home", icon: () => null, basePath: "/" },
    settings: null,
    domains: [],
    views: new Map(),
    activeModule: null,
    activePath: null,
    attention: new Map(),
    pageAttention: new Map(),
    hiddenSections: new Set(["attention", "favorites", "activity"]),
  }),
}));

let phone = false;

beforeEach(() => {
  vi.stubGlobal(
    "matchMedia",
    (query: string) =>
      ({
        matches: phone,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  useNavigationStore.setState({ sidebarCollapsed: false, compactSidebarOpen: false });
  phone = false;
});

function renderSidebar() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <WorkspaceSidebar />
    </MemoryRouter>,
  );
}

/**
 * On a phone the sidebar squeezed every page to a sliver beside it; Triage's
 * own phone layout (the lane strip, the actions sheet) only worked once the
 * sidebar had been folded by hand. Below the phone breakpoint it is a sheet
 * over the page instead, closed until asked for.
 */
describe("the workspace sidebar on a phone", () => {
  it("sits beside the page at desk widths", () => {
    renderSidebar();

    expect(screen.getByRole("complementary", { name: "Sidebar" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Home" })).toBeInTheDocument();
  });

  it("takes no room beside the page on a phone, even when the fold is open", () => {
    phone = true;
    renderSidebar();

    expect(useNavigationStore.getState().sidebarCollapsed).toBe(false);
    expect(screen.queryByRole("complementary", { name: "Sidebar" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Home" })).not.toBeInTheDocument();
  });

  it("slides over the page as a sheet when asked for on a phone", async () => {
    phone = true;
    renderSidebar();

    act(() => useNavigationStore.getState().toggleSidebarFor(true));

    expect(await screen.findByRole("dialog", { name: "Sidebar" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Home" })).toBeInTheDocument();
  });
});
