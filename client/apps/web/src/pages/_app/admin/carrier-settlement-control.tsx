import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { CarrierSettlementControlPage } from "@/routes/carrier-settlement-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/carrier-settlement-control")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.Brokerage),
    requirePermission(Resource.CarrierSettlementControl, Operation.Read),
  ),
  component: CarrierSettlementControlPage,
});
