import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { ShipmentCommodity } from "@/types/shipment";
import { faPencil, faTrash } from "@fortawesome/pro-solid-svg-icons";
import { CSSProperties, memo } from "react";

function CommodityRow({
  index,
  shipmentCommodity,
  style,
  isLast,
  onEdit,
  onDelete,
}: {
  index: number;
  shipmentCommodity: ShipmentCommodity;
  style: CSSProperties;
  isLast: boolean;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
}) {
  if (!shipmentCommodity.commodity)
    return (
      <div className="col-span-12 text-center text-sm text-muted-foreground">
        Unable to load commodity
      </div>
    );

  // Create a memoization key based on the commodity data
  const memoKey = `${shipmentCommodity.commodityId}-${shipmentCommodity.pieces}-${shipmentCommodity.weight}`;

  return (
    <div
      key={memoKey}
      className={cn(
        "grid grid-cols-10 gap-4 p-2 text-sm",
        !isLast && "border-b border-border",
      )}
      style={style}
    >
      <div className="col-span-4">
        <EntityRedirectLink
          entityId={shipmentCommodity.commodity.id}
          baseUrl="/shipments/configurations/commodities"
          modelOpen
          value={shipmentCommodity.commodity.name}
        >
          {shipmentCommodity.commodity.name}
        </EntityRedirectLink>
      </div>
      <div className="col-span-2 text-left">{shipmentCommodity.pieces}</div>
      <div className="col-span-2 text-left">{shipmentCommodity.weight}</div>
      <div className="col-span-2 flex gap-0.5 justify-end">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="xs"
                onClick={(e) => {
                  e.preventDefault();
                  onEdit(index);
                }}
              >
                <Icon icon={faPencil} className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Edit Commodity</TooltipContent>
          </Tooltip>
        </TooltipProvider>

        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger>
              <Button
                type="button"
                variant="ghost"
                className="hover:bg-red-500/30 text-red-600 hover:text-red-600"
                size="xs"
                onClick={(e) => {
                  e.preventDefault();
                  onDelete(index);
                }}
              >
                <Icon icon={faTrash} className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Delete Commodity</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}

CommodityRow.displayName = "CommodityRow";

export const MemoizedCommodityRow = memo(
  CommodityRow,
  (prevProps, nextProps) => {
    const prevCommodity = prevProps.shipmentCommodity;
    const nextCommodity = nextProps.shipmentCommodity;

    return (
      prevCommodity.commodityId === nextCommodity.commodityId &&
      prevCommodity.pieces === nextCommodity.pieces &&
      prevCommodity.weight === nextCommodity.weight
    );
  },
);
