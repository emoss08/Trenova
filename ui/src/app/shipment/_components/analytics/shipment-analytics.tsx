/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { PlainShipmentStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { formatToUserTimezone } from "@/lib/date";
import { convertToCsv, downloadFile, generateFilename } from "@/lib/file-utils";
import { queries } from "@/lib/queries";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { cn } from "@/lib/utils";
import { useUser } from "@/stores/user-store";
import { AnalyticsPage } from "@/types/analytics";
import { faEllipsis } from "@fortawesome/pro-regular-svg-icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { Link } from "react-router";

export default function ShipmentAnalytics() {
  const { data: analytics } = useSuspenseQuery({
    ...queries.analytics.getAnalytics(AnalyticsPage.ShipmentManagement),
  });

  return (
    <dl className="grid gap-6 md:grid-cols-1 lg:grid-cols-4">
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
      <ShipmentsByExpectedDeliverDateCard
        count={analytics?.shipmentsByExpectedDeliverDateCard?.count}
        date={analytics?.shipmentsByExpectedDeliverDateCard?.date}
        shipments={analytics?.shipmentsByExpectedDeliverDateCard?.shipments}
      />
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
    </dl>
  );
}

function ShipmentCountCard({
  count,
  trendPercentage,
}: {
  count: number;
  trendPercentage: number;
}) {
  return (
    <Card>
      <CardContent>
        <dt className="text-muted-foreground font-medium text-sm">
          Current Shipment Count
        </dt>
        <dd className="mt-2 flex items-baseline gap-x-2.5">
          <span className="text-primary font-semibold text-4xl">{count}</span>
          <span
            className={cn(
              trendPercentage > 0 ? "text-green-500" : "text-destructive",
              "text-primary font-medium text-sm",
            )}
          >
            {trendPercentage > 0 ? "+" : ""}
            {trendPercentage}%
          </span>
        </dd>
      </CardContent>
    </Card>
  );
}

type ShipmentSummary = {
  id: ShipmentSchema["id"];
  proNumber: ShipmentSchema["proNumber"];
  customerId: CustomerSchema["id"];
  customerName: CustomerSchema["name"];
  status: ShipmentSchema["status"];
  expectedDelivery: StopSchema["plannedArrival"];
  deliveryLocation: LocationSchema["name"];
  deliveryLocationId: LocationSchema["id"];
  createdAt: ShipmentSchema["createdAt"];
};

function ShipmentsByExpectedDeliverDateCard({
  count,
  date,
  shipments,
}: {
  count: number;
  date: number;
  shipments: ShipmentSummary[];
}) {
  const user = useUser();

  const [, setSearchParams] = useQueryStates(searchParamsParser);

  const handleShipmentClick = useCallback(
    (shipmentId: ShipmentSchema["id"]) => {
      setSearchParams({ entityId: shipmentId, modalType: "edit" });
    },
    [setSearchParams],
  );

  const handleExportToCsv = useCallback(() => {
    const csvData = convertToCsv({
      columns: [
        "Pro Number",
        "Customer Name",
        "Status",
        "Expected Delivery",
        "Delivery Location",
      ],
      rows: shipments.map((shipment) => [
        shipment.proNumber,
        shipment.customerName,
        shipment.status,
        formatToUserTimezone(shipment.expectedDelivery, {
          timeFormat: user?.timeFormat,
        }),
        shipment.deliveryLocation,
      ]),
    });

    downloadFile(
      generateFilename("shipments_expected_delivery", "csv"),
      csvData,
      "text/csv",
    );
  }, [shipments, user?.timeFormat]);

  return (
    <Card>
      <CardContent className="p-0">
        <div className="px-4">
          <div className="flex justify-between items-center">
            <dt className="text-muted-foreground font-medium text-sm">
              Shipments Planned to Deliver Today
            </dt>
            {shipments.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon">
                    <Icon icon={faEllipsis} />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuGroup>
                    <DropdownMenuLabel className="text-xs text-muted-foreground">
                      Actions
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      title="Export to CSV"
                      onClick={handleExportToCsv}
                    />
                    <DropdownMenuItem
                      title="Export to Excel (CSV)"
                      onClick={handleExportToCsv}
                    />
                    <DropdownMenuItem title="Export to JSON" />
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
          <dd className="flex items-baseline gap-x-2.5">
            <span className="text-primary font-semibold text-4xl">{count}</span>
            <span className="text-primary font-medium text-sm">
              {formatToUserTimezone(date, {
                timeFormat: user?.timeFormat,
                showSeconds: false,
                showTime: false,
              })}
            </span>
          </dd>
        </div>

        <ScrollArea className="h-[100px] px-4">
          <div className="flex flex-col gap-2 pb-4">
            {shipments.map((shipment) => (
              <div
                className="text-sm border border-border p-2 rounded-md bg-muted-foreground/10"
                key={shipment.id}
              >
                <div className="flex flex-col items-start gap-1">
                  <div className="flex items-center justify-between w-full">
                    <button
                      onClick={() => handleShipmentClick(shipment.id)}
                      className="underline cursor-pointer font-semibold text-primary hover:text-primary/70"
                    >
                      {shipment.proNumber}
                    </button>
                    <PlainShipmentStatusBadge status={shipment.status} />
                  </div>

                  <div className="flex flex-row gap-1 items-center">
                    <div className="text-sm text-muted-foreground">
                      Shipment is expected to be delivered to{" "}
                      <Link
                        to={`/dispatch/configurations/locations?entityId=${shipment.deliveryLocationId}&modalType=edit`}
                        target="_blank"
                        className="font-semibold text-primary underline hover:text-primary/70"
                      >
                        {shipment.deliveryLocation}
                      </Link>{" "}
                      on{" "}
                      <span className="font-semibold text-primary">
                        {formatToUserTimezone(shipment.expectedDelivery, {
                          timeFormat: user?.timeFormat,
                          showSeconds: false,
                          showTime: false,
                        })}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div className="pointer-events-none rounded-b-lg absolute bottom-0 z-50 left-0 right-0 h-8 bg-gradient-to-t from-card to-transparent" />
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
