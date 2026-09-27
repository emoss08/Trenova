import { useT } from "@trenova/shared/i18n/use-t";
import { KPI_VALUE_LG_CLASS } from "@/components/kpi/kpi-strip";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  formatAccountingDate,
  JournalEntryPostingCard,
  type PostingEntry,
} from "@/components/accounting/journal-entry-posting-card";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  PlainCustomerPaymentStatusBadge,
  PlainSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { AccountingSyncStateLine } from "@/components/accounting-sync/sync-state-line";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import { formatExchangeRate } from "@/components/accounting/exchange-rate-line";
import type { CustomerPaymentDetail } from "@/lib/graphql/customer-payment";
import { reverseCustomerPayment } from "@/lib/graphql/customer-payment";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { CustomerPaymentStatus } from "@trenova/shared/types/customer-payment";
import type { SettlementStatus } from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon, ExternalLinkIcon, HandCoinsIcon, Undo2Icon } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { ApplyUnappliedForm } from "./apply-unapplied-form";

export function PaymentDetail({ paymentId, onClose }: { paymentId: string; onClose: () => void }) {
  const t = useT();

  const [view, setView] = useState<"detail" | "apply">("detail");
  const { data: payment, isLoading } = useQuery(queries.customerPayment.detail(paymentId));

  if (isLoading || !payment) {
    return <ComponentLoader message={t("Loading payment...")} />;
  }

  if (view === "apply") {
    return (
      <ApplyUnappliedForm payment={payment} onBack={() => setView("detail")} onDone={onClose} />
    );
  }

  return <PaymentDetailView payment={payment} onApplyUnapplied={() => setView("apply")} />;
}

