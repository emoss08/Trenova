import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, type Control, type FieldValues } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { NumberField } from "../number-field";

afterEach(cleanup);

// Money, miles and gallons are decimal strings from the schema through to the
// GraphQL `Decimal` scalar: the digits the user typed are the value. A field
// that hands back a float has already rounded them, and the schema then reports
// the field as the wrong type entirely.

type Values = {
  miles: string;
  odometer: number | null;
  loose: string;
};

function State({ values }: { values: unknown }) {
  return <span data-testid="state">{JSON.stringify(values)}</span>;
}

function StringHarness({
  decimalScale = 2,
  thousandSeparator = false,
  step,
  min,
  max,
  initial = "",
}: {
  decimalScale?: number;
  thousandSeparator?: boolean;
  step?: number;
  min?: number;
  max?: number;
  initial?: string;
}) {
  const form = useForm<Values>({
    defaultValues: { miles: initial, odometer: null, loose: "" },
  });
  return (
    <>
      <NumberField<Values>
        control={form.control}
        name="miles"
        label="Miles"
        valueType="string"
        decimalScale={decimalScale}
        thousandSeparator={thousandSeparator}
        step={step}
        min={min}
        max={max}
      />
      <State values={form.watch("miles")} />
    </>
  );
}

function NumberHarness({ initial = null }: { initial?: number | null }) {
  const form = useForm<Values>({
    defaultValues: { miles: "", odometer: initial, loose: "" },
  });
  return (
    <>
      <NumberField<Values>
        control={form.control}
        name="odometer"
        label="Odometer"
        step={10}
        min={0}
      />
      <State values={form.watch("odometer")} />
    </>
  );
}

// Custom fields are bound through an untyped control, so nothing at the call
// site can declare the kind; the field has to settle it from the value it holds.
function UndeclaredHarness({ initial }: { initial: string | number | null }) {
  const form = useForm<Values>({
    defaultValues: { miles: "", odometer: null, loose: initial as string },
  });
  return (
    <>
      <NumberField<FieldValues>
        control={form.control as unknown as Control<FieldValues>}
        name="loose"
        label="Loose"
        decimalScale={2}
      />
      <State values={form.watch("loose")} />
    </>
  );
}

function input(name: string): HTMLInputElement {
  const element = document.getElementById(`input-${name}`);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`input-${name} not rendered`);
  }
  return element;
}

function state(): string {
  return screen.getByTestId("state").textContent ?? "";
}

describe("NumberField value kind", () => {
  it("writes a decimal string field back as the digits that were typed", async () => {
    const user = userEvent.setup();
    render(<StringHarness />);

    await user.type(input("miles"), "412.55");

    expect(state()).toBe('"412.55"');
  });

  it("strips the display grouping from the value it stores", async () => {
    const user = userEvent.setup();
    render(<StringHarness thousandSeparator />);

    await user.type(input("miles"), "1234.5");

    expect(input("miles")).toHaveValue("1,234.5");
    expect(state()).toBe('"1234.5"');
  });

  it("clears a string field to an empty string, which nullable schemas read as unset", async () => {
    const user = userEvent.setup();
    render(<StringHarness initial="12.50" />);

    await user.clear(input("miles"));

    expect(state()).toBe('""');
  });

  it("steps a string field exactly at its scale rather than through a float", async () => {
    const user = userEvent.setup();
    render(<StringHarness initial="0.10" step={0.2} />);

    await user.click(screen.getByRole("button", { name: "Increment" }));

    expect(state()).toBe('"0.30"');
  });

  it("holds a stepped string field at its bounds", async () => {
    const user = userEvent.setup();
    render(<StringHarness initial="9.90" step={1} max={10} min={0} />);

    await user.click(screen.getByRole("button", { name: "Increment" }));
    expect(state()).toBe('"10.00"');

    for (let click = 0; click < 11; click += 1) {
      await user.click(screen.getByRole("button", { name: "Decrement" }));
    }

    expect(state()).toBe('"0.00"');
  });

  it("steps a string field from empty as if it were zero", async () => {
    const user = userEvent.setup();
    render(<StringHarness step={0.25} />);

    await user.click(screen.getByRole("button", { name: "Increment" }));

    expect(state()).toBe('"0.25"');
  });

  it("keeps a numeric field numeric, and clears it to null", async () => {
    const user = userEvent.setup();
    render(<NumberHarness />);

    await user.type(input("odometer"), "412113");
    expect(state()).toBe("412113");

    await user.click(screen.getByRole("button", { name: "Increment" }));
    expect(state()).toBe("412123");

    await user.clear(input("odometer"));
    expect(state()).toBe("null");
  });

  it("settles the kind from the value when the call site cannot declare one", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<UndeclaredHarness initial="1.5" />);

    await user.type(input("loose"), "5");
    expect(state()).toBe('"1.55"');
    unmount();

    render(<UndeclaredHarness initial={12} />);
    await user.type(input("loose"), "3");
    expect(state()).toBe("123");
  });
});
