/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { useShipments } from "@/app/shipment/queries/shipment";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { HeaderBackButton } from "@/components/ui/sheet-header-components";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { ZoomableImage } from "@/components/zoomable-image";
import { formatDurationFromSeconds } from "@/lib/date";
import { queries } from "@/lib/queries";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import {
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import {
  calculateShipmentDuration,
  calculateShipmentMileage,
  getShipmentStopCount,
  ShipmentLocations,
} from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { Resource } from "@/types/audit-entry";
import type { TableSheetProps } from "@/types/data-table";
import {
  faBox,
  faChevronDown,
  faChevronUp,
} from "@fortawesome/pro-regular-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

export function TransferDialog({ ...props }: TableSheetProps) {
  const { open, onOpenChange } = props;

  return (
    <Sheet {...props}>
      <SheetContent withClose={false} className="w-[600px] sm:max-w-[640px]">
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Transfer Shipments</SheetTitle>
            <SheetDescription>
              Transfer the selected shipments to the billing team
            </SheetDescription>
          </SheetHeader>
        </VisuallyHidden>
        <SheetBody>
          <div className="flex items-center pb-4 justify-between">
            <HeaderBackButton onBack={() => onOpenChange?.(false)} />
          </div>
          <TransferDialogContent open={open} />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function TransferDialogContent({ open }: { open: boolean }) {
  const { data } = useShipments({
    expandShipmentDetails: true,
    filters: [
      {
        field: "status",
        operator: "eq",
        value: ShipmentStatus.enum.ReadyToBill,
      },
    ],
    limit: 10,
    offset: 0,
    enabled: open,
  });

  return (
    <div className="flex flex-col gap-2 border-t border-border pt-4">
      <div className="flex items-center justify-between text-xl font-medium">
        <p className="text-muted-foreground">Total Shipments:</p>
        <span className="text-xl font-semibold">{data?.count ?? 0}</span>
      </div>
      {data?.results && (
        <div className="flex flex-col gap-2">
          {data?.results.map((shipment) => (
            <ShipmentCard key={shipment.id} shipment={shipment} />
          ))}
        </div>
      )}
    </div>
  );
}

function ShipmentCard({ shipment }: { shipment: ShipmentSchema }) {
  const [detailsOpen, setDetailsOpen] = useState<boolean>(false);

  const { data: documents } = useQuery({
    ...queries.document.documentsByResourceID(
      Resource.Shipment,
      shipment.id ?? "",
      10,
      0,
      detailsOpen,
    ),
  });

  const { origin, destination } = ShipmentLocations.useLocations(shipment);

  const mileage = calculateShipmentMileage(shipment);
  const transitTime = calculateShipmentDuration(shipment);
  const stopCount = getShipmentStopCount(shipment);

  return (
    <div className="bg-card border border-border rounded-lg p-4">
      <div className="flex items-center gap-2">
        <div className="bg-muted rounded-lg p-4 flex items-center justify-center size-12">
          <Icon icon={faBox} className="size-6" />
        </div>
        <div className="flex items-center justify-between w-full">
          <div className="flex flex-col">
            <p className="text-sm font-medium text-muted-foreground">
              Shipment ID
            </p>
            <p className="text-xl font-semibold">
              {shipment?.proNumber || "-"}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setDetailsOpen(!detailsOpen)}
          >
            <Icon icon={detailsOpen ? faChevronUp : faChevronDown} />
          </Button>
        </div>
      </div>
      {detailsOpen && (
        <div className="flex flex-col gap-2 border-t border-border my-4">
          <div className="flex flex-col gap-2 py-2">
            <ShipmentDetailSectionItem
              label="Customer"
              value={shipment?.customer?.name ?? "-"}
            />
            <ShipmentDetailSectionItem
              label="Pro #"
              value={shipment?.proNumber ?? "-"}
            />
            <ShipmentDetailSectionItem
              label="BOL"
              value={shipment?.bol ?? "-"}
            />
            <ShipmentDetailSectionItem
              label="Actual Ship Date"
              value={shipment?.actualShipDate ?? "-"}
            />
            <ShipmentDetailSectionItem
              label="Actual Delivery Date"
              value={shipment?.actualDeliveryDate ?? "-"}
            />
          </div>
          <div className="flex flex-col gap-4">
            <div className="flex items-center border-t border-border pt-4">
              <h3 className="text-xl font-medium">Delivery Information</h3>
            </div>
            <div className="grid grid-cols-3 items-center text-center justify-between gap-2">
              <div className="flex flex-col items-center gap-1 rounded-lg bg-green-50 dark:bg-green-600/30 p-2">
                <p className="text-sm font-medium">Distance</p>
                <h3 className="text-xl font-semibold">{mileage} miles</h3>
              </div>
              <div className="flex flex-col items-center gap-1 rounded-lg bg-amber-50 dark:bg-amber-600/30 p-2">
                <p className="text-sm font-medium">Transit Time</p>
                <h3 className="text-xl font-semibold">
                  {formatDurationFromSeconds(transitTime)}
                </h3>
              </div>
              <div className="flex flex-col items-center gap-1 rounded-lg bg-blue-50 dark:bg-blue-600/30 p-2">
                <p className="text-sm font-medium">Stop Count</p>
                <h3 className="text-xl font-semibold">{stopCount}</h3>
              </div>
            </div>
            <div className="grid grid-cols-2 items-center justify-between gap-2">
              <ShipmentLocationCard
                location={origin as LocationSchema}
                label="Origin"
              />
              <ShipmentLocationCard
                location={destination as LocationSchema}
                label="Destination"
              />
            </div>
          </div>
          <div className="flex flex-col gap-4">
            <div className="flex items-center border-t border-border pt-4">
              <h3 className="text-xl font-medium">Documents</h3>
            </div>
            <div className="grid grid-cols-2 gap-2">
              {documents?.results?.map((document) => (
                <div
                  key={document.id}
                  className="flex items-center justify-center gap-2 p-2 bg-muted dark:bg-background border border-border rounded-lg"
                >
                  <div className="relative size-[200px] overflow-hidden shadow-sm">
                    <ZoomableImage
                      src={document.previewUrl ?? ""}
                      alt={document.fileName ?? ""}
                      className="size-full object-cover"
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ShipmentDetailSectionItem({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  const valueType = typeof value;

  const valueComponent =
    valueType === "string" ? (
      <p className="text-sm font-medium">{value}</p>
    ) : (
      <HoverCardTimestamp
        align="start"
        side="left"
        className="text-sm font-medium"
        timestamp={value as number}
      />
    );

  return (
    <div className="flex items-center justify-between">
      <p className="text-sm text-muted-foreground">{label}</p>
      {valueComponent}
    </div>
  );
}

function ShipmentLocationCard({
  location,
  label,
}: {
  location: LocationSchema;
  label: string;
}) {
  return (
    <div className="flex flex-col max-w-[250px]">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="text-sm font-medium truncate">
        {formatLocation(location) ?? "-"}
      </p>
    </div>
  );
}
