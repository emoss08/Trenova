import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  MAX_SCHEDULE_PROMPT,
  scheduleEditSchema,
  scheduleEditValues,
} from "../conversation/schedule-edit-form";

const t: TranslateFn = (message) => message ?? "";

/*
The editor opens on what a schedule asks, how often and when, read from the
cron the server keeps. Every cron a message can make is one of its choices.
*/
describe("scheduleEditValues", () => {
  it("reads each cadence a schedule can keep", () => {
    const at = (cronExpression: string) =>
      scheduleEditValues({ prompt: "Delivered shipments", cronExpression });

    expect(at("15 10 * * 5")).toEqual({
      prompt: "Delivered shipments",
      repeat: "friday",
      time: "10:15",
    });
    expect(at("30 7 * * 1-5")).toMatchObject({ repeat: "weekday", time: "07:30" });
    expect(at("0 0 * * *")).toMatchObject({ repeat: "day", time: "00:00" });
    expect(at("5 17 * * 0")).toMatchObject({ repeat: "sunday", time: "17:05" });
    expect(at("0 9 * * 7")).toMatchObject({ repeat: "sunday", time: "09:00" });
    expect(at("0 8 * * 6")).toMatchObject({ repeat: "saturday", time: "08:00" });
  });

  it("opens a cron it does not offer on every day, keeping the time it names", () => {
    expect(scheduleEditValues({ prompt: "x", cronExpression: "45 6 * * 1,3" })).toMatchObject({
      repeat: "day",
      time: "06:45",
    });
    expect(scheduleEditValues({ prompt: "x", cronExpression: "*/15 * * * *" })).toMatchObject({
      repeat: "day",
      time: "08:00",
    });
    expect(scheduleEditValues({ prompt: "x", cronExpression: "" })).toMatchObject({
      repeat: "day",
      time: "08:00",
    });
  });
});

describe("scheduleEditSchema", () => {
  const schema = scheduleEditSchema(t);

  it("takes a request, a cadence it offers and a time of day", () => {
    expect(
      schema.parse({ prompt: "  Delivered shipments ", repeat: "friday", time: "10:15" }),
    ).toEqual({ prompt: "Delivered shipments", repeat: "friday", time: "10:15" });
  });

  it("refuses each field it cannot send, on that field", () => {
    const fieldsOf = (values: Record<string, string>) => {
      const result = schema.safeParse(values);
      return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
    };

    expect(fieldsOf({ prompt: "   ", repeat: "day", time: "08:00" })).toEqual(["prompt"]);
    expect(
      fieldsOf({ prompt: "a".repeat(MAX_SCHEDULE_PROMPT + 1), repeat: "day", time: "08:00" }),
    ).toEqual(["prompt"]);
    expect(fieldsOf({ prompt: "x", repeat: "monthly", time: "08:00" })).toEqual(["repeat"]);
    expect(fieldsOf({ prompt: "x", repeat: "day", time: "24:00" })).toEqual(["time"]);
    expect(fieldsOf({ prompt: "x", repeat: "day", time: "" })).toEqual(["time"]);
  });
});
