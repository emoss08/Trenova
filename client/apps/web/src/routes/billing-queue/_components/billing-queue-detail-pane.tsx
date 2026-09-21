import { useT } from "@trenova/shared/i18n/use-t";
import { KPI_VALUE_LG_CLASS } from "@/components/kpi/kpi-strip";
import AuditTab from "@/components/audit-tab";
import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { PlainBillingQueueStatusBadge } from "@trenova/shared/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { exceptionReasonLabels } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { CommentsTabSkeleton } from "@/routes/shipment/_components/comments/comments-skeleton";
import type { ExceptionReasonCode } from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ChevronDownIcon, RefreshCwIcon, TimerIcon } from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { BillingQueueActionBar } from "./billing-queue-action-bar";
import { BillingQueueAssignDialog } from "./billing-queue-assign-dialog";
import { BillingQueueChargesTab } from "./billing-queue-charges-tab";
import { BillingQueueDocumentsTab } from "./billing-queue-documents-tab";

// The comments tab brings the realtime comment stack with it; the billing queue
// opens on the Charges tab, so it only pays for that once someone switches.
const ShipmentCommentsTab = lazy(() => import("@/routes/shipment/_components/comments"));

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
  const t = useT();

  const [assignDialogOpen, setAssignDialogOpen] = useState(false);

  const { data: item, isLoading } = useQuery({
    ...queries.billingQueue.get(selectedItemId ?? ""),
    enabled: !!selectedItemId,
  });

  const handleAssignBiller = useCallback(() => {
    setAssignDialogOpen(true);
  }, []);

  if (!selectedItemId) {
    return (
      <BillingDetailUnselected
        layout="tabs"
        title={t("Nothing open")}
        description={t(
          "Pick an item from the queue to review it here, or press J to start at the top.",
        )}
      />
    );
  }

  if (isLoading || !item) {
    return (
      <div className="flex flex-col gap-4 p-4">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const shipment = item.shipment;
  const customerName = shipment?.customer?.name ?? "Unknown Customer";
  const payerName = item.billToCustomer?.name ?? customerName;
  const onBehalfOf =
    item.billToCustomerId && shipment?.customerId && item.billToCustomerId !== shipment.customerId
      ? customerName
      : null;
  const proNumber = shipment?.proNumber ?? item.shipmentId.slice(0, 12);
  const totalCharge = Number(shipment?.totalChargeAmount ?? 0);
  const allocated = item.allocatedTotalAmount != null ? Number(item.allocatedTotalAmount) : null;
  const isPartial = allocated != null && Math.abs(allocated - totalCharge) >= 0.005;
  const originLocation = shipment ? getOriginLocation(shipment) : null;
  const destLocation = shipment ? getDestinationLocation(shipment) : null;

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 space-y-3 border-b px-4 py-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <Link
              to={`/shipment-management/shipments?item=${item.shipmentId}`}
              className="text-lg font-semibold hover:underline"
            >
              {proNumber}
            </Link>
            <PlainBillingQueueStatusBadge status={item.status} />
          </div>
          <div className="flex items-center gap-2">
            {item.status === "InReview" && item.reviewStartedAt && (
              <ReviewTimer startedAt={item.reviewStartedAt} />
            )}
          </div>
        </div>

        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <span className={KPI_VALUE_LG_CLASS}>
            {formatCurrency(isPartial && allocated != null ? allocated : totalCharge)}
          </span>
          {isPartial ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("of {0} shipment total", formatCurrency(totalCharge))}
            </span>
          ) : null}
        </div>

        <div className="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-3 lg:grid-cols-4">
          {item.number ? <MetadataCell label={t("Queue #")} value={item.number} /> : null}
          <MetadataCell label={t("Bill To")} value={payerName} />
          {onBehalfOf ? <MetadataCell label={t("On behalf of")} value={onBehalfOf} /> : null}
          {shipment?.bol ? <MetadataCell label={t("BOL")} value={shipment.bol} /> : null}
          {item.assignedBiller ? (
            <MetadataCell label={t("Assigned biller")} value={item.assignedBiller.name} />
          ) : null}
          {originLocation && destLocation ? (
            <MetadataCell
              label={t("Route")}
              value={`${originLocation.city}, ${originLocation.state?.abbreviation} → ${destLocation.city}, ${destLocation.state?.abbreviation}`}
            />
          ) : null}
        </div>
      </div>
      <BillingQueueActionBar
        item={item}
        onAssignBiller={handleAssignBiller}
        onAutoAdvance={onAutoAdvance}
      />
      {shipment?.customer?.billingProfile?.billingNotes && (
        <div className="p-2">
          <Alert variant="info">
            <AlertTriangleIcon className="size-4" />
            <AlertTitle>{t("Billing notes")}</AlertTitle>
            <AlertDescription>{shipment.customer.billingProfile.billingNotes}</AlertDescription>
          </Alert>
        </div>
      )}
      {item.exceptionReasonCode && (
        <div className="shrink-0 px-4 pt-2">
          <Alert variant="destructive">
            <AlertTriangleIcon className="size-4" />
            <AlertTitle>
              {exceptionReasonLabels[item.exceptionReasonCode as ExceptionReasonCode] ??
                item.exceptionReasonCode}
            </AlertTitle>
            {item.exceptionNotes && <AlertDescription>{item.exceptionNotes}</AlertDescription>}
          </Alert>
        </div>
      )}
      {item.isAdjustmentOrigin ? (
        <AdjustmentOriginBanner
          rebillStrategy={item.rebillStrategy}
          requiresReplacementReview={item.requiresReplacementReview}
          rerateVariancePercent={item.rerateVariancePercent}
          sourceInvoiceId={item.sourceInvoiceId}
          sourceCreditMemoInvoiceId={item.sourceCreditMemoInvoiceId}
          sourceInvoiceAdjustmentId={item.sourceInvoiceAdjustmentId}
          correctionGroupId={item.correctionGroupId}
        />
      ) : null}
      <Tabs defaultValue="charges" className="flex min-h-0 flex-1 flex-col">
        <TabsList variant="underline" className="border-border w-full border-b">
          <TabsTrigger value="charges">{t("Charges")}</TabsTrigger>
          <TabsTrigger value="documents">{t("Documents")}</TabsTrigger>
          <TabsTrigger value="comments">{t("Comments")}</TabsTrigger>
          <TabsTrigger value="activity">{t("Activity")}</TabsTrigger>
        </TabsList>
        <TabsContent value="charges" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <BillingQueueChargesTab item={item} />
          </ScrollArea>
        </TabsContent>
        <TabsContent value="documents" className="mt-0 min-h-0 flex-1">
          <BillingQueueDocumentsTab
            shipmentId={item.shipmentId}
            selectedDocumentId={selectedDocumentId ?? null}
            onDocumentSelect={onDocumentSelect}
            isEditable={item.status === "InReview"}
          />
        </TabsContent>
        <TabsContent value="comments" className="mt-0 min-h-0 flex-1">
          <Suspense fallback={<CommentsTabSkeleton />}>
            <ShipmentCommentsTab shipmentId={item.shipmentId} />
          </Suspense>
        </TabsContent>
        <TabsContent value="activity" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="px-4">
              <AuditTab resourceId={item.shipmentId} />
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>
      {assignDialogOpen && (
        <BillingQueueAssignDialog
          open={assignDialogOpen}
          onOpenChange={setAssignDialogOpen}
          itemId={item.id}
        />
      )}
    </div>
  );
}

