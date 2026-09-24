import type { ToolEffect } from "@/types/assistant";
import type { ToolStep } from "../activity";
import { isWebTool } from "../tool-presentation";
import { isTurnActive, type TurnState } from "../turn-stream";

/**
 * What the lamp does while a tool runs, one motion per kind of tool. The
 * kind is the tool's effect, so a new tool is drawn the moment the server
 * classifies it and nothing here lists tool names; reading the web is the
 * one exception, because it is a lookup that leaves the product.
 *
 * - lookup: the head sweeps and holds, reading along the desk
 * - discover: the head sways while the beam widens and narrows
 * - navigate: the light slides off the desk and a new pool arrives
 * - change: the head stamps twice and a tick appears in the light
 * - present: a card rises in the light
 * - ask: the lamp turns to face the reader and blinks
 * - delegate: a second lamp arrives and the light passes to it
 * - web: the head turns up and out, sending signal waves
 */
export type DeskToolPose =
  | "lookup"
  | "discover"
  | "navigate"
  | "change"
  | "present"
  | "ask"
  | "delegate"
  | "web";

/**
 * What the lamp does while the agent works.
 *
 * - start: the question is being checked; the light switches on, once
 * - think: the model is deciding; the light swells and eases
 * - write: the answer is arriving; lines light up on the desk
 * - retry: a reply is starting over; the light cuts out and comes back
 * - a tool pose while a tool runs
 */
export type DeskWorkingPose = "start" | "think" | "write" | "retry" | DeskToolPose;

/**
 * The beat a turn closes on, once, before the mark is gone.
 *
 * - done: the head dips and the light goes out
 * - await: the light turns amber, because a proposed write waits on a person
 * - failed: the light flickers out, the head droops, and the bulb goes red
 */
export type DeskClosingPose = "done" | "await" | "failed";

export type DeskPose = DeskWorkingPose | DeskClosingPose;

const CLOSING: ReadonlySet<DeskPose> = new Set<DeskClosingPose>(["done", "await", "failed"]);

export function isClosingPose(pose: DeskPose): pose is DeskClosingPose {
  return CLOSING.has(pose);
}

const EFFECT_POSE: Readonly<Record<ToolEffect, DeskToolPose>> = {
  lookup: "lookup",
  discover: "discover",
  navigate: "navigate",
  change: "change",
  present: "present",
  ask: "ask",
  delegate: "delegate",
};

/** The motion for one running call: the web by name, everything else by its effect. */
export function toolPose(step: Pick<ToolStep, "name" | "effect">): DeskToolPose {
  return isWebTool(step.name) ? "web" : EFFECT_POSE[step.effect];
}

/**
 * The pose for the moment of a turn, read the same way as the working line's
 * words, so the drawing and the sentence beside it never disagree: the guard
 * checking is the light coming on, a retry is the light cutting out, the step
 * under way is the latest running call (the one the words name), words
 * arriving are lines on the desk, and anything else is the model thinking.
 *
 * A turn that is over closes on what it came to: a failure, a proposed write
 * still waiting on a person, or done.
 */
export function thinkingPose(turn: TurnState, steps: readonly ToolStep[]): DeskPose {
  if (!isTurnActive(turn)) {
    if (turn.status === "error") {
      return "failed";
    }
    return steps.some((step) => step.status === "proposed") ? "await" : "done";
  }
  if (turn.status === "guarding") {
    return "start";
  }
  if (turn.retrying) {
    return "retry";
  }
  for (let index = steps.length - 1; index >= 0; index -= 1) {
    const step = steps[index];
    if (step?.status === "running") {
      return toolPose(step);
    }
  }
  const last = turn.segments.at(-1);
  if (last?.kind === "text" && !last.closed) {
    return "write";
  }

  return "think";
}
