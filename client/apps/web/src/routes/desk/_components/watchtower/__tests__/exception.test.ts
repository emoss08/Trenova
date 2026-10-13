import type { WatchtowerItemEvent } from "@/lib/graphql/watchtower";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  EXCEPTION_VIEWS,
  dueLabel,
  eventSentence,
  exceptionDueState,
  exceptionStatusAttrs,
  exceptionViewFilter,
} from "../exception";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const NOW = 1_800_000_000;

const event = (overrides: Partial<WatchtowerItemEvent>): WatchtowerItemEvent => ({
  id: "wte_1",
  itemId: "wt_1",
  kind: "Created",
  actorId: null,
  actor: null,
  fromValue: "",
  toValue: "",
  note: "",
  createdAt: NOW,
  ...overrides,
});

describe("exceptionDueState", () => {
  it("is overdue once the due time has passed", () => {
    expect(
      exceptionDueState(
        { status: "Open", dueAt: NOW - 600, snoozedUntil: null, resolvedAt: null },
        NOW,
      ),
    ).toEqual({ kind: "overdue", by: 600 });
  });

  it("counts down to a due time still ahead", () => {
    expect(
      exceptionDueState(
        { status: "InProgress", dueAt: NOW + 7200, snoozedUntil: null, resolvedAt: null },
        NOW,
      ),
    ).toEqual({ kind: "due", in: 7200 });
  });

  it("never calls a snoozed exception overdue", () => {
    expect(
      exceptionDueState(
        { status: "Snoozed", dueAt: NOW - 600, snoozedUntil: NOW + 3600, resolvedAt: null },
        NOW,
      ),
    ).toEqual({ kind: "snoozed", until: NOW + 3600 });
  });

  it("never calls a resolved exception overdue", () => {
    expect(
      exceptionDueState(
        { status: "Resolved", dueAt: NOW - 600, snoozedUntil: null, resolvedAt: NOW - 60 },
        NOW,
      ),
    ).toEqual({ kind: "none" });
  });

  it("has nothing to say without a due time", () => {
    expect(
      exceptionDueState({ status: "Open", dueAt: null, snoozedUntil: null, resolvedAt: null }, NOW),
    ).toEqual({ kind: "none" });
  });
});

describe("dueLabel", () => {
  it("names how late and how soon", () => {
    expect(dueLabel(t, { kind: "overdue", by: 2 * 3600 })).toBe("Overdue 2h");
    expect(dueLabel(t, { kind: "due", in: 45 * 60 })).toBe("Due in 45m");
    expect(dueLabel(t, { kind: "none" })).toBeNull();
  });
});

describe("exceptionViewFilter", () => {
  it("reads every work list soonest due first and the whole feed newest first", () => {
    expect(exceptionViewFilter("all")).toEqual({});
    expect(exceptionViewFilter("mine")).toEqual({ assignedToMe: true, orderByDue: true });
    expect(exceptionViewFilter("unassigned")).toEqual({ unassigned: true, orderByDue: true });
    expect(exceptionViewFilter("overdue")).toEqual({ overdueOnly: true, orderByDue: true });
    expect(EXCEPTION_VIEWS).toEqual(["all", "mine", "unassigned", "overdue"]);
  });
});

describe("exceptionStatusAttrs", () => {
  it("gives every status a lifecycle phase", () => {
    const attrs = exceptionStatusAttrs(t);
    expect(attrs.Open.phase).toBe("attention");
    expect(attrs.Acknowledged.phase).toBe("awaiting");
    expect(attrs.InProgress.phase).toBe("active");
    expect(attrs.Snoozed.phase).toBe("queued");
    expect(attrs.Resolved.phase).toBe("complete");
  });
});

describe("eventSentence", () => {
  const labels = new Map([["CarrierDelay", "Carrier delay"]]);

  it("shows a resolution by its label", () => {
    expect(eventSentence(t, event({ kind: "Resolved", toValue: "CarrierDelay" }), labels)).toBe(
      "Resolved as Carrier delay",
    );
  });

  it("falls back to the stored code when no label is known", () => {
    expect(eventSentence(t, event({ kind: "Resolved", toValue: "Duplicate" }), labels)).toBe(
      "Resolved as Duplicate",
    );
  });

  it("says a cleared due time in words rather than as an empty instant", () => {
    expect(eventSentence(t, event({ kind: "DueChanged", toValue: "" }), labels)).toBe(
      "Cleared the due time",
    );
  });

  it("names both severities of an escalation", () => {
    expect(
      eventSentence(
        t,
        event({ kind: "Escalated", fromValue: "Warning", toValue: "Critical" }),
        labels,
      ),
    ).toBe("Escalated from Warning to Critical");
  });
});
