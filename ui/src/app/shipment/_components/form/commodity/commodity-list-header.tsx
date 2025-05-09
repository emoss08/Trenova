import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { memo } from "react";

export function CommodityListHeader({
  commodities,
  handleAddCommodity,
}: {
  commodities: ShipmentSchema["commodities"];
  handleAddCommodity: () => void;
}) {
  return (
    <CommodityListHeaderInner>
      <CommodityListHeaderDetails commodities={commodities} />
      <AddCommodityButton onClick={handleAddCommodity} />
    </CommodityListHeaderInner>
  );
}

function CommodityListHeaderInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-between">{children}</div>;
}

function CommodityListHeaderDetails({
  commodities,
}: {
  commodities: ShipmentSchema["commodities"];
}) {
  return (
    <div className="flex items-center gap-1">
      <h3 className="text-sm font-medium">Commodities</h3>
      <span className="text-2xs text-muted-foreground">
        ({commodities?.length ?? 0})
      </span>
    </div>
  );
}

const AddCommodityButton = memo(function AddCommodityButton({
  onClick,
}: {
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="outline" size="xs" onClick={onClick}>
      <Icon icon={faPlus} className="size-4" />
      Add Commodity
    </Button>
  );
});
