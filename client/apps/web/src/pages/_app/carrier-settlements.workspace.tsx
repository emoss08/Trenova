import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { CarrierSettlementWorkspacePage } from "@/routes/carrier-settlement-workspace/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/carrier-settlements/workspace")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.Brokerage),
    requirePermission(Resource.CarrierSettlement),
  ),
  component: CarrierSettlementWorkspacePage,
});
