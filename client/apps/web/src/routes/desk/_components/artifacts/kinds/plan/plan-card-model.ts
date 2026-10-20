import { classifyPlan, planStepState } from "@/components/assistant/plan-state";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import {
  dependencySteps,
  formatPreviewAmount,
} from "@/components/assistant/proposal-preview/preview-format";
import type {
  PlanPreview,
  PreviewRecordChange,
  PreviewRecordLink,
  ProposalPreview,
} from "@/lib/graphql/agent-preview";
import type { AssistantPlan, AssistantProposal, PlanRunMode } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PlanArtifact } from "../../artifact-payloads";

/**
 * Where a plan stands as its card draws it. `undoable` is an approval still
 * inside its undo window: decided, and nothing has run.
 */
export type PlanPhase =
  | "awaiting"
  | "held"
  | "undoable"
  | "running"
  | "failed"
  | "done"
  | "declined"
  | "expired";

/** How one step reads on the track: its node, its segment and its time. */
export type PlanStepTone = "wait" | "next" | "run" | "done" | "fail" | "skip";

export function planPhase(plan: AssistantPlan, now: number): PlanPhase {
  if (plan.status === "Approving" && (plan.commitsAt ?? 0) > now) {
    return "undoable";
  }

  switch (classifyPlan(plan, now)) {
    case "awaiting":
      return "awaiting";
    case "held":
      return "held";
    case "running":
      return "running";
    case "done":
      return "done";
    case "failed":
      return "failed";
    case "declined":
      return "declined";
    default:
      return "expired";
  }
}

/** Whether the plan is still to be decided: every change is shown in full until it is. */
export function undecided(phase: PlanPhase): boolean {
  return phase === "awaiting" || phase === "held";
}

/** Whether the plan was decided and its steps had their turn, or never will. */
export function settled(phase: PlanPhase): boolean {
  return phase === "failed" || phase === "done" || phase === "declined" || phase === "expired";
}

export function stepTone(step: AssistantProposal | undefined, phase: PlanPhase): PlanStepTone {
  if (step === undefined) {
    return undecided(phase) || phase === "undoable" ? "wait" : "skip";
  }

  switch (planStepState(step)) {
    case "done":
    case "simulated":
      return "done";
    case "failed":
      return "fail";
    case "running":
      return phase === "undoable" ? "wait" : "run";
    case "skipped":
    case "declined":
      return "skip";
    default:
      if (undecided(phase) || phase === "undoable") {
        return "wait";
      }
      return phase === "running" ? "next" : "skip";
  }
}

/** One step of the plan as the card draws it. */
export type PlanStepView = {
  number: number;
  proposalId: string;
  /** The agent's own sentence for the step, else its tool in words. */
  title: string;
  toolName: string;
  tone: PlanStepTone;
  /** When the step ran, in Unix seconds; null for one that has not. */
  ranAt: number | null;
  /** Why the step's write did not go through, as the server said it. */
  error: string;
  /** Where the record stands once the step ran, as the tool said it. */
  result: string;
  /** How many of a write over many records did not go through. */
  partial: { done: number; total: number } | null;
  preview: ProposalPreview | null;
  /** The earlier steps whose records this one starts from. */
  dependsOn: number[];
};

function stepNumber(step: AssistantProposal, index: number): number {
  return step.planStep > 0 ? step.planStep : index + 1;
}

/**
 * The plan's steps in the order they run, each with what the preview says it
 * changes. The steps the thread lists come first; a step the artifact named
 * that is no longer listed is still drawn, from the artifact alone.
 */
export function planSteps(
  payload: PlanArtifact,
  steps: readonly AssistantProposal[],
  preview: PlanPreview | undefined,
  phase: PlanPhase,
): PlanStepView[] {
  const previews = new Map((preview?.steps ?? []).map((step) => [step.proposalId, step.preview]));
  const listed = [...steps].sort((a, b) => a.planStep - b.planStep);
  const views: PlanStepView[] = listed.map((step, index) => {
    const stepPreview = previews.get(step.id) ?? null;
    const result = step.executionResult;
    const failed = result?.failed ?? [];
    const total = result?.total ?? failed.length;

    return {
      number: stepNumber(step, index),
      proposalId: step.id,
      title: step.rationale.trim() || humanizeToolName(step.toolName),
      toolName: step.toolName,
      tone: stepTone(step, phase),
      ranAt: step.executedAt ?? null,
      error: step.executionError,
      result: result?.state ?? "",
      partial: failed.length > 0 ? { done: Math.max(0, total - failed.length), total } : null,
      preview: stepPreview,
      dependsOn: stepPreview ? dependencySteps(stepPreview) : [],
    };
  });

  const known = new Set(views.map((view) => view.proposalId));
  for (const step of payload.steps) {
    if (step.proposalId !== "" && known.has(step.proposalId)) {
      continue;
    }
    views.push({
      number: step.step > 0 ? step.step : views.length + 1,
      proposalId: step.proposalId,
      title: step.rationale.trim() || humanizeToolName(step.toolName),
      toolName: step.toolName,
      tone: stepTone(undefined, phase),
      ranAt: null,
      error: "",
      result: "",
      partial: null,
      preview: null,
      dependsOn: [],
    });
  }

  return views.sort((a, b) => a.number - b.number);
}

