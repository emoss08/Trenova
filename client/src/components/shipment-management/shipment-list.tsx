/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { shipmentStatusToReadable } from "@/lib/utils";
import { Shipment } from "@/types/order";
import { ArrowDown, ArrowUp } from "lucide-react";
import React from "react";
import { Badge } from "../ui/badge";

const ShipmentProgressIndicator = ({
  currentStatus,
  finalStatuses,
  progressStatuses,
}: {
  currentStatus: string;
  finalStatuses: string[];
  progressStatuses: string[];
}) => {
  // Check if the current status is a final status
  const isFinalStatus = finalStatuses.includes(currentStatus);

  // Determine the index for the progress bar; if it's a final status, set it to complete
  const currentStatusIndex = isFinalStatus
    ? progressStatuses.length - 1
    : progressStatuses.indexOf(currentStatus);

  return (
    <div className="flex items-center w-full">
      {progressStatuses.map((status, index) => (
        <React.Fragment key={status}>
          <div
            className={`flex-1 h-1 ${
              index <= currentStatusIndex
                ? "bg-foreground"
                : "bg-muted-foreground/40 animate-pulse"
            }`}
          />
          {/* Render a spacer after each line except the last one */}
          {index < progressStatuses.length - 1 && <div className="w-1" />}
        </React.Fragment>
      ))}
    </div>
  );
};

export function ShipmentList({
  shipments,
  finalStatuses,
  progressStatuses,
}: {
  shipments: Shipment[];
  finalStatuses: string[];
  progressStatuses: string[];
}) {
  const formatDate = (dateString: string) =>
    new Date(dateString).toLocaleString();

  // Function to check if the shipment is delayed
  const isShipmentDelayed = (item: any, finalStatuses: any) => {
    const deliveryEndDate = new Date(item.destinationAppointmentWindowEnd);
    const today = new Date();
    return !finalStatuses.includes(item.status) && today > deliveryEndDate;
  };

  return (
    <ul role="list" className="space-y-5">
      {shipments &&
        shipments.map((shipment) => (
          <li
            key={shipment.id}
            className="group overflow-hidden bg-background hover:bg-muted/50 hover:cursor-pointer ring-1 ring-accent-foreground/20 rounded-md p-4 sm:px-6 relative"
          >
            {/* Check and render the badge if the shipment is delayed */}
            {isShipmentDelayed(shipment, finalStatuses) && (
              <Badge className="absolute top-0 right-0 p-1 rounded-none rounded-bl text-xs">
                Delayed
              </Badge>
            )}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-center">
              {/* Shipment status, pro number, and progress indicator */}
              <div className="md:col-span-1 flex flex-col">
                <p className="text-xs font-semibold text-muted-foregrounds">
                  #{shipment.proNumber}
                </p>
                <h4 className="text-xl font-semibold text-foreground">
                  {shipmentStatusToReadable(shipment.status)}
                </h4>
                <p className="text-sm text-muted-foreground">
                  {formatDate(shipment.created)}
                </p>
                {/* Shipment progress indicator directly below the status */}
                <div className="w-full mt-2">
                  <ShipmentProgressIndicator
                    currentStatus={shipment.status}
                    finalStatuses={finalStatuses}
                    progressStatuses={progressStatuses}
                  />
                </div>
              </div>
              {/* Shipment origin and destination with appointment */}
              <div className="md:col-span-2 grid grid-cols-2 gap-4 ml-4">
                {/* Shipment origin and appointment */}
                <div className="text-sm">
                  <div className="flex items-center mb-2">
                    <div className="flex items-center justify-center rounded-full w-4 h-4 bg-foreground mr-2">
                      <ArrowUp className="inline-block h-3 w-3 text-background" />
                    </div>
                    <span className="font-semibold text-foreground">
                      {shipment.originAddress}
                    </span>
                  </div>
                  <div className="text-foreground">
                    <div className="flex">
                      <p className="font-semibold pr-2">Window Start: </p>{" "}
                      {formatDate(shipment.originAppointmentWindowStart)}
                    </div>
                    <div className="flex">
                      <p className="font-semibold pr-2">Window End: </p>{" "}
                      {formatDate(shipment.originAppointmentWindowEnd)}
                    </div>
                  </div>
                </div>
                {/* Shipment destination and appointment */}
                <div className="text-sm">
                  <div className="flex items-center mb-2">
                    <div className="flex items-center justify-center rounded-full w-4 h-4 bg-blue-500 mr-2">
                      <ArrowDown className="inline-block h-3 w-3 text-white" />
                    </div>
                    <span>
                      <span className="font-semibold text-foreground">
                        {shipment.destinationAddress}
                      </span>
                    </span>
                  </div>
                  <div className="text-foreground">
                    <div className="flex">
                      <p className="font-semibold pr-2">Window Start: </p>{" "}
                      {formatDate(shipment.destinationAppointmentWindowStart)}
                    </div>
                    <div className="flex">
                      <p className="font-semibold pr-2">Window End: </p>{" "}
                      {formatDate(shipment.destinationAppointmentWindowEnd)}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </li>
        ))}
    </ul>
  );
}