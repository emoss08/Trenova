import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import React, { memo } from "react";

export function AdditionalChargeListHeader({
  additionalCharges,
  handleAddAdditionalCharge,
}: {
  additionalCharges: ShipmentSchema["additionalCharges"];
  handleAddAdditionalCharge: () => void;
}) {
  return (
    <AdditionalChargeListHeaderInner>
      <AdditionalChargeHeaderDetails additionalCharges={additionalCharges} />
      <AddAdditionalChargeButton onClick={handleAddAdditionalCharge} />
    </AdditionalChargeListHeaderInner>
  );
}

function AdditionalChargeHeaderDetails({
  additionalCharges,
}: {
  additionalCharges: ShipmentSchema["additionalCharges"];
}) {
  return (
    <div className="flex items-center gap-1">
      <h3 className="text-sm font-medium">Additional Charges</h3>
      <span className="text-2xs text-muted-foreground">
        ({additionalCharges?.length ?? 0})
      </span>
    </div>
  );
}

function AdditionalChargeListHeaderInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex items-center justify-between">{children}</div>;
}

const AddAdditionalChargeButton = memo(function AddAdditionalChargeButton({
  onClick,
}: {
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="outline" size="xs" onClick={onClick}>
      <Icon icon={faPlus} className="size-4" />
      Add Additional Charge
    </Button>
  );
});

AddAdditionalChargeButton.displayName = "AddAdditionalChargeButton";
