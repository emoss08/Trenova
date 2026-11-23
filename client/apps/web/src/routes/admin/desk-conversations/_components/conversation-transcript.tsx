import { stepsFromExchanges, workSummary, type ToolStep } from "@/components/assistant/activity";
import { ReasoningDisclosure } from "@/components/assistant/reasoning-disclosure";
import { ToolActivity } from "@/components/assistant/tool-activity";
import { DeskCompactMark } from "@/components/desk-chat/conversation/desk-compact-mark";
import { DeskQuestion, DeskReply, DeskRow } from "@/components/desk-chat/conversation/desk-turns";
import { turnTime } from "@/components/desk-chat/turn-time";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import {
  conversationItems,
  type ConversationItem,
  type ConversationReply,
} from "@/lib/desk-conversation-transcript";
import type { AgentRunTranscript } from "@/lib/graphql/agent-activity-tables";
import { withoutRecordIds } from "@/lib/record-ids";
import type { AssistantArtifact, AssistantMessage } from "@/types/assistant";
import { AlertCircleIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo, useState } from "react";

type ConversationTranscriptProps = {
  conversationId: string;
  transcript: AgentRunTranscript | null;
};

const NO_ARTIFACTS: readonly AssistantArtifact[] = [];
const NO_OP = () => {};

/*
The Desk's rows (DeskRow) are `display: contents`: their time, words and
margin cells sit in the columns of the grid around them. Here that grid is a
gutter for the turn's time and one reading column, with the margin column at
nothing; on a phone the gutter goes and the time with it, as in the Desk's
narrow room. The trailing 0px column keeps its gap, so the right padding is
that gap plus 4px.
*/
const TRANSCRIPT_GRID =
  "grid min-w-0 grid-cols-[0px_minmax(0,1fr)_0px] px-4 max-sm:[&_.dk-g]:hidden sm:grid-cols-[4rem_minmax(0,1fr)_0px] sm:gap-x-4 sm:pr-1 sm:pl-5";

/**
 * Another person's conversation read back as they saw it in the Desk: each
 * question set as the Desk sets it, the agent's answer as the Desk's prose,
 * and everything the agent did on the way folded into one line above the
 * answer. Nothing here can be sent, decided, undone or asked again; copying
 * an answer and reading it aloud are the only things a reply offers.
 */
export function ConversationTranscript({
  conversationId,
  transcript,
}: ConversationTranscriptProps) {
  const t = useT();
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const items = useMemo(
    () => (transcript ? conversationItems(conversationId, transcript) : []),
    [conversationId, transcript],
  );

  if (items.length === 0) {
    return (
      <p className="text-foreground-muted px-4 text-sm sm:px-5">
        {t("Nothing was said in this conversation.")}
      </p>
    );
  }

  return (
    <div className={cn("dk-chat", TRANSCRIPT_GRID)}>
      {items.map((item, index) => (
        <TranscriptItem key={item.key} item={item} first={index === 0} timezone={timezone} />
      ))}
    </div>
  );
}

function TranscriptItem({
  item,
  first,
  timezone,
}: {
  item: ConversationItem;
  first: boolean;
  timezone: string;
}) {
  const t = useT();

  switch (item.kind) {
    case "asked":
      return (
        <DeskRow kind="question" first={first}>
          <DeskQuestion
            text={item.message.content}
            muted={item.tag === "declined"}
            tag={
              item.tag === "steer"
                ? t("Sent while it was working")
                : item.tag === "declined"
                  ? t("Not sent to the agent")
                  : undefined
            }
          />
        </DeskRow>
      );
    case "gap":
      return (
        <DeskRow kind="event" first={first}>
          <p
            role="note"
            className="text-foreground-subtle border-border border-y border-dashed py-1.5 text-center text-xs"
          >
            {t(
              "{0, plural, one {# message from the middle of this conversation is left out here; download the transcript to read all of it} other {# messages from the middle of this conversation are left out here; download the transcript to read all of them}}",
              item.count,
            )}
          </p>
        </DeskRow>
      );
    case "event":
      return (
        <DeskRow kind="event" first={first}>
          {item.message.kind === "Compaction" ? (
            <DeskCompactMark message={item.message} />
          ) : (
            <ConversationEvent message={item.message} />
          )}
        </DeskRow>
      );
    case "turn":
      return (
        <DeskRow
          kind="reply"
          first={first}
          time={item.at > 0 ? turnTime(item.at, timezone, t) : undefined}
        >
          <AgentTurn turn={item} />
        </DeskRow>
      );
  }
}

function eventHeading(t: TranslateFn, message: AssistantMessage): string {
  switch (message.kind) {
    case "Handoff":
    case "HandoffBrief":
      return t("Handed over from another agent");
    case "WorldChange":
      return t("Records changed while the agent worked");
    case "WaitNote":
      return t("Picked up after a wait");
    default:
      return t("Decision");
  }
}

