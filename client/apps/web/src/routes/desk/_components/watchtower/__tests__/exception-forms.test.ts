import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  EMPTY_EXCEPTION_ACTION,
  MAX_DUE_AHEAD_SECONDS,
  MAX_SNOOZE_SECONDS,
  exceptionActionSchema,
} from "../exception-forms";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);
const NOW = 1_800_000_000;
const CODES = [
  { code: "Fixed" as const, label: "Fixed", requiresNote: false },
  { code: "Other" as const, label: "Other", requiresNote: true },
];

const issuesOf = (result: { success: boolean; error?: { issues: { path: PropertyKey[] }[] } }) =>
  result.success ? [] : (result.error?.issues.map((issue) => issue.path.join(".")) ?? []);

describe("exceptionActionSchema", () => {
  it("needs a future snooze time within thirty days", () => {
    const schema = exceptionActionSchema(t, "snooze", CODES, NOW);
    expect(issuesOf(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION }))).toEqual(["until"]);
    expect(issuesOf(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, until: NOW }))).toEqual([
      "until",
    ]);
    expect(
      issuesOf(
        schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, until: NOW + MAX_SNOOZE_SECONDS + 1 }),
      ),
    ).toEqual(["until"]);
    expect(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, until: NOW + 3600 }).success).toBe(true);
  });

  it("allows clearing the due time but not pushing it past ninety days", () => {
    const schema = exceptionActionSchema(t, "due", CODES, NOW);
    expect(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, dueAt: null }).success).toBe(true);
    expect(
      issuesOf(
        schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, dueAt: NOW + MAX_DUE_AHEAD_SECONDS + 1 }),
      ),
    ).toEqual(["dueAt"]);
  });

  it("asks for a listed code and a note only where the code needs one", () => {
    const schema = exceptionActionSchema(t, "resolve", CODES, NOW);
    expect(issuesOf(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, code: "Cleared" }))).toEqual([
      "code",
    ]);
    expect(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, code: "Fixed" }).success).toBe(true);
    expect(
      issuesOf(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, code: "Other", note: "  " })),
    ).toEqual(["note"]);
    expect(
      schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, code: "Other", note: "Phoned in" }).success,
    ).toBe(true);
  });

  it("caps a note at two thousand characters", () => {
    const schema = exceptionActionSchema(t, "reopen", CODES, NOW);
    expect(
      issuesOf(schema.safeParse({ ...EMPTY_EXCEPTION_ACTION, note: "x".repeat(2001) })),
    ).toEqual(["note"]);
  });
});
