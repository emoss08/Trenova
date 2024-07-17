/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { formatToUserTimezone } from "@/lib/date";
import { cn, shipmentStatusToReadable } from "@/lib/utils";
import { Shipment, ShipmentStatus, Stop } from "@/types/shipment";
import { VariantProps } from "class-variance-authority";
import React from "react";
import { Badge, badgeVariants } from "../ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";

const statusColors: Record<
  ShipmentStatus,
  VariantProps<typeof badgeVariants>["variant"]
> = {
  New: "info",
  InProgress: "purple",
  Completed: "active",
  Hold: "warning",
  Billed: "pink",
  Voided: "inactive",
};

const statusColorClasses: Record<ShipmentStatus, string> = {
  New: "bg-blue-600",
  InProgress: "bg-purple-600",
  Completed: "bg-green-600",
  Hold: "bg-yellow-600",
  Billed: "bg-pink-600",
  Voided: "bg-red-600",
};

const ShipmentProgressIndicator: React.FC<{
  currentStatus: ShipmentStatus;
  finalStatuses: ShipmentStatus[];
}> = ({ currentStatus, finalStatuses }) => {
  const isFinalStatus = finalStatuses.includes(currentStatus);
  const isVoided = currentStatus === "Voided";

  const displayStatuses: ShipmentStatus[] = [
    "New",
    "InProgress",
    "Completed",
    "Billed",
  ];

  let currentStatusIndex = displayStatuses.indexOf(currentStatus);
  if (currentStatusIndex === -1) {
    currentStatusIndex = isFinalStatus
      ? 2
      : displayStatuses.indexOf("InProgress");
  }

  return (
    <div className="flex w-full items-center">
      {displayStatuses.map((status, index) => (
        <React.Fragment key={status}>
          <div
            className={cn(
              "h-1 flex-1",
              isVoided
                ? "bg-red-500"
                : index <= currentStatusIndex
                ? statusColorClasses[currentStatus]
                : "bg-muted-foreground/40",
            )}
          />
          {index < displayStatuses.length - 1 && <div className="w-1" />}
        </React.Fragment>
      ))}
    </div>
  );
};

export function ShipmentInfo({
  shipment,
  finalStatuses,
}: {
  shipment: Shipment;
  finalStatuses: ShipmentStatus[];
  progressStatuses: ShipmentStatus[];
}) {
  const isDelayed = (): boolean => {
    if (!shipment.estimatedDeliveryDate) return false;
    const deliveryEndDate = new Date(shipment.estimatedDeliveryDate);
    const today = new Date();
    return !finalStatuses.includes(shipment.status) && today > deliveryEndDate;
  };

  const getFirstPickup = (): Stop | undefined => {
    for (const move of shipment.moves) {
      const pickup = move.stops.find((stop) => stop.type === "Pickup");
      if (pickup) return pickup;
    }
    return undefined;
  };

  const getLastDelivery = (): Stop | undefined => {
    for (let i = shipment.moves.length - 1; i >= 0; i--) {
      const move = shipment.moves[i];
      for (let j = move.stops.length - 1; j >= 0; j--) {
        if (move.stops[j].type === "Delivery") return move.stops[j];
      }
    }
    return undefined;
  };

  const firstPickup = getFirstPickup();
  const lastDelivery = getLastDelivery();

  return (
    <Card className="mb-4 w-full select-none border border-dashed hover:cursor-pointer hover:bg-muted/30">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">
          Shipment #{shipment.proNumber}
          <p className="text-xs text-muted-foreground">
            Created: {formatToUserTimezone(shipment.createdAt)}
          </p>
        </CardTitle>
        <Badge variant={statusColors[shipment.status]}>
          {shipmentStatusToReadable(shipment.status)}
        </Badge>
      </CardHeader>
      <CardContent className="px-6 py-0">
        <div className="mt-2 space-y-1">
          <ShipmentProgressIndicator
            currentStatus={shipment.status}
            finalStatuses={finalStatuses}
          />
        </div>
        <div className="mt-4 grid grid-cols-2 gap-4">
          <div>
            <h4 className="mb-1 text-sm font-semibold">Origin</h4>
            <p className="text-xs">{firstPickup?.addressLine || "N/A"}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {firstPickup?.plannedArrival
                ? formatToUserTimezone(firstPickup.plannedArrival)
                : "N/A"}
            </p>
          </div>
          <div>
            <h4 className="mb-1 text-sm font-semibold">Destination</h4>
            <p className="text-xs">{lastDelivery?.addressLine || "N/A"}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {lastDelivery?.plannedDeparture
                ? formatToUserTimezone(lastDelivery.plannedDeparture)
                : "N/A"}
            </p>
          </div>
        </div>
        <div className="mt-4 flex items-center justify-between">
          {isDelayed() && <Badge variant="inactive">Delayed</Badge>}
        </div>
      </CardContent>
    </Card>
  );
}
