import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ChargeAllocation } from "@trenova/shared/types/shipment";
import {
  FormProvider,
  useController,
  useForm,
  type Control,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChargePayerControl } from "../charge-payer-control";

const CUSTOMERS = [
  { id: "cus_intel", label: "Intel", description: null, meta: { code: "INTEL" } },
  { id: "cus_amd", label: "AMD", description: null, meta: { code: "AMD" } },
];

/**
 * The real picker fetches options over GraphQL. This stand-in keeps its
 * contract: the form value is the customer id and the chosen option comes back
 * through `onOptionChange`.
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

let latestForm: UseFormReturn<FieldValues> | null = null;

function Harness({ allocations }: { allocations: ChargeAllocation[] }) {
  const form = useForm<FieldValues>({ defaultValues: { allocations } });
  latestForm = form;
  return (
    <FormProvider {...form}>
      <ChargePayerControl
        name="allocations"
        chargeAmount={1000}
        defaultPayer={{ id: "cus_intel", label: "Intel" }}
        splitTitle="Split freight charge"
        splitDescription="Divide the freight charge."
      />
    </FormProvider>
  );
}

async function openControl(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("charge-payer-control"));
}

describe("ChargePayerControl", () => {
  it("reads as billed to the shipment when nobody else pays the charge", () => {
    render(<Harness allocations={[]} />);

    expect(screen.getByTestId("charge-payer-control")).toHaveTextContent(
      "Bill to: Same as shipment",
    );
  });

  // Giving a charge to one customer is the common case, so it is one pick, not a
  // split with a single row at 100%.
  it("gives the whole charge to the customer picked", async () => {
    const user = userEvent.setup();
    render(<Harness allocations={[]} />);

    await openControl(user);
    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Bill this charge to" }),
      "cus_amd",
    );

    await waitFor(() => {
      expect(latestForm?.getValues("allocations")).toEqual([
        {
          billToCustomerId: "cus_amd",
          method: "Percent",
          percent: 100,
          amount: null,
          sequence: 0,
          billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
        },
      ]);
    });
    expect(screen.getByTestId("charge-payer-control")).toHaveTextContent("Bill to: AMD");
  });

  // The server syncs allocation rows by id. Re-pointing the one existing row keeps
  // its id and version instead of deleting it and inserting a new one.
  it("keeps the existing row's identity when the payer changes", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        allocations={[
          {
            id: "chal_1",
            version: 3,
            billToCustomerId: "cus_amd",
            method: "Percent",
            percent: 100,
            amount: null,
            sequence: 0,
            billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
          } as ChargeAllocation,
        ]}
      />,
    );

    await openControl(user);
    const picker = await screen.findByRole("combobox", { name: "Bill this charge to" });
    await user.selectOptions(picker, "");
    await user.selectOptions(picker, "cus_amd");

    await waitFor(() => {
      const rows = latestForm?.getValues("allocations") as ChargeAllocation[];
      expect(rows).toHaveLength(1);
      expect(rows[0]).toMatchObject({ id: "chal_1", version: 3, billToCustomerId: "cus_amd" });
    });
  });

  it("returns the charge to the shipment's payer", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        allocations={[
          {
            billToCustomerId: "cus_amd",
            method: "Percent",
            percent: 100,
            amount: null,
            sequence: 0,
          } as ChargeAllocation,
        ]}
      />,
    );

    await openControl(user);
    await user.click(await screen.findByRole("button", { name: "Same as shipment payer" }));

    await waitFor(() => {
      expect(latestForm?.getValues("allocations")).toEqual([]);
    });
  });

  // Picking the customer who already pays the shipment is the same as no split:
  // storing a 100% row for them would only be noise the server discards.
  it("stores no row when the shipment's own payer is picked", async () => {
    const user = userEvent.setup();
    render(<Harness allocations={[]} />);

    await openControl(user);
    await user.selectOptions(
      await screen.findByRole("combobox", { name: "Bill this charge to" }),
      "cus_intel",
    );

    await waitFor(() => {
      expect(latestForm?.getValues("allocations")).toEqual([]);
    });
  });

  it("opens the split editor for sharing the charge between payers", async () => {
    const user = userEvent.setup();
    render(<Harness allocations={[]} />);

    await openControl(user);
    await user.click(await screen.findByRole("button", { name: "Split this charge…" }));

    expect(await screen.findByText("Split freight charge")).toBeInTheDocument();
    expect(screen.getByText("Divide the freight charge.")).toBeInTheDocument();
  });

  it("counts the payers of a split charge", () => {
    render(
      <Harness
        allocations={[
          {
            billToCustomerId: "cus_intel",
            method: "Percent",
            percent: 60,
            sequence: 0,
          } as ChargeAllocation,
          {
            billToCustomerId: "cus_amd",
            method: "Percent",
            percent: 40,
            sequence: 1,
          } as ChargeAllocation,
        ]}
      />,
    );

    expect(screen.getByTestId("charge-payer-control")).toHaveTextContent("Split 2 ways");
  });
});
