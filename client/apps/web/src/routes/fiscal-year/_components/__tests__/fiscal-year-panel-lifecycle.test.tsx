import type { FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FiscalYearPanel } from "../fiscal-year-panel";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  close: vi.fn(),
  closePreview: vi.fn(),
  reopen: vi.fn(),
  denied: new Set<number>(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

vi.mock("react-lazy-load-image-component", () => ({
  LazyLoadComponent: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermissionCheck: () => ({
    check: (_resource: string, operation: number) => !mocks.denied.has(operation),
    checkAny: () => true,
    checkAll: () => true,
    checkRoute: () => true,
    isReady: true,
  }),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    fiscalYearService: {
      activate: mocks.activate,
      close: mocks.close,
      closePreview: mocks.closePreview,
      reopen: mocks.reopen,
    },
    fiscalPeriodService: {},
  },
}));

// Contract: fiscalyearservice. A year's status and current flag never move through
// the edit form's save; Activate (Draft or Open → Open and current), Close (Open) and
// Reopen (Closed, reason required) are the only paths, so the edit panel offers them
// in its header, next to the close button, rather than inside the form body.

const JAN_1_2026 = 1_767_225_600;

function fiscalYear(overrides: Partial<FiscalYearRow> = {}): FiscalYearRow {
  return {
    id: "fy_2026",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    version: 1,
    status: "Draft",
    year: 2026,
    name: "FY 2026",
    description: "",
    startDate: JAN_1_2026,
    endDate: JAN_1_2026 + 365 * 86_400 - 1,
    isCurrent: false,
    isCalendarYear: true,
    allowAdjustingEntries: false,
    createdAt: JAN_1_2026,
    updatedAt: JAN_1_2026,
    periods: [],
    ...overrides,
  } as FiscalYearRow;
}

function renderForm(year: FiscalYearRow) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <NuqsTestingAdapter>
      <QueryClientProvider client={client}>
        <FiscalYearPanel open mode="edit" row={year} onOpenChange={() => {}} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

function headerButton(name: string) {
  const button = screen.getByRole("button", { name });
  expect(button.closest("form")).toBeNull();
  return button;
}

beforeEach(() => {
  mocks.denied.clear();
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("fiscal year lifecycle actions in the edit panel header", () => {
  it("sets a draft year that is not current as the current year", async () => {
    const user = userEvent.setup();
    mocks.activate.mockResolvedValue({ ...fiscalYear(), status: "Open", isCurrent: true });
    renderForm(fiscalYear());

    await user.click(headerButton("Set as current"));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Set as current" }));

    expect(mocks.activate).toHaveBeenCalledWith("fy_2026");
    expect(await screen.findByText("This is the current fiscal year.")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Set as current" })).not.toBeInTheDocument();
    expect(headerButton("Close year")).toBeVisible();
  });

  it("keeps unsaved edits when the year is set as current", async () => {
    const user = userEvent.setup();
    mocks.activate.mockResolvedValue({
      ...fiscalYear(),
      status: "Open",
      isCurrent: true,
      version: 2,
    });
    renderForm(fiscalYear());

    await user.type(screen.getByLabelText("Description"), "Primary books");
    await user.click(screen.getByRole("button", { name: "Set as current" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Set as current" }));

    expect(await screen.findByText("This is the current fiscal year.")).toBeVisible();
    expect(screen.getByLabelText("Description")).toHaveValue("Primary books");
  });

  it("offers setting an open year that is not current as current", async () => {
    renderForm(fiscalYear({ status: "Open" }));

    expect(headerButton("Set as current")).toBeVisible();
    expect(headerButton("Close year")).toBeVisible();
  });

  it("offers closing the current open year but not setting it current again", () => {
    renderForm(fiscalYear({ status: "Open", isCurrent: true }));

    expect(screen.queryByRole("button", { name: "Set as current" })).not.toBeInTheDocument();
    expect(headerButton("Close year")).toBeVisible();
  });

  it("offers only reopening a closed year", () => {
    renderForm(fiscalYear({ status: "Closed" }));

    expect(screen.queryByRole("button", { name: "Set as current" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close year" })).not.toBeInTheDocument();
    expect(headerButton("Reopen year")).toBeVisible();
  });

  it("offers nothing for a permanently closed year", () => {
    renderForm(fiscalYear({ status: "PermanentlyClosed" }));

    expect(screen.queryByRole("button", { name: "Set as current" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close year" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reopen year" })).not.toBeInTheDocument();
  });

  it("hides actions the user is not permitted to take", async () => {
    const { Operation } = await import("@trenova/shared/types/permission");
    mocks.denied.add(Operation.Activate);
    renderForm(fiscalYear());

    expect(screen.queryByRole("button", { name: "Set as current" })).not.toBeInTheDocument();
  });
});
