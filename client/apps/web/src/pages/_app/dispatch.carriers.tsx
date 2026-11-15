import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { CarriersPage } from "@/routes/carrier/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/carriers")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.Brokerage),
    requirePermission(Resource.Carrier),
  ),
  component: CarriersPage,
});
