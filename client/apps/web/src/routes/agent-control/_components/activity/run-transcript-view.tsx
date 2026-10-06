import { DeskProse } from "@/components/desk-chat/conversation/desk-turns";
import { stepsFromExchanges } from "@/components/assistant/activity";
import { ToolActivity } from "@/components/assistant/tool-activity";
import type { AgentRunTranscript } from "@/lib/graphql/agent-activity-tables";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useMemo } from "react";
import { ReasoningDisclosure } from "./reasoning-disclosure";
import { transcriptBlocks, type TranscriptBlock } from "./run-transcript";

/**
 * A finished run read back: each of the model's turns with what it thought,
 * the tools it called and what they returned, and what it said, drawn with
 * the conversation's own pieces and nothing to act on.
 */
export function RunTranscriptView({
  runId,
  transcript,
}: {
  runId: string;
  transcript: AgentRunTranscript;
}) {
  const t = useT();
  const blocks = useMemo(() => transcriptBlocks(runId, transcript), [runId, transcript]);

  if (blocks.length === 0) {
    return <p className="text-foreground-muted text-sm">{t("The run kept no messages.")}</p>;
  }

  return (
    <ol className="flex min-w-0 flex-col gap-4">
      {blocks.map((block) => (
        <li key={block.key} className="min-w-0">
          <TranscriptBlockView block={block} />
        </li>
      ))}
    </ol>
  );
}

function TranscriptBlockView({ block }: { block: TranscriptBlock }) {
  const t = useT();

  switch (block.kind) {
    case "gap":
      return (
        <p
          role="note"
          className="text-foreground-subtle border-border border-y border-dashed py-1.5 text-center text-xs"
        >
          {t(
            "{0, plural, one {# message from the middle of the run was left out to keep the record bounded} other {# messages from the middle of the run were left out to keep the record bounded}}",
            block.count,
          )}
        </p>
      );
    case "note":
      return (
        <div className="bg-sunken flex min-w-0 flex-col gap-1 rounded-surface px-3 py-2 text-sm">
          <span className="text-foreground-subtle text-xs">
            {block.message.createdAt > 0
              ? t("Task · {0}", formatUnixDateTimeMedium(block.message.createdAt))
              : t("Task")}
          </span>
          <p className="min-w-0 break-words whitespace-pre-wrap">{block.message.content}</p>
        </div>
      );
    default:
      return <TranscriptTurn block={block} />;
  }
}

function TranscriptTurn({ block }: { block: Extract<TranscriptBlock, { kind: "turn" }> }) {
  const t = useT();
  const { message, tools } = block.entry;
  const steps = useMemo(
    () => stepsFromExchanges(tools, message.createdAt),
    [tools, message.createdAt],
  );

  return (
    <article className="flex min-w-0 flex-col gap-2">
      {message.createdAt > 0 && (
        <span className="text-foreground-subtle text-xs">
          {formatUnixDateTimeMedium(message.createdAt)}
        </span>
      )}
      {message.reasoning?.text ? <ReasoningDisclosure text={message.reasoning.text} /> : null}
      {steps.length > 0 && <ToolActivity steps={steps} />}
      {message.content !== "" && <DeskProse content={message.content} />}
      {block.clipped && (
        <p className="text-foreground-subtle text-xs">
          {t("Part of this step was too large to keep, so only what was called is shown.")}
        </p>
      )}
    </article>
  );
}
