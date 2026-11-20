import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { FieldLayoutProvider } from "../field-components";
import { SelectField } from "../select-field";
import { TextareaField } from "../textarea-field";

afterEach(cleanup);

type Values = { terms: string; term: string };

function Harness() {
  const form = useForm<Values>({ defaultValues: { terms: "", term: "" } });

  return (
    <FieldLayoutProvider layout="row">
      <TextareaField
        control={form.control}
        name="terms"
        label="Invoice terms"
        description="Printed on each invoice."
      />
      <SelectField
        control={form.control}
        name="term"
        label="Payment term"
        options={[{ value: "Net30", label: "Net 30" }]}
      />
    </FieldLayoutProvider>
  );
}

function controlColumn(element: HTMLElement) {
  return element.closest("[data-slot='setting-row']")!.lastElementChild!;
}

/**
 * Text written out needs room to read: in a settings row a textarea takes the row's wide
 * control, not the narrower one a choice takes.
 */
describe("textarea field in a settings row", () => {
  it("takes the wide control", () => {
    render(<Harness />);

    const textarea = screen.getByRole("textbox", { name: "Invoice terms" });
    expect(controlColumn(textarea)).toHaveClass("w-80");
    expect(controlColumn(textarea)).not.toHaveClass("w-52");
    expect(controlColumn(screen.getByText("Payment term"))).toHaveClass("w-52");
  });

  it("is named by its label and described by its description", () => {
    render(<Harness />);

    const textarea = screen.getByRole("textbox", { name: "Invoice terms" });
    expect(screen.getByText("Invoice terms", { selector: "label" })).toHaveAttribute(
      "for",
      textarea.id,
    );
    expect(textarea).toHaveAccessibleDescription("Printed on each invoice.");
  });
});
