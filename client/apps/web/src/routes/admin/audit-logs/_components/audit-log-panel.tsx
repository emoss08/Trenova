import { useT } from "@trenova/shared/i18n/use-t";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { AuditEntryDetail, AuditEntryRow } from "@/lib/graphql/audit-log-table";
import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useState } from "react";
import {
  changeTypeLabel,
  formatAuditValue,
  formatAuditValueWithDates,
  formatFieldLabel,
  isRecordValue,
  isSensitiveOmittedValue,
  auditEntryTitle,
  channelLabel,
  normalizeAuditChanges,
  operationLabel,
  resourceLabel,
} from "./audit-log-formatters";
import { AuditJsonBlock } from "./audit-json-block";

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="space-y-2">
      <div>
        <h3 className="text-foreground text-sm font-semibold">{title}</h3>
        {description && <p className="text-muted-foreground text-xs">{description}</p>}
      </div>
      {children}
    </section>
  );
}

function AuditValueCell({ value, path }: { value: unknown; path?: string }) {
  const t = useT();

  const [expanded, setExpanded] = useState(false);

  if (!Array.isArray(value) && !isRecordValue(value)) {
    const formatted = formatAuditValueWithDates(value, path);
    const isSensitiveOmitted = isSensitiveOmittedValue(value);

    return (
      <div className="space-y-1">
        {isSensitiveOmitted && (
          <Badge variant="warning" className="h-5 px-1.5 text-2xs">
            {t("Sensitive")}
          </Badge>
        )}
        <p className="text-foreground text-xs wrap-break-word">{formatted.value}</p>
        {formatted.transformed && (
          <p className="text-muted-foreground font-mono text-xs">
            {t("Raw: {0}", formatAuditValue(value))}
          </p>
        )}
      </div>
    );
  }

  const isArray = Array.isArray(value);
  const count = isArray ? value.length : Object.keys(value).length;
  const summary = isArray
    ? t("{0, plural, one {Array (# item)} other {Array (# items)}}", count)
    : t("{0, plural, one {Object (# field)} other {Object (# fields)}}", count);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant="neutral" appearance="outline">
          {summary}
        </Badge>
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          className="h-6 px-2"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t("Hide JSON") : t("View JSON")}
        </Button>
      </div>
      {expanded && <AuditJsonBlock value={value} />}
    </div>
  );
}

