import googleLogo from "@/assets/brand-icons/google-ar21.svg";
import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { PulsatingDots } from "@/components/ui/pulsating-dots";
import { useDebounce } from "@/hooks/use-debounce";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { APIError } from "@/types/errors";
import { IntegrationType } from "@/types/integrations/integration";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { CheckIcon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { type Control, useFormContext } from "react-hook-form";

interface AddressLocationData {
  placeId?: string;
  name?: string;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  state?: string;
  stateId?: string;
  postalCode?: string;
  longitude?: number;
  latitude?: number;
}

interface AddressFieldProps {
  control: Control<any>;
  onLocationSelect?: (location: AddressLocationData) => void;
  populateFields?: boolean;
  nameField?: string;
  addressLine1Field?: string;
  addressLine2Field?: string;
  cityField?: string;
  stateIdField?: string;
  postalCodeField?: string;
  placeIdField?: string;
  longitudeField?: string;
  latitudeField?: string;
}

export function AddressField({
  control,
  onLocationSelect,
  populateFields = true,
  nameField = "name",
  addressLine1Field = "addressLine1",
  addressLine2Field = "addressLine2",
  cityField = "city",
  stateIdField = "stateId",
  postalCodeField = "postalCode",
  placeIdField = "placeId",
  longitudeField = "longitude",
  latitudeField = "latitude",
}: AddressFieldProps) {
  const { setValue } = useFormContext();
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const [selectedLocation, setSelectedLocation] = useState<string | null>(null);
  const debouncedInput = useDebounce(searchValue, 400);

  // Get integration by type
  const { data: integration, isLoading: integrationLoading } = useQuery({
    ...queries.integration.getIntegrationByType(IntegrationType.GoogleMaps),
  });

  // Fetch locations when search changes using React Query
  const {
    data: locationsData,
    isLoading,
    error,
  } = useQuery({
    ...queries.googleMaps.locationAutocomplete(debouncedInput),
  });

  // Parse locations data and filter out invalid locations
  const locations = useMemo(() => {
    if (!locationsData?.data?.details) return [];
    return (locationsData.data.details || []).filter(
      (loc): loc is NonNullable<typeof loc> =>
        !!loc &&
        !!loc.placeId &&
        !!loc.addressLine1 &&
        !!loc.city &&
        !!loc.state,
    );
  }, [locationsData]);

  // Fill in address details when a location is selected
  const handleSelect = (locationId: string) => {
    if (!locationId) return;

    const location = locations.find((loc) => loc.placeId === locationId);

    if (!location) return;

    if (populateFields) {
      if (nameField) setValue(nameField, location.name || "");
      if (addressLine1Field)
        setValue(addressLine1Field, location.addressLine1 || "");
      if (addressLine2Field)
        setValue(addressLine2Field, location.addressLine2 || "");
      if (cityField) setValue(cityField, location.city || "");
      if (postalCodeField) setValue(postalCodeField, location.postalCode || "");
      if (stateIdField) setValue(stateIdField, location.stateId || "");
      if (placeIdField) setValue(placeIdField, locationId);
      if (longitudeField) setValue(longitudeField, location.longitude || 0);
      if (latitudeField) setValue(latitudeField, location.latitude || 0);
    }

    if (onLocationSelect) {
      onLocationSelect(location);
    }

    setSelectedLocation(locationId);
    setOpen(false);
  };

  // Get API key error message if there's an error
  const apiKeyError = error
    ? (error as APIError)?.data?.detail || "Unknown error"
    : null;

  return (
    <div className="flex flex-col space-y-1.5">
      <div className="relative">
        <InputField
          control={control}
          rules={{ required: true }}
          name={addressLine1Field}
          label="Address Line 1"
          placeholder="Address Line 1"
          description="The primary address line."
        />
        {integrationLoading ? (
          <div className="absolute right-0 top-6 inset-y-0 mr-2 flex items-center size-6">
            <PulsatingDots size={1} color="foreground" />
          </div>
        ) : integration?.configuration?.apiKey ? (
          <div className="absolute right-0 top-6 inset-y-0 mr-2 flex items-center size-6">
            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger asChild>
                <Button
                  id="address-search-button"
                  variant="outline"
                  size="sm"
                  type="button"
                  className="size-6 [&>svg]:size-3"
                >
                  <Icon icon={faSearch} className="size-4" />
                  <span className="sr-only">Search addresses</span>
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-96 p-0">
                <Command>
                  <CommandInput
                    placeholder="Search for an address..."
                    value={searchValue}
                    onValueChange={setSearchValue}
                    className="h-9"
                  />
                  <CommandList>
                    <CommandEmpty>
                      {isLoading ? (
                        <PulsatingDots size={2} color="foreground" />
                      ) : apiKeyError ? (
                        <LocationSearchError error={apiKeyError} />
                      ) : (
                        "No locations found."
                      )}
                    </CommandEmpty>
                    <CommandGroup>
                      {locations.map((location) => (
                        <CommandItem
                          key={location.placeId}
                          value={`${location.placeId} ${location.addressLine1} ${location.name}`}
                          onSelect={() =>
                            location.placeId && handleSelect(location.placeId)
                          }
                        >
                          <CheckIcon
                            className={cn(
                              "mr-2 h-4 w-4",
                              selectedLocation === location.placeId
                                ? "opacity-100"
                                : "opacity-0",
                            )}
                          />
                          <div className="flex flex-col">
                            <span>{location.name || "Unknown Location"}</span>
                            <span className="text-sm text-muted-foreground">
                              {location.addressLine1}, {location.city || ""},
                              {location.state ? ` ${location.state}` : ""}
                              {location.postalCode
                                ? ` ${location.postalCode}`
                                : ""}
                            </span>
                          </div>
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </CommandList>
                </Command>
                <div className="flex items-center gap-0.5 p-1 text-xs text-muted-foreground border-t">
                  Powered by{" "}
                  <LazyImage
                    src={googleLogo}
                    layout="constrained"
                    objectFit="contain"
                    width={50}
                    height={10}
                  />
                </div>
              </PopoverContent>
            </Popover>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function LocationSearchError({ error }: { error: string }) {
  return (
    <div className="flex flex-col items-center gap-2.5 py-3 px-4 animate-in fade-in duration-300">
      <div className="flex items-center gap-2 p-3 w-full max-w-md border border-destructive/30 rounded-lg bg-destructive/5 text-destructive shadow-sm">
        <div className="flex flex-col space-y-0.5">
          <span className="font-medium text-sm">API Key Error</span>
          <span className="text-xs text-destructive/80">{error}</span>
        </div>
      </div>
      <div className="text-center text-sm text-muted-foreground flex flex-col gap-1">
        <span>This feature requires a valid Google Maps API key.</span>
        <span>Please contact your IT administrator for assistance.</span>
      </div>
    </div>
  );
}
