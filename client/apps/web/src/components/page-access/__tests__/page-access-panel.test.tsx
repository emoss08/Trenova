import type { PageAccess, PageAccessSubject } from "@/lib/graphql/page-access";
import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PageAccessPanel } from "../page-access-panel";

const fetchPageAccess = vi.fn<(location: string) => Promise<PageAccess | null>>();
const fetchPageAccessPreview =
  vi.fn<(location: string, subject: PageAccessSubject) => Promise<PageAccess | null>>();

vi.mock("@/lib/graphql/page-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/page-access")>()),
  fetchPageAccess: (location: string) => fetchPageAccess(location),
  fetchPageAccessPreview: (location: string, subject: PageAccessSubject) =>
    fetchPageAccessPreview(location, subject),
}));

vi.mock("@/components/fields/autocomplete/autocomplete", () => ({
  Autocomplete: ({ placeholder }: { placeholder: string }) => (
    <button type="button">{placeholder}</button>
  ),
}));

afterEach(() => {
  cleanup();
  fetchPageAccess.mockReset();
  fetchPageAccessPreview.mockReset();
});

const access: PageAccess = {
  path: "/shipments/:id",
  name: "",
  operations: ["ShipmentDetail"],
  requires: [],
  endpoints: [],
  kinds: [
    {
      resource: "customer",
      resourceName: "Customer",
      use: "DECLARED",
      canRead: false,
      withheld: 1,
      fields: [{ name: "creditLimit", inView: false, decision: "WITHHELD_KIND" }],
    },
  ],
  withheld: 1,
  blocked: false,
};

function renderPanel(location: string, canPreview: boolean) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <Sheet open>
        <SheetContent>
          <PageAccessPanel location={location} canPreview={canPreview} />
        </SheetContent>
      </Sheet>
    </QueryClientProvider>,
  );
}

describe("PageAccessPanel", () => {
  it("describes the reader's own access to the screen at the location", async () => {
    fetchPageAccess.mockResolvedValue(access);
    renderPanel("/shipments/shp_1", false);

    expect(await screen.findByText("creditLimit")).toBeInTheDocument();
    expect(fetchPageAccess).toHaveBeenCalledWith("/shipments/shp_1");
    expect(fetchPageAccessPreview).not.toHaveBeenCalled();
    expect(screen.getByText("/shipments/:id")).toBeInTheDocument();
    expect(screen.queryByText("View as")).toBeNull();
  });

  it("offers an administrator to view the screen as a role", async () => {
    fetchPageAccess.mockResolvedValue(access);
    renderPanel("/shipments/shp_1", true);

    expect(await screen.findByText("creditLimit")).toBeInTheDocument();
    expect(screen.getByText("View as")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "You, with your own roles" })).toBeInTheDocument();
  });

  it("says when no screen is at the location", async () => {
    fetchPageAccess.mockResolvedValue(null);
    renderPanel("/nowhere", false);

    expect(
      await screen.findByText("This location is not a screen the page access manifest describes."),
    ).toBeInTheDocument();
  });

  it("says when the access could not be loaded", async () => {
    fetchPageAccess.mockRejectedValue(new Error("boom"));
    renderPanel("/shipments", false);

    expect(await screen.findByText("Page access could not be loaded.")).toBeInTheDocument();
  });
});
