import { askRequestsFrom } from "@/components/assistant/ask-requests";
import { ChoicePrompt } from "@/components/assistant/choice-prompt";
import { ReportRunCard } from "@/components/assistant/report-run-card";
import { reportRunsFrom } from "@/components/assistant/report-runs";
import type { ThreadEntry } from "@/components/assistant/thread-view";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { AiMarkdown, StreamingAiMarkdown } from "@/components/elements/ai-markdown";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
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
export function DeskQuestion({ text, muted = false, tag }: { text: string; muted?: boolean; tag?: string }) {
  const size = questionSize(text);

  return (
    <div
      className={cn(
        "dk-q",
        size === "mid" && "dk-mid",
        size === "long" && "dk-long",
        muted && "dk-ec-q dk-muted",
      )}
    >
      {text}
      {tag && <span className="dk-ec-qtag">{tag}</span>}
    </div>
  );
}

/** The artifacts a reply produced, as dark-blue badges that open them in the workspace. */
export function DeskArtifactBadges({
  artifacts,
  activeId,
  onOpen,
}: {
  artifacts: readonly Pick<AssistantArtifact, "id" | "kind" | "title">[];
  activeId: string | null;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  if (artifacts.length === 0) {
    return null;
  }

  return (
    <div className="dk-abadges">
      {artifacts.map((artifact) => (
        <button
          key={artifact.id}
          type="button"
          className={cn("dk-abadge", activeId === artifact.id && "dk-on")}
          aria-label={t("Open {0}", artifact.title)}
          onClick={() => onOpen(artifact.id)}
        >
          <ArtifactKindIcon kind={artifact.kind} className="dk-abadge-i" />
          <span>{artifact.title}</span>
        </button>
      ))}
    </div>
  );
}

/** A saved reply: its words, what it produced, and the actions under it. */
export function DeskReply({
  entry,
  artifacts,
  activeArtifactId,
  latestUserSequence,
  chapter,
  onTogglePin,
  onAnswer,
  onOpenArtifact,
}: {
  entry: Extract<ThreadEntry, { kind: "assistant" }>;
  artifacts: readonly AssistantArtifact[];
  activeArtifactId: string | null;
  latestUserSequence: number;
  chapter: number;
  onTogglePin: () => void;
  onAnswer?: (value: string) => void;
  onOpenArtifact: (id: string) => void;
}) {
  const { message, tools } = entry;
  const asks = askRequestsFrom(tools);
  const reportRuns = reportRunsFrom(tools);

  return (
    <>
      {message.content !== "" && (
        <div className="dk-prose">
          <AiMarkdown content={message.content} className="dk-md" />
        </div>
      )}
      <DeskArtifactBadges artifacts={artifacts} activeId={activeArtifactId} onOpen={onOpenArtifact} />
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
        <DeskMessageActions text={message.content} chapter={chapter} onTogglePin={onTogglePin} />
      )}
    </>
  );
}

/** The reply being written, word by word; nothing shows until its first words arrive. */
export function DeskStreamingReply({ text }: { text: string }) {
  return (
    <div className="dk-prose dk-streaming">
      <StreamingAiMarkdown content={text} className="dk-md" />
    </div>
  );
}
