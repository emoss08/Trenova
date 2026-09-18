import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  unapplyCreditMemoApplication,
  type InvoiceArContext,
  type InvoiceCreditApplication,
  type InvoicePaymentApplicationRow,
} from "@/lib/graphql/invoice";
import { invoicePanelPath } from "@/lib/invoice-links";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { useQueryClient } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { PlainCustomerPaymentStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
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
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { HandCoinsIcon, ReceiptTextIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { ApplyCreditMemoDialog } from "./apply-credit-memo-dialog";

function formatDate(unix: number | null | undefined): string {
  return unix ? formatUnixDateMedium(unix, { fallback: "—" }) : "—";
}

/**
 * The cash side of an invoice: what is owed, every payment and credit that
 * settled part of it, and the way to record the next one. A credit memo reads
 * the other way round, listing the invoices its credit was applied to.
 */
export function InvoicePaymentsTab({
  invoice,
  arContext,
  isLoading,
}: {
  invoice: Invoice;
  arContext: InvoiceArContext | null | undefined;
  isLoading: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRecordPayment } = usePermission(Resource.CustomerPayment, Operation.Create);
  const { allowed: canUnapply } = usePermission(Resource.CustomerPayment, Operation.Update);
  const [applyOpen, setApplyOpen] = useState(false);

  const unapply = useApiMutation({
    resourceName: "credit memo application",
    mutationFn: (applicationId: string) =>
      unapplyCreditMemoApplication({ applicationId, reason: null }),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("Credit unapplied"));
    },
  });

  if (isLoading && !arContext) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const isCreditMemo = invoice.billType === "CreditMemo";
  const openBalance = Number(arContext?.openBalance ?? 0);
  const applied = Number(arContext?.appliedAmount ?? invoice.appliedAmount ?? 0);
  const creditRemaining = Number(arContext?.creditRemaining ?? 0);
  const daysPastDue = arContext?.daysPastDue ?? null;
  const isOpen = invoice.status === "Posted" && !isCreditMemo && openBalance > 0;
  const payments = arContext?.paymentApplications ?? [];
  const credits = arContext?.creditApplications ?? [];
  const canApplyCredit =
    isCreditMemo && invoice.status === "Posted" && creditRemaining > 0 && canUnapply;

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-4 px-4 py-3">
        <div className="grid grid-cols-2 gap-2 md:grid-cols-4" data-testid="invoice-payment-totals">
          <TotalCell
            label={t("Total")}
            value={formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
          />
          <TotalCell label={t("Applied")} value={formatCurrency(applied, invoice.currencyCode)} />
          {isCreditMemo ? (
            <TotalCell
              label={t("Credit remaining")}
              value={formatCurrency(creditRemaining, invoice.currencyCode)}
              tone={creditRemaining > 0 ? "positive" : undefined}
            />
          ) : (
            <TotalCell
              label={t("Open balance")}
              value={formatCurrency(openBalance, invoice.currencyCode)}
              tone={openBalance > 0 ? "negative" : "positive"}
            />
          )}
          <TotalCell
            label={t("Aging")}
            value={
              daysPastDue !== null && daysPastDue > 0
                ? t("{0} days past due", daysPastDue)
                : isOpen
                  ? t("Current")
                  : t("Settled")
            }
            tone={daysPastDue !== null && daysPastDue > 0 ? "negative" : undefined}
          />
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-sm font-medium">
            {isCreditMemo ? t("Applied to invoices") : t("Applications")}
          </p>
          <div className="flex items-center gap-2">
            {canApplyCredit ? (
              <Button size="sm" variant="outline" onClick={() => setApplyOpen(true)}>
                <ReceiptTextIcon className="size-3.5" />
                {t("Apply to invoices")}
              </Button>
            ) : null}
            {canRecordPayment && isOpen ? (
              <Link
                to={`/accounting/ar/payments?panelType=create&customerId=${invoice.customerId}&invoiceIds=${invoice.id}`}
                className={cn(buttonVariants({ size: "sm" }))}
              >
                <HandCoinsIcon className="size-3.5" />
                {t("Record payment")}
              </Link>
            ) : null}
          </div>
        </div>

        {payments.length === 0 && credits.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm">
            {isCreditMemo ? t("Not applied to any invoice yet") : t("No payments applied yet")}
          </p>
        ) : null}

        {payments.length > 0 ? (
          <PaymentApplicationsTable rows={payments} currencyCode={invoice.currencyCode} />
        ) : null}

        {credits.length > 0 ? (
          <CreditApplicationsTable
            rows={credits}
            perspective={isCreditMemo ? "memo" : "invoice"}
            currencyCode={invoice.currencyCode}
            canUnapply={canUnapply}
            pendingId={unapply.isPending ? (unapply.variables ?? null) : null}
            onUnapply={(id) => unapply.mutate(id)}
          />
        ) : null}
      </div>
      {isCreditMemo ? (
        <ApplyCreditMemoDialog
          creditMemo={invoice}
          creditRemaining={creditRemaining}
          open={applyOpen}
          onOpenChange={setApplyOpen}
        />
      ) : null}
    </ScrollArea>
  );
}

