import type {
  LateChargeAssessmentResult,
  LateChargeCustomerResult,
} from "@/lib/graphql/accounts-receivable";
import { invoicePanelPath } from "@/lib/invoice-links";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";

const COLUMNS = [
  { label: "Customer" },
  { label: "Invoice" },
  { label: "Period" },
  { label: "Basis", numeric: true },
  { label: "Rate", numeric: true },
  { label: "Charge", numeric: true },
] as const;

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

/**
 * One block per customer: the periods the run would charge, what the charge
 * comes to, and, once it has run, the memo it raised. Skipped customers keep
 * their row so the reason is visible rather than the customer simply absent.
 */
export function LateChargePreviewTable({
  result,
  selected,
  onSelectedChange,
  hasActiveFilters,
  onClearFilters,
}: {
  result: LateChargeAssessmentResult;
  selected: Record<string, boolean>;
  onSelectedChange: (next: Record<string, boolean>) => void;
  hasActiveFilters: boolean;
  onClearFilters: () => void;
}) {
  const t = useT();
  const customers = result.customers;
  const selectable = customers.filter((row) => !row.skipped && !row.debitMemoId);
  const allSelected = selectable.length > 0 && selectable.every((row) => selected[row.customerId]);

  if (customers.length === 0) {
    return (
      <EmptyTable
        title={hasActiveFilters ? "Nothing matches" : "Nothing to charge"}
        description={
          hasActiveFilters
            ? "No overdue invoice under that filter has an unassessed period. Widen it, or clear it to see every customer."
            : "Every overdue invoice has been assessed for every period that has begun, or no customer applies late charges."
        }
        columns={COLUMNS}
        onClearFilters={hasActiveFilters ? onClearFilters : undefined}
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-8">
              <Checkbox
                checked={allSelected}
                indeterminate={!allSelected && selectable.some((row) => selected[row.customerId])}
                onCheckedChange={(checked) =>
                  onSelectedChange(
                    checked === true
                      ? Object.fromEntries(selectable.map((row) => [row.customerId, true]))
                      : {},
                  )
                }
                aria-label={t("Select all")}
              />
            </TableHead>
            <TableHead>{t("Customer")}</TableHead>
            <TableHead>{t("Invoice")}</TableHead>
            <TableHead>{t("Period")}</TableHead>
            <TableHead className="text-right">{t("Basis")}</TableHead>
            <TableHead className="text-right">{t("Rate")}</TableHead>
            <TableHead className="text-right">{t("Charge")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {customers.map((customer) => (
            <CustomerRows
              key={customer.customerId}
              customer={customer}
              checked={Boolean(selected[customer.customerId])}
              onCheckedChange={(checked) =>
                onSelectedChange({ ...selected, [customer.customerId]: checked })
              }
            />
          ))}
        </TableBody>
      </Table>
      <div className="bg-muted/30 flex items-center justify-between border-t px-3 py-2 text-xs">
        <span className="text-muted-foreground">
          {result.preview
            ? t("Preview as of {0}", formatDate(result.asOfDate))
            : t(
                "Run as of {0}: {1, plural, one {# memo} other {# memos}} raised, {2} skipped",
                formatDate(result.asOfDate),
                result.memosCreated,
                result.customersSkipped,
              )}
        </span>
        <span className="font-semibold tabular-nums">
          <AmountDisplay value={result.totalChargeMinor} />
        </span>
      </div>
    </div>
  );
}

function CustomerRows({
  customer,
  checked,
  onCheckedChange,
}: {
  customer: LateChargeCustomerResult;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  const t = useT();
  const selectable = !customer.skipped && !customer.debitMemoId;

  return (
    <>
      <TableRow className={cn("bg-muted/20", customer.skipped && "opacity-70")}>
        <TableCell>
          {selectable ? (
            <Checkbox
              checked={checked}
              onCheckedChange={(next) => onCheckedChange(next === true)}
              aria-label={t("Select {0}", customer.customerName)}
            />
          ) : null}
        </TableCell>
        <TableCell colSpan={5}>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium">{customer.customerName}</span>
            {customer.debitMemoId ? (
              <Link
                to={invoicePanelPath(customer.debitMemoId)}
                className="font-mono text-xs font-medium hover:underline"
              >
                {customer.debitMemoNumber}
              </Link>
            ) : null}
            {customer.debitMemoId ? (
              <Badge variant={customer.posted ? "success" : "neutral"}>
                {customer.posted ? t("Posted") : t("Draft")}
              </Badge>
            ) : null}
            {customer.skipped ? (
              <span className="text-muted-foreground text-xs">{customer.skipReason}</span>
            ) : null}
          </div>
        </TableCell>
        <TableCell className="text-right text-sm font-semibold">
          <AmountDisplay value={customer.totalChargeMinor} currency={customer.currencyCode} />
        </TableCell>
      </TableRow>
      {customer.lines.map((line) => (
        <TableRow key={`${line.invoiceId}-${line.periodIndex}`}>
          <TableCell />
          <TableCell />
          <TableCell>
            <Link
              to={invoicePanelPath(line.invoiceId)}
              className="font-mono text-xs font-medium hover:underline"
            >
              {line.invoiceNumber}
            </Link>
          </TableCell>
          <TableCell className="text-xs">
            {t("Period {0}", line.periodIndex)} · {formatDate(line.periodStart)} –{" "}
            {formatDate(line.periodEnd)}
          </TableCell>
          <TableCell className="text-right text-xs">
            <AmountDisplay value={line.basisOpenBalanceMinor} currency={customer.currencyCode} />
          </TableCell>
          <TableCell className="text-right text-xs tabular-nums">{line.ratePercent}%</TableCell>
          <TableCell className="text-right text-xs">
            <AmountDisplay value={line.chargeMinor} currency={customer.currencyCode} />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}
