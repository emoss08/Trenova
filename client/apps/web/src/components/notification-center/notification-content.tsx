import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { downloadReportRun } from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatCurrency, formatFileSize } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Notification } from "@trenova/shared/types/notification";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { Download01Icon } from "@trenova/shared/components/icons";
import { MentionQuote, MentionReply } from "./notification-mention-reply";
import {
  notificationDataBoolean,
  notificationDataNumber,
  notificationDataString,
} from "./notification-registry";

const REPORT_ATTACHMENT_EVENT_TYPES = new Set(["report_run_completed", "report_run_delivered"]);

function ReportRunAttachment({ notification }: { notification: Notification }) {
  const t = useT();

  const { allowed: canExport } = usePermission(Resource.Report, Operation.Export);
  const now = useNow();

  const runId = notificationDataString(notification, "runId");
  if (!runId) return null;

  const name = notificationDataString(notification, "reportName") ?? t("Report");
  const format = notificationDataString(notification, "format");
  const byteSize = notificationDataNumber(notification, "byteSize");
  const rowCount = notificationDataNumber(notification, "rowCount");
  const truncated = notificationDataBoolean(notification, "truncated");
  const expiresAt = notificationDataNumber(notification, "artifactExpiresAt");
  const expired = expiresAt !== null && expiresAt <= now;

  const meta: string[] = [];
  if (format) meta.push(format.toUpperCase());
  if (byteSize !== null && byteSize > 0) meta.push(formatFileSize(byteSize));
  if (rowCount !== null) meta.push(t("{0, plural, one {# row} other {# rows}}", rowCount));
  if (expiresAt !== null) {
    meta.push(
      expired
        ? t("Expired — run the report again")
        : t("Expires {0}", formatRelativeTime(expiresAt - now)),
    );
  }

  return (
    <div className="mt-2 flex items-center gap-2.5 rounded-md border border-border bg-card px-2.5 py-2">
      <DocumentFileTypeIcon fileName={format ? `${name}.${format}` : name} size="sm" />

      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-xs font-medium text-foreground">{name}</span>
        {meta.length > 0 && (
          <span className="truncate text-2xs text-muted-foreground">{meta.join(" · ")}</span>
        )}
      </div>
      {truncated && (
        <Badge variant="warning" className="h-4.5 shrink-0 text-2xs">
          {t("Truncated")}
        </Badge>
      )}
      {canExport && !expired && (
        <Button
          variant="secondary"
          size="icon-xs"
          className="shrink-0"
          aria-label={t("Download {0}", name)}
          onClick={(event) => {
            event.stopPropagation();
            downloadReportRun({ id: runId });
          }}
        >
          <Download01Icon className="size-3" />
        </Button>
      )}
    </div>
  );
}

function TableChangeDetails({ notification }: { notification: Notification }) {
  const t = useT();

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
        <Badge variant="info" className="h-4.5 text-2xs uppercase">
          {operation}
        </Badge>
      )}
      {tableName && (
        <Badge variant="neutral" appearance="outline" className="h-4.5 font-mono text-2xs">
          {tableName}
        </Badge>
      )}
      {visibleFields.map((field) => (
        <Badge key={field} variant="neutral" className="h-4.5 font-mono text-2xs">
          {field}
        </Badge>
      ))}
      {hiddenCount > 0 && (
        <span className="text-2xs text-muted-foreground">{t("+{0} more", hiddenCount)}</span>
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
            ? "text-2xs font-medium text-warning tabular-nums"
            : "text-2xs font-medium text-foreground tabular-nums"
        }
      >
        {value}
      </span>
    </div>
  );
}

function ReconciliationDetails({ notification }: { notification: Notification }) {
  const t = useT();

  const expected = notificationDataNumber(notification, "expectedTotal");
  const posted = notificationDataNumber(notification, "invoiceTotal");
  const discrepancy = notificationDataNumber(notification, "discrepancyAmount");

  if (expected === null && posted === null && discrepancy === null) return null;

  return (
    <div className="mt-2 flex flex-col gap-1 rounded-md border border-border bg-card px-2.5 py-2">
      {expected !== null && <AmountRow label={t("Expected")} value={formatCurrency(expected)} />}
      {posted !== null && <AmountRow label={t("Posted")} value={formatCurrency(posted)} />}
      {discrepancy !== null && (
        <AmountRow label={t("Discrepancy")} value={formatCurrency(discrepancy)} emphasis />
      )}
    </div>
  );
}

function BillingExceptionDetails({ notification }: { notification: Notification }) {
  const t = useT();

  const missing = notificationDataNumber(notification, "missingRequirementCount");
  const failures = notificationDataNumber(notification, "validationFailureCount");

  if (!missing && !failures) return null;

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1">
      {missing !== null && missing > 0 && (
        <Badge variant="warning" className="h-4.5 text-2xs">
          {t("{0, plural, one {# missing requirement} other {# missing requirements}}", missing)}
        </Badge>
      )}
      {failures !== null && failures > 0 && (
        <Badge variant="warning" className="h-4.5 text-2xs">
          {t(
            "{0, plural, one {# rate validation failure} other {# rate validation failures}}",
            failures,
          )}
        </Badge>
      )}
    </div>
  );
}

function BankReceiptDetails({ notification }: { notification: Notification }) {
  const t = useT();

  const amountMinor = notificationDataNumber(notification, "amountMinor");
  const reference = notificationDataString(notification, "referenceNumber");

  if (amountMinor === null && !reference) return null;

  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-1">
      {reference && (
        <Badge variant="neutral" appearance="outline" className="h-4.5 font-mono text-2xs">
          {t("Ref {0}", reference)}
        </Badge>
      )}
      {amountMinor !== null && (
        <Badge variant="neutral" className="h-4.5 text-2xs tabular-nums">
          {formatCurrency(amountMinor / 100)}
        </Badge>
      )}
    </div>
  );
}

const MENTION_EVENT_TYPE = "shipment_comment_mention";

/** What a kind of notification shows under its message: an attachment, figures, a quote. */
export function NotificationContent({ notification }: { notification: Notification }) {
  if (REPORT_ATTACHMENT_EVENT_TYPES.has(notification.eventType)) {
    return <ReportRunAttachment notification={notification} />;
  }
  if (notification.eventType.startsWith("tca.")) {
    return <TableChangeDetails notification={notification} />;
  }
  switch (notification.eventType) {
    case MENTION_EVENT_TYPE:
      return <MentionQuote notification={notification} />;
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

/** What a kind of notification lets the person write back from the row, under its actions. */
export function NotificationReply({
  notification,
  onReplied,
}: {
  notification: Notification;
  /** Called when the person answered from the row. */
  onReplied: () => void;
}) {
  return notification.eventType === MENTION_EVENT_TYPE ? (
    <MentionReply notification={notification} onReplied={onReplied} />
  ) : null;
}
