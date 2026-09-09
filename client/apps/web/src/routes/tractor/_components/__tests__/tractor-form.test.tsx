import { cleanup, render, screen } from "@testing-library/react";
import type { Tractor } from "@/types/tractor";
import { FormProvider, useController, useForm, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TractorForm } from "../tractor-form";
import { buildTractorDefaults } from "../tractor-panel";

// Fuel types are an IFTA_FUEL_TYPE select-option resource, so the tractor form
// must reach the server for them instead of rendering a bundled option list.

type PickerProps = {
  control: Control;
  name: string;
  label: string;
};

function pickerStub(source: string) {
  return function Picker({ control, name, label }: PickerProps) {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        data-source={source}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  };
}

vi.mock("@/components/autocomplete-fields", () => ({
  EquipmentManufacturerAutocompleteField: pickerStub("autocomplete"),
  EquipmentTypeAutocompleteField: pickerStub("autocomplete"),
  FleetCodeAutocompleteField: pickerStub("autocomplete"),
  IftaFuelTypeAutocompleteField: pickerStub("ifta-fuel-type-autocomplete"),
  UsStateAutocompleteField: pickerStub("autocomplete"),
  WorkerAutocompleteField: pickerStub("autocomplete"),
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
        data-source="static-options"
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

vi.mock("@/components/custom-fields-section", () => ({
  CustomFieldsSection: () => null,
}));

function Harness() {
  const form = useForm<Tractor>({ defaultValues: buildTractorDefaults() });

  return (
    <FormProvider {...form}>
      <TractorForm />
    </FormProvider>
  );
}

describe("TractorForm fuel type", () => {
  afterEach(cleanup);

  it("picks the fuel from the server-backed IFTA fuel type resource", () => {
    render(<Harness />);

    const fuelType = screen.getByLabelText("Fuel Type");
    expect(fuelType).toHaveAttribute("data-source", "ifta-fuel-type-autocomplete");
    expect(fuelType).toHaveValue("Diesel");
  });

  it("leaves status on the bundled option list", () => {
    render(<Harness />);

    expect(screen.getByLabelText("Status")).toHaveAttribute("data-source", "static-options");
  });
});
