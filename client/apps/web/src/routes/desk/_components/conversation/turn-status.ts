import { currentActivity, stepsFromSegments } from "@/components/assistant/activity";
import type { TurnState } from "@/components/assistant/turn-stream";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DeskComposerStatus } from "../composer/desk-composer";

/**
 * What the composer says while a reply is being made, the one place the Desk
 * shows the agent's work in progress: the question being checked, a model
 * being asked again, the step under way, or the answer being written.
 * Nothing once the turn has finished, failed or been refused.
 */
export function composerStatus(turn: TurnState | null, t: TranslateFn): DeskComposerStatus | null {
  if (!turn || turn.status === "done" || turn.status === "error" || turn.status === "refused") {
    return null;
  }
  if (turn.status === "guarding") {
    return { text: t("Checking the question…"), pose: "check" };
  }
  if (turn.retrying) {
    const who = turn.retrying.provider !== "" ? turn.retrying.provider : t("The model");
    return turn.retrying.kind === "busy"
      ? {
          text: t("{0} is busy · retrying", who),
          pose: "retry",
          extra: t("Attempt {0}", turn.retrying.attempt + 1),
        }
      : { text: t("Starting the reply again…"), pose: "retry" };
  }

  const last = turn.segments.at(-1);
  if (last?.kind === "text" && !last.closed && last.text.trim() !== "") {
    return { text: t("Writing the answer…"), pose: "work" };
  }

  const activity = currentActivity(stepsFromSegments(turn.segments), t);
  if (activity && activity.state === "running") {
    return { text: `${activity.phrase}…`, pose: "work" };
  }

  return { text: t("Thinking it through…"), pose: "work" };
}

/** The words of the reply in progress, once there are any; the thread shows nothing before. */
export function streamingText(turn: TurnState | null): string {
  if (!turn) {
    return "";
  }

  return turn.segments
    .filter((segment) => segment.kind === "text")
    .map((segment) => segment.text)
    .join("")
    .trim();
}
