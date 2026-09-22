import type { FiscalPeriod } from "@/types/fiscal-period";
import type { FiscalYear } from "@/types/fiscal-year";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FiscalYearForm } from "../fiscal-year-form";

const mocks = vi.hoisted(() => ({
  activate: vi.fn(),
  close: vi.fn(),
  closeBlockers: vi.fn(),
  lock: vi.fn(),
  reopen: vi.fn(),
  unlock: vi.fn(),
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
    fiscalPeriodService: {
      activate: mocks.activate,
      close: mocks.close,
      closeBlockers: mocks.closeBlockers,
      lock: mocks.lock,
      reopen: mocks.reopen,
      unlock: mocks.unlock,
    },
  },
}));

// Contract: services/tms/internal/core/services/fiscalperiodservice. Inactive → Open
// (activate), Open → Locked (lock), Locked → Open (unlock), Open|Locked → Closed
// (close), Closed → Open (reopen, reason required). PermanentlyClosed is terminal,
// and nothing loosens a period whose fiscal year is Closed or PermanentlyClosed.

const DAY = 86_400;
const JAN_1_2026 = 1_767_225_600;

function period(number: number, overrides: Partial<FiscalPeriod> = {}): FiscalPeriod {
  const startDate = JAN_1_2026 + (number - 1) * 31 * DAY;
  return {
    id: `fp_${String(number).padStart(2, "0")}`,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    fiscalYearId: "fy_2026",
    version: 1,
    status: "Open",
    periodType: "Month",
    periodNumber: number,
    name: `Period ${number}`,
    startDate,
    endDate: startDate + 30 * DAY,
    closedAt: null,
    ...overrides,
  };
}

function adjustingPeriod(overrides: Partial<FiscalPeriod> = {}): FiscalPeriod {
  return period(13, {
    name: "Adjusting Period - FY 2026",
    periodType: "Adjusting",
    isAdjusting: true,
    status: "Inactive",
    startDate: JAN_1_2026 + 334 * DAY,
    endDate: JAN_1_2026 + 365 * DAY - 1,
    ...overrides,
  });
}

function fiscalYear(periods: FiscalPeriod[], overrides: Partial<FiscalYear> = {}): FiscalYear {
  return {
    id: "fy_2026",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    version: 4,
    status: "Open",
    year: 2026,
    name: "FY 2026",
    description: "",
    startDate: JAN_1_2026,
    endDate: JAN_1_2026 + 365 * DAY - 1,
    isCurrent: true,
    isCalendarYear: true,
    allowAdjustingEntries: true,
    periods,
    ...overrides,
  };
}

function Harness({ year }: { year: FiscalYear }) {
  const form = useForm<FiscalYear>({ defaultValues: year });
  return (
    <FormProvider {...form}>
      <FiscalYearForm mode="edit" />
    </FormProvider>
  );
}

function renderForm(year: FiscalYear) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <Harness year={year} />
    </QueryClientProvider>,
  );
}

function rowFor(name: string) {
  const cell = screen.getByText(name);
  const row = cell.closest("tr");
  if (!row) throw new Error(`no row for ${name}`);
  return row;
}

async function openMenu(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(within(rowFor(name)).getByRole("button", { name: `Actions for ${name}` }));
}

async function menuItemNames() {
  const items = await screen.findAllByRole("menuitem");
  return items.map((item) => item.textContent?.trim());
}

