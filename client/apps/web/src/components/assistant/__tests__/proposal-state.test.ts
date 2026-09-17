import type { AssistantMessage, AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  argumentRows,
  classifyProposal,
  groupProposalsByMessage,
  humanizeToolName,
  isDecidable,
} from "../proposal-state";

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "ap_1",
    runId: "ar_1",
    toolName: "reassign_move",
    arguments: { moveId: "mv_1" },
    rationale: "The assigned driver is out of hours",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    ...overrides,
  };
}

function message(overrides: Partial<AssistantMessage> = {}): AssistantMessage {
  return {
    id: "amsg_1",
    threadId: "thr_1",
    sequence: 1,
    role: "Assistant",
    content: "",
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 0,
    ...overrides,
  };
}

describe("classifyProposal", () => {
  it("offers a decision only while nobody has made one", () => {
    expect(classifyProposal(proposal({ status: "Pending" }))).toBe("awaiting");
    expect(isDecidable(proposal({ status: "Pending" }))).toBe(true);
  });

  it("does not claim an approved proposal ran before it has", () => {
    expect(classifyProposal(proposal({ status: "Accepted" }))).toBe("running");
    expect(classifyProposal(proposal({ status: "Modified" }))).toBe("running");
    expect(isDecidable(proposal({ status: "Accepted" }))).toBe(false);
  });

  it("reports a completed proposal from either the status or the timestamp", () => {
    expect(classifyProposal(proposal({ status: "Executed" }))).toBe("done");
    expect(classifyProposal(proposal({ status: "Accepted", executedAt: 1700000000 }))).toBe("done");
  });

  // An approval that failed must never read as an approval that worked: that is
  // the difference between "your instruction was carried out" and "it was not".
  it("shows a failed execution as failed even while the status still says accepted", () => {
    expect(
      classifyProposal(
        proposal({ status: "Accepted", executionError: "shipment is already delivered" }),
      ),
    ).toBe("failed");
    expect(classifyProposal(proposal({ status: "ExecutionFailed" }))).toBe("failed");
  });

  // A retried approval that succeeds clears the old error, so a stale timestamp
  // must not resurrect a failure that no longer applies.
  it("prefers a live error over a timestamp from an earlier attempt", () => {
    expect(
      classifyProposal(
        proposal({ status: "Accepted", executedAt: 1700000000, executionError: "tool refused" }),
      ),
    ).toBe("failed");
  });

  it("separates a rejection from a proposal that simply lapsed", () => {
    expect(classifyProposal(proposal({ status: "Rejected" }))).toBe("declined");
    expect(classifyProposal(proposal({ status: "Expired" }))).toBe("closed");
    expect(classifyProposal(proposal({ status: "Superseded" }))).toBe("closed");
    expect(isDecidable(proposal({ status: "Expired" }))).toBe(false);
  });
});

describe("groupProposalsByMessage", () => {
  it("files each proposal under the turn that asked for it", () => {
    const first = proposal({ id: "ap_1", sourceMessageId: "amsg_1" });
    const second = proposal({ id: "ap_2", sourceMessageId: "amsg_2" });
    const third = proposal({ id: "ap_3", sourceMessageId: "amsg_1" });

    const { byMessage, orphans } = groupProposalsByMessage(
      [first, second, third],
      [message({ id: "amsg_1" }), message({ id: "amsg_2" })],
    );

    expect(byMessage.get("amsg_1")).toEqual([first, third]);
    expect(byMessage.get("amsg_2")).toEqual([second]);
    expect(orphans).toEqual([]);
  });

  // A pending write nobody can see is worse than one shown out of position.
  it("keeps a proposal whose message is not in the thread", () => {
    const unmatched = proposal({ id: "ap_9", sourceMessageId: "amsg_missing" });

    const { byMessage, orphans } = groupProposalsByMessage(
      [unmatched],
      [message({ id: "amsg_1" })],
    );

    expect(byMessage.size).toBe(0);
    expect(orphans).toEqual([unmatched]);
  });

  it("keeps a proposal that was never tied to a message", () => {
    const untied = proposal({ id: "ap_9", sourceMessageId: "" });

    const { orphans } = groupProposalsByMessage([untied], [message({ id: "amsg_1" })]);

    expect(orphans).toEqual([untied]);
  });
});

describe("humanizeToolName", () => {
  it("reads a tool identifier as words", () => {
    expect(humanizeToolName("reassign_move")).toBe("Reassign move");
    expect(humanizeToolName("hold-shipment")).toBe("Hold shipment");
  });

  it("returns the name unchanged when there is nothing to split", () => {
    expect(humanizeToolName("")).toBe("");
    expect(humanizeToolName("__")).toBe("__");
  });
});

describe("argumentRows", () => {
  it("shows every argument in a stable order", () => {
    expect(argumentRows({ moveId: "mv_1", assignedTo: "wrk_2" })).toEqual([
      { key: "assignedTo", value: "wrk_2" },
      { key: "moveId", value: "mv_1" },
    ]);
  });

  // The approver has to see what would actually be sent, so a nested value is
  // rendered as JSON rather than collapsing to "[object Object]".
  it("renders nested and non-string values readably", () => {
    expect(argumentRows({ stops: [{ id: "stp_1" }], force: true, attempts: 2 })).toEqual([
      { key: "attempts", value: "2" },
      { key: "force", value: "true" },
      { key: "stops", value: '[{"id":"stp_1"}]' },
    ]);
  });

  it("marks empty and absent values rather than showing nothing", () => {
    expect(argumentRows({ note: "", reason: null })).toEqual([
      { key: "note", value: "—" },
      { key: "reason", value: "—" },
    ]);
  });

  it("has no rows when a tool takes no arguments", () => {
    expect(argumentRows({})).toEqual([]);
    expect(argumentRows(null)).toEqual([]);
    expect(argumentRows(undefined)).toEqual([]);
  });
});
