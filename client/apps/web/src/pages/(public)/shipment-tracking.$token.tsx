import { createFileRoute } from "@tanstack/react-router";
import { ShipmentTrackingPublicPage } from "@/routes/shipment-tracking-public/page";

export const Route = createFileRoute("/(public)/shipment-tracking/$token")({
  component: ShipmentTrackingPublicPage,
});
