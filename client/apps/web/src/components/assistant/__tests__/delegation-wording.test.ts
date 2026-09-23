import { translate } from "@trenova/shared/i18n/runtime";
import { delegateWriteSchema, type DelegateWrite } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  currentActivity,
  describeActivity,
  groupActivity,
  isActionEffect,
  toolEffect,
  type ToolStep,
} from "../activity";
import {
  awaitingLine,
  delegateView,
  handOffStatus,
  madeLine,
  madeRecordPath,
  splitOnSubject,
} from "../delegation";
import { proposedByOther } from "../proposal-state";
import { emptyDelegateProgress } from "../turn-stream";

const t = translate;

/** A write as delegate_finished carries it (ports/services DelegateWrite). */
function write(fields: Record<string, unknown>): DelegateWrite {
  return delegateWriteSchema.parse({ toolName: "create_report", callId: "c1", ...fields });
}

function liveStep(overrides: Partial<ToolStep> = {}): ToolStep {
  return {
    id: "call_delegate_1",
    name: "delegate_task",
    arguments: { agentId: "agdef_rb", task: "Make a report" },
    status: "running",
    content: "",
    effect: "delegate",
    summary: "",
    durationSeconds: null,
    delegate: {
      kind: "live",
      progress: { ...emptyDelegateProgress("agdef_rb"), agentName: "Report Builder" },
    },
    ...overrides,
  };
}

describe("hand-off effect", () => {
  it("reads delegate_task as a hand-off when the server does not say", () => {
    expect(toolEffect("delegate_task")).toBe("delegate");
    expect(toolEffect("delegate_task", "delegate")).toBe("delegate");
  });

  it("stands on its own line in ink, like every other action", () => {
    expect(isActionEffect("delegate")).toBe(true);
    expect(groupActivity([liveStep({ id: "a" }), liveStep({ id: "b" })])).toHaveLength(2);
  });

  it("says who is being asked while the task runs, and who was asked after", () => {
    const [running] = groupActivity([liveStep()]);
    expect(describeActivity(running, t)).toMatchObject({
      phrase: "Asking Report Builder…",
      state: "running",
    });
    expect(currentActivity([liveStep()], t)?.phrase).toBe("Asking Report Builder…");

    const [done] = groupActivity([liveStep({ status: "done", summary: "Report Builder" })]);
    expect(describeActivity(done, t)).toMatchObject({ phrase: "Asked Report Builder" });
  });

  it("names the agent from the call's summary when nothing else does", () => {
    const [group] = groupActivity([
      liveStep({ delegate: undefined, status: "done", summary: "Dispatch desk" }),
    ]);

    expect(describeActivity(group, t).phrase).toBe("Asked Dispatch desk");
  });

  // The account arrives before the call settles; from then the hand-off is
  // over, whatever the call's own state says.
  it("counts the hand-off over once its account has arrived", () => {
    const step = liveStep({
      delegate: {
        kind: "live",
        progress: {
          ...emptyDelegateProgress("agdef_rb"),
          agentName: "Report Builder",
          report: {
            delegateCallId: "call_delegate_1",
            agentId: "agdef_rb",
            agentName: "Report Builder",
            icon: "",
            accent: "",
            status: "exhausted",
            reply: "Here is what I have.",
            reason: "Report Builder used every tool call its settings allow.",
            made: [],
            awaiting: [],
            published: [],
            toolCallsUsed: 12,
            moreMade: 0,
            moreAwaiting: 0,
            morePublished: 0,
          },
        },
      },
    });

    const view = delegateView(step);
    expect(view?.outcome).toBe("exhausted");
    expect(view?.reply).toBe("Here is what I have.");
    expect(handOffStatus(view!, t)?.tone).toBe("warning");
    expect(describeActivity(groupActivity([step])[0], t).state).toBe("done");
  });
});

/**
 * What the other agent made is said in our verb, from the result's action
 * and kind; the name is the record's own and links to it where the registry
 * has a page for its kind.
 */
