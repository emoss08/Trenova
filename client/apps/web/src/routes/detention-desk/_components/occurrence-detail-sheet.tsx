import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { deliveryStatusLabel, noticeKindLabel } from "@/lib/detention-labels";
import { detention } from "@/lib/queries/detention";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
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
  type CollectabilityAssessment,
  type DetentionEvidence,
  type DetentionNotice,
  type DetentionOccurrence,
  type OccurrenceDetail,
} from "@trenova/shared/types/detention";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";
import type { ReactNode } from "react";
import { useState } from "react";
import { CalculationReceipt } from "./calculation-receipt";
import { DisputeDialog, WaiveDialog } from "./occurrence-action-dialogs";
import { useInvalidateDetention, useSendDetentionNotice } from "./use-detention-actions";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";

const METER_TRANSITION = { type: "spring", stiffness: 160, damping: 28, mass: 0.7 } as const;

/** Below this the claim needs work before anyone leans on it in a dispute. */
const WEAK_SCORE = 65;

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
            <h3 className="text-xs text-muted-foreground font-medium">{title}</h3>
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
  const t = useT();

  const netMargin = occurrence.netMargin;
  const marginNegative = netMargin !== null && netMargin < 0;

  return (
    <KpiStrip minItemWidth="8rem">
      <KpiStripItem
        label={t("Billable")}
        value={formatCurrency(occurrence.billableAmount, occurrence.currency)}
        sub={t(
          "{0} of {1}",
          formatDetentionMinutes(occurrence.roundedMinutes),
          formatDetentionMinutes(occurrence.rawDwellMinutes),
        )}
      />
      <KpiStripItem
        label={t("Driver pay")}
        value={
          occurrence.driverPayAmount === null ? (
            <WithheldValue group="Margins" />
          ) : (
            formatCurrency(occurrence.driverPayAmount, occurrence.currency)
          )
        }
        sub={t("{0} payable", formatDetentionMinutes(occurrence.driverPayMinutes))}
      />
      <KpiStripItem
        label={t("Net margin")}
        value={
          netMargin === null ? (
            <WithheldValue group="Margins" />
          ) : (
            formatCurrency(netMargin, occurrence.currency)
          )
        }
        sub={
          netMargin === null
            ? undefined
            : marginNegative
              ? t("You pay more than you bill")
              : t("After driver pay")
        }
        tone={marginNegative ? "danger" : undefined}
      />
    </KpiStrip>
  );
}

