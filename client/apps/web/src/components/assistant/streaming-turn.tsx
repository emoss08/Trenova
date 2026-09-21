import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { CircleAlertIcon } from "lucide-react";
import {
  AssistantFrame,
  AssistantProse,
  ReasoningDisclosure,
  RefusalNotice,
  UserBubble,
} from "./message-items";
import { askRequestsFromSteps } from "./ask-requests";
import { ChoicePrompt } from "./choice-prompt";
import { ReportRunCard } from "./report-run-card";
import { reportRunsFromSteps } from "./report-runs";
import { ToolTimeline, type ToolStep } from "./tool-activity";
import { describeToolCall } from "./tool-presentation";
import type { TurnState } from "./turn-stream";

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

  const steps = turn.segments.flatMap((segment) =>
    segment.kind === "tool"
      ? [{ id: segment.callId, name: segment.name, content: segment.content }]
      : [],
  );
  const asks = askRequestsFromSteps(steps);
  const reportRuns = reportRunsFromSteps(steps);

  const hasBody = turn.segments.length > 0;
  const showFrame = hasBody || turn.status === "guarding" || turn.status === "working";

  return (
    <>
      <UserBubble content={turn.userContent} pageContext={turn.pageContext} />

      {turn.status === "refused" && turn.refusal && (
        <RefusalNotice message={turn.refusal.message} />
      )}

      {showFrame && turn.status !== "refused" && (
        <AssistantFrame>
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
              return (
                <AssistantProse
                  key={`text-${index}`}
                  content={group.text}
                  streaming={!group.closed && turn.status === "streaming"}
                />
              );
            }
            return <ToolTimeline key={`tools-${index}`} steps={group.steps} live />;
          })}
          {reportRuns.map((run) => (
            <ReportRunCard key={run.runId} run={run} />
          ))}
          {asks.map((ask) => (
            <ChoicePrompt key={ask.callId} request={ask} answered={false} onAnswer={onAnswer} />
          ))}
          <StatusLine turn={turn} />
        </AssistantFrame>
      )}

      {turn.status === "error" && (
        <AssistantFrame>
          <Alert variant="destructive" className="">
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
        </AssistantFrame>
      )}
    </>
  );
}

/**
 * What the assistant is doing right now, in words, so a pause never reads as
 * a hang: the guard is checking, a record is being read, or the model is
 * composing.
 */
function StatusLine({ turn }: { turn: TurnState }) {
  const t = useT();

  if (turn.status === "guarding") {
    return (
      <TextShimmer as="span" className="text-muted-foreground text-xs">
        {t("Checking the question…")}
      </TextShimmer>
    );
  }

  if (turn.status !== "working") {
    return null;
  }

  const running = turn.segments.find(
    (segment) => segment.kind === "tool" && segment.status === "running",
  );
  if (running && running.kind === "tool") {
    const description = describeToolCall(running.name, running.arguments);
    const label =
      description.subject !== ""
        ? `${description.title} · ${description.subject}…`
        : `${description.title}…`;
    return (
      <TextShimmer as="span" className="text-muted-foreground text-xs">
        {label}
      </TextShimmer>
    );
  }

  return (
    <TextShimmer as="span" className="text-muted-foreground text-xs">
      {t("Thinking…")}
    </TextShimmer>
  );
}

type SegmentGroup =
  | { kind: "text"; text: string; closed: boolean }
  | { kind: "reasoning"; text: string; closed: boolean }
  | { kind: "tools"; steps: ToolStep[] };

/** Consecutive tool calls share one timeline rather than a row each. */
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
    const step: ToolStep = {
      id: segment.callId,
      name: segment.name,
      arguments: segment.arguments,
      status: segment.status,
      content: segment.content,
    };
    const last = groups.at(-1);
    if (last && last.kind === "tools") {
      last.steps.push(step);
    } else {
      groups.push({ kind: "tools", steps: [step] });
    }
  }

  return groups;
}