function TotalCell({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "positive" | "negative";
}) {
  return (
    <div className="bg-card rounded-md border p-3">
      <p className="text-muted-foreground text-xs">{label}</p>
      <p
        className={cn(
          "mt-1 text-base font-semibold tabular-nums",
          tone === "negative" && "text-danger-foreground",
          tone === "positive" && "text-success-foreground",
        )}
      >
        {value}
      </p>
    </div>
  );
}

function PaymentApplicationsTable({
  rows,
  currencyCode,
}: {
  rows: InvoicePaymentApplicationRow[];
  currencyCode: string;
}) {
  const t = useT();

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("Payment")}</TableHead>
            <TableHead>{t("Date")}</TableHead>
            <TableHead>{t("Method")}</TableHead>
            <TableHead className="text-right">{t("Applied")}</TableHead>
            <TableHead className="text-right">{t("Short pay")}</TableHead>
            <TableHead>{t("Status")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.id}>
              <TableCell>
                <Link
                  to={`/accounting/ar/payments?panelType=edit&panelEntityId=${row.customerPaymentId}`}
                  className="font-mono text-xs font-medium hover:underline"
                >
                  {row.payment?.referenceNumber || row.customerPaymentId}
                </Link>
              </TableCell>
              <TableCell className="text-xs">{formatDate(row.payment?.paymentDate)}</TableCell>
              <TableCell className="text-xs">{row.payment?.paymentMethod ?? "—"}</TableCell>
              <TableCell className="text-right text-xs">
                <AmountDisplay value={row.appliedAmountMinor} currency={currencyCode} />
              </TableCell>
              <TableCell className="text-right text-xs">
                {row.shortPayAmountMinor > 0 ? (
                  <AmountDisplay
                    value={row.shortPayAmountMinor}
                    currency={currencyCode}
                    variant="negative"
                  />
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </TableCell>
              <TableCell>
                {row.payment ? (
                  <PlainCustomerPaymentStatusBadge status={row.payment.status} />
                ) : (
                  <span className="text-muted-foreground text-xs">—</span>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function CreditApplicationsTable({
  rows,
  perspective,
  currencyCode,
  canUnapply,
  pendingId,
  onUnapply,
}: {
  rows: InvoiceCreditApplication[];
  perspective: "invoice" | "memo";
  currencyCode: string;
  canUnapply: boolean;
  pendingId: string | null;
  onUnapply: (applicationId: string) => void;
}) {
  const t = useT();

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{perspective === "memo" ? t("Invoice") : t("Credit memo")}</TableHead>
            <TableHead>{t("Accounting date")}</TableHead>
            <TableHead className="text-right">{t("Applied")}</TableHead>
            <TableHead>{t("Status")}</TableHead>
            {canUnapply ? <TableHead className="w-24" /> : null}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => {
            const counterpart = perspective === "memo" ? row.invoice : row.creditMemo;
            const counterpartId = perspective === "memo" ? row.invoiceId : row.creditMemoInvoiceId;
            return (
              <TableRow key={row.id} className={cn(row.status === "Unapplied" && "opacity-60")}>
                <TableCell>
                  <Link
                    to={invoicePanelPath(counterpartId)}
                    className="font-mono text-xs font-medium hover:underline"
                  >
                    {counterpart?.number ?? counterpartId}
                  </Link>
                </TableCell>
                <TableCell className="text-xs">{formatDate(row.accountingDate)}</TableCell>
                <TableCell className="text-right text-xs">
                  <AmountDisplay value={row.appliedAmountMinor} currency={currencyCode} />
                </TableCell>
                <TableCell>
                  <Badge variant={row.status === "Applied" ? "success" : "neutral"}>
                    {row.status === "Applied" ? t("Applied") : t("Unapplied")}
                  </Badge>
                </TableCell>
                {canUnapply ? (
                  <TableCell className="text-right">
                    {row.status === "Applied" ? (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-7 text-xs"
                        disabled={pendingId === row.id}
                        onClick={() => onUnapply(row.id)}
                      >
                        {t("Unapply")}
                      </Button>
                    ) : null}
                  </TableCell>
                ) : null}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
