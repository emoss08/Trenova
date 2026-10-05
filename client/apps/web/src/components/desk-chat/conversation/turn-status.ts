import { currentActivity, stepsFromSegments } from "@/components/assistant/activity";
import type { TurnState } from "@/components/assistant/turn-stream";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DeskComposerStatus } from "../composer/desk-composer";

const MEMORY_TOOLS: ReadonlySet<string> = new Set(["recall_memory", "remember"]);

/**
 * What the composer says while a reply is being made, the one place the Desk
 * shows the agent's work in progress: the question being checked, a model
 * being asked again, the step under way, or the answer being written.
 * Nothing once the turn has finished, failed or been refused.
 */
export function composerStatus(
  turn: TurnState | null,
  t: TranslateFn,
  onSwitchModel?: () => void,
): DeskComposerStatus | null {
  if (!turn || turn.status === "done" || turn.status === "error" || turn.status === "refused") {
    return null;
  }
  if (turn.status === "guarding") {
    return { text: t("Checking the question…"), pose: "check" };
  }
  if (turn.retrying) {
    const who = turn.retrying.provider !== "" ? turn.retrying.provider : t("The model");
    if (turn.retrying.kind !== "busy") {
      return { text: t("Starting the reply again…"), pose: "retry" };
    }
    const attempt = turn.retrying.attempt + 1;
    return {
      text: t("{0} is overloaded · retrying", who),
      pose: "retry",
      extra:
        turn.retrying.maxAttempts > 0
          ? t("Attempt {0} of {1}", attempt, turn.retrying.maxAttempts)
          : t("Attempt {0}", attempt),
      countdown: turn.retrying.waitSeconds > 0 ? turn.retrying.waitSeconds : undefined,
      action: onSwitchModel ? { label: t("Switch model"), onClick: onSwitchModel } : undefined,
    };
  }

  const last = turn.segments.at(-1);
  if (last?.kind === "text" && !last.closed && last.text.trim() !== "") {
    return { text: t("Writing the answer…"), pose: "work" };
  }

  const steps = stepsFromSegments(turn.segments);
  const activity = currentActivity(steps, t);
  if (activity && activity.state === "running") {
    // Reading or keeping a memory carries the memory mark, as the design draws it.
    const running = steps.filter((step) => step.status === "running").at(-1);
    const pose = running && MEMORY_TOOLS.has(running.name) ? "memory" : "work";
    return { text: `${activity.phrase}…`, pose };
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
