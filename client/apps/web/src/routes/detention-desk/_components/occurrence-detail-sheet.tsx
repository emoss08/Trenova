import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { detentionWaiverReasonChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { Separator } from "@trenova/shared/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import {
  OCCURRENCE_STATUS_STYLES,
  SCORE_BAND_STYLES,
  formatDetentionMinutes,
  scoreBand,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import {
  waiverReasonSchema,
  type DetentionEvidence,
  type DetentionNotice,
  type DetentionOccurrence,
  type OccurrenceDetail,
} from "@trenova/shared/types/detention";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CopyIcon,
  FileWarningIcon,
  MailIcon,
  ScaleIcon,
  ShieldCheckIcon,
  ShieldXIcon,
} from "lucide-react";
import { useCallback, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { CalculationReceipt } from "./calculation-receipt";

const waiveFormSchema = z.object({
  reason: waiverReasonSchema,
  note: z.string().min(1, { error: "Explain why this charge is being forgiven" }),
});
type WaiveFormValues = z.infer<typeof waiveFormSchema>;

const disputeFormSchema = z.object({
  note: z.string().min(1, { error: "Record what the customer is disputing" }),
});
type DisputeFormValues = z.infer<typeof disputeFormSchema>;

type OccurrenceDetailSheetProps = {
  occurrenceId: string | null;
  onOpenChange: (open: boolean) => void;
};

function useInvalidateDetention(occurrenceId: string | null) {
  const queryClient = useQueryClient();

  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queries.detention.desk().queryKey });
    if (occurrenceId) {
      void queryClient.invalidateQueries({
        queryKey: queries.detention.occurrence(occurrenceId).queryKey,
      });
    }
  }, [queryClient, occurrenceId]);
}

function MoneySummary({ occurrence }: { occurrence: DetentionOccurrence }) {
  const marginNegative = occurrence.netMargin < 0;

  return (
    <div className="grid grid-cols-3 gap-px overflow-hidden rounded-lg border bg-border">
      <div className="bg-card p-3">
        <p className="text-muted-foreground text-[11px]">Billable</p>
        <p className="text-lg font-semibold tabular-nums">
          {formatCurrency(occurrence.billableAmount, occurrence.currency)}
        </p>
        <p className="text-muted-foreground text-[11px] tabular-nums">
          {formatDetentionMinutes(occurrence.roundedMinutes)} of{" "}
          {formatDetentionMinutes(occurrence.rawDwellMinutes)} on site
        </p>
      </div>
      <div className="bg-card p-3">
        <p className="text-muted-foreground text-[11px]">Driver pay</p>
        <p className="text-lg font-semibold tabular-nums">
          {formatCurrency(occurrence.driverPayAmount, occurrence.currency)}
        </p>
        <p className="text-muted-foreground text-[11px] tabular-nums">
          {formatDetentionMinutes(occurrence.driverPayMinutes)} payable
        </p>
      </div>
      <div className="bg-card p-3">
        <p className="text-muted-foreground text-[11px]">Net margin</p>
        <p
          className={cn(
            "text-lg font-semibold tabular-nums",
            marginNegative && "text-red-600 dark:text-red-400",
          )}
        >
          {formatCurrency(occurrence.netMargin, occurrence.currency)}
        </p>
        <p className="text-muted-foreground text-[11px]">
          {marginNegative ? "You pay more than you bill" : "After driver detention pay"}
        </p>
      </div>
    </div>
  );
}

