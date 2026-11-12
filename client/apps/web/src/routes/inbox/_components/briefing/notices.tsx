import type { TriageBriefing, TriageRuleInput } from "@/lib/graphql/inbox";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useRef, useState, type ReactNode } from "react";
import { mailboxName } from "../folder-labels";
import type { TriageFolder } from "../folders";
import { inlineErrorText } from "../inline-error";
import { KIND_PLURAL } from "../kind-phrases";
import { senderShortName } from "../rules/rule-model";
import { useTriageMailboxes } from "../use-triage-data";

type Suggestion = NonNullable<TriageBriefing["suggestion"]>;

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

/** Whether the briefing has anything worth a look, so its block is drawn only when it does. */
export function hasNotices(
  briefing: Pick<TriageBriefing, "unhealthySources" | "suggestion">,
): boolean {
  return briefing.unhealthySources.length > 0 || briefing.suggestion !== null;
}

function Notice({ lead, children }: { lead: ReactNode; children: ReactNode }) {
  return (
    <li className="flex min-h-11 flex-wrap items-center gap-x-2.5 gap-y-1.5 px-3 py-2 text-dsk-base text-dsk-fg2">
      {lead}
      {children}
    </li>
  );
}

/**
 * What is worth a look beside the work itself: a source that went quiet,
 * and one rule worth handing to Nova, each with the one thing to do about it.
 */
export function BriefingNotices({
  briefing,
  onOpenFolder,
  onPreview,
  onTeach,
}: {
  briefing: Pick<TriageBriefing, "unhealthySources" | "suggestion">;
  onOpenFolder: (folder: TriageFolder) => void;
  onPreview: (input: TriageRuleInput) => void;
  onTeach: (input: TriageRuleInput) => Promise<unknown>;
}) {
  const t = useT();
  const sending = useRef(false);
  const [teaching, setTeaching] = useState(false);
  const [taught, setTaught] = useState<Suggestion | null>(null);
  const [refusal, setRefusal] = useState<string | null>(null);
  const mailboxes = useTriageMailboxes().data;
  const suggestion = taught ?? briefing.suggestion;

  const teach = (rule: Suggestion) => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    setTeaching(true);
    setRefusal(null);
    onTeach(suggestionInput(rule))
      .then(
        () => setTaught(rule),
        (error: unknown) => setRefusal(inlineErrorText(t, error)),
      )
      .finally(() => {
        sending.current = false;
        setTeaching(false);
      });
  };

  return (
    <ul className="flex flex-col divide-y divide-dsk-b-sub overflow-hidden rounded-[8px] border border-dsk-b bg-dsk-card">
      {briefing.unhealthySources.map((mailbox) => (
        <Notice
          key={mailbox.id}
          lead={
            <Badge variant="warning" dot>
              {t("Source")}
            </Badge>
          }
        >
          <span className="min-w-0 flex-1">
            {mailboxes?.find((row) => row.id === mailbox.id)?.health === "NeverReceived"
              ? t("{0} has not received mail yet", mailboxName(mailbox))
              : t("{0} stopped receiving mail", mailboxName(mailbox))}
          </span>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => onOpenFolder({ kind: "mailbox", mailboxId: mailbox.id })}
          >
            {t("Fix")}
          </Button>
        </Notice>
      ))}
      {suggestion !== null && (
        <Notice lead={<NovaBadge size="sm" spin={teaching ? "always" : "none"} />}>
          {taught !== null ? (
            <>
              <span role="status" className="min-w-0 flex-1">
                {taughtText(t, taught)}
              </span>
              <Badge variant="success" dot>
                {t("Handed to Nova")}
              </Badge>
            </>
          ) : (
            <>
              <span
                role={refusal === null ? undefined : "alert"}
                className={refusal === null ? "min-w-0 flex-1" : "min-w-0 flex-1 text-dsk-error"}
              >
                {refusal ?? suggestionText(t, suggestion)}
              </span>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => onPreview(suggestionInput(suggestion))}
              >
                {t("Preview")}
              </Button>
              <Button size="sm" isLoading={teaching} onClick={() => teach(suggestion)}>
                {t("Hand it to Nova")}
              </Button>
            </>
          )}
        </Notice>
      )}
    </ul>
  );
}
