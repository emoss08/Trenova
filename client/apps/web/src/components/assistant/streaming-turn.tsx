import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { EASE_SETTLE } from "@/lib/motion";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { CircleAlertIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { Fragment, useMemo } from "react";
import {
  ArtifactChips,
  AssistantProse,
  AssistantTurn,
  DecisionNote,
  ReasoningDisclosure,
  RefusalNotice,
  UserTurn,
} from "./message-items";
import { currentActivity, segmentStep, stepsFromSegments, type ToolStep } from "./activity";
import { askRequestsFromSteps, type ThreadAskRequest } from "./ask-requests";
import { ChoicePrompt } from "./choice-prompt";
import { ReportRunCard } from "./report-run-card";
import { reportRunOrigins, type ThreadReportRun } from "./report-runs";
import { ToolActivity } from "./tool-activity";
import { isTurnActive, type TurnState } from "./turn-stream";
import type { AssistantArtifactEvent } from "@/types/assistant";
import { SourcesFooter } from "./web-citations";
import { webSourcesOf } from "./web-sources";
import { thinkingPose } from "./voice/desk-pose";
import { DeskThinking, useThinkingPresence } from "./voice/desk-thinking";

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
  onOpenArtifact,
}: {
  /**
   * Answering a question the live turn asked, before the turn is saved.
   * Absent where the conversation can no longer continue.
   */
  onAnswer?: (value: string) => void;
  /** Opens an artifact the turn produced; absent where there is no pane to open it in. */
  onOpenArtifact?: (id: string) => void;
  turn: TurnState;
  onRetry?: () => void;
  onDismiss: () => void;
}) {
  const t = useT();

  const steps = useMemo(() => stepsFromSegments(turn.segments), [turn.segments]);
  const outputs = useMemo(
    () => outputsByStep(steps, onOpenArtifact ? turn.artifacts : []),
    [steps, onOpenArtifact, turn.artifacts],
  );
  const active = isTurnActive(turn);
  const answer = useMemo(
    () => turn.segments.map((segment) => (segment.kind === "text" ? segment.text : "")).join("\n"),
    [turn.segments],
  );
  const sources = useMemo(() => webSourcesOf(steps, answer), [steps, answer]);

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
                    sources={sources}
                  />
                </div>
              );
            }
            return (
              <Fragment key={`tools-${group.steps[0].id}`}>
                <ToolActivity steps={group.steps} live />
                <StepOutputs
                  outputs={outputs.forSteps(group.steps)}
                  onAnswer={onAnswer}
                  onOpenArtifact={onOpenArtifact}
                />
              </Fragment>
            );
          })}
          {!active && answer !== "" && <SourcesFooter sources={sources} />}
          <StepOutputs
            outputs={outputs.unplaced}
            onAnswer={onAnswer}
            onOpenArtifact={onOpenArtifact}
          />
          <WorkingLine turn={turn} steps={steps} active={active} />
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

type StepOutput = {
  artifacts: AssistantArtifactEvent[];
  runs: ThreadReportRun[];
  asks: ThreadAskRequest[];
};

const NO_OUTPUT: StepOutput = { artifacts: [], runs: [], asks: [] };

/**
 * What each step produced besides its result: the artifacts it published,
 * the report runs it started and the questions it asked. The saved thread
 * shows these under the step that made them, before the words written after
 * it; the live turn places them the same way, so nothing moves when the reply
 * is saved. An artifact whose step this reader never saw is kept apart and
 * shown after the steps rather than dropped.
 */
function outputsByStep(
  steps: readonly ToolStep[],
  artifacts: readonly AssistantArtifactEvent[],
): { forSteps: (group: readonly ToolStep[]) => StepOutput; unplaced: StepOutput } {
  const known = new Set(steps.map((step) => step.id));
  const runs = reportRunOrigins(steps);
  const asks = new Map(askRequestsFromSteps(steps).map((ask) => [ask.callId, ask]));
  const artifactsByStep = new Map<string, AssistantArtifactEvent[]>();
  const unplaced: AssistantArtifactEvent[] = [];

  for (const artifact of artifacts) {
    if (artifact.sourceToolCallId !== "" && known.has(artifact.sourceToolCallId)) {
      artifactsByStep.set(artifact.sourceToolCallId, [
        ...(artifactsByStep.get(artifact.sourceToolCallId) ?? []),
        artifact,
      ]);
    } else {
      unplaced.push(artifact);
    }
  }

  return {
    forSteps: (group) => {
      const output: StepOutput = { artifacts: [], runs: [], asks: [] };
      for (const step of group) {
        output.artifacts.push(...(artifactsByStep.get(step.id) ?? []));
        output.runs.push(...(runs.get(step.id) ?? []));
        const ask = asks.get(step.id);
        if (ask) {
          output.asks.push(ask);
        }
      }
      return output;
    },
    unplaced: unplaced.length > 0 ? { ...NO_OUTPUT, artifacts: unplaced } : NO_OUTPUT,
  };
}

