import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { detentionWaiverReasonChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
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
  NOTICE_DELIVERY_DOT,
  OCCURRENCE_STATUS_DOT,
  OCCURRENCE_STATUS_LABEL,
  formatDetentionMinutes,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency, toTitleCase } from "@trenova/shared/lib/utils";
import {
  waiverReasonSchema,
  type CollectabilityAssessment,
  type DetentionEvidence,
  type DetentionNotice,
  type DetentionOccurrence,
  type OccurrenceDetail,
} from "@trenova/shared/types/detention";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";
import type { ReactNode } from "react";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { CalculationReceipt } from "./calculation-receipt";
import { DeskMetric } from "./desk-metric";
import { useInvalidateDetention, useSendDetentionNotice } from "./use-detention-actions";

const METER_TRANSITION = { type: "spring", stiffness: 160, damping: 28, mass: 0.7 } as const;

/** Below this the claim needs work before anyone leans on it in a dispute. */
const WEAK_SCORE = 65;

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

function Section({
  title,
  action,
  children,
}: {
  title?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="border-b px-4 py-4 last:border-b-0">
      {(title || action) && (
        <div className="mb-2.5 flex items-center justify-between gap-2">
          {title ? (
            <h3 className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
              {title}
            </h3>
          ) : (
            <span />
          )}
          {action}
        </div>
      )}
      {children}
    </section>
  );
}

function MoneySummary({ occurrence }: { occurrence: DetentionOccurrence }) {
  const marginNegative = occurrence.netMargin < 0;

  return (
    <dl className="grid grid-cols-3 divide-x divide-border">
      <DeskMetric
        size="md"
        label="Billable"
        value={formatCurrency(occurrence.billableAmount, occurrence.currency)}
        sub={`${formatDetentionMinutes(occurrence.roundedMinutes)} of ${formatDetentionMinutes(
          occurrence.rawDwellMinutes,
        )}`}
        className="pr-4"
      />
      <DeskMetric
        size="md"
        label="Driver pay"
        value={formatCurrency(occurrence.driverPayAmount, occurrence.currency)}
        sub={`${formatDetentionMinutes(occurrence.driverPayMinutes)} payable`}
        className="px-4"
      />
      <DeskMetric
        size="md"
        label="Net margin"
        value={formatCurrency(occurrence.netMargin, occurrence.currency)}
        sub={marginNegative ? "You pay more than you bill" : "After driver pay"}
        valueClassName={cn(marginNegative && "text-red-600 dark:text-red-400")}
        className="pl-4"
      />
    </dl>
  );
}

function CollectabilityPanel({ collectability }: { collectability: CollectabilityAssessment }) {
  const isWeak = collectability.score < WEAK_SCORE;

  return (
    <>
      <div className="h-1 w-full rounded-full bg-border">
        <m.div
          className={cn("h-1 rounded-full", isWeak ? "bg-amber-500" : "bg-foreground/70")}
          initial={{ width: 0 }}
          animate={{ width: `${Math.min(100, Math.max(0, collectability.score))}%` }}
          transition={METER_TRANSITION}
        />
      </div>

      {collectability.summary && (
        <p className="mt-2 text-xs text-muted-foreground">{collectability.summary}</p>
      )}

      <p
        className={cn(
          "mt-1 text-2xs",
          collectability.chainValid ? "text-muted-foreground/70" : "text-red-600 dark:text-red-400",
        )}
      >
        {collectability.chainValid
          ? "Evidence chain verified"
          : "Evidence chain broken — the record no longer hashes clean"}
      </p>

      <ul className="mt-2 divide-y divide-border/60">
        {collectability.factors.map((factor) => {
          const full = factor.earned >= factor.possible;

          return (
            <li key={factor.key} className="flex items-start justify-between gap-3 py-2">
              <div className="min-w-0">
                <p className="text-xs">{factor.label}</p>
                {factor.detail && <p className="text-2xs text-muted-foreground">{factor.detail}</p>}
                {!full && factor.remedy && (
                  <p className="mt-0.5 text-2xs text-amber-700 dark:text-amber-500">
                    {factor.remedy}
                  </p>
                )}
              </div>
              <span
                className={cn(
                  "shrink-0 text-xs tabular-nums",
                  full ? "text-muted-foreground" : "text-foreground",
                )}
              >
                {factor.earned}/{factor.possible}
              </span>
            </li>
          );
        })}
      </ul>
    </>
  );
}

