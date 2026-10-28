import type { ToolStep } from "@/components/assistant/activity";
import type { ThreadAsk } from "@/components/assistant/ask-requests";
import { ReportRunCard } from "@/components/assistant/report-run-card";
import { reportRunsFrom } from "@/components/assistant/report-runs";
import type { ThreadEntry } from "@/components/assistant/thread-view";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { withoutRecordIds } from "@/lib/record-ids";
import { AiMarkdown, StreamingAiMarkdown } from "@/components/elements/ai-markdown";
import type {
  AssistantArtifact,
  AssistantEntityRef,
  AssistantMessageAttachment,
} from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo, type ReactNode } from "react";
import { DeskMentionText } from "../composer/desk-mentions";
import { DeskMessageAttachments } from "../composer/desk-uploads";
import { DeskMemoryRecall, DeskMemorySavedList } from "../memory/desk-memory-notes";
import { citeSteps, withCitations } from "./citations";
import { DeskAskRecord, type AskAnswerSent } from "./desk-ask";
import { useCitationOverrides } from "./desk-citations";
import { DeskMessageActions } from "./desk-message-actions";
import { DeskStepFailures } from "./desk-tool-failures";
import { DeskWebCites, DeskWebSources } from "./desk-web";
import { deskWebSources, webQueryOf, withWebCites, type DeskWebSource } from "./web-cites";
import "../desk-frame.css";
import "../desk-chat.css";

/** Where a row sits in the conversation's three columns: time, words, margin. */
export function DeskRow({
  kind,
  first = false,
  time,
  className,
  children,
}: {
  kind: "question" | "reply" | "continued" | "event";
  first?: boolean;
  time?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "dk-r",
        className,
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

/**
 * The row a chapter's heading sits in, above the question that opens it. It
 * carries the turn it opens so the conversation can tell which chapter is in
 * view and scroll to one.
 */
export function DeskChapterRow({
  turnId,
  first = false,
  children,
}: {
  turnId: string;
  first?: boolean;
  children: ReactNode;
}) {
  return (
    <div className={cn("dk-r dk-chap", first && "dk-first")} data-chapter-turn={turnId}>
      <div className="dk-g" />
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
    <>
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
    </>
  );
}

/**
 * An artifact a reply names inside its sentence, as a link in the brand's blue
 * that wraps with the words around it and is lit while its artifact is open.
 * It reads as the artifact's own title wherever a reply names it, whatever
 * words the model wrote; the icon says what kind it is.
 */
export function DeskInlineArtifact({
  artifact,
  active,
  onOpen,
}: {
  artifact: Pick<AssistantArtifact, "id" | "kind" | "title">;
  active: boolean;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  return (
    <Button
      variant="bare"
      size="bare"
      className={cn("dk-alink", active && "dk-on")}
      aria-label={t("Open {0}", artifact.title)}
      onClick={() => onOpen(artifact.id)}
    >
      <ArtifactKindIcon kind={artifact.kind} className="dk-alink-i" />
      {artifact.title}
    </Button>
  );
}

const NO_IDS: readonly string[] = [];
const NO_ASKS: readonly ThreadAsk[] = [];

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
  asks = NO_ASKS,
  dockedAskId = null,
  sentAnswer = null,
  closesTurn = false,
  onStartChapter,
  onOpenArtifact,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
  /** Every step the reply has taken so far, from its question on; the numbers cite them. */
  steps: readonly ToolStep[];
  /** The conversation's artifacts, so a cited step can offer the one it made. */
  threadArtifacts: readonly AssistantArtifact[];
  /** The questions the reply asked, each with the message that answered it once one has. */
  asks?: readonly ThreadAsk[];
  /** The reply's question that waits above the composer, if one of its questions does. */
  dockedAskId?: string | null;
  /** The answer just sent to one of the reply's questions, before it is saved. */
  sentAnswer?: AskAnswerSent | null;
  /**
   * The reply is the last of its turn, so it says which of the turn's steps
   * did not go through, between its words and the actions under them.
   */
  closesTurn?: boolean;
  /** Starts a chapter where the reply's question was asked. Left out, the reply offers none. */
  onStartChapter?: () => void;
  onOpenArtifact: (id: string) => void;
}) {
  const { message, tools } = entry;
  const reportRuns = reportRunsFrom(tools);
  // A reply that broke off carries the server's note saying so; the card
  // under it says it instead.
  // Record ids are the agent's; one that slips into a reply is kept off the page.
  const content = withoutRecordIds(
    message.truncated ? withoutCutNote(message.content) : message.content,
  );
  const citations = useMemo(() => citeSteps(content, steps), [content, steps]);
  // The pages the turn found on the web so far; the reply cites them by
  // number, and the one that closes the turn lists them under it.
  const webSources = useMemo(() => deskWebSources(steps), [steps]);
  const cited = useMemo(
    () => withCitations(withWebCites(content, webSources), citations),
    [citations, content, webSources],
  );
  const overrides = useCitationOverrides(citations, threadArtifacts, onOpenArtifact);
  const usedMemories = message.usedMemoryIds ?? NO_IDS;
  const savedMemories = useMemo(
    () => (message.savedMemories ?? []).map((saved) => saved.id),
    [message.savedMemories],
  );

  return (
    <>
      {usedMemories.length > 0 && <DeskMemoryRecall ids={usedMemories} served={message.memories} />}
      {cited !== "" && (
        <div className={cn("dk-prose", entry.message.truncated && "dk-cut")}>
          <DeskWebCites sources={webSources}>
            <AiMarkdown content={cited} className="dk-md" overrides={overrides} deskSubset />
          </DeskWebCites>
        </div>
      )}
      {closesTurn && <DeskWebSources sources={webSources} query={webQueryOf(steps)} />}
      {reportRuns.map((run) => (
        <div key={run.runId} className="dk-extra">
          <ReportRunCard run={run} />
        </div>
      ))}
      {asks.map((ask) => (
        <div key={ask.request.callId} className="dk-extra">
          <DeskAskRecord
            ask={ask}
            docked={ask.request.callId === dockedAskId}
            sent={sentAnswer?.callId === ask.request.callId ? sentAnswer : null}
          />
        </div>
      ))}
      {closesTurn && (
        <div className="dk-ec-after">
          <DeskStepFailures steps={steps} />
        </div>
      )}
      {savedMemories.length > 0 && (
        <DeskMemorySavedList ids={savedMemories} served={message.memories} />
      )}
      {message.content !== "" && (
        <DeskMessageActions text={content} onStartChapter={onStartChapter} />
      )}
    </>
  );
});

