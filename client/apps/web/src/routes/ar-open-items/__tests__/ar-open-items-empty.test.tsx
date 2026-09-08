import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FormFieldStub, PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { AROpenItemsPage } from "../page";

const mocks = vi.hoisted(() => ({ fetchArOpenItems: vi.fn() }));

vi.mock("@/lib/graphql/accounts-receivable", () => ({ fetchArOpenItems: mocks.fetchArOpenItems }));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/autocomplete-fields", () => ({ CustomerAutocompleteField: FormFieldStub }));
vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: FormFieldStub,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

beforeEach(() => {
  mocks.fetchArOpenItems.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AR open items empty states", () => {
  it("says every invoice is settled when nothing is open", async () => {
    renderAccountingPage(<AROpenItemsPage />);

    expect(await screen.findByText("Nothing open")).toBeInTheDocument();
    expect(screen.getByText(/paid in full/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the table", async () => {
    const user = userEvent.setup();
    renderAccountingPage(<AROpenItemsPage />);
    await screen.findByText("Nothing open");

    await user.type(screen.getByLabelText("Customer"), "cus_1");
    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Nothing open")).toBeInTheDocument();
    expect(screen.getByLabelText("Customer")).toHaveValue("");
  });
});