/** The steps' warnings a person should read before the step runs, as the preview raised them. */
export function stepWarnings(step: PlanStepView): ProposalPreview["warnings"] {
  if (step.preview === null) {
    return [];
  }

  return step.preview.warnings.filter(
    (warning) => warning.code !== "depends_on_step" && warning.code !== "withheld",
  );
}

/** Whether a step's changes are folded to one line: once decided, except where something needs reading. */
export function foldedByDefault(step: PlanStepView, phase: PlanPhase): boolean {
  if (undecided(phase) || phase === "undoable") {
    return false;
  }

  return step.tone !== "fail" && stepWarnings(step).length === 0;
}

/** One record the plan changes, with every step that changes it. */
export type PlanRecordView = {
  key: string;
  label: string;
  record: PreviewRecordLink | null;
  steps: number[];
  /** The fields the plan changes on it, by their labels, in the order first changed. */
  fields: string[];
  sends: number;
  creates: boolean;
};

function recordKey(change: PreviewRecordChange): string {
  return `${change.resource}:${change.entityId || change.label}`;
}

/**
 * Every record the plan touches, once each, in the order the steps first
 * reach it. A record the reader may not see is counted, never named.
 */
export function recordsTouched(steps: readonly PlanStepView[]): {
  records: PlanRecordView[];
  withheld: number;
} {
  const records = new Map<string, PlanRecordView>();
  let withheld = 0;

  for (const step of steps) {
    for (const change of step.preview?.changes ?? []) {
      if (change.withheld) {
        withheld++;
        continue;
      }
      if (change.operation === "Run") {
        continue;
      }
      const key = recordKey(change);
      const record = records.get(key) ?? {
        key,
        label: change.label,
        record: change.record ?? null,
        steps: [],
        fields: [],
        sends: 0,
        creates: false,
      };
      if (!record.steps.includes(step.number)) {
        record.steps.push(step.number);
      }
      for (const field of change.fields) {
        if (!field.withheld && field.label !== "" && !record.fields.includes(field.label)) {
          record.fields.push(field.label);
        }
      }
      if (change.operation === "Send") {
        record.sends++;
      }
      if (change.operation === "Create") {
        record.creates = true;
      }
      records.set(key, record);
    }
    withheld += step.preview?.withheldCount ?? 0;
  }

  return { records: [...records.values()], withheld };
}

const RECORD_FIELDS_SHOWN = 3;

/** What the plan does to one record, as the line under its name. */
export function recordLine(record: PlanRecordView, t: TranslateFn): string {
  const parts: string[] = [];
  if (record.creates) {
    parts.push(t("New record"));
  }
  if (record.fields.length > 0 && !record.creates) {
    const shown = record.fields.slice(0, RECORD_FIELDS_SHOWN).join(", ");
    const more = record.fields.length - RECORD_FIELDS_SHOWN;
    parts.push(more > 0 ? t("{0} and {1} more", shown, more) : shown);
  }
  if (record.sends > 0) {
    parts.push(t("{0, plural, one {# message} other {# messages}}", record.sends));
  }

  return parts.join(" · ");
}

function moneyDelta(change: PreviewRecordChange): string | null {
  const money = change.money;
  if (!money || money.withheld || money.delta === null || money.delta === undefined) {
    return null;
  }
  if (Number(money.delta) === 0) {
    return null;
  }

  return formatPreviewAmount(money.delta, money.currency, { signed: true });
}

/** A step's changes folded to one line: "Changed 3 fields on S-240201 · +$120.00". */
export function foldedLine(preview: ProposalPreview | null, t: TranslateFn): string {
  if (preview === null) {
    return "";
  }

  const parts: string[] = [];
  for (const change of preview.changes) {
    if (change.withheld) {
      parts.push(t("A record you cannot see"));
      continue;
    }
    const label = change.label;
    switch (change.operation) {
      case "Create":
        parts.push(label === "" ? t("New record") : t("New {0}", label));
        break;
      case "Send": {
        const recipients = change.message?.to.length ?? 0;
        parts.push(
          recipients > 0
            ? t(
                "{0, plural, one {Message to # recipient} other {Message to # recipients}}",
                recipients,
              )
            : t("A message"),
        );
        break;
      }
      case "Delete":
        parts.push(t("Deleted {0}", label));
        break;
      case "Archive":
        parts.push(t("Archived {0}", label));
        break;
      case "Run":
        parts.push(preview.summary || t("Ran the tool"));
        break;
      default: {
        const fields = change.fields.length;
        parts.push(
          label === ""
            ? t("{0, plural, one {Changed # field} other {Changed # fields}}", fields)
            : fields === 1
              ? t("Changed 1 field on {0}", label)
              : t("Changed {0} fields on {1}", fields, label),
        );
      }
    }
    const delta = moneyDelta(change);
    if (delta !== null) {
      parts.push(delta);
    }
  }

  return parts.join(" · ");
}

