import type { InboundMessageCounts, TriageBriefing, TriageRuleInput } from "@/lib/graphql/inbox";
import { CheckIcon, ClockIcon } from "@trenova/shared/components/icons";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { mailboxName } from "../folder-labels";
import { inlineErrorText } from "../inline-error";
import type { TriageFolder } from "../folders";
import { KIND_PLURAL } from "../kind-phrases";
import { senderShortName } from "../rules/rule-model";

type Suggestion = NonNullable<TriageBriefing["suggestion"]>;

const PILL =
  "ui-focus-ring inline-flex h-8 items-center gap-1.75 rounded-full border px-3 text-sm transition-colors";
const QUIET = "border-border text-foreground-muted hover:border-border-strong hover:bg-surface-hover";

function Count({ children }: { children: ReactNode }) {
  return <b className="text-foreground font-mono text-xs font-semibold tabular-nums">{children}</b>;
}

function Tag({ children }: { children: ReactNode }) {
  return (
    <em className="bg-surface-hover text-foreground-subtle -mr-1.5 rounded-full px-2 py-0.5 text-xs not-italic">
      {children}
    </em>
  );
}

function suggestionInput(suggestion: Suggestion): TriageRuleInput {
  return {
    classification: suggestion.classification,
    senderScope: suggestion.senderScope,
    sender: suggestion.sender === "" ? null : suggestion.sender,
    mailboxId: suggestion.mailboxId ?? null,
    action: suggestion.action,
    assigneeId: null,
    enabled: true,
  };
}

function suggestionText(t: TranslateFn, suggestion: Suggestion): string {
  const kinds = KIND_PLURAL[suggestion.classification];
  const who = senderShortName(suggestion.senderScope, suggestion.sender);
  switch (suggestion.action) {
    case "FileOnShipment":
    case "FileAndQueuePay":
      return t(
        "You filed {0} of {1} {2} from {3} by hand",
        suggestion.handledByHand,
        suggestion.total,
        kinds,
        who,
      );
    case "Ignore":
      return t(
        "You ignored {0} of {1} {2} from {3} by hand",
        suggestion.handledByHand,
        suggestion.total,
        kinds,
        who,
      );
    default:
      return t(
        "You handled {0} of {1} {2} from {3} the same way by hand",
        suggestion.handledByHand,
        suggestion.total,
        kinds,
        who,
      );
  }
}

function taughtText(t: TranslateFn, suggestion: Suggestion): string {
  const kinds = KIND_PLURAL[suggestion.classification];
  switch (suggestion.action) {
    case "FileOnShipment":
      return t("Nova files {0} from {1} now", kinds, suggestion.sender);
    case "FileAndQueuePay":
      return t("Nova files and queues {0} from {1} now", kinds, suggestion.sender);
    case "Ignore":
      return t("Nova ignores {0} from {1} now", kinds, suggestion.sender);
    default:
      return t("Nova handles {0} from {1} now", kinds, suggestion.sender);
  }
}

type FooterPillsProps = {
  briefing: Pick<
    TriageBriefing,
    "handledByNovaToday" | "corrections" | "comingBack" | "unhealthySources" | "suggestion"
  >;
  counts: InboundMessageCounts | undefined;
  onOpenFolder: (folder: TriageFolder) => void;
  onPreview: (input: TriageRuleInput) => void;
  onTeach: (input: TriageRuleInput) => Promise<unknown>;
};

/**
 * The quiet line under the deck: what Nova handled (to spot-check), how its
 * reading is doing, what comes back later, a source that went quiet, and
 * one rule worth handing to Nova. Every pill with a number opens what is
 * behind it.
 */
