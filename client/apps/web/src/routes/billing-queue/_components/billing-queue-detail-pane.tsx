import { useT } from "@trenova/shared/i18n/use-t";
import AuditTab from "@/components/audit-tab";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { LinkedRecordLink } from "@/components/linked-record-link";
import { recordPath } from "@/config/record-links";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { exceptionReasonLabels } from "@/lib/choices";
import { billingQueue } from "@/lib/queries/billing-queue";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatCompactAge, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { CommentsTabSkeleton } from "@/routes/shipment/_components/comments/comments-skeleton";
import type { BillingQueueItem, ExceptionReasonCode } from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  LinkExternal01Icon,
  RefreshCw02Icon,
} from "@trenova/shared/components/icons";
import { lazy, Suspense, useState } from "react";
import { Link } from "react-router";
import { billingQueueItemLabel } from "@/lib/billing-queue-label";
import { shipmentDocumentsQuery } from "../billing-queue-queries";
import { BillingQueueActionBar } from "./billing-queue-action-bar";
import { BillingQueueAssignDialog } from "./billing-queue-assign-dialog";
import { BillingQueueChargesTab } from "./billing-queue-charges-tab";
import { BillingQueueDetentionHolds } from "./billing-queue-detention-holds";
import { BillingQueueDocumentsTab } from "./billing-queue-documents-tab";
import { BillingQueueInvoiceTab } from "./billing-queue-invoice-tab";
import { BillingQueueRatingReviewHold } from "./billing-queue-rating-review-hold";
import { BillingQueueReviewChecks } from "./billing-queue-review-checks";
import { BillingQueueShipmentTabSkeleton } from "./billing-queue-shipment-tab-skeleton";
import {
  BillingQueueDetailSkeleton,
  BillingQueueDetailUnselected,
} from "./billing-queue-detail-skeleton";
import { useScrolled } from "@/hooks/use-scrolled";
import { shipmentMargin } from "@/lib/billing-queue-margin";
import { useInvoiceDetails } from "./use-invoice-details";
import { useBulkRun } from "./use-bulk-run";

// The comments bring the realtime comment stack with them; the queue opens on the
// Charges tab, so it only pays for them once someone opens Activity.
const ShipmentCommentsTab = lazy(() => import("@/routes/shipment/_components/comments"));
// The Shipment tab reads its shipment when it opens, so its code waits for it too.
const BillingQueueShipmentTab = lazy(() => import("./billing-queue-shipment-tab"));

type DetailTab = "charges" | "shipment" | "documents" | "invoice" | "activity";

export default function BillingQueueDetailPane({
  selectedItemId,
  selectedDocumentId,
  onDocumentSelect,
  onAutoAdvance,
}: {
  selectedItemId: string | null;
  selectedDocumentId?: string | null;
  onDocumentSelect: (docId: string, fileName: string) => void;
  onAutoAdvance?: () => void;
}) {
  const [assignDialogOpen, setAssignDialogOpen] = useState(false);
  const { scrolled, ref: scrollRef } = useScrolled();
  // Each "Show me" from the action bar, counted so the review replays its pulse.
  const [reviewCalls, setReviewCalls] = useState(0);
  const [tab, setTab] = useState<DetailTab>("charges");

  const { data: item, isLoading } = useQuery({
    ...billingQueue.get(selectedItemId ?? ""),
    enabled: !!selectedItemId,
  });

  if (!selectedItemId) {
    return <BillingQueueDetailUnselected />;
  }

  if (isLoading || !item) {
    return <BillingQueueDetailSkeleton />;
  }

  return (
    <div className="flex h-full flex-col">
      {/* The record arrives in a staggered reveal, in place of its skeleton or the item
          before it: the header, then what the biller must settle, then the tabs. Keyed by
          item so a refetch of the same item stays still; reduced motion settles it at once. */}
      <div key={item.id} className="animate-reveal" data-testid="billing-queue-detail-enter">
        <DetailHeader item={item} scrolled={scrolled} />
      </div>
      <ScrollArea maskVariant="card" className="min-h-0 flex-1" viewportRef={scrollRef}>
        <div className="flex flex-col gap-5.5 px-5 pt-1 pb-6">
          <div
            key={item.id}
            className="animate-reveal flex flex-col gap-5.5 [animation-delay:70ms]"
          >
            <ItemNotices item={item} />
            <BillingQueueReviewChecks
              item={item}
              attention={reviewCalls}
              onDocumentSelect={onDocumentSelect}
            />
          </div>
          <div key={`${item.id}-tabs`} className="animate-reveal [animation-delay:140ms]">
            <DetailTabs
              item={item}
              tab={tab}
              onTabChange={setTab}
              selectedDocumentId={selectedDocumentId ?? null}
              onDocumentSelect={onDocumentSelect}
            />
          </div>
        </div>
      </ScrollArea>
      <BillingQueueActionBar
        item={item}
        onAssignBiller={() => setAssignDialogOpen(true)}
        onAutoAdvance={onAutoAdvance}
        onShowReview={() => setReviewCalls((calls) => calls + 1)}
      />
      {assignDialogOpen && (
        <BillingQueueAssignDialog
          open={assignDialogOpen}
          onOpenChange={setAssignDialogOpen}
          targets={[{ id: item.id, label: billingQueueItemLabel(item) }]}
        />
      )}
    </div>
  );
}

