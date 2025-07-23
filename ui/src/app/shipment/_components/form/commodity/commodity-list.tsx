/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import {
  faBoxesStacked,
  faTrailer,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { CommodityRow } from "./commodity-row";

function CommodityRowHeader() {
  return (
    <div className="sticky top-0 z-10 grid grid-cols-10 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
      <div className="col-span-4">Commodity</div>
      <div className="col-span-2 text-left">Pieces</div>
      <div className="col-span-2 text-left">Weight</div>
      <div className="col-span-2" />
    </div>
  );
}

export function CommodityList({
  commodities,
  handleEdit,
  handleDelete,
}: {
  commodities: ShipmentCommoditySchema[];
  handleEdit: (index: number) => void;
  handleDelete: (index: number) => void;
}) {
  return !commodities.length ? (
    <EmptyState
      className="max-h-[200px] p-4 border rounded-lg border-bg-sidebar-border"
      title="No Commodities"
      description="Shipment has no associated commodities"
      icons={[faTrailer, faBoxesStacked, faTruckContainer]}
    />
  ) : (
    <CommodityListInner>
      <CommodityRowHeader />
      <CommodityListScrollArea>
        {commodities.map((field, index) => (
          <CommodityRow
            key={field.id}
            shipmentCommodity={field}
            isLast={index === commodities.length - 1}
            onEdit={handleEdit}
            onDelete={handleDelete}
            index={index}
          />
        ))}
      </CommodityListScrollArea>
    </CommodityListInner>
  );
}

export function CommodityListScrollArea({
  children,
}: {
  children: React.ReactNode;
}) {
  return <ScrollArea className="flex max-h-40 flex-col">{children}</ScrollArea>;
}

export function CommodityListInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-bg-sidebar-border bg-card">
      {children}
    </div>
  );
}
