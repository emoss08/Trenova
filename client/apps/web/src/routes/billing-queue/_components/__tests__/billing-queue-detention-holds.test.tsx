import { recordPath } from "@/config/record-links";
import { cleanup, render, screen } from "@testing-library/react";
import type { DetentionHold } from "@trenova/shared/types/billing-queue";
import { MemoryRouter } from "react-router";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { BillingQueueDetentionHolds } from "../billing-queue-detention-holds";

beforeEach(() => seedPermissions({ [Resource.Detention]: Operation.Read }));
afterEach(() => {
  cleanup();
  clearPermissions();
});

function hold(overrides: Partial<DetentionHold> = {}): DetentionHold {
  return {
    occurrenceId: "dto_1",
    stopId: "stp_1",
    stopType: "Delivery",
    locationName: "Riverside DC",
    clockStartAt: 1_790_000_000,
    billableAmount: 425,
    currency: "USD",
    reason: "OverApprovalThreshold",
    ...overrides,
  };
}

function renderHolds(holds: DetentionHold[]) {
  return render(
    <MemoryRouter>
      <BillingQueueDetentionHolds holds={holds} />
    </MemoryRouter>,
  );
}

describe("BillingQueueDetentionHolds", () => {
  it("shows nothing when no detention charge is held", () => {
    renderHolds([]);

    expect(screen.queryByTestId("billing-queue-detention-holds")).not.toBeInTheDocument();
  });

  it("lists each held charge with its amount and reason, linked to the charge", () => {
    renderHolds([
      hold(),
      hold({
        occurrenceId: "dto_2",
        locationName: "",
        billableAmount: 180,
        reason: "NoticeNotSent",
      }),
    ]);

    expect(screen.getByText("2 detention charges need approval")).toBeInTheDocument();
    expect(screen.getByText("$425.00")).toBeInTheDocument();
    expect(screen.getByText("Over the approval threshold")).toBeInTheDocument();
    expect(screen.getByText("Notice not sent in time")).toBeInTheDocument();

    expect(screen.getByRole("link", { name: "Riverside DC" })).toHaveAttribute(
      "href",
      recordPath("detention_occurrence", "dto_1"),
    );
    expect(screen.getByRole("link", { name: "Detention charge" })).toHaveAttribute(
      "href",
      recordPath("detention_occurrence", "dto_2"),
    );
  });

  it("names a held charge without a link for someone who cannot open the detention desk", () => {
    seedPermissions({});
    renderHolds([hold()]);

    expect(screen.queryByRole("link", { name: "Riverside DC" })).not.toBeInTheDocument();
    expect(screen.getByText("Riverside DC").closest("[data-slot=locked-reference]")).not.toBeNull();
    expect(screen.getByText("$425.00")).toBeInTheDocument();
  });
});
