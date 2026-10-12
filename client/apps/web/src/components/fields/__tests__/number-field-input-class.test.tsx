import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { NumberField } from "../number-field";

afterEach(cleanup);

type Values = { limit: number | null };

function Harness() {
  const form = useForm<Values>({ defaultValues: { limit: 3 } });

  return (
    <NumberField
      control={form.control}
      name="limit"
      label="Daily limit"
      className="wrapper-only"
      inputClassName="h-8.5"
    />
  );
}

/**
 * A page that sizes its fields sizes the box a person types in. The wrapper
 * also holds the label and the message under the box, so a height given to it
 * would squeeze those instead.
 */
describe("NumberField inputClassName", () => {
  it("sizes the input and leaves the wrapper alone", () => {
    const { container } = render(<Harness />);
    const input = screen.getByRole("textbox");
    const wrapper = container.firstElementChild;

    expect(input).toHaveClass("h-8.5");
    expect(wrapper).toHaveClass("wrapper-only");
    expect(wrapper).not.toHaveClass("h-8.5");
  });
});
