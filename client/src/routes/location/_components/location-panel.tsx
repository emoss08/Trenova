import type { DataTablePanelProps } from "@/types/data-table";
import { locationSchema, type Location } from "@/types/location";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import type { z } from "zod";
import { LocationDialog } from "./location-dialog";

type LocationFormInput = z.input<typeof locationSchema>;

export function LocationPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Location>) {
  const form = useForm<LocationFormInput, unknown, Location>({
    resolver: zodResolver(locationSchema),
    defaultValues: {
      status: "Active",
      code: "",
      name: "",
      locationCategoryId: "",
      description: null,
      addressLine1: "",
      addressLine2: null,
      city: "",
      stateId: "",
      postalCode: "",
      isGeocoded: false,
      longitude: null,
      latitude: null,
      placeId: null,
      geofenceType: "auto",
      geofenceRadiusMeters: 250,
      geofenceVertices: [],
    },
    mode: "onChange",
  });

  return (
    <LocationDialog open={open} onOpenChange={onOpenChange} mode={mode} row={row} form={form} />
  );
}
