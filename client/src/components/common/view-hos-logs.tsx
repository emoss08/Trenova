/*
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

import { useShipmentStore } from "@/stores/ShipmentStore";
import { TableSheetProps } from "@/types/tables";
import { DialogTitle } from "@radix-ui/react-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
} from "../ui/dialog";

const hours = [
  "Midnight",
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7",
  "8",
  "9",
  "10",
  "11",
  "Noon",
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7",
  "8",
  "9",
  "10",
  "11",
  "Midnight",
];
const statusTypes = ["OFF", "SB", "D", "ON", "V"];
const serviceHours = [
  { status: "ON", start: 8, end: 12 },
  { status: "OFF", start: 12, end: 13 },
  // Add more status blocks as needed
];

const VerticalLines = ({ serviceHours }: { serviceHours: any[] }) => {
  const getGridColumn = (hour: number) => {
    if (hour === 24) {
      return 25;
    }
    return hour + 1;
  };

  return (
    <>
      {serviceHours.map((serviceHour, index) => {
        const nextServiceHour = serviceHours[index + 1];
        if (nextServiceHour && nextServiceHour.start === serviceHour.end) {
          const gridColumnEnd = getGridColumn(serviceHour.end);
          return (
            <div
              key={index}
              className="absolute border-l-2 border-foreground"
              style={{
                left: `${(gridColumnEnd - 1) * 4.1667}%`,
                height: "28%",
                bottom: 50,
              }}
            ></div>
          );
        }
        return null;
      })}
    </>
  );
};

export function HourGrid() {
  // A helper function to calculate the grid column based on the hour
  const getGridColumn = (hour: number) => {
    // Adjust the hour to fit into the 24-hour grid system where Midnight is both 0 and 24
    if (hour === 24) {
      return 25; // The 25th cell, which is the last "Midnight"
    }
    return hour + 1; // Add 1 because CSS grid columns start at 1, not 0
  };

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="flex justify-between border-b-2">
        {hours.map((hour, index) => (
          <div key={index} className="p-1 text-xs">
            {hour}
          </div>
        ))}
      </div>

      {/* Status Rows */}
      {statusTypes.map((statusType, rowIndex) => (
        <div key={rowIndex} className="relative flex">
          {" "}
          {/* Add relative positioning here */}
          <div className="w-12 border-r-2 p-1 text-xs">{statusType}</div>
          {hours.map((_, index) => (
            <div key={index} className="flex-1 border-b border-r">
              &nbsp;
            </div>
          ))}
          {/* Service Hours Lines */}
          {serviceHours.map((serviceHour, index) => {
            if (serviceHour.status === statusType) {
              const gridColumnStart = getGridColumn(serviceHour.start);
              const gridColumnEnd = getGridColumn(serviceHour.end);
              return (
                <div
                  key={index}
                  className="absolute h-full"
                  style={{
                    left: `${(gridColumnStart - 1) * 4.1667}%`, // Convert grid column to percentage
                    width: `${(gridColumnEnd - gridColumnStart) * 4.1667}%`, // Convert grid column difference to percentage
                  }}
                >
                  <div className="h-full border-b-2 border-foreground" />
                </div>
              );
            }
            return null;
          })}
        </div>
      ))}
    </div>
  );
}

export function HourGridDialog({ onOpenChange, open }: TableSheetProps) {
  const [currentWorker] = useShipmentStore.use("currentWorker");

  const fullName = `${currentWorker?.firstName} ${currentWorker?.lastName}`;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[700px]">
        <DialogHeader>
          <DialogTitle>HOS logs for {fullName}</DialogTitle>
          <DialogDescription>
            View logs for {fullName} for the past 24 hours.
          </DialogDescription>
        </DialogHeader>
        <div className="flex h-full flex-col overflow-y-auto">
          <HourGrid />
          <VerticalLines serviceHours={serviceHours} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
