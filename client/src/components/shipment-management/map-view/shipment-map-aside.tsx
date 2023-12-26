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
import { InputField } from "@/components/common/fields/input";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useForm } from "react-hook-form";
import { WorkerSortOptions } from "@/components/shipment-management/map-view/shipment-map-filter";
import { useFleetCodes, useUsers } from "@/hooks/useQueries";

const workersItems = [
  {
    id: 1,
    fullName: "John Doe",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Driving",
  },
  {
    id: 2,
    fullName: "Jane Doe",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Off-Duty",
  },
  {
    id: 3,
    fullName: "John Smith",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "On-Duty (Not Driving)",
  },
  {
    id: 4,
    fullName: "Jane Smith",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Driving",
  },
  {
    id: 5,
    fullName: "John Doe",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Sleeper Berth",
  },
  {
    id: 6,
    fullName: "Jane Doe",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Off-Duty",
  },
  {
    id: 7,
    fullName: "John Smith",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Driving",
  },
  {
    id: 8,
    fullName: "Jane Smith",
    onDutyClock: "14:00",
    driveTime: "11:00",
    seventyHourClock: "70:00",
    currentStatus: "Driving",
  },
];

const randomWorkerBackground = () => {
  const colors = [
    "bg-red-500",
    "bg-green-500",
    "bg-blue-500",
    "bg-yellow-500",
    "bg-indigo-500",
    "bg-pink-500",
    "bg-purple-500",
  ];
  return colors[Math.floor(Math.random() * colors.length)];
};

const currentStatusColor = (status: string) => {
  switch (status) {
    case "Driving":
      return "text-red-500";
    case "Off-Duty":
      return "text-green-500";
    case "On-Duty (Not Driving)":
      return "text-blue-500";
    case "Sleeper Berth":
      return "text-yellow-500";
    default:
      return "text-muted-foreground";
  }
};

export function ShipmentMapAside() {
  const { control } = useForm();
  const { selectFleetCodes, isLoading: isFleetCodesLoading } = useFleetCodes();
  const { selectUsersData, isLoading: isUsersLoading } = useUsers();

  const sortOptions = [
    {
      id: "status", // TODO: Change this once the HOS integration is an option.
      title: "Status",
      options: [
        {
          value: true,
          label: "Active",
        },
        {
          value: false,
          label: "Inactive",
        },
      ],
    },

    {
      id: "fleetCode",
      title: "Fleet",
      options: selectFleetCodes,
      loading: isFleetCodesLoading,
    },
    {
      id: "manager",
      title: "Manager",
      options: selectUsersData,
      loading: isUsersLoading,
    },

    {
      id: "endorsements",
      title: "Endorsements",
      options: [
        {
          value: "H",
          label: "Hazmat",
        },
        {
          value: "T",
          label: "Tanker",
        },
        {
          value: "X",
          label: "Tanker & Hazmat",
        },
        {
          value: "P",
          label: "Doubles",
        },
      ],
    },
  ];

  return (
    <aside className="w-96 p-4 border ring-accent-foreground/20 rounded-md">
      {/* Fixed search field at the top */}
      <InputField
        name="searchQuery"
        control={control}
        placeholder="Search Workers..."
        icon={<MagnifyingGlassIcon className="h-4 w-4 text-muted-foreground" />}
      />

      {/* Worker Sort Options */}
      <WorkerSortOptions sortOptions={sortOptions} />

      {/* Scrollable list of workers */}
      <ScrollArea className="mt-2">
        <ul className="space-y-3 p-3 h-[610px]">
          {workersItems.map((item) => (
            <li
              key={item.id}
              className="group relative flex items-center space-x-3 rounded-lg border ring-accent-foreground/20 bg-background px-6 py-3 shadow-sm hover:bg-foreground"
            >
              <div className="flex-shrink-0">
                <div
                  className={`h-10 w-10 rounded-full ${randomWorkerBackground()} flex items-center justify-center text-white font-bold`}
                >
                  {item.fullName[0]}
                </div>
              </div>
              <div className="min-w-0 flex-1">
                <a href="#" className="focus:outline-none">
                  <span className="absolute inset-0" aria-hidden="true" />
                  <p className="text-sm font-bold text-foreground group-hover:text-background">
                    {item.fullName}
                  </p>
                  <p
                    className={`text-xs ${currentStatusColor(
                      item.currentStatus,
                    )} group-hover:text-background truncate`}
                  >
                    Current Status: {item.currentStatus}
                  </p>
                  <div className="flex">
                    <p className="text-xs text-muted-foreground group-hover:text-background truncate">
                      On Duty Clock: {item.onDutyClock}
                    </p>
                    <p className="text-xs text-muted-foreground group-hover:text-background truncate ml-3">
                      Drive Time: {item.driveTime}
                    </p>
                  </div>
                </a>
              </div>
            </li>
          ))}
        </ul>
      </ScrollArea>
    </aside>
  );
}
