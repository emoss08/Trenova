import type { ToolStep } from "@/components/assistant/activity";
import { askRequestsFrom } from "@/components/assistant/ask-requests";
import { ChoicePrompt } from "@/components/assistant/choice-prompt";
import { ReportRunCard } from "@/components/assistant/report-run-card";
import { reportRunsFrom } from "@/components/assistant/report-runs";
import type { ThreadEntry } from "@/components/assistant/thread-view";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { withArtifactRefs } from "@/lib/artifact-ref";
import { AiMarkdown, StreamingAiMarkdown } from "@/components/elements/ai-markdown";
import type {
  AssistantArtifact,
  AssistantEntityRef,
  AssistantMessageAttachment,
} from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo, type ReactNode } from "react";
import { DeskMentionText } from "../composer/desk-mentions";
import { DeskMessageAttachments } from "../composer/desk-uploads";
import { citeSteps, withCitations } from "./citations";
import { useCitationOverrides } from "./desk-citations";
import { DeskMessageActions } from "./desk-message-actions";

/** Where a row sits in the conversation's three columns: time, words, margin. */
export function DeskRow({
  kind,
  first = false,
  time,
  children,
}: {
  kind: "question" | "reply" | "continued" | "event";
  first?: boolean;
  time?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "dk-r",
        (kind === "reply" || kind === "continued") && "dk-a",
        kind === "continued" && "dk-cont",
        kind === "event" && "dk-ev",
        first && "dk-first",
      )}
    >
      <div className="dk-g">{time && <span className="dk-time">{time}</span>}</div>
      <div className="dk-c">{children}</div>
      <div className="dk-m" />
    </div>
  );
}

/** How a question is set: a heading when it is short, a quiet block when it runs long. */
export function questionSize(text: string): "short" | "mid" | "long" {
  if (text.length > 90) {
    return "long";
  }
  return text.length > 48 ? "mid" : "short";
}

/** What the person asked, as the heading of the exchange it opens. */
export function DeskQuestion({
  text,
  muted = false,
  tag,
  mentions,
  attachments,
}: {
  text: string;
  muted?: boolean;
  tag?: string;
  /** The records the question named with @, drawn as chips where they were named. */
  mentions?: readonly AssistantEntityRef[] | null;
  /** The files the question carried. */
  attachments?: readonly AssistantMessageAttachment[] | null;
}) {
  const size = questionSize(text);

  // The page it was asked from still travels with the message; the design
  // no longer says so under the question.
  return (
    <div className="dk-qb">
      <div
        className={cn(
          "dk-q",
          size === "mid" && "dk-mid",
          size === "long" && "dk-long",
          muted && "dk-ec-q dk-muted",
        )}
      >
        {mentions && mentions.length > 0 ? (
          <DeskMentionText text={text} mentions={mentions} />
        ) : (
          text
        )}
        {tag && <span className="dk-ec-qtag">{tag}</span>}
      </div>
      {attachments && attachments.length > 0 && (
        <DeskMessageAttachments attachments={attachments} />
      )}
    </div>
  );
}

/**
 * An artifact a reply names inside its sentence, as the same dark-blue badge
 * the list under a reply uses, sized to sit in a line of text. Its words are
 * the reply's, so the sentence still reads; the icon says what kind it is.
 */
export function DeskInlineArtifact({
  artifact,
  active,
  onOpen,
  children,
}: {
  artifact: Pick<AssistantArtifact, "id" | "kind" | "title">;
  active: boolean;
  onOpen: (id: string) => void;
  children: ReactNode;
}) {
  const t = useT();
  return (
    <button
      type="button"
      className={cn("dk-abadge dk-inline", active && "dk-on")}
      aria-label={t("Open {0}", artifact.title)}
      onClick={() => onOpen(artifact.id)}
    >
      <ArtifactKindIcon kind={artifact.kind} className="dk-abadge-i" />
      <span>{children}</span>
    </button>
  );
}

const CUT_NOTE = /\s*_This reply was cut off before it finished\.[^_]*_\s*$/u;

/** A cut-off reply's words, without the note the server adds to say it was cut. */
export function withoutCutNote(content: string): string {
  return content.replace(CUT_NOTE, "");
}

/**
 * A saved reply: its words, what it produced, and the actions under it.
 * Memoized, so a reply arriving below does not set every earlier one again.
 */
export const DeskReply = memo(function DeskReply({
  entry,
  steps,
  threadArtifacts,
  artifacts,
  latestUserSequence,
  chapter,
  onTogglePin,
  onAnswer,
  onOpenArtifact,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
  /** Every step the reply has taken so far, from its question on; the numbers cite them. */
  steps: readonly ToolStep[];
  /** The conversation's artifacts, so a cited step can offer the one it made. */
  threadArtifacts: readonly AssistantArtifact[];
  artifacts: readonly AssistantArtifact[];
  latestUserSequence: number;
  chapter: number;
  /** Pins or unpins the reply as a chapter, by its message id. */
  onTogglePin: (messageId: string) => void;
  onAnswer?: (value: string) => void;
  onOpenArtifact: (id: string) => void;
}) {
  const { message, tools } = entry;
  const asks = askRequestsFrom(tools);
  const reportRuns = reportRunsFrom(tools);
  // A reply that broke off carries the server's note saying so; the card
  // under it says it instead.
  const content = message.truncated ? withoutCutNote(message.content) : message.content;
  const citations = useMemo(() => citeSteps(content, steps), [content, steps]);
  // Every artifact the reply made is opened from its words: one it named
  // where it named it, any other at the end of its last sentence.
  const cited = useMemo(
    () => withArtifactRefs(withCitations(content, citations), artifacts),
    [artifacts, citations, content],
  );
  const overrides = useCitationOverrides(citations, threadArtifacts, onOpenArtifact);

  return (
    <>
      {cited !== "" && (
        <div className={cn("dk-prose", entry.message.truncated && "dk-cut")}>
          <AiMarkdown content={cited} className="dk-md" overrides={overrides} deskSubset />
        </div>
      )}
      {reportRuns.map((run) => (
        <div key={run.runId} className="dk-extra">
          <ReportRunCard run={run} />
        </div>
      ))}
      {asks.map((ask) => (
        <div key={ask.callId} className="dk-extra">
          <ChoicePrompt
            request={ask}
            answered={latestUserSequence > ask.sequence}
            onAnswer={onAnswer}
          />
        </div>
      ))}
      {message.content !== "" && (
        <DeskMessageActions
          text={content}
          chapter={chapter}
          onTogglePin={() => onTogglePin(message.id)}
        />
      )}
    </>
  );
});

/** The reply being written, word by word; nothing shows until its first words arrive. */
export function DeskStreamingReply({ text }: { text: string }) {
  return (
    <div className="dk-prose dk-streaming">
      <StreamingAiMarkdown
        content={text}
        className="dk-md"
        wordClassName="dk-w"
        caretClassName="dk-caret"
        deskSubset
      />
    </div>
  );
}