describe("madeLine", () => {
  it("words a created report and links its name to the report", () => {
    const line = madeLine(
      write({
        summary: "ignored when the result names it",
        result: {
          action: "created",
          kind: "report",
          name: "Shipments for Peak Distributing",
          ids: { definitionId: "rd_1" },
        },
      }),
      t,
    );

    expect(line).toMatchObject({
      text: "Created report Shipments for Peak Distributing",
      subject: "Shipments for Peak Distributing",
      path: "/reports/explore/rd_1",
      state: "made",
    });
    expect(splitOnSubject(line)).toEqual({
      before: "Created report ",
      subject: "Shipments for Peak Distributing",
      after: "",
    });
  });

  it("names the kind from the record a result names, in the registry's words", () => {
    const line = madeLine(
      write({
        result: {
          action: "created",
          kind: "",
          name: "Lanes",
          ids: {},
          record: { entityType: "rate_matrix", id: "rm_1" },
        },
      }),
      t,
    );

    expect(line.text).toBe("Created rate matrix Lanes");
    expect(line.path).toContain("panelEntityId=rm_1");
  });

  it("reads a result from a server that sends no record the old way", () => {
    const parsed = write({
      result: { action: "created", kind: "report", name: "Ops", ids: { definitionId: "rd_1" } },
    });

    expect(parsed.result?.record ?? null).toBeNull();
    expect(madeLine(parsed, t).path).toBe("/reports/explore/rd_1");
  });

  it("words a write whose tool says nothing from the tool, unlinked", () => {
    const line = madeLine(write({ toolName: "update_customer", summary: "Peak Distributing" }), t);

    expect(line).toMatchObject({ text: "Updated Peak Distributing", path: null, state: "made" });
    expect(splitOnSubject(line)).toBeNull();
  });

  it("never words a failed or previewed write as made", () => {
    expect(
      madeLine(write({ summary: "On-time", error: "The name is already used." }), t),
    ).toMatchObject({
      text: "On-time didn't go through",
      state: "failed",
      error: "The name is already used.",
    });
    expect(madeLine(write({ summary: "On-time", simulated: true }), t)).toMatchObject({
      text: "Previewed a change to On-time",
      state: "simulated",
    });
  });

  it("names what waits on the person by its tool and subject", () => {
    expect(awaitingLine(write({ toolName: "share_report", summary: "On-time" }), t)).toMatchObject({
      state: "awaiting",
      path: null,
    });
    expect(awaitingLine(write({ toolName: "share_report", summary: "On-time" }), t).text).toMatch(
      /On-time$/u,
    );
  });
});

describe("madeRecordPath", () => {
  it("links the record a result names outright, whatever its kind and ids say", () => {
    expect(
      madeRecordPath({
        action: "created",
        kind: "saved query",
        name: "Ops",
        ids: { definitionId: "rd_9", folderId: "f_1" },
        record: { entityType: "report", id: "rd_9" },
      }),
    ).toBe("/reports/explore/rd_9");
  });

  it("falls back to the kind and ids when the named record is not one the app opens", () => {
    expect(
      madeRecordPath({
        action: "created",
        kind: "report",
        name: "Ops",
        ids: { definitionId: "rd_1" },
        record: { entityType: "not_a_page", id: "x_1" },
      }),
    ).toBe("/reports/explore/rd_1");
    expect(
      madeRecordPath({
        action: "created",
        kind: "report",
        name: "Ops",
        ids: { definitionId: "rd_1" },
        record: { entityType: "report", id: "  " },
      }),
    ).toBe("/reports/explore/rd_1");
  });

  it("takes the id named for the record's own kind first", () => {
    expect(
      madeRecordPath({
        action: "created",
        kind: "dashboard",
        name: "Ops",
        ids: { tileId: "dt_1", dashboardId: "db_1" },
      }),
    ).toBe("/reports/dashboards/db_1");
  });

  it("reads a kind in words as the registry's entity", () => {
    expect(
      madeRecordPath({
        action: "updated",
        kind: "Rate matrix",
        name: "Lanes",
        ids: { id: "rm_1" },
      }),
    ).toContain("panelEntityId=rm_1");
  });

  it("links nothing for a kind with no page or ids that name nothing certain", () => {
    expect(madeRecordPath({ action: "created", kind: "memory", name: "", ids: { id: "m1" } })).toBe(
      null,
    );
    expect(
      madeRecordPath({ action: "created", kind: "report", name: "", ids: { a: "1", b: "2" } }),
    ).toBeNull();
    expect(madeRecordPath({ action: "created", kind: "report", name: "", ids: {} })).toBeNull();
    expect(madeRecordPath(null)).toBeNull();
  });
});

/** The conversation's own agent heads the reply; only another agent's cards say who proposed them. */
describe("proposedByOther", () => {
  it("attributes a proposal only to an agent other than the conversation's", () => {
    expect(proposedByOther("agdef_widgets", "agdef_rb")).toBe(true);
    expect(proposedByOther("agdef_widgets", "agdef_widgets")).toBe(false);
  });

  it("says nothing when either side is unknown", () => {
    expect(proposedByOther(null, "agdef_rb")).toBe(false);
    expect(proposedByOther("agdef_widgets", undefined)).toBe(false);
    expect(proposedByOther("agdef_widgets", "")).toBe(false);
  });
});
