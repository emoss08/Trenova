import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FormFieldStub, PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { ARAgingPage } from "../page";

const mocks = vi.hoisted(() => ({ fetchArAgingSummary: vi.fn() }));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchArAgingSummary: mocks.fetchArAgingSummary,
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/autocomplete-fields", () => ({ CustomerAutocompleteField: FormFieldStub }));
vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: FormFieldStub,
}));
vi.mock("../_components/aging-summary-header", () => ({ AgingSummaryHeader: () => null }));

const ZERO = {
  currentMinor: 0,
  days1To30Minor: 0,
  days31To60Minor: 0,
  days61To90Minor: 0,
  daysOver90Minor: 0,
  totalOpenMinor: 0,
};

beforeEach(() => {
  mocks.fetchArAgingSummary.mockResolvedValue({ rows: [], totals: ZERO });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AR aging empty states", () => {
  it("says the book is clear when nobody owes anything", async () => {
    renderAccountingPage(<ARAgingPage />);

    expect(await screen.findByText("Nothing outstanding")).toBeInTheDocument();
    expect(screen.getByText(/invoice posts/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // A customer filter that hides every row is not the same as a clear book;
  // the way forward is to drop the filter, and the button does that.
  it("offers to clear the filters when they are what emptied the table", async () => {
    const user = userEvent.setup();
    renderAccountingPage(<ARAgingPage />);
    await screen.findByText("Nothing outstanding");

    await user.type(screen.getByLabelText("Customer"), "cus_1");
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Nothing outstanding")).toBeInTheDocument();
    expect(screen.getByLabelText("Customer")).toHaveValue("");
  });
});
