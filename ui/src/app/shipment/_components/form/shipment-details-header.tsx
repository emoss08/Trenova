import { ShipmentStatusBadge } from "@/components/status-badge";
import { formatToUserTimezone } from "@/lib/date";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useUser } from "@/stores/user-store";
import { useFormContext } from "react-hook-form";

export default function ShipmentDetailsHeader() {
  return (
    <ShipmentDetailsHeaderInner>
      <ShipmentDetailsHeaderTitle />
      <ShipmentDetailsHeaderDescription />
    </ShipmentDetailsHeaderInner>
  );
}

function ShipmentDetailsHeaderInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col px-4 pb-2 border-b border-bg-sidebar-border">
      {children}
    </div>
  );
}

function ShipmentDetailsHeaderTitle() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const { proNumber, status } = getValues();

  return (
    <div className="flex items-center justify-between">
      <h2 className="font-semibold leading-none tracking-tight flex items-center gap-x-2">
        {proNumber || "Add New Shipment"}
      </h2>
      <ShipmentStatusBadge status={status} />
    </div>
  );
}

function ShipmentDetailsHeaderDescription() {
  const { getValues } = useFormContext<ShipmentSchema>();

  const { updatedAt } = getValues();

  const user = useUser();

  return updatedAt ? (
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
  );
}
