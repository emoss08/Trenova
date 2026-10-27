// A Desk conversation in which the Dispatch agent proposed a four-step plan.
// The `deskPlan` scenario field picks where the plan stands: "awaiting" (the
// person has not decided it), "approving" (approved, inside its undo window),
// "running" (two steps ran, the third is running), "failed" (the third step
// was refused, so the fourth was skipped), "done", "declined", "expired",
// "superseded" (failed, and a later plan picks up from it) or "held" (an
// agent's shadow switch keeps it from being decided). Approving, declining
// and undoing from the card move the scenario on, so the card can be driven.
import {
  DESK_PLAN_AGENT_ID,
  DESK_PLAN_ARTIFACT_ID,
  DESK_PLAN_DIGEST,
  DESK_PLAN_ID,
  DESK_PLAN_LATER_ID,
  DESK_PLAN_SLUG,
  DESK_PLAN_STEPS,
  DESK_PLAN_SUMMARY,
  DESK_PLAN_THREAD_ID,
  DESK_PLAN_TITLE,
  deskPlanStepChanges,
} from "../fixtures/desk-plan.mjs";
import { TriageError } from "./triage-model.mjs";

const THREAD = DESK_PLAN_THREAD_ID;
const STAGES = [
  "awaiting",
  "held",
  "approving",
  "running",
  "failed",
  "done",
  "declined",
  "expired",
  "superseded",
];
const QUESTION = "Tyson turned the truck away on S-240201. Sort it out for tomorrow morning.";
const FAILURE =
  "Shipment S-240201 is on invoice INV-10392, which was posted at 10:44 AM; a posted invoice cannot take new charges.";
const DECLINE_NOTE = "Hold off. I'm calling Tyson myself; keep Rosa on it.";
const APPROVE_NOTE = "Go ahead, Samuel is already at the yard.";

function stageOf(state) {
  const stage = state.scenario.deskPlan ?? "awaiting";
  return STAGES.includes(stage) ? stage : "awaiting";
}

/** When each thing happened, relative to the board's anchor. */
function times(anchor) {
  return {
    asked: anchor - 50 * 60,
    proposed: anchor - 48 * 60,
    decided: anchor - 47 * 60,
    committed: anchor - 47 * 60 + 10,
    step1: anchor - 46 * 60,
    step2: anchor - 46 * 60 + 20,
    step3: anchor - 45 * 60,
    step4: anchor - 45 * 60 + 15,
  };
}

function message(id, sequence, role, content, createdAt, extra = {}) {
  return {
    id,
    threadId: THREAD,
    sequence,
    role,
    content,
    kind: "Message",
    toolFailed: false,
    refused: false,
    createdAt,
    ...extra,
  };
}

function messages(state) {
  const at = times(state.anchor);
  return [
    message("amsg_plan_u1", 1, "User", QUESTION, at.asked),
    message(
      "amsg_plan_a1",
      2,
      "Assistant",
      "Tyson can take it at 8:00 tomorrow under confirmation TY-55187. Rosa runs out of hours before then, so I've put Samuel on it, billed the four hours at the dock and drafted the note to Tyson's receiving desk. The four changes wait on your approval as one plan.",
      at.proposed,
    ),
  ];
}

/** How each step stands at each stage, read in order: the statuses the server writes. */
function stepStatuses(stage) {
  switch (stage) {
    case "awaiting":
    case "held":
      return ["Pending", "Pending", "Pending", "Pending"];
    case "approving":
      return ["Approving", "Approving", "Approving", "Approving"];
    case "running":
      return ["Executed", "Executed", "Accepted", "Pending"];
    case "failed":
    case "superseded":
      return ["Executed", "Executed", "ExecutionFailed", "Skipped"];
    case "done":
      return ["Executed", "Executed", "Executed", "Executed"];
    case "declined":
      return ["Rejected", "Rejected", "Rejected", "Rejected"];
    default:
      return ["Expired", "Expired", "Expired", "Expired"];
  }
}

const RESULTS = [
  "Appointment confirmed with Tyson (TY-55187)",
  "Samuel Okafor dispatched on T-101",
  "Detention added; S-240201 now totals $2,490.00",
  "Sent to 2 recipients",
];

