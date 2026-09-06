import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { utcMidnight } from "@trenova/shared/lib/holiday";
import { afterEach, describe, expect, it, vi } from "vitest";
import { HolidayCalendar } from "../holiday-calendar";

const { fetchOrgHolidays, deleteOrgHoliday, toast } = vi.hoisted(() => ({
  fetchOrgHolidays: vi.fn(),
  deleteOrgHoliday: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/org-holiday", () => ({
  fetchOrgHolidays,
  deleteOrgHoliday,
  createOrgHoliday: vi.fn(),
  updateOrgHoliday: vi.fn(),
  ORG_HOLIDAYS_KEY: "org-holidays",
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../holiday-dialog", () => ({
  HolidayDialog: ({ open, entry }: { open: boolean; entry?: { name: string } | null }) =>
    open ? <div data-testid="holiday-dialog">{entry ? `Edit ${entry.name}` : "New"}</div> : null,
}));

const YEAR = new Date().getUTCFullYear();

const rows = [
  {
    id: "ohol_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    name: "Independence Day",
    holidayDate: utcMidnight(YEAR - 1, 6, 4),
    kind: "Holiday" as const,
    recursAnnually: true,
    description: null,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
  },
  {
    id: "ohol_2",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    name: "Peak freeze",
    holidayDate: utcMidnight(YEAR, 11, 22),
    kind: "Blackout" as const,
    recursAnnually: false,
    description: "No new time off during the surge",
    version: 1,
    createdAt: 1,
    updatedAt: 1,
  },
];

function renderCalendar() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <HolidayCalendar />
    </QueryClientProvider>,
  );
}

describe("HolidayCalendar", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lays out the year with every month and lists each entry under its month", async () => {
    fetchOrgHolidays.mockResolvedValue(rows);
    renderCalendar();

    expect(await screen.findByText("Independence Day")).toBeInTheDocument();
    expect(fetchOrgHolidays).toHaveBeenCalledWith(YEAR, { signal: expect.any(AbortSignal) });
    expect(screen.getAllByRole("heading", { level: 3 })).toHaveLength(12);
    expect(screen.getByText("Peak freeze")).toBeInTheDocument();
    expect(screen.getByText("Blackout")).toBeInTheDocument();
    expect(screen.getByText("Every year")).toBeInTheDocument();
  });

  it("steps between years and refetches", async () => {
    fetchOrgHolidays.mockResolvedValue(rows);
    renderCalendar();
    await screen.findByText("Independence Day");

    fireEvent.click(screen.getByRole("button", { name: "Next year" }));
    await waitFor(() =>
      expect(fetchOrgHolidays).toHaveBeenCalledWith(YEAR + 1, { signal: expect.any(AbortSignal) }),
    );
    expect(screen.getByText(String(YEAR + 1))).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "This year" }));
    await waitFor(() => expect(screen.getByText(String(YEAR))).toBeInTheDocument());
  });

  it("opens the editor for an entry and removes one after confirming", async () => {
    fetchOrgHolidays.mockResolvedValue(rows);
    deleteOrgHoliday.mockResolvedValue(true);
    renderCalendar();
    await screen.findByText("Peak freeze");

    fireEvent.click(screen.getByRole("button", { name: "Add date" }));
    expect(screen.getByTestId("holiday-dialog")).toHaveTextContent("New");

    fireEvent.click(screen.getByRole("button", { name: "Edit Peak freeze" }));
    expect(screen.getByTestId("holiday-dialog")).toHaveTextContent("Edit Peak freeze");

    fireEvent.click(screen.getByRole("button", { name: "Remove Peak freeze" }));
    fireEvent.click(await screen.findByRole("button", { name: "Remove" }));
    await waitFor(() => expect(deleteOrgHoliday).toHaveBeenCalledWith("ohol_2"));
    expect(toast.success).toHaveBeenCalled();
  });
});
