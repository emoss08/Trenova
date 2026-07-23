import { EDITransferStatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { EDIMappingResolution, EDITransfer } from "@/types/edi";
import {
  ArrowRightIcon,
  CalendarClockIcon,
  DollarSignIcon,
  MapPinIcon,
  PackageIcon,
  RouteIcon,
} from "lucide-react";
import { Link } from "react-router";
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
  sourceValueLabel,
} from "../edi-display-utils";
import { EDIEmptyState, InfoTile } from "./edi-panel-primitives";

type TenderReviewProps = {
  transfer: EDITransfer;
  mappingRows: EDIMappingResolution[];
};

export function TransferOverview({ transfer, mappingRows }: TenderReviewProps) {
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
          <InfoTile
            label="Charges"
            value={
              payload.additionalCharges?.length === undefined ||
              payload.additionalCharges?.length === 0
                ? "N/A"
                : payload.additionalCharges?.length.toLocaleString()
            }
          />
        </div>
      </div>
    </div>
  );
}

export function TenderRouteReview({ transfer, mappingRows }: TenderReviewProps) {
  const moves = transfer.tenderPayload.moves;

  if (moves.length === 0) {
    return <EDIEmptyState message="No moves were included in this load tender." />;
  }

  return (
    <div className={cn("grid gap-3 lg:grid-cols-1", moves.length > 1 && "lg:grid-cols-2")}>
      {moves?.map((move) => {
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
      {!isLast && <div className="absolute top-7 -bottom-3 left-[13.5px] w-px bg-border" />}
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

export function TenderFreightReview({ transfer, mappingRows }: TenderReviewProps) {
  const payload = transfer.tenderPayload;

  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <ReviewSection
        icon={<PackageIcon className="size-4" />}
        title="Commodities"
        count={payload.commodities?.length ?? 0}
        empty="No commodities were included in this tender."
      >
        {payload.commodities?.map((commodity) => {
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
        count={payload.additionalCharges?.length ?? 0}
        empty="No additional charges were included in this tender."
      >
        {payload.additionalCharges?.map((charge) => {
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
        {count === 0 ? <EDIEmptyState message={empty} /> : children}
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
