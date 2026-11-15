import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { SettlementWorkspacePage } from "@/routes/settlement-workspace/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/payroll/workspace")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.DriverSettlement),
  ),
  component: SettlementWorkspacePage,
});
