import { ShipmentSearchForm } from "@/types/shipment";
import {
    CaretSortIcon,
    DownloadIcon,
    MagnifyingGlassIcon,
    PlusIcon,
} from "@radix-ui/react-icons";
import { useFormContext } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { Button } from "../ui/button";

export function ShipmentToolbar() {
  return (
    <div className="flex justify-between">
      <ShipmentSearch />
      <div className="space-x-2">
        <Button variant="outline" size="sm">
          <DownloadIcon className="mr-1 size-4" />
          Export
        </Button>
        <Button variant="outline" size="sm">
          <CaretSortIcon className="mr-1 size-4" />
          Filter
        </Button>
        <Button variant="outline" size="sm">
          <PlusIcon className="mr-1 size-4" />
          New Shipment
        </Button>
      </div>
    </div>
  );
}

function ShipmentSearch() {
  const { control } = useFormContext<ShipmentSearchForm>();

  return (
    <div className="relative">
      <InputField
        name="searchQuery"
        control={control}
        placeholder="Search Shipments..."
        icon={<MagnifyingGlassIcon className="text-muted-foreground size-4" />}
      />
    </div>
  );
}
