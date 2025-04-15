import googleLogo from "@/assets/brand-icons/google-ar21.svg";
import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { PulsatingDots } from "@/components/ui/pulsating-dots";
import { useDebounce } from "@/hooks/use-debounce";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { cn } from "@/lib/utils";
import type { APIError } from "@/types/errors";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { CheckIcon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useFormContext } from "react-hook-form";

export function LocationForm() {
  const { control } = useFormContext<LocationSchema>();

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Defines the current operational status of the location."
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="A unique identifier for the location."
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The official name of the location."
        />
      </FormControl>
      <FormControl cols="full">
        <FormControl>
          <AutocompleteField<LocationCategorySchema, LocationSchema>
            name="locationCategoryId"
            control={control}
            rules={{ required: true }}
            link="/location-categories/"
            label="Location Category"
            placeholder="Select Location Category"
            description="Select the location category for the location."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.name} />
            )}
            renderOption={(option) => (
              <ColorOptionValue color={option.color} value={option.name} />
            )}
          />
        </FormControl>
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Additional details or notes about the location."
        />
      </FormControl>
      <FormControl cols="full">
        <AddressLineWithSearch control={control} />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="addressLine2"
          label="Address Line 2"
          placeholder="Address Line 2"
          description="Additional address details, if applicable."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="city"
          rules={{ required: true }}
          label="City"
          placeholder="City"
          description="The city where the location is situated."
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="stateId"
          label="State"
          placeholder="State"
          menuPlacement="top"
          description="The U.S. state where the location is situated."
          options={usStateOptions}
          isLoading={usStates.isLoading}
          isFetchError={usStates.isError}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="postalCode"
          label="Postal Code"
          placeholder="Postal Code"
          description="The ZIP code for the location."
        />
      </FormControl>
    </FormGroup>
  );
}

function AddressLineWithSearch({ control }: { control: any }) {
  const { setValue } = useFormContext<LocationSchema>();
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const [selectedLocation, setSelectedLocation] = useState<string | null>(null);
  const debouncedInput = useDebounce(searchValue, 400);

  // Check if Google Maps API key is configured
  const { data: apiKeyCheck, isLoading: apiKeyCheckLoading } = useQuery({
    ...queries.googleMaps.checkAPIKey(),
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

    setValue("name", location.name || "");
    setValue("addressLine1", location.addressLine1 || "");
    setValue("addressLine2", location.addressLine2 || "");
    setValue("city", location.city || "");
    setValue("postalCode", location.postalCode || "");
    setValue("stateId", location.stateId || "");
    setValue("placeId", locationId);
    setValue("longitude", location.longitude || 0);
    setValue("latitude", location.latitude || 0);

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
          name="addressLine1"
          label="Address Line 1"
          placeholder="Address Line 1"
          description="The primary address line."
        />
        {apiKeyCheckLoading ? (
          <div className="absolute right-0 top-6 inset-y-0 mr-2 flex items-center size-6">
            <PulsatingDots size={1} color="foreground" />
          </div>
        ) : apiKeyCheck?.data?.valid ? (
          <div className="absolute right-0 top-6 inset-y-0 mr-2 flex items-center size-6">
            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger asChild>
                <Button
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
                <div className="flex items-center gap-0.5  p-1 text-xs text-muted-foreground border-t">
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
