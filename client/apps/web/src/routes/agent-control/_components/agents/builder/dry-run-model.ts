import { z } from "zod";

export const dryRunOutcomes = [
  "Runs",
  "AskFirst",
  "Propose",
  "Recorded",
  "Simulated",
  "NotHeld",
  "Failed",
] as const;
export type DryRunOutcome = (typeof dryRunOutcomes)[number];

export type DryRunStep = {
  callId: string;
  tool: string;
  /** Absent while the call is still going. */
  outcome: DryRunOutcome | null;
  summary: string;
};

export type DryRunState = {
  status: "running" | "done" | "failed";
  steps: DryRunStep[];
  reply: string;
  error: string | null;
};

export const INITIAL_DRY_RUN: DryRunState = { status: "running", steps: [], reply: "", error: null };

const toolStarted = z.object({
  callId: z.string(),
  name: z.string(),
  agentId: z.string().nullish(),
});
const toolFinished = toolStarted.extend({ failed: z.boolean().default(false) });
const delta = z.object({ text: z.string(), agentId: z.string().nullish() });
const steps = z.object({
  steps: z
    .array(
      z.object({
        callId: z.string(),
        tool: z.string(),
        outcome: z.enum(dryRunOutcomes).catch("Runs"),
        summary: z.string().optional().default(""),
      }),
    )
    .default([]),
  reply: z.string().default(""),
});
const failure = z.object({ message: z.string() });

function parse<T>(schema: z.ZodType<T>, raw: string): T | null {
  try {
    const result = schema.safeParse(raw === "" ? {} : JSON.parse(raw));
    return result.success ? result.data : null;
  } catch {
    return null;
  }
}

/**
 * Folds one event of a dry run's stream into what the Try it panel shows. Calls a delegate
 * made are left out: they belong to the other agent. The closing account of the steps
 * replaces what was pieced together while streaming, since it knows what each call came to.
 */
export function reduceDryRun(state: DryRunState, event: string, raw: string): DryRunState {
  switch (event) {
    case "tool_started": {
      const data = parse(toolStarted, raw);
      if (!data || data.agentId || state.steps.some((step) => step.callId === data.callId)) {
        return state;
      }
      return {
        ...state,
        steps: [...state.steps, { callId: data.callId, tool: data.name, outcome: null, summary: "" }],
      };
    }
    case "tool_finished": {
      const data = parse(toolFinished, raw);
      if (!data || data.agentId) {
        return state;
      }
      const known = state.steps.some((step) => step.callId === data.callId);
      const finished: DryRunStep = {
        callId: data.callId,
        tool: data.name,
        outcome: data.failed ? "Failed" : "Runs",
        summary: "",
      };
      return {
        ...state,
        steps: known
          ? state.steps.map((step) =>
              step.callId === data.callId ? { ...step, outcome: finished.outcome } : step,
            )
          : [...state.steps, finished],
      };
    }
    case "delta": {
      const data = parse(delta, raw);
      if (!data || data.agentId) {
        return state;
      }
      return { ...state, reply: state.reply + data.text };
    }
    case "dry_run_steps": {
      const data = parse(steps, raw);
      if (!data) {
        return state;
      }
      return {
        ...state,
        steps: data.steps.map((step) => ({ ...step })),
        reply: data.reply || state.reply,
      };
    }
    case "done":
      return state.status === "failed" ? state : { ...state, status: "done" };
    case "error": {
      const data = parse(failure, raw);
      return {
        ...state,
        status: "failed",
        error: data?.message ?? "The dry run could not finish. Try again in a moment.",
      };
    }
    default:
      return state;
  }
}

/** The class the prototype colours each outcome tag with. */
export const OUTCOME_TONE: Record<DryRunOutcome, string> = {
  Runs: "r",
  AskFirst: "w",
  Propose: "p",
  Recorded: "p",
  Simulated: "s",
  NotHeld: "x",
  Failed: "x",
};
