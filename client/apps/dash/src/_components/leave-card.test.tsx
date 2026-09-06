import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LeaveCard } from "./leave-card";

const { fetchMyLeave } = vi.hoisted(() => ({ fetchMyLeave: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({ fetchMyLeave }));

function makeCase(overrides: Record<string, unknown> = {}) {
  return {
    id: "lc_1",
    leaveType: "FMLA",
    status: "Approved",
    frequency: "Continuous",
    fmlaDesignated: true,
    reason: "Serious health condition",
    startsAt: 1_800_086_400,
    endsAt: null,
    decidedAt: 1_800_000_000,
    certificationStatus: "NotRequired",
    certificationDueAt: null,
    certificationLate: false,
    hoursUsed: "40",
    hoursCharged: "40",
    ...overrides,
  };
}

const baseLeave = {
  entitlement: {
    method: "RollingBackward",
    totalHours: "480",
    usedHours: "40",
    remainingHours: "440",
    totalWeeks: "12",
    usedWeeks: "1",
    remainingWeeks: "11",
    exhausted: false,
    eligibleOnTenure: true,
    monthsEmployed: 30,
    openCaseCount: 1,
    militaryCaregiver: false,
    window: { from: 1_768_000_000, through: 1_799_536_000 },
  },
  cases: [makeCase()],
};

function renderCard(leave: unknown = baseLeave) {
  fetchMyLeave.mockResolvedValue(leave);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <LeaveCard />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("LeaveCard", () => {
  it("shows the hours left against the measurement window", async () => {
    renderCard();

    expect(await screen.findByText("440")).toBeInTheDocument();
    expect(screen.getByText(/11 of 12 weeks/)).toBeInTheDocument();
  });

  // A driver who has never taken leave has no balance worth showing; an empty
  // entitlement card reads as time they are owed.
  it("renders nothing when the driver has no leave case", async () => {
    renderCard({ ...baseLeave, cases: [] });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(screen.queryByTestId("leave-card")).not.toBeInTheDocument();
  });

  // Once the deadline passes the employer may deny the leave, so the warning
  // has to say so rather than just naming the paperwork.
  it("warns when a certification is past its deadline", async () => {
    renderCard({
      ...baseLeave,
      cases: [
        makeCase({
          certificationStatus: "Overdue",
          certificationDueAt: 1_800_200_000,
          certificationLate: true,
        }),
      ],
    });

    expect(
      await screen.findByText(/past its deadline — leave can be denied once it is/),
    ).toBeInTheDocument();
  });

  // FMLA runs concurrently with paid leave, and a day on an undesignated case
  // is time away that drew nothing down. The two figures must be readable apart.
  it("separates the hours taken from the hours charged", async () => {
    renderCard({
      ...baseLeave,
      cases: [makeCase({ hoursUsed: "40", hoursCharged: "24" })],
    });

    expect(await screen.findByText(/40 h taken \(24 h counted\)/)).toBeInTheDocument();
  });
});
