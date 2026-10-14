import { useNavigationStore } from "@/stores/navigation-store";
import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RailShell } from "../rail-shell";

vi.mock("../module-rail", () => ({
  ModuleRail: () => <nav aria-label="Modules" />,
}));
vi.mock("../rail-panel", () => ({
  RailPanel: ({ shown }: { shown: boolean }) => (
    <aside aria-label="Sidebar" hidden={!shown || undefined}>
      <a href="/">Home</a>
    </aside>
  ),
  RailPanelBody: () => <a href="/">Home</a>,
}));
vi.mock("@/hooks/use-sidebar-navigation", () => ({
  useSidebarNavigation: () => ({ views: new Map(), home: null, settings: null }),
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

function renderShell() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <RailShell>
        <p>page body</p>
      </RailShell>
    </MemoryRouter>,
  );
}

/**
 * On a phone the rail and the panel never sit beside the page: together they
 * are a sheet over it, closed until asked for. The shell decides the sheet is
 * needed and whether it is out from one reading of the viewport, so it does
 * not open on the render that first mounts it.
 */
describe("the rail shell on a phone", () => {
  it("sits beside the page at desk widths", () => {
    renderShell();

    expect(screen.getByRole("navigation", { name: "Modules" })).toBeInTheDocument();
    expect(screen.getByRole("complementary", { name: "Sidebar" })).toBeVisible();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("folds the panel and keeps the rail", () => {
    useNavigationStore.setState({ sidebarCollapsed: true });
    const { container } = renderShell();

    expect(screen.getByRole("navigation", { name: "Modules" })).toBeInTheDocument();
    expect(container.querySelector("aside[aria-label=Sidebar]")).not.toBeVisible();
  });

  it("takes no room beside the page on a phone and never opens its sheet uninvited", async () => {
    phone = true;
    renderShell();

    await act(async () => {});

    expect(useNavigationStore.getState().compactSidebarOpen).toBe(false);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Modules" })).not.toBeInTheDocument();
    expect(screen.queryByRole("complementary", { name: "Sidebar" })).not.toBeInTheDocument();
    expect(screen.getByText("page body")).toBeInTheDocument();
  });

  it("slides the rail and the panel over the page as one sheet when asked for", async () => {
    phone = true;
    renderShell();

    act(() => useNavigationStore.getState().toggleSidebarFor(true));

    const sheet = await screen.findByRole("dialog", { name: "Sidebar" });
    expect(sheet).toContainElement(screen.getByRole("navigation", { name: "Modules" }));
    expect(sheet).toContainElement(screen.getByRole("link", { name: "Home" }));
  });
});
