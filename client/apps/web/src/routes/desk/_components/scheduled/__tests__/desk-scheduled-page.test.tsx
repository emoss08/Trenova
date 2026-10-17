import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskScheduledPage } from "../desk-scheduled-page";
import { nextScheduleOffset, SCHEDULE_PAGE_SIZE } from "../schedules-query";

const server = vi.hoisted(() => ({
  get: vi.fn(),
  patch: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}));
const access = vi.hoisted(() => ({ canRun: true }));
const toasts = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, ...server } };
});

vi.mock("@/hooks/use-permission", async () => {
  const { Operation } = await import("@trenova/shared/types/permission");
  return {
    usePermission: (resource: string, operation: number) => ({
      allowed: resource === "assistant" && operation === Operation.Create ? access.canRun : true,
      isLoading: false,
    }),
  };
});

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: Object.assign(vi.fn(), toasts),
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({ user: { id: "usr_1", name: "Avery Lane", timezone: "America/Chicago" } }),
}));

// Saturday 3 October 2026, noon in Chicago.
const saturdayNoon = 1791046800;

/** A schedule as GET /assistant/schedules/ lists it (conversationscheduleports.ListedConversationSchedule). */
function listed(overrides: Record<string, unknown> & { id: string }) {
  return {
    businessUnitId: "bu_1",
    organizationId: "org_1",
    threadId: "athr_ar",
    userId: "usr_1",
    prompt: "Send me invoices more than 60 days past due",
    cadence: "Every Monday · 8:00 AM",
    cronExpression: "0 8 * * 1",
    timezone: "America/Chicago",
    enabled: true,
    lastRunAt: null,
    // Monday 5 October, 8:00 in Chicago.
    nextRunAt: 1791205200,
    lastTurnId: null,
    version: 0,
    createdAt: saturdayNoon - 86400,
    updatedAt: saturdayNoon - 86400,
    threadTitle: "AR follow-ups",
    ...overrides,
  };
}

const invoices = listed({
  id: "csch_invoices",
  // Friday 2 October, 8:00 in Chicago.
  lastRunAt: 1790946000,
  lastTurnId: "atrn_1",
});
const dispatch = listed({
  id: "csch_dispatch",
  threadId: "athr_untitled",
  threadTitle: "",
  prompt: "List shipments not yet dispatched",
  cadence: "Every weekday · 7:30 AM",
  cronExpression: "30 7 * * 1-5",
  // Monday 5 October, 7:30 in Chicago.
  nextRunAt: 1791203400,
});
const permits = listed({
  id: "csch_permits",
  prompt: "Permits expiring this month",
  enabled: false,
});

function respondWith(...pages: { items: unknown[]; total: number }[]) {
  server.get.mockImplementation(async (url: string) => {
    const offset = Number(new URL(url, "http://desk").searchParams.get("offset"));
    let start = 0;
    for (const page of pages) {
      if (start === offset) {
        return page;
      }
      start += page.items.length;
    }
    return { items: [], total: pages[0]?.total ?? 0 };
  });
}