beforeEach(() => {
  mocks.denied.clear();
  mocks.closeBlockers.mockResolvedValue({ canClose: true, blockers: [] });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("fiscal period actions", () => {
  it("opens the inactive adjusting period with the period's own id", async () => {
    const user = userEvent.setup();
    const months = Array.from({ length: 12 }, (_, i) => period(i + 1));
    mocks.activate.mockResolvedValue({ ...adjustingPeriod(), status: "Open", version: 2 });
    renderForm(fiscalYear([...months, adjustingPeriod()]));

    await openMenu(user, "Adjusting Period - FY 2026");
    await user.click(await screen.findByRole("menuitem", { name: /Open period/ }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Open period" }));

    expect(mocks.activate).toHaveBeenCalledWith("fp_13");
    expect(await within(rowFor("Adjusting Period - FY 2026")).findByText("Open")).toBeVisible();
  });

  it("closes a period using its id rather than the form's row key", async () => {
    const user = userEvent.setup();
    mocks.close.mockResolvedValue({ ...period(1), status: "Closed", version: 2 });
    renderForm(fiscalYear([period(1), period(2)]));

    await openMenu(user, "Period 1");
    await user.click(await screen.findByRole("menuitem", { name: /Close period/ }));
    const dialog = await screen.findByRole("alertdialog");
    const confirm = within(dialog).getByRole("button", { name: "Close period" });
    await vi.waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);

    expect(mocks.closeBlockers).toHaveBeenCalledWith("fp_01");
    expect(mocks.close).toHaveBeenCalledWith("fp_01");
    expect(await within(rowFor("Period 1")).findByText("Closed")).toBeVisible();
  });

  it("offers lock and close for an open period, unlock and close for a locked one", async () => {
    const user = userEvent.setup();
    renderForm(fiscalYear([period(1, { status: "Locked" }), period(2)]));

    await openMenu(user, "Period 1");
    expect(await menuItemNames()).toEqual(["Unlock period", "Close period"]);
    await user.keyboard("{Escape}");

    await openMenu(user, "Period 2");
    expect(await menuItemNames()).toEqual(["Lock period", "Close period"]);
  });

  it("offers only reopen for a closed period, and never lock", async () => {
    const user = userEvent.setup();
    renderForm(fiscalYear([period(1, { status: "Closed" }), period(2)]));

    await openMenu(user, "Period 1");

    expect(await menuItemNames()).toEqual(["Reopen period"]);
  });

  it("requires a reason to reopen and sends it", async () => {
    const user = userEvent.setup();
    mocks.reopen.mockResolvedValue({ ...period(1), status: "Open", version: 2 });
    renderForm(fiscalYear([period(1, { status: "Closed" }), period(2)]));

    await openMenu(user, "Period 1");
    await user.click(await screen.findByRole("menuitem", { name: /Reopen period/ }));
    const dialog = await screen.findByRole("alertdialog");
    const confirm = within(dialog).getByRole("button", { name: "Reopen period" });

    expect(confirm).toBeDisabled();
    await user.type(within(dialog).getByLabelText("Reason"), "  Late vendor invoice ");
    await user.click(confirm);

    expect(mocks.reopen).toHaveBeenCalledWith("fp_01", "Late vendor invoice");
  });

  it("locks a period and shows the new status in its row", async () => {
    const user = userEvent.setup();
    mocks.lock.mockResolvedValue({ ...period(2), status: "Locked", version: 2 });
    renderForm(fiscalYear([period(1, { status: "Closed" }), period(2)]));

    await openMenu(user, "Period 2");
    await user.click(await screen.findByRole("menuitem", { name: /Lock period/ }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Lock period" }));

    expect(mocks.lock).toHaveBeenCalledWith("fp_02");
    expect(await within(rowFor("Period 2")).findByText("Locked")).toBeVisible();
  });

  it("keeps unsaved edits to the fiscal year when a period action completes", async () => {
    const user = userEvent.setup();
    mocks.lock.mockResolvedValue({ ...period(2), status: "Locked", version: 2 });
    renderForm(fiscalYear([period(1, { status: "Closed" }), period(2)]));

    const description = screen.getByLabelText("Description");
    await user.type(description, "Audit in progress");

    await openMenu(user, "Period 2");
    await user.click(await screen.findByRole("menuitem", { name: /Lock period/ }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Lock period" }));

    expect(await within(rowFor("Period 2")).findByText("Locked")).toBeVisible();
    expect(screen.getByLabelText("Description")).toHaveValue("Audit in progress");
  });

  it("disables reopening an earlier period while a later one is closed, and says why", async () => {
    const user = userEvent.setup();
    renderForm(fiscalYear([period(1, { status: "Closed" }), period(2, { status: "Closed" })]));

    await openMenu(user, "Period 1");
    const item = await screen.findByRole("menuitem", { name: /Reopen period/ });

    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(item).toHaveTextContent("Period 2 is already closed");
  });

  it("disables loosening actions when the fiscal year is closed", async () => {
    const user = userEvent.setup();
    const months = Array.from({ length: 12 }, (_, i) => period(i + 1, { status: "Closed" }));
    renderForm(fiscalYear([...months, adjustingPeriod()], { status: "Closed", isCurrent: false }));

    await openMenu(user, "Adjusting Period - FY 2026");
    const item = await screen.findByRole("menuitem", { name: /Open period/ });

    expect(item).toHaveAttribute("aria-disabled", "true");
    expect(item).toHaveTextContent("Reopen the fiscal year first");
  });

  it("explains that a permanently closed period has no actions", async () => {
    const user = userEvent.setup();
    renderForm(fiscalYear([period(1, { status: "PermanentlyClosed" })]));

    await openMenu(user, "Period 1");
    const item = await screen.findByRole("menuitem", { name: /Permanently closed/ });

    expect(item).toHaveAttribute("aria-disabled", "true");
  });

  it("hides actions the user is not permitted to take", async () => {
    const user = userEvent.setup();
    const { Operation } = await import("@trenova/shared/types/permission");
    mocks.denied.add(Operation.Close);
    renderForm(fiscalYear([period(1)]));

    await openMenu(user, "Period 1");

    expect(await menuItemNames()).toEqual(["Lock period"]);
  });
});
