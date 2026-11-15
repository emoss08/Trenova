import { createFileRoute } from "@tanstack/react-router";
import { RateConfirmationPublicPage } from "@/routes/rate-confirmation-public/page";

export const Route = createFileRoute("/(public)/rate-confirmation/$token")({
  component: RateConfirmationPublicPage,
});
