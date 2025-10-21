"use no memo";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { GoogleMapsNotice } from "@/components/google-maps-tour";
import { Tour } from "@/components/tour/tour";
import { TourProvider } from "@/components/tour/tour-provider";
import { LocationCategoryAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { Spinner } from "@/components/ui/shadcn-io/spinner";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { cn } from "@/lib/utils";
import { aiAPI } from "@/services/ai";
import { faCheck, faSparkle } from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function LocationForm() {
  const { control, setValue } = useFormContext<LocationSchema>();
  const [showSuccess, setShowSuccess] = useState(false);

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data ?? [];
  const [
    locationName,
    locationDescription,
    locationAddress,
    locationCity,
    locationState,
    locationPostalCode,
    locationLatitude,
    locationLongitude,
    locationPlaceId,
    locationCode,
  ] = useWatch({
    control,
    name: [
      "name",
      "description",
      "addressLine1",
      "city",
      "state",
      "postalCode",
      "latitude",
      "longitude",
      "placeId",
      "code",
    ],
  });

  const classifyMutation = useMutation({
    mutationFn: aiAPI.classifyLocation,
    onSuccess: (data) => {
      if (data.categoryId) {
        setValue("locationCategoryId", data.categoryId, {
          shouldValidate: true,
        });
        setShowSuccess(true);
        setTimeout(() => setShowSuccess(false), 3000);
      }
    },
    onError: (error) => {
      console.error("Failed to get AI suggestion:", error);
    },
  });

  useEffect(() => {
    if (showSuccess) {
      setShowSuccess(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locationName]);

  const handleGetAISuggestion = () => {
    if (!locationName) return;

    setShowSuccess(false);
    classifyMutation.mutate({
      name: locationName,
      description: locationDescription,
      address: locationAddress,
      city: locationCity,
      state: locationState?.name ?? undefined,
      postalCode: locationPostalCode,
      latitude: locationLatitude ?? undefined,
      longitude: locationLongitude ?? undefined,
      placeId: locationPlaceId,
      code: locationCode,
    });
  };

  return (
    <TourProvider>
      <div className="flex flex-col gap-2">
        <GoogleMapsNotice />
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
              maxLength={10}
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
              maxLength={100}
            />
          </FormControl>
          <FormControl cols="full">
            <div className="relative">
              <LocationCategoryAutocompleteField<LocationSchema>
                name="locationCategoryId"
                control={control}
                rules={{ required: true }}
                label="Location Category"
                placeholder="Select Location Category"
                description="Select the location category for the location."
              />
              <div className="absolute top-6 right-5 flex items-center">
                <button
                  type="button"
                  onClick={handleGetAISuggestion}
                  disabled={!locationName || showSuccess}
                  className={cn(
                    "size-5 mr-1 rounded-md flex items-center justify-center",
                    "transition-all duration-300 ease-in-out cursor-pointer",
                    "disabled:cursor-not-allowed disabled:opacity-50",
                    "text-muted-foreground hover:text-foreground",
                    classifyMutation.isPending && "cursor-wait",
                    showSuccess
                      ? "bg-green-500/20 hover:bg-green-500/30"
                      : "hover:bg-purple-500/30",
                  )}
                  title={
                    showSuccess ? "Category set!" : "Get AI category suggestion"
                  }
                >
                  {classifyMutation.isPending ? (
                    <Spinner variant="ring" className="size-4" />
                  ) : showSuccess ? (
                    <Icon
                      icon={faCheck}
                      className="size-3 text-green-500 animate-in fade-in zoom-in duration-300"
                    />
                  ) : (
                    <Icon
                      icon={faSparkle}
                      className="size-3 text-purple-500 animate-in fade-in duration-200"
                    />
                  )}
                </button>
              </div>
            </div>
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
          <FormControl cols="full" id="address-field-container">
            <AddressField control={control} rules={{ required: true }} />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Address Line 2"
              placeholder="Address Line 2"
              description="Additional address details, if applicable."
              maxLength={150}
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
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="stateId"
              label="State"
              placeholder="State"
              description="The U.S. state where the location is situated."
              options={usStateOptions}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="The ZIP code for the location."
              rules={{ required: true }}
              maxLength={150}
            />
          </FormControl>
        </FormGroup>
        <Tour />
      </div>
    </TourProvider>
  );
}
