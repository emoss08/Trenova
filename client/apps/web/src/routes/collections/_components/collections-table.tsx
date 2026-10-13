import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  ACTIVITY_KIND_LABEL,
  PRIORITY_LABEL,
  PRIORITY_TONE,
  PROMISE_STATUS_LABEL,
  PROMISE_STATUS_TONE,
} from "@/lib/collection-labels";
import type { CollectionWorklistRow } from "@/types/collection";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";

/**
 * The worklist, ranked by the server: a broken promise or anything sixty days
 * late first, an open promise or a follow-up not yet due last. Each row is
 * one customer, opened into its account panel.
 */
export function CollectionsTable({
  rows,
  selectedCustomerId,
  onSelect,
}: {
  rows: CollectionWorklistRow[];
  selectedCustomerId: string | null;
  onSelect: (customerId: string) => void;
}) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Customer")}</TableHead>
          <TableHead className="text-right">{t("Overdue")}</TableHead>
          <TableHead className="text-right">{t("Oldest")}</TableHead>
          <TableHead>{t("Promise")}</TableHead>
          <TableHead>{t("Follow-up")}</TableHead>
          <TableHead>{t("Collector")}</TableHead>
          <TableHead>{t("Last contact")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow
            key={row.customerId}
            data-state={row.customerId === selectedCustomerId ? "selected" : undefined}
            className="cursor-pointer"
            onClick={() => onSelect(row.customerId)}
          >
            <TableCell>
              <div className="flex min-w-0 items-center gap-2">
                <Badge variant={PRIORITY_TONE[row.priority]} appearance="outline">
                  {PRIORITY_LABEL[row.priority]}
                </Badge>
                <div className="flex min-w-0 flex-col">
                  <button
                    type="button"
                    className="ui-focus-ring truncate rounded-control text-left font-medium"
                    onClick={(event) => {
                      event.stopPropagation();
                      onSelect(row.customerId);
                    }}
                  >
                    {row.customerName}
                  </button>
                  <span className="text-muted-foreground text-xs">
                    {row.customerCode}
                    {row.disputedInvoiceCount > 0
                      ? ` · ${t("{0, plural, one {# disputed} other {# disputed}}", row.disputedInvoiceCount)}`
                      : ""}
                    {row.creditStatus === "Hold" || row.creditStatus === "Suspended"
                      ? ` · ${t("Credit hold")}`
                      : ""}
                  </span>
                </div>
              </div>
            </TableCell>
            <TableCell className="text-right tabular-nums">
              <div className="flex flex-col items-end">
                <span>{formatMinor(row.overdueMinor, row.currencyCode)}</span>
                <span className="text-muted-foreground text-xs">
                  {t("{0} open", formatMinor(row.totalOpenMinor, row.currencyCode))}
                </span>
              </div>
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {row.oldestDaysPastDue > 0 ? t("{0}d", row.oldestDaysPastDue) : "—"}
            </TableCell>
            <TableCell>
              {row.promise ? (
                <div className="flex flex-col gap-0.5">
                  <Badge variant={PROMISE_STATUS_TONE[row.promise.status]}>
                    {PROMISE_STATUS_LABEL[row.promise.status]}
                  </Badge>
                  <span className="text-muted-foreground text-xs tabular-nums">
                    {t(
                      "{0} by {1}",
                      formatMinor(row.promise.amountMinor, row.promise.currencyCode),
                      formatUnixDateMedium(row.promise.promisedOn),
                    )}
                  </span>
                </div>
              ) : (
                <span className="text-muted-foreground">—</span>
              )}
            </TableCell>
            <TableCell>
              {row.nextFollowUpOn ? (
                <span className={cn("tabular-nums", row.followUpDue && "text-warning-foreground")}>
                  {row.followUpDue
                    ? t("Due {0}", formatUnixDateMedium(row.nextFollowUpOn))
                    : formatUnixDateMedium(row.nextFollowUpOn)}
                </span>
              ) : (
                <span className="text-muted-foreground">{t("Not scheduled")}</span>
              )}
            </TableCell>
            <TableCell>
              {row.collectorName || (
                <span className="text-muted-foreground">{t("Unassigned")}</span>
              )}
            </TableCell>
            <TableCell className="max-w-64">
              {row.lastActivityAt && row.lastActivityKind ? (
                <div className="flex min-w-0 flex-col">
                  <span className="text-xs">
                    {t(
                      "{0} · {1}",
                      ACTIVITY_KIND_LABEL[row.lastActivityKind],
                      formatDistanceToNowStrict(fromUnixTime(row.lastActivityAt), {
                        addSuffix: true,
                      }),
                    )}
                  </span>
                  {row.lastActivitySummary ? (
                    <span className="text-muted-foreground truncate text-xs">
                      {row.lastActivitySummary}
                    </span>
                  ) : null}
                </div>
              ) : (
                <span className="text-muted-foreground">{t("Never contacted")}</span>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
