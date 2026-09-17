import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SelectField } from "../select-field";

afterEach(cleanup);

type Values = { protocol: string };

/**
 * A vendor mark drawn to fill its box, which is how the brand-mark registry
 * hands one over: the caller cannot know what container it will land in.
 */
function FullBleedMark() {
  return <svg data-testid="mark" className="size-full" />;
}

function Harness() {
  const form = useForm<Values>({ defaultValues: { protocol: "" } });

  return (
    <SelectField
      control={form.control}
      name="protocol"
      label="Protocol"
      placeholder="Choose a protocol"
      options={[
        { value: "anthropic", label: "Anthropic Messages", icon: <FullBleedMark /> },
        { value: "plain", label: "Ollama" },
      ]}
    />
  );
}

/**
 * The row decides how big an option's icon is, not the icon.
 *
 * The list row's own `[&_svg:not([class*='size-'])]:size-4` guard cannot do it:
 * a mark carrying `size-full` matches `[class*='size-']`, so the guard skips it
 * and the mark grows to the height of the row. That is what happened to the
 * vendor logos in the Protocol picker — each one filled the whole dropdown.
 */
describe("SelectField option icons", () => {
  it("constrains an option icon that asks for its full box", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByRole("button", { name: /protocol/i }));

    const mark = await screen.findByTestId("mark");
    const slot = mark.parentElement;

    expect(slot).not.toBeNull();
    expect(slot?.getAttribute("data-slot")).toBe("command-item-icon");
    expect(slot?.className).toContain("size-4");
  });
});
