import { translate } from "@trenova/shared/i18n/runtime";
import { tariffTraceEventSchema } from "@trenova/shared/types/tariff";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { describeTraceEvent, TraceView } from "../trace-view";

// Events in the shape src/tariff/eval/trace.rs writes, with the values
// testdata/tariff/eval/{limits,reviews,so-far}.json record.
const trace = [
  {
    kind: "charge",
    label: "Linehaul",
    code: null,
    value: { $q: "450", u: "$" },
    display: "$450.00",
    limit: { kind: "minimum", computed: { $q: "200", u: "$" } },
    range: { from: 40, to: 112 },
  },
  {
    kind: "charge",
    label: "Detention",
    code: "DET",
    value: { $q: "300", u: "$" },
    display: "$300.00",
    limit: { kind: "maximum", computed: { $q: "390", u: "$" } },
    range: { from: 113, to: 170 },
  },
  {
    kind: "charge",
    label: "Fuel",
    code: "FSC",
    value: { $q: "45", u: "$" },
    display: "$45.00",
    limit: null,
    range: { from: 171, to: 200 },
  },
  { kind: "review", text: "Hazmat over 500 mi", range: { from: 59, to: 99 } },
  {
    kind: "choice",
    label: "Linehaul",
    chosen: "Flat",
    subtotal: { $q: "900", u: "$" },
    range: { from: 201, to: 330 },
  },
  { kind: "choice", label: "Stop", chosen: null, subtotal: null, range: { from: 331, to: 400 } },
].map((event) => tariffTraceEventSchema.parse(event));

describe("TraceView", () => {
  it("says a line limit set a charge's amount and what the charge came to", () => {
    expect(trace.slice(0, 3).map((event) => describeTraceEvent(event, [], translate))).toEqual([
      "Billed Linehaul: $450.00, its minimum (it came to $200.00)",
      "Billed Detention (DET): $300.00, its maximum (it came to $390.00)",
      "Billed Fuel (FSC): $45.00",
    ]);
  });

  it("shows a review and the subtotal of the option a choice kept", () => {
    expect(trace.slice(3).map((event) => describeTraceEvent(event, [], translate))).toEqual([
      "Needs review: Hazmat over 500 mi",
      "Linehaul: chose Flat at $900.00",
      "Stop: every option refused",
    ]);
  });

  it("reveals a review's statement when it is picked", () => {
    const onSelectRange = vi.fn();
    render(<TraceView trace={trace} onSelectRange={onSelectRange} />);
    const list = screen.getByRole("list", { name: "Trace" });
    fireEvent.click(within(list).getByRole("button", { name: "Needs review: Hazmat over 500 mi" }));
    expect(onSelectRange).toHaveBeenCalledWith({ from: 59, to: 99 });
  });
});
