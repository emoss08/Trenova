import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { EtaCell } from "./eta-cell";

function shipmentWithEta(eta: Shipment["eta"]): Shipment {
  return { id: "shp_1", stage: "Moving", moves: [], eta } as unknown as Shipment;
}

describe("EtaCell", () => {
  afterEach(cleanup);

  it("warns on an at-risk load even when it is ahead of the window", () => {
    render(
      <EtaCell
        shipment={shipmentWithEta({
          estimatedArrival: 1_800_000_000,
          slackMinutes: 40,
          verdict: "AtRisk",
        })}
      />,
    );

    const delta = screen.getByText("40m to spare");
    expect(delta.className).toContain("text-warning");
  });

  it("shows a load past its window as late by the overrun", () => {
    render(
      <EtaCell
        shipment={shipmentWithEta({
          estimatedArrival: 1_800_000_000,
          slackMinutes: -75,
          verdict: "Late",
        })}
      />,
    );

    const delta = screen.getByText("+1h 15m");
    expect(delta.className).toContain("text-danger");
  });

  it("calls a comfortable load early", () => {
    render(
      <EtaCell
        shipment={shipmentWithEta({
          estimatedArrival: 1_800_000_000,
          slackMinutes: 120,
          verdict: "OnTime",
        })}
      />,
    );

    expect(screen.getByText("2h 0m early").className).toContain("text-success");
  });
});
