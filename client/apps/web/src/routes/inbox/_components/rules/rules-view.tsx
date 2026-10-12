import type { TriageRule } from "@/lib/graphql/inbox";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { ruleActionPhrase } from "../action-copy";
import { classificationLabel } from "../classification";
import { mailboxName } from "../folder-labels";
import { useTriageActions } from "../use-triage-actions";
import { useTriageMailboxes, useTriageRules } from "../use-triage-data";

/**
 * The Rules page's contract with the page: it takes nothing, reads its data
 * from `useTriageRules()` / `useTriageMailboxes()` and changes rules through
 * `useTriageActions()`.
 */
export type RulesViewProps = Record<string, never>;

function Blank({ children }: { children: ReactNode }) {
  return <b className="text-foreground font-medium">{children}</b>;
}

function RuleSentence({ rule }: { rule: TriageRule }) {
  const t = useT();
  const rt = useRichT();
  const mailboxes = useTriageMailboxes().data ?? [];
  const mailbox = mailboxes.find((row) => row.id === rule.mailboxId);
  let sender: string;
  switch (rule.senderScope) {
    case "Anyone":
      sender = t("anyone");
      break;
    case "AnyCustomer":
      sender = t("any customer");
      break;
    case "AnyCarrier":
      sender = t("any carrier");
      break;
    case "Domain":
    case "Address":
      sender = rule.sender;
      break;
  }

  return (
    <p className="text-foreground-muted text-base leading-relaxed">
      {rt(
        "When a <b>{0}</b> arrives from <b>{1}</b> on <b>{2}</b>, Nova should <b>{3}</b>.",
        { b: (chunks) => <Blank>{chunks}</Blank> },
        toSentenceFragment(classificationLabel(t, rule.classification)),
        sender,
        mailbox === undefined ? t("any source") : mailboxName(mailbox),
        ruleActionPhrase(t, rule.action),
      )}
    </p>
  );
}

/**
 * Every rule as the sentence it is, with the switch that pauses it and how
 * often it has matched. Editing the blanks in place, the add row and the
 * match preview are built on this.
 */
export function RulesView(_: RulesViewProps) {
  const t = useT();
  const actions = useTriageActions();
  const query = useTriageRules();
  const rules = query.data;
  const enabled = rules?.filter((rule) => rule.enabled) ?? [];
  const handled = enabled.reduce((sum, rule) => sum + rule.matchesThisMonth, 0);

  return (
    <div className="scrollbar-overlay min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-180 flex-col gap-6 px-8 pt-10 pb-16">
        <div className="flex items-baseline justify-between gap-4">
          <h2 className="text-2xl font-semibold">{t("Rules")}</h2>
          <span className="text-foreground-subtle font-mono text-xs tabular-nums">
            {t("{0} on · {1} handled this month", enabled.length, handled)}
          </span>
        </div>
        {rules === undefined ? (
          <div aria-busy="true" className="flex flex-col gap-4">
            <Skeleton className="h-6 w-full" />
            <Skeleton className="h-6 w-5/6" />
          </div>
        ) : (
          <ul className="flex flex-col">
            {rules.map((rule) => (
              <li
                key={rule.id}
                className={cn(
                  "border-border-subtle flex items-start gap-4 border-b py-4",
                  !rule.enabled && "opacity-60",
                )}
              >
                <Switch
                  checked={rule.enabled}
                  aria-label={rule.enabled ? t("Pause this rule") : t("Turn this rule on")}
                  onCheckedChange={(checked) => actions.setRuleEnabled(rule, checked)}
                  className="mt-1"
                />
                <div className="flex min-w-0 flex-col gap-1">
                  <RuleSentence rule={rule} />
                  <span className="text-foreground-subtle text-xs">
                    {!rule.enabled
                      ? t("Paused")
                      : rule.matchesThisMonth > 0
                        ? t("{0} this month", rule.matchesThisMonth)
                        : t("Waiting for a first match")}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
