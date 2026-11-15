import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { FleetCodesPage } from "@/routes/fleet-code/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/configuration-files/fleet-codes")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.FleetCode, Operation.Read),
  ),
  component: FleetCodesPage,
});
