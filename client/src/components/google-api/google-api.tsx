/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import {
  GoogleAPI as GoogleAPIType,
  GoogleAPIFormValues,
} from "@/types/organization";
import React from "react";
import { useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { googleAPISchema } from "@/lib/validations/OrganizationSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { Button } from "@/components/ui/button";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { routeDistanceUnitChoices, routeModelChoices } from "@/lib/choices";
import { useGoogleAPI } from "@/hooks/useQueries";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { InfoIcon } from "lucide-react";

function GoogleAPIAlert() {
  return (
    <Alert className="mb-5 bg-foreground text-background">
      <InfoIcon className="h-5 w-5 stroke-background" />
      <AlertTitle>Information!</AlertTitle>
      <AlertDescription>
        <ul className="list-disc">
          <li>
            <strong>Google API Key:</strong> Required to access Google's mapping
            services, including routing and geocoding. Ensure you have the
            correct API permissions and billing set up in your Google Cloud
            Platform account. For more details, see{" "}
            <a
              href="https://developers.google.com/maps/documentation/javascript/get-api-key"
              target="_blank"
              className="underline"
              rel="noopener noreferrer"
            >
              Google's API Key Documentation
            </a>
          </li>
          <li>
            <strong>Mileage Unit:</strong> Choose the unit for distance
            measurement (e.g., imperial, metric). For more details, see{" "}
            <a
              href="https://support.google.com/merchants/answer/14156166?hl=en"
              target="_blank"
              className="underline"
              rel="noopener noreferrer"
            >
              Google's Unit Systems Documentation
            </a>
          </li>
          <li>
            <strong>Traffic Model:</strong> Determines how traffic conditions
            affect route calculation. For more details, see{" "}
            <a
              href="https://developers.google.com/maps/documentation/distance-matrix/distance-matrix#traffic_model"
              target="_blank"
              className="underline"
              rel="noopener noreferrer"
            >
              Google's Traffic Model Documentation
            </a>
            .
          </li>
        </ul>
      </AlertDescription>
    </Alert>
  );
}

function GoogleApiForm({ googleApi }: { googleApi: GoogleAPIType }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const [showAPIKey, setShowAPIKey] = React.useState(false);

  const toggleAPIKeyVisibility = () => {
    setShowAPIKey(!showAPIKey);
  };

  const { control, handleSubmit, reset, watch, formState } =
    useForm<GoogleAPIFormValues>({
      resolver: yupResolver(googleAPISchema),
      defaultValues: googleApi,
    });

  const apiKeyValue = watch("apiKey");

  const mutation = useCustomMutation<GoogleAPIFormValues>(
    control,
    {
      method: "PUT",
      path: "/organization/google_api_details/", // Does not require an ID
      successMessage: "Google API settings updated successfully.",
      queryKeysToInvalidate: ["googleAPI"],
      errorMessage: "Failed to update google api settings.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: GoogleAPIFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <GoogleAPIAlert />
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="mileageUnit"
              control={control}
              options={routeDistanceUnitChoices}
              rules={{ required: true }}
              label="Mileage Unit"
              placeholder="Mileage Unit"
              description="Select the unit of measurement for mileage to ensure accurate distance tracking across different regions."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="trafficModel"
              control={control}
              options={routeModelChoices}
              rules={{ required: true }}
              label="Traffic Model"
              placeholder="Traffic Model"
              description="Choose a traffic model for enhanced route calculation, factoring in real-time traffic conditions for optimal routing."
            />
          </div>
          <div className="relative col-span-4">
            <InputField
              name="apiKey"
              control={control}
              type={showAPIKey ? "text" : "password"}
              rules={{ required: true }}
              label="Google API Key"
              placeholder="API Key"
              description="Securely input your Google API Key to access and integrate Google's mapping services."
            />
            {apiKeyValue && formState.isValid && (
              <button
                type="button"
                className="absolute inset-y-0 right-0 mt-2 flex items-center pr-3 text-sm leading-5"
                onClick={toggleAPIKeyVisibility}
              >
                <p className="text-xs uppercase text-foreground">
                  {showAPIKey ? "hide" : "show"}
                </p>
              </button>
            )}
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="addCustomerLocation"
              control={control}
              label="Add Customer Location"
              description="Enable this to utilize and enforce the usage of the Google Places API for adding customer locations, ensuring accurate and standardized data entry."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="addLocation"
              control={control}
              label="Add Location"
              description="Activate to mandate the use of the Google Places API when adding new locations, promoting consistency and precision in location data."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoGeocode"
              control={control}
              label="Auto Geocode"
              description="Automatically convert addresses into geographical coordinates for accurate and hassle-free location mapping."
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-6 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function GoogleApi() {
  const { googleAPIData, isLoading, isError } = useGoogleAPI();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Google API
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Empower your dispatch system with Google API integration. Configure
          settings for advanced routing, precise geocoding, and dynamic map
          views to streamline your fleet management. Enhance route planning,
          location accuracy, and visual data representation for an optimal
          operational experience.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="m-4 bg-background p-8 ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <ErrorLoadingData message="Failed to load dispatch control." />
        </div>
      ) : (
        (googleAPIData as GoogleAPIType) && (
          <GoogleApiForm googleApi={googleAPIData as GoogleAPIType} />
        )
      )}
    </div>
  );
}
