import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkerBenefitsSection } from "../benefits/worker-benefits-section";

const mocks = vi.hoisted(() => ({
  fetchWorkerBenefitEnrollments: vi.fn(),
  fetchWorkerTotalCompensation: vi.fn(),
  endBenefitEnrollment: vi.fn(),
}));

vi.mock("@/lib/graphql/benefits", () => ({
  ...mocks,
  enrollBenefit: vi.fn(),
  BENEFIT_ENROLLMENTS_KEY: "benefit-enrollments",
  TOTAL_COMPENSATION_KEY: "total-compensation",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

function enrollment(overrides: Record<string, unknown>) {
  return {
    id: "ben",
    workerId: "wrk_1",
    benefitPlanId: "plan",
    status: "Active",
    coverageTier: "Employee",
    effectiveFrom: 1_735_689_600,
    effectiveTo: null,
    employeeCostMinor: 0,
    employerCostMinor: 0,
    recurringDeductionId: null,
    waivedReason: null,
    notes: null,
    version: 3,
    benefitPlan: { id: "plan", code: "P", name: "Plan", planType: "Medical", carrier: null },
    ...overrides,
  };
}

const rows = [
  enrollment({
    id: "ben_401k",
    benefitPlanId: "plan_401k",
    status: "Pending",
    benefitPlan: {
      id: "plan_401k",
      code: "K",
      name: "401(k)",
      planType: "Retirement",
      carrier: null,
    },
  }),
  enrollment({
    id: "ben_medical",
    benefitPlanId: "plan_medical",
    coverageTier: "Family",
    employeeCostMinor: 18_240,
    employerCostMinor: 40_000,
    benefitPlan: {
      id: "plan_medical",
      code: "M",
      name: "Blue Cross PPO",
      planType: "Medical",
      carrier: null,
    },
  }),
];

function connection(nodes: { id: string }[]) {
  return {
    edges: nodes.map((node) => ({ cursor: node.id, node })),
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function renderSection() {
  const onOpen = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerBenefitsSection workerId="wrk_1" open={CLOSED_WORKER_OPEN_STATE} onOpen={onOpen} />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerBenefitsSection", () => {
  it("lists covered plans before elections still to make, with what each costs a period", async () => {
    mocks.fetchWorkerBenefitEnrollments.mockResolvedValue(connection(rows));
    mocks.fetchWorkerTotalCompensation.mockResolvedValue({
      asOf: 1,
      planYear: 2026,
      grossPayMinor: 7_124_000,
      employerBenefitMinor: 986_000,
      employeeBenefitMinor: 442_000,
      accruedTimeOffDays: 0,
      totalCompensationMinor: 8_110_000,
      enrollments: [],
    });
    renderSection();

    const enrolments = await screen.findAllByTestId(/^enrollment-/);
    expect(enrolments.map((row) => row.getAttribute("data-testid"))).toEqual([
      "enrollment-ben_medical",
      "enrollment-ben_401k",
    ]);
    expect(within(enrolments[0]).getByText("$182.40")).toBeTruthy();
    expect(within(enrolments[1]).getByText("Not started")).toBeTruthy();
    expect(await screen.findByTestId("compensation-bar")).toBeTruthy();
    expect(screen.getByText("$81,100.00")).toBeTruthy();
  });

  it("opens the election for a plan not started in the work pane, preset to that plan", async () => {
    mocks.fetchWorkerBenefitEnrollments.mockResolvedValue(connection(rows));
    mocks.fetchWorkerTotalCompensation.mockResolvedValue(null);
    const onOpen = renderSection();

    const row = within(await screen.findByTestId("enrollment-ben_401k"));
    await userEvent.setup().click(row.getByRole("button", { name: "Enroll or decline" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "pay",
      view: "benefits",
      form: "enroll-benefit",
      kind: "plan_401k",
    });
  });

  it("ends cover only after it is confirmed with a reason", async () => {
    mocks.fetchWorkerBenefitEnrollments.mockResolvedValue(connection(rows));
    mocks.fetchWorkerTotalCompensation.mockResolvedValue(null);
    mocks.endBenefitEnrollment.mockResolvedValue({ id: "ben_medical" });
    renderSection();
    const user = userEvent.setup();

    const row = within(await screen.findByTestId("enrollment-ben_medical"));
    await user.click(row.getByRole("button", { name: "Enrollment options" }));
    await user.click(await screen.findByRole("menuitem", { name: "End cover…" }));
    const dialog = await screen.findByRole("dialog", { name: "End Blue Cross PPO cover?" });
    await user.type(within(dialog).getByRole("textbox"), "Moving to a spouse's plan");
    await user.click(within(dialog).getByRole("button", { name: "End cover" }));

    await waitFor(() =>
      expect(mocks.endBenefitEnrollment).toHaveBeenCalledWith({
        id: "ben_medical",
        notes: "Moving to a spouse's plan",
        version: 3,
      }),
    );
  });

  it("asks for the first election when nothing is recorded", async () => {
    mocks.fetchWorkerBenefitEnrollments.mockResolvedValue(connection([]));
    mocks.fetchWorkerTotalCompensation.mockResolvedValue(null);
    renderSection();

    expect(await screen.findByText("No enrollments recorded")).toBeTruthy();
  });
});
