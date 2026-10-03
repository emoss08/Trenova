import type { ConversationSchedule } from "@/types/assistant";
import { fireEvent, render, screen } from "@testing-library/react";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it, vi } from "vitest";
import { DeskScheduleCard } from "../conversation/desk-schedule-card";
import { nextRunLabel, patchScheduleList, scheduleLine } from "../conversation/desk-schedules";

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
