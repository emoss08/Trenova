import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { checkDetail, openIssuesOf } from "@/lib/billing-queue-checks";
import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueueService } from "@/services/billing-queue";
import {
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type {
  AgentRead,
  BillingCheck,
  BillingCheckKey,
  BillingIssue,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { useReducedMotionConfig } from "motion/react";
import { Fragment, useEffect, useRef, useState, type ReactNode } from "react";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

function checkLabel(key: BillingCheckKey, t: TranslateFn): string {
  switch (key) {
    case "biller":
      return t("Biller");
    case "charges":
      return t("Charges");
    case "pod":
      return t("POD");
    case "terms":
      return t("Terms");
    case "duplicate":
      return t("Duplicate");
  }
}

/**
 * The five checks an item must clear before it is approved, each with what was
 * found, the choices that settle a flagged one where the biller already is, an undo
 * for what was settled, and what Nova made of the item under them. Settling answers
 * with the item, which replaces the cached one, so the charges, these checks and
 * the approve action follow at once.
 */
export function BillingQueueReviewChecks({
  item,
  attention = 0,
  onDocumentSelect,
}: {
  item: BillingQueueItem;
  /**
   * How many times the biller was sent here to settle what closes Approve. Each new
   * call brings the review into view and pulses the checks that wait on them.
   */
  attention?: number;
  onDocumentSelect: (docId: string, fileName: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useInvalidateBillingQueue();
  const { allowed: canUpdate } = usePermission(Resource.BillingQueue, Operation.Update);
  const sectionRef = useRef<HTMLElement>(null);
  // Calls made before this item opened are not this item's to answer.
  const [answered] = useState(attention);
  const called = attention > answered;
  // Reduced motion, asked for in Trenova or in the system (MotionConfig carries
  // both), means no pulse and no smooth scroll: the review is simply brought into view.
  const reduceMotion = useReducedMotionConfig() ?? false;
  // Two names on the same keyframes, so a second call replays the pulse.
  const pulse =
    called && !reduceMotion
      ? attention % 2 === 0
        ? "animate-attention-a"
        : "animate-attention-b"
      : undefined;

  useEffect(() => {
    if (!called) return;
    sectionRef.current?.scrollIntoView({
      behavior: reduceMotion ? "auto" : "smooth",
      block: "start",
    });
    sectionRef.current?.focus({ preventScroll: true });
  }, [attention, called, reduceMotion]);

  const settle = (next: BillingQueueItem) => {
    if (next.review) {
      queryClient.setQueryData(billingQueue.get(item.id).queryKey, next);
    }
    invalidate();
  };
  const resolve = useApiMutation({
    mutationFn: ({ issueId, optionKey }: { issueId: string; optionKey: string }) =>
      billingQueueService.resolveIssue(item.id, issueId, optionKey),
    resourceName: "BillingQueueItem",
    onSuccess: settle,
  });
  const undo = useApiMutation({
    mutationFn: (issueId: string) => billingQueueService.undoIssue(item.id, issueId),
    resourceName: "BillingQueueItem",
    onSuccess: settle,
  });

  const review = item.status === "InReview" ? item.review : null;
  const hasNote = Boolean(item.agentSummary && item.agentVerdict);
  if (!review && !hasNote) {
    return null;
  }

  const busy = resolve.isPending || undo.isPending;
  const flagged = review?.checks.filter((check) => check.state !== "ok") ?? [];
  const settled = review?.issues.filter((issue) => issue.resolvedAt && issue.undoable) ?? [];

  return (
    <section
      ref={sectionRef}
      tabIndex={-1}
      aria-label={t("Review")}
      className="border-border scroll-mt-4 rounded-md border text-sm outline-none"
      data-testid={review ? "billing-queue-review-checks" : undefined}
    >
      {review ? (
        <>
          <div className="border-border flex items-center gap-2 border-b px-3 py-2.5">
            <h3 className="text-foreground font-semibold">{t("Review")}</h3>
            <span className="flex-1" />
            <span
              className={cn(
                "text-xs font-medium",
                flagged.length === 0 ? "text-success-foreground" : "text-warning-foreground",
              )}
            >
              {flagged.length === 0
                ? t("All five clear")
                : t("{0, plural, one {# needs you} other {# need you}}", review.needsCount)}
            </span>
          </div>
          <ul className="py-1">
            {review.checks.map((check) => (
              <CheckRow
                key={check.key}
                item={item}
                check={check}
                pulse={check.state === "ok" ? undefined : pulse}
                canAct={canUpdate}
                busy={busy}
                onResolve={(issueId, optionKey) => resolve.mutate({ issueId, optionKey })}
              />
            ))}
          </ul>
          {settled.length > 0 ? (
            <SettledIssues
              issues={settled}
              canUndo={canUpdate}
              busy={busy}
              onUndo={(issueId) => undo.mutate(issueId)}
            />
          ) : null}
        </>
      ) : null}
      {hasNote ? (
        <NovaNote
          key={item.id}
          item={item}
          bordered={review !== null}
          onDocumentSelect={onDocumentSelect}
        />
      ) : null}
    </section>
  );
}

function CheckRow({
  item,
  check,
  pulse,
  canAct,
  busy,
  onResolve,
}: {
  item: BillingQueueItem;
  check: BillingCheck;
  /** The animation that marks a check the biller was just sent to settle. */
  pulse?: string;
  canAct: boolean;
  busy: boolean;
  onResolve: (issueId: string, optionKey: string) => void;
}) {
  const t = useT();
  const [shown, setShown] = useState(0);
  const issues = openIssuesOf(item, check);
  const ok = check.state === "ok";
  const at = Math.min(shown, Math.max(issues.length - 1, 0));
  const issue = issues[at];

  return (
    <li
      className={cn("mx-1 flex items-start gap-2.5 rounded-md px-2 py-1.5", pulse)}
      data-testid={`billing-queue-check-${check.key}`}
    >
      {ok ? (
        <span
          role="img"
          aria-label={t("Clear")}
          className="bg-success-subtle text-success-foreground flex size-4 shrink-0 items-center justify-center rounded-full"
        >
          <CheckIcon className="size-2.5" />
        </span>
      ) : (
        <span
          role="img"
          aria-label={t("Needs you")}
          className="bg-warning-subtle text-warning-foreground flex size-4 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
        >
          !
        </span>
      )}
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <div className="flex flex-wrap items-baseline gap-2">
          <span className="text-foreground w-21 shrink-0 font-medium">
            {checkLabel(check.key, t)}
          </span>
          <span
            className={cn(
              "min-w-0 flex-[1_1_200px]",
              ok ? "text-foreground-muted" : "text-foreground",
            )}
          >
            {issue?.summary ?? checkDetail(check, t)}
          </span>
          {issues.length > 1 ? (
            <IssuePager at={at} count={issues.length} onTurn={setShown} />
          ) : null}
        </div>
        {check.key === "biller" && issues.length === 0 && !ok ? (
          <p className="text-foreground-muted pl-23 text-xs">
            {t("Start review or assign a biller to clear it.")}
          </p>
        ) : null}
        {issue ? (
          <div className="flex flex-col gap-1.5 pl-23">
            <IssueChoices issue={issue} canAct={canAct} busy={busy} onResolve={onResolve} />
          </div>
        ) : null}
      </div>
    </li>
  );
}

/**
 * Turns through a check's open issues one at a time. Settling the one shown takes
 * it out of the list, so the next one moves into its place.
 */
function IssuePager({
  at,
  count,
  onTurn,
}: {
  at: number;
  count: number;
  onTurn: (index: number) => void;
}) {
  const t = useT();

  return (
    <span className="flex shrink-0 items-center gap-1 self-center">
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        aria-label={t("Previous issue")}
        disabled={at === 0}
        onClick={() => onTurn(at - 1)}
      >
        <ChevronLeftIcon className="size-3.5" />
      </Button>
      <span className="text-foreground-muted text-xs tabular-nums">
        {t("{0} of {1}", at + 1, count)}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        aria-label={t("Next issue")}
        disabled={at === count - 1}
        onClick={() => onTurn(at + 1)}
      >
        <ChevronRightIcon className="size-3.5" />
      </Button>
    </span>
  );
}

/** Why an issue was raised and the choices that settle it. */
function IssueChoices({
  issue,
  canAct,
  busy,
  onResolve,
}: {
  issue: BillingIssue;
  canAct: boolean;
  busy: boolean;
  onResolve: (issueId: string, optionKey: string) => void;
}) {
  const t = useT();

  return (
    <>
      {issue.reasoning ? <p className="text-foreground-muted text-xs">{issue.reasoning}</p> : null}
      {canAct && issue.options.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {issue.options.map((option, index) => {
            const asked = option.effect.kind === "request" && issue.requestedAt;
            return (
              <Button
                key={option.key}
                size="xs"
                variant={index === 0 && !asked ? "default" : "outline"}
                disabled={busy || Boolean(asked)}
                onClick={() => onResolve(issue.id, option.key)}
              >
                {asked
                  ? t(
                      "Asked {0}",
                      formatUnixInUserTimezone(issue.requestedAt, {
                        month: "short",
                        day: "numeric",
                        hour: "numeric",
                        minute: "2-digit",
                      }),
                    )
                  : option.label}
              </Button>
            );
          })}
        </div>
      ) : null}
    </>
  );
}

function SettledIssues({
  issues,
  canUndo,
  busy,
  onUndo,
}: {
  issues: BillingIssue[];
  canUndo: boolean;
  busy: boolean;
  onUndo: (issueId: string) => void;
}) {
  const t = useT();

  return (
    <ul className="border-border flex flex-col gap-1 border-t px-3 py-2 text-xs">
      {issues.map((issue) => (
        <li
          key={issue.id}
          className="text-foreground-muted flex items-center justify-between gap-3"
        >
          <span>{t("Settled · {0}", issue.resolutionText || issue.summary)}</span>
          {canUndo && (
            <Button size="xs" variant="ghost" disabled={busy} onClick={() => onUndo(issue.id)}>
              {t("Undo")}
            </Button>
          )}
        </li>
      ))}
    </ul>
  );
}

const bold = { b: (chunk: ReactNode) => <b className="font-semibold">{chunk}</b> };

/**
 * What Nova made of the item after the checks ran: agreement in one line, or the
 * thing that looks off, with what it read to get there. It informs; it never
 * blocks approval.
 */
function NovaNote({
  item,
  bordered,
  onDocumentSelect,
}: {
  item: BillingQueueItem;
  bordered: boolean;
  onDocumentSelect: (docId: string, fileName: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const now = useNow("minute");
  const [showReads, setShowReads] = useState(false);
  const hasReads = item.agentReads.length > 0;

  return (
    <div
      className={cn(
        "bg-muted flex gap-2.5 rounded-b-md px-3 py-2.5",
        bordered ? "border-border border-t" : "rounded-t-md",
      )}
      data-testid="billing-queue-nova-note"
    >
      <NovaBadge size="sm" className="mt-0.5" />
      <Collapsible
        open={showReads}
        onOpenChange={setShowReads}
        className="flex min-w-0 flex-1 flex-col gap-1.5"
      >
        <p className="text-foreground leading-4.5">
          {rt("<b>Nova</b> · {0}", bold, item.agentSummary ?? "")}
        </p>
        {item.agentReviewedAt || hasReads ? (
          <div className="text-foreground-muted flex flex-wrap items-center gap-3 text-xs">
            {item.agentReviewedAt ? (
              <span>{t("Reviewed {0}", formatSecondsAgo(now - item.agentReviewedAt))}</span>
            ) : null}
            {hasReads ? (
              <CollapsibleTrigger className="text-brand ui-focus-ring inline-flex items-center gap-0.5 rounded-sm font-medium hover:underline">
                {t("See what it read")}
                <ChevronDownIcon
                  aria-hidden
                  className={cn(
                    "size-3 transition-transform duration-200 ease-settle motion-reduce:transition-none",
                    showReads && "rotate-180",
                  )}
                />
              </CollapsibleTrigger>
            ) : null}
          </div>
        ) : null}
        {hasReads ? (
          <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height,opacity] duration-200 ease-settle data-ending-style:h-0 data-ending-style:opacity-0 data-starting-style:h-0 data-starting-style:opacity-0 motion-reduce:transition-none">
            <NovaReads item={item} onDocumentSelect={onDocumentSelect} />
          </CollapsibleContent>
        ) : null}
      </Collapsible>
    </div>
  );
}

/**
 * What Nova read to review the item, laid out like the checks: the paperwork and how
 * far it read each (a document opens beside the item), the shipment, what the payer
 * keeps on file, and the payer's history with each charge it looked up across it.
 */
function NovaReads({
  item,
  onDocumentSelect,
}: {
  item: BillingQueueItem;
  onDocumentSelect: (docId: string, fileName: string) => void;
}) {
  const t = useT();
  const reads = item.agentReads;
  const of = (kind: AgentRead["kind"]) => reads.filter((read) => read.kind === kind);
  const documents = of("Document").filter((read) => read.documentId);
  const shipment = [
    ...of("Invoice").map((read) =>
      t("{0, plural, one {# charge} other {# charges}}", read.count ?? 0),
    ),
    ...of("Stops").map((read) =>
      t("{0, plural, one {# stop with times} other {# stops with times}}", read.count ?? 0),
    ),
    ...of("Comments").map((read) =>
      t("{0, plural, one {# comment} other {# comments}}", read.count ?? 0),
    ),
  ];
  const onFile = [
    ...of("Terms").map(() => t("Terms")),
    ...of("BillingNotes").map(() => t("Billing notes")),
  ];
  const history = [
    ...of("PastInvoices").map((read) =>
      t("{0, plural, one {Last invoice} other {Last # invoices}}", read.count ?? 0),
    ),
    ...of("ChargeHistory").map((read) => {
      const code = read.code === "FREIGHT" ? t("Freight") : (read.code ?? "");
      return read.of ? t("{0} on {1} of {2}", code, read.count ?? 0, read.of) : code;
    }),
  ];

  return (
    <dl
      className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 pt-1 text-xs"
      data-testid="billing-queue-nova-reads"
    >
      {documents.length > 0 ? (
        <ReadRow label={t("Paperwork")}>
          {documents.map((read, index) => (
            <Fragment key={read.documentId}>
              {index > 0 ? <span aria-hidden="true"> · </span> : null}
              <span className="whitespace-nowrap">
                <button
                  type="button"
                  className="text-brand ui-focus-ring rounded-sm font-medium hover:underline"
                  onClick={() => onDocumentSelect(read.documentId ?? "", read.fileName ?? "")}
                >
                  {read.name || read.fileName}
                </button>
                {documentReadText(read, t)}
              </span>
            </Fragment>
          ))}
        </ReadRow>
      ) : null}
      {shipment.length > 0 ? (
        <ReadRow label={t("Shipment")}>
          <Joined pieces={shipment} />
        </ReadRow>
      ) : null}
      {onFile.length > 0 ? (
        <ReadRow label={t("Payer")}>
          <Joined pieces={onFile} />
        </ReadRow>
      ) : null}
      {history.length > 0 ? (
        <ReadRow label={t("History")}>
          <Joined pieces={history} />
        </ReadRow>
      ) : null}
    </dl>
  );
}

/** One row of what Nova read: what kind of thing, then what it read of it. */
function ReadRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-foreground font-medium">{label}</dt>
      <dd className="text-foreground-muted min-w-0">{children}</dd>
    </>
  );
}

/** Short pieces of one row, set apart by a dot. */
function Joined({ pieces }: { pieces: string[] }) {
  return pieces.map((piece, index) => (
    <Fragment key={piece}>
      {index > 0 ? <span aria-hidden="true"> · </span> : null}
      <span className="whitespace-nowrap">{piece}</span>
    </Fragment>
  ));
}

function documentReadText(read: AgentRead, t: TranslateFn): string {
  const parts = read.parts ?? 0;
  const part = read.part ?? 0;
  if (parts === 0) {
    return t(", no text");
  }
  if (parts > 1 && part < parts) {
    return t(", {0} of {1} parts", part, parts);
  }

  return "";
}
