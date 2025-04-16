import { integrationImages } from "@/app/integrations/_utils/integration";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Tour } from "@/components/tour/tour";
import { useTour } from "@/components/tour/tour-provider";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import { Separator } from "@/components/ui/separator";
import { GOOGLE_MAPS_NOTICE_KEY } from "@/constants/env";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { IntegrationType } from "@/types/integrations/integration";
import { faLightbulbOn, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useFormContext } from "react-hook-form";

export default function CustomerForm() {
  const { control } = useFormContext<CustomerSchema>();

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">General Information</h2>
        <p className="text-xs text-muted-foreground">
          Enter essential customer identification details including status,
          contact information, and physical address to establish the customer
          profile for shipment processing and billing.
        </p>
      </div>
      <Separator className="mt-2" />
      <GoogleMapsNotice />

      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="Defines the current operational status of the customer."
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
              description="A unique identifier for the customer."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              placeholder="Name"
              description="The official name of the customer."
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="Description"
              description="Additional details or notes about the customer."
            />
          </FormControl>
          <FormControl cols="full" id="address-field-container">
            <AddressField control={control} />
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
              description="The city where the customer is situated."
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
              description="The U.S. state where the customer is situated."
              options={usStateOptions}
              isLoading={usStates.isLoading}
              isFetchError={usStates.isError}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              rules={{ required: true }}
              control={control}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="The ZIP code for the customer."
            />
          </FormControl>
        </FormGroup>
      </div>
      <Tour />
    </div>
  );
}

function GoogleMapsNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    GOOGLE_MAPS_NOTICE_KEY,
    true,
  );

  // Access tour context
  const { openTour } = useTour();

  // Get integration by type
  const { data: integration } = useQuery({
    ...queries.integration.getIntegrationByType(IntegrationType.GoogleMaps),
  });

  const handleClose = () => {
    setNoticeVisible(false);
  };

  // Google Maps tour steps
  const googleMapsTourSteps = [
    {
      target: "#address-field-container",
      title: "Address Field with Google Maps",
      content:
        "The Address Line 1 field now features Google Maps integration. Click the search icon to access the address search functionality.",
      position: "right" as const,
    },
    {
      target: "#address-search-button",
      title: "Address Search",
      content:
        "Click this search icon to open the Google Maps address search tool.",
      position: "top" as const,
      action: () => {
        // Find and click the search button to open the popover
        const searchButton = document.querySelector("#address-search-button");
        if (searchButton && searchButton instanceof HTMLButtonElement) {
          // Check if popover is already open
          const popoverContent = document.querySelector(
            "[data-radix-popper-content-wrapper]",
          );
          if (!popoverContent) {
            searchButton.click();
          }
        }
      },
    },
    {
      target: "[data-radix-popper-content-wrapper]",
      title: "Search for Addresses",
      content:
        "Type an address in the search box to see suggestions from Google Maps. Select an address to auto-fill all your address fields.",
      position: "bottom" as const,
      action: () => {
        // Make sure the popover is open
        const popoverContent = document.querySelector(
          "[data-radix-popper-content-wrapper]",
        );
        if (!popoverContent) {
          // If not open, open it
          const searchButton = document.querySelector("#address-search-button");
          if (searchButton && searchButton instanceof HTMLButtonElement) {
            searchButton.click();
          }
        }

        // Find the search input and simulate typing
        setTimeout(() => {
          const searchInput = document.querySelector(
            "[data-radix-popper-content-wrapper] input",
          );
          if (searchInput && searchInput instanceof HTMLInputElement) {
            // Create a fake input event to simulate typing
            const simulatedEvent = new Event("input", { bubbles: true });

            // Update the value property
            searchInput.value = "123 Main Street";

            // Dispatch the event
            searchInput.dispatchEvent(simulatedEvent);

            // Focus the input
            searchInput.focus();
          }
        }, 500); // Allow time for the popover to fully open
      },
    },
    {
      target: "#address-field-container",
      title: "Auto-fill Address Details",
      content:
        "When you select an address from Google Maps, it automatically fills in all address fields including city, state, and postal code.",
      position: "bottom" as const,
    },
    {
      target: "#address-field-container",
      title: "Geocoding Benefits",
      content:
        "Using Google Maps addresses provides accurate geocoding data, which improves routing and distance calculations in shipments.",
      position: "left" as const,
    },
  ];

  const handleTakeTour = () => {
    // Track original form values
    const addressLine1Input = document.querySelector(
      'input[name="addressLine1"]',
    ) as HTMLInputElement | null;
    const addressLine2Input = document.querySelector(
      'input[name="addressLine2"]',
    ) as HTMLInputElement | null;
    const cityInput = document.querySelector(
      'input[name="city"]',
    ) as HTMLInputElement | null;
    const postalCodeInput = document.querySelector(
      'input[name="postalCode"]',
    ) as HTMLInputElement | null;

    // Save original values
    const originalValues = {
      addressLine1: addressLine1Input?.value || "",
      addressLine2: addressLine2Input?.value || "",
      city: cityInput?.value || "",
      postalCode: postalCodeInput?.value || "",
    };

    // Define a cleanup function that restores original values
    const cleanup = () => {
      // Close any open popovers
      const searchButton = document.querySelector("#address-search-button");
      const popoverContent = document.querySelector(
        "[data-radix-popper-content-wrapper]",
      );

      // remove the notice
      setNoticeVisible(false);

      if (popoverContent && searchButton instanceof HTMLButtonElement) {
        searchButton.click(); // Click to close if open

        // Add a short delay to ensure the popover is closed before resetting values
        setTimeout(() => {
          // Restore original form values
          if (addressLine1Input)
            addressLine1Input.value = originalValues.addressLine1;
          if (addressLine2Input)
            addressLine2Input.value = originalValues.addressLine2;
          if (cityInput) cityInput.value = originalValues.city;
          if (postalCodeInput)
            postalCodeInput.value = originalValues.postalCode;
        }, 300);
      } else {
        // If no popover is open, restore values immediately
        if (addressLine1Input)
          addressLine1Input.value = originalValues.addressLine1;
        if (addressLine2Input)
          addressLine2Input.value = originalValues.addressLine2;
        if (cityInput) cityInput.value = originalValues.city;
        if (postalCodeInput) postalCodeInput.value = originalValues.postalCode;
      }
    };

    // Start the tour with our defined steps
    openTour(googleMapsTourSteps, cleanup);
    // Optionally hide the notice when tour starts
    // setNoticeVisible(false);
  };

  const showNotice = noticeVisible && integration?.configuration?.apiKey;

  return showNotice ? (
    <div className="flex bg-blue-500/20 border border-blue-600 p-4 rounded-md justify-between items-center m-2">
      <div className="flex items-center gap-2 w-full text-blue-600 pr-2">
        <LazyImage
          src={integrationImages[IntegrationType.GoogleMaps]}
          layout="fixed"
          width={10}
          height={10}
          className="size-6"
        />
        <div className="flex flex-col">
          <p className="text-sm font-medium">Google Maps API</p>
          <p className="text-xs dark:text-blue-100">
            Your organization has configured the Google Maps API! Take a tour to
            learn how to use it.
          </p>
        </div>
      </div>
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <Button
              variant="outline"
              size="sm"
              type="button"
              className="text-blue-600 border-blue-600 hover:bg-blue-400/10 hover:text-blue-600 bg-blue-400/10"
              onClick={handleTakeTour}
            >
              <Icon icon={faLightbulbOn} className="size-3 mr-1" />
              Take a Tour
            </Button>
          </div>
        </div>
        <div className="absolute top-24 right-4">
          <Button
            variant="ghost"
            type="button"
            className="group -my-1.5 -me-2 size-8 shrink-0 p-0 text-blue-600 hover:text-blue-400 hover:bg-transparent"
            onClick={handleClose}
            aria-label="Close banner"
          >
            <Icon
              icon={faXmark}
              className="opacity-60 transition-opacity group-hover:opacity-100"
              aria-hidden="true"
            />
          </Button>
        </div>
      </div>
    </div>
  ) : null;
}
