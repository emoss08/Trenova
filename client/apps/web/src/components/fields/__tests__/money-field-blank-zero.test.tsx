import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { MoneyField } from "../money-field";

afterEach(cleanup);

type Values = { capCents: number };

function Harness({
  initial,
  showZero,
  readOnly,
  disabled,
}: {
  initial: number;
  showZero?: boolean;
  readOnly?: boolean;
  disabled?: boolean;
}) {
  const form = useForm<Values>({ defaultValues: { capCents: initial } });
  const value = useWatch({ control: form.control, name: "capCents" });
  return (
    <>
      <MoneyField<Values>
        control={form.control}
        name="capCents"
        label="Cap"
        placeholder="Automatic"
        showZero={showZero}
        readOnly={readOnly}
        disabled={disabled}
      />
      <span data-testid="state">{value}</span>
    </>
  );
}

/**
 * The form holds a number, so it cannot tell an amount nobody entered from a zero somebody
 * did. A field that can be typed in is drawn empty at 0 rather than claiming "0.00".
 */
describe("a money field holding zero", () => {
  it("is drawn empty, so the placeholder says what 0 means", () => {
    render(<Harness initial={0} />);

    const input = screen.getByLabelText("Cap");
    expect(input).toHaveValue("");
    expect(input).toHaveAttribute("placeholder", "Automatic");
  });

  it("shows 0.00 where the form asks for the zero to be shown", () => {
    render(<Harness initial={0} showZero />);

    expect(screen.getByLabelText("Cap")).toHaveValue("0.00");
  });

  it.each([
    ["read-only", { readOnly: true }],
    ["disabled", { disabled: true }],
  ])("shows 0.00 while it is %s, where it is a figure and not an entry", (_, props) => {
    render(<Harness initial={0} {...props} />);

    expect(screen.getByLabelText("Cap")).toHaveValue("0.00");
  });

  it("stores a typed amount in cents and an emptied field as 0, drawn empty again", async () => {
    const user = userEvent.setup();
    render(<Harness initial={0} />);

    const input = screen.getByLabelText("Cap");
    await user.type(input, "650");
    expect(screen.getByTestId("state")).toHaveTextContent("65000");

    await user.clear(input);
    expect(screen.getByTestId("state")).toHaveTextContent("0");
    expect(input).toHaveValue("");
  });

  it("takes an amount under one unit, typed from its leading zero", async () => {
    const user = userEvent.setup();
    render(<Harness initial={0} />);

    await user.type(screen.getByLabelText("Cap"), "0.50");

    expect(screen.getByTestId("state")).toHaveTextContent("50");
    expect(screen.getByLabelText("Cap")).toHaveValue("0.50");
  });
});
