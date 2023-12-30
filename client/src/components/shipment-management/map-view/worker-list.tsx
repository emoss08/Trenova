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

import { Avatar, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn, upperFirst } from "@/lib/utils";
import { useShipmentStore } from "@/stores/ShipmentStore";
import { DispatchControl } from "@/types/dispatch";
import { Worker } from "@/types/worker";
import { AvatarFallback } from "@radix-ui/react-avatar";

const currentStatusColor = (status: string) => {
  switch (status) {
    case "driving":
      return "text-green-700";
    case "off-duty":
      return "text-red-700";
    case "on-duty":
      return "text-violet-700";
    case "sleeper-berth":
      return "text-blue-700";
    default:
      return "text-foreground";
  }
};

const currentStatusColorBg = (status: string) => {
  switch (status) {
    case "driving":
      return "bg-green-700";
    case "off-duty":
      return "bg-red-700";
    case "on-duty":
      return "bg-violet-700";
    case "sleeper-berth":
      return "bg-blue-700";
    default:
      return "bg-foreground";
  }
};

const convertSecondsToHours = (seconds?: number) => {
  if (!seconds) return;

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  const paddedHours = hours.toString().padStart(2, "0");
  const paddedMinutes = minutes.toString().padStart(2, "0");

  return `${paddedHours}:${paddedMinutes}`;
};

function isRegInformationExpired(worker: Worker, enforceRegCheck: boolean) {
  if (!enforceRegCheck) return { isExpired: false, expiredItemsDetails: [] };

  const currentDate = new Date().toISOString().split("T")[0]; // Format as YYYY-MM-DD

  const isDateExpired = (dateString: string | undefined) => {
    return dateString && dateString < currentDate;
  };

  const formatExpiredItem = (item: string, date: string | undefined) =>
    date && `${item}: ${date}`;

  const expiredItemsDetails = [
    isDateExpired(worker.profile.hazmatExpirationDate?.toString()) &&
      formatExpiredItem(
        "Hazmat",
        worker.profile.hazmatExpirationDate?.toString(),
      ),
    isDateExpired(worker.profile.medicalCertDate?.toString()) &&
      formatExpiredItem(
        "Medical Certification",
        worker.profile.medicalCertDate?.toString(),
      ),
    isDateExpired(worker.profile.licenseExpirationDate?.toString()) &&
      formatExpiredItem(
        "License",
        worker.profile.licenseExpirationDate?.toString(),
      ),
  ].filter(Boolean); // Remove falsy values

  return {
    isExpired: expiredItemsDetails.length > 0,
    expiredItemsDetails,
  };
}

function WorkerRegBadge({
  worker,
  enforceRegCheck,
}: {
  worker: Worker;
  enforceRegCheck: boolean;
}) {
  const { isExpired, expiredItemsDetails } = isRegInformationExpired(
    worker,
    enforceRegCheck,
  );

  return (
    <TooltipProvider>
      {isExpired && (
        <Tooltip>
          <TooltipTrigger>
            <Badge className="absolute top-0 right-0 p-1 rounded-none rounded-bl rounded-tr text-xs h-5 w-32 bg-destructive text-destructive-foreground hover:bg-destructive/50 hover:text-background">
              Attention Required
            </Badge>
          </TooltipTrigger>
          <TooltipContent side="top" sideOffset={40} align="center">
            <p className="font-semibold">
              The following regulatory information for this worker has expired:
            </p>
            <ul className="list-disc ml-4">
              {expiredItemsDetails.map((detail, index) => (
                <li key={index} className="text-sm">
                  {detail}
                </li>
              ))}
            </ul>
            <p className="text-muted-foreground mt-2 border-t font-semibold">
              You are seeing this because your organization enforces regulatory
              checks.
            </p>
          </TooltipContent>
        </Tooltip>
      )}
    </TooltipProvider>
  );
}

