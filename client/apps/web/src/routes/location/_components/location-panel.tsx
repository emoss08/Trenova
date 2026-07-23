import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { Location } from "@trenova/shared/types/location";
import { LocationDialog } from "./location-dialog";

export function LocationPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Location>) {
  return <LocationDialog open={open} onOpenChange={onOpenChange} mode={mode} row={row} />;
}
