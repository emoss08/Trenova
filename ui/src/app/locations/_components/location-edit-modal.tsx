import { GeocodedBadge } from "@/components/geocode-badge";
import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  locationSchema,
  type LocationSchema,
} from "@/lib/schemas/location-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function EditLocationModal({
  currentRecord,
}: EditTableSheetProps<LocationSchema>) {
  const form = useForm({
    resolver: zodResolver(locationSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      title="Location"
      formComponent={<LocationForm />}
      className="sm:max-w-[500px]"
      form={form}
      url="/locations/"
      queryKey="location-list"
      fieldKey="name"
      titleComponent={(currentRecord) => {
        return currentRecord ? (
          <div className="flex items-center gap-x-2">
            <span className="truncate size-full">{currentRecord.name}</span>
            {currentRecord.isGeocoded ? (
              <GeocodedBadge
                longitude={currentRecord.longitude as unknown as number}
                latitude={currentRecord.latitude as unknown as number}
                placeId={currentRecord.placeId}
              />
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <div className="rounded-full bg-red-500 size-2 animate-pulse" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>Not Geocoded</p>
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        ) : null;
      }}
    />
  );
}