function PaymentDetailView({
  payment,
  onApplyUnapplied,
}: {
  payment: CustomerPaymentDetail;
  onApplyUnapplied: () => void;
}) {
  const t = useT();

  const { allowed: canManage } = usePermission(Resource.CustomerPayment, Operation.Update);
  const isPosted = payment.status === "Posted";
  const isReversed = payment.status === "Reversed";
  const shortPayMinor = (payment.applications ?? []).reduce(
    (sum, application) => sum + application.shortPayAmountMinor,
    0,
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className={KPI_VALUE_LG_CLASS}>{formatCurrency(payment.amountMinor / 100)}</span>
            <PlainCustomerPaymentStatusBadge status={payment.status as CustomerPaymentStatus} />
          </div>
          <AccountingSyncStateLine objectId={payment.id} className="mt-1" />
          <Link
            to={`/accounting/ar/customer-ledger?customerId=${payment.customerId}`}
            className="text-muted-foreground hover:text-foreground mt-1 inline-flex items-center gap-1 text-xs hover:underline"
          >
            {payment.customer
              ? `${payment.customer.code} — ${payment.customer.name}`
              : payment.customerId}
            <ExternalLinkIcon className="size-3" />
          </Link>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <CopyIdButton id={payment.id} />
          {canManage && isPosted && payment.unappliedAmountMinor > 0 ? (
            <Button size="sm" variant="outline" onClick={onApplyUnapplied}>
              <HandCoinsIcon className="size-4" />
              {t("Apply unapplied")}
            </Button>
          ) : null}
          {canManage && isPosted ? <ReversePaymentButton payment={payment} /> : null}
        </div>
      </div>

      {isReversed ? (
        <Alert variant="destructive" size="sm">
          <Undo2Icon />
          <AlertTitle>
            {t(
              "Reversed {0} — cash was backed out and the applied invoices were reopened.",
              formatAccountingDate(payment.reversedAt),
            )}
          </AlertTitle>
          {payment.reversalReason ? (
            <AlertDescription>{payment.reversalReason}</AlertDescription>
          ) : null}
        </Alert>
      ) : null}

      <CashAllocationBar
        amountMinor={payment.amountMinor}
        appliedMinor={payment.appliedAmountMinor}
        unappliedMinor={payment.unappliedAmountMinor}
        shortPayMinor={shortPayMinor}
      />

      <DescriptionList columns={3} className="bg-card rounded-lg border p-3">
        <DescriptionItem label={t("Payment date")} numeric>
          {formatAccountingDate(payment.paymentDate)}
        </DescriptionItem>
        <DescriptionItem label={t("Accounting date")} numeric>
          {formatAccountingDate(payment.accountingDate)}
        </DescriptionItem>
        <DescriptionItem label={t("Method")}>{payment.paymentMethod}</DescriptionItem>
        <DescriptionItem label={t("Reference")} numeric>
          {payment.referenceNumber || <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Currency")}>{payment.currencyCode}</DescriptionItem>
        {payment.exchangeRate ? (
          <DescriptionItem label={t("Exchange rate")} numeric>
            {t(
              "1 {0} = {1}, quoted {2}",
              payment.currencyCode,
              formatExchangeRate(payment.exchangeRate),
              formatUnixDateMedium(payment.exchangeRateDate),
            )}
          </DescriptionItem>
        ) : null}
        <DescriptionItem label={t("Recorded")} numeric>
          {formatAccountingDate(payment.createdAt)}
        </DescriptionItem>
        {payment.memo ? (
          <DescriptionItem label={t("Memo")} span="full">
            {payment.memo}
          </DescriptionItem>
        ) : null}
      </DescriptionList>

      <ApplicationsSection payment={payment} shortPayMinor={shortPayMinor} />

      <GLActivitySection paymentId={payment.id} />
    </div>
  );
}

function CashAllocationBar({
  amountMinor,
  appliedMinor,
  unappliedMinor,
  shortPayMinor,
}: {
  amountMinor: number;
  appliedMinor: number;
  unappliedMinor: number;
  shortPayMinor: number;
}) {
  const t = useT();

  if (amountMinor <= 0) return null;
  const appliedShare = (appliedMinor / amountMinor) * 100;
  const unappliedShare = (unappliedMinor / amountMinor) * 100;

  return (
    <div>
      <div className="bg-muted flex h-2.5 w-full gap-px overflow-hidden rounded-full">
        {appliedMinor > 0 ? (
          <div className="h-full bg-success" style={{ width: `${appliedShare}%` }} />
        ) : null}
        {unappliedMinor > 0 ? (
          <div className="h-full bg-accent-sky" style={{ width: `${unappliedShare}%` }} />
        ) : null}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
        <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
          <span className="size-2 rounded-full bg-success" />
          {t("Applied · {0}", formatCurrency(appliedMinor / 100))}
        </span>
        <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
          <span className="size-2 rounded-full bg-accent-sky" />
          {t("Unapplied · {0}", formatCurrency(unappliedMinor / 100))}
        </span>
        {shortPayMinor > 0 ? (
          <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
            <span className="size-2 rounded-full bg-warning" />
            {t("Short-pay written off · {0}", formatCurrency(shortPayMinor / 100))}
          </span>
        ) : null}
      </div>
    </div>
  );
}

function ApplicationsSection({
  payment,
  shortPayMinor,
}: {
  payment: CustomerPaymentDetail;
  shortPayMinor: number;
}) {
  const t = useT();

  const applications = payment.applications ?? [];

  return (
    <div>
      <p className="mb-2 text-sm font-medium">
        {t("Applications")}
        <span className="text-muted-foreground ml-1.5 text-xs font-normal">
          {applications.length} {applications.length === 1 ? "invoice" : "invoices"}
        </span>
      </p>
      {applications.length === 0 ? (
        <div className="text-muted-foreground flex h-24 items-center justify-center rounded-md border border-dashed text-sm">
          {t("Nothing applied — the full amount is unapplied cash")}
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="bg-muted/50">
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-8 text-xs">{t("Invoice")}</TableHead>
                <TableHead className="h-8 text-xs">{t("Due")}</TableHead>
                <TableHead className="h-8 text-right text-xs">{t("Invoice total")}</TableHead>
                <TableHead className="h-8 text-right text-xs">{t("Applied")}</TableHead>
                <TableHead className="h-8 text-right text-xs">{t("Short-pay")}</TableHead>
                <TableHead className="h-8 text-xs">{t("Settlement")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {applications.map((application) => (
                <TableRow key={application.id} className="transition-colors">
                  <TableCell className="py-2">
                    <div className="flex flex-col">
                      {application.invoice ? (
                        <Link
                          to={`/billing/invoices?item=${application.invoiceId}`}
                          className="font-mono text-xs font-medium hover:underline"
                        >
                          {application.invoice.number}
                        </Link>
                      ) : (
                        <span className="font-mono text-xs font-medium">
                          {application.invoiceId}
                        </span>
                      )}
                      {application.invoice ? (
                        <span className="text-muted-foreground text-xs">
                          {application.invoice.billToName}
                        </span>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell className="py-2 text-xs">
                    {formatAccountingDate(application.invoice?.dueDate)}
                  </TableCell>
                  <TableCell className="py-2 text-right">
                    {application.invoice ? (
                      <span className="text-xs tabular-nums">
                        {formatCurrency(Number(application.invoice.totalAmount))}
                      </span>
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
                  </TableCell>
                  <TableCell className="py-2 text-right">
                    <AmountDisplay
                      value={application.appliedAmountMinor}
                      className="text-xs font-medium"
                    />
                  </TableCell>
                  <TableCell className="py-2 text-right">
                    {application.shortPayAmountMinor > 0 ? (
                      <AmountDisplay
                        value={application.shortPayAmountMinor}
                        className="text-xs text-warning-foreground"
                      />
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
                  </TableCell>
                  <TableCell className="py-2">
                    {application.invoice ? (
                      <PlainSettlementStatusBadge
                        status={application.invoice.settlementStatus as SettlementStatus}
                      />
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              <TableRow className="bg-muted/30 hover:bg-muted/30 border-t font-medium">
                <TableCell colSpan={3} className="py-2 text-right text-xs">
                  {t("Totals")}
                </TableCell>
                <TableCell className="py-2 text-right">
                  <AmountDisplay
                    value={payment.appliedAmountMinor}
                    className="text-xs font-semibold"
                  />
                </TableCell>
                <TableCell className="py-2 text-right">
                  {shortPayMinor > 0 ? (
                    <AmountDisplay
                      value={shortPayMinor}
                      className="text-xs font-semibold text-warning-foreground"
                    />
                  ) : (
                    <span className="text-muted-foreground text-xs">—</span>
                  )}
                </TableCell>
                <TableCell />
              </TableRow>
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function GLActivitySection({ paymentId }: { paymentId: string }) {
  const t = useT();

  const { data: entries, isLoading } = useQuery(
    queries.journalEntry.bySource("CustomerPayment", paymentId),
  );
  const postings = (entries ?? []) as PostingEntry[];

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <p className="text-sm font-medium">
          {t("GL postings")}
          {postings.length > 0 ? (
            <span className="text-muted-foreground ml-1.5 text-xs font-normal">
              {postings.length} {postings.length === 1 ? "entry" : "entries"}
            </span>
          ) : null}
        </p>
        <Link
          to={`/accounting/journal-entries/source/CustomerPayment/${paymentId}`}
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs hover:underline"
        >
          {t("Open full view")}
          <ExternalLinkIcon className="size-3" />
        </Link>
      </div>
      {isLoading ? (
        <Skeleton className="h-24 w-full rounded-md" />
      ) : postings.length === 0 ? (
        <div className="text-muted-foreground flex h-20 items-center justify-center rounded-md border border-dashed text-sm">
          {t("Nothing has been posted to the ledger yet")}
        </div>
      ) : (
        <div className="space-y-2">
          {postings.map((entry) => (
            <JournalEntryPostingCard
              key={entry.id}
              entry={entry}
              defaultOpen={postings.length === 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function CopyIdButton({ id }: { id: string }) {
  const t = useT();

  const [copied, setCopied] = useState(false);

  return (
    <Button
      size="icon-sm"
      variant="ghost"
      title={t("Copy payment ID")}
      onClick={() => {
        void navigator.clipboard.writeText(id);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
    >
      {copied ? (
        <CheckIcon className="size-3.5 text-success-foreground" />
      ) : (
        <CopyIcon className="size-3.5" />
      )}
    </Button>
  );
}

type ReversePaymentFormValues = {
  accountingDate: number;
  reason: string;
};

function ReversePaymentButton({ payment }: { payment: CustomerPaymentDetail }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);

  const form = useForm<ReversePaymentFormValues>({
    defaultValues: { accountingDate: getTodayDate(), reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (values: ReversePaymentFormValues) =>
      reverseCustomerPayment({
        paymentId: payment.id,
        accountingDate: values.accountingDate,
        reason: values.reason || undefined,
      }),
    onSuccess: () => {
      toast.success(t("Payment reversed"), {
        description: t("The cash receipt and invoice applications were backed out."),
      });
      void queryClient.invalidateQueries({ queryKey: ["customer-payment-list"] });
      void queryClient.invalidateQueries({
        queryKey: queries.customerPayment.detail(payment.id).queryKey,
      });
      void queryClient.invalidateQueries({ queryKey: queries.ar._def });
      void queryClient.invalidateQueries({
        queryKey: queries.journalEntry.bySource("CustomerPayment", payment.id).queryKey,
      });
      setOpen(false);
    },
    resourceName: "Customer Payment",
  });

  const onSubmit = async (values: ReversePaymentFormValues) => {
    await mutateAsync(values);
  };

  return (
    <>
      <Button size="sm" variant="destructive" onClick={() => setOpen(true)}>
        <Undo2Icon className="size-4" />
        {t("Reverse")}
      </Button>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) reset({ accountingDate: getTodayDate(), reason: "" });
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("Reverse payment")}</DialogTitle>
            <DialogDescription>
              {t(
                "This backs out {0} of cash, reopens the applied invoices, and posts a reversing GL entry. This cannot be undone.",
                formatCurrency(payment.amountMinor / 100),
              )}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <AutoCompleteDateField
              control={control}
              name="accountingDate"
              label={t("Accounting date")}
              rules={{ required: true }}
            />
            <TextareaField
              control={control}
              name="reason"
              label={t("Reason")}
              placeholder={t("NSF check, posted to wrong customer, ...")}
              rows={3}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>
              {t("Cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleSubmit(onSubmit)()}
              isLoading={isPending}
            >
              {t("Reverse payment")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
