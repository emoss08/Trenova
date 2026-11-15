import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RateAgreementPage } from "@/routes/rate-agreement/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/rate-agreements")({
  beforeLoad: gates(requirePermission(Resource.RateAgreement)),
  component: RateAgreementPage,
});
