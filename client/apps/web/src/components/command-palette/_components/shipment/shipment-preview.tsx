import { useT } from "@trenova/shared/i18n/use-t";
import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { formatSplitDateTime } from "@trenova/shared/lib/date";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { MoveStatus, Shipment, ShipmentMove, Stop } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { ArrowRightIcon } from "lucide-react";
import { PALETTE_ENTITIES } from "../../palette-entities";
import type { PaletteAction, PaletteIntent, PaletteRecord } from "../../palette-model";
import { shipmentPreviewQuery } from "../preview/preview-queries";
import {
  PreviewError,
  PreviewFrame,
  PreviewSection,
  PreviewSkeleton,
} from "../preview/preview-frame";
import { ShipmentRouteMap } from "./shipment-preview-map";

const moveStatusConfig: Record<
  MoveStatus,
  { label: string; variant: "neutral" | "info" | "warning" | "success" | "danger" }
> = {
  New: { label: "New", variant: "neutral" },
  Assigned: { label: "Assigned", variant: "info" },
  InTransit: { label: "In transit", variant: "info" },
  Completed: { label: "Completed", variant: "success" },
  Canceled: { label: "Canceled", variant: "danger" },
};

const stopDotColor: Record<MoveStatus, string> = {
  New: "bg-accent-violet",
  Assigned: "bg-info",
  InTransit: "bg-info",
  Completed: "bg-success",
  Canceled: "bg-danger",
};

export function ShipmentPreview({
  record,
  actions,
  onRun,
}: {
  record: PaletteRecord;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}) {
  const { data, isLoading, isError } = useQuery(shipmentPreviewQuery(record.id));

  if (isLoading) {
    return <PreviewSkeleton />;
  }

  if (isError || !data) {
    return <PreviewError />;
  }

  return <ShipmentPreviewContent shipment={data} actions={actions} onRun={onRun} />;
}

function formatCityState(stop: Stop | null | undefined): string | null {
  if (!stop?.location) return null;
  const city = stop.location.city;
  const stateAbbr = stop.location.state?.abbreviation;
  if (city && stateAbbr) return `${city}, ${stateAbbr}`;
  return city || stateAbbr || null;
}

function ShipmentPreviewContent({
  shipment,
  actions,
  onRun,
}: {
  shipment: Shipment;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}) {
  const t = useT();
  const entity = PALETTE_ENTITIES.shipment;

  const origin = getOriginStop(shipment);
  const destination = getDestinationStop(shipment);
  const originLabel = formatCityState(origin);
  const destLabel = formatCityState(destination);

  const totalMileage = shipment.moves.reduce((sum, m) => sum + (m.distance ?? 0), 0);
  const total = Number(shipment.totalChargeAmount);

  return (
    <PreviewFrame
      icon={entity.icon}
      tileClass={entity.tileClass}
      title={shipment.proNumber || shipment.id}
      subtitle={[
        shipment.customer?.name &&
          `${shipment.customer.name}${shipment.customer.code ? ` (${shipment.customer.code})` : ""}`,
        shipment.bol && t("BOL {0}", shipment.bol),
      ]
        .filter(Boolean)
        .join(" · ")}
      badge={<ShipmentStatusBadge status={shipment.status} />}
      actions={actions}
      onRun={onRun}
    >
      <ShipmentRouteMap
        moves={shipment.moves}
        containerClassName="h-36 rounded-surface border-border-subtle"
      />
      {(originLabel || destLabel) && (
        <PreviewSection title={t("Route")}>
          <div className="flex items-center gap-1.5 text-sm">
            <span className="truncate">{originLabel ?? "—"}</span>
            <ArrowRightIcon className="text-foreground-subtle size-3.5 shrink-0" />
            <span className="truncate">{destLabel ?? "—"}</span>
          </div>
        </PreviewSection>
      )}
      <PreviewSection title={t("Details")}>
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Pieces")} numeric>
            {shipment.pieces != null ? shipment.pieces.toLocaleString() : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Weight")} numeric>
            {shipment.weight != null ? (
              t("{0} lbs", shipment.weight.toLocaleString())
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Mileage")} numeric>
            {totalMileage > 0 ? t("{0} mi", totalMileage.toLocaleString()) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Total")} numeric>
            {Number.isFinite(total) && total > 0 ? formatCurrency(total) : <DescriptionEmpty />}
          </DescriptionItem>
        </DescriptionList>
      </PreviewSection>
      {shipment.moves.length > 0 && (
        <PreviewSection title={t("Moves")}>
          <div className="flex flex-col gap-2">
            {shipment.moves
              .slice()
              .sort((a, b) => a.sequence - b.sequence)
              .map((move) => (
                <MoveCard key={move.id} move={move} />
              ))}
          </div>
        </PreviewSection>
      )}
    </PreviewFrame>
  );
}

function MoveCard({ move }: { move: ShipmentMove }) {
  const t = useT();

  const config = moveStatusConfig[move.status];
  const sortedStops = [...move.stops].sort((a, b) => a.sequence - b.sequence);

  return (
    <div className="rounded-surface border-border-subtle bg-card border p-2.5">
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium">{t("Move {0}", move.sequence + 1)}</span>
          {move.distance != null && move.distance > 0 && (
            <span className="text-2xs text-foreground-subtle">
              {t("· {0} mi", move.distance.toLocaleString())}
            </span>
          )}
        </div>
        <Badge variant={config.variant} className="text-2xs">
          {t(config.label)}
        </Badge>
      </div>
      <div className="relative flex flex-col">
        {sortedStops.map((stop, idx) => {
          const dotColor = stopDotColor[move.status];
          const isLast = idx === sortedStops.length - 1;
          const windowStart = stop.scheduledWindowStart
            ? formatSplitDateTime(stop.scheduledWindowStart)
            : null;
          const windowEnd =
            stop.scheduledWindowEnd != null && stop.scheduledWindowEnd > 0
              ? formatSplitDateTime(stop.scheduledWindowEnd)
              : null;

          return (
            <div key={stop.id} className="relative flex gap-2.5 pb-4 last:pb-0">
              {!isLast && <div className="bg-border absolute top-3 -bottom-2 left-1 w-px" />}
              <div className="relative z-1 flex flex-col items-center">
                <div className={cn("mt-0.5 size-2.5 shrink-0 rounded-full", dotColor)} />
              </div>
              <div className="flex min-w-0 flex-col gap-0.5">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium">{stop.type}</span>
                  {stop.location?.name && (
                    <span className="text-2xs text-foreground-subtle truncate">
                      – {stop.location.name}
                    </span>
                  )}
                </div>
                {windowStart && (
                  <span className="text-2xs text-foreground-subtle">
                    {windowStart.date} · {windowStart.time}
                    {windowEnd &&
                      ` - ${windowEnd.date === windowStart.date ? windowEnd.time : `${windowEnd.date} · ${windowEnd.time}`}`}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
      {move.assignment && (move.assignment.tractor || move.assignment.primaryWorker) && (
        <div className="mt-2 grid grid-cols-2 gap-2 border-t pt-2">
          {move.assignment.tractor && (
            <div className="flex flex-col">
              <span className="text-2xs text-foreground-subtle">{t("Tractor")}</span>
              <span className="text-xs">{move.assignment.tractor.code}</span>
            </div>
          )}
          {move.assignment.primaryWorker && (
            <div className="flex flex-col">
              <span className="text-2xs text-foreground-subtle">{t("Worker")}</span>
              <span className="truncate text-xs">
                {move.assignment.primaryWorker.firstName}{" "}
                {move.assignment.primaryWorker.lastName?.charAt(0)}.
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
