import type { TriageBriefing } from "@/lib/graphql/inbox";
import { senderName } from "@/lib/triage/message-state";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { USER_FORMATTERS, actionCopy } from "../action-copy";
import { CLASSIFICATION_DOT } from "../classification";
import { dueSpan, dueText } from "../due";
import { WAITING, type TriageFolder } from "../folders";
import { useTriageBriefing } from "../use-triage-data";

/**
 * The Briefing's contract with the page: it opens messages and folders
 * through these, and reads its data from `useTriageBriefing()` and performs
 * every action through `useTriageActions()`.
 */
export type BriefingViewProps = {
  /** Opens a message in Waiting on you, as every name, dot and card row does. */
  onOpenMessage: (id: string) => void;
  /** Opens a folder of the queue, as a count or a footer pill does. */
  onOpenFolder: (folder: TriageFolder) => void;
};

type DeckCard = TriageBriefing["deck"][number];

function deckTitle(t: ReturnType<typeof useT>, card: DeckCard): string {
  switch (card.group) {
    case "File":
      return t("{0, plural, one {# document to file} other {# documents to file}}", card.count);
    case "Draft":
      return t(
        "{0, plural, one {# draft Nova can write} other {# drafts Nova can write}}",
        card.count,
      );
    case "Decide":
      return t("{0, plural, one {# needs your call} other {# need your call}}", card.count);
    case "Link":
      return t("{0, plural, one {# without a record} other {# without a record}}", card.count);
    case "Held":
      return t("{0, plural, one {# held back} other {# held back}}", card.count);
  }
}

function BriefingSkeleton() {
  return (
    <div aria-busy="true" className="mx-auto flex w-full max-w-180 flex-col gap-5 px-8 pt-12">
      <Skeleton className="size-20 rounded-full" />
      <Skeleton className="h-6 w-3/4" />
      <Skeleton className="h-6 w-1/2" />
      <Skeleton className="h-40 w-full" />
    </div>
  );
}

/**
 * Triage's landing page: how much is left for you, what Nova took care of,
 * and the work grouped by what it needs. The ring, the river, the deck's
 * own treatment and the footer pills are built on this.
 */
export function BriefingView({ onOpenMessage, onOpenFolder }: BriefingViewProps) {
  const t = useT();
  const rt = useRichT();
  const now = useNowSeconds();
  const query = useTriageBriefing();
  const briefing = query.data;

  if (briefing === undefined) {
    return <BriefingSkeleton />;
  }

  const longest = briefing.longestWaiting;

  return (
    <div className="scrollbar-overlay min-h-0 flex-1 overflow-y-auto">
      <div className="animate-rise mx-auto flex w-full max-w-180 flex-col gap-6 px-8 pt-12 pb-16">
        <button
          type="button"
          onClick={() => onOpenFolder(WAITING)}
          className="ui-focus-ring border-border flex size-20 flex-col items-center justify-center rounded-full border-4"
        >
          <span className="text-3xl font-semibold tabular-nums">{briefing.waiting}</span>
          <span className="text-foreground-subtle text-2xs">{t("for you")}</span>
        </button>

        <p className="text-foreground-muted text-xl leading-snug">
          {rt(
            "I took care of <b>{0}</b> today. <b>{1}</b> are yours.",
            { b: (chunks) => <b className="text-foreground font-semibold">{chunks}</b> },
            briefing.handledByNovaToday,
            briefing.waiting,
          )}{" "}
          {longest !== null && (
            <button
              type="button"
              onClick={() => onOpenMessage(longest.messageId)}
              className="ui-focus-ring text-warning underline underline-offset-4"
            >
              {t("{0} has waited longest.", senderName(longest))}
            </button>
          )}{" "}
          {briefing.estimateMinutes > 0 &&
            t(
              "{0, plural, one {About # minute of work.} other {About # minutes of work.}}",
              briefing.estimateMinutes,
            )}
        </p>

        <div className="flex flex-col gap-3">
          {briefing.deck.map((card) => (
            <section
              key={card.group}
              aria-label={deckTitle(t, card)}
              className="border-border bg-card rounded-lg border px-5 py-4"
            >
              <h2 className="text-lg font-semibold">{deckTitle(t, card)}</h2>
              <ul className="mt-2 flex flex-col">
                {card.items.map((item) => (
                  <li key={item.messageId}>
                    <button
                      type="button"
                      onClick={() => onOpenMessage(item.messageId)}
                      className="ui-focus-ring hover:bg-surface-hover flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left"
                    >
                      <span
                        aria-hidden
                        className={cn(
                          "size-2 shrink-0 rounded-full",
                          CLASSIFICATION_DOT[item.classification ?? "Other"],
                        )}
                      />
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate text-sm font-medium">{senderName(item)}</span>
                        <span className="text-foreground-subtle truncate text-xs">
                          {item.action === null
                            ? item.subject
                            : actionCopy(t, item.action, item, USER_FORMATTERS).title}
                        </span>
                      </span>
                      {item.dueAt !== null && (
                        <span className="text-foreground-subtle font-mono text-xs tabular-nums">
                          {dueText(t, dueSpan(item.dueAt, now))}
                        </span>
                      )}
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          ))}
          {briefing.deck.length === 0 && (
            <div className="flex flex-col items-start gap-2">
              <p className="text-foreground-muted text-sm">{t("Nothing is waiting on you.")}</p>
              <Button size="sm" variant="outline" onClick={() => onOpenFolder(WAITING)}>
                {t("Open Waiting on you")}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