/**
 * The PRO, its status and route, the amount this payer owes, when it delivered, and
 * margin. Its bottom edge shows once the item's content scrolls under it.
 */
function DetailHeader({ item, scrolled }: { item: BillingQueueItem; scrolled: boolean }) {
  const t = useT();
  // The PRO is shown either way; only someone who can read shipments is offered its page.
  const canReadShipment = useCanReadKind("shipment");
  const now = useNow("minute");
  const shipment = item.shipment;
  const proNumber = shipment?.proNumber ?? billingQueueItemLabel(item);
  const totalCharge = Number(shipment?.totalChargeAmount ?? 0);
  const allocated = item.allocatedTotalAmount != null ? Number(item.allocatedTotalAmount) : null;
  const isPartial =
    shipment != null && allocated != null && Math.abs(allocated - totalCharge) >= 0.005;
  const amount = shipment ? (isPartial && allocated != null ? allocated : totalCharge) : allocated;
  const origin = shipment ? getOriginLocation(shipment) : null;
  const destination = shipment ? getDestinationLocation(shipment) : null;
  const routeWithheld = shipment != null && shipment.moves === null;
  const payerName = item.billToCustomer?.name ?? shipment?.customer?.name;
  const onBehalfOf =
    item.billToCustomerId && shipment?.customerId && item.billToCustomerId !== shipment.customerId
      ? shipment.customer?.name
      : null;
  const deliveredAt = shipment?.actualDeliveryDate ?? null;
  // The line under the amount names the payer until the shipment delivers.
  const sublineNamesPayer = !deliveredAt;

  return (
    <div
      className="relative flex shrink-0 flex-col gap-3 px-5 pt-4 pb-3"
      data-scrolled={scrolled || undefined}
    >
      <span
        aria-hidden
        className={cn(
          "bg-border pointer-events-none absolute inset-x-0 bottom-0 h-px transition-opacity duration-300 ease-settle",
          scrolled ? "opacity-100" : "opacity-0",
        )}
      />
      <div className="flex flex-wrap items-center gap-2.5">
        <h2 className="text-foreground font-mono text-2xl font-semibold tracking-tight">
          {proNumber}
        </h2>
        <PlainBillingQueueStatusBadge status={item.status} />
        <span className="text-foreground-muted text-sm">
          {origin && destination ? (
            t(
              "{0}, {1} → {2}, {3}",
              origin.city,
              origin.state?.abbreviation ?? "",
              destination.city,
              destination.state?.abbreviation ?? "",
            )
          ) : routeWithheld ? (
            <WithheldValue kind="shipment" />
          ) : sublineNamesPayer ? null : (
            payerName
          )}
        </span>
        <span className="flex-1" />
        {item.shipmentId && canReadShipment ? (
          <Link
            to={recordPath("shipment", item.shipmentId)}
            className="text-brand inline-flex items-center gap-1 text-sm font-medium hover:underline"
          >
            {t("Open shipment")}
            <LinkExternal01Icon className="size-3" />
          </Link>
        ) : null}
      </div>
      <div className="flex flex-wrap items-baseline gap-x-3.5 gap-y-1">
        <span className="text-foreground text-3xl font-semibold tracking-tight tabular-nums">
          {formatCurrency(amount ?? 0)}
        </span>
        {isPartial ? (
          <span className="text-foreground-muted text-sm tabular-nums">
            {t("of {0} shipment total", formatCurrency(totalCharge))}
          </span>
        ) : null}
        <span className="text-foreground-muted text-sm">
          {deliveredAt
            ? t(
                "Delivered {0} · {1} ago",
                formatUnixInUserTimezone(deliveredAt, {
                  month: "short",
                  day: "numeric",
                  hour: "numeric",
                  minute: "2-digit",
                }),
                formatCompactAge(now - deliveredAt),
              )
            : onBehalfOf
              ? t("Bills {0} on behalf of {1}", payerName ?? "", onBehalfOf)
              : payerName}
        </span>
        <span className="flex-1" />
        {item.shipmentId ? <ShipmentMargin shipmentId={item.shipmentId} /> : null}
      </div>
    </div>
  );
}

