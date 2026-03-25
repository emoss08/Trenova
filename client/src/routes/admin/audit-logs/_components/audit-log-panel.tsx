import { ComponentLoader } from "@/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { formatToUserTimezone } from "@/lib/date";
import type { AuditEntry } from "@/types/audit-entry";
import type { DataTablePanelProps } from "@/types/data-table";
import { useState } from "react";
import {
  changeTypeLabel,
  formatAuditValue,
  formatAuditValueWithDates,
  formatFieldLabel,
  isRecordValue,
  isSensitiveOmittedValue,
  normalizeAuditChanges,
  operationLabel,
  resourceLabel,
} from "./audit-log-formatters";
import { ShikiJsonBlock } from "./audit-shiki-json";

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
        <h3 className="text-sm font-semibold text-foreground">{title}</h3>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </div>
      {children}
    </section>
  );
}

function EntryDetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[140px_1fr] gap-2 border-b border-border/60 py-2 last:border-b-0">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm text-foreground">{value}</dd>
    </div>
  );
}

function AuditValueCell({ value, path }: { value: unknown; path?: string }) {
  const [expanded, setExpanded] = useState(false);

  if (!Array.isArray(value) && !isRecordValue(value)) {
    const formatted = formatAuditValueWithDates(value, path);
    const isSensitiveOmitted = isSensitiveOmittedValue(value);

    return (
      <div className="space-y-1">
        {isSensitiveOmitted && (
          <Badge variant="warning" className="h-5 px-1.5 text-[10px]">
            Sensitive
          </Badge>
        )}
        <p className="text-xs break-words text-foreground">{formatted.value}</p>
        {formatted.transformed && (
          <p className="font-mono text-[11px] text-muted-foreground">
            Raw: {formatAuditValue(value)}
          </p>
        )}
      </div>
    );
  }

  const isArray = Array.isArray(value);
  const count = isArray ? value.length : Object.keys(value).length;
  const summary = isArray ? `Array (${count} items)` : `Object (${count} fields)`;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant="outline">{summary}</Badge>
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          className="h-6 px-2"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Hide JSON" : "View JSON"}
        </Button>
      </div>
      {expanded && <ShikiJsonBlock value={value} />}
    </div>
  );
}

function ChangeRow({
  path,
  type,
  from,
  to,
}: {
  path: string;
  type: "added" | "removed" | "changed";
  from: unknown;
  to: unknown;
}) {
  return (
    <div className="space-y-2 border-b border-border/60 py-3 last:border-b-0">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{formatFieldLabel(path)}</p>
        </div>
        <div className="flex items-center gap-1">
          <p className="text-xs font-medium text-muted-foreground">
            Action: {changeTypeLabel(type)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
        <div className="space-y-1 rounded-md border border-red-500/20 bg-red-500/8 p-2.5">
          <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Previous Value
          </p>
          <AuditValueCell value={from} path={`${path}.from`} />
        </div>
        <div className="space-y-1 rounded-md border border-green-500/20 bg-green-500/8 p-2.5">
          <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Current Value
          </p>
          <AuditValueCell value={to} path={`${path}.to`} />
        </div>
      </div>
    </div>
  );
}

export function AuditLogPanel({ open, onOpenChange, row }: DataTablePanelProps<AuditEntry>) {
  if (!row) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title="Audit Entry"
        description="Loading audit details"
        size="xl"
      >
        <ComponentLoader message="Loading audit entry..." />
      </DataTablePanelContainer>
    );
  }

  const changedFields = normalizeAuditChanges(row.changes ?? {});

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row.comment || `${operationLabel(row.operation)} ${resourceLabel(row.resource)}`}
      description={`Recorded on ${formatToUserTimezone(row.timestamp, { showTimeZone: true })}`}
      size="xl"
    >
      <div className="space-y-5">
        <Section title="Entry Details" description="Detailed information about this audit event">
          <dl className="rounded-md border border-border/70 px-3">
            <EntryDetailRow
              label="Event ID"
              value={<span className="font-mono text-xs break-all">{row.id}</span>}
            />
            <EntryDetailRow
              label="Resource ID"
              value={<span className="font-mono text-xs break-all">{row.resourceId}</span>}
            />
            <EntryDetailRow label="Operation" value={operationLabel(row.operation)} />
            <EntryDetailRow label="Resource" value={resourceLabel(row.resource)} />
            <EntryDetailRow
              label="User"
              value={row.user?.name || row.user?.emailAddress || "Unknown user"}
            />
            <EntryDetailRow label="Critical" value={row.critical ? "Yes" : "No"} />
            <EntryDetailRow label="IP Address" value={row.ipAddress || "-"} />
            <EntryDetailRow label="Category" value={row.category || "-"} />
            <EntryDetailRow
              label="Timestamp"
              value={formatToUserTimezone(row.timestamp, { showTimeZone: true })}
            />
            <EntryDetailRow label="Correlation ID" value={row.correlationId || "-"} />
            <EntryDetailRow label="User Agent" value={row.userAgent || "-"} />
          </dl>
        </Section>

        <Section title="Changes" description="Field-level before/after values">
          <ScrollArea className="h-76">
            <div className="rounded-md border border-border/70 px-3">
              {changedFields.length === 0 ? (
                <div className="py-3 text-xs text-muted-foreground italic">
                  No changes recorded.
                </div>
              ) : (
                changedFields.map((change) => (
                  <ChangeRow
                    key={change.path}
                    path={change.path}
                    type={change.type}
                    from={change.from}
                    to={change.to}
                  />
                ))
              )}
            </div>
          </ScrollArea>
        </Section>

        <Section title="Metadata" description="Additional contextual information">
          <ShikiJsonBlock value={row.metadata} />
        </Section>

        <Section title="Previous State" description="State before the operation">
          <ShikiJsonBlock value={row.previousState} />
        </Section>

        <Section title="Current State" description="State after the operation">
          <ShikiJsonBlock value={row.currentState} />
        </Section>

        <Section title="Full Event Data" description="Complete raw event payload">
          <ShikiJsonBlock value={row} />
        </Section>
      </div>
    </DataTablePanelContainer>
  );
}
