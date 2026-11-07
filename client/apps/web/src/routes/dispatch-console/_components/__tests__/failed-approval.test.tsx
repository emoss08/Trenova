import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { FailedApproval } from "../failed-approval";

afterEach(cleanup);

// Written from DispatchMoveFailedProposal in dispatch_console.graphqls.
const failed = {
  id: "ap_9",
  workerId: "wrk_1",
  workerName: "Dana Ortiz",
  reason:
    "Move 1, stop 1: Scheduled window end must be before the next stop's scheduled window start",
  failedAt: 1_800_000_500,
};

function renderApproval(overrides: Partial<typeof failed> = {}) {
  render(
    <MemoryRouter>
      <FailedApproval move={{ shipmentId: "shp_1" }} failed={{ ...failed, ...overrides }} />
    </MemoryRouter>,
  );
}

describe("FailedApproval", () => {
  it("says who was approved, that it wasn't assigned, and why", () => {
    renderApproval();

    expect(
      screen.getByText(/Approved Dana Ortiz at .+, but it wasn't assigned/),
    ).toBeInTheDocument();
    expect(screen.getByText(failed.reason)).toBeInTheDocument();
  });

  it("links to the shipment, where what refused it is fixed", () => {
    renderApproval();

    expect(screen.getByRole("link", { name: "Open shipment" })).toHaveAttribute(
      "href",
      expect.stringContaining("shp_1"),
    );
  });

  it("still reads when the approved driver could not be named", () => {
    renderApproval({ workerName: "" });

    expect(screen.getByText(/^Approved at .+, but it wasn't assigned$/)).toBeInTheDocument();
  });
});