/** How many of the plan's steps have gone through. */
export function stepsRun(steps: readonly PlanStepView[]): number {
  return steps.filter((step) => step.tone === "done").length;
}

/** The step the plan is on, or stopped at: the first running, else the first failed. */
export function currentStep(steps: readonly PlanStepView[]): PlanStepView | undefined {
  return (
    steps.find((step) => step.tone === "run") ??
    steps.find((step) => step.tone === "next") ??
    steps.find((step) => step.tone === "fail")
  );
}

/** The plan's status, as the pill under its name says it. */
export function phaseLabel(
  phase: PlanPhase,
  steps: readonly PlanStepView[],
  runMode: PlanRunMode | null | undefined,
  canDecide: boolean,
  t: TranslateFn,
): string {
  switch (phase) {
    case "awaiting":
      return canDecide ? t("Awaiting your approval") : t("Awaiting approval");
    case "held":
      return t("On hold");
    case "undoable":
      return t("Approved");
    case "running": {
      const step = currentStep(steps);
      return step === undefined || runMode === "EachStep"
        ? t("Running")
        : t("Running step {0}", step.number);
    }
    case "failed": {
      const failed = steps.filter((step) => step.tone === "fail");
      if (runMode === "EachStep" || failed.length > 1) {
        return t(
          "{0, plural, one {# step did not go through} other {# steps did not go through}}",
          Math.max(1, failed.length),
        );
      }
      const step = failed[0];
      return step === undefined ? t("Stopped") : t("Stopped at step {0}", step.number);
    }
    case "done":
      return t("Done");
    case "declined":
      return t("Declined");
    case "expired":
      return t("Expired");
  }
}

/** How the steps run, as the aside on the steps' section says it. */
export function runModeLine(
  phase: PlanPhase,
  runMode: PlanRunMode | null | undefined,
  steps: readonly PlanStepView[],
  t: TranslateFn,
): string {
  switch (phase) {
    case "done":
      return t("{0, plural, one {The step ran} other {All # steps ran}}", steps.length);
    case "declined":
    case "expired":
      return t("None of these ran");
    case "failed":
      return runMode === "EachStep"
        ? t("Every step had its turn")
        : t("Stopped at the first failure");
    default:
      return runMode === "EachStep"
        ? t("Decided whole · every step runs on its own")
        : t("Decided whole · stops at the first failure");
  }
}

/** Whether a later plan in the conversation took over from this one. */
export function supersededBy(
  plan: AssistantPlan,
  plans: readonly AssistantPlan[],
): AssistantPlan | null {
  return (
    plans
      .filter((candidate) => candidate.id !== plan.id && candidate.createdAt > plan.createdAt)
      .sort((a, b) => a.createdAt - b.createdAt)[0] ?? null
  );
}

/** Whether the plan stopped with work left undone, which the agent may be asked to finish. */
export function unfinished(phase: PlanPhase, steps: readonly PlanStepView[]): boolean {
  if (phase !== "failed" && phase !== "expired") {
    return false;
  }

  return steps.some((step) => step.tone !== "done");
}

function stepList(numbers: readonly number[]): string {
  return numbers.join(", ");
}

/**
 * What the person sends the agent to finish a plan that stopped: the plan by
 * name, the step it stopped at and the server's own words for why, what did
 * not run, and that the rest is to be proposed again for their approval.
 */
