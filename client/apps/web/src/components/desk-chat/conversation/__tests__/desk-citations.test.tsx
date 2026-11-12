import type { ToolStep } from "@/components/assistant/activity";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { CITATION_HREF, type Citation } from "../citations";
import { useCitationOverrides } from "../desk-citations";

afterEach(cleanup);

function step(reasoning: string, call: Partial<ToolStep> = {}): ToolStep {
  return {
    id: "call_1",
    name: "shipment_get",
    arguments: {},
    status: "done",
    content: "",
    effect: "lookup",
    summary: "In transit",
    durationSeconds: 1.2,
    reasoning,
    ...call,
  };
}

function Mark({ citation }: { citation: Citation }): ReactNode {
  const overrides = useCitationOverrides([citation], [], () => {});
  const Link = overrides?.a;
  if (!Link || typeof Link === "string") return null;
  return <Link href={`${CITATION_HREF}1`}>1</Link>;
}

function openMark(reasoning: string, call: Partial<ToolStep> = {}) {
  render(<Mark citation={{ n: 1, offset: 0, step: step(reasoning, call) }} />);
  fireEvent.mouseEnter(screen.getByRole("button", { name: "Step 1" }).parentElement!);
}

/**
 * A citation's card shows the step's own reasoning: what the model thought in
 * the reply that asked for the call, folded until the reader opens it. A step
 * whose reply returned no readable thinking offers nothing to open.
 */
describe("a citation's step card", () => {
  it("folds the model's reasoning for the step under the step", () => {
    openMark("S1 is the load the person named; look it up first.");

    const fold = screen.getByRole("button", { name: "Thought it through" });
    expect(screen.queryByText("S1 is the load the person named; look it up first.")).toBeNull();

    fireEvent.click(fold);

    expect(
      screen.getByText("S1 is the load the person named; look it up first."),
    ).toBeInTheDocument();
  });

  it("offers no reasoning when the model returned none", () => {
    openMark("");

    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Thought it through" })).toBeNull();
  });

  // The card is read by the person the reply is for. The tool's code name
  // and the ids the call was narrowed by mean nothing to them.
  it("names the step in words, with no tool name and no record id", () => {
    openMark("", {
      name: "shipment_list",
      summary: "3 shipments",
      arguments: {
        filters: [
          {
            field: "customerId",
            operator: "in",
            values: ["cus_01JW3K18A8TPC7T7SPX7745V1H", "cus_01JSV4G0Q80KNYKQ254QWT5T0V"],
          },
        ],
      },
    });

    const card = screen.getByRole("tooltip");
    expect(card).toHaveTextContent("Find shipments");
    expect(card).toHaveTextContent("Customer");
    expect(card).toHaveTextContent("3 shipments");
    expect(card).not.toHaveTextContent("shipment_list");
    expect(card).not.toHaveTextContent("customerId");
    expect(card).not.toHaveTextContent("cus_01");
  });

  // The server names the records a read was narrowed by; the card says
  // them, in place of the bare field the call's arguments alone can give.
  it("names the records the server said the read was narrowed by", () => {
    openMark("", {
      name: "shipment_list",
      summary: "3 shipments",
      asked: "Customer Granite Textiles, Granite Brands · Status New, Delayed",
      arguments: {
        filters: [
          { field: "customerId", operator: "in", values: ["cus_01JW3K18A8TPC7T7SPX7745V1H"] },
        ],
      },
    });

    expect(screen.getByRole("tooltip")).toHaveTextContent(
      "Customer Granite Textiles, Granite Brands · Status New, Delayed",
    );
  });

  it("takes out an id that reached the card in the server's account", () => {
    openMark("", {
      name: "shipment_list",
      summary: "3 shipments",
      asked: "Customer Granite Textiles, cus_01JW3K18A8TPC7T7SPX7745V1H",
    });

    const card = screen.getByRole("tooltip");
    expect(card).toHaveTextContent("Customer Granite Textiles");
    expect(card).not.toHaveTextContent("cus_01");
  });
});