function FieldHistory({ resourceId, field }: { resourceId: string; field: string }) {
  const t = useT();
  const { data, isPending, isError } = useQuery(audit.fieldHistory(resourceId, field));

  if (isPending) {
    return (
      <div aria-busy aria-label={t("Loading this field's history")} className="space-y-1">
        <Skeleton aria-hidden className="h-6 w-full" />
        <Skeleton aria-hidden className="h-6 w-full" />
      </div>
    );
  }
  if (isError || !data || data.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">{t("No earlier changes to this field.")}</p>
    );
  }

  return (
    <ol className="space-y-1">
      {data.map((change) => {
        const value = isRecordValue(change.changes) ? change.changes[field] : undefined;
        const to = isRecordValue(value) ? value.to : undefined;
        return (
          <li key={change.id} className="flex items-baseline justify-between gap-3 text-xs">
            <span className="text-foreground wrap-break-word">{formatAuditValue(to)}</span>
            <span className="text-muted-foreground shrink-0">
              {t(
                "{0} by {1}",
                formatToUserTimezone(change.timestamp),
                change.actorName || t("unknown"),
              )}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

function ChangeRow({
  path,
  type,
  from,
  to,
  resourceId,
}: {
  path: string;
  type: "added" | "removed" | "changed";
  from: unknown;
  to: unknown;
  resourceId: string;
}) {
  const t = useT();
  const [showHistory, setShowHistory] = useState(false);

  return (
    <div className="border-border/60 space-y-2 border-b py-3 last:border-b-0">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-foreground text-sm font-medium">{formatFieldLabel(path)}</p>
        </div>
        <div className="flex items-center gap-2">
          <p className="text-muted-foreground text-xs font-medium">
            {t("Action: {0}", changeTypeLabel(type))}
          </p>
          <Button
            type="button"
            variant="ghost"
            size="xxs"
            className="h-6 px-2"
            aria-expanded={showHistory}
            onClick={() => setShowHistory((current) => !current)}
          >
            {showHistory ? t("Hide history") : t("History")}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
        <div className="space-y-1 rounded-md border border-danger-border bg-danger-subtle p-2.5">
          <p className="text-muted-foreground text-xs font-medium">{t("Previous value")}</p>
          <AuditValueCell value={from} path={`${path}.from`} />
        </div>
        <div className="space-y-1 rounded-md border border-success-border bg-success-subtle p-2.5">
          <p className="text-muted-foreground text-xs font-medium">{t("Current value")}</p>
          <AuditValueCell value={to} path={`${path}.to`} />
        </div>
      </div>
      {showHistory ? (
        <div className="rounded-md border p-2.5">
          <FieldHistory resourceId={resourceId} field={path} />
        </div>
      ) : null}
    </div>
  );
}

function SameRequest({ correlationId, entryId }: { correlationId: string; entryId: string }) {
  const t = useT();
  const { data, isPending } = useQuery(audit.requestEntries(correlationId));

  if (isPending) {
    return (
      <div aria-busy aria-label={t("Loading the other entries from this request")}>
        <Skeleton aria-hidden className="h-16 w-full" />
      </div>
    );
  }
  const others = (data ?? []).filter((entry) => entry.id !== entryId);
  if (others.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">{t("This request wrote no other entry.")}</p>
    );
  }

  return (
    <ol className="divide-border/60 divide-y rounded-md border">
      {others.map((entry) => (
        <li key={entry.id} className="flex items-baseline justify-between gap-3 px-3 py-2 text-xs">
          <span className="text-foreground">
            {entry.comment || auditEntryTitle(entry.operation, entry.resource)}
            {entry.entityRef ? (
              <span className="text-muted-foreground"> · {entry.entityRef}</span>
            ) : null}
          </span>
          <span className="text-muted-foreground shrink-0">{entry.action}</span>
        </li>
      ))}
    </ol>
  );
}

function PanelSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading audit entry")} className="space-y-5">
      <div aria-hidden className="contents">
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-76 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    </div>
  );
}

function EntryDetail({ entry }: { entry: AuditEntryDetail }) {
  const t = useT();
  const changedFields = normalizeAuditChanges(isRecordValue(entry.changes) ? entry.changes : {});

  return (
    <div className="space-y-5">
      <Section
        title={t("Entry details")}
        description={t("Who did what, how it arrived and how long it is kept")}
      >
        <DescriptionList layout="inline" className="rounded-md border p-3">
          <DescriptionItem label={t("Event ID")} valueClassName="font-mono text-xs break-all">
            {entry.id}
          </DescriptionItem>
          <DescriptionItem label={t("Action")} valueClassName="font-mono text-xs">
            {entry.action}
          </DescriptionItem>
          <DescriptionItem label={t("Resource")}>{resourceLabel(entry.resource)}</DescriptionItem>
          <DescriptionItem label={t("Resource ID")} valueClassName="font-mono text-xs break-all">
            {entry.resourceId}
          </DescriptionItem>
          <DescriptionItem label={t("Record")}>
            {entry.entityRef || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Operation")}>
            {operationLabel(entry.operation)}
          </DescriptionItem>
          <DescriptionItem label={t("Who")}>
            {entry.actorName || entry.user?.name || t("Unknown")}
          </DescriptionItem>
          <DescriptionItem label={t("Channel")}>{channelLabel(entry.channel)}</DescriptionItem>
          <DescriptionItem label={t("Agent run")} valueClassName="font-mono text-xs break-all">
            {entry.aiRunId || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Category")}>{entry.category}</DescriptionItem>
          <DescriptionItem label={t("Kept as")}>{entry.retentionClass}</DescriptionItem>
          <DescriptionItem label={t("Critical")}>
            {entry.critical ? t("Yes") : t("No")}
          </DescriptionItem>
          <DescriptionItem label={t("IP address")} numeric>
            {entry.ipAddress || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Timestamp")} numeric>
            {formatToUserTimezone(entry.timestamp, { showTimeZone: true })}
          </DescriptionItem>
          <DescriptionItem label={t("Request ID")} valueClassName="font-mono text-xs break-all">
            {entry.correlationId || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Trace ID")} valueClassName="font-mono text-xs break-all">
            {entry.traceId || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("User agent")}>
            {entry.userAgent || <DescriptionEmpty />}
          </DescriptionItem>
        </DescriptionList>
      </Section>

      <Section title={t("Changes")} description={t("Field-level before/after values")}>
        <ScrollArea className="h-76">
          <div className="border-border/70 rounded-md border px-3">
            {changedFields.length === 0 ? (
              <div className="text-muted-foreground py-3 text-xs italic">
                {t("No changes recorded.")}
              </div>
            ) : (
              changedFields.map((change) => (
                <ChangeRow
                  key={change.path}
                  path={change.path}
                  type={change.type}
                  from={change.from}
                  to={change.to}
                  resourceId={entry.resourceId}
                />
              ))
            )}
          </div>
        </ScrollArea>
      </Section>
      {entry.correlationId ? (
        <Section
          title={t("Same request")}
          description={t("Everything else written while serving the request that made this change")}
        >
          <SameRequest correlationId={entry.correlationId} entryId={entry.id} />
        </Section>
      ) : null}
      {isRecordValue(entry.metadata) && Object.keys(entry.metadata).length > 0 ? (
        <Section title={t("Metadata")} description={t("Additional contextual information")}>
          <AuditJsonBlock value={entry.metadata} searchable />
        </Section>
      ) : null}
      {entry.previousState ? (
        <Section title={t("Previous state")} description={t("State before the operation")}>
          <AuditJsonBlock value={entry.previousState} searchable copyPath />
        </Section>
      ) : null}
      {entry.currentState ? (
        <Section title={t("Current state")} description={t("State after the operation")}>
          <AuditJsonBlock value={entry.currentState} searchable copyPath />
        </Section>
      ) : null}
    </div>
  );
}

export function AuditLogPanel({ open, onOpenChange, row }: DataTablePanelProps<AuditEntryRow>) {
  const t = useT();
  const detail = useQuery({ ...audit.entry(row?.id ?? ""), enabled: Boolean(row?.id) && open });

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row ? row.comment || auditEntryTitle(row.operation, row.resource) : t("Audit entry")}
      description={
        row
          ? t("Recorded on {0}", formatToUserTimezone(row.timestamp, { showTimeZone: true }))
          : t("Loading audit details")
      }
      size="xl"
    >
      {detail.data ? (
        <EntryDetail entry={detail.data} />
      ) : detail.isError || (detail.isSuccess && !detail.data) ? (
        <p className="text-muted-foreground text-sm">
          {t("This entry could not be loaded, or you can no longer read it.")}
        </p>
      ) : (
        <PanelSkeleton />
      )}
    </DataTablePanelContainer>
  );
}