export function finishMessage(
  plan: AssistantPlan,
  steps: readonly PlanStepView[],
  phase: PlanPhase,
  t: TranslateFn,
): string {
  const left = steps.filter((step) => step.tone !== "done").map((step) => step.number);
  const ran = steps.filter((step) => step.tone === "done").map((step) => step.number);

  if (phase === "expired") {
    return ran.length === 0
      ? t(
          'The plan "{0}" expired before anyone decided it, so none of its steps ran. Propose what is still needed again, with the records as they are now, for me to approve.',
          plan.title,
        )
      : t(
          'The plan "{0}" expired with steps {1} not run. Propose them again, with the records as they are now, for me to approve.',
          plan.title,
          stepList(left),
        );
  }

  const failed = steps.find((step) => step.tone === "fail");
  const stepNumber = failed?.number ?? plan.failedStep ?? 0;
  const reason = (failed?.error || plan.failureError).trim();
  const stopped =
    reason === ""
      ? t('The plan "{0}" stopped at step {1}.', plan.title, stepNumber)
      : t('The plan "{0}" stopped at step {1}: {2}', plan.title, stepNumber, reason);
  const done =
    ran.length === 0
      ? t("None of its steps went through.")
      : ran.length === 1
        ? t("Step {0} went through.", ran[0])
        : t("Steps {0} went through.", stepList(ran));
  const undone =
    left.length === 1
      ? t("Step {0} did not run.", left[0])
      : t("Steps {0} did not run.", stepList(left));

  return [
    stopped,
    done,
    ...(ran.length === 0 ? [] : [undone]),
    t("Propose the rest of the plan, fixed so it goes through, for me to approve."),
  ].join(" ");
}

/** One entry of the plan's approval history, oldest first. */
export type PlanLogEntry = {
  id: string;
  text: string;
  at: number | null;
  /** Something the person wrote, quoted under the entry. */
  note: string;
  mine: boolean;
};

/** What happened to the plan, from its proposal to its last step. */
export function planLog(
  plan: AssistantPlan,
  steps: readonly PlanStepView[],
  phase: PlanPhase,
  viewerId: string,
  t: TranslateFn,
): PlanLogEntry[] {
  const agent = plan.agentName?.trim() || t("The agent");
  const entries: PlanLogEntry[] = [
    {
      id: "proposed",
      text: t(
        "{0} proposed {1, plural, one {# change} other {# changes}}",
        agent,
        Math.max(plan.stepCount, steps.length),
      ),
      at: plan.createdAt,
      note: "",
      mine: false,
    },
  ];
  const decider = (plan.decidedByName ?? "").trim();
  const mine = plan.decidedByUserId !== "" && plan.decidedByUserId === viewerId;
  const who = mine ? t("You") : decider === "" ? t("Someone") : decider;

  if ((plan.undoneAt ?? 0) > 0) {
    entries.push({
      id: "undone",
      text: t("An approval was taken back in its undo window"),
      at: plan.undoneAt ?? null,
      note: "",
      mine: false,
    });
  }

  if (plan.status === "Rejected") {
    entries.push({
      id: "declined",
      text: t("{0} declined the plan", who),
      at: plan.decidedAt ?? null,
      note: plan.decisionNote ?? "",
      mine,
    });
    return entries;
  }

  if (phase === "expired") {
    entries.push({
      id: "expired",
      text: t("Expired without a decision"),
      at: plan.expiresAt || null,
      note: "",
      mine: false,
    });
    return entries;
  }

  if (undecided(phase)) {
    if ((plan.expiresAt ?? 0) > 0) {
      entries.push({
        id: "waiting",
        text: t("Waiting for a decision"),
        at: null,
        note: "",
        mine: false,
      });
    }
    return entries;
  }

  entries.push({
    id: "approved",
    text: plan.previewDigest
      ? t("{0} approved the plan as previewed", who)
      : t("{0} approved the plan", who),
    at: plan.decidedAt ?? null,
    note: plan.decisionNote ?? "",
    mine,
  });

  if (phase === "undoable") {
    return entries;
  }
  if ((plan.commitsAt ?? 0) > 0) {
    entries.push({
      id: "committed",
      text: t("Went through after the undo window"),
      at: plan.commitsAt ?? null,
      note: "",
      mine: false,
    });
  }

  for (const step of steps) {
    if (step.tone === "done") {
      entries.push({
        id: `ran-${step.proposalId}`,
        text: t("Step {0} ran", step.number),
        at: step.ranAt,
        note: "",
        mine: false,
      });
    }
    if (step.tone === "fail") {
      entries.push({
        id: `failed-${step.proposalId}`,
        text: t("Step {0} did not go through", step.number),
        at: step.ranAt,
        note: "",
        mine: false,
      });
    }
  }

  const skipped = steps.filter((step) => step.tone === "skip").length;
  if (skipped > 0) {
    entries.push({
      id: "skipped",
      text: t("{0, plural, one {# step was skipped} other {# steps were skipped}}", skipped),
      at: null,
      note: "",
      mine: false,
    });
  }

  return entries;
}

/** The short form of a digest a person can compare: its first and last few characters. */
export function shortDigest(digest: string): string {
  const hex = digest.slice(digest.indexOf(":") + 1);
  return hex.length <= 12 ? hex : `${hex.slice(0, 4)}…${hex.slice(-4)}`;
}

/** Seconds left in an undo window, as a clock reads them: "9:54". */
export function countdown(seconds: number): string {
  const whole = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(whole / 60);
  return `${minutes}:${String(whole % 60).padStart(2, "0")}`;
}
