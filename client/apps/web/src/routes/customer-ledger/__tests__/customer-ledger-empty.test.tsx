import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FormFieldStub, PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { CustomerLedgerPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchArCustomerLedger: vi.fn(),
  fetchArCustomerProfile: vi.fn(),
}));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchArCustomerLedger: mocks.fetchArCustomerLedger,
  fetchArCustomerProfile: mocks.fetchArCustomerProfile,
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/autocomplete-fields", () => ({ CustomerAutocompleteField: FormFieldStub }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../_components/customer-snapshot-header", () => ({
  CustomerSnapshotHeader: () => null,
}));

beforeEach(() => {
  mocks.fetchArCustomerLedger.mockResolvedValue([]);
  mocks.fetchArCustomerProfile.mockResolvedValue(null);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("customer ledger empty states", () => {
  it("asks for a customer before there is a ledger to show", () => {
    renderAccountingPage(<CustomerLedgerPage />);

    expect(screen.getByText("Pick a customer")).toBeInTheDocument();
    expect(mocks.fetchArCustomerLedger).not.toHaveBeenCalled();
  });

  it("says nothing has posted when the chosen customer has no activity", async () => {
    renderAccountingPage(<CustomerLedgerPage />, ["/ledger?customerId=cus_1"]);

    expect(await screen.findByText("No activity yet")).toBeInTheDocument();
    expect(screen.getByText(/first invoice or payment/)).toBeInTheDocument();
    expect(screen.queryByText("Pick a customer")).not.toBeInTheDocument();
  });
});