/**
 * The reply being written, word by word; nothing shows until its first words
 * arrive. Each word fades in as the design's do, with no caret after them.
 * The memories the turn has used so far sit above it.
 */
export function DeskStreamingReply({
  text,
  usedMemoryIds = NO_IDS,
  webSources = NO_SOURCES,
}: {
  text: string;
  usedMemoryIds?: readonly string[];
  /** The pages the turn has found so far, so citations draw as the words reach them. */
  webSources?: readonly DeskWebSource[];
}) {
  const content = useMemo(
    () => withWebCites(withoutRecordIds(text), webSources),
    [text, webSources],
  );

  return (
    <>
      {usedMemoryIds.length > 0 && <DeskMemoryRecall ids={usedMemoryIds} />}
      {content !== "" && (
        <div className="dk-prose dk-streaming">
          <DeskWebCites sources={webSources} live>
            <StreamingAiMarkdown
              content={content}
              className="dk-md"
              wordClassName="dk-w"
              deskSubset
            />
          </DeskWebCites>
        </div>
      )}
    </>
  );
}

const NO_SOURCES: readonly DeskWebSource[] = [];

/**
 * A reply's words away from a conversation: a quick answer in the command
 * palette, or a step of a run read back. The Desk's prose and markdown, word
 * by word while it streams; no citations, actions or artifacts.
 */
export function DeskProse({
  content,
  streaming = false,
}: {
  content: string;
  streaming?: boolean;
}) {
  const text = useMemo(() => withoutRecordIds(content), [content]);

  return (
    <div className={cn("dk-chat dk-prose", streaming && "dk-streaming")}>
      {streaming ? (
        <StreamingAiMarkdown content={text} className="dk-md" wordClassName="dk-w" deskSubset />
      ) : (
        <AiMarkdown content={text} className="dk-md" deskSubset />
      )}
    </div>
  );
}