/** Something that happened in the conversation that neither side said, set quietly between the turns. */
function ConversationEvent({ message }: { message: AssistantMessage }) {
  const t = useT();
  const content = withoutRecordIds(message.content).trim();

  return (
    <div className="flex min-w-0 flex-col gap-0.5 text-xs">
      <span className="text-foreground-subtle">{eventHeading(t, message)}</span>
      {content !== "" && (
        <p className="text-foreground-muted min-w-0 break-words whitespace-pre-wrap">{content}</p>
      )}
    </div>
  );
}

type WorkPart = {
  key: string;
  reasoning: string;
  note: string;
  steps: ToolStep[];
};

/** The steps of a turn in the order they were taken. */
function stepsInOrder(turn: Extract<ConversationItem, { kind: "turn" }>): ConversationReply[] {
  const all = turn.reply ? [...turn.work, turn.reply] : [...turn.work];
  return all.sort((a, b) => a.message.sequence - b.message.sequence);
}

function AgentTurn({ turn }: { turn: Extract<ConversationItem, { kind: "turn" }> }) {
  const t = useT();
  const ordered = useMemo(() => stepsInOrder(turn), [turn]);
  const parts = useMemo(
    () =>
      ordered.map((entry): WorkPart => ({
        key: entry.message.id,
        reasoning: entry.message.reasoning?.text ?? "",
        note: entry === turn.reply ? "" : withoutRecordIds(entry.message.content).trim(),
        steps: stepsFromExchanges(entry.tools, turn.askedAt),
      })),
    [ordered, turn.askedAt, turn.reply],
  );
  const steps = useMemo(() => parts.flatMap((part) => part.steps), [parts]);
  const reply = useMemo(() => (turn.reply ? { ...turn.reply, tools: [] } : null), [turn.reply]);

  return (
    <>
      <TurnWork parts={parts} steps={steps} />
      {reply && (
        <DeskReply
          entry={reply}
          steps={steps}
          threadArtifacts={NO_ARTIFACTS}
          closesTurn
          onOpenArtifact={NO_OP}
        />
      )}
      {turn.clipped && (
        <p className="text-foreground-subtle mt-2 text-xs">
          {t(
            "Part of this turn is too large to show here; download the transcript to read it whole.",
          )}
        </p>
      )}
    </>
  );
}

/**
 * Everything the agent did on the way to its answer, folded into one line the
 * way the Desk folds a reply's work ("Worked through 4 steps"): the notes it
 * wrote, what it thought and the tools it called, in order. A turn that only
 * thought shows the thought's own fold and nothing around it.
 */
function TurnWork({ parts, steps }: { parts: readonly WorkPart[]; steps: ToolStep[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const notes = parts.filter((part) => part.note !== "").length;
  const shown = parts.filter(
    (part) => part.note !== "" || part.reasoning !== "" || part.steps.length > 0,
  );

  if (shown.length === 0) {
    return null;
  }

  if (steps.length === 0 && notes === 0) {
    return (
      <div className="mb-2 flex min-w-0 flex-col gap-1">
        {shown.map((part) => (
          <ReasoningDisclosure key={part.key} text={part.reasoning} />
        ))}
      </div>
    );
  }

  const failed = steps.some((step) => step.status === "failed");
  const label =
    steps.length > 0
      ? workSummary(
          steps.map((step) => ({ ...step, durationSeconds: null })),
          t,
        )
      : t("{0, plural, one {# working note} other {# working notes}}", notes);

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="mb-2 min-w-0">
      <CollapsibleTrigger
        className={cn(
          "group/work ui-focus-ring rounded-control -mx-1.5 flex max-w-[calc(100%+0.75rem)] min-w-0 items-center gap-1.5 px-1.5 py-0.5 text-left text-xs transition-colors",
          "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
        )}
      >
        <ChevronRightIcon
          aria-hidden
          className={cn("size-3 shrink-0 transition-transform duration-200", open && "rotate-90")}
        />
        <span className="min-w-0 truncate">{label}</span>
        {failed && <AlertCircleIcon aria-hidden className="text-danger size-3 shrink-0" />}
      </CollapsibleTrigger>
      <CollapsibleContent className="ease-settle h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 data-ending-style:h-0 data-starting-style:h-0">
        <ol className="border-border-subtle mt-1 ml-1.25 flex min-w-0 flex-col gap-2 overflow-x-auto border-l pr-1.5 pl-3">
          {shown.map((part) => (
            <li key={part.key} className="flex min-w-0 flex-col gap-1">
              {part.reasoning !== "" && <ReasoningDisclosure text={part.reasoning} />}
              {part.note !== "" && (
                <AiMarkdown
                  content={part.note}
                  className="text-foreground-muted min-w-0 text-sm"
                  deskSubset
                />
              )}
              {part.steps.length > 0 && <ToolActivity steps={part.steps} />}
            </li>
          ))}
        </ol>
      </CollapsibleContent>
    </Collapsible>
  );
}
