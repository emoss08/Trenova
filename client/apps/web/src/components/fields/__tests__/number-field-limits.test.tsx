import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { NumberField } from "../number-field";

afterEach(cleanup);

type Values = { quantity: string };

function Harness({ stepper }: { stepper?: boolean }) {
  const form = useForm<Values>({ defaultValues: { quantity: "" } });

  return (
    <NumberField
      control={form.control}
      name="quantity"
      valueType="string"
      aria-label="Quantity"
      max={999}
      stepper={stepper}
    />
  );
}

describe("NumberField limits", () => {
  it("refuses a keystroke that would take the value past its max", async () => {
    const user = userEvent.setup();
    render(<Harness />);
    const input = screen.getByRole("textbox", { name: "Quantity" });

    await user.type(input, "12345");

    expect(input).toHaveValue("123");
  });

  it("draws no increment or decrement buttons when the stepper is off", () => {
    render(<Harness stepper={false} />);

    expect(screen.queryByRole("button", { name: "Increment" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Decrement" })).not.toBeInTheDocument();
  });

  it("keeps the buttons by default", () => {
    render(<Harness />);

    expect(screen.getByRole("button", { name: "Increment" })).toBeInTheDocument();
  });
});
