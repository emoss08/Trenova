import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { MoneyField } from "../money-field";

afterEach(cleanup);

type Values = { capCents: number };

function Harness({ initial, blankZero }: { initial: number; blankZero?: boolean }) {
  const form = useForm<Values>({ defaultValues: { capCents: initial } });
  const value = useWatch({ control: form.control, name: "capCents" });
  return (
    <>
      <MoneyField<Values>
        control={form.control}
        name="capCents"
        label="Cap"
        placeholder="Automatic"
        blankZero={blankZero}
      />
      <span data-testid="state">{value}</span>
    </>
  );
}

describe("MoneyField blankZero", () => {
  it("shows a stored 0 as empty so the placeholder says what 0 means", () => {
    render(<Harness initial={0} blankZero />);

    const input = screen.getByLabelText("Cap");
    expect(input).toHaveValue("");
    expect(input).toHaveAttribute("placeholder", "Automatic");
  });

  it("shows a stored 0 as 0.00 by default", () => {
    render(<Harness initial={0} />);

    expect(screen.getByLabelText("Cap")).toHaveValue("0.00");
  });

  it("stores a typed amount in cents and an emptied field as 0", async () => {
    const user = userEvent.setup();
    render(<Harness initial={0} blankZero />);

    const input = screen.getByLabelText("Cap");
    await user.type(input, "650");
    expect(screen.getByTestId("state")).toHaveTextContent("65000");

    await user.clear(input);
    expect(screen.getByTestId("state")).toHaveTextContent("0");
    expect(input).toHaveValue("");
  });
});
