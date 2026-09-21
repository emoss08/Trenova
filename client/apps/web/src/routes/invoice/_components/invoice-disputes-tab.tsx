import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { invoiceDisputeReasonCodeChoices, invoiceDisputeResolutionChoices } from "@/lib/choices";
import {
  openInvoiceDispute,
  resolveInvoiceDispute,
  withdrawInvoiceDispute,
  type InvoiceArContext,
  type InvoiceDisputeCase,
} from "@/lib/graphql/invoice";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { InvoiceDisputeCaseStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  DISPUTE_RESOLUTIONS_REQUIRING_ADJUSTMENT,
  openDisputeFormSchema,
  resolveDisputeFormSchema,
  type Invoice,
  type OpenDisputeFormValues,
  type ResolveDisputeFormValues,
} from "@trenova/shared/types/invoice";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ShieldAlertIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const OPEN_FORM_ID = "invoice-dispute-open-form";
const RESOLVE_FORM_ID = "invoice-dispute-resolve-form";

function reasonLabel(code: InvoiceDisputeCase["reasonCode"]): string {
  return invoiceDisputeReasonCodeChoices.find((choice) => choice.value === code)?.label ?? code;
}

function resolutionLabel(resolution: InvoiceDisputeCase["resolution"]): string | null {
  if (!resolution) return null;
  return (
    invoiceDisputeResolutionChoices.find((choice) => choice.value === resolution)?.label ??
    resolution
  );
}

function decimalString(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(4).replace(/0+$/, "");
}

/**
 * The dispute cases on an invoice. At most one is open at a time; while it is,
 * the invoice reads Disputed everywhere and the late-charge run leaves it
 * alone. Resolving records how it ended; withdrawing drops it.
 */