/** A step's artifacts, report runs and questions, in the order the saved thread shows them. */
function StepOutputs({
  outputs,
  onAnswer,
  onOpenArtifact,
}: {
  outputs: StepOutput;
  onAnswer?: (value: string) => void;
  onOpenArtifact?: (id: string) => void;
}) {
  return (
    <>
      {outputs.artifacts.length > 0 && onOpenArtifact && (
        <ArtifactChips artifacts={outputs.artifacts} onOpen={onOpenArtifact} />
      )}
      {outputs.runs.map((run) => (
        <ReportRunCard key={run.runId} run={run} />
      ))}
      {outputs.asks.map((ask) => (
        <ChoicePrompt key={ask.callId} request={ask} answered={false} onAnswer={onAnswer} />
      ))}
    </>
  );
}

/**
 * What the agent is doing right now, in words, with how long it has been at
 * it, so a pause never reads as a hang.
 *
 * The words change only when the work does — a step starting, a step
 * finishing, the model turning to write — and each change rises into place
 * while the last one lifts away, so the line moves because something
 * happened and at no other time. The desk beside them is the one scene the
 * product draws, and its pose follows the same moment: dots on the screen
 * while the model thinks, hands on the keys while a tool runs, lines written
 * onto the screen while the answer arrives.
 *
 * When the turn ends the words go at once and the desk settles for a beat,
 * then the line is gone.
 */
function WorkingLine({
  turn,
  steps,
  active,
}: {
  turn: TurnState;
  steps: ToolStep[];
  active: boolean;
}) {
  const reduceMotion = useReducedMotion() ?? false;
  const presence = useThinkingPresence(active, !reduceMotion);

  if (presence === "gone") {
    return null;
  }

  return (
    <div className="text-foreground-muted flex h-6 min-w-0 items-center gap-2 text-xs">
      <DeskThinking pose={thinkingPose(turn, steps)} className="-ml-px" />
      {active && <WorkingWords turn={turn} steps={steps} reduceMotion={reduceMotion} />}
    </div>
  );
}

/** The sentence and the tally, announced politely as the sentence changes. */
function WorkingWords({
  turn,
  steps,
  reduceMotion,
}: {
  turn: TurnState;
  steps: ToolStep[];
  reduceMotion: boolean;
}) {
  const t = useT();
  const now = useNowSeconds(1000);
  const elapsed = Math.max(0, now - Math.floor(turn.startedAt / 1000));
  const label = workingLabel(turn, steps, t);
  const travel = reduceMotion ? 0 : 6;

  return (
    <>
      <span aria-live="polite" className="relative flex min-w-0 flex-1 overflow-hidden">
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
        {workingTally(steps, elapsed, t)}
      </span>
    </>
  );
}

/**
 * The quiet count beside the words: the steps taken so far, how many are
 * running at once when there is more than one, and how long it has been.
 */
export function workingTally(steps: readonly ToolStep[], elapsed: number, t: TranslateFn): string {
  let taken = 0;
  let running = 0;
  for (const step of steps) {
    if (step.status === "running") {
      running += 1;
    } else {
      taken += 1;
    }
  }
  const parts: string[] = [];
  if (taken > 0) {
    parts.push(t("{0, plural, one {# step} other {# steps}}", taken));
  }
  if (running > 1) {
    parts.push(t("{0} running", running));
  }
  parts.push(formatWorkDuration(elapsed));

  return parts.join(" · ");
}

/**
 * The words for the moment: the guard checking, a retry waiting, the step
 * under way, the thinking or the writing. A step that has just finished is
 * not the moment any more; the model is deciding what to do with it.
 *
 * The plain moments — getting going, thinking, writing — each have a few
 * ways of being said, and a turn keeps one of them from start to finish, so
 * two replies do not read as the same machine while one reply never
 * changes its words for no reason.
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
  const variant = turnVariant(turn);
  const last = turn.segments.at(-1);
  if (last?.kind === "reasoning" && !last.closed) {
    return variant % 2 === 1 ? t("Thinking it through…") : t("Thinking…");
  }
  if (last?.kind === "text" && !last.closed) {
    return writingLine(variant, t);
  }
  if (last?.kind === "tool") {
    return t("Reading what came back…");
  }
  if (!last) {
    return openingLine(variant, t);
  }

  return t("Thinking…");
}

/** Which of a moment's ways of being said this turn keeps: fixed for the turn, different between turns. */
function turnVariant(turn: TurnState): number {
  return Math.abs(Math.floor(turn.startedAt / 1000));
}

function openingLine(variant: number, t: TranslateFn): string {
  switch (variant % 3) {
    case 1:
      return t("Pulling up a chair…");
    case 2:
      return t("Getting started…");
    default:
      return t("Thinking…");
  }
}

function writingLine(variant: number, t: TranslateFn): string {
  switch (variant % 3) {
    case 1:
      return t("Writing…");
    case 2:
      return t("Putting it into words…");
    default:
      return t("Writing the answer…");
  }
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
