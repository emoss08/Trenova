import type { FactoringAgreement } from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  FactoringAgreementActivity,
  FactoringAgreementBookedTo,
  FactoringAgreementHistory,
} from "../factoring-agreement-panels";

const mocks = vi.hoisted(() => ({ fetchFactoringAgreementActivity: vi.fn() }));

vi.mock("@/lib/graphql/factoring-company-page", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-company-page")>()),
  fetchFactoringAgreementActivity: mocks.fetchFactoringAgreementActivity,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const AGREEMENT: FactoringAgreement = {
  id: "fag_1",
  factoringCompanyId: "fac_1",
  status: "Active",
  effectiveFrom: 1_695_859_200,
  effectiveTo: null,
  advanceRateBp: 9300,
  reserveRateBp: 300,
  minimumFeeMinor: 1000,
  chargebackWindowDays: 90,
  recourse: "Recourse",
  accountingTreatment: "SecuredBorrowing",
  fundingDays: 1,
  submissionTime: 600,
  dueFromFactorAccountId: "gla_1",
  feeExpenseAccountId: "gla_3",
  liabilityAccountId: "gla_2",
  cashAccountId: null,
  notes: "Signed master purchase agreement.",
  activatedAt: 1_695_859_200,
  endedAt: null,
  version: 3,
  createdAt: 1_694_044_800,
  updatedAt: 1_695_859_200,
  feeTiers: [
    { id: "t2", daysFrom: 30, daysTo: null, rateBp: 325, flatMinor: 0 },
    { id: "t1", daysFrom: 0, daysTo: 30, rateBp: 250, flatMinor: 0 },
  ],
  documentTypes: [],
  dueFromFactorAccount: { id: "gla_1", accountCode: "1150", name: "Due From Factor" },
  feeExpenseAccount: { id: "gla_3", accountCode: "6840", name: "Factoring Fees" },
  liabilityAccount: { id: "gla_2", accountCode: "2420", name: "Line of Credit" },
  cashAccount: null,
} as FactoringAgreement;

function withClient(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("FactoringAgreementActivity", () => {
  it("shows what the schedules sent under the agreement carried, as the server counted it", async () => {
    mocks.fetchFactoringAgreementActivity.mockResolvedValue({
      schedules: 591,
      fundedSchedules: 567,
      invoices: 1429,
      faceMinor: 622_886_499,
      expectedAdvanceMinor: 579_284_449,
      expectedFeeMinor: 19_551_728,
      firstSentAt: 1_731_196_800,
      lastSentAt: 1_762_214_400,
    });
    render(withClient(<FactoringAgreementActivity agreement={AGREEMENT} />));

    await waitFor(() => expect(screen.getByText("591")).toBeTruthy());
    expect(mocks.fetchFactoringAgreementActivity).toHaveBeenCalledWith("fag_1", {
      signal: expect.any(AbortSignal),
    });
    expect(screen.getByText("567 funded")).toBeTruthy();
    expect(screen.getByText("1,429")).toBeTruthy();
    expect(screen.getByText("$6,228,864.99")).toBeTruthy();
    expect(screen.getByText("$5,792,844.49 advance expected")).toBeTruthy();
    expect(screen.getByText("$195,517.28")).toBeTruthy();
    expect(screen.getByText("3.14% of face value")).toBeTruthy();
  });

  it("says nothing has been sent when the agreement has no schedules", async () => {
    mocks.fetchFactoringAgreementActivity.mockResolvedValue({
      schedules: 0,
      fundedSchedules: 0,
      invoices: 0,
      faceMinor: 0,
      expectedAdvanceMinor: 0,
      expectedFeeMinor: 0,
      firstSentAt: null,
      lastSentAt: null,
    });
    render(withClient(<FactoringAgreementActivity agreement={AGREEMENT} />));

    expect(
      await screen.findByText("No schedule has been sent under this agreement yet."),
    ).toBeTruthy();
  });

  it("offers a retry when the read fails", async () => {
    mocks.fetchFactoringAgreementActivity.mockRejectedValue(new Error("boom"));
    render(withClient(<FactoringAgreementActivity agreement={AGREEMENT} />));

    expect(
      await screen.findByText("What this agreement carried could not be loaded."),
    ).toBeTruthy();
    expect(screen.getByRole("button", { name: "Retry" })).toBeTruthy();
  });
});

describe("FactoringAgreementBookedTo", () => {
  it("names each account the agreement books to and falls back to the organization default", () => {
    render(<FactoringAgreementBookedTo agreement={AGREEMENT} />);

    expect(screen.getByText("Line of Credit")).toBeTruthy();
    expect(screen.getByText("What is owed the factor")).toBeTruthy();
    expect(screen.getByText("Factoring Fees")).toBeTruthy();
    const cash = screen.getByText("Cash").parentElement as HTMLElement;
    expect(within(cash).getByText("Organization default")).toBeTruthy();
  });

  it("explains an empty account behind an info icon beside the title, not inline", () => {
    render(<FactoringAgreementBookedTo agreement={AGREEMENT} />);

    expect(screen.getByRole("img", { name: "About Booked to" })).toBeTruthy();
    expect(
      screen.queryByText("An empty account uses your organization's factoring default."),
    ).toBeNull();
  });

  it("marks the account a treatment does not use", () => {
    const { rerender } = render(<FactoringAgreementBookedTo agreement={AGREEMENT} />);
    expect(screen.getByText("Not used for secured borrowing")).toBeTruthy();

    rerender(
      <FactoringAgreementBookedTo agreement={{ ...AGREEMENT, accountingTreatment: "Sale" }} />,
    );
    expect(screen.getByText("The reserve the factor holds")).toBeTruthy();
    expect(screen.getByText("Not used for a sale")).toBeTruthy();
  });
});

describe("FactoringAgreementHistory", () => {
  const NOW = 1_762_560_000;

  it("lists what changed from the version before it", () => {
    const previous = {
      ...AGREEMENT,
      id: "fag_0",
      advanceRateBp: 9000,
      fundingDays: 2,
    } as FactoringAgreement;
    render(
      <FactoringAgreementHistory
        agreement={AGREEMENT}
        previous={previous}
        previousNumber={1}
        nowUnix={NOW}
      />,
    );

    expect(screen.getByText("Changes from v1")).toBeTruthy();
    expect(screen.getByText("advance 90%→93%")).toBeTruthy();
    expect(screen.getByText("funding 2→1 days")).toBeTruthy();
    expect(screen.getByText("Signed master purchase agreement.")).toBeTruthy();
  });

  it("says the first agreement has nothing to compare with and runs until someone ends it", () => {
    render(
      <FactoringAgreementHistory
        agreement={AGREEMENT}
        previous={null}
        previousNumber={null}
        nowUnix={NOW}
      />,
    );

    expect(
      screen.getByText("The first agreement with this factor; nothing to compare it with."),
    ).toBeTruthy();
    expect(screen.getByText("In force from")).toBeTruthy();
    expect(screen.getByText("No end date; runs until someone ends it")).toBeTruthy();
  });

  it("names a start still ahead as when it takes effect", () => {
    render(
      <FactoringAgreementHistory
        agreement={{ ...AGREEMENT, effectiveFrom: NOW + 86_400, activatedAt: null }}
        previous={null}
        previousNumber={null}
        nowUnix={NOW}
      />,
    );

    expect(screen.getByText("Takes effect")).toBeTruthy();
    expect(screen.getByText("Not yet")).toBeTruthy();
  });
});
