import { DataTableV2 } from "@/components/data-table/data-table-v2";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LiveModePresets } from "@/lib/live-mode-utils";
import { queries } from "@/lib/queries";
import { getShipmentStatusRowClassName } from "@/lib/table-styles";
import { cn } from "@/lib/utils";
import { AnalyticsPage } from "@/types/analytics";
import { Resource } from "@/types/audit-entry";
import { Shipment, ShipmentStatus } from "@/types/shipment";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { getColumns } from "./shipment-columns";
import { ShipmentCreateSheet } from "./shipment-create-sheet";
import { ShipmentEditSheet } from "./shipment-edit-sheet";

export default function ShipmentTable() {
  const columns = useMemo(() => getColumns(), []);
  const [status, setStatus] = useState<ShipmentStatus | undefined>(undefined);

  return (
    <>
      <ShipmentTabs status={status} setStatus={setStatus} />
      <DataTableV2<Shipment>
        name="Shipment"
        link="/shipments/"
        extraSearchParams={{
          expandShipmentDetails: true,
          ...(status && { status: status.toString() }),
        }}
        queryKey="shipment-list"
        exportModelName="shipment"
        resource={Resource.Shipment}
        TableModal={ShipmentCreateSheet}
        TableEditModal={ShipmentEditSheet}
        columns={columns}
        getRowClassName={(row) => {
          return cn(getShipmentStatusRowClassName(row.original.status));
        }}
        liveMode={LiveModePresets.shipments()}
        // extraActions={[
        //   {
        //     key: "import-from-rate",
        //     label: "Import from Rate Conf.",
        //     description: "Import shipment from rate confirmation",
        //     icon: faFileImport,
        //     onClick: () => {
        //       console.log("Import from Rate Conf.");
        //     },
        //     endContent: <BetaTag label="Preview" />,
        //   },
        // ]}
      />
    </>
  );
}

function ShipmentTabs({
  status,
  setStatus,
}: {
  status: ShipmentStatus | undefined;
  setStatus: (status: ShipmentStatus | undefined) => void;
}) {
  const { data: analytics } = useSuspenseQuery({
    ...queries.analytics.getAnalytics(AnalyticsPage.ShipmentManagement),
  });

  interface StatusCount {
    status: string;
    count: number;
  }

  const statusCounts = (analytics?.countByShipmentStatus ||
    []) as StatusCount[];
  const totalShipments = analytics?.shipmentCountCard?.count || 0;

  // Sort status counts by count in descending order
  const sortedStatusCounts = [...statusCounts].sort(
    (a, b) => b.count - a.count,
  );

  // For each status in the countByShipmentStatus array, create a tab, however, make sure there is a tab for all shipments and if the count is 0, disable the tab
  const handleTabChange = (value: string) => {
    if (value === "all") {
      setStatus(undefined); // All shipments
    } else {
      // Convert the value (kebab-case) to enum value
      const enumValue = value
        .split("-")
        .map((word, index) =>
          index === 0
            ? word.charAt(0).toUpperCase() + word.slice(1)
            : word.charAt(0).toUpperCase() + word.slice(1),
        )
        .join("");

      setStatus(ShipmentStatus[enumValue as keyof typeof ShipmentStatus]);
    }
  };

  // Determine the active tab based on current status
  const getActiveTab = () => {
    if (!status) return "all";

    // Convert enum value to kebab-case
    return status
      .toString()
      .replace(/([a-z])([A-Z])/g, "$1-$2")
      .toLowerCase();
  };

  // Helper to convert enum values to readable text
  const formatStatusText = (statusValue: string): string => {
    return statusValue.replace(/([A-Z])/g, " $1").trim();
  };

  // Helper to convert enum values to kebab-case values
  const getValueFromStatus = (statusValue: string): string => {
    return statusValue.replace(/([a-z])([A-Z])/g, "$1-$2").toLowerCase();
  };

  return (
    <Tabs
      defaultValue={getActiveTab()}
      className="items-center"
      onValueChange={handleTabChange}
    >
      <TabsList className="h-auto rounded-none border-b gap-4 bg-transparent p-0 w-full justify-start">
        <TabsTrigger
          value="all"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          All Shipments{" "}
          {totalShipments > 0 && (
            <div className="text-xs text-muted-foreground bg-muted rounded-md px-1.5 items-center justify-center py-0.5 size-full group-data-[state=active]:bg-primary group-data-[state=active]:text-background">
              {totalShipments}
            </div>
          )}
        </TabsTrigger>

        {sortedStatusCounts.map((statusItem) => {
          const value = getValueFromStatus(statusItem.status);
          const count = statusItem.count;
          const isDisabled = count === 0;

          return (
            <TabsTrigger
              key={value}
              value={value}
              disabled={isDisabled}
              className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
            >
              {formatStatusText(statusItem.status)}{" "}
              {count > 0 && (
                <div className="text-xs text-muted-foreground bg-muted rounded-md px-1.5 items-center justify-center py-0.5 size-full group-data-[state=active]:bg-primary group-data-[state=active]:text-background">
                  {count}
                </div>
              )}
            </TabsTrigger>
          );
        })}
      </TabsList>
    </Tabs>
  );
}
