import type { RoleResourceGuide } from "@/lib/graphql/page-access";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ResourceGuideLine, useRoleResourceGuides } from "../role-resource-guide";

const fetchRoleResourceGuides =
  vi.fn<(resources: readonly string[], maxSensitivity: string) => Promise<RoleResourceGuide[]>>();

vi.mock("@/lib/graphql/page-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/page-access")>()),
  fetchRoleResourceGuides: (resources: readonly string[], maxSensitivity: string) =>
    fetchRoleResourceGuides(resources, maxSensitivity),
}));

afterEach(() => {
  cleanup();
  fetchRoleResourceGuides.mockReset();
});

// Written from schema/page_access.graphqls: RoleResourceGuide.
const customer: RoleResourceGuide = {
  resource: "customer",
  name: "Customer",
  granted:
    "Can see every field of Customer records, on their own screens and wherever another record shows one.",
  withheld:
    "Customer records show only their code, name and address wherever a record they can read links to one (Shipment, Invoice and Order); the rest needs View on Customer.",
  ceiling:
    "Without View on Customer, its name stay withheld there too: they rank above this role's sensitivity ceiling.",
  screens: [
    { path: "/shipments", name: "Shipments", use: "DECLARED" },
    { path: "/billing/invoices/:id", name: "", use: "REFERENCE" },
  ],
};

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

describe("ResourceGuideLine", () => {
  it("says what the role sees with the resource's read", () => {
    render(<ResourceGuideLine guide={customer} reads isPending={false} />, { wrapper });

    expect(screen.getByText(customer.granted)).toBeInTheDocument();
    expect(screen.queryByText(customer.withheld)).toBeNull();
    expect(screen.queryByText(customer.ceiling!)).toBeNull();
  });

  it("says what other records still show without it, and what the ceiling withholds", () => {
    render(<ResourceGuideLine guide={customer} reads={false} isPending={false} />, { wrapper });

    expect(screen.getByText(customer.withheld)).toBeInTheDocument();
    expect(screen.getByText(customer.ceiling!)).toBeInTheDocument();
    expect(screen.queryByText(customer.granted)).toBeNull();
  });

  it("lists the screens the grant affects, linking those that need no record", async () => {
    render(<ResourceGuideLine guide={customer} reads={false} isPending={false} />, { wrapper });

    await userEvent.click(screen.getByRole("button", { name: "2 screens this affects" }));

    const list = await screen.findByRole("list", { name: "Screens this affects" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByRole("link", { name: "Shipments" })).toHaveAttribute(
      "href",
      "/shipments",
    );
    expect(rows[0]).toHaveTextContent("Asks for more");
    expect(within(rows[1]).queryByRole("link")).toBeNull();
    expect(rows[1]).toHaveTextContent("/billing/invoices/:id");
    expect(rows[1]).toHaveTextContent("Reference view");
  });

  it("draws nothing for a resource the server has not worded once it answered", () => {
    const { container } = render(
      <ResourceGuideLine guide={undefined} reads={false} isPending={false} />,
      { wrapper },
    );
    expect(container).toBeEmptyDOMElement();
  });
});

describe("useRoleResourceGuides", () => {
  it("asks for each open category's resources at most 64 at a time, only once it is open", async () => {
    fetchRoleResourceGuides.mockImplementation(async (resources) =>
      resources.map((resource) => ({ ...customer, resource })),
    );
    const billing = Array.from({ length: 70 }, (_, i) => `billing_${i}`);
    const dispatch = ["location", "location_category"];

    const { result, rerender } = renderHook(({ open }) => useRoleResourceGuides(open, "internal"), {
      wrapper,
      initialProps: { open: [] as string[][] },
    });
    expect(fetchRoleResourceGuides).not.toHaveBeenCalled();

    rerender({ open: [billing] });
    await waitFor(() => expect(result.current.guides.size).toBe(70));
    expect(fetchRoleResourceGuides.mock.calls.map(([batch]) => batch.length)).toEqual([64, 6]);
    expect(fetchRoleResourceGuides.mock.calls[0][1]).toBe("internal");

    rerender({ open: [billing, dispatch] });
    await waitFor(() => expect(result.current.guides.size).toBe(72));
    expect(fetchRoleResourceGuides.mock.calls.map(([batch]) => batch)).toEqual([
      billing.slice(0, 64),
      billing.slice(64),
      dispatch,
    ]);
    expect(result.current.isPending).toBe(false);
  });
});
