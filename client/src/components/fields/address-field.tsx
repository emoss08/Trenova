import { useDebounce } from "@/hooks/use-debounce";
import type { ApiRequestError } from "@/lib/api";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { LocationDetails } from "@/types/google-maps";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, SearchIcon } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import {
  useFormContext,
  type Control,
  type FieldValues,
  type Path,
  type PathValue,
  type RegisterOptions,
} from "react-hook-form";
import { Button } from "../ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { Spinner } from "../ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { InputField } from "./input-field";

type AddressFieldProps<TForm extends FieldValues> = {
  control: Control<TForm>;
  onLocationSelect?: (location: LocationDetails) => void;
  populateFields?: boolean;
  nameField?: Path<TForm>;
  addressLine1Field?: Path<TForm>;
  cityField?: Path<TForm>;
  stateIdField?: Path<TForm>;
  postalCodeField?: Path<TForm>;
  placeIdField?: Path<TForm>;
  longitudeField?: Path<TForm>;
  latitudeField?: Path<TForm>;
  label?: string;
  placeholder?: string;
  description?: string;
  rules?: RegisterOptions<TForm, Path<TForm>>;
};

export function AddressField<TForm extends FieldValues>({
  control,
  onLocationSelect,
  populateFields = true,
  nameField = "name" as Path<TForm>,
  addressLine1Field = "addressLine1" as Path<TForm>,
  cityField = "city" as Path<TForm>,
  stateIdField = "stateId" as Path<TForm>,
  postalCodeField = "postalCode" as Path<TForm>,
  placeIdField = "placeId" as Path<TForm>,
  longitudeField = "longitude" as Path<TForm>,
  latitudeField = "latitude" as Path<TForm>,
  label = "Address Line 1",
  placeholder = "Address Line 1",
  description = "The primary address line.",
  rules,
}: AddressFieldProps<TForm>) {
  const { setValue } = useFormContext<TForm>();
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const [selectedLocation, setSelectedLocation] = useState<string | null>(null);
  const sessionTokenRef = useRef(crypto.randomUUID());
  const sessionToken = sessionTokenRef.current;
  const debouncedInput = useDebounce(searchValue, 400);

  const configQuery = useQuery({
    ...queries.integration.config("GoogleMaps"),
    staleTime: 5 * 60 * 1000,
  });

  const isSearchAvailable =
    configQuery.data?.enabled === true &&
    (configQuery.data?.fields?.some((f) => f.key === "apiKey" && f.hasValue) ?? false);

  const enabled = isSearchAvailable && debouncedInput.length > 3;

  const {
    data: autocompleteResult,
    isFetching,
    error,
  } = useQuery({
    ...queries.googleMaps.autocomplete({
      input: debouncedInput,
      sessionToken,
    }),
    enabled,
  });

  const locations = useMemo(() => {
    if (!autocompleteResult?.details) return [] as LocationDetails[];

    const filtered = autocompleteResult.details.filter(
      (loc): loc is NonNullable<typeof loc> =>
        !!loc && !!loc.placeId && !!loc.addressLine1 && !!loc.city && !!loc.state,
    );

    return filtered;
  }, [autocompleteResult]);

  const setFieldValue = useCallback(
    <TPath extends Path<TForm>>(field: TPath, value: PathValue<TForm, TPath>) => {
      setValue(field, value, {
        shouldDirty: true,
        shouldValidate: true,
      });
    },
    [setValue],
  );

  const handleSelect = useCallback(
    (locationId: LocationDetails["placeId"]) => {
      if (!locationId) return;

      const location = locations.find((loc) => loc.placeId === locationId);

      if (!location) return;

      if (populateFields) {
        if (nameField) setFieldValue(nameField, (location.name || "") as PathValue<TForm, typeof nameField>);
        if (addressLine1Field) {
          setFieldValue(
            addressLine1Field,
            (location.addressLine1 || "") as PathValue<TForm, typeof addressLine1Field>,
          );
        }
        if (cityField) setFieldValue(cityField, (location.city || "") as PathValue<TForm, typeof cityField>);
        if (postalCodeField) {
          setFieldValue(
            postalCodeField,
            (location.postalCode || "") as PathValue<TForm, typeof postalCodeField>,
          );
        }
        if (stateIdField) {
          setFieldValue(
            stateIdField,
            (location.stateId || "") as PathValue<TForm, typeof stateIdField>,
          );
        }
        if (placeIdField) {
          setFieldValue(placeIdField, locationId as PathValue<TForm, typeof placeIdField>);
        }
        if (longitudeField) {
          setFieldValue(
            longitudeField,
            (location.longitude || 0) as PathValue<TForm, typeof longitudeField>,
          );
        }
        if (latitudeField) {
          setFieldValue(
            latitudeField,
            (location.latitude || 0) as PathValue<TForm, typeof latitudeField>,
          );
        }
      }

      onLocationSelect?.(location);

      setSelectedLocation(locationId);
      setOpen(false);

      sessionTokenRef.current = crypto.randomUUID();
    },
    [
      locations,
      populateFields,
      nameField,
      addressLine1Field,
      cityField,
      postalCodeField,
      stateIdField,
      placeIdField,
      longitudeField,
      latitudeField,
      setFieldValue,
      onLocationSelect,
    ],
  );

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen && !isSearchAvailable) return;
      setOpen(nextOpen);
      if (nextOpen) {
        sessionTokenRef.current = crypto.randomUUID();
      }
    },
    [isSearchAvailable],
  );

  const apiKeyError = error ? (error as ApiRequestError)?.data?.title || "Unknown error" : null;

  return (
    <div className="flex flex-col space-y-1.5">
      <div className="relative">
        <InputField
          control={control}
          rules={rules}
          name={addressLine1Field}
          label={label}
          placeholder={placeholder}
          description={description}
        />

        {isSearchAvailable ? (
          <Popover open={open} onOpenChange={handleOpenChange}>
            <PopoverTrigger
              render={
                <Button
                  className="absolute top-1/2 right-2 mt-0.5 -translate-y-1/2"
                  size="icon-xs"
                  variant="ghost"
                >
                  <span
                    id="address-search-button"
                    className="flex cursor-pointer items-center gap-1 rounded-md p-1 text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground"
                  >
                    <SearchIcon className="size-3 text-muted-foreground" />
                    <span className="sr-only">Search addresses</span>
                  </span>
                </Button>
              }
            />
            <PopoverContent className="w-96 p-0">
              <Command shouldFilter={false}>
                <CommandInput
                  placeholder="Search for an address..."
                  value={searchValue}
                  onValueChange={setSearchValue}
                  className="h-9"
                />
                <CommandList>
                  {isFetching ? (
                    <div className="flex justify-center py-4">
                      <Spinner className="size-4" />
                    </div>
                  ) : locations.length === 0 ? (
                    <CommandEmpty>
                      {apiKeyError ? (
                        <LocationSearchError error={apiKeyError} />
                      ) : (
                        <div className="flex min-h-[100px] flex-col justify-center gap-1 text-center text-sm text-muted-foreground">
                          <span>No locations found.</span>
                          <span>Please try a different search.</span>
                        </div>
                      )}
                    </CommandEmpty>
                  ) : (
                    <CommandGroup>
                      {locations.map((location) => (
                        <CommandItem
                          key={location.placeId}
                          value={location.placeId}
                          className="justify-between py-1"
                          onSelect={() => {
                            if (location.placeId) handleSelect(location.placeId);
                          }}
                        >
                          <div className="flex flex-col">
                            <span>{location.name || "Unknown Location"}</span>
                            <span className="text-2xs text-muted-foreground">
                              {location.addressLine1}
                              {location.city ? `, ${location.city}` : ""}
                              {location.state ? `, ${location.state}` : ""}
                              {location.postalCode ? ` ${location.postalCode}` : ""}
                            </span>
                          </div>
                          <CheckIcon
                            className={cn(
                              "size-3",
                              selectedLocation === location.placeId ? "opacity-100" : "opacity-0",
                            )}
                          />
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  )}
                </CommandList>
                <div className="flex items-center justify-between gap-0.5 border-t bg-muted px-2 py-0.5 text-2xs text-muted-foreground">
                  <div className="flex items-center gap-0.5">Powered by Google Maps</div>
                  <div>Found {locations.length} locations</div>
                </div>
              </Command>
            </PopoverContent>
          </Popover>
        ) : (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="absolute top-1/2 right-2 mt-0.5 inline-flex -translate-y-1/2">
                  <Button size="icon-xs" variant="ghost" disabled>
                    <span className="flex items-center gap-1 rounded-md p-1">
                      <SearchIcon className="size-3 text-muted-foreground/40" />
                    </span>
                  </Button>
                </span>
              }
            />
            <TooltipContent>
              Address lookup requires the Google Maps integration to be configured.
            </TooltipContent>
          </Tooltip>
        )}
      </div>
    </div>
  );
}

function LocationSearchError({ error }: { error: string }) {
  return (
    <div className="flex animate-in flex-col items-center gap-2.5 px-4 py-3 duration-300 fade-in">
      <div className="flex w-full max-w-md items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-destructive shadow-sm">
        <div className="flex flex-col space-y-0.5">
          <span className="text-sm font-medium">API Key Error</span>
          <span className="text-xs text-destructive/80">{error}</span>
        </div>
      </div>
      <div className="flex flex-col gap-1 text-center text-sm text-muted-foreground">
        <span>This feature requires a valid Google Maps API key.</span>
        <span>Please contact your IT administrator for assistance.</span>
      </div>
    </div>
  );
}
