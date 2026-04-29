import type { DataTablePanelProps } from "@/types/data-table";
import type { Location } from "@/types/location";
import { LocationDialog } from "./location-dialog";

export function LocationPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Location>) {
  return <LocationDialog open={open} onOpenChange={onOpenChange} mode={mode} row={row} />;
}
