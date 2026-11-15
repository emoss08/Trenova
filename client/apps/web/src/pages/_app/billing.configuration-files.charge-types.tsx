import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { PlaceholderPage } from "@/routes/placeholder-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/charge-types")({
  beforeLoad: gates(requirePermission(Resource.ChargeType)),
  component: PlaceholderPage,
});
