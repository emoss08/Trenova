import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { InputField } from "../input-field";
import { NumberField } from "../number-field";

afterEach(cleanup);

type Values = { reference: string; payDelayDays: number | null };

type FieldProps = {
  required?: boolean;
  clearable?: boolean;
  disabled?: boolean;
  readOnly?: boolean;
};

function Harness({
  values,
  field,
  props,
  onForm,
}: {
  values: Values;
  field: "text" | "number";
  props: FieldProps;
  onForm: (form: UseFormReturn<Values>) => void;
}) {
  const form = useForm<Values>({ defaultValues: values });
  onForm(form);

  return field === "text" ? (
    <InputField control={form.control} name="reference" label="Reference" {...props} />
  ) : (
    <NumberField control={form.control} name="payDelayDays" label="Pay delay" {...props} />
  );
}

function renderField(field: "text" | "number", values: Partial<Values>, props: FieldProps = {}) {
  const held: { form?: UseFormReturn<Values> } = {};
  render(
    <Harness
      field={field}
      values={{ reference: "", payDelayDays: null, ...values }}
      props={props}
      onForm={(form) => (held.form = form)}
    />,
  );
  return {
    values: () => held.form!.getValues(),
    clear: () => fireEvent.click(screen.getByRole("button", { name: "Clear" })),
    canClear: () => screen.queryByRole("button", { name: "Clear" }) !== null,
  };
}

describe("a text field that is not required", () => {
  it("can be emptied with the control at its end", () => {
    const field = renderField("text", { reference: "Statement 2026-04" });

    field.clear();

    expect(field.values().reference).toBe("");
    expect(screen.getByRole("textbox", { name: "Reference" })).toHaveValue("");
    expect(field.canClear()).toBe(false);
  });

  it("offers nothing to clear while it is empty", () => {
    expect(renderField("text", { reference: "" }).canClear()).toBe(false);
  });

  it.each([
    ["it is required", { required: true }],
    ["the form says it must always hold a value", { clearable: false }],
    ["it is disabled", { disabled: true }],
    ["it is read-only", { readOnly: true }],
  ])("cannot be emptied that way when %s", (_, props) => {
    expect(renderField("text", { reference: "Statement 2026-04" }, props).canClear()).toBe(false);
  });

  it("keeps the clear control out of the form's tab stops", () => {
    renderField("text", { reference: "Statement 2026-04" });

    expect(screen.getByRole("button", { name: "Clear" })).toHaveAttribute("tabindex", "-1");
  });
});

describe("a number field that is not required", () => {
  it("can be emptied, and then holds nothing rather than zero", () => {
    const field = renderField("number", { payDelayDays: 14 });

    field.clear();

    expect(field.values().payDelayDays).toBeNull();
    expect(field.canClear()).toBe(false);
  });

  it("can be emptied of a zero, which is a value like any other", () => {
    const field = renderField("number", { payDelayDays: 0 });

    expect(field.canClear()).toBe(true);
    field.clear();

    expect(field.values().payDelayDays).toBeNull();
  });

  it("offers nothing to clear while it is empty", () => {
    expect(renderField("number", { payDelayDays: null }).canClear()).toBe(false);
  });

  it.each([
    ["it is required", { required: true }],
    ["the form says it must always hold a value", { clearable: false }],
    ["it is disabled", { disabled: true }],
    ["it is read-only", { readOnly: true }],
  ])("cannot be emptied that way when %s", (_, props) => {
    expect(renderField("number", { payDelayDays: 14 }, props).canClear()).toBe(false);
  });

  it("keeps its stepper beside the clear control", () => {
    renderField("number", { payDelayDays: 14 });

    expect(screen.getByRole("button", { name: "Increment" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Decrement" })).toBeInTheDocument();
  });
});
