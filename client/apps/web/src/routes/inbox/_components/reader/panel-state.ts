import type { TriageAction } from "@/lib/graphql/inbox";

export type DraftKind = "load" | "reply";
export type PanelState = {
  open: string | null;
  drafting: Record<DraftKind, boolean>;
  sheet: boolean;
  done: ReadonlyMap<string, boolean>;
};
export type PanelEvent =
  | { type: "toggle"; id: string }
  | { type: "open"; id: string }
  | { type: "close" }
  | { type: "draftStarted"; draft: DraftKind; stepId: string }
  | { type: "draftSettled"; draft: DraftKind }
  | { type: "stepsChanged"; steps: readonly Pick<TriageAction, "id" | "done">[] }
  | { type: "sheet"; open: boolean };
export type StepContext = { hasLoadDraft: boolean; hasReplyDraft: boolean; replyInFlight: boolean };
export type StepStart = "link" | "draftLoad" | "draftReply" | "compose" | "toggle";
export const INITIAL_PANEL: PanelState = {
  open: null,
  drafting: { load: false, reply: false },
  sheet: false,
  done: new Map(),
};
export function panelReducer(state: PanelState, _event: PanelEvent): PanelState {
  return state;
}
export function stepStart(_action: TriageAction, _context: StepContext): StepStart {
  return "toggle";
}
