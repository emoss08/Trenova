/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import type { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import { cn } from "@/lib/utils";
import { faPencil, faTrash } from "@fortawesome/pro-solid-svg-icons";
import { memo } from "react";

export function CommodityRow({
  index,
  shipmentCommodity,
  isLast,
  onEdit,
  onDelete,
}: {
  index: number;
  shipmentCommodity: ShipmentCommoditySchema;
  isLast: boolean;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
}) {
  if (!shipmentCommodity.commodity)
    return (
      <div className="col-span-12 text-center text-sm text-muted-foreground">
        Unable to display commodity due to commodity not being found
      </div>
    );

  return (
    <CommodityRowInner isLast={isLast}>
      <CommodityRowContent>
        <CommodityRedirectLink shipmentCommodity={shipmentCommodity} />
      </CommodityRowContent>
      <CommodityRowInformation
        pieces={shipmentCommodity.pieces}
        weight={shipmentCommodity.weight}
      />
      <CommodityRowActions index={index} onEdit={onEdit} onDelete={onDelete} />
    </CommodityRowInner>
  );
}

function CommodityRowContent({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-2 col-span-4">{children}</div>;
}

function CommodityRedirectLink({
  shipmentCommodity,
}: {
  shipmentCommodity: ShipmentCommoditySchema;
}) {
  if (!shipmentCommodity.commodity) return null;

  return (
    <EntityRedirectLink
      entityId={shipmentCommodity.commodity.id}
      baseUrl="/shipments/configurations/commodities"
      modelOpen
    >
      {shipmentCommodity.commodity.name}
    </EntityRedirectLink>
  );
}

const CommodityRowInformation = memo(function CommodityRowInformation({
  pieces,
  weight,
}: {
  pieces: number;
  weight: number;
}) {
  return (
    <>
      <div className="col-span-2 text-left">{pieces}</div>
      <div className="col-span-2 text-left">{weight}</div>
    </>
  );
});

function CommodityRowInner({
  isLast,
  children,
}: {
  isLast: boolean;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-10 gap-4 p-2 text-sm",
        !isLast && "border-b",
      )}
    >
      {children}
    </div>
  );
}

const CommodityRowActions = memo(function CommodityRowActions({
  index,
  onEdit,
  onDelete,
}: {
  index: number;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
}) {
  return (
    <div className="col-span-2 flex gap-0.5 justify-end">
      <Button
        type="button"
        variant="ghost"
        size="xs"
        title="Edit Commodity"
        onClick={(e) => {
          e.preventDefault();
          onEdit(index);
        }}
      >
        <Icon icon={faPencil} className="size-4" />
      </Button>

      <Button
        type="button"
        variant="ghost"
        className="hover:bg-red-500/30 text-red-600 hover:text-red-600"
        size="xs"
        title="Delete Commodity"
        onClick={(e) => {
          e.preventDefault();
          onDelete(index);
        }}
      >
        <Icon icon={faTrash} className="size-4" />
      </Button>
    </div>
  );
});
