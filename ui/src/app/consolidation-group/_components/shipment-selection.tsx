/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import UsFlagIcon from "@/assets/brand-icons/Flag_of_the_United_States.svg";
import { PlainShipmentStatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import { queries } from "@/lib/queries";
import { type ConsolidationGroupSchema } from "@/lib/schemas/consolidation-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import { ShipmentSchema, ShipmentStatus } from "@/lib/schemas/shipment-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { cn, formatLocation } from "@/lib/utils";
import { faArrowRight } from "@fortawesome/pro-solid-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ChevronDown } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useFormContext } from "react-hook-form";

export function ShipmentSelection({
  shipmentErrors,
}: {
  shipmentErrors?: string | null;
}) {
  const { setValue, watch } = useFormContext<ConsolidationGroupSchema>();
  const selectedShipments = watch("shipments") || [];
  const [openShipments, setOpenShipments] = useState<Set<string>>(new Set());
  const parentRef = useRef<HTMLDivElement>(null);

  const selectedShipmentIds = selectedShipments.map((s) => s.id);

  const { data: shipments, isLoading } = useQuery({
    ...queries.shipment.list({
      limit: 100,
      offset: 0,
      expandShipmentDetails: true,
      filters: [
        {
          field: "status",
          operator: "in",
          value: [ShipmentStatus.enum.New, ShipmentStatus.enum.Delayed],
        },
        {
          field: "consolidationGroupId",
          operator: "isnull",
          value: true,
        },
      ],
    }),
  });

  const rowVirtualizer = useVirtualizer({
    count: shipments?.results?.length || 0,
    getScrollElement: () => parentRef.current,
    estimateSize: useCallback(
      (index: number) => {
        const shipmentId = shipments?.results?.[index]?.id;
        // Return larger estimate for expanded items
        return openShipments.has(shipmentId ?? "") ? 260 : 70;
      },
      [openShipments, shipments?.results],
    ),
    overscan: 5,
    measureElement: (element) => {
      // Use ResizeObserver to get the actual height
      return element?.getBoundingClientRect().height ?? 70;
    },
  });

  const handleToggle = (shipment: ShipmentSchema) => {
    const isSelected = selectedShipmentIds.includes(shipment.id);

    if (isSelected) {
      const filteredShipments = selectedShipments.filter(
        (s) => s.id !== shipment.id,
      );
      setValue("shipments", filteredShipments);
    } else {
      setValue("shipments", [...selectedShipments, shipment]);
    }
  };

  const handleCollapsibleChange = useCallback(
    (shipmentId: string, open: boolean) => {
      const newOpenShipments = new Set(openShipments);
      if (open) {
        newOpenShipments.add(shipmentId);
      } else {
        newOpenShipments.delete(shipmentId);
      }
      setOpenShipments(newOpenShipments);

      // Force remeasure after state change
      setTimeout(() => {
        rowVirtualizer.measure();
      }, 0);
    },
    [openShipments, rowVirtualizer],
  );

  if (isLoading) {
    return <div>Loading shipments...</div>;
  }

  if (!shipments?.results?.length) {
    return <div>No shipments available for consolidation</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            const allShipments = shipments.results.map((s) => s);
            setValue("shipments", allShipments);
          }}
        >
          Select All
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setValue("shipments", [])}
        >
          Clear All
        </Button>
      </div>

      <div
        ref={parentRef}
        className="border rounded-lg p-4 overflow-y-auto bg-background h-[400px]"
      >
        {shipmentErrors && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-2 mb-4">
            {shipmentErrors.split(";").map((error, idx) => (
              <div key={idx} className="text-red-100 text-sm">
                {error}
              </div>
            ))}
          </div>
        )}

        <div
          style={{
            height: `${rowVirtualizer.getTotalSize()}px`,
            width: "100%",
            position: "relative",
          }}
        >
          {rowVirtualizer.getVirtualItems().map((virtualRow) => {
            const shipment = shipments.results[virtualRow.index];
            if (!shipment) return null;

            return (
              <div
                key={virtualRow.key}
                data-index={virtualRow.index}
                ref={rowVirtualizer.measureElement}
                style={{
                  position: "absolute",
                  top: 0,
                  left: 0,
                  width: "100%",
                  transform: `translateY(${virtualRow.start}px)`,
                }}
                className="pb-2"
              >
                <Collapsible
                  open={openShipments.has(shipment.id ?? "")}
                  onOpenChange={(open) =>
                    handleCollapsibleChange(shipment.id ?? "", open)
                  }
                >
                  <div className="border rounded-lg p-2 bg-card">
                    <div className="flex items-center space-x-3">
                      <Checkbox
                        checked={selectedShipmentIds.includes(shipment.id)}
                        onCheckedChange={() => handleToggle(shipment)}
                      />
                      <div className="flex-1">
                        <div className="font-medium">
                          {shipment.proNumber || "No Pro #"}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {shipment.customer?.name || "Unknown Customer"}
                        </div>
                      </div>
                      <CollapsibleTrigger asChild>
                        <Button variant="ghost" size="icon" type="button">
                          <ChevronDown
                            className={cn(
                              "size-4",
                              openShipments.has(shipment.id ?? "")
                                ? "rotate-180"
                                : "",
                            )}
                          />
                        </Button>
                      </CollapsibleTrigger>
                    </div>
                    <CollapsibleContent className="pt-4">
                      <div className="pl-9 space-y-6 text-sm">
                        <div className="grid grid-cols-2 gap-2">
                          <div className="flex items-center gap-x-2">
                            <span className="text-muted-foreground">
                              Status:
                            </span>{" "}
                            <PlainShipmentStatusBadge
                              status={shipment.status}
                            />
                          </div>
                          <div>
                            <span className="text-muted-foreground">
                              Service Type:
                            </span>{" "}
                            {shipment.serviceType?.code || "N/A"}
                          </div>
                        </div>
                        {(() => {
                          const origin = ShipmentLocations.getOrigin(shipment);
                          const destination =
                            ShipmentLocations.getDestination(shipment);

                          return (
                            <div className="flex items-center gap-x-6 w-full">
                              <div className="flex justify-start items-center gap-x-1">
                                <LazyImage
                                  src={UsFlagIcon}
                                  alt="US Flag"
                                  className="rounded-full size-4 object-cover"
                                />
                                <span className="text-muted-foreground truncate">
                                  {formatLocation(origin as LocationSchema)}
                                </span>
                              </div>
                              <span className="text-muted-foreground">
                                <Icon icon={faArrowRight} />
                              </span>
                              <div className="flex justify-start items-center gap-x-1">
                                <LazyImage
                                  src={UsFlagIcon}
                                  alt="US Flag"
                                  className="rounded-full size-4 object-cover"
                                />
                                <span className="text-muted-foreground truncate">
                                  {formatLocation(
                                    destination as LocationSchema,
                                  )}
                                </span>
                              </div>
                            </div>
                          );
                        })()}
                      </div>
                    </CollapsibleContent>
                  </div>
                </Collapsible>
              </div>
            );
          })}
        </div>
      </div>

      {selectedShipments.length > 0 && (
        <div className="text-sm text-muted-foreground">
          {selectedShipments.length} shipment(s) selected
        </div>
      )}
    </div>
  );
}
