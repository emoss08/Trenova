import {
  describeActivity,
  groupActivity,
  stepsFromSegments,
} from "@/components/assistant/activity";
import { turnCitedRecords, type TurnState } from "@/components/assistant/turn-stream";
import { DeskProse } from "@/components/desk-chat/conversation/desk-turns";
import { MarkdownLinkContext, type MarkdownLinkRenderer } from "@/components/elements/ai-markdown";
import type { AssistantEntityRef } from "@/types/assistant";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CheckIcon,
  Copy01Icon,
  CornerDownLeftIcon,
  LayoutAlt02Icon,
  SquareIcon,
} from "@trenova/shared/components/icons";
import { useCallback, useMemo } from "react";
import { answerText } from "./ask-answer";
import { CITE_HREF_PREFIX, linkCitedRecords } from "./cited-text";
import { ShortcutKeys } from "./palette-parts";

type AnswerPhase = "working" | "answering" | "done" | "refused" | "error";

function phaseOf(turn: TurnState, text: string): AnswerPhase {
  switch (turn.status) {
    case "done":
      return "done";
    case "refused":
      return "refused";
    case "error":
      return "error";
    case "streaming":
      return text === "" ? "working" : "answering";
    default:
      return text === "" ? "working" : "answering";
  }
}

function RecordChip({
  record,
  children,
  onPeek,
  onOpen,
}: {
  record: AssistantEntityRef;
  children: React.ReactNode;
  onPeek: (record: AssistantEntityRef | null) => void;
  onOpen: (record: AssistantEntityRef) => void;
}) {
  return (
    <button
      type="button"
      tabIndex={-1}
      onMouseDown={(event) => event.preventDefault()}
      onMouseEnter={() => onPeek(record)}
      onMouseLeave={() => onPeek(null)}
      onFocus={() => onPeek(record)}
      onBlur={() => onPeek(null)}
      onClick={() => onOpen(record)}
      className="border-border bg-card text-foreground hover:bg-surface-hover mx-px inline-flex h-5 max-w-56 items-center rounded-[5px] border px-1.5 align-[1px] font-mono text-sm leading-none transition-colors"
    >
      <span className="truncate">{children}</span>
    </button>
  );
}

/**
 * Nova's answer to a quick question, at the head of the list. It shows the
 * turn's own steps while it works, then streams the answer with every record
 * it used drawn as a chip: hover one to look at it beside the list, click it
 * to open it. Which records are chips comes from the tools' results, so a
 * record the model only named is never made to look checked.
 */
