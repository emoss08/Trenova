import {
  LocationCategoryAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import type { Location } from "@/types/location";
import { ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { LocationGeofenceControls } from "./location-geofence-editor";

export function LocationForm() {
  const { control } = useFormContext<Location>();
  const [addressDetailsOpen, setAddressDetailsOpen] = useState(false);

  return (
    <div className="space-y-6 p-3">
      <FormSection
        title="Basic Details"
        description="Identification, address, and operating boundary for this location."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              placeholder="Name"
              description="The name of the location."
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="The current status of the location."
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
              description="A unique code for this location."
              maxLength={10}
            />
          </FormControl>
          <FormControl cols="full">
            <LocationCategoryAutocompleteField
              control={control}
              rules={{ required: true }}
              name="locationCategoryId"
              label="Location Category"
              placeholder="Location Category"
              description="The category this location belongs to."
            />
          </FormControl>
          <FormControl cols="full" id="address-field-container">
            <AddressField
              control={control}
              rules={{ required: true }}
              label="Location"
              placeholder="Search for an address"
              description="Search for the address; the map and breakdown fields update automatically."
            />
          </FormControl>
          <FormControl cols="full">
            <div className="space-y-1.5">
              <p className="text-sm leading-none font-medium">Geofence shape</p>
              <LocationGeofenceControls />
            </div>
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="description"
              label="Notes"
              placeholder="Add any extra detail about this location"
              description="Optional notes for dispatchers and drivers."
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <Collapsible
        open={addressDetailsOpen}
        onOpenChange={setAddressDetailsOpen}
        className="rounded-md border bg-muted/20"
      >
        <CollapsibleTrigger className="flex w-full items-center justify-between gap-3 rounded-md px-3 py-2 text-left transition-colors hover:bg-muted/40">
          <span className="flex flex-col">
            <span className="text-sm font-medium">Address details</span>
            <span className="text-xs text-muted-foreground">
              Override the auto-populated address breakdown when needed.
            </span>
          </span>
          <ChevronRightIcon
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
              addressDetailsOpen && "rotate-90",
            )}
          />
        </CollapsibleTrigger>
        <CollapsibleContent className="border-t px-3 pt-3 pb-4">
          <FormGroup cols={2}>
            <FormControl cols="full">
              <InputField
                control={control}
                name="addressLine2"
                label="Address Line 2"
                placeholder="Address Line 2"
                description="Suite, unit, floor, or other secondary detail."
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
              <UsStateAutocompleteField
                control={control}
                name="stateId"
                label="State"
                placeholder="State"
                description="The U.S. state where the location is situated."
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                rules={{ required: true }}
                control={control}
                name="postalCode"
                label="Postal Code"
                placeholder="Postal Code"
                description="The ZIP code for the location."
              />
            </FormControl>
          </FormGroup>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
