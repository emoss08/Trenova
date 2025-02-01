import { ShipmentStatusBadge } from "@/components/status-badge";
import { Icon } from "@/components/ui/icons";
import { InternalLink } from "@/components/ui/link";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import { faSignalStream } from "@fortawesome/pro-solid-svg-icons";
import { Timeline } from "./shipment-timeline";

export function ShipmentCard({ shipment }: { shipment: ShipmentResponse }) {
  const { status, customer } = shipment;

  return (
    <div className="p-2 border-b border-sidebar-border text-sm last:border-b-0">
      <div className="flex flex-col gap-2">
        <div className="flex justify-between w-full items-center">
          <ShipmentStatusBadge status={status} />
          <Icon icon={faSignalStream} className="size-4" />
        </div>
        <div className="flex justify-between gap-2">
          <ProNumber shipment={shipment} />
          <div className="flex items-center gap-2">
            <CustomerBadge customer={customer} />
          </div>
        </div>
        <StopInformation shipment={shipment} />
      </div>
    </div>
  );
}

function ProNumber({ shipment }: { shipment: ShipmentResponse }) {
  return (
    <div className="flex items-center gap-0.5">
      <InternalLink to={`/shipment/${shipment.id}`}>
        {shipment.proNumber}
      </InternalLink>
    </div>
  );
}

function CustomerBadge({ customer }: { customer: CustomerSchema }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <InternalLink
            to={{
              pathname: "/billing/configurations/customers",
              search: `?entityId=${customer.id}&modal=edit`,
            }}
            state={{
              isNavigatingToModal: true,
            }}
            className="text-muted-foreground underline hover:text-foreground/70"
            replace
            preventScrollReset
          >
            {customer.code}
          </InternalLink>
        </TooltipTrigger>
        <TooltipContent>
          <p>Click to view {customer.name}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function StopInformation({ shipment }: { shipment: ShipmentResponse }) {
  const { destination, origin } = ShipmentLocations.useLocations(shipment);

  if (!origin || !destination) {
    return <p>-</p>;
  }

  const items = [
    {
      id: "location-1",
      content: (
        <div className="rounded-lg">
          <p className="text-2xs text-muted-foreground">
            {formatLocation(origin)}
          </p>
        </div>
      ),
    },
    {
      id: "location-2",
      content: (
        <div className="rounded-lg">
          <p className="text-2xs text-muted-foreground">
            {formatLocation(destination)}
          </p>
        </div>
      ),
    },
  ];

  return <Timeline items={items} />;
}
