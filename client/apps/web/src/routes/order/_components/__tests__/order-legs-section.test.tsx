import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { toast } from "sonner";
import { OrderLegsSection } from "../order-legs-section";

const mocks = vi.hoisted(() => ({
  fetchOrderDetail: vi.fn(),
  createInvoiceFromShipments: vi.fn(),
  createInvoiceFromOrder: vi.fn(),
  detachOrderShipment: vi.fn(),
}));

vi.mock("@/lib/graphql/order", () => ({
  fetchOrderDetail: mocks.fetchOrderDetail,
  createInvoiceFromShipments: mocks.createInvoiceFromShipments,
  createInvoiceFromOrder: mocks.createInvoiceFromOrder,
  detachOrderShipment: mocks.detachOrderShipment,
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("../add-leg-dialog", () => ({ AddLegDialog: () => null }));
vi.mock("../use-order-invalidation", () => ({
  useOrderInvalidation: () => vi.fn(),
  useOrderInvoiceInvalidation: () => vi.fn(),
}));

function leg(id: string, proNumber: string, status: string) {
  return {
    id,
    proNumber,
    status,
    freightChargeAmount: "100.00",
    totalChargeAmount: "100.00",
  };
}

function Harness() {
  const form = useForm({ defaultValues: { id: "ord_1" } });
  return (
    <FormProvider {...form}>
      <OrderLegsSection />
    </FormProvider>
  );
}

function renderSection(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.createInvoiceFromShipments.mockResolvedValue({ id: "inv_1", number: "INV-1" });
});

afterEach(cleanup);

describe("OrderLegsSection leg selection", () => {
  it("invoices only the checked legs", async () => {
    const user = userEvent.setup();
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [
        leg("shp_1", "PRO-1", "Completed"),
        leg("shp_2", "PRO-2", "Completed"),
        leg("shp_3", "PRO-3", "Completed"),
      ],
    });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-1" }));
    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-3" }));

    const button = screen.getByRole("button", { name: /Create invoice from 2 legs/ });
    await user.click(button);

    await waitFor(() =>
      expect(mocks.createInvoiceFromShipments).toHaveBeenCalledWith(["shp_1", "shp_3"], undefined),
    );
    expect(mocks.createInvoiceFromOrder).not.toHaveBeenCalled();
  });

  it("uses the shipments path even when every leg is selected", async () => {
    const user = userEvent.setup();
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [leg("shp_1", "PRO-1", "Completed"), leg("shp_2", "PRO-2", "ReadyToInvoice")],
    });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select all invoiceable legs" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 2 legs/ }));

    await waitFor(() =>
      expect(mocks.createInvoiceFromShipments).toHaveBeenCalledWith(["shp_1", "shp_2"], undefined),
    );
    expect(mocks.createInvoiceFromOrder).not.toHaveBeenCalled();
  });

  it("disables selection for a leg that cannot be invoiced and skips it in select-all", async () => {
    const user = userEvent.setup();
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [leg("shp_1", "PRO-1", "Completed"), leg("shp_2", "PRO-2", "InTransit")],
    });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    // Base UI renders the checkbox as a span with role="checkbox", so the
    // disabled state is aria-disabled rather than a native disabled attribute.
    const blocked = screen.getByRole("checkbox", { name: "Leg PRO-2 cannot be invoiced" });
    expect(blocked).toHaveAttribute("aria-disabled", "true");

    await user.click(screen.getByRole("checkbox", { name: "Select all invoiceable legs" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 1 leg/ }));

    // A lone leg is confirmed first because it produces a standalone invoice.
    await user.click(screen.getByRole("button", { name: "Create invoice" }));

    await waitFor(() => expect(mocks.createInvoiceFromShipments).toHaveBeenCalledWith(["shp_1"], undefined));
  });

  it("confirms before billing a single leg on its own invoice", async () => {
    const user = userEvent.setup();
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [leg("shp_1", "PRO-1", "Completed"), leg("shp_2", "PRO-2", "Completed")],
    });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-1" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 1 leg/ }));

    expect(screen.getByText("Bill this leg on its own invoice?")).toBeInTheDocument();
    expect(mocks.createInvoiceFromShipments).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Create invoice" }));
    await waitFor(() => expect(mocks.createInvoiceFromShipments).toHaveBeenCalledWith(["shp_1"], undefined));
  });

  it("keeps the invoice button disabled until something is selected", async () => {
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [leg("shp_1", "PRO-1", "Completed")],
    });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    expect(screen.getByRole("button", { name: /Create invoice/ })).toBeDisabled();
  });
});

describe("OrderLegsSection statement cadence guard", () => {
  /** What the API returns when the customer is on a periodic statement. */
  function cadenceRefusal() {
    return {
      getFieldErrors: (field?: string) =>
        [
          {
            field: "offCycleReason",
            message: "Acme Freight is billed on a monthly statement.",
          },
        ].filter((e) => field === undefined || e.field === field),
    };
  }

  beforeEach(() => {
    mocks.fetchOrderDetail.mockResolvedValue({
      status: "InTransit",
      currencyCode: "USD",
      legs: [leg("shp_1", "PRO-1", "Completed"), leg("shp_2", "PRO-2", "Completed")],
    });
  });

  // The refusal is a prompt, not a failure: it must open the reason dialog rather
  // than surface as an error toast the biller can only give up on.
  it("asks for a reason instead of failing when the customer is on a statement", async () => {
    const user = userEvent.setup();
    mocks.createInvoiceFromShipments.mockRejectedValueOnce(cadenceRefusal());

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-1" }));
    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-2" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 2 legs/ }));

    expect(
      await screen.findByText("Acme Freight is billed on a monthly statement."),
    ).toBeInTheDocument();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it("retries with the reason the biller typed", async () => {
    const user = userEvent.setup();
    mocks.createInvoiceFromShipments
      .mockRejectedValueOnce(cadenceRefusal())
      .mockResolvedValueOnce({ id: "inv_1", number: "INV-1" });

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-1" }));
    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-2" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 2 legs/ }));

    await screen.findByText("Acme Freight is billed on a monthly statement.");
    await user.type(
      screen.getByLabelText(/reason for invoicing outside the statement/i),
      "Billing to the broker",
    );
    await user.click(screen.getByRole("button", { name: /invoice anyway/i }));

    await waitFor(() =>
      expect(mocks.createInvoiceFromShipments).toHaveBeenLastCalledWith(
        ["shp_1", "shp_2"],
        "Billing to the broker",
      ),
    );
  });

  // A real failure still has to read as one, or the biller types a reason at a
  // problem no reason can fix.
  it("still reports an ordinary failure as an error", async () => {
    const user = userEvent.setup();
    mocks.createInvoiceFromShipments.mockRejectedValueOnce(new Error("network down"));

    renderSection(<Harness />);
    await screen.findByText("PRO-1");

    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-1" }));
    await user.click(screen.getByRole("checkbox", { name: "Select leg PRO-2" }));
    await user.click(screen.getByRole("button", { name: /Create invoice from 2 legs/ }));

    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    expect(
      screen.queryByLabelText(/reason for invoicing outside the statement/i),
    ).not.toBeInTheDocument();
  });
});
