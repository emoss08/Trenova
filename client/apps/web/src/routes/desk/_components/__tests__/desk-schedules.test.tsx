import { assistant } from "@/lib/queries/assistant";
import type { ConversationSchedule } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import type { ReactNode } from "react";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { registerCatalogSource, setLocale, translate } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { DeskScheduleCard } from "../conversation/desk-schedule-card";
import { cadenceLabel, nextRunLabel, scheduleLine } from "@/lib/schedule-labels";
import {
  mergeSchedule,
  orderSchedules,
  patchScheduleList,
  patchSchedulePages,
  type SchedulePages,
  useCreateSchedule,
} from "../conversation/desk-schedules";

const server = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, post: server.post } };
});

const t: TranslateFn = (message, ...args) =>
  (message ?? "").replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]));

// Saturday 3 October 2026, noon in Chicago.
const saturdayNoon = 1791046800;
const chicago = "America/Chicago";

function schedule(overrides: Partial<ConversationSchedule> = {}): ConversationSchedule {
  return {
    id: "csch_1",
    threadId: "athr_1",
    userId: "usr_1",
    prompt: "What's blocking the billing queue?",
    cadence: "Every weekday · 7:30 AM",
    cronExpression: "30 7 * * 1-5",
    timezone: chicago,
    enabled: true,
    lastRunAt: null,
    // Monday 5 October, 7:30 in Chicago.
    nextRunAt: 1791203400,
    lastTurnId: null,
    createdAt: saturdayNoon,
    threadTitle: "",
    ...overrides,
  };
}

describe("nextRunLabel", () => {
  it("names today and tomorrow, and dates anything later", () => {
    expect(nextRunLabel(saturdayNoon + 3600, saturdayNoon, chicago, t)).toBe("Today · 1:00 PM");
    expect(nextRunLabel(saturdayNoon + 24 * 3600, saturdayNoon, chicago, t)).toBe(
      "Tomorrow · 12:00 PM",
    );
    expect(nextRunLabel(1791203400, saturdayNoon, chicago, t)).toBe("Mon, Oct 5 · 7:30 AM");
  });

  it("reads the day on the reader's clock", () => {
    // 01:30 UTC on Sunday is still Saturday evening in Chicago.
    expect(nextRunLabel(1791077400, saturdayNoon, chicago, t)).toBe("Today · 8:30 PM");
  });
});

describe("scheduleLine", () => {
  it("says when it runs and next, or that it is paused", () => {
    expect(scheduleLine(schedule(), saturdayNoon, chicago, t)).toBe(
      "Every weekday · 7:30 AM · next Mon, Oct 5 · 7:30 AM",
    );
    expect(scheduleLine(schedule({ enabled: false }), saturdayNoon, chicago, t)).toBe(
      "Every weekday · 7:30 AM · paused",
    );
  });
});

describe("patchScheduleList", () => {
  const list = { items: [schedule()], total: 1 };

  it("replaces, adds and removes one schedule", () => {
    const paused = schedule({ enabled: false });
    expect(patchScheduleList(list, "csch_1", paused)?.items).toEqual([paused]);

    const added = schedule({ id: "csch_2" });
    expect(patchScheduleList(list, "csch_2", added)).toEqual({
      items: [added, list.items[0]],
      total: 2,
    });

    expect(patchScheduleList(list, "csch_1", null)).toEqual({ items: [], total: 0 });
    expect(patchScheduleList(undefined, "csch_1", null)).toBeUndefined();
  });
});

/*
A pause, resume or run is answered with the schedule but not its
conversation's title, which has not changed. Writing that answer over a row on
the Scheduled page must not blank where the row says its answers go.
*/
describe("mergeSchedule", () => {
  it("keeps the conversation title an answer leaves out", () => {
    const shown = schedule({ threadTitle: "AR follow-ups" });
    const answered = schedule({ enabled: false });

    expect(mergeSchedule(shown, answered)).toEqual({ ...answered, threadTitle: "AR follow-ups" });
  });

  it("takes a title the answer does carry", () => {
    const shown = schedule({ threadTitle: "AR follow-ups" });
    const answered = schedule({ threadTitle: "Receivables" });

    expect(mergeSchedule(shown, answered)).toBe(answered);
  });

  it("keeps the title through a change written into a conversation's cards", () => {
    const list = { items: [schedule({ threadTitle: "AR follow-ups" })], total: 1 };

    expect(patchScheduleList(list, "csch_1", schedule({ enabled: false }))?.items).toEqual([
      schedule({ enabled: false, threadTitle: "AR follow-ups" }),
    ]);
  });
});

