import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MoneyField } from "../money-field";

afterEach(cleanup);

type Values = { capMinor: number };

type FieldProps = Partial<
  Pick<
    Parameters<typeof MoneyField<Values>>[0],
    | "required"
    | "clearable"
    | "disabled"
    | "readOnly"
    | "allowNegative"
    | "stepper"
    | "step"
    | "currency"
    | "ledger"
    | "placeholder"
    | "description"
    | "onAmountChange"
  >
>;

function Harness({
  initial,
  props,
  onForm,
}: {
  initial: number;
  props: FieldProps;
  onForm: (form: UseFormReturn<Values>) => void;
}) {
  const form = useForm<Values>({ defaultValues: { capMinor: initial } });
  onForm(form);

  return <MoneyField<Values> control={form.control} name="capMinor" label="Cap" {...props} />;
}

function renderField(initial: number, props: FieldProps = {}) {
  const held: { form?: UseFormReturn<Values> } = {};
  render(<Harness initial={initial} props={props} onForm={(form) => (held.form = form)} />);
  return {
    value: () => held.form!.getValues("capMinor"),
    input: () => screen.getByRole("textbox", { name: "Cap" }),
    canClear: () => screen.queryByRole("button", { name: "Clear" }) !== null,
    press: (name: string) => fireEvent.click(screen.getByRole("button", { name })),
  };
}

describe("a money field's currency", () => {
  it("is shown in the field, reading US dollars when none is given", () => {
    renderField(1_250);

    expect(screen.getByText("$")).toBeInTheDocument();
  });

  it("is the currency the amount is in", () => {
    renderField(1_250, { currency: "EUR" });

    expect(screen.getByText("€")).toBeInTheDocument();
    expect(screen.queryByText("$")).not.toBeInTheDocument();
  });
});

describe("a money field's amount", () => {
  it("is shown in whole units with its cents and thousands marked", () => {
    const field = renderField(258_222);

    expect(field.input()).toHaveValue("2,582.22");
  });

  it("is stored in cents exactly as typed, with no cent lost", async () => {
    const user = userEvent.setup();
    const onAmountChange = vi.fn();
    const field = renderField(0, { onAmountChange });

    await user.clear(field.input());
    await user.type(field.input(), "0.29");

    expect(field.value()).toBe(29);
    expect(onAmountChange).toHaveBeenLastCalledWith(29);
  });

  it("is tied to its label, its description and its error", () => {
    const field = renderField(1_250, { description: "The most one settlement may pay." });

    fireEvent.click(screen.getByText("Cap"));
    expect(field.input()).toHaveAttribute("id", "input-capMinor");
    expect(screen.getByText("The most one settlement may pay.")).toHaveAttribute(
      "id",
      field.input().getAttribute("aria-describedby"),
    );
  });
});

describe("a money field's stepper", () => {
  it("moves the amount by one whole unit", () => {
    const field = renderField(1_250);

    field.press("Increment");
    expect(field.value()).toBe(1_350);
    expect(field.input()).toHaveValue("13.50");

    field.press("Decrement");
    field.press("Decrement");
    expect(field.value()).toBe(1_150);
  });

  it("moves by the step it is given, in minor units", () => {
    const field = renderField(1_000, { step: 25 });

    field.press("Increment");

    expect(field.value()).toBe(1_025);
  });

  it("stops at zero unless the amount may be negative", () => {
    const field = renderField(50);
    field.press("Decrement");
    expect(field.value()).toBe(0);
    cleanup();

    const signed = renderField(50, { allowNegative: true });
    signed.press("Decrement");
    expect(signed.value()).toBe(-50);
  });

  it("tells the form's owner the new amount, as typing does", () => {
    const onAmountChange = vi.fn();
    renderField(1_250, { onAmountChange }).press("Increment");

    expect(onAmountChange).toHaveBeenCalledExactlyOnceWith(1_350);
  });

  it.each([
    ["disabled", { disabled: true }],
    ["read-only", { readOnly: true }],
  ])("cannot be pressed while the field is %s", (_, props) => {
    renderField(1_250, props);

    expect(screen.getByRole("button", { name: "Increment" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Decrement" })).toBeDisabled();
  });

  it("is left out where the form asks", () => {
    renderField(1_250, { stepper: false });

    expect(screen.queryByRole("button", { name: "Increment" })).not.toBeInTheDocument();
  });
});

describe("emptying a money field", () => {
  it("is offered when the field is not required, and leaves it holding zero", () => {
    const field = renderField(1_250);

    field.press("Clear");

    expect(field.value()).toBe(0);
    expect(field.canClear()).toBe(false);
  });

  it("is not offered while the amount is already zero", () => {
    expect(renderField(0).canClear()).toBe(false);
  });

  it.each([
    ["it is required", { required: true }],
    ["the form says it must always hold an amount", { clearable: false }],
    ["it is disabled", { disabled: true }],
    ["it is read-only", { readOnly: true }],
  ])("is not offered when %s", (_, props) => {
    expect(renderField(1_250, props).canClear()).toBe(false);
  });
});

/**
 * A journal's debit and credit cells: the column's heading and its total say what the
 * figures are, so a cell is its digits and nothing that would crowd them out of a narrow
 * column.
 */
describe("a money field drawn as a ledger cell", () => {
  it("is its digits alone: no currency mark, stepper or clear control", () => {
    const field = renderField(125_050, { ledger: true });

    expect(field.input()).toHaveValue("1,250.50");
    expect(screen.queryByText("$")).not.toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("sets its digits against the right edge, under the column's heading", () => {
    expect(renderField(125_050, { ledger: true }).input()).toHaveClass("text-right");
    cleanup();
    expect(renderField(125_050).input()).toHaveClass("text-left");
  });

  it("is blank, with no placeholder, while it holds nothing", () => {
    const input = renderField(0, { ledger: true }).input();

    expect(input).toHaveValue("");
    expect(input).toHaveAttribute("placeholder", "");
  });

  it("is emptied by deleting its digits, which leaves it holding zero", async () => {
    const user = userEvent.setup();
    const field = renderField(125_050, { ledger: true });

    await user.clear(field.input());

    expect(field.value()).toBe(0);
    expect(field.input()).toHaveValue("");
  });

  it("still takes the controls a form asks for by name", () => {
    const field = renderField(125_050, { ledger: true, stepper: true, clearable: true });

    expect(screen.getByRole("button", { name: "Increment" })).toBeInTheDocument();
    expect(field.canClear()).toBe(true);
  });
});
