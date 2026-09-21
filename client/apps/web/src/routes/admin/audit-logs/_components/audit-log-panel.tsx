import { useT } from "@trenova/shared/i18n/use-t";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
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
import type { AuditEntryRow } from "@/lib/graphql/audit-log-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
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
  const summary = isArray ? `Array (${count} items)` : `Object (${count} fields)`;

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
  const t = useT();

  return (
    <div className="border-border/60 space-y-2 border-b py-3 last:border-b-0">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-foreground text-sm font-medium">{formatFieldLabel(path)}</p>
        </div>
        <div className="flex items-center gap-1">
          <p className="text-muted-foreground text-xs font-medium">
            {t("Action: {0}", changeTypeLabel(type))}
          </p>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
        <div className="space-y-1 rounded-md border border-danger-border bg-danger-subtle p-2.5">
          <p className="text-muted-foreground text-xs font-medium">{t("Previous Value")}</p>
          <AuditValueCell value={from} path={`${path}.from`} />
        </div>
        <div className="space-y-1 rounded-md border border-success-border bg-success-subtle p-2.5">
          <p className="text-muted-foreground text-xs font-medium">{t("Current Value")}</p>
          <AuditValueCell value={to} path={`${path}.to`} />
        </div>
      </div>
    </div>
  );
}

export function AuditLogPanel({ open, onOpenChange, row }: DataTablePanelProps<AuditEntryRow>) {
  const t = useT();

  if (!row) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={t("Audit Entry")}
        description={t("Loading audit details")}
        size="xl"
      >
        <ComponentLoader message={t("Loading audit entry...")} />
      </DataTablePanelContainer>
    );
  }

  const changedFields = normalizeAuditChanges(isRecordValue(row.changes) ? row.changes : {});

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row.comment || `${operationLabel(row.operation)} ${resourceLabel(row.resource)}`}
      description={`Recorded on ${formatToUserTimezone(row.timestamp, { showTimeZone: true })}`}
      size="xl"
    >
      <div className="space-y-5">
        <Section
          title={t("Entry Details")}
          description={t("Detailed information about this audit event")}
        >
          <DescriptionList layout="inline" className="rounded-md border p-3">
            <DescriptionItem label={t("Event ID")} valueClassName="font-mono text-xs break-all">
              {row.id}
            </DescriptionItem>
            <DescriptionItem label={t("Resource ID")} valueClassName="font-mono text-xs break-all">
              {row.resourceId}
            </DescriptionItem>
            <DescriptionItem label={t("Operation")}>
              {operationLabel(row.operation)}
            </DescriptionItem>
            <DescriptionItem label={t("Resource")}>{resourceLabel(row.resource)}</DescriptionItem>
            <DescriptionItem label={t("User")}>
              {row.user?.name || row.user?.emailAddress || "Unknown user"}
            </DescriptionItem>
            <DescriptionItem label={t("Critical")}>{row.critical ? "Yes" : "No"}</DescriptionItem>
            <DescriptionItem label={t("IP Address")} numeric>
              {row.ipAddress || <DescriptionEmpty />}
            </DescriptionItem>
            <DescriptionItem label={t("Category")}>
              {row.category || <DescriptionEmpty />}
            </DescriptionItem>
            <DescriptionItem label={t("Timestamp")} numeric>
              {formatToUserTimezone(row.timestamp, {
                showTimeZone: true,
              })}
            </DescriptionItem>
            <DescriptionItem label={t("Correlation ID")}>
              {row.correlationId || <DescriptionEmpty />}
            </DescriptionItem>
            <DescriptionItem label={t("User Agent")}>
              {row.userAgent || <DescriptionEmpty />}
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
                  />
                ))
              )}
            </div>
          </ScrollArea>
        </Section>
        <Section title={t("Metadata")} description={t("Additional contextual information")}>
          <ShikiJsonBlock value={row.metadata} searchable />
        </Section>
        <Section title={t("Previous State")} description={t("State before the operation")}>
          <ShikiJsonBlock value={row.previousState} searchable copyPath />
        </Section>
        <Section title={t("Current State")} description={t("State after the operation")}>
          <ShikiJsonBlock value={row.currentState} searchable copyPath />
        </Section>
        <Section title={t("Full Event Data")} description={t("Complete raw event payload")}>
          <ShikiJsonBlock value={row} searchable copyPath />
        </Section>
      </div>
    </DataTablePanelContainer>
  );
}
