import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SelectField } from "../select-field";

afterEach(cleanup);

type Values = { choice: string | number | null };

const DRIVER_TYPES = [
  { value: "Company", label: "Company driver" },
  { value: "Owner", label: "Owner operator" },
];

const WEEKDAYS = [
  { value: 0, label: "Sunday" },
  { value: 5, label: "Friday" },
];

function Harness({
  value,
  options,
  required,
  clearable,
  isReadOnly,
  onForm,
}: {
  value: Values["choice"];
  options: typeof DRIVER_TYPES | typeof WEEKDAYS;
  required?: boolean;
  clearable?: boolean;
  isReadOnly?: boolean;
  onForm: (form: UseFormReturn<Values>) => void;
}) {
  const form = useForm<Values>({ defaultValues: { choice: value } });
  onForm(form);

  return (
    <SelectField
      control={form.control}
      name="choice"
      label="Choice"
      options={options}
      required={required}
      clearable={clearable}
      isReadOnly={isReadOnly}
    />
  );
}

function renderField(props: Omit<Parameters<typeof Harness>[0], "onForm">) {
  const held: { form?: UseFormReturn<Values> } = {};
  render(<Harness {...props} onForm={(form) => (held.form = form)} />);
  return {
    value: () => held.form!.getValues("choice"),
    clear: () => fireEvent.click(screen.getByTitle("Clear")),
    canClear: () => screen.queryByTitle("Clear") !== null,
  };
}

describe("a select that is not required", () => {
  it("can be emptied, and holds an empty string where its options are text", () => {
    const field = renderField({ value: "Owner", options: DRIVER_TYPES });

    field.clear();

    expect(field.value()).toBe("");
    expect(field.canClear()).toBe(false);
  });

  it("holds nothing once emptied where its options are numbers", () => {
    const field = renderField({ value: 5, options: WEEKDAYS });

    field.clear();

    expect(field.value()).toBeNull();
  });

  it("can be emptied of a zero, which is a value like any other", () => {
    const field = renderField({ value: 0, options: WEEKDAYS });

    expect(field.canClear()).toBe(true);
    field.clear();

    expect(field.value()).toBeNull();
  });

  it("offers nothing to clear while it is empty", () => {
    expect(renderField({ value: "", options: DRIVER_TYPES }).canClear()).toBe(false);
    cleanup();
    expect(renderField({ value: null, options: WEEKDAYS }).canClear()).toBe(false);
  });

  it("cannot be emptied where the form says it must always hold a value", () => {
    const field = renderField({ value: "Owner", options: DRIVER_TYPES, clearable: false });

    expect(field.canClear()).toBe(false);
  });

  it("cannot be emptied while it is read-only", () => {
    const field = renderField({ value: "Owner", options: DRIVER_TYPES, isReadOnly: true });

    expect(field.canClear()).toBe(false);
  });
});

describe("a required select", () => {
  it("cannot be emptied", () => {
    const field = renderField({ value: "Owner", options: DRIVER_TYPES, required: true });

    expect(field.canClear()).toBe(false);
  });

  it("can be, where the form asks for it", () => {
    const field = renderField({
      value: "Owner",
      options: DRIVER_TYPES,
      required: true,
      clearable: true,
    });

    field.clear();

    expect(field.value()).toBe("");
  });
});

describe("the trigger of a select that can be emptied", () => {
  it("is still named for the option it holds, not for the clear control inside it", () => {
    renderField({ value: "Owner", options: DRIVER_TYPES });

    expect(screen.getByRole("button", { name: "Owner operator" })).toBeInTheDocument();
  });
});
