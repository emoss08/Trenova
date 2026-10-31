import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { InputField } from "../input-field";
import { NumberField } from "../number-field";

afterEach(cleanup);

type Values = { description: string; amount: number | null };

function Harness({ hideLabel }: { hideLabel: boolean }) {
  const form = useForm<Values>({ defaultValues: { description: "", amount: null } });

  return (
    <>
      <InputField
        control={form.control}
        name="description"
        label="Description"
        hideLabel={hideLabel}
      />
      <NumberField control={form.control} name="amount" label="Amount" hideLabel={hideLabel} />
    </>
  );
}

/**
 * Rows of a repeated group after the first hide their labels so the columns line up
 * under the first row's headings, while each control still announces its name.
 */
describe("field hideLabel", () => {
  it.each(["Description", "Amount"])("keeps %s named but off screen", (name) => {
    render(<Harness hideLabel />);

    const input = screen.getByLabelText(name);
    const label = screen.getByText(name, { selector: "label" });

    expect(input.tagName).toBe("INPUT");
    expect(label.parentElement).toHaveClass("sr-only");
    expect(input).not.toHaveAttribute("hidelabel");
  });

  it.each(["Description", "Amount"])("shows the %s label by default", (name) => {
    render(<Harness hideLabel={false} />);

    const label = screen.getByText(name, { selector: "label" });

    expect(label.parentElement).not.toHaveClass("sr-only");
  });
});
