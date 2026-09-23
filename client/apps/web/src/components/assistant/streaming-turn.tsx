import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { EASE_SETTLE } from "@/lib/motion";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { CircleAlertIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useMemo } from "react";
import {
  AssistantProse,
  AssistantTurn,
  DecisionNote,
  ReasoningDisclosure,
  RefusalNotice,
  UserTurn,
} from "./message-items";
import { currentActivity, segmentStep, stepsFromSegments, type ToolStep } from "./activity";
import { askRequestsFromSteps } from "./ask-requests";
import { ChoicePrompt } from "./choice-prompt";
import { ReportRunCard } from "./report-run-card";
import { reportRunsFromSteps } from "./report-runs";
import { ToolActivity } from "./tool-activity";
import { isTurnActive, type TurnState } from "./turn-stream";
import { WorkingDot } from "./voice/working-dot";

/**
 * The turn in progress, rendered straight from its state.
 *
 * Everything the reader sees here is provisional and is replaced by the saved
 * thread once the server confirms it; the point of showing it is that a
 * question about a shipment should start being answered within a second, not
 * after every lookup has finished.
 */
export function StreamingTurn({
  turn,
  onRetry,
  onDismiss,
  onAnswer,
}: {
  /** Answering a question the live turn asked, before the turn is saved. */
  onAnswer: (value: string) => void;
  turn: TurnState;
  onRetry?: () => void;
  onDismiss: () => void;
}) {
  const t = useT();

  const steps = useMemo(() => stepsFromSegments(turn.segments), [turn.segments]);
  const asks = askRequestsFromSteps(steps);
  const reportRuns = reportRunsFromSteps(steps);
  const active = isTurnActive(turn);

  const hasBody = turn.segments.length > 0;
  const showFrame = hasBody || active;

  return (
    <div className="flex flex-col gap-4">
      {turn.followUp ? (
        <DecisionNote content="" />
      ) : (
        <UserTurn
          content={turn.userContent}
          pageContext={turn.pageContext}
          attachments={turn.attachments}
          mentions={turn.mentions}
        />
      )}

      {turn.status === "refused" && turn.refusal && (
        <RefusalNotice message={turn.refusal.message} />
      )}

      {showFrame && turn.status !== "refused" && (
        <AssistantTurn working={active}>
          {groupSegments(turn).map((group, index) => {
            if (group.kind === "reasoning") {
              return (
                <ReasoningDisclosure
                  key={`reasoning-${index}`}
                  text={group.text}
                  streaming={!group.closed && turn.status === "streaming"}
                />
              );
            }
            if (group.kind === "text") {
              // The first words of an answer rise into place under the work
              // that led to them; after that the caret carries the motion.
              return (
                <div key={`text-${index}`} className="animate-rise">
                  <AssistantProse
                    content={group.text}
                    streaming={!group.closed && turn.status === "streaming"}
                  />
                </div>
              );
            }
            return <ToolActivity key={`tools-${group.steps[0].id}`} steps={group.steps} live />;
          })}
          {reportRuns.map((run) => (
            <ReportRunCard key={run.runId} run={run} />
          ))}
          {asks.map((ask) => (
            <ChoicePrompt key={ask.callId} request={ask} answered={false} onAnswer={onAnswer} />
          ))}
          {active && <WorkingLine turn={turn} steps={steps} />}
        </AssistantTurn>
      )}

      {turn.status === "error" && (
        <AssistantTurn>
          <Alert variant="destructive" size="sm">
            <CircleAlertIcon className="size-4" />
            <AlertTitle>{t("The assistant could not finish")}</AlertTitle>
            <AlertDescription className="flex flex-col gap-2">
              <p>{turn.error}</p>
              <div className="flex gap-2">
                {onRetry && (
                  <Button size="xs" variant="outline" onClick={onRetry}>
                    {t("Try again")}
                  </Button>
                )}
                <Button size="xs" variant="ghost" onClick={onDismiss}>
                  {t("Dismiss")}
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        </AssistantTurn>
      )}
    </div>
  );
}

