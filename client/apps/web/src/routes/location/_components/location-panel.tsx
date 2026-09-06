import type { LocationRow } from "@/lib/graphql/location-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { LocationDialog } from "./location-dialog";

export function LocationPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<LocationRow>) {
  return <LocationDialog open={open} onOpenChange={onOpenChange} mode={mode} row={row} />;
}