/** The shipment's estimated margin in dollars and percent, toned against its target. */
function ShipmentMargin({ shipmentId }: { shipmentId: string }) {
  const t = useT();
  const { data } = useQuery(shipmentQueries.profitability(shipmentId));
  const margin = shipmentMargin(data, t);
  if (!margin) {
    return null;
  }

  return (
    <span className="text-foreground-muted inline-flex items-center gap-1.5 text-xs">
      {t("Margin")}
      <span
        className={cn(
          "font-mono",
          margin.thin ? "text-warning-foreground" : "text-success-foreground",
        )}
      >
        {margin.text}
      </span>
    </span>
  );
}

/** What the biller must know before reading the charges: notes, exceptions and holds. */
function ItemNotices({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const billingNotes = item.shipment?.customer?.billingProfile?.billingNotes;

  if (
    !billingNotes &&
    !item.exceptionReasonCode &&
    item.detentionHolds.length === 0 &&
    !item.ratingReviewHold &&
    !item.isAdjustmentOrigin
  ) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {billingNotes ? (
        <Alert size="sm" variant="info">
          <AlertTriangleIcon />
          <AlertTitle>{t("Billing notes")}</AlertTitle>
          <AlertDescription>{billingNotes}</AlertDescription>
        </Alert>
      ) : null}
      {item.exceptionReasonCode ? (
        <Alert size="sm" variant="destructive">
          <AlertTriangleIcon />
          <AlertTitle>
            {exceptionReasonLabels[item.exceptionReasonCode as ExceptionReasonCode] ??
              item.exceptionReasonCode}
          </AlertTitle>
          {item.exceptionNotes ? <AlertDescription>{item.exceptionNotes}</AlertDescription> : null}
        </Alert>
      ) : null}
      <BillingQueueDetentionHolds holds={item.detentionHolds} />
      <BillingQueueRatingReviewHold hold={item.ratingReviewHold} />
      {item.isAdjustmentOrigin ? <AdjustmentOriginNotice item={item} /> : null}
    </div>
  );
}

function DetailTabs({
  item,
  tab,
  onTabChange,
  selectedDocumentId,
  onDocumentSelect,
}: {
  item: BillingQueueItem;
  tab: DetailTab;
  onTabChange: (tab: DetailTab) => void;
  selectedDocumentId: string | null;
  onDocumentSelect: (docId: string, fileName: string) => void;
}) {
  const t = useT();
  const shipmentId = item.shipmentId;
  const { save, editable } = useInvoiceDetails(item);
  const bulk = useBulkRun();
  const { data: documents } = useQuery({
    ...shipmentDocumentsQuery(shipmentId ?? ""),
    enabled: Boolean(shipmentId),
  });
  const chargeCount =
    (item.shipment?.freightLines?.length || (item.shipment ? 1 : 0)) +
    (item.shipment?.additionalCharges?.length ?? 0);

  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as DetailTab)}>
      <TabsList variant="underline" className="border-border w-full justify-start gap-4.5 border-b">
        <TabsTrigger value="charges">
          {t("Charges")} <TabCount value={chargeCount} />
        </TabsTrigger>
        {shipmentId ? <TabsTrigger value="shipment">{t("Shipment")}</TabsTrigger> : null}
        {shipmentId ? (
          <TabsTrigger value="documents">
            {t("Documents")} <TabCount value={documents?.length} />
          </TabsTrigger>
        ) : null}
        {item.billType === "Invoice" ? (
          <TabsTrigger value="invoice">{t("Invoice")}</TabsTrigger>
        ) : null}
        <TabsTrigger value="activity">{t("Activity")}</TabsTrigger>
      </TabsList>
      <TabsContent value="charges" className="mt-4">
        <BillingQueueChargesTab item={item} />
      </TabsContent>
      {shipmentId ? (
        <TabsContent value="shipment" className="mt-4">
          <LazyComponent fallback={<BillingQueueShipmentTabSkeleton />}>
            <BillingQueueShipmentTab item={item} />
          </LazyComponent>
        </TabsContent>
      ) : null}
      {shipmentId ? (
        <TabsContent value="documents" className="mt-4">
          <BillingQueueDocumentsTab
            shipmentId={shipmentId}
            selectedDocumentId={selectedDocumentId}
            onDocumentSelect={onDocumentSelect}
            isEditable={item.status === "InReview"}
            customerName={item.billToCustomer?.name ?? item.shipment?.customer?.name}
            onRequestFromDriver={() => void bulk.start("RequestPaperwork", [item.id])}
            packet={
              item.billType === "Invoice"
                ? {
                    documentIds: item.packetDocumentIds,
                    editable,
                    itemId: item.id,
                    onChange: (packetDocumentIds) => void save({ packetDocumentIds }),
                  }
                : undefined
            }
          />
        </TabsContent>
      ) : null}
      {item.billType === "Invoice" ? (
        <TabsContent value="invoice" className="mt-4">
          <BillingQueueInvoiceTab key={item.id} item={item} />
        </TabsContent>
      ) : null}
      <TabsContent value="activity" className="mt-4 flex flex-col gap-4">
        {shipmentId ? (
          <div className="border-border h-96 overflow-hidden rounded-md border">
            <Suspense fallback={<CommentsTabSkeleton />}>
              <ShipmentCommentsTab shipmentId={shipmentId} />
            </Suspense>
          </div>
        ) : null}
        <AuditTab resourceId={shipmentId ?? item.id} />
      </TabsContent>
    </Tabs>
  );
}

