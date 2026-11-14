import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SelectField } from "../select-field";

afterEach(cleanup);

type Values = { periodEndDayOfWeek: number };

// A settlement control's period end day: weekdays as the server holds them, 0 for Sunday.
const WEEKDAYS = [
  { value: 0, label: "Sunday" },
  { value: 5, label: "Friday" },
  { value: 6, label: "Saturday" },
];

function Harness({ day, onForm }: { day: number; onForm: (form: UseFormReturn<Values>) => void }) {
  const form = useForm<Values>({ defaultValues: { periodEndDayOfWeek: day } });
  onForm(form);

  return (
    <SelectField
      control={form.control}
      name="periodEndDayOfWeek"
      label="Period end day"
      options={WEEKDAYS}
      required
    />
  );
}

function renderField(day: number) {
  const held: { form?: UseFormReturn<Values> } = {};
  render(<Harness day={day} onForm={(form) => (held.form = form)} />);
  return () => held.form!.getValues("periodEndDayOfWeek");
}

/**
 * An option's value may be a number. The list hands a pick back as text, and the field once
 * wrote that text to the form: the value then matched no option, so the field drew blank,
 * and a form that checks for a number refused to save.
 */
describe("SelectField with number values", () => {
  it("keeps the number the option holds when one is picked, and shows its label", async () => {
    const user = userEvent.setup();
    const value = renderField(5);

    await user.click(screen.getByRole("button", { name: "Friday" }));
    await user.click(screen.getAllByText("Saturday").at(-1)!);

    expect(value()).toBe(6);
    expect(screen.getByRole("button", { name: "Saturday" })).toBeInTheDocument();
  });

  it("takes a pick of zero as a value like any other", async () => {
    const user = userEvent.setup();
    const value = renderField(5);

    await user.click(screen.getByRole("button", { name: "Friday" }));
    await user.click(screen.getAllByText("Sunday").at(-1)!);

    expect(value()).toBe(0);
    expect(screen.getByRole("button", { name: "Sunday" })).toBeInTheDocument();
  });

  it("draws a stored zero as chosen, not as an empty field", () => {
    renderField(0);

    const label = screen.getByRole("button", { name: "Sunday" }).querySelector("div");
    expect(label).toHaveTextContent("Sunday");
    expect(label).not.toHaveClass("text-muted-foreground");
  });

  it("leaves the value alone when the option already chosen is picked again", async () => {
    const user = userEvent.setup();
    const value = renderField(5);

    await user.click(screen.getByRole("button", { name: "Friday" }));
    await user.click(screen.getAllByText("Friday").at(-1)!);

    expect(value()).toBe(5);
  });
});
