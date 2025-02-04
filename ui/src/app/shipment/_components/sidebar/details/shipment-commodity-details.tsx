import { EmptyState } from "@/components/ui/empty-state";
import { EntityRedirectLink } from "@/components/ui/link";
import { VirtualizedScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { Shipment, ShipmentCommodity } from "@/types/shipment";
import {
  faBoxesStacked,
  faTrailer,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CSSProperties, memo, useCallback, useRef } from "react";

const ROW_HEIGHT = 38;
const OVERSCAN = 5;

const CommodityRow = memo(function CommodityRow({
  shipmentCommodity,
  style,
  isLast,
}: {
  shipmentCommodity: ShipmentCommodity;
  style: CSSProperties;
  isLast: boolean;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-12 gap-4 p-2 text-sm",
        !isLast && "border-b border-border",
      )}
      style={style}
    >
      <div className="col-span-6">
        <EntityRedirectLink
          entityId={shipmentCommodity.commodity.id}
          baseUrl="/shipments/configurations/commodities"
          modelOpen
        >
          {shipmentCommodity.commodity.name}
        </EntityRedirectLink>
      </div>
      <div className="col-span-3 text-left">{shipmentCommodity.pieces}</div>
      <div className="col-span-3 text-left">{shipmentCommodity.weight}</div>
    </div>
  );
});

CommodityRow.displayName = "CommodityRow";

// Header component
const TableHeader = memo(() => (
  <div className="sticky top-0 z-10 grid grid-cols-12 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
    <div className="col-span-6">Commodity</div>
    <div className="col-span-3 text-left">Pieces</div>
    <div className="col-span-3 text-left">Weight</div>
  </div>
));

TableHeader.displayName = "TableHeader";

export function ShipmentCommodityDetails({
  shipment,
  className = "",
}: {
  shipment: Shipment;
  className?: string;
}) {
  const { commodities } = shipment;
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: commodities?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: useCallback(() => ROW_HEIGHT, []),
    overscan: OVERSCAN,
    enabled: !!commodities?.length,
  });

  if (!commodities?.length) {
    return (
      <div className="flex flex-col gap-2 border-y border-bg-sidebar-border py-4">
        <EmptyState
          className="max-h-[200px] p-4"
          title="No Commodities"
          description="Shipment has no associated commodities"
          icons={[faTrailer, faBoxesStacked, faTruckContainer]}
        />
      </div>
    );
  }

  return (
    <div
      className={cn(
        "flex flex-col gap-2 border-y border-bg-sidebar-border py-4",
        className,
      )}
    >
      <div className="flex items-center gap-1">
        <h3 className="text-sm font-medium">Commodities</h3>
        <span className="text-2xs text-muted-foreground">
          ({commodities.length ?? 0})
        </span>
      </div>

      <div className="rounded-lg border border-bg-sidebar-border bg-card">
        <TableHeader />
        <VirtualizedScrollArea
          ref={parentRef}
          className="flex max-h-40 flex-col"
        >
          <div
            className="relative w-full rounded-b-lg"
            style={{ height: `${virtualizer.getTotalSize()}px` }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const shipmentCommodity = commodities[virtualRow.index];
              return (
                <CommodityRow
                  key={shipmentCommodity.id}
                  shipmentCommodity={shipmentCommodity}
                  isLast={virtualRow.index === commodities.length - 1}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                />
              );
            })}
          </div>
        </VirtualizedScrollArea>
      </div>
    </div>
  );
}
