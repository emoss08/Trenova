import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { formatUnixDate } from "./date";
import {
  describeTrainingRecordOn,
  describeTrainingTiming,
  sortTrainingWorstFirst,
  trainingHealthMeta,
  trainingProgress,
  trainingRecordStatusLabel,
} from "./training";

describe("trainingHealthMeta", () => {
  it("ranks blocking states ahead of scheduled and current ones", () => {
    const order = [
      "Missing",
      "Failed",
      "Expired",
      "Overdue",
      "DueSoon",
      "ExpiringSoon",
      "Scheduled",
      "Current",
    ] as const;
    const ranks = order.map((health) => trainingHealthMeta(health).rank);
    expect([...ranks].sort((a, b) => a - b)).toEqual(ranks);
    expect(trainingHealthMeta("Overdue").blocks).toBe(true);
    expect(trainingHealthMeta("DueSoon").blocks).toBe(false);
    expect(trainingHealthMeta("Current").badgeVariant).toBe("success");
  });
});

describe("describeTrainingTiming", () => {
  it("talks about due dates for open work and expiry for completions", () => {
    expect(describeTrainingTiming({ health: "DueSoon", daysUntilDue: 3 })).toBe("Due in 3 days");
    expect(describeTrainingTiming({ health: "DueSoon", daysUntilDue: 0 })).toBe("Due today");
    expect(describeTrainingTiming({ health: "Overdue", daysUntilDue: -4 })).toBe(
      "Overdue by 4 days",
    );
    expect(describeTrainingTiming({ health: "Scheduled", daysUntilDue: null })).toBe("No due date");
    expect(describeTrainingTiming({ health: "ExpiringSoon", daysUntilExpiry: 12 })).toBe(
      "Expires in 12 days",
    );
    expect(describeTrainingTiming({ health: "Expired", daysUntilExpiry: -1 })).toBe(
      "Expired yesterday",
    );
    expect(describeTrainingTiming({ health: "Current", daysUntilExpiry: null })).toBe(
      "Does not expire",
    );
    expect(describeTrainingTiming({ health: "Missing" })).toBe("Not assigned");
    expect(describeTrainingTiming({ health: "Failed" })).toBe("Failed — retake needed");
  });
});

describe("sortTrainingWorstFirst", () => {
  it("puts blocking required items first, then by name", () => {
    const items = [
      { name: "Winter", health: "Current" as const, required: false },
      { name: "Hazmat", health: "Overdue" as const, required: true },
      { name: "Defensive", health: "DueSoon" as const, required: true },
      { name: "Forklift", health: "Overdue" as const, required: false },
    ];
    expect(sortTrainingWorstFirst(items).map((item) => item.name)).toEqual([
      "Hazmat",
      "Forklift",
      "Defensive",
      "Winter",
    ]);
  });
});

describe("trainingProgress", () => {
  it("counts satisfied required courses against the total", () => {
    expect(
      trainingProgress([
        { required: true, health: "Current" },
        { required: true, health: "ExpiringSoon" },
        { required: true, health: "Overdue" },
        { required: false, health: "Missing" },
      ]),
    ).toEqual({ satisfied: 2, required: 3, ratio: 2 / 3 });
    expect(trainingProgress([])).toEqual({ satisfied: 0, required: 0, ratio: 1 });
  });
});

describe("training record status", () => {
  // 2026-03-04 12:00 UTC: midday, so the calendar day is the same in every test time zone.
  const completedAt = 1772625600;

  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "Completed {0}": "Completado el {0}",
        "In progress since {0}": "En curso desde el {0}",
        "In progress": "En curso",
      }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("lets a translation place the date where its grammar needs it", () => {
    const date = formatUnixDate(completedAt);
    expect(describeTrainingRecordOn("Completed", completedAt)).toBe(`Completado el ${date}`);
    expect(describeTrainingRecordOn("InProgress", completedAt)).toBe(`En curso desde el ${date}`);
  });

  it("names a status without a date in the reader's language", () => {
    expect(trainingRecordStatusLabel("InProgress")).toBe("En curso");
    expect(trainingRecordStatusLabel("Waived")).toBe("Waived");
  });
});