function EvidenceChain({ evidence }: { evidence: DetentionEvidence[] }) {
  if (evidence.length === 0) {
    return <p className="text-xs text-muted-foreground">No evidence has been recorded yet.</p>;
  }

  return (
    <ol className="flex flex-col">
      {evidence.map((item, index) => (
        <li key={item.id} className="relative flex gap-2.5 pb-3 last:pb-0">
          {/* The connector runs from the bottom of this dot to the top of the next
              one, which sits 6px (the dot's own top margin) into the row below —
              so it has to overhang this row's padding by exactly that much. */}
          {index < evidence.length - 1 && (
            <span className="absolute top-3 -bottom-1.5 left-[3px] w-px -translate-x-1/2 bg-border" />
          )}
          <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-muted-foreground/50" />
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <p className="truncate text-xs">
                {toTitleCase(item.kind)}
                <span className="text-muted-foreground"> · {toTitleCase(item.source)}</span>
              </p>
              <span className="shrink-0 text-2xs text-muted-foreground tabular-nums">
                {formatToUserTimezone(item.observedAt)}
              </span>
            </div>
            <p className="text-2xs text-muted-foreground">{item.summary}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}

function NoticeHistory({ notices }: { notices: DetentionNotice[] }) {
  if (notices.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">No notices have been sent for this stop.</p>
    );
  }

  return (
    <ul className="divide-y divide-border/60">
      {notices.map((notice) => (
        <li key={notice.id} className="flex flex-col gap-0.5 py-2 first:pt-0 last:pb-0">
          <div className="flex items-baseline justify-between gap-2">
            <span className="flex min-w-0 items-center gap-1.5 text-xs">
              <span
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  NOTICE_DELIVERY_DOT[notice.deliveryStatus],
                )}
              />
              <span className="truncate">{toTitleCase(notice.kind)} notice</span>
              <span className="shrink-0 text-muted-foreground">
                {toTitleCase(notice.deliveryStatus)}
                {notice.satisfiesRequirement ? " · in window" : ""}
              </span>
            </span>
            <span className="shrink-0 text-2xs text-muted-foreground tabular-nums">
              {notice.sentAt
                ? formatToUserTimezone(notice.sentAt)
                : formatToUserTimezone(notice.scheduledFor)}
            </span>
          </div>
          {notice.recipients && notice.recipients.length > 0 && (
            <p className="truncate text-2xs text-muted-foreground">
              To {notice.recipients.join(", ")}
            </p>
          )}
          {notice.failureReason && (
            <p className="text-2xs text-red-600 dark:text-red-400">{notice.failureReason}</p>
          )}
        </li>
      ))}
    </ul>
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
            Waiving forgives the money but keeps the record, so discretionary revenue loss stays
            measurable instead of disappearing into free text.
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
              <Button type="submit" isLoading={isPending} loadingText="Waiving">
                Waive charge
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
            Recording the customer&apos;s rejection keeps the original computation intact — exactly
            what working the claim requires.
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
              <Button type="submit" isLoading={isPending} loadingText="Recording">
                Record dispute
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

type OccurrenceAction = {
  key: string;
  label: string;
  pendingLabel?: string;
  isPending?: boolean;
  onSelect: () => void;
};

/**
 * Everything a clerk can still do to this charge, in the order it matters. The
 * first available action carries the weight — the rest stay quiet, because a
 * row of five equally loud buttons tells you nothing about which one to press.
 */
function OccurrenceActions({ detail, onDone }: { detail: OccurrenceDetail; onDone: () => void }) {
  const { occurrence } = detail;
  const [waiveOpen, setWaiveOpen] = useState(false);
  const [disputeOpen, setDisputeOpen] = useState(false);
  const { copy, isCopied } = useCopyToClipboard();

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.approve(occurrence.id),
    onSuccess: () => {
      toast.success("Charge approved", {
        description: "The detention charge will post to the shipment.",
      });
      onDone();
    },
    resourceName: "Detention Occurrence",
  });

  const sendNotice = useSendDetentionNotice(occurrence.id);

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

  const actions: OccurrenceAction[] = [];

  if (occurrence.status === "Pending") {
    actions.push({
      key: "approve",
      label: "Approve charge",
      pendingLabel: "Approving",
      isPending: approve.isPending,
      onSelect: () => approve.mutate(undefined),
    });
  }

  if (occurrence.notificationStatus !== "NotRequired" && occurrence.notificationStatus !== "Sent") {
    actions.push({
      key: "notice",
      label: "Send notice",
      pendingLabel: "Sending",
      isPending: sendNotice.isPending,
      onSelect: () => sendNotice.mutate(undefined),
    });
  }

  if (occurrence.status !== "Disputed" && occurrence.billableAmount > 0) {
    actions.push({
      key: "dispute",
      label: "Record dispute",
      onSelect: () => setDisputeOpen(true),
    });
  }

  if (occurrence.status !== "Waived" && occurrence.status !== "NotBillable") {
    actions.push({ key: "waive", label: "Waive", onSelect: () => setWaiveOpen(true) });
  }

  actions.push({
    key: "packet",
    label: isCopied ? "Packet copied" : "Copy dispute packet",
    pendingLabel: "Building",
    isPending: copyPacket.isPending,
    onSelect: () => copyPacket.mutate(undefined),
  });

  return (
    <>
      <div className="flex flex-wrap gap-1.5">
        {actions.map((action, index) => (
          <Button
            key={action.key}
            size="sm"
            variant={index === 0 ? "default" : "outline"}
            className="text-xs"
            isLoading={action.isPending}
            loadingText={action.pendingLabel}
            onClick={action.onSelect}
          >
            {action.label}
          </Button>
        ))}
      </div>

      <WaiveDialog
        occurrenceId={occurrence.id}
        open={waiveOpen}
        onOpenChange={setWaiveOpen}
        onDone={onDone}
      />
      <DisputeDialog
        occurrenceId={occurrence.id}
        open={disputeOpen}
        onOpenChange={setDisputeOpen}
        onDone={onDone}
      />
    </>
  );
}

function DetailSkeleton() {
  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-3 w-64" />
      </div>
      <div className="grid grid-cols-3 gap-4">
        {Array.from({ length: 3 }).map((_, index) => (
          <Skeleton key={index} className="h-14 w-full" />
        ))}
      </div>
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-48 w-full" />
    </div>
  );
}