function WorkerCard({
  worker,
  enforceRegCheck,
}: {
  worker: Worker;
  enforceRegCheck: boolean;
}) {
  const workerFullName = `${worker.firstName} ${worker.lastName}`;
  const currentStatus = worker.currentHos?.currentStatus || "";
  const statusColor = currentStatusColor(currentStatus);
  const statusColorBg = currentStatusColorBg(currentStatus);
  const { isExpired } = isRegInformationExpired(worker, enforceRegCheck);

  return (
    <li
      className={cn(
        "group relative flex items-center space-x-3 rounded-lg border px-4 py-3 shadow-sm hover:bg-foreground mb-2",
        `ring-accent-foreground/20 ${statusColor}`,
      )}
    >
      <div className="flex-shrink-0">
        <Avatar className="flex items-center justify-center">
          <AvatarImage
            src={worker?.thumbnail}
            alt={worker.code}
            className="h-10 w-10 rounded-full"
          />
          <AvatarFallback
            className={cn(
              "h-10 w-10 rounded-full flex items-center justify-center text-background font-bold",
              `${statusColorBg}`,
            )}
          >
            {worker.firstName[0]}
          </AvatarFallback>
        </Avatar>
      </div>
      <div className="min-w-0 flex-1">
        <a href="#" className="focus:outline-none">
          <span
            className={cn(
              "absolute inset-0",
              isExpired && "cursor-not-allowed",
            )}
            aria-hidden="true"
          />
          <p className="text-sm font-bold text-foreground group-hover:text-background">
            {workerFullName}
          </p>
          <p className="text-xs group-hover:text-background truncate">
            Current Status:{" "}
            {upperFirst(worker.currentHos?.currentStatus || "-")}
          </p>
          <div className="flex">
            <p className="text-xs text-muted-foreground group-hover:text-background truncate">
              On Duty Clock:{" "}
              {convertSecondsToHours(worker.currentHos?.onDutyTime) || "-"}
            </p>
            <p className="text-xs text-muted-foreground group-hover:text-background truncate ml-3">
              Drive Time:{" "}
              {convertSecondsToHours(worker.currentHos?.driveTime) || "-"}
            </p>
          </div>
        </a>
      </div>
      {/* Add the WorkerRegBadge component */}
      <WorkerRegBadge worker={worker} enforceRegCheck={enforceRegCheck} />
    </li>
  );
}

export function WorkerList({
  workersData,
  dispatchControlData,
}: {
  workersData: Worker[];
  dispatchControlData: DispatchControl;
}) {
  return workersData?.length ? (
    <>
      {/* Scrollable list of workers */}
      <ScrollArea className="mt-2">
        <ul className="p-3 h-[600px]">
          {workersData?.map((item) => (
            <ContextMenu key={item.id}>
              <ContextMenuTrigger>
                <WorkerCard
                  worker={item}
                  enforceRegCheck={
                    dispatchControlData?.regulatoryCheck || false
                  }
                />
              </ContextMenuTrigger>
              <ContextMenuContent className="w-64">
                <ContextMenuLabel>
                  Viewing: {item.firstName} {item.lastName}
                </ContextMenuLabel>
                <ContextMenuSeparator />
                <ContextMenuItem>Edit Worker Information</ContextMenuItem>
                <ContextMenuItem>View Schedule</ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem>Assign New Shipment</ContextMenuItem>
                <ContextMenuItem>Schedule Maintenance</ContextMenuItem>
                <ContextMenuItem>Monitor Performance</ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem
                  onClick={() => {
                    // Set the current worker to currentWorker in the store.
                    useShipmentStore.set("currentWorker", item);

                    // Open the Review Logs Dialog.
                    useShipmentStore.set("reviewLogDialogOpen", true);
                  }}
                >
                  Review HOS Logs
                </ContextMenuItem>
                <ContextMenuItem>View Incident Reports</ContextMenuItem>
                <ContextMenuItem>Vehicle Maintenance Logs</ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem
                  onClick={() => {
                    // Set the current worker to currentWorker in the store.
                    useShipmentStore.set("currentWorker", item);

                    // Open the send message dialog.
                    useShipmentStore.set("sendMessageDialogOpen", true);
                  }}
                >
                  Send Message
                </ContextMenuItem>
                <ContextMenuItem>Send Notifications</ContextMenuItem>
              </ContextMenuContent>
            </ContextMenu>
          ))}
        </ul>
      </ScrollArea>
    </>
  ) : (
    <div className="flex flex-col items-center justify-center mt-52">
      <p className="text-foreground text-lg font-bold">No Workers Found</p>
      <p className="text-muted-foreground text-sm">
        Try adjusting your search query or filters.
      </p>
    </div>
  );
}
