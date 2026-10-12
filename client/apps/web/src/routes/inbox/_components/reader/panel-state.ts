import type { TriageAction } from "@/lib/graphql/inbox";

/**
 * The actions panel's own state: which step is open, which drafts Nova is
 * writing, and whether the phone's sheet is up. One reducer serves the panel,
 * the floating action bar and the sheet, so the bar's "Do it" and a click on
 * a step cannot disagree about what is open.
 */
export type DraftKind = "load" | "reply";

export type PanelState = {
  /** The step finished inline right now, by its stable id. */
  open: string | null;
  /** Nova is writing these, so the step shows the draft arriving and the bar waits. */
  drafting: Readonly<Record<DraftKind, boolean>>;
  /** The actions sheet on a phone. */
  sheet: boolean;
  /** Each step's done flag as last seen, to tell a step that just finished. */
  done: ReadonlyMap<string, boolean>;
};

export type PanelEvent =
  | { type: "toggle"; id: string }
  | { type: "open"; id: string }
  | { type: "close" }
  | { type: "draftStarted"; draft: DraftKind; stepId: string }
  | { type: "draftSettled"; draft: DraftKind }
  /** The message's steps as the server now says they are. */
  | { type: "stepsChanged"; steps: readonly Pick<TriageAction, "id" | "done">[] }
  | { type: "sheet"; open: boolean };

/** What the message holds that decides how a step starts. */
export type StepContext = {
  hasLoadDraft: boolean;
  hasReplyDraft: boolean;
  /** A reply is going out or was refused, so the composer has something to show. */
  replyInFlight: boolean;
};

/**
 * What starting a step does: open the link dialog, have Nova draft the load
 * or the reply, open an empty form or composer, or open (or close) the step.
 */
export type StepStart = "link" | "draftLoad" | "draftReply" | "compose" | "toggle";

export const INITIAL_PANEL: PanelState = {
  open: null,
  drafting: { load: false, reply: false },
  sheet: false,
  done: new Map(),
};

function sameDone(
  known: ReadonlyMap<string, boolean>,
  steps: readonly Pick<TriageAction, "id" | "done">[],
): boolean {
  return known.size === steps.length && steps.every((step) => known.get(step.id) === step.done);
}

export function panelReducer(state: PanelState, event: PanelEvent): PanelState {
  switch (event.type) {
    case "toggle":
      return { ...state, open: state.open === event.id ? null : event.id };
    case "open":
      return state.open === event.id ? state : { ...state, open: event.id };
    case "close":
      return state.open === null ? state : { ...state, open: null };
    case "draftStarted":
      return {
        ...state,
        open: event.stepId,
        drafting: { ...state.drafting, [event.draft]: true },
      };
    case "draftSettled":
      return state.drafting[event.draft]
        ? { ...state, drafting: { ...state.drafting, [event.draft]: false } }
        : state;
    case "stepsChanged": {
      if (sameDone(state.done, event.steps)) {
        return state;
      }
      const done = new Map(event.steps.map((step) => [step.id, step.done] as const));
      let open = state.open;
      if (open !== null) {
        const now = done.get(open);
        const finished = now === true && state.done.get(open) === false;
        if (now === undefined || finished) {
          open = null;
        }
      }
      return { ...state, open, done };
    }
    case "sheet":
      return state.sheet === event.open ? state : { ...state, sheet: event.open };
  }
}

export function stepStart(action: TriageAction, context: StepContext): StepStart {
  switch (action.kind) {
    case "Link":
      return "link";
    case "BuildLoad":
      if (action.done || context.hasLoadDraft) {
        return "toggle";
      }
      return action.novaCanDo ? "draftLoad" : "compose";
    case "Reply":
      if (action.done || context.hasReplyDraft || context.replyInFlight) {
        return "toggle";
      }
      return action.novaCanDo ? "draftReply" : "compose";
    case "File":
    case "CheckStops":
    case "LogException":
    case "BlockSender":
      return "toggle";
  }
}
