import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { TriageRule, TriageRuleInput, TriageRulePreview } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import type { RuleMatchesTarget } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { ruleActionPhrase } from "../../action-copy";
import { classificationLabel } from "../../classification";
import { mailboxName } from "../../folder-labels";
import { inlineErrorText } from "../../inline-error";
import { idleRuleInput } from "../../rules/rule-model";
import { useTriageMailboxes, useTriageRules } from "../../use-triage-data";

type Match = TriageRulePreview["matches"][number];
type Mismatch = NonNullable<Match["mismatch"]>;

const ROW_HEIGHT = 44;

function mismatchText(t: TranslateFn, mismatch: Mismatch): string {
  switch (mismatch) {
    case "KindCorrected":
      return t("Kind looks wrong");
    case "NoMatchedRecord":
      return t("No matching shipment");
    case "PersonDisagreed":
      return t("A person did something else");
  }
}

type RuleShape = Pick<
  TriageRuleInput,
  "classification" | "senderScope" | "sender" | "mailboxId" | "action"
>;

function ruleOf(
  target: RuleMatchesTarget,
  rules: readonly TriageRule[] | undefined,
): RuleShape | null {
  if ("input" in target) {
    return target.input;
  }

  return rules?.find((rule) => rule.id === target.ruleId) ?? null;
}

/**
 * What a rule matched over the last thirty days, or would have before it
 * was turned on: each message with the reference it was about, ticked when
 * the rule would have done the right thing and marked with why when a person
 * did something else. The list is virtual, so a busy sender's month opens as
 * quickly as a quiet one.
 */
export function RuleMatchesDialog({
  target,
  onClose,
}: {
  target: RuleMatchesTarget;
  onClose: () => void;
}) {
  const t = useT();
  const rules = useTriageRules().data;
  const mailboxes = useTriageMailboxes().data;
  const drafted = "input" in target;
  const previewQuery = useQuery({
    ...queries.inbox.rulePreview(drafted ? target.input : idleRuleInput()),
    enabled: drafted,
  });
  const matchesQuery = useQuery({
    ...queries.inbox.ruleMatches(drafted ? "" : target.ruleId),
    enabled: !drafted,
  });
  const preview = drafted ? previewQuery : matchesQuery;
  const rule = ruleOf(target, rules);
  const data = preview.data;

  const subtitle = useMemo(() => {
    if (rule === null) {
      return t("Last 30 days");
    }
    const kind = toSentenceFragment(classificationLabel(t, rule.classification));
    const sender =
      rule.senderScope === "Anyone"
        ? t("anyone")
        : rule.senderScope === "AnyCustomer"
          ? t("any customer")
          : rule.senderScope === "AnyCarrier"
            ? t("any carrier")
            : (rule.sender ?? "");
    const mailbox =
      rule.mailboxId === null || rule.mailboxId === undefined
        ? undefined
        : mailboxes?.find((row) => row.id === String(rule.mailboxId));
    const source = mailbox === undefined ? t("any source") : mailboxName(mailbox);
    return t(
      "{0} from {1} on {2} → {3} · last 30 days",
      kind.charAt(0).toLocaleUpperCase() + kind.slice(1),
      sender,
      source,
      ruleActionPhrase(t, rule.action),
    );
  }, [rule, mailboxes, t]);

  const rows = useMemo<VirtualRow[]>(
    () =>
      (data?.matches ?? []).map((match) => ({
        key: match.messageId,
        render: () => <MatchRow match={match} />,
      })),
    [data],
  );
  const more = data === undefined ? 0 : Math.max(0, data.matched - data.matches.length);

  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent size="md" className="gap-0 p-0">
        <DialogHeader className="border-border-subtle gap-0.5 border-b px-4 pt-3.5 pb-3">
          <DialogTitle className="text-base font-semibold">
            {data === undefined
              ? t("What it would have handled")
              : t(
                  "{0, plural, one {Would have handled # message} other {Would have handled # messages}}",
                  data.matched,
                )}
          </DialogTitle>
          <DialogDescription className="text-foreground-subtle text-sm">
            {subtitle}
          </DialogDescription>
        </DialogHeader>
        <div className="px-2 py-1">
          {preview.isError ? (
            <p role="alert" className="text-danger p-3 text-sm">
              {inlineErrorText(t, preview.error)}
            </p>
          ) : data === undefined ? (
            <div aria-busy="true" className="flex flex-col gap-2 p-2">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-5/6" />
            </div>
          ) : (
            <VirtualRows
              rows={rows}
              estimateSize={ROW_HEIGHT}
              className="max-h-[min(420px,60vh)]"
              aria-label={t("Messages it matched")}
              empty={
                <p className="text-foreground-subtle p-3 text-sm">
                  {t("Nothing in the last 30 days would have matched.")}
                </p>
              }
              footer={
                more > 0 ? (
                  <p className="text-foreground-subtle px-2 py-2 text-xs">
                    {t("{0, plural, one {+ # more} other {+ # more}}", more)}
                  </p>
                ) : undefined
              }
            />
          )}
        </div>
        <DialogFooter className="border-border-subtle mx-0 mb-0 flex items-center gap-2 rounded-b-xl border-t px-3 py-2.5">
          {data !== undefined && data.off > 0 && (
            <span className="text-warning mr-auto text-sm">
              {t(
                "{0, plural, one {# would have needed a person — narrow the sender to avoid it} other {# would have needed a person — narrow the sender to avoid them}}",
                data.off,
              )}
            </span>
          )}
          <Button onClick={onClose}>{t("Done")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MatchRow({ match }: { match: Match }) {
  const t = useT();
  const sender = match.fromName.trim() === "" ? match.fromAddress : match.fromName;
  const about = match.reference.trim() === "" ? match.subject : match.reference;
  const off = match.mismatch !== null;

  return (
    <div
      className={cn(
        "grid h-11 grid-cols-[64px_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md px-2 text-sm",
        off ? "bg-warning-subtle" : "hover:bg-surface-hover",
      )}
    >
      <span className="text-foreground-subtle font-mono text-2xs tabular-nums">
        {formatUnixInUserTimezone(match.receivedAt, { month: "short", day: "numeric" }, "")}
      </span>
      <span className="flex min-w-0 flex-col leading-tight">
        <b className="truncate font-medium">{sender}</b>
        <em className="text-foreground-subtle truncate text-xs not-italic">{about}</em>
      </span>
      {match.mismatch === null ? (
        <CheckIcon
          aria-label={t("Looks right")}
          className="text-success size-3"
          strokeWidth={2.2}
        />
      ) : (
        <span className="text-warning-subtle-foreground inline-flex items-center gap-1.25 text-xs whitespace-nowrap">
          <AlertTriangleIcon aria-hidden className="size-2.75" />
          {mismatchText(t, match.mismatch)}
        </span>
      )}
    </div>
  );
}
