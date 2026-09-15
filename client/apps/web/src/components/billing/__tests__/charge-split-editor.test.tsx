import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import {
  FormProvider,
  useController,
  useForm,
  type Control,
  type FieldValues,
} from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChargeSplitEditor } from "../charge-split-editor";

const CUSTOMERS = [
  { id: "cus_intel", label: "Intel", description: null, meta: { code: "INTEL" } },
  { id: "cus_amd", label: "AMD", description: null, meta: { code: "AMD" } },
];

/**
 * The real picker fetches options over GraphQL. This stand-in exposes the same
 * contract the editor relies on: the form value is the customer id and the
 * chosen option is handed back through `onOptionChange`.
 */
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
    <label>
      {label ?? placeholder}
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
    </label>
  );
}

vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: CustomerPickerStub,
}));

afterEach(cleanup);

function Harness({
  allocations,
  chargeAmount = 500,
  percentOnly = false,
  onValues,
}: {
  allocations: ChargeAllocation[];
  chargeAmount?: number | null;
  percentOnly?: boolean;
  onValues: (values: FieldValues) => void;
}) {
  const form = useForm<FieldValues>({ defaultValues: { allocations } });
  return (
    <FormProvider {...form}>
      <ChargeSplitEditor
        control={form.control}
        name="allocations"
        chargeAmount={chargeAmount}
        percentOnly={percentOnly}
        defaultPayer={{ id: "cus_shipper", label: "Shipper Co" }}
      />
      <button type="button" onClick={() => onValues(form.getValues())}>
        read
      </button>
    </FormProvider>
  );
}

describe("ChargeSplitEditor", () => {
  it("starts in the single bill-to form and names the shipment payer as the default", () => {
    render(<Harness allocations={[]} onValues={vi.fn()} />);

    expect(screen.getByTestId("charge-split-simple")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Bill to" })).toBeInTheDocument();
    expect(screen.getByText("Same as shipment (Shipper Co)")).toBeInTheDocument();
  });

  // Picking a customer bills the whole charge to them: one 100% row, with the
  // option snapshot kept so the list can show the payer's name without a fetch.
  it("writes a single 100% allocation when one payer is chosen", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(<Harness allocations={[]} onValues={onValues} />);

    await user.selectOptions(screen.getByRole("combobox", { name: "Bill to" }), "cus_intel");
    await user.click(screen.getByText("read"));

    expect(onValues).toHaveBeenLastCalledWith({
      allocations: [
        expect.objectContaining({
          billToCustomerId: "cus_intel",
          method: "Percent",
          percent: 100,
          billToCustomer: { id: "cus_intel", name: "Intel", code: "INTEL" },
        }),
      ],
    });
  });

  it("clears every allocation when the single payer is cleared", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 100 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Bill to" }), "");
    await user.click(screen.getByText("read"));

    expect(onValues).toHaveBeenLastCalledWith({ allocations: [] });
  });

  it("opens the row editor seeded with the default payer and reports the remainder", async () => {
    const user = userEvent.setup();
    render(<Harness allocations={[]} onValues={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Split between payers" }));

    expect(screen.getByTestId("charge-split-rows")).toBeInTheDocument();
    expect(screen.getAllByRole("combobox", { name: /payer/i })).toHaveLength(2);
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "Allocated 100% · 0% remaining",
    );
  });

  it("opens straight into the row editor for an existing multi-payer split", () => {
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 25 } as ChargeAllocation,
        ]}
        onValues={vi.fn()}
      />,
    );

    expect(screen.getByTestId("charge-split-rows")).toBeInTheDocument();
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "Allocated 85% · 15% remaining",
    );
  });

  it("flags an over-allocated amount split against the charge total", () => {
    render(
      <Harness
        chargeAmount={500}
        allocations={[
          { billToCustomerId: "cus_intel", method: "Amount", amount: 400 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Amount", amount: 250 } as ChargeAllocation,
        ]}
        onValues={vi.fn()}
      />,
    );

    const remainder = screen.getByTestId("charge-split-remainder");
    expect(remainder).toHaveTextContent("Allocated $650.00 · -$150.00 remaining");
    expect(remainder).toHaveClass("text-destructive");
  });

  // A percentage-of-freight accessorial has no dollar total on the client, so
  // an amount split cannot be checked and is not offered.
  it("hides the method switch when only percentages are allowed", () => {
    render(
      <Harness
        percentOnly
        chargeAmount={null}
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 50 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 50 } as ChargeAllocation,
        ]}
        onValues={vi.fn()}
      />,
    );

    expect(screen.queryByRole("combobox", { name: "Split method" })).not.toBeInTheDocument();
  });

  it("returns to a single payer and drops the rows", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 50 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 50 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Bill to one payer" }));
    await user.click(screen.getByText("read"));

    expect(screen.getByTestId("charge-split-simple")).toBeInTheDocument();
    expect(onValues).toHaveBeenLastCalledWith({ allocations: [] });
  });
});
