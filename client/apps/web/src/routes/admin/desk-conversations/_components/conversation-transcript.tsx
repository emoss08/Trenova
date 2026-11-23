import { stepsFromExchanges } from "@/components/assistant/activity";
import { ReasoningDisclosure } from "@/components/assistant/reasoning-disclosure";
import { ToolActivity } from "@/components/assistant/tool-activity";
import { DeskProse } from "@/components/desk-chat/conversation/desk-turns";
import type { AgentRunTranscript } from "@/lib/graphql/agent-activity-tables";
import {
  transcriptBlocks,
  type TranscriptBlock,
} from "@/routes/agent-control/_components/activity/run-transcript";
import type { AssistantMessage } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useMemo } from "react";

type ConversationTranscriptProps = {
  conversationId: string;
  transcript: AgentRunTranscript | null;
  ownerName: string;
  agentName: string;
};

/**
 * Another person's conversation read back as they saw it: what they asked,
 * each of the agent's turns with what it thought, the tools it called and
 * what they returned, and what it said. Built from the same blocks as an
 * agent run's transcript, with nothing to act on.
 */
export function ConversationTranscript({
  conversationId,
  transcript,
  ownerName,
  agentName,
}: ConversationTranscriptProps) {
  const t = useT();
  const blocks = useMemo(
    () => (transcript ? transcriptBlocks(conversationId, transcript) : []),
    [conversationId, transcript],
  );

  if (blocks.length === 0) {
    return (
      <p className="text-foreground-muted text-sm">{t("Nothing was said in this conversation.")}</p>
    );
  }

  return (
    <ol className="flex min-w-0 flex-col gap-4">
      {blocks.map((block) => (
        <li key={block.key} className="min-w-0">
          <TranscriptBlockView block={block} ownerName={ownerName} agentName={agentName} />
        </li>
      ))}
    </ol>
  );
}

function noteHeading(t: TranslateFn, message: AssistantMessage, ownerName: string): string {
  switch (message.kind) {
    case "DecisionNote":
      return t("Decision");
    case "Compaction":
      return t("Conversation compacted");
    case "HandoffBrief":
      return t("Handed over from another agent");
    case "WorldChange":
      return t("Records changed while the agent worked");
    case "WaitNote":
      return t("Picked up after a wait");
    case "Schedule":
      return t("Scheduled request");
    case "Steer":
      return t("{0}, while it worked", ownerName);
    default:
      return ownerName;
  }
}

function TranscriptBlockView({
  block,
  ownerName,
  agentName,
}: {
  block: TranscriptBlock;
  ownerName: string;
  agentName: string;
}) {
  const t = useT();

  switch (block.kind) {
    case "gap":
      return (
        <p
          role="note"
          className="text-foreground-subtle border-border border-y border-dashed py-1.5 text-center text-xs"
        >
          {t(
            "{0, plural, one {# message from the middle of this conversation is left out here; download the transcript to read all of it} other {# messages from the middle of this conversation are left out here; download the transcript to read all of them}}",
            block.count,
          )}
        </p>
      );
    case "note":
      return (
        <div className="bg-sunken rounded-surface flex min-w-0 flex-col gap-1 px-3 py-2 text-sm">
          <span className="text-foreground-subtle text-xs">
            {block.message.createdAt > 0
              ? t(
                  "{0} · {1}",
                  noteHeading(t, block.message, ownerName),
                  formatUnixDateTimeMedium(block.message.createdAt),
                )
              : noteHeading(t, block.message, ownerName)}
          </span>
          <p className="min-w-0 break-words whitespace-pre-wrap">{block.message.content}</p>
        </div>
      );
    default:
      return <TranscriptTurn block={block} agentName={agentName} />;
  }
}

function TranscriptTurn({
  block,
  agentName,
}: {
  block: Extract<TranscriptBlock, { kind: "turn" }>;
  agentName: string;
}) {
  const t = useT();
  const { message, tools } = block.entry;
  const steps = useMemo(
    () => stepsFromExchanges(tools, message.createdAt),
    [tools, message.createdAt],
  );

  return (
    <article className="flex min-w-0 flex-col gap-2">
      <span className="text-foreground-subtle text-xs">
        {message.createdAt > 0
          ? t("{0} · {1}", agentName, formatUnixDateTimeMedium(message.createdAt))
          : agentName}
      </span>
      {message.reasoning?.text ? <ReasoningDisclosure text={message.reasoning.text} /> : null}
      {steps.length > 0 && <ToolActivity steps={steps} />}
      {message.content !== "" && <DeskProse content={message.content} />}
      {block.clipped && (
        <p className="text-foreground-subtle text-xs">
          {t(
            "Part of this turn is too large to show here; download the transcript to read it whole.",
          )}
        </p>
      )}
    </article>
  );
}