export function FooterPills({
  briefing,
  counts,
  onOpenFolder,
  onPreview,
  onTeach,
}: FooterPillsProps) {
  const t = useT();
  const [teaching, setTeaching] = useState(false);
  const [taught, setTaught] = useState<Suggestion | null>(null);
  const [refusal, setRefusal] = useState<string | null>(null);
  const { corrections, suggestion } = briefing;
  const shownSuggestion = taught ?? suggestion;

  const comingBackLane = (counts?.snoozed ?? 0) > 0 || (counts?.waitingOnThem ?? 0) === 0;

  return (
    <footer className="flex flex-wrap justify-center gap-2 @max-2xl:justify-start">
      <button
        type="button"
        onClick={() => onOpenFolder({ kind: "lane", lane: "handled" })}
        className={cn(PILL, QUIET, "pr-1.5")}
      >
        <Count>{briefing.handledByNovaToday}</Count>
        <span>{t("handled by Nova")}</span>
        <Tag>{t("Spot-check")}</Tag>
      </button>

      {corrections.thisWeek > 0 && (
        <button
          type="button"
          onClick={() => onOpenFolder({ kind: "lane", lane: "all" })}
          className={cn(PILL, QUIET)}
        >
          <CheckIcon aria-hidden className="size-3" strokeWidth={2.2} />
          <Count>{corrections.thisWeek}</Count>
          <span>
            {corrections.accuracyChange > 0
              ? t(
                  "{0, plural, one {correction this week} other {corrections this week}} · reading accuracy up {1}%",
                  corrections.thisWeek,
                  Math.round(corrections.accuracyChange),
                )
              : corrections.accuracyChange < 0
                ? t(
                    "{0, plural, one {correction this week} other {corrections this week}} · reading accuracy down {1}%",
                    corrections.thisWeek,
                    Math.round(Math.abs(corrections.accuracyChange)),
                  )
                : t(
                    "{0, plural, one {correction this week} other {corrections this week}}",
                    corrections.thisWeek,
                  )}
          </span>
        </button>
      )}

      {briefing.comingBack > 0 && (
        <button
          type="button"
          onClick={() =>
            onOpenFolder({ kind: "lane", lane: comingBackLane ? "snoozed" : "them" })
          }
          className={cn(PILL, QUIET)}
        >
          <ClockIcon aria-hidden className="size-3" />
          <Count>{briefing.comingBack}</Count>
          <span>{t("coming back later")}</span>
        </button>
      )}

      {briefing.unhealthySources.map((mailbox) => (
        <button
          key={mailbox.id}
          type="button"
          onClick={() => onOpenFolder({ kind: "mailbox", mailboxId: mailbox.id })}
          className={cn(
            PILL,
            "border-warning-border text-warning hover:bg-warning-subtle pr-1.5",
          )}
        >
          <span aria-hidden className="bg-warning size-1.75 rounded-full" />
          <span>
            {mailbox.health === "NeverReceived"
              ? t("{0} has not received mail yet", mailboxName(mailbox))
              : t("{0} stopped receiving mail", mailboxName(mailbox))}
          </span>
          <Tag>{t("Fix")}</Tag>
        </button>
      ))}

      {shownSuggestion !== null &&
        (taught !== null ? (
          <span
            role="status"
            className={cn(PILL, "border-success-border text-success animate-confirm")}
          >
            <CheckIcon aria-hidden className="size-2.75" strokeWidth={2.6} />
            <span>{taughtText(t, taught)}</span>
          </span>
        ) : (
          <span
            className={cn(
              PILL,
              "border-accent-violet-border text-foreground-muted max-w-full gap-1 border-dashed pr-1 @max-2xl:h-auto @max-2xl:flex-wrap @max-2xl:rounded-lg @max-2xl:py-1.5",
            )}
          >
            <span
              role={refusal === null ? undefined : "alert"}
              className={cn("min-w-0 truncate pr-1", refusal !== null && "text-danger")}
              title={refusal ?? shownSuggestion.sender}
            >
              {refusal ?? suggestionText(t, shownSuggestion)}
            </span>
            <button
              type="button"
              onClick={() => onPreview(suggestionInput(shownSuggestion))}
              className="ui-focus-ring text-foreground-subtle hover:bg-surface-hover hover:text-foreground h-6 shrink-0 rounded-full px-2.25 text-xs"
            >
              {t("Preview")}
            </button>
            <button
              type="button"
              disabled={teaching}
              onClick={() => {
                setTeaching(true);
                setRefusal(null);
                onTeach(suggestionInput(shownSuggestion)).then(
                  () => {
                    setTeaching(false);
                    setTaught(shownSuggestion);
                  },
                  (error: unknown) => {
                    setTeaching(false);
                    setRefusal(inlineErrorText(t, error));
                  },
                );
              }}
              className="ui-focus-ring ui-press bg-ink text-ink-foreground hover:bg-ink-hover inline-flex h-6 shrink-0 items-center gap-1 rounded-full px-2.25 text-xs font-medium disabled:opacity-60"
            >
              {teaching && <Spinner className="size-3" />}
              {t("Hand it to Nova")}
            </button>
          </span>
        ))}
    </footer>
  );
}
