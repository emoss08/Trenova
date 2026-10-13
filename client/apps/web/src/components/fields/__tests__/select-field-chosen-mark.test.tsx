import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SelectField } from "../select-field";

afterEach(cleanup);

type Values = { provider: string };

const OPTIONS = [
  { value: "Comdata", label: "Comdata" },
  { value: "WEX", label: "WEX" },
];

function Harness({ renderOption }: { renderOption?: (option: { label: string }) => string }) {
  const form = useForm<Values>({ defaultValues: { provider: "WEX" } });

  return (
    <SelectField
      control={form.control}
      name="provider"
      label="Provider"
      options={OPTIONS}
      renderOption={renderOption}
    />
  );
}

function checkSlotOf(label: string): Element | null {
  const row = screen.getAllByText(label).at(-1)?.closest("[data-slot=command-item]");
  return row?.querySelector("[data-slot=command-item-check]") ?? null;
}

/**
 * Every row keeps a slot for the chosen mark at its end, so a long label stops
 * before the mark rather than running under it, and only the chosen row fills it.
 */
describe("SelectField chosen mark", () => {
  it.each([
    ["the default row", undefined],
    ["a custom row", (option: { label: string }) => option.label],
  ])("reserves the mark on every row and fills it on the chosen one in %s", async (_, render_) => {
    const user = userEvent.setup();
    render(<Harness renderOption={render_} />);

    await user.click(screen.getByRole("button", { name: "WEX" }));

    const chosen = checkSlotOf("WEX");
    const other = checkSlotOf("Comdata");

    expect(chosen?.hasAttribute("data-checked")).toBe(true);
    expect(chosen?.querySelector("svg")).not.toBeNull();
    expect(other).not.toBeNull();
    expect(other?.hasAttribute("data-checked")).toBe(false);
    expect(other?.querySelector("svg")).toBeNull();
    expect(chosen?.parentElement?.lastElementChild).toBe(chosen);
  });
});