function CollectabilityPanel({ detail }: { detail: OccurrenceDetail }) {
  const { collectability } = detail;
  const band = scoreBand(collectability.score);

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {collectability.chainValid ? (
            <ShieldCheckIcon className="size-4 text-emerald-600 dark:text-emerald-400" />
          ) : (
            <ShieldXIcon className="size-4 text-red-600 dark:text-red-400" />
          )}
          <h4 className="text-sm font-medium">Dispute defensibility</h4>
        </div>
        <Badge className={cn("border-none tabular-nums", SCORE_BAND_STYLES[band])}>
          {collectability.score}/100
        </Badge>
      </div>

      {collectability.summary && (
        <p className="text-muted-foreground text-xs">{collectability.summary}</p>
      )}

      <div className="flex flex-col gap-1.5">
        {collectability.factors.map((factor) => {
          const full = factor.earned >= factor.possible;

          return (
            <div
              key={factor.key}
              className="flex items-start justify-between gap-3 rounded-md border px-2.5 py-2"
            >
              <div className="min-w-0">
                <p className="text-xs font-medium">{factor.label}</p>
                {factor.detail && (
                  <p className="text-muted-foreground text-[11px]">{factor.detail}</p>
                )}
                {!full && factor.remedy && (
                  <p className="mt-0.5 text-[11px] text-amber-700 dark:text-amber-400">
                    {factor.remedy}
                  </p>
                )}
              </div>
              <span
                className={cn(
                  "shrink-0 text-xs tabular-nums",
                  full
                    ? "text-emerald-600 dark:text-emerald-400"
                    : "text-muted-foreground",
                )}
              >
                {factor.earned}/{factor.possible}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function EvidenceChain({ evidence }: { evidence: DetentionEvidence[] }) {
  if (evidence.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">No evidence has been recorded yet.</p>
    );
  }

  return (
    <ol className="flex flex-col">
      {evidence.map((item, index) => (
        <li key={item.id} className="relative flex gap-3 pb-3 last:pb-0">
          {index < evidence.length - 1 && (
            <span className="bg-border absolute top-4 left-[5px] h-full w-px" />
          )}
          <span className="bg-muted-foreground/60 mt-1.5 size-[11px] shrink-0 rounded-full border-2 border-background" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs font-medium">{item.kind}</span>
              <Badge variant="outline" className="text-[10px]">
                {item.source}
              </Badge>
            </div>
            <p className="text-muted-foreground text-xs">{item.summary}</p>
            <p className="text-muted-foreground/70 text-[11px] tabular-nums">
              {formatToUserTimezone(item.observedAt)}
            </p>
          </div>
        </li>
      ))}
    </ol>
  );
}

const NOTICE_STATUS_STYLES: Record<string, string> = {
  Queued: "bg-muted text-muted-foreground",
  Sent: "bg-sky-500/15 text-sky-700 dark:text-sky-400",
  Delivered: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  Opened: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  Bounced: "bg-red-500/15 text-red-700 dark:text-red-400",
  Failed: "bg-red-500/15 text-red-700 dark:text-red-400",
};

function NoticeHistory({ notices }: { notices: DetentionNotice[] }) {
  if (notices.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        No notices have been sent for this stop.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      {notices.map((notice) => (
        <div key={notice.id} className="rounded-md border px-2.5 py-2">
          <div className="flex flex-wrap items-center justify-between gap-1.5">
            <div className="flex items-center gap-1.5">
              <span className="text-xs font-medium">{notice.kind} notice</span>
              <Badge
                className={cn(
                  "border-none text-[10px]",
                  NOTICE_STATUS_STYLES[notice.deliveryStatus] ??
                    "bg-muted text-muted-foreground",
                )}
              >
                {notice.deliveryStatus}
              </Badge>
              {notice.satisfiesRequirement && (
                <Badge variant="outline" className="text-[10px]">
                  In window
                </Badge>
              )}
            </div>
            <span className="text-muted-foreground text-[11px] tabular-nums">
              {notice.sentAt
                ? formatToUserTimezone(notice.sentAt)
                : formatToUserTimezone(notice.scheduledFor)}
            </span>
          </div>
          {notice.recipients && notice.recipients.length > 0 && (
            <p className="text-muted-foreground mt-0.5 truncate text-[11px]">
              To {notice.recipients.join(", ")}
            </p>
          )}
          {notice.failureReason && (
            <p className="mt-0.5 text-[11px] text-red-600 dark:text-red-400">
              {notice.failureReason}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}

type ActionDialogProps = {
  occurrenceId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDone: () => void;
};

function WaiveDialog({ occurrenceId, open, onOpenChange, onDone }: ActionDialogProps) {
  const form = useForm<WaiveFormValues>({
    resolver: zodResolver(waiveFormSchema),
    defaultValues: { reason: "CustomerGoodwill", note: "" },
  });
  const { control, handleSubmit, reset, setError } = form;

  const { mutateAsync, isPending } = useApiMutation<
    DetentionOccurrence,
    WaiveFormValues,
    unknown,
    WaiveFormValues
  >({
    mutationFn: (values) => apiService.detentionService.waive(occurrenceId, values),
    onSuccess: () => {
      toast.success("Detention waived", {
        description: "The charge was forgiven and the reason recorded as evidence.",
      });
      reset();
      onOpenChange(false);
      onDone();
    },
    setFormError: setError,
    resourceName: "Detention Occurrence",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Waive this charge</DialogTitle>
          <DialogDescription>
            Waiving forgives the money but keeps the record, so discretionary revenue loss
            stays measurable instead of disappearing into free text.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit((values) => mutateAsync(values))}>
            <FormGroup cols={1}>
              <FormControl>
                <SelectField
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="Select a coded reason"
                  rules={{ required: true }}
                  options={detentionWaiverReasonChoices}
                  description="The coded reason waiver leakage is reported against."
                />
              </FormControl>
              <FormControl>
                <TextareaField
                  control={control}
                  name="note"
                  label="Note"
                  placeholder="Dock crew was short-staffed; customer asked for a one-time concession"
                  rules={{ required: true }}
                  description="Context the next person reviewing this waiver will need."
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="mt-4">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? "Waiving…" : "Waive charge"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function DisputeDialog({ occurrenceId, open, onOpenChange, onDone }: ActionDialogProps) {
  const form = useForm<DisputeFormValues>({
    resolver: zodResolver(disputeFormSchema),
    defaultValues: { note: "" },
  });
  const { control, handleSubmit, reset, setError } = form;

  const { mutateAsync, isPending } = useApiMutation<
    DetentionOccurrence,
    DisputeFormValues,
    unknown,
    DisputeFormValues
  >({
    mutationFn: (values) => apiService.detentionService.dispute(occurrenceId, values),
    onSuccess: () => {
      toast.success("Dispute recorded", {
        description: "The original computation is preserved for the claim file.",
      });
      reset();
      onOpenChange(false);
      onDone();
    },
    setFormError: setError,
    resourceName: "Detention Occurrence",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Record a dispute</DialogTitle>
          <DialogDescription>
            Recording the customer&apos;s rejection keeps the original computation intact —
            exactly what working the claim requires.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit((values) => mutateAsync(values))}>
            <FormGroup cols={1}>
              <FormControl>
                <TextareaField
                  control={control}
                  name="note"
                  label="What is the customer disputing?"
                  placeholder="Customer claims the driver arrived 40 minutes later than our records show"
                  rules={{ required: true }}
                  description="Their claim, verbatim where possible — it decides which evidence matters."
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="mt-4">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? "Recording…" : "Record dispute"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function OccurrenceActions({
  detail,
  onDone,
}: {
  detail: OccurrenceDetail;
  onDone: () => void;
}) {
  const { occurrence } = detail;
  const [waiveOpen, setWaiveOpen] = useState(false);
  const [disputeOpen, setDisputeOpen] = useState(false);
  const { copy, isCopied } = useCopyToClipboard();

  const invalidate = onDone;

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.approve(occurrence.id),
    onSuccess: () => {
      toast.success("Charge approved", {
        description: "The detention charge will post to the shipment.",
      });
      invalidate();
    },
    resourceName: "Detention Occurrence",
  });

  const sendNotice = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.sendNotice(occurrence.id),
    onSuccess: () => {
      toast.success("Notice sent", {
        description: "The customer notice went out and was recorded as evidence.",
      });
      invalidate();
    },
    resourceName: "Detention Notice",
  });

  const copyPacket = useApiMutation<boolean, undefined>({
    mutationFn: async () => {
      const packet = await apiService.detentionService.disputePacket(occurrence.id);
      return copy(JSON.stringify(packet, null, 2));
    },
    onSuccess: () => {
      toast.success("Dispute packet copied", {
        description: "The full claim file — receipt, evidence, and notices — is on your clipboard.",
      });
    },
    resourceName: "Dispute Packet",
  });

  const canApprove = occurrence.status === "Pending";
  const canSendNotice =
    occurrence.notificationStatus !== "NotRequired" &&
    occurrence.notificationStatus !== "Sent";
  const canWaive = occurrence.status !== "Waived" && occurrence.status !== "NotBillable";
  const canDispute = occurrence.status !== "Disputed" && occurrence.billableAmount > 0;

  return (
    <>
      <div className="flex flex-wrap gap-1.5">
        {canApprove && (
          <Button
            size="sm"
            className="h-7 gap-1.5 text-xs"
            disabled={approve.isPending}
            onClick={() => approve.mutate(undefined)}
          >
            <CheckIcon className="size-3.5" />
            {approve.isPending ? "Approving…" : "Approve charge"}
          </Button>
        )}
        {canSendNotice && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1.5 text-xs"
            disabled={sendNotice.isPending}
            onClick={() => sendNotice.mutate(undefined)}
          >
            <MailIcon className="size-3.5" />
            {sendNotice.isPending ? "Sending…" : "Send notice"}
          </Button>
        )}
        {canDispute && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1.5 text-xs"
            onClick={() => setDisputeOpen(true)}
          >
            <ScaleIcon className="size-3.5" />
            Record dispute
          </Button>
        )}
        {canWaive && (
          <Button
            size="sm"
            variant="outline"
            className="h-7 gap-1.5 text-xs"
            onClick={() => setWaiveOpen(true)}
          >
            <FileWarningIcon className="size-3.5" />
            Waive
          </Button>
        )}
        <Button
          size="sm"
          variant="outline"
          className="h-7 gap-1.5 text-xs"
          disabled={copyPacket.isPending}
          onClick={() => copyPacket.mutate(undefined)}
        >
          <CopyIcon className="size-3.5" />
          {isCopied ? "Copied" : "Copy dispute packet"}
        </Button>
      </div>

      <WaiveDialog
        occurrenceId={occurrence.id}
        open={waiveOpen}
        onOpenChange={setWaiveOpen}
        onDone={invalidate}
      />
      <DisputeDialog
        occurrenceId={occurrence.id}
        open={disputeOpen}
        onOpenChange={setDisputeOpen}
        onDone={invalidate}
      />
    </>
  );
}

/**
 * The claim file for one detention occurrence: the money, the derivation, the
 * evidence chain, and every notice — plus the actions a billing clerk takes
 * from the desk. Everything a customer dispute will ask for lives here.
 */
export function OccurrenceDetailSheet({
  occurrenceId,
  onOpenChange,
}: OccurrenceDetailSheetProps) {
  const invalidate = useInvalidateDetention(occurrenceId);

  const { data: detail, isLoading } = useQuery({
    ...queries.detention.occurrence(occurrenceId ?? ""),
    enabled: Boolean(occurrenceId),
  });

  const occurrence = detail?.occurrence;

  return (
    <Sheet open={Boolean(occurrenceId)} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[520px]">
        {isLoading || !detail || !occurrence ? (
          <div className="flex flex-col gap-3 p-4">
            <Skeleton className="h-8 w-2/3" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-40 w-full" />
          </div>
        ) : (
          <>
            <SheetHeader className="border-b px-4 py-3">
              <div className="flex items-start justify-between gap-2 pr-8">
                <div className="min-w-0">
                  <SheetTitle className="truncate">
                    {occurrence.locationName || "Unknown facility"}
                  </SheetTitle>
                  <SheetDescription className="truncate text-xs">
                    {occurrence.customerName || "Unknown customer"}
                    {occurrence.shipmentProNumber && (
                      <> · PRO {occurrence.shipmentProNumber}</>
                    )}{" "}
                    · {occurrence.stopType}
                  </SheetDescription>
                </div>
                <Badge
                  className={cn(
                    "shrink-0 border-none",
                    OCCURRENCE_STATUS_STYLES[occurrence.status],
                  )}
                >
                  {occurrence.status}
                </Badge>
              </div>
            </SheetHeader>

            <div className="flex-1 overflow-y-auto">
              <div className="flex flex-col gap-5 p-4">
                <MoneySummary occurrence={occurrence} />

                <OccurrenceActions detail={detail} onDone={invalidate} />

                <Separator />

                <CollectabilityPanel detail={detail} />

                <Separator />

                <CalculationReceipt
                  trace={occurrence.calculationTrace}
                  currency={occurrence.currency}
                />

                <Separator />

                <div className="flex flex-col gap-2.5">
                  <h4 className="text-sm font-medium">Evidence chain</h4>
                  <EvidenceChain evidence={detail.evidence ?? []} />
                </div>

                <Separator />

                <div className="flex flex-col gap-2.5">
                  <h4 className="text-sm font-medium">Notices</h4>
                  <NoticeHistory notices={detail.notices ?? []} />
                </div>
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