export function AskAnswerCard({
  question,
  turn,
  subject,
  onAsk,
  onStop,
  onContinue,
  onCopy,
  onFollowUp,
  onPeek,
  onOpenRecord,
  mac,
  className,
}: {
  question: string;
  turn: TurnState;
  subject: string | null;
  onAsk: () => void;
  onStop: () => void;
  onContinue: () => void;
  onCopy: () => void;
  onFollowUp: () => void;
  onPeek: (record: AssistantEntityRef | null) => void;
  onOpenRecord: (record: AssistantEntityRef) => void;
  mac: boolean;
  className?: string;
}) {
  const t = useT();
  const text = answerText(turn);
  const phase = phaseOf(turn, text);
  const busy = phase === "working" || phase === "answering";
  const steps = useMemo(
    () =>
      groupActivity(stepsFromSegments(turn.segments)).map((group) => describeActivity(group, t)),
    [t, turn.segments],
  );
  const sources = useMemo(() => turnCitedRecords(turn.segments), [turn.segments]);
  const linked = useMemo(() => linkCitedRecords(text, sources), [sources, text]);

  const renderLink = useCallback<MarkdownLinkRenderer>(
    (href, children) => {
      if (!href.startsWith(CITE_HREF_PREFIX)) {
        return null;
      }
      const record = linked.records.get(href);
      return record ? (
        <RecordChip record={record} onPeek={onPeek} onOpen={onOpenRecord}>
          {children}
        </RecordChip>
      ) : null;
    },
    [linked.records, onOpenRecord, onPeek],
  );

  const status =
    phase === "working"
      ? t("Working…")
      : phase === "answering"
        ? t("Answering")
        : phase === "done"
          ? t("Answered from your records")
          : phase === "refused"
            ? t("Couldn't answer that")
            : t("Something went wrong");

  return (
    <section
      aria-label={t("Nova's answer")}
      aria-busy={busy}
      data-busy={busy}
      onMouseDown={(event) => event.preventDefault()}
      className={cn(
        "cp-spectrum rounded-surface border-border-subtle bg-surface-hover/40 mx-0.5 mt-1 mb-2 flex flex-col gap-1.5 border px-3.5 py-3",
        className,
      )}
    >
      <header className="flex h-5 items-center gap-1.5 text-sm">
        <NovaMark spin={busy ? "always" : "none"} className="size-3.5 shrink-0 translate-y-px" />
        <span className="text-foreground font-semibold">{t("Nova")}</span>
        <span className="text-foreground-subtle truncate">{status}</span>
        <span className="flex-1" />
        {subject && (
          <span className="text-foreground-muted truncate font-mono text-xs">{subject}</span>
        )}
        {busy && (
          <Button
            type="button"
            size="icon-xs"
            variant="ghost"
            tabIndex={-1}
            onClick={onStop}
            aria-label={t("Stop")}
            className="text-foreground-subtle"
          >
            <SquareIcon className="size-2.5 fill-current" />
          </Button>
        )}
      </header>
      <p className="text-foreground-subtle text-base">{question}</p>

      {text === "" && steps.length > 0 && (
        <ul className="flex flex-col gap-1">
          {steps.map((step, index) => (
            <li key={index} className="text-foreground-muted flex items-center gap-2 text-sm">
              {step.state === "running" ? (
                <Spinner className="size-3 shrink-0" />
              ) : (
                <CheckIcon
                  className={cn(
                    "size-3 shrink-0",
                    step.state === "failed" ? "text-danger" : "text-foreground-subtle",
                  )}
                  strokeWidth={2}
                />
              )}
              <span className="truncate">
                {step.phrase}
                {step.detail !== "" && (
                  <span className="text-foreground-subtle"> · {step.detail}</span>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
      {text === "" && steps.length === 0 && busy && (
        <p className="text-foreground-muted flex items-center gap-2 text-sm">
          <Spinner className="size-3" />
          {turn.status === "guarding" ? t("Checking the question…") : t("Reading your records…")}
        </p>
      )}

      {text !== "" && (
        <MarkdownLinkContext value={renderLink}>
          <div className="text-foreground text-base leading-[1.6]">
            <DeskProse content={linked.text} streaming={phase === "answering"} />
          </div>
        </MarkdownLinkContext>
      )}

      {phase === "refused" && turn.refusal && (
        <p className="text-foreground-muted text-base">{turn.refusal.message}</p>
      )}
      {phase === "error" && (
        <div className="flex flex-col items-start gap-2">
          {turn.error && <p className="text-foreground-muted text-base">{turn.error}</p>}
          <Button type="button" size="xs" variant="outline" tabIndex={-1} onClick={onAsk}>
            {t("Ask again")}
          </Button>
        </div>
      )}

      {phase === "done" && (
        <footer className="border-border-subtle mt-1 flex flex-nowrap items-center gap-1 border-t pt-2.5 whitespace-nowrap">
          <Button type="button" size="xs" variant="outline" tabIndex={-1} onClick={onContinue}>
            <LayoutAlt02Icon className="size-3.5" />
            {t("Continue in Desk")}
            <ShortcutKeys keys={[mac ? "⌘" : "Ctrl", "↵"]} className="ml-0.5" />
          </Button>
          <Button type="button" size="xs" variant="ghost" tabIndex={-1} onClick={onCopy}>
            <Copy01Icon className="size-3.5" />
            {t("Copy")}
          </Button>
          <Button type="button" size="xs" variant="ghost" tabIndex={-1} onClick={onFollowUp}>
            <CornerDownLeftIcon className="size-3.5" />
            {t("Follow up")}
          </Button>
          {sources.length > 0 && (
            <span className="text-foreground-subtle ml-auto hidden truncate text-sm sm:inline">
              {t("Hover a record to look at it")}
            </span>
          )}
        </footer>
      )}
    </section>
  );
}
