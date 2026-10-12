import type { ConversationSchedule } from "@/types/assistant";
import { fireEvent, render, screen } from "@testing-library/react";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { registerCatalogSource, setLocale, translate } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { DeskScheduleCard } from "../conversation/desk-schedule-card";
import {
  cadenceLabel,
  nextRunLabel,
  patchScheduleList,
  scheduleLine,
} from "../conversation/desk-schedules";

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

describe("DeskScheduleCard", () => {
  it("shows the schedule under its heading, with pause and delete", () => {
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

    fireEvent.click(screen.getByRole("button", { name: "Pause" }));
    expect(onToggle).toHaveBeenCalledWith(expect.objectContaining({ id: "csch_1" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(onDelete).toHaveBeenCalledWith(expect.objectContaining({ id: "csch_1" }));
  });

  it("offers to resume a paused schedule and dims it", () => {
    const { container } = render(
      <DeskScheduleCard
        schedule={schedule({ enabled: false })}
        now={saturdayNoon}
        timezone={chicago}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Resume" })).toBeTruthy();
    expect(container.querySelector(".dk-sch-r.dk-off")).not.toBeNull();
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
