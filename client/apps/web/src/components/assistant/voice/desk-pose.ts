import type { ToolStep } from "../activity";
import { isTurnActive, type TurnState } from "../turn-stream";

/**
 * What the drawn desk is doing while an agent works.
 *
 * - arrive: waiting on the model; the chair rolls out, someone sits, it rolls back in
 * - busy: a tool is running; the desk gives a small busy shake
 * - write: the answer is arriving; the monitor pulses softly
 */
export type DeskWorkingPose = "arrive" | "busy" | "write";

/**
 * Every pose the drawing has. `settle` closes a turn (the chair tucks in and
 * the screen goes dark); `idle` is the desk at rest, as the Desk's greeting
 * draws it.
 */
export type DeskPose = DeskWorkingPose | "settle" | "idle";

/**
 * The pose for the moment of a turn, read the same way as the working line's
 * words, so the drawing and the sentence beside it never disagree: a tool
 * under way is the desk at work, words arriving are the screen, and anything
 * that is the model deciding what to do next is someone sitting down to it.
 * A turn that is over settles.
 */
export function thinkingPose(
  turn: TurnState,
  steps: readonly ToolStep[],
): DeskWorkingPose | "settle" {
  if (!isTurnActive(turn)) {
    return "settle";
  }
  if (turn.status === "guarding" || turn.retrying) {
    return "arrive";
  }
  if (steps.some((step) => step.status === "running")) {
    return "busy";
  }
  const last = turn.segments.at(-1);
  if (last?.kind === "text" && !last.closed) {
    return "write";
  }

  return "arrive";
}
