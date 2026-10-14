import { useNavigationStore } from "@/stores/navigation-store";
import { afterEach, describe, expect, it } from "vitest";

afterEach(() => {
  useNavigationStore.setState({ sidebarCollapsed: false, compactSidebarOpen: false });
  localStorage.clear();
});

/**
 * On a phone the sidebar never sits beside the page. Its sheet has its own
 * open state, so sliding it out never unfolds the sidebar a person folded on
 * a desk, and it is never kept, so every visit on a phone starts on the page.
 */
describe("the sidebar's fold", () => {
  it("slides the phone's sheet in and out without touching the kept fold", () => {
    const store = useNavigationStore.getState();

    store.setSidebarCollapsed(true);
    store.toggleSidebarFor(true);

    expect(useNavigationStore.getState().compactSidebarOpen).toBe(true);
    expect(useNavigationStore.getState().sidebarCollapsed).toBe(true);

    useNavigationStore.getState().toggleSidebarFor(true);

    expect(useNavigationStore.getState().compactSidebarOpen).toBe(false);
  });

  it("folds the sidebar beside the page without opening the phone's sheet", () => {
    useNavigationStore.getState().toggleSidebarFor(false);

    expect(useNavigationStore.getState().sidebarCollapsed).toBe(true);
    expect(useNavigationStore.getState().compactSidebarOpen).toBe(false);
  });

  it("keeps the fold but never the phone's sheet", () => {
    useNavigationStore.getState().setSidebarCollapsed(true);
    useNavigationStore.getState().setCompactSidebarOpen(true);

    const kept = JSON.parse(localStorage.getItem("navigation-storage") ?? "{}") as {
      state?: Record<string, unknown>;
    };

    expect(kept.state?.sidebarCollapsed).toBe(true);
    expect(kept.state).not.toHaveProperty("compactSidebarOpen");
  });
});

/**
 * The panel beside the rail is as wide as the person dragged it, within the
 * bounds the layout can hold, and a width that was kept by an older build or
 * edited by hand lands inside those bounds too.
 */
describe("the panel's width", () => {
  it("holds a dragged width between its bounds", () => {
    useNavigationStore.getState().setRailPanelWidth(260);
    expect(useNavigationStore.getState().railPanelWidth).toBe(260);

    useNavigationStore.getState().setRailPanelWidth(120);
    expect(useNavigationStore.getState().railPanelWidth).toBe(200);

    useNavigationStore.getState().setRailPanelWidth(900);
    expect(useNavigationStore.getState().railPanelWidth).toBe(320);

    useNavigationStore.getState().setRailPanelWidth(Number.NaN);
    expect(useNavigationStore.getState().railPanelWidth).toBe(224);
  });

  it("keeps the width with the fold", () => {
    useNavigationStore.getState().setRailPanelWidth(288);

    const kept = JSON.parse(localStorage.getItem("navigation-storage") ?? "{}") as {
      state?: Record<string, unknown>;
    };

    expect(kept.state?.railPanelWidth).toBe(288);
  });
});