function proposals(state) {
  const stage = stageOf(state);
  const at = times(state.anchor);
  const statuses = stepStatuses(stage);
  const ranAt = [at.step1, at.step2, at.step3, at.step4];
  const decided = !["awaiting", "held", "expired"].includes(stage);

  return DESK_PLAN_STEPS.map((step, index) => {
    const status = statuses[index];
    const executed = status === "Executed";
    const failed = status === "ExecutionFailed";
    return {
      id: step.id,
      runId: "arun_tyson_recover",
      toolName: step.toolName,
      arguments: {},
      rationale: step.rationale,
      autonomyTier: "Propose",
      status,
      sourceMessageId: "amsg_plan_a1",
      confidence: 0.9,
      executedAt: executed || failed ? ranAt[index] : null,
      executionError: failed ? FAILURE : "",
      executionResult: executed
        ? { action: "updated", kind: "", name: "", ids: {}, state: RESULTS[index] }
        : null,
      expiresAt: at.proposed + 7 * 86400,
      hold: stage === "held" ? { reason: "AgentShadow", agentName: "Dispatch" } : null,
      planId: DESK_PLAN_ID,
      planStep: index + 1,
      fields: [],
      agentId: DESK_PLAN_AGENT_ID,
      agentName: "Dispatch",
      createdAt: at.proposed,
      decidedAt: decided ? at.decided : null,
      decidedByUserId: decided ? "usr_mock" : "",
      decisionNote: stage === "declined" ? DECLINE_NOTE : decided ? APPROVE_NOTE : "",
      reviewRequired: index === 3,
    };
  });
}

function plan(state) {
  const stage = stageOf(state);
  const at = times(state.anchor);
  const base = {
    id: DESK_PLAN_ID,
    runId: "arun_tyson_recover",
    title: DESK_PLAN_TITLE,
    summary: DESK_PLAN_SUMMARY,
    stepCount: 4,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    decidedByName: "",
    decisionNote: "",
    previewDigest: "",
    commitsAt: null,
    undoneAt: null,
    runMode: "InOrder",
    expiresAt: at.proposed + 7 * 86400,
    hold: null,
    createdAt: at.proposed,
    agentId: DESK_PLAN_AGENT_ID,
    agentName: "Dispatch",
  };
  const approved = {
    decidedAt: at.decided,
    decidedByUserId: "usr_mock",
    decidedByName: "Maya Chen",
    decisionNote: APPROVE_NOTE,
    previewDigest: DESK_PLAN_DIGEST,
    commitsAt: at.committed,
  };

  switch (stage) {
    case "held":
      return { ...base, status: "Pending", hold: { reason: "AgentShadow", agentName: "Dispatch" } };
    case "approving":
      return {
        ...base,
        ...approved,
        status: "Approving",
        decidedAt: state.anchor - 4,
        commitsAt: state.anchor + 600,
      };
    case "running":
      return { ...base, ...approved, status: "Approved", completedSteps: 2 };
    case "failed":
    case "superseded":
      return {
        ...base,
        ...approved,
        status: "Failed",
        completedSteps: 2,
        failedStep: 3,
        failureError: FAILURE,
      };
    case "done":
      return { ...base, ...approved, status: "Completed", completedSteps: 4 };
    case "declined":
      return {
        ...base,
        status: "Rejected",
        decidedAt: at.decided,
        decidedByUserId: "usr_mock",
        decidedByName: "Maya Chen",
        decisionNote: DECLINE_NOTE,
      };
    case "expired":
      return { ...base, status: "Expired", expiresAt: state.anchor - 3600 };
    default:
      return { ...base, status: "Pending" };
  }
}

function plans(state) {
  const out = [plan(state)];
  if (stageOf(state) === "superseded") {
    out.push({
      ...plan({ ...state, scenario: { ...state.scenario, deskPlan: "awaiting" } }),
      id: DESK_PLAN_LATER_ID,
      title: "Finish recovering S-240201",
      summary: "Bills the detention on a credit memo and tells Tyson.",
      stepCount: 2,
      createdAt: state.anchor - 30 * 60,
    });
  }
  return out;
}

function artifact(state) {
  const at = times(state.anchor);
  return {
    id: DESK_PLAN_ARTIFACT_ID,
    threadId: THREAD,
    messageId: "amsg_plan_a1",
    runId: "arun_tyson_recover",
    proposalId: "",
    planId: DESK_PLAN_ID,
    kind: "plan",
    status: "Pending",
    title: DESK_PLAN_TITLE,
    payload: {
      title: DESK_PLAN_TITLE,
      summary: DESK_PLAN_SUMMARY,
      stepCount: 4,
      steps: DESK_PLAN_STEPS.map((step, index) => ({
        step: index + 1,
        proposalId: step.id,
        toolName: step.toolName,
        rationale: step.rationale,
      })),
    },
    sourceToolCallId: "",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: DESK_PLAN_SLUG,
    turn: QUESTION,
    createdAt: at.proposed,
    updatedAt: at.proposed,
  };
}

/** The plan's preview: what each step would do, or, once decided, what it was approved as. */
function preview(state) {
  const stage = stageOf(state);
  const statuses = stepStatuses(stage);
  const recorded = !["awaiting", "held"].includes(stage);
  const steps = deskPlanStepChanges().map((step, index) => ({
    proposalId: DESK_PLAN_STEPS[index].id,
    step: index + 1,
    preview: {
      proposalId: DESK_PLAN_STEPS[index].id,
      tool: DESK_PLAN_STEPS[index].toolName,
      summary: step.summary,
      coverage: "Full",
      changes: step.changes,
      warnings: statuses[index] === "Pending" || recorded ? step.warnings : [],
      stale: false,
      staleness: { pinned: true, proposedVersion: 12, currentVersion: 12, missing: false },
      targetVersion: 12,
      withheldCount: 0,
      omittedRecords: 0,
      recorded,
      computedAt: state.anchor - 30,
      digest: `sha256:step${index + 1}`,
      reviewRequired: index === 3 && !recorded,
    },
  }));
  return {
    planId: DESK_PLAN_ID,
    steps,
    digest: recorded ? "" : DESK_PLAN_DIGEST,
    stale: false,
    withheldCount: 0,
    computedAt: state.anchor - 30,
    reviewRequired: !recorded,
  };
}

