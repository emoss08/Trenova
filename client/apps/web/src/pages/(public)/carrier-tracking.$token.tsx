import { createFileRoute } from "@tanstack/react-router";
import { CarrierTrackingPublicPage } from "@/routes/carrier-tracking-public/page";

export const Route = createFileRoute("/(public)/carrier-tracking/$token")({
  component: CarrierTrackingPublicPage,
});
