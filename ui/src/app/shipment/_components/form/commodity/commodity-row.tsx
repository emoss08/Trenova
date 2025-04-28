import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import { cn } from "@/lib/utils";
import { ShipmentCommodity } from "@/types/shipment";
import { faPencil, faTrash } from "@fortawesome/pro-solid-svg-icons";
import { memo } from "react";

const ROW_HEIGHT = 38;

export function CommodityRow({
  index,
  shipmentCommodity,
  isLast,
  onEdit,
  onDelete,
}: {
  index: number;
  shipmentCommodity: ShipmentCommodity;
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

  // Check for errors specific to this commodity
  // const commodityError = errors.commodities?.[index]?.commodityId?.message;
  // const hasError = !!commodityError;

  // Create a memoization key based on the commodity data
  const memoKey = `${shipmentCommodity.commodityId}-${shipmentCommodity.pieces}-${shipmentCommodity.weight}`;

  return (
    <CommodityRowInner key={memoKey} isLast={isLast}>
      <CommodityRowContent>
        <CommodityRedirectLink shipmentCommodity={shipmentCommodity} />
        {/* {hasError && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="text-red-600">
                <Icon icon={faTriangleExclamation} className="size-4 mb-0.5" />
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
        )} */}
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
  shipmentCommodity: ShipmentCommodity;
}) {
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
  // hasError,
  children,
}: {
  isLast: boolean;
  // hasError: boolean;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-10 gap-4 p-2 text-sm",
        !isLast && "border-b",
        // hasError && "bg-red-600/30",
      )}
      style={{ height: ROW_HEIGHT }}
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
