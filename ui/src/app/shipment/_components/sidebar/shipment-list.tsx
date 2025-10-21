/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { statusChoices } from "@/lib/choices";
import { type ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import type { ShipmentListProps } from "@/types/shipment";
import {
  faBox,
  faFilter,
  faSearch,
  faTruck,
} from "@fortawesome/pro-regular-svg-icons";
import { useFormContext } from "react-hook-form";
import { ShipmentCard } from "./shipment-card";
import { FilterOptions } from "./shipment-filter-options";

// Define a loading shipment card component
function ShipmentCardSkeleton() {
  return (
    <div className="p-4 border border-sidebar-border rounded-md bg-card space-y-2">
      <div className="flex justify-between items-start">
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-8" />
      </div>
      <div className="flex justify-between items-start">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-4 w-16" />
      </div>
      <div className="space-y-1">
        <Skeleton className="h-3 w-52" />
        <Skeleton className="h-3 w-52" />
      </div>
    </div>
  );
}

export function ShipmentList({
  displayData,
  isLoading,
  onShipmentSelect,
  inputValue,
}: ShipmentListProps) {
  const { control } = useFormContext<ShipmentFilterSchema>();
  const renderContent = () => {
    // During loading, always show skeletons
    if (isLoading) {
      return Array.from({ length: 10 }, (_, index) => (
        <ShipmentCardSkeleton key={`skeleton-${index}`} />
      ));
    }

    // If we have data, show it
    if (displayData?.length > 0) {
      return displayData.map((shipment) => (
        <ShipmentCard
          key={shipment?.id}
          shipment={shipment}
          onSelect={onShipmentSelect}
          inputValue={inputValue}
        />
      ));
    }

    // Only show empty state if we're not loading and have no data
    return (
      <EmptyState
        title="No Shipments Found"
        description="Adjust your search criteria and try again"
        className="size-full border-none bg-transparent"
        icons={[faBox, faSearch, faTruck]}
      />
    );
  };

  return (
    <>
      <div className="flex-none p-2 space-y-2">
        <FilterOptions />
        <div className="flex flex-row gap-2 justify-start">
          <InputField
            control={control}
            name="search"
            placeholder="Search by Pro # or BOL"
            className="h-7 w-[250px]"
            icon={
              <Icon
                icon={faSearch}
                className="size-3.5 text-muted-foreground"
              />
            }
          />
          <SelectField
            control={control}
            name="status"
            placeholder="Status"
            className="h-7 w-30"
            options={statusChoices}
          />
          <Button
            variant="outline"
            size="icon"
            className="border-muted-foreground/20 bg-muted border"
          >
            <Icon icon={faFilter} className="size-3.5" />
          </Button>
        </div>
      </div>

      <div className="flex-1 min-h-0">
        <ScrollArea className="h-full">
          <div className="p-2 space-y-2">{renderContent()}</div>
        </ScrollArea>
      </div>
    </>
  );
}
