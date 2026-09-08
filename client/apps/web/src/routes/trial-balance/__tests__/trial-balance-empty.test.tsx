import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { TrialBalancePage } from "../page";

const mocks = vi.hoisted(() => ({ getTrialBalance: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { accountingReportService: { getTrialBalance: mocks.getTrialBalance } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/accounting/fiscal-period-selector", () => ({
  FiscalPeriodSelector: ({ onChange }: { onChange: (id: string) => void }) => (
    <button type="button" onClick={() => onChange("fp_1")}>
      Pick period
    </button>
  ),
}));

beforeEach(() => {
  mocks.getTrialBalance.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("trial balance empty states", () => {
  it("asks for a period, then says nothing posted to the one chosen", async () => {
    const user = userEvent.setup();
    renderAccountingPage(<TrialBalancePage />);

    expect(screen.getByText("Pick a period")).toBeInTheDocument();
    expect(mocks.getTrialBalance).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Pick period" }));
    expect(await screen.findByText("Nothing posted")).toBeInTheDocument();
    expect(screen.queryByText("Pick a period")).not.toBeInTheDocument();
  });
});