/**
 * The claim file for one detention occurrence: the money, the derivation, the
 * evidence chain, and every notice — plus the actions a billing clerk takes
 * from the desk. Everything a customer dispute will ask for lives here, set in
 * the same hairline sections the desk itself uses.
 */
export function OccurrenceDetailSheet({ occurrenceId, onOpenChange }: OccurrenceDetailSheetProps) {
  const invalidate = useInvalidateDetention();

  const { data: detail, isLoading } = useQuery({
    ...queries.detention.occurrence(occurrenceId ?? ""),
    enabled: Boolean(occurrenceId),
  });

  const occurrence = detail?.occurrence;

  return (
    <Sheet open={Boolean(occurrenceId)} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-[540px]">
        {isLoading || !detail || !occurrence ? (
          <DetailSkeleton />
        ) : (
          <>
            <SheetHeader className="gap-1 border-b px-4 py-3 pr-12">
              <div className="flex items-center gap-1.5">
                <span
                  className={cn(
                    "size-1.5 shrink-0 rounded-full",
                    OCCURRENCE_STATUS_DOT[occurrence.status],
                  )}
                />
                <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
                  {OCCURRENCE_STATUS_LABEL[occurrence.status]}
                </span>
              </div>
              <SheetTitle className="truncate">
                {occurrence.locationName || "Unknown facility"}
              </SheetTitle>
              <SheetDescription className="truncate text-xs">
                {occurrence.customerName || "Unknown customer"}
                {occurrence.shipmentProNumber && <> · PRO {occurrence.shipmentProNumber}</>} ·{" "}
                {occurrence.stopType}
              </SheetDescription>
            </SheetHeader>

            <ScrollArea className="min-h-0 flex-1">
              <Section>
                <MoneySummary occurrence={occurrence} />
              </Section>

              <Section>
                <OccurrenceActions detail={detail} onDone={invalidate} />
              </Section>

              <Section
                title="Defensibility"
                action={
                  <span className="text-xs tabular-nums">
                    {detail.collectability.score}
                    <span className="text-muted-foreground">/100</span>
                  </span>
                }
              >
                <CollectabilityPanel collectability={detail.collectability} />
              </Section>

              <Section>
                <CalculationReceipt
                  trace={occurrence.calculationTrace}
                  currency={occurrence.currency}
                />
              </Section>

              <Section title="Evidence chain">
                <EvidenceChain evidence={detail.evidence ?? []} />
              </Section>

              <Section title="Notices">
                <NoticeHistory notices={detail.notices ?? []} />
              </Section>
            </ScrollArea>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
