/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";
import { AnthropicIcon } from "@/components/brand-icons";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { LocationCategoryAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Icon } from "@/components/ui/icons";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { aiAPI } from "@/services/ai";
import { faSparkle } from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Bot } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

export function LocationForm() {
  const { control, setValue } = useFormContext<LocationSchema>();

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];
  const [locationName, locationDescription, locationAddress] = useWatch({
    control,
    name: ["name", "description", "addressLine1"],
  });

  // AI classification mutation
  const classifyMutation = useMutation({
    mutationFn: aiAPI.classifyLocation,
    onSuccess: (data) => {
      if (data.categoryId) {
        setValue("locationCategoryId", data.categoryId, {
          shouldValidate: true,
        });
      }
    },
    onError: (error) => {
      console.error("Failed to get AI suggestion:", error);
    },
  });

  const handleGetAISuggestion = () => {
    if (!locationName) return;

    classifyMutation.mutate({
      name: locationName,
      description: locationDescription,
      address: locationAddress,
    });
  };

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
            <HoverCard>
              <HoverCardTrigger asChild>
                <button
                  type="button"
                  onClick={handleGetAISuggestion}
                  disabled={!locationName || classifyMutation.isPending}
                  className="[&>svg]:size-3 size-5 mr-1 disabled:cursor-not-allowed disabled:opacity-50 rounded-md flex items-center justify-center hover:bg-purple-500/30 text-muted-foreground hover:text-foreground transition-colors duration-200 ease-in-out cursor-pointer"
                  title="Get AI category suggestion"
                >
                  {classifyMutation.isPending ? (
                    <Bot className="h-4 w-4" />
                  ) : (
                    <Icon icon={faSparkle} className="size-4 text-purple-500" />
                  )}
                </button>
              </HoverCardTrigger>
              <HoverCardContent className="p-0 w-auto max-w-sm text-sm border shadow-lg">
                <div className="p-4 space-y-3">
                  <div className="space-y-2">
                    <h4 className="font-medium text-foreground">
                      AI Category Suggestions
                    </h4>
                    <p className="text-muted-foreground leading-relaxed">
                      Get intelligent category suggestions for your location
                      using advanced AI analysis.
                    </p>
                  </div>

                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <div className="w-1.5 h-1.5 bg-blue-500 rounded-full"></div>
                      <span>Processing time: ~2-5 seconds</span>
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <div className="w-1.5 h-1.5 bg-green-500 rounded-full"></div>
                      <span>Cost: ~500 tokens per suggestion</span>
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <div className="w-1.5 h-1.5 bg-purple-500 rounded-full"></div>
                      <span>Model: Claude 3 Haiku</span>
                    </div>
                  </div>
                </div>

                <div className="flex items-center justify-center gap-0.5 border-t bg-muted/30 px-4 py-2 rounded-b-md">
                  <span className="text-xs text-muted-foreground">
                    Powered by
                  </span>
                  <button
                    onClick={() =>
                      window.open("https://www.anthropic.com", "_blank")
                    }
                    className="flex items-center gap-1 text-xs font-medium text-foreground hover:text-primary transition-colors cursor-pointer"
                  >
                    <AnthropicIcon className="size-4 fill-current" />
                    Anthropic
                  </button>
                </div>
              </HoverCardContent>
            </HoverCard>
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
      <FormControl cols="full">
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
  );
}
