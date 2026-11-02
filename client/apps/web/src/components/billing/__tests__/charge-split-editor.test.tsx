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
  chargeAmount?: number | string | null;
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

  // A split starts in half between whoever pays now and a payer still to choose, each row
  // showing the dollars its percent bills.
  it("opens split in half between the shipment's payer and a payer still to choose", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(<Harness allocations={[]} onValues={onValues} />);

    await user.click(screen.getByRole("button", { name: "Split between payers" }));
    await user.click(screen.getByText("read"));

    expect(screen.getAllByRole("combobox", { name: /payer/i })).toHaveLength(2);
    expect(screen.getByTestId("charge-split-share-0")).toHaveTextContent("$250.00");
    expect(screen.getByTestId("charge-split-share-1")).toHaveTextContent("$250.00");
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "The full $500.00 is covered",
    );
    expect(onValues).toHaveBeenLastCalledWith({
      allocations: [
        expect.objectContaining({
          billToCustomerId: "cus_shipper",
          method: "Percent",
          percent: 50,
        }),
        expect.objectContaining({ billToCustomerId: "", method: "Percent", percent: 50 }),
      ],
    });
  });

  it("opens straight into the rows for an existing split and says what is left", () => {
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
    expect(screen.getByTestId("charge-split-share-0")).toHaveTextContent("$300.00");
    expect(screen.getByTestId("charge-split-share-1")).toHaveTextContent("$125.00");
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent("15% left to assign");
  });

  it("fills what is left into the last row when every row has a value", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 25 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Fill" }));
    await user.click(screen.getByText("read"));

    expect(onValues).toHaveBeenLastCalledWith({
      allocations: [
        expect.objectContaining({ billToCustomerId: "cus_intel", percent: 60 }),
        expect.objectContaining({ billToCustomerId: "cus_amd", percent: 40 }),
      ],
    });
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "The full $500.00 is covered",
    );
    expect(screen.queryByRole("button", { name: "Fill" })).not.toBeInTheDocument();
  });

  it("gives a new payer whatever is left", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 25 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Add payer" }));
    await user.click(screen.getByText("read"));

    expect(onValues.mock.lastCall?.[0].allocations).toHaveLength(3);
    expect(onValues.mock.lastCall?.[0].allocations[2]).toEqual(
      expect.objectContaining({ billToCustomerId: "", method: "Percent", percent: 15 }),
    );
  });

  // Thirds are cut to the hundredth and the last row takes the rest, so they add up exactly.
  it("splits evenly to the hundredth, the last row taking the rest", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 25 } as ChargeAllocation,
          { billToCustomerId: "cus_shipper", method: "Percent", percent: null } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Split evenly" }));
    await user.click(screen.getByText("read"));

    expect(
      onValues.mock.lastCall?.[0].allocations.map((row: ChargeAllocation) => row.percent),
    ).toEqual([33.33, 33.33, 33.34]);
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "The full $500.00 is covered",
    );
  });

  it("flags an over-allocated amount split by how far over it is", () => {
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
    expect(remainder).toHaveTextContent("$150.00 over");
    expect(remainder).toHaveClass("text-destructive");
    expect(screen.getByTestId("charge-split-share-0")).toHaveTextContent("80%");
    expect(screen.getByTestId("charge-split-share-1")).toHaveTextContent("50%");
    expect(screen.queryByRole("button", { name: "Fill" })).not.toBeInTheDocument();
  });

  it("measures amount rows exactly against a decimal total the rules module computed", () => {
    render(
      <Harness
        chargeAmount="1000.01"
        allocations={[
          { billToCustomerId: "cus_intel", method: "Amount", amount: 600.01 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Amount", amount: 400 } as ChargeAllocation,
        ]}
        onValues={vi.fn()}
      />,
    );

    const remainder = screen.getByTestId("charge-split-remainder");
    expect(remainder).toHaveTextContent("The full $1,000.01 is covered");
    expect(remainder).not.toHaveClass("text-destructive");
  });

  it("shows only what is assigned, and offers no even split, for a charge of unknown total", () => {
    render(
      <Harness
        chargeAmount={null}
        allocations={[
          { billToCustomerId: "cus_intel", method: "Amount", amount: 400 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Amount", amount: 250 } as ChargeAllocation,
        ]}
        onValues={vi.fn()}
      />,
    );

    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent("Assigned $650.00");
    expect(screen.getByRole("button", { name: "Split evenly" })).toBeDisabled();
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

    expect(screen.queryByRole("radiogroup", { name: "Split method" })).not.toBeInTheDocument();
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent("All 100% is assigned");
  });

  it("keeps the split when switching from percent to amount", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        allocations={[
          { billToCustomerId: "cus_intel", method: "Percent", percent: 60 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Percent", percent: 40 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("radio", { name: "By amount" }));
    await user.click(screen.getByText("read"));

    expect(onValues).toHaveBeenLastCalledWith({
      allocations: [
        expect.objectContaining({ method: "Amount", amount: 300, percent: null }),
        expect.objectContaining({ method: "Amount", amount: 200, percent: null }),
      ],
    });
  });

  // $100 three ways is 33.333333% each at six places, which totals 99.999999; the last row
  // takes the rest so the converted split still adds up to exactly 100.
  it("keeps a complete split exact when switching from amount to percent", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(
      <Harness
        chargeAmount={300}
        allocations={[
          { billToCustomerId: "cus_intel", method: "Amount", amount: 100 } as ChargeAllocation,
          { billToCustomerId: "cus_amd", method: "Amount", amount: 100 } as ChargeAllocation,
          { billToCustomerId: "cus_shipper", method: "Amount", amount: 100 } as ChargeAllocation,
        ]}
        onValues={onValues}
      />,
    );

    await user.click(screen.getByRole("radio", { name: "By percent" }));
    await user.click(screen.getByText("read"));

    expect(
      onValues.mock.lastCall?.[0].allocations.map((row: ChargeAllocation) => row.percent),
    ).toEqual([33.333333, 33.333333, 33.333334]);
    expect(screen.getByTestId("charge-split-remainder")).toHaveTextContent(
      "The full $300.00 is covered",
    );
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
