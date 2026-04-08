import AuditTab from "@/components/audit-tab";
import { PlainBillingQueueStatusBadge } from "@/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { formatCurrency } from "@/lib/utils";
import ShipmentCommentsTab from "@/routes/shipment/_components/shipment-comments";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ClipboardListIcon, TimerIcon } from "lucide-react";
import { lazy, useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { BillingQueueActionBar } from "./billing-queue-action-bar";
import { BillingQueueAssignDialog } from "./billing-queue-assign-dialog";
import { BillingQueueChargesTab } from "./billing-queue-charges-tab";
import { BillingQueueDocumentsTab } from "./billing-queue-documents-tab";

const ShipmentRouteMap = lazy(() =>
  import("@/components/command-palette/_components/shipment/shipment-preview-map").then((m) => ({
    default: m.ShipmentRouteMap,
  })),
);

const EXCEPTION_REASON_LABELS: Record<string, string> = {
  MissingDocumentation: "Missing Documentation",
  IncorrectRates: "Incorrect Rates",
  WeightDiscrepancy: "Weight Discrepancy",
  AccessorialDispute: "Accessorial Dispute",
  DuplicateCharge: "Duplicate Charge",
  MissingReferenceNumber: "Missing Reference Number",
  CustomerInformationError: "Customer Information Error",
  ServiceFailure: "Service Failure",
  RateNotOnFile: "Rate Not On File",
  Other: "Other",
};

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
      <div className="flex h-full flex-col items-center justify-center gap-3 text-muted-foreground p-4">
        <ClipboardListIcon className="size-12" />
        <div className="text-center">
          <p className="text-sm font-medium">No item selected</p>
          <p className="text-xs mt-1">Select a billing queue item from the sidebar to review it</p>
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
      <div className="flex flex-col gap-2 border-b px-4 py-3 shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Link
              to={`/shipment-management/shipments?item=${item.shipmentId}`}
              className="text-lg font-semibold hover:underline"
            >
              {proNumber}
            </Link>
            <PlainBillingQueueStatusBadge status={item.status} />
          </div>
          <span className="text-lg font-bold tabular-nums">{formatCurrency(totalCharge)}</span>
        </div>
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <span>{customerName}</span>
          {item.number && (
            <>
              <Separator orientation="vertical" className="h-3" />
              <span className="font-mono">{item.number}</span>
            </>
          )}
          {shipment?.bol && (
            <>
              <Separator orientation="vertical" className="h-3" />
              <span>BOL: {shipment.bol}</span>
            </>
          )}
          {item.assignedBiller && (
            <>
              <Separator orientation="vertical" className="h-3" />
              <span>Biller: {item.assignedBiller.name}</span>
            </>
          )}
          {item.status === "InReview" && item.reviewStartedAt && (
            <>
              <Separator orientation="vertical" className="h-3" />
              <ReviewTimer startedAt={item.reviewStartedAt} />
            </>
          )}
        </div>
        {originLocation && destLocation && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span>
              {originLocation.city}, {originLocation.state?.abbreviation}
            </span>
            <span className="text-muted-foreground/50">→</span>
            <span>
              {destLocation.city}, {destLocation.state?.abbreviation}
            </span>
          </div>
        )}
      </div>
      {shipment?.moves && shipment.moves.length > 0 && (
        <div className="h-32 w-full shrink-0 border-b">
          <ShipmentRouteMap moves={shipment.moves} containerClassName="rounded-none border-b" />
        </div>
      )}
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
        <div className="px-4 pt-2 shrink-0">
          <Alert variant="destructive">
            <AlertTriangleIcon className="size-4" />
            <AlertTitle>
              {EXCEPTION_REASON_LABELS[item.exceptionReasonCode] ?? item.exceptionReasonCode}
            </AlertTitle>
            {item.exceptionNotes && <AlertDescription>{item.exceptionNotes}</AlertDescription>}
          </Alert>
        </div>
      )}
      <Tabs defaultValue="charges" className="flex flex-1 flex-col min-h-0">
        <TabsList variant="underline" className="w-full border-b border-border">
          <TabsTrigger value="charges">Charges</TabsTrigger>
          <TabsTrigger value="documents">Documents</TabsTrigger>
          <TabsTrigger value="comments">Comments</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
        </TabsList>
        <TabsContent value="charges" className="flex-1 min-h-0 mt-0">
          <ScrollArea className="h-full">
            <BillingQueueChargesTab item={item} />
          </ScrollArea>
        </TabsContent>
        <TabsContent value="documents" className="flex-1 min-h-0 mt-0">
          <BillingQueueDocumentsTab
            shipmentId={item.shipmentId}
            selectedDocumentId={selectedDocumentId ?? null}
            onDocumentSelect={onDocumentSelect}
            isEditable={item.status === "InReview"}
          />
        </TabsContent>
        <TabsContent value="comments" className="flex-1 min-h-0 mt-0">
          <ShipmentCommentsTab shipmentId={item.shipmentId} />
        </TabsContent>
        <TabsContent value="activity" className="flex-1 min-h-0 mt-0">
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
    <span className="inline-flex items-center gap-1 tabular-nums text-muted-foreground">
      <TimerIcon className="size-3" />
      {pad(hours)}:{pad(minutes)}:{pad(seconds)}
    </span>
  );
}
