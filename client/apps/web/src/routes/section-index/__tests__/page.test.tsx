import type { NavModule } from "@/config/navigation.types";
import type { SidebarModuleView } from "@/components/navigation/sidebar-model";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SectionIndexPage } from "../page";

const Icon = () => null;

const billing = {
  id: "billing",
  label: "Billing management",
  description: "Invoices, rates and what customers owe",
  icon: Icon,
  basePath: "/billing",
  navigation: [],
} as unknown as NavModule;

const billingView: SidebarModuleView = {
  module: billing,
  sections: [
    {
      id: "billing:queue",
      label: null,
      items: [{ id: "queue", label: "Billing queue", path: "/billing/queue", groupId: null }],
    },
  ],
  configuration: [
    {
      id: "formula-libraries",
      label: "Formula libraries",
      path: "/billing/configuration-files/formula-libraries",
      groupId: "billing-config-group",
    },
  ],
  landingPath: "/billing/queue",
};

let views = new Map<string, SidebarModuleView>();

vi.mock("@/hooks/use-sidebar-navigation", () => ({
  useSidebarNavigation: () => ({
    views,
    pageAttention: new Map([["/billing/queue", { count: 4, tone: "warning" }]]),
  }),
}));

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    pageHeaderProps,
    children,
  }: {
    pageHeaderProps: { title: string; description: string };
    children: React.ReactNode;
  }) => (
    <main>
      <h1>{pageHeaderProps.title}</h1>
      <p>{pageHeaderProps.description}</p>
      {children}
    </main>
  ),
}));

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SectionIndexPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  views = new Map([["billing", billingView]]);
});

afterEach(cleanup);

describe("SectionIndexPage", () => {
  it("titles a module section after the module and links every page it may open", () => {
    renderAt("/billing");

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Billing management");
    expect(screen.getByText("Invoices, rates and what customers owe")).toBeInTheDocument();

    const pages = screen.getByRole("region", { name: "Pages" });
    expect(within(pages).getByRole("link", { name: /Billing queue/ })).toHaveAttribute(
      "href",
      "/billing/queue",
    );
    expect(within(pages).getByText("4")).toBeInTheDocument();

    const configuration = screen.getByRole("region", { name: "Configuration" });
    expect(within(configuration).getByRole("link", { name: /Formula libraries/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/formula-libraries",
    );
  });

  it("lists only the group's own pages under a group section", () => {
    renderAt("/billing/configuration-files");

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Configuration files");
    expect(screen.getByText("Every page in Configuration files you can open.")).toBeInTheDocument();
    expect(screen.getAllByRole("link").map((link) => link.getAttribute("href"))).toEqual([
      "/billing/configuration-files/formula-libraries",
    ]);
  });

  it("says nothing here is open when the module is hidden from the person", () => {
    views = new Map();
    renderAt("/billing");

    expect(screen.getByText("No pages you can open here")).toBeInTheDocument();
    expect(screen.queryAllByRole("link")).toEqual([]);
  });
});