describe("patchSchedulePages", () => {
  const first = schedule({ id: "csch_1", threadTitle: "AR follow-ups" });
  const second = schedule({ id: "csch_2" });
  const third = schedule({ id: "csch_3" });
  const pages: SchedulePages = {
    pages: [
      { items: [first, second], total: 3 },
      { items: [third], total: 3 },
    ],
    pageParams: [0, 2],
  };

  it("replaces a schedule on whichever page holds it, keeping its title", () => {
    const paused = schedule({ id: "csch_1", enabled: false });

    expect(patchSchedulePages(pages, "csch_1", paused)).toEqual({
      pages: [
        { items: [{ ...paused, threadTitle: "AR follow-ups" }, second], total: 3 },
        { items: [third], total: 3 },
      ],
      pageParams: [0, 2],
    });
  });

  it("removes a schedule and lowers every page's total", () => {
    expect(patchSchedulePages(pages, "csch_3", null)).toEqual({
      pages: [
        { items: [first, second], total: 2 },
        { items: [], total: 2 },
      ],
      pageParams: [0, 2],
    });
  });

  it("leaves pages alone for a schedule they do not hold, and nothing read as nothing", () => {
    expect(patchSchedulePages(pages, "csch_9", null)).toBe(pages);
    expect(patchSchedulePages(pages, "csch_9", schedule({ id: "csch_9" }))).toBe(pages);
    expect(patchSchedulePages(undefined, "csch_1", null)).toBeUndefined();
  });
});

/*
The Scheduled page lists what will fire soonest first and the paused ones
after, the order the server reads them in, so one paused or resumed on the
page moves to its group at once.
*/
describe("orderSchedules", () => {
  it("puts the ones that fire first, soonest first, then the paused ones", () => {
    const later = schedule({ id: "csch_later", nextRunAt: 2_000 });
    const soonest = schedule({ id: "csch_soon", nextRunAt: 1_000 });
    const unslotted = schedule({ id: "csch_none", nextRunAt: null });
    const pausedSoon = schedule({ id: "csch_ps", enabled: false, nextRunAt: 500 });
    const pausedLater = schedule({ id: "csch_pl", enabled: false, nextRunAt: 3_000 });

    expect(
      orderSchedules([pausedLater, unslotted, later, pausedSoon, soonest]).map((item) => item.id),
    ).toEqual(["csch_soon", "csch_later", "csch_none", "csch_ps", "csch_pl"]);
  });

  it("breaks a tie on the next run by the newest made, then the id", () => {
    const older = schedule({ id: "csch_a", nextRunAt: 1_000, createdAt: 10 });
    const newer = schedule({ id: "csch_b", nextRunAt: 1_000, createdAt: 20 });
    const sameTimeLowId = schedule({ id: "csch_c", nextRunAt: 1_000, createdAt: 20 });
    const sameTimeHighId = schedule({ id: "csch_d", nextRunAt: 1_000, createdAt: 20 });

    expect(
      orderSchedules([older, sameTimeLowId, newer, sameTimeHighId]).map((item) => item.id),
    ).toEqual(["csch_d", "csch_c", "csch_b", "csch_a"]);
  });

  it("leaves the list it was given as it was", () => {
    const given = [
      schedule({ id: "csch_2", nextRunAt: 2 }),
      schedule({ id: "csch_1", nextRunAt: 1 }),
    ];

    orderSchedules(given);
    expect(given.map((item) => item.id)).toEqual(["csch_2", "csch_1"]);
  });
});

/** Opens a schedule's menu and chooses one of its actions. */
async function chooseFromMenu(action: string, scope: HTMLElement = document.body) {
  fireEvent.click(within(scope).getByRole("button", { name: "Schedule actions" }));
  fireEvent.click(await screen.findByRole("menuitem", { name: action }));
}

