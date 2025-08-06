/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ShipmentStatusBadge } from "@/components/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useUser } from "@/stores/user-store";
import { useQuery } from "@tanstack/react-query";

export function ShipmentDetailsHeader({
  selectedShipment,
}: {
  selectedShipment?: ShipmentSchema | null;
}) {
  return (
    <ShipmentDetailsHeaderInner>
      <ShipmentDetailsHeaderTitle selectedShipment={selectedShipment} />
      <ShipmentDetailsHeaderDescription selectedShipment={selectedShipment} />
    </ShipmentDetailsHeaderInner>
  );
}

function ShipmentDetailsHeaderInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col px-4 pb-2">{children}</div>;
}

function ShipmentDetailsHeaderTitle({
  selectedShipment,
}: {
  selectedShipment?: ShipmentSchema | null;
}) {
  const { proNumber, status } = selectedShipment ?? {};

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center">
        <h2 className="font-semibold leading-none tracking-tight flex items-center gap-x-2">
          {proNumber || "Add New Shipment"}
        </h2>
      </div>
      <ShipmentStatusBadge status={status} />
    </div>
  );
}

function ShipmentDetailsHeaderDescription({
  selectedShipment,
}: {
  selectedShipment?: ShipmentSchema | null;
}) {
  const { updatedAt } = selectedShipment ?? {};
  const user = useUser();

  const { data: ownerInfo, isLoading: isLoadingOwnerInfo } = useQuery({
    ...queries.user.getUserById(selectedShipment?.ownerId ?? ""),
    enabled: !!selectedShipment?.ownerId,
  });

  return (
    <div className="flex justify-between items-center">
      {updatedAt ? (
        <p className="text-2xs text-muted-foreground font-normal">
          Last updated on{" "}
          {formatToUserTimezone(updatedAt, {
            timeFormat: user?.timeFormat,
          })}
        </p>
      ) : (
        <p className="text-2xs text-muted-foreground font-normal">
          Please fill out the form below to create a new shipment.
        </p>
      )}
      {isLoadingOwnerInfo ? (
        <Skeleton className="w-34 h-2.5" />
      ) : selectedShipment?.ownerId ? (
        <div className="flex items-center gap-x-1">
          <p className="text-2xs text-muted-foreground font-normal">Owner:</p>
          <p className="text-2xs text-blue-500 font-normal">
            {ownerInfo?.name}
          </p>
        </div>
      ) : (
        <p className="text-2xs text-foreground font-normal">
          No owner assigned
        </p>
      )}
    </div>
  );
}
