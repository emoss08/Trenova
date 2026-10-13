import { cleanup, render, screen } from "@testing-library/react";
import { stopEtaSchema, type StopEta } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { StopEtaLine } from "../stop-eta";

function eta(overrides: Partial<StopEta>): StopEta {
  return stopEtaSchema.parse({
    stopId: "stp_1",
    arrived: false,
    estimatedArrival: 1_800_000_000,
    slackMinutes: 45,
    verdict: "AtRisk",
    basis: "Position",
    driveMethod: "Routed",
    milesRemaining: 120,
    driveMinutes: 150,
    restMinutes: 0,
    inputStale: false,
    thresholdMinutes: 60,
    computedAt: 1_799_999_000,
    ...overrides,
  });
}

describe("StopEtaLine", () => {
  afterEach(cleanup);

  it("shows the verdict and projected arrival of a stop the truck has not reached", () => {
    render(<StopEtaLine eta={eta({})} />);

    expect(screen.getByText("At risk")).toBeTruthy();
    expect(screen.getByTestId("stop-eta-line").textContent).toContain("ETA ");
  });

  it("names a projection past the window as projected late", () => {
    render(<StopEtaLine eta={eta({ verdict: "Late", slackMinutes: -30 })} />);

    expect(screen.getByText("Projected late")).toBeTruthy();
  });

  it("shows nothing once the truck is at the stop", () => {
    render(<StopEtaLine eta={eta({ arrived: true })} />);

    expect(screen.queryByTestId("stop-eta-line")).toBeNull();
  });

  it("shows nothing for a stop with no stored projection", () => {
    render(<StopEtaLine eta={undefined} />);

    expect(screen.queryByTestId("stop-eta-line")).toBeNull();
  });

  it("says when there is nothing to estimate from without inventing an arrival", () => {
    render(
      <StopEtaLine
        eta={eta({ verdict: "Unknown", estimatedArrival: null, slackMinutes: null, basis: "None" })}
      />,
    );

    expect(screen.getByText("No projection")).toBeTruthy();
    expect(screen.getByTestId("stop-eta-line").textContent).not.toContain("ETA ");
  });
});
