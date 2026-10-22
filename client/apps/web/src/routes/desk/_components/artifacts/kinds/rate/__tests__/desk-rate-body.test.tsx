import { cleanup, render, screen } from "@testing-library/react";
import type { AssistantArtifact } from "@/types/assistant";
import { afterEach, describe, expect, it } from "vitest";
import { DeskRateBody } from "../desk-rate-body";

afterEach(cleanup);

// A rate explanation as the agent's explain-rate tool writes it: the contract that won and its
// tariff's bill in wire form, a coded accessorial and a minimum adjustment among its lines.
function artifact(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "thr_1",
    kind: "rate_explanation",
    status: "Ready",
    title: "Why this rate",
    payload,
    sourceToolCallId: "",
    pinned: false,
  } as unknown as AssistantArtifact;
}

const bill = {
  currency: "USD",
  lines: [
    {
      kind: "Charge",
      label: "Linehaul",
      freight: true,
      amount: "1908.2",
      quantity: "812",
      quantityUnit: "mi",
      rate: "2.35",
      rateUnit: "$/mi",
    },
    { kind: "Charge", label: "Hazmat", code: "HAZ", freight: false, amount: "150" },
    { kind: "Minimum", freight: true, amount: "41.8" },
  ],
  notes: [{ text: "Hazmat priced per contract schedule B" }],
  freightTotal: "1950",
  accessorialTotal: "150",
  total: "2100",
};

describe("DeskRateBody", () => {
  it("lays the tariff's bill out as a ledger, labels as written and adjustments named", () => {
    render(
      <DeskRateBody
        artifact={artifact({
          currency: "USD",
          winner: { agreementCode: "ACME-2026", agreementName: "Acme", ruleLabel: "TX to IL" },
          bill,
          totals: { total: "2100" },
        })}
      />,
    );

    expect(screen.getByText("Linehaul")).toBeInTheDocument();
    expect(screen.getByText("812 mi × $2.35/mi")).toBeInTheDocument();
    expect(screen.getByText("HAZ")).toBeInTheDocument();
    expect(screen.getByText("Minimum charge adjustment")).toBeInTheDocument();
    expect(screen.getByText("Hazmat priced per contract schedule B")).toBeInTheDocument();
    expect(screen.getByText("$2,100.00")).toBeInTheDocument();
  });

  it("names the tariff that priced a shipment no agreement covered", () => {
    render(
      <DeskRateBody
        artifact={artifact({
          currency: "USD",
          winner: null,
          pricedBy: { method: "Dry van TL", explanation: "No contract covered the lane." },
          bill,
        })}
      />,
    );

    expect(screen.getByText("Dry van TL")).toBeInTheDocument();
    expect(screen.getByText("Tariff")).toBeInTheDocument();
    expect(screen.getByText("No contract covered the lane.")).toBeInTheDocument();
  });

  it("draws an empty ledger with the stated total when the bill is not readable", () => {
    render(
      <DeskRateBody
        artifact={artifact({ currency: "USD", bill: { lines: "broken" }, totals: { total: "90" } })}
      />,
    );

    expect(screen.getByText("$90.00")).toBeInTheDocument();
  });
});