function TabCount({ value }: { value: number | undefined }) {
  if (value == null) return null;
  return <span className="text-foreground-subtle font-mono">{value}</span>;
}

function AdjustmentOriginNotice({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border-info-border bg-info-subtle overflow-hidden rounded-md border">
      <button
        type="button"
        className="flex w-full items-center gap-2.5 px-3 py-2 text-left"
        onClick={() => setExpanded((prev) => !prev)}
      >
        <RefreshCw02Icon className="text-info-foreground size-3.5 shrink-0" />
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <span className="text-info-foreground text-xs font-medium">
            {t("Adjustment-origin rebill")}
          </span>
          {item.rebillStrategy ? (
            <span className="bg-info-subtle text-info-foreground rounded-md px-1.5 py-0.5 text-2xs font-medium">
              {item.rebillStrategy}
            </span>
          ) : null}
          {item.requiresReplacementReview ? (
            <span className="bg-warning-subtle text-warning-foreground rounded-md px-1.5 py-0.5 text-2xs font-medium">
              {t("Review required")}
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-1.5">
          {item.sourceInvoiceId ? (
            <LinkedRecordLink
              kind="invoice"
              id={item.sourceInvoiceId}
              className="text-info-foreground text-2xs font-medium hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              {t("Original")}
            </LinkedRecordLink>
          ) : null}
          {item.sourceCreditMemoInvoiceId ? (
            <LinkedRecordLink
              kind="invoice"
              id={item.sourceCreditMemoInvoiceId}
              className="text-info-foreground text-2xs font-medium hover:underline"
              onClick={(e) => e.stopPropagation()}
            >
              {t("Credit memo")}
            </LinkedRecordLink>
          ) : null}
          <ChevronDownIcon
            className={cn(
              "text-info-foreground/50 size-3.5 transition-transform duration-150",
              expanded && "rotate-180",
            )}
          />
        </div>
      </button>
      {expanded ? (
        <div className="border-info-border text-2xs flex flex-wrap gap-x-5 gap-y-1 border-t px-3 py-2">
          {item.sourceInvoiceAdjustmentId ? (
            <span className="text-muted-foreground">
              {t("Adjustment")}{" "}
              <span className="text-foreground font-medium">
                {item.sourceInvoiceAdjustmentId.slice(0, 12)}
              </span>
            </span>
          ) : null}
          {item.correctionGroupId ? (
            <span className="text-muted-foreground">
              {t("Group")}{" "}
              <span className="text-foreground font-medium">
                {item.correctionGroupId.slice(0, 12)}
              </span>
            </span>
          ) : null}
          {item.rerateVariancePercent != null ? (
            <span className="text-muted-foreground">
              {t("Rerate variance")}{" "}
              <span className="text-foreground font-medium">
                {Number(item.rerateVariancePercent).toFixed(2)}%
              </span>
            </span>
          ) : null}
          {item.requiresReplacementReview ? (
            <span className="text-muted-foreground">
              {t("Replacement review")}{" "}
              <span className="text-foreground font-medium">
                {t("Required before invoice creation")}
              </span>
            </span>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
