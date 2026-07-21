import { AmountDisplay } from "@/components/accounting/amount-display";
import {
  formatAccountingDate,
  JournalEntryPostingCard,
  type PostingEntry,
} from "@/components/accounting/journal-entry-posting-card";
import { ComponentLoader } from "@/components/component-loader";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  PlainCustomerPaymentStatusBadge,
  PlainSettlementStatusBadge,
} from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { getTodayDate } from "@/lib/date";
import type { CustomerPaymentDetail } from "@/lib/graphql/customer-payment";
import { reverseCustomerPayment } from "@/lib/graphql/customer-payment";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import type { CustomerPaymentStatus } from "@/types/customer-payment";
import type { SettlementStatus } from "@/types/invoice";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CopyIcon,
  ExternalLinkIcon,
  HandCoinsIcon,
  Undo2Icon,
} from "lucide-react";
import { m } from "motion/react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { ApplyUnappliedForm } from "./apply-unapplied-form";

export function PaymentDetail({
  paymentId,
  onClose,
}: {
  paymentId: string;
  onClose: () => void;
}) {
  const [view, setView] = useState<"detail" | "apply">("detail");
  const { data: payment, isLoading } = useQuery(queries.customerPayment.detail(paymentId));

  if (isLoading || !payment) {
    return <ComponentLoader message="Loading payment..." />;
  }

  if (view === "apply") {
    return (
      <ApplyUnappliedForm
        payment={payment}
        onBack={() => setView("detail")}
        onDone={onClose}
      />
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
            <span className="text-2xl font-semibold tracking-tight tabular-nums">
              {formatCurrency(payment.amountMinor / 100)}
            </span>
            <PlainCustomerPaymentStatusBadge
              status={payment.status as CustomerPaymentStatus}
            />
          </div>
          <Link
            to={`/accounting/ar/customer-ledger?customerId=${payment.customerId}`}
            className="mt-1 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground hover:underline"
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
              Apply Unapplied
            </Button>
          ) : null}
          {canManage && isPosted ? <ReversePaymentButton payment={payment} /> : null}
        </div>
      </div>

      {isReversed ? (
        <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2.5 dark:border-red-900 dark:bg-red-950">
          <p className="text-xs font-medium text-red-700 dark:text-red-300">
            Reversed {formatAccountingDate(payment.reversedAt)} — cash was backed out and the
            applied invoices were reopened.
          </p>
          {payment.reversalReason ? (
            <p className="mt-0.5 text-xs text-red-600/90 dark:text-red-400/90">
              {payment.reversalReason}
            </p>
          ) : null}
        </div>
      ) : null}

      <CashAllocationBar
        amountMinor={payment.amountMinor}
        appliedMinor={payment.appliedAmountMinor}
        unappliedMinor={payment.unappliedAmountMinor}
        shortPayMinor={shortPayMinor}
      />

      <div className="grid grid-cols-2 gap-x-6 gap-y-2.5 rounded-md border bg-muted/30 p-3 text-xs md:grid-cols-3">
        <DetailItem label="Payment date" value={formatAccountingDate(payment.paymentDate)} />
        <DetailItem
          label="Accounting date"
          value={formatAccountingDate(payment.accountingDate)}
        />
        <DetailItem label="Method" value={payment.paymentMethod} />
        <DetailItem label="Reference" value={payment.referenceNumber || "—"} />
        <DetailItem label="Currency" value={payment.currencyCode} />
        <DetailItem label="Recorded" value={formatAccountingDate(payment.createdAt)} />
        {payment.memo ? (
          <div className="col-span-2 md:col-span-3">
            <DetailItem label="Memo" value={payment.memo} />
          </div>
        ) : null}
      </div>

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
  if (amountMinor <= 0) return null;
  const appliedShare = (appliedMinor / amountMinor) * 100;
  const unappliedShare = (unappliedMinor / amountMinor) * 100;

  return (
    <div>
      <div className="flex h-2.5 w-full gap-px overflow-hidden rounded-full bg-muted">
        {appliedMinor > 0 ? (
          <m.div
            className="h-full bg-emerald-500"
            initial={{ width: 0 }}
            animate={{ width: `${appliedShare}%` }}
            transition={{ duration: 0.5, ease: "easeOut" }}
          />
        ) : null}
        {unappliedMinor > 0 ? (
          <m.div
            className="h-full bg-sky-500"
            initial={{ width: 0 }}
            animate={{ width: `${unappliedShare}%` }}
            transition={{ duration: 0.5, delay: 0.05, ease: "easeOut" }}
          />
        ) : null}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
        <span className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <span className="size-2 rounded-full bg-emerald-500" />
          Applied · {formatCurrency(appliedMinor / 100)}
        </span>
        <span className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <span className="size-2 rounded-full bg-sky-500" />
          Unapplied · {formatCurrency(unappliedMinor / 100)}
        </span>
        {shortPayMinor > 0 ? (
          <span className="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground">
            <span className="size-2 rounded-full bg-amber-500" />
            Short-pay written off · {formatCurrency(shortPayMinor / 100)}
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
  const applications = payment.applications ?? [];

  return (
    <div>
      <p className="mb-2 text-sm font-medium">
        Applications
        <span className="ml-1.5 text-xs font-normal text-muted-foreground">
          {applications.length} {applications.length === 1 ? "invoice" : "invoices"}
        </span>
      </p>
      {applications.length === 0 ? (
        <div className="flex h-24 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground">
          Nothing applied — the full amount is unapplied cash
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="bg-muted/50">
              <TableRow className="hover:bg-transparent">
                <TableHead className="h-8 text-xs">Invoice</TableHead>
                <TableHead className="h-8 text-xs">Due</TableHead>
                <TableHead className="h-8 text-right text-xs">Invoice Total</TableHead>
                <TableHead className="h-8 text-right text-xs">Applied</TableHead>
                <TableHead className="h-8 text-right text-xs">Short-pay</TableHead>
                <TableHead className="h-8 text-xs">Settlement</TableHead>
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
                        <span className="text-[11px] text-muted-foreground">
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
                      <span className="text-xs text-muted-foreground">—</span>
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
                        className="text-xs text-amber-600 dark:text-amber-400"
                      />
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="py-2">
                    {application.invoice ? (
                      <PlainSettlementStatusBadge
                        status={application.invoice.settlementStatus as SettlementStatus}
                      />
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              <TableRow className="border-t bg-muted/30 font-medium hover:bg-muted/30">
                <TableCell colSpan={3} className="py-2 text-right text-xs">
                  Totals
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
                      className="text-xs font-semibold text-amber-600 dark:text-amber-400"
                    />
                  ) : (
                    <span className="text-xs text-muted-foreground">—</span>
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
  const { data: entries, isLoading } = useQuery(
    queries.journalEntry.bySource("CustomerPayment", paymentId),
  );
  const postings = (entries ?? []) as PostingEntry[];

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <p className="text-sm font-medium">
          GL Postings
          {postings.length > 0 ? (
            <span className="ml-1.5 text-xs font-normal text-muted-foreground">
              {postings.length} {postings.length === 1 ? "entry" : "entries"}
            </span>
          ) : null}
        </p>
        <Link
          to={`/accounting/journal-entries/source/CustomerPayment/${paymentId}`}
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground hover:underline"
        >
          Open full view
          <ExternalLinkIcon className="size-3" />
        </Link>
      </div>
      {isLoading ? (
        <Skeleton className="h-24 w-full rounded-md" />
      ) : postings.length === 0 ? (
        <div className="flex h-20 items-center justify-center rounded-md border border-dashed text-sm text-muted-foreground">
          Nothing has been posted to the ledger yet
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
  const [copied, setCopied] = useState(false);

  return (
    <Button
      size="icon-sm"
      variant="ghost"
      title="Copy payment ID"
      onClick={() => {
        void navigator.clipboard.writeText(id);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
    >
      {copied ? (
        <CheckIcon className="size-3.5 text-emerald-600 dark:text-emerald-400" />
      ) : (
        <CopyIcon className="size-3.5" />
      )}
    </Button>
  );
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-muted-foreground">{label}</p>
      <p className="mt-0.5 font-medium tabular-nums">{value}</p>
    </div>
  );
}

type ReversePaymentFormValues = {
  accountingDate: number;
  reason: string;
};

function ReversePaymentButton({ payment }: { payment: CustomerPaymentDetail }) {
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
      toast.success("Payment reversed", {
        description: "The cash receipt and invoice applications were backed out.",
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
        Reverse
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
            <DialogTitle>Reverse payment</DialogTitle>
            <DialogDescription>
              This backs out {formatCurrency(payment.amountMinor / 100)} of cash, reopens the
              applied invoices, and posts a reversing GL entry. This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <AutoCompleteDateField
              control={control}
              name="accountingDate"
              label="Accounting Date"
              rules={{ required: true }}
            />
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              placeholder="NSF check, posted to wrong customer, ..."
              rows={3}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleSubmit(onSubmit)()}
              isLoading={isPending}
            >
              Reverse Payment
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
