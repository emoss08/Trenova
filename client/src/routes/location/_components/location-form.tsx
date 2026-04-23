import {
  LocationCategoryAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { Location } from "@/types/location";
import { useFormContext } from "react-hook-form";
import { LocationGeofenceControls } from "./location-geofence-editor";

export function LocationForm() {
  const { control } = useFormContext<Location>();

  return (
    <div className="space-y-6 p-3">
      <FormSection
        title="Basic Details"
        description="Identification, address, and operating boundary for this location."
      >
        <FormGroup cols={1}>
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
          <FormControl cols="full">
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
          <FormControl cols="full">
            <div className="space-y-1.5">
              <p className="text-sm leading-none font-medium required">Geofence</p>
              <LocationGeofenceControls />
            </div>
          </FormControl>
          <FormGroup cols={2}>
            <FormControl cols="full" id="address-field-container">
              <AddressField control={control} />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="addressLine2"
                label="Address Line 2"
                placeholder="Suite, floor, building, etc."
                description="Additional address details such as suite number, floor, or building name."
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="city"
                rules={{ required: true }}
                label="City"
                placeholder="City"
                description="City where the customer's primary office or billing address is located."
              />
            </FormControl>
            <FormControl>
              <UsStateAutocompleteField
                control={control}
                name="stateId"
                label="State"
                placeholder="State"
                description="U.S. state for the billing address. Used for tax jurisdiction determination and regional reporting."
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                rules={{ required: true }}
                control={control}
                name="postalCode"
                label="Postal Code"
                placeholder="e.g., 90210"
                description="ZIP or ZIP+4 code. Used for geocoding, mileage calculations, and tax jurisdiction lookups."
              />
            </FormControl>
          </FormGroup>

          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Notes"
              placeholder="Add any extra detail about this location"
              description="Optional notes for dispatchers and drivers."
              minRows={3}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
