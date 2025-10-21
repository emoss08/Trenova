import { integrationImages } from "@/app/integrations/_utils/integration";
import { GOOGLE_MAPS_NOTICE_KEY } from "@/constants/env";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { cn } from "@/lib/utils";
import { IntegrationType } from "@/types/integration";
import { faLightbulbOn } from "@fortawesome/pro-regular-svg-icons";
import { useTour } from "./tour/tour-provider";
import { Button } from "./ui/button";
import { Icon } from "./ui/icons";
import { LazyImage } from "./ui/image";

export function GoogleMapsNotice({ className }: { className?: string }) {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    GOOGLE_MAPS_NOTICE_KEY,
    true,
  );

  const { openTour } = useTour();

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
          const searchButton = document.querySelector("#address-search-button");
          if (searchButton && searchButton instanceof HTMLButtonElement) {
            searchButton.click();
          }
        }

        setTimeout(() => {
          const searchInput = document.querySelector(
            "[data-radix-popper-content-wrapper] input",
          );
          if (searchInput && searchInput instanceof HTMLInputElement) {
            const simulatedEvent = new Event("input", { bubbles: true });

            searchInput.value = "123 Main Street";

            searchInput.dispatchEvent(simulatedEvent);

            searchInput.focus();
          }
        }, 500);
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

    const cleanup = () => {
      const searchButton = document.querySelector("#address-search-button");
      const popoverContent = document.querySelector(
        "[data-radix-popper-content-wrapper]",
      );

      // remove the notice
      setNoticeVisible(false);

      if (popoverContent && searchButton instanceof HTMLButtonElement) {
        searchButton.click();
        setTimeout(() => {
          if (addressLine1Input)
            addressLine1Input.value = originalValues.addressLine1;
          if (addressLine2Input)
            addressLine2Input.value = originalValues.addressLine2;
          if (cityInput) cityInput.value = originalValues.city;
          if (postalCodeInput)
            postalCodeInput.value = originalValues.postalCode;
        }, 300);
      } else {
        if (addressLine1Input)
          addressLine1Input.value = originalValues.addressLine1;
        if (addressLine2Input)
          addressLine2Input.value = originalValues.addressLine2;
        if (cityInput) cityInput.value = originalValues.city;
        if (postalCodeInput) postalCodeInput.value = originalValues.postalCode;
      }
    };

    openTour(googleMapsTourSteps, cleanup);
  };

  return noticeVisible ? (
    <div
      className={cn(
        "flex bg-blue-500/20 border border-blue-600 p-4 rounded-md justify-between items-center",
        className,
      )}
    >
      <div className="flex items-center gap-2 w-full text-blue-600 pr-2">
        <LazyImage
          src={integrationImages[IntegrationType.GoogleMaps]}
          className="size-6"
        />
        <div className="flex flex-col">
          <p className="text-sm font-medium">Google Maps API</p>
          <p className="text-xs text-blue-600 dark:text-blue-300">
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
              className="text-blue-600 border-blue-600 hover:bg-blue-400/10 hover:text-blue-600 bg-blue-600/10"
              onClick={handleTakeTour}
            >
              <Icon icon={faLightbulbOn} className="size-3" />
              Take a Tour
            </Button>
          </div>
        </div>
      </div>
    </div>
  ) : null;
}
