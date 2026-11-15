import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ServiceFailuresPage } from "@/routes/service-failure/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/service-failures")({
  beforeLoad: gates(requirePermission(Resource.ServiceFailure)),
  component: ServiceFailuresPage,
});
