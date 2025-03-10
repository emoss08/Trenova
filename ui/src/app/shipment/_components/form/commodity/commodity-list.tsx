import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea, VirtualizedScrollArea } from "@/components/ui/scroll-area";
import { ShipmentCommodity } from "@/types/shipment";
import {
  faBoxesStacked,
  faTrailer,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { useVirtualizer } from "@tanstack/react-virtual";
import { memo, useCallback, useRef } from "react";
import { MemoizedCommodityRow } from "./commodity-row";

const ROW_HEIGHT = 38;
const OVERSCAN = 5;

const ListHeader = memo(() => (
  <div className="sticky top-0 z-10 grid grid-cols-10 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
    <div className="col-span-4">Commodity</div>
    <div className="col-span-2 text-left">Pieces</div>
    <div className="col-span-2 text-left">Weight</div>
    <div className="col-span-2" />
  </div>
));

ListHeader.displayName = "ListHeader";

export function CommodityList({
  commodities,
  handleEdit,
  handleDelete,
}: {
  commodities: ShipmentCommodity[];
  handleEdit: (index: number) => void;
  handleDelete: (index: number) => void;
}) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: commodities?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: useCallback(() => ROW_HEIGHT, []),
    overscan: OVERSCAN,
    enabled: !!commodities?.length,
  });

  return !commodities.length ? (
    <EmptyState
      className="max-h-[200px] p-4 border rounded-lg border-bg-sidebar-border bg-card"
      title="No Commodities"
      description="Shipment has no associated commodities"
      icons={[faTrailer, faBoxesStacked, faTruckContainer]}
    />
  ) : (
    <div className="rounded-lg border border-bg-sidebar-border bg-card">
      <ListHeader />
      {commodities.length > 20 ? (
        <VirtualizedScrollArea
          ref={parentRef}
          className="flex max-h-40 flex-col"
        >
          <div style={{ height: `${virtualizer.getTotalSize()}px` }}>
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const shipmentCommodity = commodities[virtualRow.index];
              return (
                <MemoizedCommodityRow
                  key={virtualRow.index} // Use virtualRow.index as key, not shipmentCommodity.id
                  shipmentCommodity={shipmentCommodity as ShipmentCommodity}
                  isLast={virtualRow.index === commodities.length - 1}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  index={virtualRow.index}
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
      ) : (
        <ScrollArea className="flex max-h-40 flex-col">
          {commodities.map((shipmentCommodity, index) => (
            <MemoizedCommodityRow
              key={index} // Use index as key, not shipmentCommodity.id
              shipmentCommodity={shipmentCommodity as ShipmentCommodity}
              isLast={index === commodities.length - 1}
              onEdit={handleEdit}
              onDelete={handleDelete}
              index={index}
              style={{ height: ROW_HEIGHT }}
            />
          ))}
        </ScrollArea>
      )}
    </div>
  );
}
