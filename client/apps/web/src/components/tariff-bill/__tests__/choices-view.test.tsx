import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { tariffChoicesSchema } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChoicesView } from "../choices-view";

afterEach(cleanup);

// A `best` as tariff.evaluate reports it (rules-engine.md, "tariff.evaluate"): three options, one
// kept, one beaten and one refused, each with its source range; and a `worst` every option of
// which refused.
const choices = tariffChoicesSchema.parse([
  {
    label: "Linehaul",
    kind: "best",
    range: { from: 120, to: 400 },
    chosen: "Flat",
    options: [
      {
        label: "Per mile",
        range: { from: 140, to: 200 },
        status: "lost",
        subtotal: { $q: "1175", u: "$" },
        refusal: null,
      },
      {
        label: "Flat",
        range: { from: 203, to: 250 },
        status: "chosen",
        subtotal: { $q: "900", u: "$" },
        refusal: null,
      },
      {
        label: "Volume",
        range: { from: 253, to: 398 },
        status: "refused",
        subtotal: null,
        refusal: { message: "Volume rates start at 6,000 lb", range: { from: 270, to: 330 } },
      },
    ],
  },
  {
    label: "Floor",
    kind: "worst",
    range: { from: 410, to: 500 },
    chosen: null,
    options: [
      {
        label: "Contract",
        range: { from: 420, to: 460 },
        status: "refused",
        subtotal: null,
        refusal: {
          message: { message: "The input {0} has no value and no default", args: ["`floor`"] },
          range: null,
        },
      },
    ],
  },
]);

describe("ChoicesView", () => {
  it("shows each choice's rule and the option it kept", () => {
    render(<ChoicesView choices={choices} />);
    expect(screen.getByText("Lowest wins")).toBeInTheDocument();
    expect(screen.getByText("Chose Flat")).toBeInTheDocument();
    expect(screen.getByText("Highest wins")).toBeInTheDocument();
    expect(screen.getByText("Every option refused")).toBeInTheDocument();
  });

  it("gives every option its subtotal or its refusal, the beaten ones marked", () => {
    render(<ChoicesView choices={choices.slice(0, 1)} />);
    const rows = screen.getAllByRole("listitem");
    expect(rows.map((row) => row.getAttribute("data-status"))).toEqual([
      "lost",
      "chosen",
      "refused",
    ]);
    expect(within(rows[0]!).getByText("$1,175.00")).toHaveClass("line-through");
    expect(within(rows[1]!).getByText("$900.00")).not.toHaveClass("line-through");
    expect(within(rows[2]!).getByText("Volume rates start at 6,000 lb")).toBeInTheDocument();
    expect(within(rows[2]!).getByText("Refused")).toBeInTheDocument();
  });

  it("translates a refusal the module wrote", () => {
    render(<ChoicesView choices={choices.slice(1)} />);
    expect(screen.getByText("The input `floor` has no value and no default")).toBeInTheDocument();
  });

  it("reveals an option's source when it is picked", () => {
    const reveal = vi.fn();
    render(<ChoicesView choices={choices.slice(0, 1)} onSelectRange={reveal} />);
    fireEvent.click(screen.getByRole("button", { name: /Flat/ }));
    expect(reveal).toHaveBeenCalledWith({ from: 203, to: 250 });
  });

  it("draws nothing for a rating without choices", () => {
    const { container } = render(<ChoicesView choices={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
