import { createFileRoute } from "@tanstack/react-router";
import { TenderOfferPublicPage } from "@/routes/tender-offer-public/page";

export const Route = createFileRoute("/(public)/tender-offer/$token")({
  component: TenderOfferPublicPage,
});
