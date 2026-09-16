import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { reassignChargeResultSchema } from "@trenova/shared/types/billing-queue";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import { useController, type Control, type FieldValues } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueReassignChargeDialog } from "../billing-queue-reassign-charge-dialog";
import { queueItem } from "./split-shipment-fixture";

const { reassignCharge, toastSuccess } = vi.hoisted(() => ({
  reassignCharge: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { reassignCharge } },
}));
vi.mock("sonner", () => ({ toast: { success: toastSuccess, error: vi.fn() } }));

const CUSTOMERS = [
  { id: "cus_acme", label: "Acme Manufacturing", description: null, meta: { code: "ACME" } },
  { id: "cus_peak", label: "Peak Distributing", description: null, meta: { code: "PEAK" } },
];

function CustomerPickerStub({
  control,
  name,
  label,
  placeholder,
  onOptionChange,
}: {
  control: Control<FieldValues>;
  name: string;
  label?: string;
  placeholder?: string;
  onOptionChange?: (option: (typeof CUSTOMERS)[number] | null) => void;
}) {
  const { field } = useController({ control, name });
  return (
    <select
      aria-label={label ?? placeholder ?? name}
      value={field.value ?? ""}
      onChange={(event) => {
        const option = CUSTOMERS.find((c) => c.id === event.target.value) ?? null;
        field.onChange(option?.id ?? "");
        onOptionChange?.(option);
      }}
    >
      <option value="">{placeholder}</option>
      {CUSTOMERS.map((c) => (
        <option key={c.id} value={c.id}>
          {c.label}
        </option>
      ))}
    </select>
  );
}

vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: CustomerPickerStub,
}));

afterEach(cleanup);
beforeEach(() => vi.clearAllMocks());

function renderDialog({
  payer,
  lineIndex,
  other = false,
  onUrlUpdate = vi.fn(),
}: {
  payer: "acme" | "peak";
  lineIndex: number;
  other?: boolean;
  onUrlUpdate?: OnUrlUpdateFunction;
}) {
  const item = queueItem({ payer });
  const share = item.payerShare!;
  const line = (other ? share.otherPayerLines : share.lines)[lineIndex];
  const onOpenChange = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <NuqsTestingAdapter searchParams={`?item=${item.id}`} onUrlUpdate={onUrlUpdate}>
      <QueryClientProvider client={client}>
        <BillingQueueReassignChargeDialog
          open
          onOpenChange={onOpenChange}
          item={item}
          line={line}
        />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
  return { item, onOpenChange };
}

describe("BillingQueueReassignChargeDialog", () => {
  // Peak's item shows detention under other payers. Giving it to Peak whole is
  // one pick; the body names the charge by kind and id, as the endpoint needs.
  it("gives a charge to one payer and sends the contract body", async () => {
    const user = userEvent.setup();
    reassignCharge.mockResolvedValue(
      reassignChargeResultSchema.parse({
        item: { ...queueItem({ payer: "peak" }), payerShare: null, shipment: undefined },
        items: [],
        createdItemIds: [],
        canceledItemIds: ["bqi_acme"],
      }),
    );
    const { onOpenChange } = renderDialog({ payer: "peak", lineIndex: 0, other: true });

    await user.selectOptions(screen.getByRole("combobox", { name: "Bill to" }), "cus_peak");
    await user.click(screen.getByRole("button", { name: "Save payers" }));

    await waitFor(() => expect(reassignCharge).toHaveBeenCalledTimes(1));
    expect(reassignCharge).toHaveBeenCalledWith("bqi_peak", {
      chargeKind: "Accessorial",
      additionalChargeId: "ac_det",
      allocations: [
        { billToCustomerId: "cus_peak", method: "Percent", percent: "100", amount: null },
      ],
    });
    expect(toastSuccess).toHaveBeenCalledWith("Detention Fee moved. 1 queue item canceled.");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  // Freight is named by kind alone, and a row already dividing it keeps its id
  // so the server updates it rather than inserting a duplicate payer.
  it("sends the freight split with the ids of the rows it already has", async () => {
    const user = userEvent.setup();
    reassignCharge.mockResolvedValue(
      reassignChargeResultSchema.parse({
        item: { ...queueItem({ payer: "acme" }), payerShare: null, shipment: undefined },
        items: [],
        createdItemIds: [],
        canceledItemIds: [],
      }),
    );
    renderDialog({ payer: "acme", lineIndex: 0 });

    await user.click(screen.getByRole("button", { name: "Save payers" }));

    await waitFor(() => expect(reassignCharge).toHaveBeenCalledTimes(1));
    expect(reassignCharge).toHaveBeenCalledWith("bqi_acme", {
      chargeKind: "Freight",
      allocations: [
        {
          id: "chal_peak",
          billToCustomerId: "cus_peak",
          method: "Amount",
          percent: null,
          amount: "1500",
        },
        {
          id: "chal_acme",
          billToCustomerId: "cus_acme",
          method: "Amount",
          percent: null,
          amount: "1350",
        },
      ],
    });
  });

  // Giving Acme's only charge away leaves Acme paying nothing, so its item is
  // canceled; the queue moves to the shipment's remaining item.
  it("moves the selection when the item on screen is canceled", async () => {
    const user = userEvent.setup();
    const onUrlUpdate = vi.fn();
    reassignCharge.mockResolvedValue(
      reassignChargeResultSchema.parse({
        item: {
          ...queueItem({ payer: "acme" }),
          status: "Canceled",
          payerShare: null,
          shipment: undefined,
        },
        items: [{ ...queueItem({ payer: "peak" }), payerShare: null, shipment: undefined }],
        createdItemIds: [],
        canceledItemIds: ["bqi_acme"],
      }),
    );
    renderDialog({ payer: "acme", lineIndex: 1, onUrlUpdate });

    await user.selectOptions(screen.getByRole("combobox", { name: "Bill to" }), "cus_peak");
    await user.click(screen.getByRole("button", { name: "Save payers" }));

    await waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    const last = onUrlUpdate.mock.calls.at(-1)?.[0];
    expect(last.searchParams.get("item")).toBe("bqi_peak");
  });
});
