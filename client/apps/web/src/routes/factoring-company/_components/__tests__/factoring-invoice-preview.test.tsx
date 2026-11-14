import type { FactoringCompany } from "@/types/factoring-company";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  FactoringInvoicePreview,
  type FactorState,
  type InvoiceRegion,
} from "../factoring-invoice-preview";

const mocks = vi.hoisted(() => ({ organizations: vi.fn(), detail: vi.fn() }));

vi.mock("@/lib/queries/user", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/user")>()),
  userOrganization: {
    all: () => ({ queryKey: ["userOrganization", "all"], queryFn: mocks.organizations }),
  },
}));
vi.mock("@/lib/queries/organization", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/organization")>()),
  organization: {
    detail: (id: string) => ({ queryKey: ["organization", "detail", id], queryFn: mocks.detail }),
    branding: (id: string) => ({
      queryKey: ["organization", "branding", id],
      queryFn: async () => ({ logoUrl: "", iconUrl: "" }),
    }),
  },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const FACTOR: FactoringCompany = {
  status: "Active",
  code: "BLUE",
  name: "Bluewater Receivables Finance",
  addressLine1: "PO Box 29237",
  addressLine2: "",
  city: "Jacksonville",
  stateId: "us_fl",
  postalCode: "32202",
  email: "funding@bluewater.example",
  phoneNumber: "+17338043532",
  remittanceInstructions: "ACH routing 567272913\nAccount ending 0395",
  noticeOfAssignment: "",
  notes: "Team only",
  nonRecourse: false,
};

const FLORIDA: FactorState = { abbreviation: "FL", countryName: "United States" };

function Preview({
  values = FACTOR,
  active = new Set<InvoiceRegion>(),
}: {
  values?: FactoringCompany;
  active?: ReadonlySet<InvoiceRegion>;
}) {
  const form = useForm<FactoringCompany>({ defaultValues: values });
  return <FactoringInvoicePreview control={form.control} state={FLORIDA} active={active} />;
}

function renderPreview(props?: Parameters<typeof Preview>[0]) {
  mocks.organizations.mockResolvedValue([
    {
      id: "org_1",
      name: "Trenova Logistics",
      city: "Los Angeles",
      state: "CA",
      logoUrl: null,
      isDefault: true,
      isCurrent: true,
    },
  ]);
  mocks.detail.mockResolvedValue({
    id: "org_1",
    name: "Trenova Logistics",
    addressLine1: "1 Market Street",
    addressLine2: null,
    city: "Los Angeles",
    postalCode: "90001",
    state: { id: "us_ca", name: "California", abbreviation: "CA", countryName: "United States" },
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Preview {...props} />
    </QueryClientProvider>,
  );
}

describe("FactoringInvoicePreview", () => {
  it("prints the remit-to the way the invoice does: address, instructions, then email and phone", () => {
    renderPreview();

    const remitTo = screen.getByText("[ Remit To ]").parentElement as HTMLElement;
    expect(within(remitTo).getByText("Bluewater Receivables Finance")).toBeTruthy();
    expect(within(remitTo).getByText("Jacksonville, FL 32202")).toBeTruthy();
    expect(within(remitTo).getByText("United States")).toBeTruthy();
    expect(within(remitTo).getByText("ACH routing 567272913")).toBeTruthy();
    expect(within(remitTo).getByText("Email: funding@bluewater.example")).toBeTruthy();
    expect(within(remitTo).getByText("Phone: +17338043532")).toBeTruthy();
    expect(screen.queryByText("Team only")).toBeNull();
  });

  it("prints the standard notice naming the factor when it has no wording of its own", () => {
    renderPreview();

    expect(
      screen.getByText(
        "NOTICE OF ASSIGNMENT: This invoice has been assigned to, and is payable only to, Bluewater Receivables Finance. Payment to any other party will not discharge your obligation. Remit payment to the address below and direct any dispute to Bluewater Receivables Finance.",
      ),
    ).toBeTruthy();
  });

  it("prints the factor's own notice word for word when it has one", () => {
    renderPreview({ values: { ...FACTOR, noticeOfAssignment: "Pay Bluewater only." } });

    expect(screen.getByText("Pay Bluewater only.")).toBeTruthy();
    expect(screen.queryByText(/This invoice has been assigned to/)).toBeNull();
  });

  it("heads the invoice with your organization's name and address", async () => {
    renderPreview();

    await waitFor(() =>
      expect(
        screen.getByText("1 Market Street · Los Angeles, CA 90001 · United States"),
      ).toBeTruthy(),
    );
    expect(screen.getAllByText("Trenova Logistics").length).toBeGreaterThan(0);
  });

  it("marks only the parts of the invoice the section in focus sets", () => {
    const { container } = renderPreview({ active: new Set<InvoiceRegion>(["contact"]) });

    const marked = [...container.querySelectorAll("[data-marked]")];
    expect(marked).toHaveLength(2);
    expect(marked[0]?.textContent).toBe("Email: funding@bluewater.examplePhone: +17338043532");
    expect(marked[1]?.textContent).toContain("[ How to pay ]");
  });
});
