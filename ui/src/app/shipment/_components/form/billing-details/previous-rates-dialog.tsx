/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { BetaTag } from "@/components/ui/beta-tag";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import LetterGlitch from "@/components/ui/letter-glitch";
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
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useFormContext } from "react-hook-form";

export function PreviousRatesDialog() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const [open, setOpen] = useState(false);
  const shipment = getValues();
  const { origin, destination } = ShipmentLocations.useLocations(shipment);

  const canViewPreviousRates =
    !!shipment.serviceTypeId &&
    !!shipment.shipmentTypeId &&
    !!origin &&
    !!destination &&
    !!shipment.customerId;

  const { data: previousRates, isLoading } = useQuery({
    ...queries.shipment.getPreviousRates({
      customerId: shipment.customerId,
      originLocationId: origin?.id ?? "",
      destinationLocationId: destination?.id ?? "",
      shipmentTypeId: shipment.shipmentTypeId,
      serviceTypeId: shipment.serviceTypeId,
      excludeShipmentId: shipment.id,
    }),
    enabled: canViewPreviousRates,
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setOpen(true);
          }}
          disabled={!canViewPreviousRates}
          size="xs"
        >
          <span>View Previous Rates</span>
        </Button>
      </DialogTrigger>
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
            <div className="flex flex-col items-center justify-center max-h-[300px] rounded-md overflow-hidden p-2">
              <div className="relative size-full max-h-[300px]">
                <LetterGlitch
                  glitchColors={["#9c9c9c", "#696969", "#424242"]}
                  glitchSpeed={50}
                  centerVignette={true}
                  outerVignette={true}
                  smooth={true}
                  className="max-h-[300px]"
                  canvasClassName="max-h-[300px]"
                />
                <div className="absolute inset-0 flex flex-col gap-1 items-center justify-center pointer-events-none">
                  <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-amber-300 text-amber-950 dark:bg-amber-400">
                    No data available
                  </p>
                  <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-neutral-900 text-white dark:bg-neutral-500">
                    No previous rates associated with this lane
                  </p>
                </div>
              </div>
            </div>
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