function CollectabilityPanel({ collectability }: { collectability: CollectabilityAssessment }) {
  const t = useT();

  const isWeak = collectability.score < WEAK_SCORE;

  return (
    <>
      <div className="bg-border h-1 w-full rounded-full">
        <m.div
          className={cn("h-1 rounded-full", isWeak ? "bg-warning" : "bg-foreground/70")}
          initial={{ width: 0 }}
          animate={{ width: `${Math.min(100, Math.max(0, collectability.score))}%` }}
          transition={METER_TRANSITION}
        />
      </div>

      {collectability.summary && (
        <p className="text-muted-foreground mt-2 text-xs">{collectability.summary}</p>
      )}

      <p
        className={cn(
          "text-2xs mt-1",
          collectability.chainValid ? "text-muted-foreground/70" : "text-danger-foreground",
        )}
      >
        {collectability.chainValid
          ? t("Evidence chain verified")
          : t("Evidence chain broken — the record no longer hashes clean")}
      </p>

      <ul className="divide-border/60 mt-2 divide-y">
        {collectability.factors.map((factor) => {
          const full = factor.earned >= factor.possible;

          return (
            <li key={factor.key} className="flex items-start justify-between gap-3 py-2">
              <div className="min-w-0">
                <p className="text-xs">{t(factor.label)}</p>
                {factor.detail && <p className="text-2xs text-muted-foreground">{factor.detail}</p>}
                {!full && factor.remedy && (
                  <p className="text-2xs mt-0.5 text-warning-foreground">{factor.remedy}</p>
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
  const t = useT();

  if (evidence.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">{t("No evidence has been recorded yet.")}</p>
    );
  }

  return (
    <ol className="flex flex-col">
      {evidence.map((item, index) => (
        <li key={item.id} className="relative flex gap-2.5 pb-3 last:pb-0">
          {/* The connector runs from the bottom of this dot to the top of the next
              one, which sits 6px (the dot's own top margin) into the row below —
              so it has to overhang this row's padding by exactly that much. */}
          {index < evidence.length - 1 && (
            <span className="bg-border absolute top-3 -bottom-1.5 left-[3px] w-px -translate-x-1/2" />
          )}
          <span className="bg-muted-foreground/50 mt-1.5 size-1.5 shrink-0 rounded-full" />
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <p className="truncate text-xs">
                {toTitleCase(item.kind)}
                <span className="text-muted-foreground"> · {toTitleCase(item.source)}</span>
              </p>
              <span className="text-2xs text-muted-foreground shrink-0 tabular-nums">
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
  const t = useT();

  if (notices.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("No notices have been sent for this stop.")}
      </p>
    );
  }

  return (
    <ul className="divide-border/60 divide-y">
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
              <span className="truncate">{noticeKindLabel(notice.kind, t)}</span>
              <span className="text-muted-foreground shrink-0">
                {notice.satisfiesRequirement
                  ? t("{0} · in window", deliveryStatusLabel(notice.deliveryStatus, t))
                  : deliveryStatusLabel(notice.deliveryStatus, t)}
              </span>
            </span>
            <span className="text-2xs text-muted-foreground shrink-0 tabular-nums">
              {notice.sentAt
                ? formatToUserTimezone(notice.sentAt)
                : formatToUserTimezone(notice.scheduledFor)}
            </span>
          </div>
          {notice.recipients && notice.recipients.length > 0 && (
            <p className="text-2xs text-muted-foreground truncate">
              {t("To {0}", notice.recipients.join(", "))}
            </p>
          )}
          {notice.failureReason && (
            <p className="text-2xs text-danger-foreground">{notice.failureReason}</p>
          )}
        </li>
      ))}
    </ul>
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
  const t = useT();

  const { occurrence } = detail;
  const [waiveOpen, setWaiveOpen] = useState(false);
  const [disputeOpen, setDisputeOpen] = useState(false);
  const { copy, isCopied } = useCopyToClipboard();

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    mutationFn: () => apiService.detentionService.approve(occurrence.id),
    onSuccess: () => {
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
    resourceName: "Dispute Packet",
  });

  const actions: OccurrenceAction[] = [];

  if (occurrence.status === "Pending") {
    actions.push({
      key: "approve",
      label: t("Approve charge"),
      pendingLabel: "Approving",
      isPending: approve.isPending,
      onSelect: () => approve.mutate(undefined),
    });
  }

  if (occurrence.notificationStatus !== "NotRequired" && occurrence.notificationStatus !== "Sent") {
    actions.push({
      key: "notice",
      label: t("Send notice"),
      pendingLabel: "Sending",
      isPending: sendNotice.isPending,
      onSelect: () => sendNotice.mutate(undefined),
    });
  }

  if (occurrence.status !== "Disputed" && occurrence.billableAmount > 0) {
    actions.push({
      key: "dispute",
      label: t("Record dispute"),
      onSelect: () => setDisputeOpen(true),
    });
  }

  if (occurrence.status !== "Waived" && occurrence.status !== "NotBillable") {
    actions.push({ key: "waive", label: t("Waive"), onSelect: () => setWaiveOpen(true) });
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
            {t(action.label)}
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
  const t = useT();

  const invalidate = useInvalidateDetention();

  const { data: detail, isLoading } = useQuery({
    ...detention.occurrence(occurrenceId ?? ""),
    enabled: Boolean(occurrenceId),
  });

  const occurrence = detail?.occurrence;

  return (
    <Sheet open={Boolean(occurrenceId)} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-lg">
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
                <span className="text-xs text-muted-foreground font-medium">
                  {OCCURRENCE_STATUS_LABEL[occurrence.status]}
                </span>
              </div>
              <SheetTitle className="truncate">
                {occurrence.locationName || t("Unknown facility")}
              </SheetTitle>
              <SheetDescription className="truncate text-xs">
                {occurrence.customerName || t("Unknown customer")}
                {occurrence.shipmentProNumber && (
                  <> {t("· PRO {0}", occurrence.shipmentProNumber)}</>
                )}{" "}
                · {occurrence.stopType}
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
                title={t("Defensibility")}
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

              <Section title={t("Evidence chain")}>
                <EvidenceChain evidence={detail.evidence ?? []} />
              </Section>

              <Section title={t("Notices")}>
                <NoticeHistory notices={detail.notices ?? []} />
              </Section>
            </ScrollArea>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