/**
 * What the agent is doing right now, in words, with how long it has been at
 * it, so a pause never reads as a hang.
 *
 * The words change only when the work does — a step starting, a step
 * finishing, the model turning to write — and each change rises into place
 * while the last one lifts away, so the line moves because something
 * happened and at no other time. The dot is the one loop the product allows,
 * and it stops the moment the reply does.
 */
function WorkingLine({ turn, steps }: { turn: TurnState; steps: ToolStep[] }) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const now = useNowSeconds(1000);
  const elapsed = Math.max(0, now - Math.floor(turn.startedAt / 1000));
  const label = workingLabel(turn, steps, t);
  const taken = steps.filter((step) => step.status !== "running").length;
  const travel = reduceMotion ? 0 : 6;

  return (
    <div
      role="status"
      aria-live="polite"
      className="text-foreground-muted flex h-6 min-w-0 items-center gap-2 text-xs"
    >
      <WorkingDot working className="mx-0.75" />
      <span className="relative flex min-w-0 flex-1 overflow-hidden">
        <AnimatePresence mode="popLayout" initial={false}>
          <m.span
            key={label}
            initial={{ opacity: 0, y: travel }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -travel }}
            transition={{ duration: 0.24, ease: EASE_SETTLE }}
            className="min-w-0 truncate"
          >
            {label}
          </m.span>
        </AnimatePresence>
      </span>
      <span aria-hidden className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
        {taken > 0
          ? `${t("{0, plural, one {# step} other {# steps}}", taken)} · ${formatWorkDuration(elapsed)}`
          : formatWorkDuration(elapsed)}
      </span>
    </div>
  );
}

/**
 * The words for the moment: the guard checking, a retry waiting, the step
 * under way, the thinking or the writing. A step that has just finished is
 * not the moment any more; the model is deciding what to do with it.
 */
export function workingLabel(turn: TurnState, steps: readonly ToolStep[], t: TranslateFn): string {
  if (turn.status === "guarding") {
    return t("Checking the question…");
  }
  if (turn.retrying) {
    return retryingLine(turn.retrying, t);
  }
  if (steps.some((step) => step.status === "running")) {
    return currentActivity(steps, t)?.phrase ?? t("Working…");
  }
  const last = turn.segments.at(-1);
  if (last?.kind === "reasoning" && !last.closed) {
    return t("Thinking…");
  }
  if (last?.kind === "text" && !last.closed) {
    return t("Writing the answer…");
  }
  if (last?.kind === "tool") {
    return t("Reading what came back…");
  }

  return t("Thinking…");
}

type SegmentGroup =
  | { kind: "text"; text: string; closed: boolean }
  | { kind: "reasoning"; text: string; closed: boolean }
  | { kind: "tools"; steps: ToolStep[] };

/** Consecutive tool calls share one activity list rather than a list each. */
function groupSegments(turn: TurnState): SegmentGroup[] {
  const groups: SegmentGroup[] = [];
  for (const segment of turn.segments) {
    if (segment.kind === "text") {
      groups.push({ kind: "text", text: segment.text, closed: segment.closed });
      continue;
    }
    if (segment.kind === "reasoning") {
      groups.push({ kind: "reasoning", text: segment.text, closed: segment.closed });
      continue;
    }
    const step = segmentStep(segment);
    const last = groups.at(-1);
    if (last && last.kind === "tools") {
      last.steps.push(step);
    } else {
      groups.push({ kind: "tools", steps: [step] });
    }
  }

  return groups;
}

/**
 * What a retry means to the reader. A restart withdrew the words they were
 * reading; a busy provider is a wait, and the line says how long so the
 * silence is not mistaken for a hang.
 */
export function retryingLine(retrying: NonNullable<TurnState["retrying"]>, t: TranslateFn): string {
  if (retrying.kind === "busy") {
    const who = retrying.provider !== "" ? retrying.provider : t("The model");
    return retrying.waitSeconds > 0
      ? t(
          "{0} is busy. Trying again in {1}s (attempt {2})…",
          who,
          retrying.waitSeconds,
          retrying.attempt + 1,
        )
      : t("{0} is busy. Trying again (attempt {1})…", who, retrying.attempt + 1);
  }

  return retrying.provider !== ""
    ? t("The model stopped partway. Starting over on {0}…", retrying.provider)
    : t("The model stopped partway. Starting over…");
}
