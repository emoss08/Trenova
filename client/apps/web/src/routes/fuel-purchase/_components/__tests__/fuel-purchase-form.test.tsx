import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { FuelPurchaseFormValues } from "@trenova/shared/types/fuel-purchase";
import { FormProvider, useController, useForm, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FuelPurchaseForm } from "../fuel-purchase-form";
import { buildFuelPurchaseDefaults } from "../fuel-purchase-panel";

const TRACTOR_META: Record<string, Record<string, string>> = {
  trk_1: { primaryWorkerId: "wrk_9" },
  trk_2: { primaryWorkerId: "wrk_8" },
  trk_none: {},
};

type PickerProps = {
  control: Control;
  name: string;
  label: string;
  onOptionChange?: (option: GraphQLSelectOption | null) => void;
};

function NativePicker({ control, name, label, onOptionChange }: PickerProps) {
  const { field } = useController({ control, name });
  return (
    <input
      aria-label={label}
      value={(field.value as string) ?? ""}
      onChange={(event) => {
        const value = event.target.value;
        field.onChange(value);
        onOptionChange?.(
          value
            ? { id: value, label: value, description: null, meta: TRACTOR_META[value] ?? null }
            : null,
        );
      }}
    />
  );
}

vi.mock("@/components/autocomplete-fields", () => ({
  TractorAutocompleteField: NativePicker,
  WorkerAutocompleteField: NativePicker,
  FuelCardAutocompleteField: NativePicker,
}));

vi.mock("@/components/fields/ifta-jurisdiction-select-field", () => ({
  IftaJurisdictionSelectField: ({ control, name, label }: PickerProps) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({
    control,
    name,
    label,
    options,
  }: PickerProps & { options: { value: string; label: string }[] }) => {
    const { field } = useController({ control, name });
    return (
      <select
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    );
  },
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

vi.mock("@/components/fields/switch-field", () => ({
  SwitchField: ({ control, name, label }: PickerProps) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        type="checkbox"
        checked={Boolean(field.value)}
        onChange={(event) => field.onChange(event.target.checked)}
      />
    );
  },
}));

vi.mock("@/components/info-popover", () => ({
  InfoPopover: () => null,
}));

const NOW = 1_760_000_000;

function Harness({
  defaults,
  isEdit = false,
  imported,
}: {
  defaults?: Partial<FuelPurchaseFormValues>;
  isEdit?: boolean;
  imported?: boolean;
}) {
  const form = useForm<FuelPurchaseFormValues>({
    defaultValues: { ...buildFuelPurchaseDefaults(null, NOW), ...defaults },
  });
  return (
    <FormProvider {...form}>
      <FuelPurchaseForm isEdit={isEdit} imported={imported} />
    </FormProvider>
  );
}

function inputById(name: string): HTMLInputElement {
  const element = document.getElementById(`input-${name}`);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`input-${name} not rendered`);
  }
  return element;
}

afterEach(() => {
  cleanup();
});

describe("FuelPurchaseForm", () => {
  it("computes the total from quantity and unit price as they are typed", async () => {
    render(<Harness />);

    fireEvent.change(inputById("quantity"), { target: { value: "100" } });
    fireEvent.change(inputById("unitPrice"), { target: { value: "3.5" } });

    await waitFor(() => expect(inputById("totalAmount")).toHaveValue("350.00"));
    expect(screen.queryByRole("button", { name: /use computed/i })).not.toBeInTheDocument();

    fireEvent.change(inputById("quantity"), { target: { value: "120.5" } });
    await waitFor(() => expect(inputById("totalAmount")).toHaveValue("421.75"));
  });

  it("keeps a total the user typed over, until they choose the computed one again", async () => {
    render(<Harness />);

    fireEvent.change(inputById("quantity"), { target: { value: "100" } });
    fireEvent.change(inputById("unitPrice"), { target: { value: "3.5" } });
    await waitFor(() => expect(inputById("totalAmount")).toHaveValue("350.00"));

    fireEvent.change(inputById("totalAmount"), { target: { value: "360.00" } });
    fireEvent.change(inputById("quantity"), { target: { value: "200" } });

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /use computed/i })).toBeVisible(),
    );
    expect(inputById("totalAmount")).toHaveValue("360.00");

    fireEvent.click(screen.getByRole("button", { name: /use computed/i }));
    await waitFor(() => expect(inputById("totalAmount")).toHaveValue("700.00"));
    expect(screen.queryByRole("button", { name: /use computed/i })).not.toBeInTheDocument();

    fireEvent.change(inputById("unitPrice"), { target: { value: "4" } });
    await waitFor(() => expect(inputById("totalAmount")).toHaveValue("800.00"));
  });

  it("treats an edited row whose total already differs from quantity × price as overridden", async () => {
    render(
      <Harness isEdit defaults={{ quantity: "100", unitPrice: "3.5", totalAmount: "372.10" }} />,
    );

    expect(inputById("totalAmount")).toHaveValue("372.10");
    fireEvent.change(inputById("quantity"), { target: { value: "150" } });
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /use computed/i })).toBeVisible(),
    );
    expect(inputById("totalAmount")).toHaveValue("372.10");
  });

  it("fills an empty driver from the tractor's primary worker, and only an empty one", async () => {
    render(<Harness />);

    fireEvent.change(screen.getByLabelText("Tractor"), { target: { value: "trk_1" } });
    await waitFor(() => expect(screen.getByLabelText("Driver")).toHaveValue("wrk_9"));

    fireEvent.change(screen.getByLabelText("Driver"), { target: { value: "wrk_2" } });
    fireEvent.change(screen.getByLabelText("Tractor"), { target: { value: "trk_2" } });
    await waitFor(() => expect(screen.getByLabelText("Tractor")).toHaveValue("trk_2"));
    expect(screen.getByLabelText("Driver")).toHaveValue("wrk_2");

    fireEvent.change(screen.getByLabelText("Driver"), { target: { value: "" } });
    fireEvent.change(screen.getByLabelText("Tractor"), { target: { value: "trk_none" } });
    await waitFor(() => expect(screen.getByLabelText("Tractor")).toHaveValue("trk_none"));
    expect(screen.getByLabelText("Driver")).toHaveValue("");
  });

  it("locks the card and reference on an imported purchase and says why", () => {
    render(<Harness isEdit imported />);
    expect(screen.getByText("Imported from a fuel card statement")).toBeInTheDocument();
    expect(inputById("transactionReference")).toHaveAttribute("readonly");
    expect(inputById("cardLastFour")).toHaveAttribute("readonly");
  });
});