describe("DeskScheduleCard", () => {
  it("shows the schedule under its heading, with pause and delete in its menu", async () => {
    const onToggle = vi.fn();
    const onDelete = vi.fn();
    render(
      <DeskScheduleCard
        schedule={schedule()}
        now={saturdayNoon}
        timezone={chicago}
        onToggle={onToggle}
        onDelete={onDelete}
      />,
    );

    expect(screen.getByText("Scheduled")).toBeTruthy();
    expect(screen.getByText("Results will post into this conversation")).toBeTruthy();
    expect(screen.getByText("What's blocking the billing queue?")).toBeTruthy();
    expect(screen.getByText("Every weekday · 7:30 AM · next Mon, Oct 5 · 7:30 AM")).toBeTruthy();

    expect(screen.queryByRole("button", { name: "Pause" })).toBeNull();
    await chooseFromMenu("Pause");
    expect(onToggle).toHaveBeenCalledWith(expect.objectContaining({ id: "csch_1" }));
    await chooseFromMenu("Delete");
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ id: "csch_1" }));
  });

  it("opens the editor on the schedule from its card", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <DeskScheduleCard
          schedule={schedule()}
          now={saturdayNoon}
          timezone={chicago}
          onToggle={vi.fn()}
          onDelete={vi.fn()}
        />
      </QueryClientProvider>,
    );

    await chooseFromMenu("Edit");

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Edit schedule")).toBeInTheDocument();
    expect(screen.getByLabelText("What to ask")).toHaveValue("What's blocking the billing queue?");
    expect(screen.getByLabelText("Frequency")).toHaveTextContent("every weekday");
    expect(screen.getByLabelText("Time of day")).toHaveTextContent("7:30 AM");
    expect(screen.queryByRole("group", { name: "Day of the week" })).toBeNull();
  });

  it("offers to resume a paused schedule and dims it", async () => {
    const { container } = render(
      <DeskScheduleCard
        schedule={schedule({ enabled: false })}
        now={saturdayNoon}
        timezone={chicago}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(container.querySelector(".dk-sch-r.dk-off")).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Schedule actions" }));
    expect(await screen.findByRole("menuitem", { name: "Resume" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Pause" })).toBeNull();
  });

  it("says the schedule was deleted once it is gone, and nothing while loading", () => {
    const { container, rerender } = render(
      <DeskScheduleCard
        schedule={undefined}
        now={saturdayNoon}
        timezone={chicago}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(container.textContent).toBe("");

    rerender(
      <DeskScheduleCard
        schedule={null}
        now={saturdayNoon}
        timezone={chicago}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText("Schedule deleted")).toBeTruthy();
    expect(container.querySelector(".dk-schc.dk-gone")).not.toBeNull();
  });
});

/*
The cadence is written from the schedule's cron when it is read, so it is in
the reader's language whatever language it was asked in. The label the server
stored is the fallback for a cron the card does not describe.
*/
describe("cadenceLabel", () => {
  it("names the days and the time from the cron", () => {
    expect(cadenceLabel(schedule(), t)).toBe("Every weekday · 7:30 AM");
    expect(cadenceLabel(schedule({ cronExpression: "0 8 * * *" }), t)).toBe("Every day · 8:00 AM");
    expect(cadenceLabel(schedule({ cronExpression: "0 0 * * 1" }), t)).toBe(
      "Every Monday · 12:00 AM",
    );
    expect(cadenceLabel(schedule({ cronExpression: "45 17 * * 0" }), t)).toBe(
      "Every Sunday · 5:45 PM",
    );
    expect(cadenceLabel(schedule({ cronExpression: "0 12 * * 6" }), t)).toBe(
      "Every Saturday · 12:00 PM",
    );
  });

  it("falls back to the stored label for a cron it does not describe", () => {
    expect(cadenceLabel(schedule({ cadence: "Stored", cronExpression: "0 8 1 * *" }), t)).toBe(
      "Stored",
    );
    expect(cadenceLabel(schedule({ cadence: "Stored", cronExpression: "" }), t)).toBe("Stored");
  });
});

describe("a schedule read in another language", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "Every weekday · {0}": "Cada día laborable · {0}",
        "Every Friday · {0}": "Todos los viernes · {0}",
        "{0} · next {1}": "{0} · próxima {1}",
      }),
      "zh-CN": async () => ({
        "Every weekday · {0}": "每个工作日 · {0}",
      }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("writes the cadence, its time and the next run in Spanish", () => {
    // ICU writes the Spanish meridiem as "p.m." or "p. m." depending on its version.
    expect(cadenceLabel(schedule({ cronExpression: "30 16 * * 5" }), translate)).toMatch(
      /^Todos los viernes · 4:30 p\.\s?m\.$/,
    );
    expect(scheduleLine(schedule(), saturdayNoon, chicago, translate)).toMatch(
      /^Cada día laborable · 7:30 a\.\s?m\. · próxima lun, 5 oct\.? · 7:30 a\.\s?m\.$/,
    );
  });

  it("writes the cadence in Chinese for a schedule asked in English", async () => {
    await setLocale("zh-CN");
    expect(cadenceLabel(schedule(), translate)).toBe("每个工作日 · 上午7:30");
    await setLocale("es");
  });
});

/*
A conversation opened with "/schedule every Friday at 10:15, …" is named for
its request when the schedule is saved. The name the Desk shows for the open
conversation, in its top bar and its rail, is read again once it is.
*/
describe("useCreateSchedule", () => {
  it("reads the conversation again so the name it was given shows", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const threadKey = assistant.thread("athr_1").queryKey;
    client.setQueryData(threadKey, { id: "athr_1", title: "" });
    server.post.mockResolvedValue({
      schedule: {
        ...schedule(),
        prompt: "Send me a list of delivered shipments",
        cronExpression: "15 10 * * 5",
      },
      message: {
        id: "amsg_1",
        threadId: "athr_1",
        sequence: 0,
        role: "User",
        kind: "Schedule",
        scheduleId: "csch_1",
        content: "/schedule every Friday at 10:15, send me a list of delivered shipments",
        createdAt: saturdayNoon,
      },
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useCreateSchedule("athr_1"), { wrapper });

    result.current.create("/schedule every Friday at 10:15, send me a list of delivered shipments");

    await waitFor(() => expect(client.getQueryState(threadKey)?.isInvalidated).toBe(true));
    expect(server.post).toHaveBeenCalledWith("/assistant/threads/athr_1/schedules/", {
      content: "/schedule every Friday at 10:15, send me a list of delivered shipments",
    });
  });
});
