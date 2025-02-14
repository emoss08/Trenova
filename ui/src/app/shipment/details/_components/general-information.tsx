import DoubleClickInput from "@/components/fields/input-field";
import { PlainShipmentStatusBadge } from "@/components/status-badge";
import { formatDate, toDate } from "@/lib/date";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import { Path, useFormContext } from "react-hook-form";

interface DetailItemProps {
  label: string;

  fieldName?: Path<ShipmentSchema>;
  value?: React.ReactNode;
  className?: string;
}

function DetailItem({ label, fieldName, value, className }: DetailItemProps) {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className={cn("space-y-1", className)}>
      <dt className="text-sm font-medium text-muted-foreground uppercase">
        {label}
      </dt>
      <dd className="text-sm text-foreground max-h-4">
        {fieldName ? (
          <DoubleClickInput
            control={control}
            name={fieldName}
            displayClassName="text-foreground"
          />
        ) : (
          value
        )}
      </dd>
    </div>
  );
}

export default function GeneralInformation() {
  return (
    <div>
      <ShipmentDetails />
    </div>
  );
}

function ShipmentDetails() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const { proNumber } = getValues();

  return (
    <div className="space-y-4">
      <h3 className="text-4xl font-semibold">{proNumber ?? "CL-2467802"}</h3>
      <ShipmentStats />
    </div>
  );
}
function ShipmentStats() {
  const { getValues } = useFormContext<Shipment>();
  const { createdAt, status } = getValues();

  const createdAtDate = toDate(createdAt);
  const formatedCreatedAt = createdAtDate ? formatDate(createdAtDate) : "-";

  return (
    <dl className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <div className="space-y-1">
        <dt className="text-sm font-medium text-muted-foreground uppercase">
          Status
        </dt>
        <dd className="text-sm text-foreground max-h-4">
          <PlainShipmentStatusBadge status={status} />
        </dd>
      </div>
      <DetailItem fieldName="bol" label="BOL Number" />
      <DetailItem label="Created At" value={formatedCreatedAt} />
      <DetailItem label="CURRENT ODO" value="109,000 mi." />
    </dl>
  );
}