function MetadataCell({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-muted-foreground text-xs">{label}</p>
      <p className="text-sm font-medium">{value}</p>
    </div>
  );
}

function AdjustmentOriginBanner({
  rebillStrategy,
  requiresReplacementReview,
  rerateVariancePercent,
  sourceInvoiceId,
  sourceCreditMemoInvoiceId,
  sourceInvoiceAdjustmentId,
  correctionGroupId,
}: {
  rebillStrategy?: string | null;
  requiresReplacementReview?: boolean;
  rerateVariancePercent?: number | null;
  sourceInvoiceId?: string | null;
  sourceCreditMemoInvoiceId?: string | null;
  sourceInvoiceAdjustmentId?: string | null;
  correctionGroupId?: string | null;
}) {
  const t = useT();

  const [expanded, setExpanded] = useState(false);

  return (
    <div className="mx-4 mt-2 shrink-0 overflow-hidden rounded-lg border border-info-border bg-info-subtle">
      <button
        type="button"
        className="flex w-full items-center gap-2.5 px-3 py-2 text-left"
        onClick={() => setExpanded((prev) => !prev)}
      >
        <RefreshCwIcon className="size-3.5 shrink-0 text-info-foreground" />
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <span className="text-xs font-medium text-info-foreground">
            {t("Adjustment-origin rebill")}
          </span>
          {rebillStrategy ? (
            <span className="text-2xs rounded-md bg-info-subtle px-1.5 py-0.5 font-medium text-info-foreground">
              {rebillStrategy}
            </span>
          ) : null}
          {requiresReplacementReview ? (
            <span className="text-2xs rounded-md bg-warning-subtle px-1.5 py-0.5 font-medium text-warning-foreground">
              {t("Review required")}
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-1.5">
          {sourceInvoiceId ? (
            <Link
              to={`/billing/invoices?item=${sourceInvoiceId}`}
              className="text-2xs font-medium text-info-foreground hover:underline dark:text-info-foreground"
              onClick={(e) => e.stopPropagation()}
            >
              {t("Original")}
            </Link>
          ) : null}
          {sourceInvoiceId && sourceCreditMemoInvoiceId ? (
            <span className="text-info-foreground/30">/</span>
          ) : null}
          {sourceCreditMemoInvoiceId ? (
            <Link
              to={`/billing/invoices?item=${sourceCreditMemoInvoiceId}`}
              className="text-2xs font-medium text-info-foreground hover:underline dark:text-info-foreground"
              onClick={(e) => e.stopPropagation()}
            >
              {t("Credit memo")}
            </Link>
          ) : null}
          <ChevronDownIcon
            className={`size-3.5 text-info-foreground/50 transition-transform duration-150 dark:text-info-foreground/50 ${expanded ? "rotate-180" : ""}`}
          />
        </div>
      </button>
      {expanded ? (
        <div className="border-t border-info-border px-3 py-2">
          <div className="text-2xs flex flex-wrap gap-x-5 gap-y-1">
            {sourceInvoiceAdjustmentId ? (
              <span className="text-muted-foreground">
                {t("Adjustment")}{" "}
                <span className="text-foreground font-medium">
                  {sourceInvoiceAdjustmentId.slice(0, 12)}
                </span>
              </span>
            ) : null}
            {correctionGroupId ? (
              <span className="text-muted-foreground">
                {t("Group")}{" "}
                <span className="text-foreground font-medium">
                  {correctionGroupId.slice(0, 12)}
                </span>
              </span>
            ) : null}
            {rerateVariancePercent != null ? (
              <span className="text-muted-foreground">
                {t("Rerate variance")}{" "}
                <span className="text-foreground font-medium">
                  {Number(rerateVariancePercent).toFixed(2)}%
                </span>
              </span>
            ) : null}
            {requiresReplacementReview ? (
              <span className="text-muted-foreground">
                {t("Replacement review")}{" "}
                <span className="text-foreground font-medium">
                  {t("Required before invoice creation")}
                </span>
              </span>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ReviewTimer({ startedAt }: { startedAt: number }) {
  const [elapsed, setElapsed] = useState(() => Math.floor(Date.now() / 1000) - startedAt);

  useEffect(() => {
    setElapsed(Math.floor(Date.now() / 1000) - startedAt);
    const interval = setInterval(() => {
      setElapsed(Math.floor(Date.now() / 1000) - startedAt);
    }, 1000);
    return () => clearInterval(interval);
  }, [startedAt]);

  const hours = Math.floor(elapsed / 3600);
  const minutes = Math.floor((elapsed % 3600) / 60);
  const seconds = elapsed % 60;

  const pad = (n: number) => String(n).padStart(2, "0");

  return (
    <span className="text-muted-foreground inline-flex items-center gap-1 tabular-nums">
      <TimerIcon className="size-3" />
      {pad(hours)}:{pad(minutes)}:{pad(seconds)}
    </span>
  );
}