function thread(state) {
  const list = messages(state);
  return {
    id: THREAD,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: DESK_PLAN_AGENT_ID,
    title: QUESTION,
    status: "Active",
    lastMessageAt: list.at(-1).createdAt,
    lastReadAt: state.anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    version: 2,
    createdAt: list[0].createdAt,
    updatedAt: list.at(-1).createdAt,
  };
}

/** The conversation this file serves, for the rail's list and shelf counts. */
export function deskPlanThreads(state) {
  return [thread(state)];
}

function budget(state) {
  const tomorrow = Math.floor(state.anchor / 86400) * 86400 + 86400;
  return {
    agentName: "Dispatch",
    spentUsd: "3.10",
    limitUsd: "100.00",
    share: 0.031,
    near: false,
    monthStart: Math.floor(state.anchor / 86400) * 86400 - 15 * 86400,
    resetsAt: tomorrow + 15 * 86400,
    runsToday: 2,
    dailyRunLimit: 100,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 4, limit: 500, resetsAt: tomorrow },
  };
}

const route = (rest = "") => new RegExp(`^/api/v1/assistant/threads/${THREAD}${rest}/?$`);

export const DESK_PLAN_ROUTES = [
  ["GET", route(), thread],
  [
    "GET",
    route("/messages"),
    (state) => {
      const list = messages(state);
      return { results: list, hasMore: false, total: list.length, limit: 50 };
    },
  ],
  [
    "GET",
    route("/artifacts"),
    (state) => ({
      results: [artifact(state)],
      total: 1,
      nextCursor: "",
      counts: { all: 1, pinned: 0, families: { work: 1 } },
    }),
  ],
  [
    "GET",
    route("/artifacts/by-slug/([a-z0-9-]+)"),
    (state, path) => {
      const slug = path.match(/by-slug\/([a-z0-9-]+)/)?.[1] ?? "";
      return { results: slug === DESK_PLAN_SLUG ? [artifact(state)] : [] };
    },
  ],
  [
    "GET",
    route("/artifacts/(aart_[a-z0-9_]+)"),
    (state, path) => {
      const id = path.match(/artifacts\/(aart_[a-z0-9_]+)/)?.[1] ?? "";
      return { results: id === DESK_PLAN_ARTIFACT_ID ? [artifact(state)] : [] };
    },
  ],
  ["GET", route("/proposals"), (state) => ({ results: proposals(state) })],
  ["GET", route("/plans"), (state) => ({ results: plans(state) })],
  ["GET", route("/schedules"), () => ({ items: [], total: 0 })],
  ["GET", route("/waits"), () => ({ items: [] })],
  ["GET", route("/queue"), () => ({ items: [] })],
  ["GET", route("/budget"), budget],
  ["GET", route("/turns/active"), () => ({ turn: null })],
];

function ownPlan(id) {
  if (id !== DESK_PLAN_ID) {
    throw new Error(`no plan ${id}`);
  }
}

function decided(state) {
  const current = plan(state);
  return {
    id: current.id,
    status: current.status,
    stepCount: current.stepCount,
    completedSteps: current.completedSteps,
    failedStep: current.failedStep,
    failureError: current.failureError,
    decidedByUserId: current.decidedByUserId || null,
    decidedAt: current.decidedAt,
    commitsAt: current.commitsAt,
    version: 3,
    updatedAt: state.anchor,
  };
}

export const DESK_PLAN_HANDLERS = {
  MyPlanPreview: (state, variables) => {
    ownPlan(variables.id);
    return { myPlanPreview: preview(state) };
  },
  DecideMyPlan: (state, variables) => {
    ownPlan(variables.id);
    if (stageOf(state) !== "awaiting") {
      throw new Error("This plan is no longer waiting on a decision");
    }
    if (variables.input.decision === "Rejected") {
      state.scenario.deskPlan = "declined";
    } else {
      if (variables.input.previewDigest !== DESK_PLAN_DIGEST) {
        throw new TriageError(
          "What this plan would do has changed since you reviewed it. Review it again, then decide",
          { code: "CONFLICT", type: "resource-conflict" },
        );
      }
      state.scenario.deskPlan = "approving";
    }
    return { decideMyPlan: { refused: false, plan: decided(state) } };
  },
  UndoMyDecision: (state, variables) => {
    ownPlan(variables.planId);
    state.scenario.deskPlan = "awaiting";
    return { undoMyDecision: true };
  },
};
