import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { type CreateConsolidationSchema } from "@/lib/schemas/consolidation-schema";
import {
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { Calendar, MapPin, Package, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { useFormContext } from "react-hook-form";

export function ShipmentSelection() {
  const { setValue, watch } = useFormContext<CreateConsolidationSchema>();
  const selectedShipmentIds = watch("shipmentIds") || [];
  const [searchTerm, setSearchTerm] = useState("");

  // * Fetch available shipments
  const { data: shipments, isLoading } = useQuery({
    ...queries.shipment.list({
      limit: 100,
      offset: 0,
      filters: [
        {
          field: "status",
          operator: "eq",
          value: ShipmentStatus.enum.New,
        },
      ],
      sort: [
        {
          field: "createdAt",
          direction: "asc",
        },
      ],
      expandShipmentDetails: true,
    }),
  });

  // * Filter shipments based on search term
  const filteredShipments = useMemo(() => {
    if (!shipments) return [];
    if (!searchTerm) return shipments.results;

    const lowerSearch = searchTerm.toLowerCase();
    return shipments.results.filter(
      (shipment) =>
        shipment.proNumber?.toLowerCase().includes(lowerSearch) ||
        shipment.bol?.toLowerCase().includes(lowerSearch) ||
        shipment.customer?.name?.toLowerCase().includes(lowerSearch),
    );
  }, [shipments, searchTerm]);

  const handleToggleShipment = (shipmentId: string) => {
    const currentIds = selectedShipmentIds || [];
    const newIds = currentIds.includes(shipmentId)
      ? currentIds.filter((id) => id !== shipmentId)
      : [...currentIds, shipmentId];

    setValue("shipmentIds", newIds, {
      shouldValidate: true,
      shouldDirty: true,
    });
  };

  const handleSelectAll = () => {
    if (!filteredShipments) return;
    const allIds = filteredShipments
      .map((s) => s.id)
      .filter(Boolean) as string[];
    setValue("shipmentIds", allIds, {
      shouldValidate: true,
      shouldDirty: true,
    });
  };

  const handleClearAll = () => {
    setValue("shipmentIds", [], {
      shouldValidate: true,
      shouldDirty: true,
    });
  };

  if (isLoading) {
    return <ShipmentSelectionSkeleton />;
  }

  return (
    <div className="space-y-4">
      {/* Search and Actions */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by Pro #, BOL, or Customer..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleSelectAll}
          disabled={!filteredShipments?.length}
        >
          Select All
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleClearAll}
          disabled={!selectedShipmentIds?.length}
        >
          Clear All
        </Button>
      </div>

      {/* Selection Summary */}
      {selectedShipmentIds.length > 0 && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Package className="h-4 w-4" />
          <span>
            {selectedShipmentIds.length} shipment
            {selectedShipmentIds.length !== 1 ? "s" : ""} selected
          </span>
        </div>
      )}

      {/* Shipment List */}
      <ScrollArea className="h-[400px] rounded-md border">
        <div className="p-4 space-y-2">
          {filteredShipments?.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Package className="h-12 w-12 mx-auto mb-2 opacity-20" />
              <p>No shipments available for consolidation</p>
            </div>
          ) : (
            filteredShipments?.map((shipment) => (
              <ShipmentItem
                key={shipment.id}
                shipment={shipment}
                isSelected={selectedShipmentIds.includes(shipment.id!)}
                onToggle={() => handleToggleShipment(shipment.id!)}
              />
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

interface ShipmentItemProps {
  shipment: ShipmentSchema;
  isSelected: boolean;
  onToggle: () => void;
}

function ShipmentItem({ shipment, isSelected, onToggle }: ShipmentItemProps) {
  return (
    <div
      className={`flex items-start gap-3 p-3 rounded-lg border transition-colors cursor-pointer hover:bg-accent ${
        isSelected ? "border-primary bg-primary/5" : "border-border"
      }`}
      onClick={onToggle}
    >
      <Checkbox
        checked={isSelected}
        onCheckedChange={onToggle}
        onClick={(e) => e.stopPropagation()}
        className="mt-1"
      />
      <div className="flex-1 space-y-1">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="font-medium">
              {shipment.proNumber || "No Pro #"}
            </span>
            <Badge variant="secondary" className="text-xs">
              {shipment.bol}
            </Badge>
          </div>
          <span className="text-sm text-muted-foreground">
            {shipment.status}
            {shipment.customer?.name || "Unknown Customer"}
          </span>
        </div>

        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <div className="flex items-center gap-1">
            <MapPin className="h-3 w-3" />
            <span>{shipment.moves?.[0]?.stops?.length || 0} stops</span>
          </div>
          <div className="flex items-center gap-1">
            <Calendar className="h-3 w-3" />
            <span>
              {shipment.createdAt
                ? new Date(shipment.createdAt).toLocaleDateString()
                : "No date"}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function ShipmentSelectionSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Skeleton className="h-10 flex-1" />
        <Skeleton className="h-10 w-24" />
        <Skeleton className="h-10 w-24" />
      </div>
      <ScrollArea className="h-[400px] rounded-md border">
        <div className="p-4 space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
