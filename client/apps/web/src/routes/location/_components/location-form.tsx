import { useT } from "@trenova/shared/i18n/use-t";
import {
  LocationCategoryAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { statusChoices, timezoneGroupedChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { Location } from "@trenova/shared/types/location";
import { useFormContext } from "react-hook-form";
import { LocationGeofenceControls } from "./location-geofence-editor";

export function LocationForm() {
  const t = useT();

  const { control } = useFormContext<Location>();
  const googleMapsQuery = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
    staleTime: 5 * 60 * 1000,
  });

  return (
    <div className="space-y-6 p-3">
      <FormSection
        title={t("Basic Details")}
        description={t("Identification, address, and operating boundary for this location.")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t("The current status of the location.")}
              options={statusChoices}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("Name")}
              description={t("The name of the location.")}
              maxLength={255}
            />
          </FormControl>
          <FormControl cols="full">
            <LocationCategoryAutocompleteField
              control={control}
              rules={{ required: true }}
              name="locationCategoryId"
              label={t("Location Category")}
              placeholder={t("Location Category")}
              description={t("The category this location belongs to.")}
            />
          </FormControl>
          {googleMapsQuery.isLoading ? (
            <div className="text-muted-foreground flex h-full items-center justify-center text-sm">
              {t("Loading Maps Configuration..")}
            </div>
          ) : googleMapsQuery.data?.config.apiKey ? (
            <FormControl cols="full">
              <div className="space-y-1.5">
                <p className="required text-sm leading-none font-medium">{t("Geofence")}</p>
                <LocationGeofenceControls />
              </div>
            </FormControl>
          ) : null}
          <FormGroup cols={2}>
            <FormControl cols="full" id="address-field-container">
              <AddressField control={control} />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="addressLine2"
                label={t("Address Line 2")}
                placeholder={t("Suite, floor, building, etc.")}
                description={t("Additional address details such as suite number, floor, or building name.")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="city"
                rules={{ required: true }}
                label={t("City")}
                placeholder={t("City")}
                description={t("City where the customer's primary office or billing address is located.")}
              />
            </FormControl>
            <FormControl>
              <UsStateAutocompleteField
                control={control}
                name="stateId"
                label={t("State")}
                placeholder={t("State")}
                description={t("U.S. state for the billing address. Used for tax jurisdiction determination and regional reporting.")}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                rules={{ required: true }}
                control={control}
                name="postalCode"
                label={t("Postal Code")}
                placeholder={t("e.g., 90210")}
                description={t("ZIP or ZIP+4 code. Used for geocoding, mileage calculations, and tax jurisdiction lookups.")}
              />
            </FormControl>
            <FormControl cols="full">
              <SelectField
                control={control}
                name="timezone"
                label={t("Timezone")}
                placeholder={t("Not set (UTC)")}
                isClearable
                description={t("Local clock for this location. Rating formulas read pickup and delivery hours, weekdays, and dates in this zone; without one they use UTC.")}
                groups={timezoneGroupedChoices}
                renderOption={(option) => (
                  <span className="flex w-full items-center justify-between gap-3">
                    <span>{t(option.label)}</span>
                    {option.description && (
                      <span className="text-muted-foreground text-xs">{t(option.description)}</span>
                    )}
                  </span>
                )}
              />
            </FormControl>
          </FormGroup>

          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Notes")}
              placeholder={t("Add any extra detail about this location")}
              description={t("Optional notes for dispatchers and drivers.")}
              minRows={3}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
