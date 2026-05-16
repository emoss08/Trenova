import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDIMappingProfileItem, EDIMappingResolution, EDITransfer } from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRightIcon,
  CalendarClockIcon,
  CheckIcon,
  DollarSignIcon,
  MapPinIcon,
  PackageIcon,
  RouteIcon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import {
  findMapping,
  formatAccessorialName,
  formatCommodityName,
  formatDecimalLike,
  formatMappingDetail,
  formatNumber,
  formatStopAddress,
  formatStopName,
  formatUnix,
  formatWeight,
  formatWindow,
  mappingKey,
  sourceValueLabel,
} from "../edi-display-utils";
import { EDITransferStatusBadge } from "../edi-transfer-status-badge";
import { TargetLookup } from "../edi-target-lookup";
import { EDITransferPanelContent } from "./edi-transfer-panel-content";

export function EDITransferReviewPanel({
  direction,
  open,
  onOpenChange,
  row: transfer,
}: DataTablePanelProps<EDITransfer> & {
  direction: "inbound" | "outbound";
}) {
  return (
    <EDITransferReviewPanelContent
      direction={direction}
      onOpenChange={onOpenChange}
      open={open}
      transfer={transfer}
    />
  );
}

function EDITransferReviewPanelContent({
  direction,
  open,
  onOpenChange,
  transfer,
}: {
  transfer: EDITransfer | null;
  direction: "inbound" | "outbound";
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [rejectReason, setRejectReason] = useState("");
  const [inlineMappings, setInlineMappings] = useState<Record<string, EDIMappingProfileItem>>({});
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const { data: preview } = useQuery({
    ...queries.edi.mappingPreview(transfer?.id ?? ""),
    enabled: !!transfer && direction === "inbound" && isTransferActionable(transfer.status),
  });
  const approveMutation = useApiMutation({
    mutationFn: () =>
      apiService.ediService.approveTransfer(transfer!.id, {
        mappings: Object.values(inlineMappings),
      }),
    onSuccess: async () => {
      toast.success("EDI transfer approval started.");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to approve transfer"),
  });

  const rejectMutation = useApiMutation({
    mutationFn: () => apiService.ediService.rejectTransfer(transfer!.id, { reason: rejectReason }),
    onSuccess: async () => {
      toast.success("EDI transfer rejected");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to reject transfer"),
  });

  const cancelMutation = useApiMutation({
    mutationFn: () => apiService.ediService.cancelTransfer(transfer!.id),
    onSuccess: async () => {
      toast.success("EDI transfer canceled");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to cancel transfer"),
  });
  const unresolved = preview?.unresolved ?? [];
  const mappingRows = preview?.all ?? transfer?.mappingSnapshot ?? [];
  const approvalReady = unresolved.every(
    (row) => inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId,
  );
  const isActionable = transfer?.status ? isTransferActionable(transfer.status) : false;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      size={direction === "inbound" ? "2xl" : "xl"}
      title={direction === "inbound" ? "Review Inbound Load Tender" : "Review Outbound Load Tender"}
      description={
        transfer?.tenderPayload.bol ? `BOL ${transfer.tenderPayload.bol}` : "Load tender"
      }
      footer={
        <>
          {transfer && canUpdate && direction === "inbound" && isActionable && (
            <div className="ml-auto flex gap-2">
              <Button
                variant="outline"
                disabled={!rejectReason.trim()}
                isLoading={rejectMutation.isPending}
                onClick={() => rejectMutation.mutate(undefined)}
              >
                <XIcon data-icon="inline-start" />
                Reject
              </Button>
              <Button
                disabled={!approvalReady}
                isLoading={approveMutation.isPending}
                onClick={() => approveMutation.mutate(undefined)}
              >
                <CheckIcon data-icon="inline-start" />
                Approve
              </Button>
            </div>
          )}
          {transfer && canUpdate && direction === "outbound" && isActionable && (
            <Button
              className="ml-auto"
              variant="outline"
              isLoading={cancelMutation.isPending}
              onClick={() => cancelMutation.mutate(undefined)}
            >
              Cancel Transfer
            </Button>
          )}
        </>
      }
    >
      {transfer && (
        <EDITransferPanelContent transfer={transfer}>
          <TransferOverview transfer={transfer} mappingRows={mappingRows} />
          <Tabs defaultValue="tender" className="min-h-0 gap-3">
            <TabsList variant="underline" className="w-full border-b border-border">
              <TabsTrigger value="tender">
                <RouteIcon data-icon="inline-start" />
                Tender
              </TabsTrigger>
              <TabsTrigger value="freight">
                <PackageIcon data-icon="inline-start" />
                Freight
              </TabsTrigger>
              <TabsTrigger value="mappings">
                <ArrowRightIcon data-icon="inline-start" />
                Mappings
              </TabsTrigger>
            </TabsList>
            <TabsContent value="tender" className="mt-0 space-y-3">
              <TenderRouteReview transfer={transfer} mappingRows={mappingRows} />
            </TabsContent>
            <TabsContent value="freight" className="mt-0 space-y-3">
              <TenderFreightReview transfer={transfer} mappingRows={mappingRows} />
            </TabsContent>
            <TabsContent value="mappings" className="mt-0 space-y-3">
              <MappingReview
                direction={direction}
                isActionable={isActionable}
                inlineMappings={inlineMappings}
                mappingRows={mappingRows}
                rejectReason={rejectReason}
                setInlineMappings={setInlineMappings}
                setRejectReason={setRejectReason}
                unresolved={unresolved}
              />
            </TabsContent>
          </Tabs>
        </EDITransferPanelContent>
      )}
    </DataTablePanelContainer>
  );
}

function TransferOverview({
  transfer,
  mappingRows,
}: {
  transfer: EDITransfer;
  mappingRows: EDIMappingResolution[];
}) {
  const payload = transfer.tenderPayload;
  const stopsCount = payload.moves.reduce((count, move) => count + move.stops.length, 0);
  const unresolvedCount = mappingRows.filter((row) => !row.resolved).length;

  return (
    <div className="rounded-md border bg-muted/20">
      <div className="grid gap-4 p-4 lg:grid-cols-[1.4fr_1fr]">
        <div className="min-w-0 space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <EDITransferStatusBadge status={transfer.status} />
            <Badge variant={unresolvedCount > 0 ? "outline" : "active"}>
              {unresolvedCount > 0 ? `${unresolvedCount} unresolved mappings` : "Ready to accept"}
            </Badge>
          </div>
          <div>
            <div className="truncate text-base font-semibold">{payload.bol || "Load tender"}</div>
            <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span>Submitted {formatUnix(transfer.submittedAt)}</span>
              <span>
                Target{" "}
                {transfer.targetShipmentId ? (
                  <Link to={`/shipment-management/shipments?item=${transfer.targetShipmentId}`}>
                    Open shipment
                  </Link>
                ) : (
                  "pending"
                )}
              </span>
            </div>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2 md:grid-cols-4 lg:grid-cols-4">
          <InfoTile
            label="Customer"
            value={sourceValueLabel(payload.customerLabel, payload.customerId)}
          />
          <InfoTile
            label="Service"
            value={sourceValueLabel(payload.serviceTypeLabel, payload.serviceTypeId)}
          />
          <InfoTile
            label="Shipment Type"
            value={sourceValueLabel(payload.shipmentTypeLabel, payload.shipmentTypeId)}
          />
          <InfoTile
            label="Rating Template"
            value={sourceValueLabel(payload.formulaTemplateLabel, payload.formulaTemplateId)}
          />
          <InfoTile
            label="Route"
            value={`${payload.moves.length} / ${stopsCount}`}
            hint="moves / stops"
          />
          <InfoTile label="Pieces" value={formatNumber(payload.pieces)} />
          <InfoTile label="Weight" value={formatWeight(payload.weight)} />
          <InfoTile label="Charges" value={payload.additionalCharges.length.toLocaleString()} />
        </div>
      </div>
    </div>
  );
}

function TenderRouteReview({
  transfer,
  mappingRows,
}: {
  transfer: EDITransfer;
  mappingRows: EDIMappingResolution[];
}) {
  const moves = transfer.tenderPayload.moves;

  if (moves.length === 0) {
    return <EmptyReviewState message="No moves were included in this load tender." />;
  }

  return (
    <div className="grid gap-3 lg:grid-cols-2">
      {moves.map((move) => {
        const origin = move.stops[0];
        const destination = move.stops[move.stops.length - 1];
        const originMapping = origin
          ? findMapping(mappingRows, "Location", origin.locationId)
          : undefined;
        const destinationMapping = destination
          ? findMapping(mappingRows, "Location", destination.locationId)
          : undefined;

        return (
          <div key={`move-${move.sequence}`} className="rounded-md border bg-background">
            <div className="flex flex-wrap items-start justify-between gap-3 border-b px-3 py-2">
              <div className="flex min-w-0 items-start gap-2">
                <div className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md bg-muted">
                  <RouteIcon className="size-3.5 text-muted-foreground" />
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-medium">Move {move.sequence + 1}</div>
                  <div className="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
                    <span className="truncate">{formatStopName(origin, originMapping)}</span>
                    <ArrowRightIcon className="size-3 shrink-0" />
                    <span className="truncate">
                      {formatStopName(destination, destinationMapping)}
                    </span>
                  </div>
                </div>
              </div>
              <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                <Badge variant="outline">{move.loaded ? "Loaded" : "Empty"}</Badge>
                <Badge variant="outline">{move.stops.length} stops</Badge>
                {move.distance && (
                  <Badge variant="outline">{move.distance.toLocaleString()} mi</Badge>
                )}
              </div>
            </div>
            <div className="space-y-3 p-3">
              {move.stops.map((stop, index) => (
                <TenderStopCard
                  key={`${move.sequence}-${stop.sequence}-${stop.locationId}`}
                  mapping={findMapping(mappingRows, "Location", stop.locationId)}
                  stop={stop}
                  isLast={index === move.stops.length - 1}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function TenderFreightReview({
  transfer,
  mappingRows,
}: {
  transfer: EDITransfer;
  mappingRows: EDIMappingResolution[];
}) {
  const payload = transfer.tenderPayload;

  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <ReviewSection
        icon={<PackageIcon className="size-4" />}
        title="Commodities"
        count={payload.commodities.length}
        empty="No commodities were included in this tender."
      >
        {payload.commodities.map((commodity) => {
          const mapping = findMapping(mappingRows, "Commodity", commodity.commodityId);
          return (
            <FreightLine
              key={commodity.commodityId}
              primary={formatCommodityName(commodity, mapping)}
              secondary={formatMappingDetail(mapping)}
              values={[formatWeight(commodity.weight), `${formatNumber(commodity.pieces)} pcs`]}
            />
          );
        })}
      </ReviewSection>
      <ReviewSection
        icon={<DollarSignIcon className="size-4" />}
        title="Additional Charges"
        count={payload.additionalCharges.length}
        empty="No additional charges were included in this tender."
      >
        {payload.additionalCharges.map((charge) => {
          const mapping = findMapping(mappingRows, "AccessorialCharge", charge.accessorialChargeId);
          return (
            <FreightLine
              key={charge.accessorialChargeId}
              primary={formatAccessorialName(charge, mapping)}
              secondary={formatMappingDetail(mapping)}
              values={[charge.method, `${formatDecimalLike(charge.amount)} x ${charge.unit}`]}
            />
          );
        })}
      </ReviewSection>
    </div>
  );
}

function ReviewSection({
  icon,
  title,
  count,
  empty,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  count: number;
  empty: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-md border bg-background">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <div className="flex items-center gap-2 text-sm font-medium">
          {icon}
          {title}
        </div>
        <Badge variant="outline">{count}</Badge>
      </div>
      <div className="space-y-2 p-3">
        {count === 0 ? <EmptyReviewState message={empty} /> : children}
      </div>
    </div>
  );
}

function FreightLine({
  primary,
  secondary,
  values,
}: {
  primary: string;
  secondary: string;
  values: string[];
}) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-md border bg-muted/20 p-3">
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{primary}</div>
        <div className="truncate text-xs text-muted-foreground">{secondary}</div>
      </div>
      <div className="flex flex-col items-end gap-1 text-xs text-muted-foreground">
        {values.map((value) => (
          <span key={value}>{value}</span>
        ))}
      </div>
    </div>
  );
}

function MappingReview({
  direction,
  inlineMappings,
  isActionable,
  mappingRows,
  rejectReason,
  setInlineMappings,
  setRejectReason,
  unresolved,
}: {
  direction: "inbound" | "outbound";
  inlineMappings: Record<string, EDIMappingProfileItem>;
  isActionable: boolean;
  mappingRows: EDIMappingResolution[];
  rejectReason: string;
  setInlineMappings: React.Dispatch<React.SetStateAction<Record<string, EDIMappingProfileItem>>>;
  setRejectReason: React.Dispatch<React.SetStateAction<string>>;
  unresolved: EDIMappingResolution[];
}) {
  if (direction !== "inbound" || !isActionable) {
    return <MappingSummary mappingRows={mappingRows} />;
  }

  return (
    <div className="flex flex-col gap-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="font-medium">Mapping Preview</div>
          <div className="text-xs text-muted-foreground">
            Resolve required mappings before accepting and creating the receiving shipment.
          </div>
        </div>
        <Badge variant={unresolved.length === 0 ? "active" : "outline"}>
          {unresolved.length === 0 ? "Ready" : `${unresolved.length} unresolved`}
        </Badge>
      </div>
      {unresolved.length === 0 ? (
        <MappingSummary mappingRows={mappingRows} />
      ) : (
        unresolved.map((row) => (
          <div
            key={mappingKey(row.entityType, row.sourceId)}
            className="grid gap-2 md:grid-cols-[1fr_1fr]"
          >
            <div className="rounded-md border bg-muted/20 p-3 text-sm">
              <div className="text-xs font-medium text-muted-foreground">Source value</div>
              <div className="mt-1 font-medium">{row.sourceLabel || "Unlabeled source value"}</div>
              <div className="mt-1 text-xs text-muted-foreground">{row.entityType}</div>
            </div>
            <TargetLookup
              label="Local record"
              entityType={row.entityType}
              value={inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId ?? ""}
              onChange={(target) => {
                const key = mappingKey(row.entityType, row.sourceId);
                setInlineMappings((current) => ({
                  ...current,
                  [key]: {
                    entityType: row.entityType,
                    sourceId: row.sourceId,
                    sourceLabel: row.sourceLabel ?? "",
                    targetId: target.targetId,
                    targetLabel: target.targetLabel,
                  },
                }));
              }}
            />
          </div>
        ))
      )}
      <Input
        placeholder="Rejection reason"
        value={rejectReason}
        onChange={(event) => setRejectReason(event.target.value)}
      />
    </div>
  );
}
function MappingSummary({ mappingRows }: { mappingRows: EDIMappingResolution[] }) {
  if (mappingRows.length === 0) {
    return <EmptyReviewState message="No mapping requirements were returned for this transfer." />;
  }

  return (
    <div className="grid gap-2">
      {mappingRows.map((row) => (
        <div
          key={mappingKey(row.entityType, row.sourceId)}
          className="rounded-md border bg-muted/20 p-3"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">{row.entityType}</span>
            <Badge variant={row.resolved ? "active" : "outline"}>
              {row.resolved ? "Resolved" : "Unresolved"}
            </Badge>
          </div>
          <div className="mt-3 grid gap-2 md:grid-cols-2">
            <div>
              <div className="text-xs font-medium text-muted-foreground">Source value</div>
              <div className="mt-1 truncate text-sm">
                {row.sourceLabel || "Unlabeled source value"}
              </div>
            </div>
            <div>
              <div className="text-xs font-medium text-muted-foreground">Local record</div>
              <div className="mt-1 truncate text-sm">
                {row.targetLabel || (row.resolved ? "Mapped local record" : "No mapping saved")}
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function InfoTile({
  label,
  value,
  hint,
}: {
  label: string;
  value: React.ReactNode;
  hint?: string;
}) {
  return (
    <div className="rounded-md border bg-background p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-semibold">{value}</div>
      {hint && <div className="mt-0.5 text-[10px] text-muted-foreground">{hint}</div>}
    </div>
  );
}

function TenderStopCard({
  stop,
  mapping,
  isLast,
}: {
  stop: EDITransfer["tenderPayload"]["moves"][number]["stops"][number];
  mapping?: EDIMappingResolution;
  isLast: boolean;
}) {
  const stopAddress = formatStopAddress(stop);

  return (
    <div className="relative grid grid-cols-[28px_1fr] gap-3">
      {!isLast && <div className="absolute top-7 bottom-[-0.75rem] left-[13.5px] w-px bg-border" />}
      <div className="relative z-10 flex flex-col items-center">
        <div className="flex size-7 items-center justify-center rounded-full border bg-muted">
          <MapPinIcon className="size-3.5" />
        </div>
      </div>
      <div className="rounded-md border bg-muted/20 p-3">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={stop.type === "Pickup" ? "active" : "secondary"}>{stop.type}</Badge>
              <span className="text-xs text-muted-foreground">Stop {stop.sequence + 1}</span>
              <Badge variant="outline">{stop.scheduleType}</Badge>
            </div>
            <div className="mt-2 truncate text-sm font-medium">{formatStopName(stop, mapping)}</div>
            {stopAddress && (
              <div className="mt-1 truncate text-xs text-muted-foreground">{stopAddress}</div>
            )}
            {mapping?.targetLabel && (
              <div className="mt-1 text-xs text-muted-foreground">
                Local record: <span className="text-foreground">{mapping.targetLabel}</span>
              </div>
            )}
          </div>
          <div className="grid gap-1 text-right text-xs">
            <span className="inline-flex items-center justify-end gap-1 text-muted-foreground">
              <CalendarClockIcon className="size-3" />
              {formatWindow(stop.scheduledWindowStart, stop.scheduledWindowEnd)}
            </span>
            <span>
              {formatWeight(stop.weight)} / {formatNumber(stop.pieces)} pcs
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function EmptyReviewState({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-3 py-6 text-center text-sm text-muted-foreground">
      {message}
    </div>
  );
}

function isTransferActionable(status: string) {
  return !["Approved", "Rejected", "Expired", "Canceled", "Failed", "Processing"].includes(status);
}

async function invalidateTransfers(queryClient: ReturnType<typeof useQueryClient>) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["edi-inbound-transfer-list"] }),
    queryClient.invalidateQueries({ queryKey: ["edi-outbound-transfer-list"] }),
    queryClient.invalidateQueries({ queryKey: queries.edi.inboundTransfers._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.outboundTransfers._def }),
  ]);
}
