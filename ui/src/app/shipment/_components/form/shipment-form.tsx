import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { type Shipment } from "@/types/shipment";
import { faChevronLeft } from "@fortawesome/pro-solid-svg-icons";
import { ShipmentNotFoundOverlay } from "../sidebar/shipment-not-found-overlay";
import { ShipmentCommodityDetails } from "./commodity/shipment-commodity-details";
import {
  ShipmentBillingDetails,
  ShipmentDetailsHeader,
  ShipmentServiceDetails,
} from "./shipment-details-components";
import { ShipmentDetailsSkeleton } from "./shipment-details-skeleton";
import { ShipmentActions } from "./shipment-menu-actions";
import { ShipmentMovesDetails } from "./shipment-move-details";

// ShipmentDetails.tsx
interface ShipmentDetailsProps {
  selectedShipment?: Shipment | null;
  isLoading: boolean;
  onBack: () => void;
  dimensions: {
    contentHeight: number;
    viewportHeight: number;
  };
}

export function ShipmentForm({
  selectedShipment,
  isLoading,
  onBack,
  dimensions,
}: ShipmentDetailsProps) {
  if (isLoading) {
    return <ShipmentDetailsSkeleton />;
  }

  if (!selectedShipment) {
    return <ShipmentNotFoundOverlay onBack={onBack} />;
  }

  // Calculate the optimal height for the scroll area
  const calculateScrollAreaHeight = () => {
    const { contentHeight, viewportHeight } = dimensions;

    // Constants for height calculations
    const headerHeight = 120; // Height of the header section
    const minHeight = 400; // Minimum height for the scroll area
    const footerHeight = 60; // Height of the footer section

    // Use viewport height as base for calculation
    const baseHeight = Math.min(contentHeight, viewportHeight);
    const calculatedHeight = baseHeight - headerHeight - footerHeight;

    // Ensure we don't go below minimum height
    return Math.max(calculatedHeight, minHeight);
  };

  const scrollAreaHeight = `${calculateScrollAreaHeight()}px`;

  return (
    <div className="size-full">
      <div className="pt-4">
        <div className="flex items-center gap-2 px-4 justify-between">
          <Button variant="outline" size="sm" onClick={onBack}>
            <Icon icon={faChevronLeft} className="size-4" />
            <span className="text-sm">Back</span>
          </Button>
          <ShipmentActions shipment={selectedShipment} />
        </div>
        <div className="flex flex-col gap-2 mt-4">
          <ShipmentDetailsHeader />
          <ScrollArea
            className="flex flex-col overflow-y-auto px-4"
            style={{
              height: scrollAreaHeight,
              minHeight: "400px",
            }}
          >
            <div className="flex flex-col gap-4 pb-2">
              <ShipmentServiceDetails />
              <ShipmentBillingDetails />
              <ShipmentCommodityDetails />
              <ShipmentMovesDetails />
            </div>
          </ScrollArea>
        </div>
      </div>
    </div>
  );
}
