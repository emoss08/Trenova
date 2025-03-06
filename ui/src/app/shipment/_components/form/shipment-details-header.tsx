import { ShipmentStatusBadge } from "@/components/status-badge";
import { formatToUserTimezone } from "@/lib/date";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useUser } from "@/stores/user-store";
import { useFormContext } from "react-hook-form";

export default function ShipmentDetailsHeader() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const user = useUser();

  const { proNumber, status, updatedAt } = getValues();

  return (
    <div className="flex flex-col px-4 pb-2 border-b border-bg-sidebar-border">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold leading-none tracking-tight flex items-center gap-x-2">
          {proNumber || "Add New Shipment"}
        </h2>
        <ShipmentStatusBadge status={status} />
      </div>
      {updatedAt ? (
        <p className="text-2xs text-muted-foreground font-normal">
          Last updated on{" "}
          {formatToUserTimezone(updatedAt, {
            timezone: user?.timezone,
            timeFormat: user?.timeFormat,
          })}
        </p>
      ) : (
        <p className="text-2xs text-muted-foreground font-normal">
          Please fill out the form below to create a new shipment.
        </p>
      )}
    </div>
  );
}
