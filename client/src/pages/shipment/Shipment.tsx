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

import { ShipmentAsideMenus } from "@/components/shipment-management/shipment-aside-menu";
import React from "react";
import { useQuery } from "@tanstack/react-query";
import { QueryKeys } from "@/types";
import { getShipments } from "@/services/ShipmentRequestService";

const statusToString = (status: string) => {
  switch (status) {
    case "N":
      return "New";
    case "P":
      return "In Progress";
    case "C":
      return "Completed";
    case "H":
      return "On Hold";
    case "B":
      return "BILLED";
    case "V":
      return "Voided";
    default:
      return "Unknown";
  }
};

const ShipmentProgressIndicator = ({
  currentStatus,
}: {
  currentStatus: string;
}) => {
  // Define the order of the progress statuses
  const progressStatuses = ["N", "P", "C"];
  // Define the final statuses that indicate completion
  const finalStatuses = ["C", "H", "B", "V"];
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
                ? "bg-background"
                : "bg-muted-foreground"
            }`}
          />
          {/* Render a spacer after each line except the last one */}
          {index < progressStatuses.length - 1 && <div className="w-1" />}
        </React.Fragment>
      ))}
    </div>
  );
};

const Shipments = () => {
  const { data, isLoading, isFetched } = useQuery({
    queryKey: ["shipments"] as QueryKeys[],
    queryFn: async () => getShipments(),
  });

  const formatDate = (dateString: string) =>
    new Date(dateString).toLocaleString();

  return (
    <ul role="list" className="space-y-5">
      {data &&
        data.map((item) => (
          <li
            key={item.id}
            className="overflow-hidden bg-foreground shadow rounded-lg px-4 py-4 sm:px-6"
          >
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-center">
              {/* Shipment status, pro number, and progress indicator */}
              <div className="md:col-span-1 flex flex-col">
                <p className="text-xs font-semibold text-muted-foreground">
                  {item.proNumber}
                </p>
                <h4 className="text-xl font-semibold text-background">
                  {statusToString(item.status)}
                </h4>
                {/* Shipment progress indicator directly below the status */}
                <div className="w-full mt-2">
                  <ShipmentProgressIndicator currentStatus={item.status} />
                </div>
              </div>
              {/* Shipment origin and destination with appointment */}
              <div className="md:col-span-2 grid grid-cols-2 gap-4">
                {/* Shipment origin and appointment */}
                <div className="text-sm text-gray-500">
                  <div>
                    Origin:{" "}
                    <span className="font-semibold">{item.originAddress}</span>
                  </div>
                  <div>
                    Appointment: {formatDate(item.originAppointmentWindowStart)}{" "}
                    - {formatDate(item.originAppointmentWindowEnd)}
                  </div>
                </div>
                {/* Shipment destination and appointment */}
                <div className="text-sm text-gray-500">
                  <div>
                    Destination:{" "}
                    <span className="font-semibold">
                      {item.destinationAddress}
                    </span>
                  </div>
                  <div>
                    Appointment:{" "}
                    {formatDate(item.destinationAppointmentWindowStart)} -{" "}
                    {formatDate(item.destinationAppointmentWindowEnd)}
                  </div>
                </div>
              </div>
            </div>
          </li>
        ))}
    </ul>
  );
};

export default function ShipmentManagement() {
  return (
    <div className="flex space-x-10 p-4">
      <div className="w-1/4">
        <ShipmentAsideMenus />
      </div>
      <div className="w-3/4">
        <Shipments />
      </div>
    </div>
  );
}
