import AuditTab from "@/components/audit-tab";
import { PlainBillingQueueStatusBadge } from "@/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { exceptionReasonLabels } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { formatCurrency } from "@/lib/utils";
import ShipmentCommentsTab from "@/routes/shipment/_components/shipment-comments";
import type { ExceptionReasonCode } from "@/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ClipboardListIcon,
  RefreshCwIcon,
  TimerIcon,
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { BillingQueueActionBar } from "./billing-queue-action-bar";
import { BillingQueueAssignDialog } from "./billing-queue-assign-dialog";
import { BillingQueueChargesTab } from "./billing-queue-charges-tab";
import { BillingQueueDocumentsTab } from "./billing-queue-documents-tab";

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

  const { data: item, isLoading } = useQuery({
    ...queries.billingQueue.get(selectedItemId ?? ""),
    enabled: !!selectedItemId,
  });

  const handleAssignBiller = useCallback(() => {
    setAssignDialogOpen(true);
  }, []);

  if (!selectedItemId) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-4 text-muted-foreground">
        <ClipboardListIcon className="size-12" />
        <div className="text-center">
          <p className="text-sm font-medium">No item selected</p>
          <p className="mt-1 text-xs">Select a billing queue item from the sidebar to review it</p>
        </div>
      </div>
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
  const proNumber = shipment?.proNumber ?? item.shipmentId.slice(0, 12);
  const totalCharge = Number(shipment?.totalChargeAmount ?? 0);
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

        <div className="flex items-baseline gap-3">
          <span className="text-2xl font-bold tabular-nums">{formatCurrency(totalCharge)}</span>
          <span className="text-sm text-muted-foreground">{customerName}</span>
        </div>

        <div className="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-3 lg:grid-cols-4">
          {item.number ? <MetadataCell label="Queue #" value={item.number} /> : null}
          {shipment?.bol ? <MetadataCell label="BOL" value={shipment.bol} /> : null}
          {item.assignedBiller ? (
            <MetadataCell label="Assigned Biller" value={item.assignedBiller.name} />
          ) : null}
          {originLocation && destLocation ? (
            <MetadataCell
              label="Route"
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
            <AlertTitle>Billing Notes</AlertTitle>
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
        <TabsList variant="underline" className="w-full border-b border-border">
          <TabsTrigger value="charges">Charges</TabsTrigger>
          <TabsTrigger value="documents">Documents</TabsTrigger>
          <TabsTrigger value="comments">Comments</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
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
          <ShipmentCommentsTab shipmentId={item.shipmentId} />
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
      <p className="text-xs text-muted-foreground">{label}</p>
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
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="mx-4 mt-2 shrink-0 overflow-hidden rounded-lg border border-blue-600/20 bg-blue-600/5">
      <button
        type="button"
        className="flex w-full items-center gap-2.5 px-3 py-2 text-left"
        onClick={() => setExpanded((prev) => !prev)}
      >
        <RefreshCwIcon className="size-3.5 shrink-0 text-blue-600 dark:text-blue-400" />
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
            Adjustment-Origin Rebill
          </span>
          {rebillStrategy ? (
            <span className="rounded bg-blue-600/10 px-1.5 py-0.5 text-2xs font-medium text-blue-600 dark:text-blue-400">
              {rebillStrategy}
            </span>
          ) : null}
          {requiresReplacementReview ? (
            <span className="rounded bg-yellow-600/10 px-1.5 py-0.5 text-2xs font-medium text-yellow-700 dark:text-yellow-400">
              Review required
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-1.5">
          {sourceInvoiceId ? (
            <Link
              to={`/billing/invoices?item=${sourceInvoiceId}`}
              className="text-2xs font-medium text-blue-600 hover:underline dark:text-blue-400"
              onClick={(e) => e.stopPropagation()}
            >
              Original
            </Link>
          ) : null}
          {sourceInvoiceId && sourceCreditMemoInvoiceId ? (
            <span className="text-blue-600/30 dark:text-blue-400/30">/</span>
          ) : null}
          {sourceCreditMemoInvoiceId ? (
            <Link
              to={`/billing/invoices?item=${sourceCreditMemoInvoiceId}`}
              className="text-2xs font-medium text-blue-600 hover:underline dark:text-blue-400"
              onClick={(e) => e.stopPropagation()}
            >
              Credit Memo
            </Link>
          ) : null}
          <ChevronDownIcon
            className={`size-3.5 text-blue-600/50 transition-transform duration-150 dark:text-blue-400/50 ${expanded ? "rotate-180" : ""}`}
          />
        </div>
      </button>
      {expanded ? (
        <div className="border-t border-blue-600/10 px-3 py-2">
          <div className="flex flex-wrap gap-x-5 gap-y-1 text-2xs">
            {sourceInvoiceAdjustmentId ? (
              <span className="text-muted-foreground">
                Adjustment{" "}
                <span className="font-medium text-foreground">
                  {sourceInvoiceAdjustmentId.slice(0, 12)}
                </span>
              </span>
            ) : null}
            {correctionGroupId ? (
              <span className="text-muted-foreground">
                Group{" "}
                <span className="font-medium text-foreground">
                  {correctionGroupId.slice(0, 12)}
                </span>
              </span>
            ) : null}
            {rerateVariancePercent != null ? (
              <span className="text-muted-foreground">
                Rerate variance{" "}
                <span className="font-medium text-foreground">
                  {Number(rerateVariancePercent).toFixed(2)}%
                </span>
              </span>
            ) : null}
            {requiresReplacementReview ? (
              <span className="text-muted-foreground">
                Replacement review{" "}
                <span className="font-medium text-foreground">
                  Required before invoice creation
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
    <span className="inline-flex items-center gap-1 text-muted-foreground tabular-nums">
      <TimerIcon className="size-3" />
      {pad(hours)}:{pad(minutes)}:{pad(seconds)}
    </span>
  );
}
