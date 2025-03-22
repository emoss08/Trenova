import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { ShipmentCommodity } from "@/types/shipment";
import {
  faPencil,
  faTrash,
  faTriangleExclamation,
} from "@fortawesome/pro-solid-svg-icons";
import { CSSProperties, memo } from "react";
import { useFormContext } from "react-hook-form";

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
  const { formState } = useFormContext<ShipmentSchema>();
  const { errors } = formState;

  if (!shipmentCommodity.commodity)
    return (
      <div className="col-span-12 text-center text-sm text-muted-foreground">
        Unable to load commodity
      </div>
    );
  // Check for errors specific to this commodity
  const commodityError = errors.commodities?.[index]?.commodityId?.message;
  const hasError = !!commodityError;

  // Create a memoization key based on the commodity data
  const memoKey = `${shipmentCommodity.commodityId}-${shipmentCommodity.pieces}-${shipmentCommodity.weight}`;

  return (
    <div
      key={memoKey}
      className={cn(
        "grid grid-cols-10 gap-4 p-2 text-sm",
        !isLast && "border-b",
        hasError && "bg-red-600/30",
      )}
      style={style}
    >
      <div className="flex gap-2 col-span-4">
        <EntityRedirectLink
          entityId={shipmentCommodity.commodity.id}
          baseUrl="/shipments/configurations/commodities"
          modelOpen
          value={shipmentCommodity.commodity.name}
        >
          {shipmentCommodity.commodity.name}
        </EntityRedirectLink>
        {hasError && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-red-600">
                  <Icon
                    icon={faTriangleExclamation}
                    className="size-4 mb-0.5"
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent className="py-3">
                <div className="space-y-1">
                  <p className="text-[13px] font-medium">
                    Hazardous Material Segregation Violation
                  </p>
                  <p className="text-xs">{commodityError}</p>
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </div>
      <div className="col-span-2 text-left">{shipmentCommodity.pieces}</div>
      <div className="col-span-2 text-left">{shipmentCommodity.weight}</div>
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
      prevProps.isLast === nextProps.isLast &&
      prevCommodity.commodityId === nextCommodity.commodityId &&
      prevCommodity.pieces === nextCommodity.pieces &&
      prevCommodity.weight === nextCommodity.weight
    );
  },
);
