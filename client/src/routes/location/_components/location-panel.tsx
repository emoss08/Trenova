import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { GeocodedBadge } from "@/components/geocode-badge";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { formatToUserTimezone } from "@/lib/date";
import { useAuthStore } from "@/stores/auth-store";
import type { DataTablePanelProps } from "@/types/data-table";
import { locationSchema, type Location } from "@/types/location";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function LocationPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Location>) {
  const user = useAuthStore((s) => s.user);
  const panelDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(
        row.updatedAt as number,
        {
          timeFormat: user?.timeFormat || "24-hour",
        },
        user?.timezone,
      )}`
    : undefined;

  const form = useForm({
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

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/locations/"
        queryKey="location-list"
        title="Location"
        size="xl"
        formComponent={<LocationForm />}
        fieldKey="code"
        titleComponent={(currentRecord) => {
          return (
            <div className="flex flex-col gap-0.5">
              <DialogTitle className="flex items-center justify-start gap-x-1">
                <span className="truncate">{currentRecord.name}</span>
                {currentRecord.isGeocoded ? (
                  <GeocodedBadge
                    longitude={currentRecord.longitude as unknown as number}
                    latitude={currentRecord.latitude as unknown as number}
                    placeId={currentRecord.placeId ?? undefined}
                  />
                ) : null}
              </DialogTitle>
              <DialogDescription>{panelDescription}</DialogDescription>
            </div>
          );
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/locations/"
      queryKey="location-list"
      title="Location"
      size="xl"
      formComponent={<LocationForm />}
    />
  );
}
