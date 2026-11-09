import { isRecordEntityType, recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { AuditEntryDetail } from "@/lib/graphql/audit-log-table";
import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  ClockRewindIcon,
  Copy01Icon,
  LinkExternal01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { ChangeRow, RequestEntries } from "./audit-entry-parts";
import { actorKindOfPrincipal, entrySentence } from "./audit-entry-sentence";
import { AuditJsonBlock } from "./audit-json-block";
import {
  channelLabel,
  isRecordValue,
  normalizeAuditChanges,
  recordCurrency,
} from "./audit-log-formatters";
import { REVIEW_KIND_LABELS, reviewTone } from "./audit-review-kinds";
import { actorSubtitle, EntryTags } from "./audit-trail-columns";
import { ActorMark } from "./audit-top-actors";
import { useReviewDecision } from "./use-review-decision";

function Panel({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section aria-label={title} className="animate-rise rounded-md border bg-card">
      <header className="flex h-8.5 items-center gap-2 border-b px-3 text-sm font-semibold">
        {title}
        {aside ? (
          <span className="ml-auto flex items-center gap-1.5 text-xs font-normal text-muted-foreground">
            {aside}
          </span>
        ) : null}
      </header>
      {children}
    </section>
  );
}

function ReviewNote({ entryId }: { entryId: string }) {
  const t = useT();
  const now = useNow("minute");
  const { data: review } = useQuery(audit.entryReview(entryId));
  const decision = useReviewDecision({ id: review?.id ?? "" });

  if (!review) return null;

  if (review.reviewedAt) {
    return (
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <CheckCircleIcon aria-hidden className="size-3.5 text-success-foreground" />
        <span>
          {t(
            "Reviewed by {0} {1}",
            review.reviewedBy?.name ?? t("someone"),
            formatRelativeTime(review.reviewedAt - now),
          )}
        </span>
        <Button
          type="button"
          size="xxs"
          variant="ghost"
          className="ml-auto"
          isLoading={decision.reopening}
          onClick={() => void decision.reopen()}
        >
          {t("Reopen")}
        </Button>
      </div>
    );
  }

  const danger = reviewTone(review.kind) === "danger";
  return (
    <div
      className={cn(
        "flex animate-rise gap-2 rounded-md border px-3 py-2.5",
        danger
          ? "border-danger-border bg-danger-subtle"
          : "border-warning-border bg-warning-subtle",
      )}
    >
      <AlertTriangleIcon
        aria-hidden
        className={cn(
          "mt-0.5 size-3.5 shrink-0",
          danger ? "text-danger-foreground" : "text-warning-foreground",
        )}
      />
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-sm font-medium">{t("An alert rule flagged this")}</span>
        <span className="text-xs text-muted-foreground">
          {review.offHours
            ? t("{0} · off hours", REVIEW_KIND_LABELS[review.kind])
            : REVIEW_KIND_LABELS[review.kind]}
        </span>
        <div className="mt-1.5">
          <Button
            type="button"
            size="xs"
            variant="outline"
            isLoading={decision.marking}
            onClick={() => void decision.markReviewed()}
          >
            {t("Mark reviewed")}
          </Button>
        </div>
      </div>
    </div>
  );
}

function InspectorBody({
  entry,
  canSeeReviews,
  onOpenEntry,
}: {
  entry: AuditEntryDetail;
  canSeeReviews: boolean;
  onOpenEntry: (id: string) => void;
}) {
  const t = useT();
  const [raw, setRaw] = useState<"before" | "after" | "metadata">("before");
  const changes = normalizeAuditChanges(isRecordValue(entry.changes) ? entry.changes : {});
  const currency = recordCurrency(entry.currentState, entry.previousState);
  const name = entry.actorName || entry.user?.name || t("Unknown");
  const rawValue =
    raw === "before" ? entry.previousState : raw === "after" ? entry.currentState : entry.metadata;

  return (
    <div className="flex flex-col gap-3 p-3.5">
      {canSeeReviews ? <ReviewNote entryId={entry.id} /> : null}

      <Panel
        title={t("What changed")}
        aside={t("{0, plural, one {# field} other {# fields}}", changes.length)}
      >
        {changes.length === 0 ? (
          <p className="px-3 py-2.5 text-xs text-muted-foreground">{t("No fields changed.")}</p>
        ) : (
          changes.map((change) => (
            <ChangeRow
              key={change.path}
              change={change}
              resourceId={entry.resourceId}
              currency={currency}
            />
          ))
        )}
      </Panel>

      <Panel title={t("Who")} aside={channelLabel(entry.channel)}>
        <div className="flex flex-col gap-2.5 px-3 py-2.5">
          <div className="flex items-center gap-2.5">
            <ActorMark
              kind={actorKindOfPrincipal(entry.principalType)}
              name={name}
              user={entry.user}
              size="lg"
            />
            <div className="flex min-w-0 flex-col">
              <span className="truncate text-sm font-medium">{name}</span>
              <span className="truncate text-xs text-muted-foreground">
                {actorSubtitle(
                  t,
                  actorKindOfPrincipal(entry.principalType),
                  entry.channel,
                  entry.user?.emailAddress,
                )}
              </span>
            </div>
          </div>
          <DescriptionList layout="inline">
            {entry.aiRunId ? (
              <DescriptionItem label={t("Agent run")} valueClassName="font-mono text-xs break-all">
                {entry.aiRunId}
              </DescriptionItem>
            ) : null}
            <DescriptionItem label={t("IP address")} valueClassName="font-mono text-xs">
              {entry.ipAddress || "—"}
            </DescriptionItem>
            <DescriptionItem label={t("Device")}>{entry.userAgent || "—"}</DescriptionItem>
          </DescriptionList>
        </div>
      </Panel>

      {entry.correlationId ? (
        <Panel
          title={t("Same request")}
          aside={<span className="font-mono">{entry.correlationId}</span>}
        >
          <RequestEntries
            correlationId={entry.correlationId}
            entryId={entry.id}
            onOpen={onOpenEntry}
          />
        </Panel>
      ) : null}

      <Panel title={t("Kept as")} aside={entry.critical ? t("Critical") : undefined}>
        <DescriptionList layout="inline" className="px-3 py-2.5">
          <DescriptionItem label={t("Retention")}>{entry.retentionClass}</DescriptionItem>
          <DescriptionItem label={t("Category")}>{entry.category}</DescriptionItem>
          <DescriptionItem label={t("Event ID")} valueClassName="font-mono text-xs break-all">
            {entry.id}
          </DescriptionItem>
          {entry.traceId ? (
            <DescriptionItem label={t("Trace ID")} valueClassName="font-mono text-xs break-all">
              {entry.traceId}
            </DescriptionItem>
          ) : null}
          <DescriptionItem label={t("Recorded")}>
            {formatToUserTimezone(entry.timestamp, { showTimeZone: true })}
          </DescriptionItem>
        </DescriptionList>
      </Panel>

      <Panel title={t("Raw")}>
        <div className="flex flex-col gap-2 p-2.5">
          <SegmentedControl
            aria-label={t("Raw view")}
            items={[
              { value: "before", label: t("Before") },
              { value: "after", label: t("After") },
              { value: "metadata", label: t("Metadata") },
            ]}
            value={raw}
            onValueChange={setRaw}
          />
          {rawValue && isRecordValue(rawValue) && Object.keys(rawValue).length > 0 ? (
            <AuditJsonBlock value={rawValue} searchable copyPath />
          ) : (
            <p className="px-1 text-xs text-muted-foreground">{t("Nothing recorded.")}</p>
          )}
        </div>
      </Panel>
    </div>
  );
}

/**
 * One entry beside the trail: the sentence and its flags, the record it changed, what changed, who
 * did it and through what, the rest of its request, how long it is kept and its raw states. A
 * review the entry set off sits on top with its decision.
 */
export function AuditInspector({
  entryId,
  canSeeReviews,
  offHoursZone,
  onOpenEntry,
  onShowRecordHistory,
  onClose,
}: {
  entryId: string;
  canSeeReviews: boolean;
  offHoursZone: string | null;
  onOpenEntry: (id: string) => void;
  onShowRecordHistory: (resourceId: string) => void;
  onClose: () => void;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  const detail = useQuery(audit.entry(entryId));
  const entry = detail.data;

  return (
    <aside
      aria-label={t("Audit entry")}
      className="flex max-h-[calc(100dvh-6rem)] min-h-0 animate-panel-in flex-col overflow-hidden rounded-md border bg-background"
    >
      <header className="flex shrink-0 flex-col gap-2 border-b px-3.5 py-3">
        <div className="flex items-center gap-1.5">
          {entry ? (
            <>
              <Badge variant="neutral" className="font-mono font-normal">
                {entry.action}
              </Badge>
              <EntryTags entry={entry} offHoursZone={offHoursZone} />
            </>
          ) : (
            <span className="h-5 w-32 ui-shimmer rounded-sm" />
          )}
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="ml-auto"
            aria-label={t("Close")}
            onClick={onClose}
          >
            <XCloseIcon className="size-3.5" />
          </Button>
        </div>
        {entry ? (
          <>
            <h2 className="text-lg leading-snug font-semibold text-balance">
              {entrySentence(entry)}
            </h2>
            <p className="text-xs text-muted-foreground">
              {t(
                "{0} · {1}",
                entry.actorName || entry.user?.name || t("Unknown"),
                formatToUserTimezone(entry.timestamp, { showTimeZone: true }),
              )}
            </p>
            <div className="flex flex-wrap gap-1.5">
              {isRecordEntityType(entry.resource) && entry.operation.toLowerCase() !== "delete" ? (
                <Button
                  size="xs"
                  variant="outline"
                  nativeButton={false}
                  render={<Link to={recordPath(entry.resource, entry.resourceId)} />}
                >
                  <LinkExternal01Icon aria-hidden className="size-3" />
                  {t("Open record")}
                </Button>
              ) : null}
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={() => onShowRecordHistory(entry.resourceId)}
              >
                <ClockRewindIcon aria-hidden className="size-3" />
                {t("Record history")}
              </Button>
              <Button
                type="button"
                size="xs"
                variant="ghost"
                onClick={() => {
                  const url = new URL(window.location.href);
                  url.searchParams.set("entry", entry.id);
                  void copy(url.toString());
                }}
              >
                <Copy01Icon aria-hidden className="size-3" />
                {isCopied ? t("Copied") : t("Copy link")}
              </Button>
            </div>
          </>
        ) : (
          <div aria-hidden className="flex flex-col gap-1.5">
            <span className="h-5 w-11/12 ui-shimmer rounded-sm" />
            <span className="h-3 w-40 ui-shimmer rounded-sm" />
          </div>
        )}
      </header>
      <ScrollArea className="min-h-0 flex-1">
        {entry ? (
          <InspectorBody entry={entry} canSeeReviews={canSeeReviews} onOpenEntry={onOpenEntry} />
        ) : detail.isError || detail.isSuccess ? (
          <p className="p-4 text-sm text-muted-foreground">
            {t("This entry could not be loaded, or you can no longer read it.")}
          </p>
        ) : (
          <div
            aria-busy
            aria-label={t("Loading audit entry")}
            className="flex flex-col gap-3 p-3.5"
          >
            <span aria-hidden className="block h-40 ui-shimmer rounded-md" />
            <span aria-hidden className="block h-28 ui-shimmer rounded-md" />
            <span aria-hidden className="block h-24 ui-shimmer rounded-md" />
          </div>
        )}
      </ScrollArea>
      <footer className="flex shrink-0 gap-3.5 border-t px-3.5 py-2 text-xs text-muted-foreground">
        <span>{t("↑ ↓ next entry")}</span>
        <span>{t("Esc closes")}</span>
      </footer>
    </aside>
  );
}
