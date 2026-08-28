import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { downloadReportRun } from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { formatCurrency, formatFileSize } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Notification } from "@trenova/shared/types/notification";
import { formatDistanceToNowStrict } from "date-fns";
import { DownloadIcon } from "lucide-react";
import { MentionReply } from "./notification-mention-reply";
import {
  notificationDataBoolean,
  notificationDataNumber,
  notificationDataString,
} from "./notification-registry";

const REPORT_ATTACHMENT_EVENT_TYPES = new Set(["report_run_completed", "report_run_delivered"]);

function ReportRunAttachment({ notification }: { notification: Notification }) {
  const { allowed: canExport } = usePermission(Resource.Report, Operation.Export);

  const runId = notificationDataString(notification, "runId");
  if (!runId) return null;

  const name = notificationDataString(notification, "reportName") ?? "Report";
  const format = notificationDataString(notification, "format");
  const byteSize = notificationDataNumber(notification, "byteSize");
  const rowCount = notificationDataNumber(notification, "rowCount");
  const truncated = notificationDataBoolean(notification, "truncated");
  const expiresAt = notificationDataNumber(notification, "artifactExpiresAt");
  const expired = expiresAt !== null && expiresAt * 1000 <= Date.now();

  const meta: string[] = [];
  if (format) meta.push(format.toUpperCase());
  if (byteSize !== null && byteSize > 0) meta.push(formatFileSize(byteSize));
  if (rowCount !== null) meta.push(`${rowCount.toLocaleString()} rows`);
  if (expiresAt !== null) {
    meta.push(
      expired
        ? "Expired — run the report again"
        : `Expires ${formatDistanceToNowStrict(new Date(expiresAt * 1000), { addSuffix: true })}`,
    );
  }

  return (
    <div className="border-border bg-card mt-2 flex items-center gap-2.5 rounded-md border px-2.5 py-2">
      <DocumentFileTypeIcon fileName={format ? `${name}.${format}` : name} size="sm" />

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="text-foreground truncate text-xs font-medium">{name}</span>
        {meta.length > 0 && (
          <span className="text-2xs text-muted-foreground truncate">{meta.join(" · ")}</span>
        )}
      </div>
      {truncated && (
        <Badge variant="warning" className="text-2xs h-4.5 shrink-0">
          Truncated
        </Badge>
      )}
      {canExport && !expired && (
        <Button
          variant="outline"
          size="icon-xs"
          className="shrink-0"
          aria-label={`Download ${name}`}
          onClick={(event) => {
            event.stopPropagation();
            downloadReportRun({ id: runId });
          }}
        >
          <DownloadIcon className="size-3" />
        </Button>
      )}
    </div>
  );
}

function TableChangeDetails({ notification }: { notification: Notification }) {
  const operation = notificationDataString(notification, "operation");
  const tableName = notificationDataString(notification, "tableName");
  const changedFields = Array.isArray(notification.data?.changedFields)
    ? notification.data.changedFields.filter((field): field is string => typeof field === "string")
    : [];

  if (!operation && !tableName && changedFields.length === 0) return null;

  const visibleFields = changedFields.slice(0, 3);
  const hiddenCount = changedFields.length - visibleFields.length;

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1">
      {operation && (
        <Badge variant="info" className="text-2xs h-4.5 uppercase">
          {operation}
        </Badge>
      )}
      {tableName && (
        <Badge variant="outline" className="text-2xs h-4.5 font-mono">
          {tableName}
        </Badge>
      )}
      {visibleFields.map((field) => (
        <Badge key={field} variant="secondary" className="text-2xs h-4.5 font-mono">
          {field}
        </Badge>
      ))}
      {hiddenCount > 0 && (
        <span className="text-2xs text-muted-foreground">+{hiddenCount} more</span>
      )}
    </div>
  );
}

function AmountRow({
  label,
  value,
  emphasis,
}: {
  label: string;
  value: string;
  emphasis?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-2xs text-muted-foreground">{label}</span>
      <span
        className={
          emphasis
            ? "text-2xs text-warning font-medium tabular-nums"
            : "text-2xs text-foreground font-medium tabular-nums"
        }
      >
        {value}
      </span>
    </div>
  );
}

function ReconciliationDetails({ notification }: { notification: Notification }) {
  const expected = notificationDataNumber(notification, "expectedTotal");
  const posted = notificationDataNumber(notification, "invoiceTotal");
  const discrepancy = notificationDataNumber(notification, "discrepancyAmount");

  if (expected === null && posted === null && discrepancy === null) return null;

  return (
    <div className="border-border bg-card mt-2 flex flex-col gap-1 rounded-md border px-2.5 py-2">
      {expected !== null && <AmountRow label="Expected" value={formatCurrency(expected)} />}
      {posted !== null && <AmountRow label="Posted" value={formatCurrency(posted)} />}
      {discrepancy !== null && (
        <AmountRow label="Discrepancy" value={formatCurrency(discrepancy)} emphasis />
      )}
    </div>
  );
}

function BillingExceptionDetails({ notification }: { notification: Notification }) {
  const missing = notificationDataNumber(notification, "missingRequirementCount");
  const failures = notificationDataNumber(notification, "validationFailureCount");

  if (!missing && !failures) return null;

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1">
      {missing !== null && missing > 0 && (
        <Badge variant="warning" className="text-2xs h-4.5">
          {missing} missing requirement{missing === 1 ? "" : "s"}
        </Badge>
      )}
      {failures !== null && failures > 0 && (
        <Badge variant="warning" className="text-2xs h-4.5">
          {failures} rate validation failure{failures === 1 ? "" : "s"}
        </Badge>
      )}
    </div>
  );
}

function BankReceiptDetails({ notification }: { notification: Notification }) {
  const amountMinor = notificationDataNumber(notification, "amountMinor");
  const reference = notificationDataString(notification, "referenceNumber");

  if (amountMinor === null && !reference) return null;

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1">
      {reference && (
        <Badge variant="outline" className="text-2xs h-4.5 font-mono">
          Ref {reference}
        </Badge>
      )}
      {amountMinor !== null && (
        <Badge variant="secondary" className="text-2xs h-4.5 tabular-nums">
          {formatCurrency(amountMinor / 100)}
        </Badge>
      )}
    </div>
  );
}

export function NotificationContent({
  notification,
  onNavigate,
}: {
  notification: Notification;
  onNavigate?: (link: string) => void;
}) {
  if (REPORT_ATTACHMENT_EVENT_TYPES.has(notification.eventType)) {
    return <ReportRunAttachment notification={notification} />;
  }
  if (notification.eventType.startsWith("tca.")) {
    return <TableChangeDetails notification={notification} />;
  }
  switch (notification.eventType) {
    case "shipment_comment_mention":
      return <MentionReply notification={notification} onNavigate={onNavigate} />;
    case "invoice_reconciliation_warning":
      return <ReconciliationDetails notification={notification} />;
    case "billing_exception_recorded":
      return <BillingExceptionDetails notification={notification} />;
    case "bank_receipt_reconciliation_exception":
      return <BankReceiptDetails notification={notification} />;
    default:
      return null;
  }
}