function LocationProbe() {
  return <span data-testid="location">{useLocation().pathname}</span>;
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/desk/scheduled"]}>
        <Routes>
          <Route path="*" element={<DeskScheduledPage />} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function row(prompt: string): HTMLElement {
  return screen.getByText(prompt).closest("li") as HTMLElement;
}

/** Opens a row's menu and chooses one of its actions. */
async function chooseFromMenu(scope: HTMLElement, action: string) {
  fireEvent.click(within(scope).getByRole("button", { name: "Schedule actions" }));
  fireEvent.click(await screen.findByRole("menuitem", { name: action }));
}

function group(name: string): HTMLElement {
  return screen.getByRole("region", { name });
}

/*
The Scheduled page is the one place a person sees every request they asked
the Desk to repeat. It lists them from GET /assistant/schedules/, which reads
the soonest first and the paused ones after, each with the title of the
conversation its answers post into.
*/
describe("DeskScheduledPage", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(saturdayNoon * 1000);
    installIntersectionObserver();
    access.canRun = true;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("lists what is coming up, soonest first, then what is paused", async () => {
    respondWith({ items: [dispatch, invoices, permits], total: 3 });
    renderPage();

    const upcoming = await screen.findByRole("region", { name: "Coming up" });
    expect(
      within(upcoming)
        .getAllByRole("listitem")
        .map((item) => item.querySelector("b")?.textContent),
    ).toEqual(["List shipments not yet dispatched", "Send me invoices more than 60 days past due"]);
    expect(
      within(group("Paused"))
        .getAllByRole("listitem")
        .map((item) => item.querySelector("b")?.textContent),
    ).toEqual(["Permits expiring this month"]);
    expect(server.get).toHaveBeenCalledWith(
      `/assistant/schedules/?limit=${SCHEDULE_PAGE_SIZE}&offset=0`,
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });

  it("says when each runs, where its answers post, and when it last ran", async () => {
    respondWith({ items: [dispatch, invoices, permits], total: 3 });
    renderPage();

    const ar = within(await waitFor(() => row("Send me invoices more than 60 days past due")));
    expect(ar.getByText("Every Monday · 8:00 AM · next Mon, Oct 5 · 8:00 AM")).toBeInTheDocument();
    expect(ar.getByRole("link", { name: "Open AR follow-ups" })).toHaveAttribute(
      "href",
      "/desk/t/athr_ar",
    );
    expect(ar.getByText("Last ran Fri, Oct 2 · 8:00 AM")).toBeInTheDocument();

    const untitled = within(row("List shipments not yet dispatched"));
    expect(untitled.getByText("Untitled conversation")).toBeInTheDocument();
    expect(untitled.getByText("Hasn't run yet")).toBeInTheDocument();

    expect(
      within(row("Permits expiring this month")).getByText("Every Monday · 8:00 AM · paused"),
    ).toBeInTheDocument();
  });

  it("shows how to schedule something when nothing is scheduled", async () => {
    respondWith({ items: [], total: 0 });
    renderPage();

    expect(await screen.findByText("Nothing scheduled yet")).toBeInTheDocument();
    expect(screen.getByText("/schedule")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "New conversation" }));
    expect(screen.getByTestId("location")).toHaveTextContent(/^\/desk$/);
  });

  it("draws the list's outline while it is read", async () => {
    server.get.mockImplementation(() => new Promise(() => {}));
    renderPage();

    expect(screen.getByLabelText("Loading scheduled requests")).toHaveAttribute(
      "aria-busy",
      "true",
    );
  });

  it("offers to read the list again when it could not be read", async () => {
    server.get.mockRejectedValueOnce(new Error("unavailable"));
    renderPage();

    const retry = await screen.findByRole("button", { name: "Try again" });
    respondWith({ items: [invoices], total: 1 });
    fireEvent.click(retry);
    expect(await screen.findByText("Send me invoices more than 60 days past due")).toBeVisible();
  });

  it("moves a paused schedule to Paused at once and tells the server", async () => {
    respondWith({ items: [dispatch, invoices], total: 2 });
    // A pause is answered with the schedule alone, without its conversation's title.
    const paused: Record<string, unknown> = listed({ id: "csch_invoices", enabled: false });
    delete paused.threadTitle;
    server.patch.mockResolvedValue(paused);
    renderPage();

    await waitFor(() => row("Send me invoices more than 60 days past due"));
    await chooseFromMenu(row("Send me invoices more than 60 days past due"), "Pause");

    expect(
      await within(await screen.findByRole("region", { name: "Paused" })).findByText(
        "Send me invoices more than 60 days past due",
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(server.patch).toHaveBeenCalledWith("/assistant/schedules/csch_invoices/", {
        enabled: false,
      }),
    );
    // Once the answer, which carries no title, is written over the row, the
    // row still says where its answers go.
    await waitFor(() =>
      expect(row("Send me invoices more than 60 days past due")).not.toHaveAttribute("aria-busy"),
    );
    expect(
      within(row("Send me invoices more than 60 days past due")).getByRole("link", {
        name: "Open AR follow-ups",
      }),
    ).toBeInTheDocument();
  });

  it("puts a schedule back where it was when the server refuses to pause it", async () => {
    respondWith({ items: [dispatch, invoices], total: 2 });
    server.patch.mockRejectedValue(new Error("refused"));
    renderPage();

    await waitFor(() => row("Send me invoices more than 60 days past due"));
    await chooseFromMenu(row("Send me invoices more than 60 days past due"), "Pause");

    await waitFor(() => expect(screen.queryByRole("region", { name: "Paused" })).toBeNull());
    expect(
      within(group("Coming up")).getByText("Send me invoices more than 60 days past due"),
    ).toBeInTheDocument();
  });

  it("deletes only after asking again in the row", async () => {
    respondWith({ items: [dispatch, invoices], total: 2 });
    server.delete.mockResolvedValue(undefined);
    renderPage();

    await waitFor(() => row("Send me invoices more than 60 days past due"));
    const invoicesRow = () => within(row("Send me invoices more than 60 days past due"));
    await chooseFromMenu(row("Send me invoices more than 60 days past due"), "Delete");
    expect(invoicesRow().getByText("Delete this schedule?")).toBeInTheDocument();
    fireEvent.click(invoicesRow().getByRole("button", { name: "Cancel" }));
    expect(invoicesRow().queryByText("Delete this schedule?")).toBeNull();
    expect(server.delete).not.toHaveBeenCalled();

    await chooseFromMenu(row("Send me invoices more than 60 days past due"), "Delete");
    fireEvent.click(invoicesRow().getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(screen.queryByText("Send me invoices more than 60 days past due")).toBeNull(),
    );
    expect(server.delete).toHaveBeenCalledWith("/assistant/schedules/csch_invoices/");
    expect(screen.getByText("List shipments not yet dispatched")).toBeInTheDocument();
  });

  it("edits a schedule's request and day, and shows what was saved", async () => {
    respondWith({ items: [invoices], total: 1 });
    // An edit is answered with the schedule alone, without its conversation's title.
    const saved: Record<string, unknown> = listed({
      id: "csch_invoices",
      prompt: "Send me delivered shipments",
      cadence: "Every Friday · 8:00 AM",
      cronExpression: "0 8 * * 5",
      // Friday 9 October, 8:00 in Chicago.
      nextRunAt: 1791550800,
    });
    delete saved.threadTitle;
    server.patch.mockResolvedValue(saved);
    renderPage();

    await waitFor(() => row("Send me invoices more than 60 days past due"));
    await chooseFromMenu(row("Send me invoices more than 60 days past due"), "Edit");

    const dialog = within(await screen.findByRole("dialog"));
    expect(dialog.getByText("Times are on America/Chicago time.")).toBeVisible();
    expect(dialog.getByLabelText("What to ask")).toHaveValue(
      "Send me invoices more than 60 days past due",
    );
    expect(dialog.getByLabelText("Frequency")).toHaveTextContent("every week");
    expect(dialog.getByLabelText("Time of day")).toHaveTextContent("8:00 AM");
    expect(dialog.getByRole("button", { name: "Mon" })).toHaveAttribute("aria-pressed", "true");

    fireEvent.change(dialog.getByLabelText("What to ask"), {
      target: { value: "Send me delivered shipments" },
    });
    fireEvent.click(dialog.getByRole("button", { name: "Fri" }));
    expect(dialog.getByRole("button", { name: "Fri" })).toHaveAttribute("aria-pressed", "true");
    expect(dialog.getByRole("button", { name: "Mon" })).toHaveAttribute("aria-pressed", "false");
    fireEvent.click(dialog.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(server.patch).toHaveBeenCalledWith("/assistant/schedules/csch_invoices/", {
        prompt: "Send me delivered shipments",
        repeat: "friday",
        time: "08:00",
      }),
    );
    const edited = within(await waitFor(() => row("Send me delivered shipments")));
    expect(edited.getByText("Every Friday · 8:00 AM · next Fri, Oct 9 · 8:00 AM")).toBeVisible();
    expect(edited.getByRole("link", { name: "Open AR follow-ups" })).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("runs a schedule now and offers the way to its answer", async () => {
    respondWith({ items: [invoices], total: 1 });
    server.post.mockResolvedValue({
      turnId: "atrn_2",
      threadId: "athr_ar",
      streamUrl: "/assistant/turns/atrn_2/stream/",
      status: "Running",
    });
    renderPage();

    await waitFor(() => row("Send me invoices more than 60 days past due"));
    fireEvent.click(screen.getByRole("button", { name: "Run now" }));

    await waitFor(() => expect(toasts.success).toHaveBeenCalledTimes(1));
    expect(server.post).toHaveBeenCalledWith("/assistant/schedules/csch_invoices/run/", {});
    const [title, options] = toasts.success.mock.calls[0];
    expect(title).toBe("Running now");
    expect(options.description).toBe("The answer will post in AR follow-ups");
    expect(options.action.label).toBe("Open");
    act(() => options.action.onClick());
    expect(screen.getByTestId("location")).toHaveTextContent("/desk/t/athr_ar");
  });

  it("offers no run now to someone who may not ask the agent, but pause and delete", async () => {
    access.canRun = false;
    respondWith({ items: [invoices], total: 1 });
    renderPage();

    const invoicesRow = within(
      await waitFor(() => row("Send me invoices more than 60 days past due")),
    );
    expect(invoicesRow.queryByRole("button", { name: "Run now" })).toBeNull();
    fireEvent.click(invoicesRow.getByRole("button", { name: "Schedule actions" }));
    expect(await screen.findByRole("menuitem", { name: "Edit" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Pause" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Delete" })).toBeInTheDocument();
  });

  it("reads the next page as the reader nears the end of the list", async () => {
    const first = Array.from({ length: 2 }, (_, index) =>
      listed({ id: `csch_${index}`, prompt: `Request ${index}`, nextRunAt: 1791205200 + index }),
    );
    respondWith({ items: first, total: 3 }, { items: [permits], total: 3 });
    renderPage();

    await screen.findByText("Request 1");
    expect(screen.queryByText("Permits expiring this month")).toBeNull();
    const sentinel = document.querySelector(".dk-sd-more") as Element;
    act(() => FakeIntersectionObserver.show(sentinel));

    expect(await screen.findByText("Permits expiring this month")).toBeInTheDocument();
    expect(server.get).toHaveBeenLastCalledWith(
      `/assistant/schedules/?limit=${SCHEDULE_PAGE_SIZE}&offset=2`,
      expect.anything(),
    );
  });
});

describe("nextScheduleOffset", () => {
  const page = (count: number, total: number) => ({
    items: Array.from({ length: count }, () => ({}) as never),
    total,
  });

  it("starts the next page after everything read so far", () => {
    expect(nextScheduleOffset(page(50, 120), [page(50, 120)])).toBe(50);
    expect(nextScheduleOffset(page(50, 120), [page(50, 120), page(50, 120)])).toBe(100);
  });

  it("ends once everything is read, or on an empty page", () => {
    expect(nextScheduleOffset(page(20, 120), [page(50, 120), page(50, 120), page(20, 120)])).toBe(
      undefined,
    );
    expect(nextScheduleOffset(page(0, 5), [page(4, 5), page(0, 5)])).toBe(undefined);
  });
});
