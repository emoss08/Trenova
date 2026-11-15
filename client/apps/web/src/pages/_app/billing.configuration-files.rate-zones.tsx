import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RateZonePage } from "@/routes/rate-zone/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/rate-zones")({
  beforeLoad: gates(requirePermission(Resource.RateZone)),
  component: RateZonePage,
});