export function InvoiceDisputesTab({
  invoice,
  arContext,
  isLoading,
}: {
  invoice: Invoice;
  arContext: InvoiceArContext | null | undefined;
  isLoading: boolean;
}) {
  const t = useT();
  const { allowed: canOpen } = usePermission(Resource.InvoiceDispute, Operation.Create);
  const { allowed: canResolve } = usePermission(Resource.InvoiceDispute, Operation.Approve);
  const { allowed: canWithdraw } = usePermission(Resource.InvoiceDispute, Operation.Cancel);
  const [openDialog, setOpenDialog] = useState(false);
  const [resolving, setResolving] = useState<InvoiceDisputeCase | null>(null);
  const [withdrawing, setWithdrawing] = useState<InvoiceDisputeCase | null>(null);

  if (isLoading && !arContext) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  const disputes = arContext?.disputes ?? [];
  const openCase = arContext?.openDispute ?? null;
  const openBalance = Number(arContext?.openBalance ?? 0);
  const disputable =
    invoice.status === "Posted" &&
    (invoice.billType === "Invoice" || invoice.billType === "DebitMemo") &&
    openBalance > 0 &&
    !openCase;

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-4 px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <p className="text-sm font-medium">{t("Disputes")}</p>
            <p className="text-muted-foreground text-xs">
              {openCase
                ? t("A dispute is open; late charges pause until it closes.")
                : t("No open dispute on this invoice.")}
            </p>
          </div>
          {canOpen && disputable ? (
            <Button size="sm" variant="outline" onClick={() => setOpenDialog(true)}>
              <ShieldAlertIcon className="size-3.5" />
              {t("Open dispute")}
            </Button>
          ) : null}
        </div>

        {disputes.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-6 text-center text-sm">
            {t("No dispute has been raised on this invoice.")}
          </p>
        ) : (
          <div className="flex flex-col gap-2">
            {disputes.map((dispute) => (
              <div key={dispute.id} className="bg-card rounded-md border p-3">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <InvoiceDisputeCaseStatusBadge status={dispute.status} />
                      <span className="text-sm font-medium">
                        {t(reasonLabel(dispute.reasonCode))}
                      </span>
                    </div>
                    <p className="text-muted-foreground mt-1 text-xs">
                      {t("Opened {0}", formatUnixDateTime(dispute.openedAt))}
                      {dispute.resolvedAt
                        ? ` · ${t("Closed {0}", formatUnixDateTime(dispute.resolvedAt))}`
                        : ""}
                    </p>
                    {dispute.notes ? (
                      <p className="mt-2 text-xs whitespace-pre-line">{dispute.notes}</p>
                    ) : null}
                    {dispute.resolution ? (
                      <p className="mt-2 text-xs">
                        <span className="text-muted-foreground">{t("Resolution")}: </span>
                        {t(resolutionLabel(dispute.resolution) ?? "")}
                        {dispute.resolutionNotes ? ` — ${dispute.resolutionNotes}` : ""}
                      </p>
                    ) : null}
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-2">
                    <span className="text-sm font-semibold tabular-nums">
                      {formatCurrency(Number(dispute.disputedAmount ?? 0), invoice.currencyCode)}
                    </span>
                    {dispute.status === "Open" ? (
                      <div className="flex items-center gap-1.5">
                        {canResolve ? (
                          <Button
                            size="sm"
                            className="h-7 text-xs"
                            onClick={() => setResolving(dispute)}
                          >
                            {t("Resolve")}
                          </Button>
                        ) : null}
                        {canWithdraw ? (
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 text-xs"
                            onClick={() => setWithdrawing(dispute)}
                          >
                            {t("Withdraw")}
                          </Button>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <OpenDisputeDialog
        invoice={invoice}
        openBalance={openBalance}
        open={openDialog}
        onOpenChange={setOpenDialog}
      />
      {resolving ? (
        <ResolveDisputeDialog
          dispute={resolving}
          open
          onOpenChange={(next) => {
            if (!next) setResolving(null);
          }}
        />
      ) : null}
      {withdrawing ? (
        <WithdrawDisputeDialog
          dispute={withdrawing}
          open
          onOpenChange={(next) => {
            if (!next) setWithdrawing(null);
          }}
        />
      ) : null}
    </ScrollArea>
  );
}

function OpenDisputeDialog({
  invoice,
  openBalance,
  open,
  onOpenChange,
}: {
  invoice: Invoice;
  openBalance: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<OpenDisputeFormValues>({
    resolver: zodResolver(openDisputeFormSchema) as Resolver<OpenDisputeFormValues>,
    defaultValues: { reasonCode: undefined, disputedAmount: openBalance, notes: "" },
  });

  useEffect(() => {
    if (open) form.reset({ reasonCode: undefined, disputedAmount: openBalance, notes: "" });
  }, [open, openBalance, form]);

  const reasonOptions = useMemo(
    () =>
      invoiceDisputeReasonCodeChoices.map((choice) => ({
        value: choice.value,
        label: t(choice.label),
      })),
    [t],
  );

  const mutation = useApiMutation<
    InvoiceDisputeCase,
    OpenDisputeFormValues,
    unknown,
    OpenDisputeFormValues
  >({
    form,
    resourceName: "invoice dispute",
    mutationFn: (values) =>
      openInvoiceDispute({
        invoiceId: invoice.id,
        reasonCode: values.reasonCode,
        disputedAmount: decimalString(values.disputedAmount),
        notes: values.notes || null,
      }),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("Dispute opened on {0}", invoice.number));
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Open a dispute on {0}", invoice.number)}</DialogTitle>
          <DialogDescription>
            {t(
              "The invoice reads Disputed until the case is resolved or withdrawn, and no late charge is assessed on it meanwhile.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form id={OPEN_FORM_ID} onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
            <FormGroup cols={2}>
              <FormControl>
                <SelectField
                  control={form.control}
                  name="reasonCode"
                  label={t("Reason")}
                  placeholder={t("Select reason")}
                  options={reasonOptions}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={form.control}
                  name="disputedAmount"
                  label={t("Disputed amount")}
                  aria-label={t("Disputed amount")}
                  decimalScale={2}
                  fixedDecimalScale
                  description={t(
                    "At most the open balance of {0}",
                    formatCurrency(openBalance, invoice.currencyCode),
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField
                  control={form.control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("What the customer is contesting")}
                  rows={3}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={OPEN_FORM_ID} disabled={mutation.isPending}>
            {t("Open dispute")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ResolveDisputeDialog({
  dispute,
  open,
  onOpenChange,
}: {
  dispute: InvoiceDisputeCase;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<ResolveDisputeFormValues>({
    resolver: zodResolver(resolveDisputeFormSchema) as Resolver<ResolveDisputeFormValues>,
    defaultValues: { resolution: undefined, resolutionAdjustmentId: "", resolutionNotes: "" },
  });
  const resolution = useWatch({ control: form.control, name: "resolution" });
  const needsAdjustment = resolution
    ? DISPUTE_RESOLUTIONS_REQUIRING_ADJUSTMENT.includes(resolution)
    : false;

  const resolutionOptions = useMemo(
    () =>
      invoiceDisputeResolutionChoices.map((choice) => ({
        value: choice.value,
        label: t(choice.label),
      })),
    [t],
  );

  const mutation = useApiMutation<
    InvoiceDisputeCase,
    ResolveDisputeFormValues,
    unknown,
    ResolveDisputeFormValues
  >({
    form,
    resourceName: "invoice dispute",
    mutationFn: (values) =>
      resolveInvoiceDispute({
        disputeId: dispute.id,
        resolution: values.resolution,
        resolutionAdjustmentId: values.resolutionAdjustmentId || null,
        resolutionNotes: values.resolutionNotes || null,
      }),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("Dispute resolved"));
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Resolve dispute")}</DialogTitle>
          <DialogDescription>
            {t(
              "Record how the dispute ended. A credit or write-off must name the executed adjustment that settled it.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={RESOLVE_FORM_ID}
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <SelectField
                  control={form.control}
                  name="resolution"
                  label={t("Resolution")}
                  placeholder={t("Select resolution")}
                  options={resolutionOptions}
                />
              </FormControl>
              {needsAdjustment ? (
                <FormControl>
                  <InputField
                    control={form.control}
                    name="resolutionAdjustmentId"
                    label={t("Adjustment")}
                    placeholder={t("Executed adjustment ID on this invoice")}
                    description={t(
                      "Find it on the invoice's adjustment panel; it must have executed against this invoice.",
                    )}
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <TextareaField
                  control={form.control}
                  name="resolutionNotes"
                  label={t("Resolution notes")}
                  rows={3}
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={RESOLVE_FORM_ID} disabled={mutation.isPending}>
            {t("Resolve dispute")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function WithdrawDisputeDialog({
  dispute,
  open,
  onOpenChange,
}: {
  dispute: InvoiceDisputeCase;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [notes, setNotes] = useState("");

  const mutation = useApiMutation({
    resourceName: "invoice dispute",
    mutationFn: () =>
      withdrawInvoiceDispute({ disputeId: dispute.id, notes: notes.trim() || null }),
    onSuccess: () => {
      invalidateInvoiceQueries(queryClient);
      toast.success(t("Dispute withdrawn"));
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Withdraw dispute")}</DialogTitle>
          <DialogDescription>
            {t("Drops the case without recording an outcome. The invoice stops reading Disputed.")}
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          placeholder={t("Why it is being withdrawn, optional")}
          rows={3}
          aria-label={t("Withdrawal notes")}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="button" disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            {t("Withdraw dispute")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
