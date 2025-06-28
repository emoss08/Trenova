import { BetaTag } from "@/components/ui/beta-tag";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/shadcn-table";
import { queries } from "@/lib/queries";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { formatLocation, USDollarFormat } from "@/lib/utils";
import type { TableSheetProps } from "@/types/data-table";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function PreviousRatesDialog(props: TableSheetProps) {
  const { getValues } = useFormContext<ShipmentSchema>();
  const shipment = getValues();
  const { origin, destination } = ShipmentLocations.useLocations(shipment);

  const { data: previousRates, isLoading } = useQuery({
    ...queries.shipment.getPreviousRates({
      customerId: shipment.customerId,
      originLocationId: origin?.id ?? "",
      destinationLocationId: destination?.id ?? "",
      shipmentTypeId: shipment.shipmentTypeId,
      serviceTypeId: shipment.serviceTypeId,
    }),
  });

  return (
    <Dialog {...props}>
      <DialogContent className="w-[1200px] max-w-[1800px] max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Previous Rates <BetaTag />
          </DialogTitle>
          <DialogDescription className="text-sm">
            There are {previousRates?.total} previous rates related to this
            shipment.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="max-h-[500px] overflow-y-auto p-0">
          {isLoading ? (
            <p>Loading...</p>
          ) : previousRates?.total === 0 ? (
            <p>No previous rates found</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Pro Number</TableHead>
                  <TableHead>Service Type</TableHead>
                  <TableHead>Shipment Type</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Origin</TableHead>
                  <TableHead>Destination</TableHead>
                  <TableHead>Total Charges</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {previousRates?.items.map((shipment) => (
                  <TableRow key={shipment.id}>
                    <TableCell>
                      <a
                        href={`/shipments/management?entityId=${shipment.id}&modalType=edit`}
                        target="_blank"
                        className="underline"
                        title="Click to view shipment details"
                        rel="noreferrer"
                      >
                        {shipment.proNumber}
                      </a>
                    </TableCell>
                    <TableCell>
                      <ShipmentCell
                        href={`/shipments/configurations/service-types?entityId=${shipment.serviceTypeId}&modalType=edit`}
                        value={shipment.serviceType?.code || "N/A"}
                        description={shipment.serviceType?.description ?? ""}
                      />
                    </TableCell>
                    <TableCell>
                      <ShipmentCell
                        href={`/shipments/configurations/shipment-types?entityId=${shipment.shipmentTypeId}&modalType=edit`}
                        value={shipment.shipmentType?.code || "N/A"}
                        description={shipment.shipmentType?.description ?? ""}
                      />
                    </TableCell>
                    <TableCell>
                      <ShipmentCell
                        href={`/billing/configurations/customers?entityId=${shipment.customerId}&modalType=edit`}
                        value={shipment.customer?.code || "N/A"}
                        description={shipment.customer?.name ?? ""}
                      />
                    </TableCell>
                    <TableCell>
                      <LocationCell location={origin as LocationSchema} />
                    </TableCell>
                    <TableCell>
                      <LocationCell location={destination as LocationSchema} />
                    </TableCell>
                    <TableCell>
                      {shipment.totalChargeAmount
                        ? USDollarFormat(shipment.totalChargeAmount)
                        : "N/A"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function ShipmentCell({
  href,
  value,
  description,
}: {
  href: string;
  value: any;
  description: string;
}) {
  return (
    <div className="flex flex-col">
      <a
        href={href}
        target="_blank"
        className="underline w-fit"
        rel="noreferrer"
      >
        {value}
      </a>
      <div className="text-sm text-muted-foreground">{description}</div>
    </div>
  );
}

function LocationCell({ location }: { location: LocationSchema }) {
  if (!location) {
    return <p className="text-muted-foreground">-</p>;
  }

  return (
    <div className="flex flex-col">
      <a
        href={`/dispatch/configurations/locations?entityId=${location.id}&modalType=edit`}
        target="_blank"
        className="underline w-fit"
        rel="noreferrer"
      >
        {location.name}
      </a>
      <div className="text-sm text-muted-foreground">
        {formatLocation(location)}
      </div>
    </div>
  );
}
