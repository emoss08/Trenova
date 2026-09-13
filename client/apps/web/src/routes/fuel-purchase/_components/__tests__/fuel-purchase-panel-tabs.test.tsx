import type { FuelPurchaseRow } from "@/lib/graphql/fuel-purchase";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { useController, type Control } from "react-hook-form";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

// Receipts hang off the purchase, not off the fields the purchase is made of.
// They belong on their own tab, the way the tractor panel keeps documents, and
// the details tab has to stay a plain form that still saves.

type PickerProps = { control: Control; name: string; label: string };

function pickerStub() {
  return function Picker({ control, name, label }: PickerProps) {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  };
}

vi.mock("@/components/autocomplete-fields", () => ({
  TractorAutocompleteField: pickerStub(),
  WorkerAutocompleteField: pickerStub(),
  FuelCardAutocompleteField: pickerStub(),
  IftaJurisdictionAutocompleteField: pickerStub(),
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: pickerStub(),
}));

vi.mock("@/components/fields/date-field/datetime-field", () => ({
  AutoCompleteDateTimeField: ({ control, name, label }: PickerProps) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        type="number"
        value={(field.value as number) || ""}
        onChange={(event) => field.onChange(Number(event.target.value))}
      />
    );
  },
}));

vi.mock("@/components/info-popover", () => ({ InfoPopover: () => null }));

vi.mock("@/components/documents/documents-tab", () => ({
  default: ({ resourceType, resourceId }: { resourceType: string; resourceId: string }) => (
    <div data-testid="documents">
      {resourceType}:{resourceId}
    </div>
  ),
}));

const updateFuelPurchase = vi.fn(async () => ({}));

vi.mock("@/lib/graphql/fuel-purchase", () => ({
  FUEL_PURCHASE_LIST_KEY: "fuel-purchase-list",
  createFuelPurchase: vi.fn(async () => ({})),
  updateFuelPurchase: (...args: unknown[]) => updateFuelPurchase(...(args as [])),
}));

const { FuelPurchasePanel } = await import("../fuel-purchase-panel");

const PURCHASED_AT = 1_760_000_000;

const row = {
  id: "fpur_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  tractorId: "trk_118",
  workerId: "wrk_1",
  jurisdictionId: "ij_tx",
  fuelCardId: "fcrd_1",
  cardLastFour: "4821",
  purchasedAt: PURCHASED_AT,
  vendor: "Love's #412",
  vendorCity: "Amarillo",
  fuelType: "Diesel",
  quantity: "120.500",
  quantityUnit: "Gallon",
  gallons: "120.500",
  unitPrice: "3.8990",
  totalAmount: "469.83",
  currencyCode: "USD",
  odometer: 412_113,
  transactionReference: "EFS-77812",
  source: "Manual",
  importBatchId: null,
  taxPaid: true,
  notes: null,
  createdById: "usr_1",
  version: 3,
  createdAt: 1,
  updatedAt: 2,
  tractor: { id: "trk_118", code: "118" },
  worker: { id: "wrk_1", wholeName: "Dana Ruiz", firstName: "Dana", lastName: "Ruiz" },
  jurisdiction: { id: "ij_tx", countryCode: "US", code: "TX", name: "Texas" },
  fuelCard: { id: "fcrd_1", provider: "EFS", lastFour: "4821", label: "Unit 118 card" },
} as unknown as FuelPurchaseRow;

function newQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
}

function renderPanel() {
  return render(
    <NuqsTestingAdapter>
      <QueryClientProvider client={newQueryClient()}>
        <FuelPurchasePanel open mode="edit" row={row} onOpenChange={() => {}} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

// The tab strip measures itself and drops what does not fit into a "More" menu.
// Nothing has a width in the test DOM, so give the strip room to show them all.
let clientWidth: PropertyDescriptor | undefined;

beforeAll(() => {
  clientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientWidth");
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get: () => 800,
  });
});

afterAll(() => {
  if (clientWidth) {
    Object.defineProperty(HTMLElement.prototype, "clientWidth", clientWidth);
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, "clientWidth");
  }
});

afterEach(() => {
  cleanup();
  updateFuelPurchase.mockClear();
});

describe("FuelPurchasePanel edit tabs", () => {
  it("keeps receipts off the details tab and behind a Documents tab", async () => {
    const user = userEvent.setup();
    renderPanel();

    expect(await screen.findByRole("tab", { name: /details/i })).toBeInTheDocument();
    expect(screen.getByLabelText("Tractor")).toHaveValue("trk_118");
    expect(screen.queryByTestId("documents")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /documents/i }));

    expect(await screen.findByTestId("documents")).toHaveTextContent("fuel_purchase:fpur_1");
  });

  it("still saves the details tab through the purchase mutation", async () => {
    const user = userEvent.setup();
    renderPanel();

    const vendor = await waitFor(() => {
      const element = document.getElementById("input-vendor");
      if (!(element instanceof HTMLInputElement)) {
        throw new Error("input-vendor not rendered");
      }
      return element;
    });
    await user.clear(vendor);
    await user.type(vendor, "Pilot #88");

    await user.click(screen.getByRole("button", { name: /save & close/i }));

    await waitFor(() => expect(updateFuelPurchase).toHaveBeenCalled());
    const [id, version, input] = updateFuelPurchase.mock.calls[0] as unknown as [
      string,
      number,
      Record<string, unknown>,
    ];
    expect(id).toBe("fpur_1");
    expect(version).toBe(3);
    expect(input).toMatchObject({ vendor: "Pilot #88", tractorId: "trk_118" });
  });
});
