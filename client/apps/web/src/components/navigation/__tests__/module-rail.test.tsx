import type { NavModule } from "@/config/navigation.types";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ModuleRail } from "../module-rail";
import type { SidebarModuleView } from "../sidebar-model";

const Icon = () => null;

function module(id: NavModule["id"], label: string, basePath: string): NavModule {
  return { id, label, icon: Icon, basePath, navigation: [] } as unknown as NavModule;
}

const shipments = module("shipment", "Shipments", "/shipments");
const billing = module("billing", "Billing", "/billing");
const views = new Map<string, SidebarModuleView>([
  [
    "shipment",
    {
      module: shipments,
      sections: [
        {
          id: "pages",
          label: null,
          items: [{ id: "board", label: "Shipments", path: "/shipments" }],
        },
      ],
      configuration: [],
      landingPath: "/shipments",
    },
  ],
  [
    "billing",
    {
      module: billing,
      sections: [
        {
          id: "pages",
          label: null,
          items: [
            { id: "queue", label: "Billing queue", path: "/billing/queue" },
            { id: "invoices", label: "Invoices", path: "/billing/invoices" },
          ],
        },
      ],
      configuration: [],
      landingPath: "/billing/queue",
    },
  ],
]);

vi.mock("@/hooks/use-sidebar-navigation", () => ({
  useSidebarNavigation: () => ({
    home: null,
    settings: null,
    domains: [{ id: "ops", label: "Operations", modules: [shipments, billing] }],
    views,
    activeModule: shipments,
    activePath: "/shipments",
    attention: new Map([["billing", { count: 7, tone: "destructive" }]]),
    pageAttention: new Map(),
    hiddenSections: new Set(),
  }),
}));
vi.mock("../org-switcher", () => ({ OrgSwitcher: () => null }));
vi.mock("../user-menu", () => ({ UserMenu: () => null }));
vi.mock("../../notification-center/notification-sheet", () => ({
  NotificationSheet: () => null,
}));

afterEach(() => cleanup());

function renderRail(panelFolded: boolean) {
  return render(
    <MemoryRouter initialEntries={["/shipments"]}>
      <ModuleRail panelFolded={panelFolded} search={panelFolded} />
    </MemoryRouter>,
  );
}

/**
 * The rail is the product's map: every module, the one you are in marked,
 * how much waits behind each on its corner. With the panel open a tile only
 * needs its name; folded, the tile also opens the module's pages on hover,
 * and search comes back onto the rail because the panel's field is gone.
 */
describe("the module rail", () => {
  it("marks the module you are in and counts the work behind the others", () => {
    renderRail(false);

    expect(screen.getByRole("link", { name: "Shipments" })).toHaveAttribute("aria-current", "page");
    const billingTile = screen.getByRole("link", { name: "Billing" });
    expect(billingTile).not.toHaveAttribute("aria-current");
    expect(billingTile).toHaveAttribute("href", "/billing/queue");
    expect(billingTile).toContainElement(screen.getByLabelText("7 items need attention"));
  });

  it("opens a module's pages on hover only while the panel is folded", () => {
    const { unmount } = renderRail(false);
    expect(screen.getByRole("link", { name: "Billing" })).not.toHaveAttribute(
      "data-slot",
      "hover-card-trigger",
    );
    expect(screen.queryByRole("button", { name: "Search" })).not.toBeInTheDocument();
    unmount();

    renderRail(true);
    expect(screen.getByRole("link", { name: "Billing" })).toHaveAttribute(
      "data-slot",
      "hover-card-trigger",
    );
    expect(screen.getByRole("button", { name: "Search" })).toBeInTheDocument();
  });
});
